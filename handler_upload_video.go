package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"

	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	http.MaxBytesReader(w, r.Body, 1<<30)

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	mData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, 500, "error getting video from id", err)
		return
	}
	if mData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	f, h, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, 500, "Error parsing request body", err)
		return
	}
	defer f.Close()

	// using mime parsing (unlike in thumbnail func) because boot.dev said so
	mediaType, _, err := mime.ParseMediaType(h.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, 500, "Error checking file MIME type", err)
		return
	}
	if mediaType != "video/mp4" {
		err = fmt.Errorf("bad file type")
		respondWithError(w, http.StatusBadRequest, "Video file format is not .mp4", err)
		return
	}

	tmp, err := os.CreateTemp("/tmp", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, 500, "Error creating temp file", err)
		return
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	// copy video file contents from request body to temp file
	bytesCopied, err := io.Copy(tmp, f)
	if err != nil {
		respondWithError(w, 500, "Error copying to temp file", err)
		return
	}
	fmt.Printf("Copying video file (%d bytes) to %s", bytesCopied, tmp.Name())

	tmp.Seek(0, io.SeekStart)

	c := make([]byte, 32)
	_, err = rand.Read(c)
	if err != nil {
		respondWithError(w, 500, "error generating random bytes", err)
		return
	}
	keyStr := base64.RawURLEncoding.EncodeToString(c) + ".mp4"

	aspect, err := getVideoAspectRatio(tmp.Name())
	if err != nil {
		respondWithError(w, 500, "error getting video aspect", err)
	}
	var aspPrefix string
	switch aspect {
	case "16:9":
		aspPrefix = "landscape"
	case "9:16":
		aspPrefix = "portrait"
	default:
		aspPrefix = "other"
	}
	keyStr = aspPrefix + "/" + keyStr

	// ?? (first return value would just be a pointer)
	_, err = cfg.s3client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &keyStr,
		Body:        tmp,
		ContentType: &mediaType,
	})
	if err != nil {
		fmt.Printf("Error putting file into S3 client: %s", err)
		respondWithError(w, 500, "error putting file into S3 client", err)
		return
	}

	ff := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, keyStr)
	mData.VideoURL = &ff
	err = cfg.db.UpdateVideo(mData)
	if err != nil {
		respondWithError(w, 500, "error updating video data", err)
		return
	}

	respondWithJSON(w, http.StatusOK, mData)
}
