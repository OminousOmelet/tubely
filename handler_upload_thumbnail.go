package main

import (
	//"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	formFile, formHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, 500, "failed to get file from key", err)
		return
	}
	fileType := formHeader.Header.Get("Content-type")

	// data, err := io.ReadAll(formFile)
	// if err != nil {
	// 	respondWithError(w, 500, "error reading from file", err)
	// 	return
	// }

	mData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, 500, "error getting video from id", err)
		return
	}
	if mData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	extension := strings.Split(fileType, "/")[1]
	path := filepath.Join(cfg.assetsRoot, videoIDString+"."+extension)

	//dataB64 := base64.StdEncoding.EncodeToString(data)
	//dataURL := fmt.Sprintf("data:%v;base64,%v", fileType, dataB64)

	f, err := os.Create(path)
	if err != nil {
		respondWithError(w, 500, "error creating thumbnail file", err)
	}
	defer f.Close()
	io.Copy(f, formFile)

	tnURL := fmt.Sprintf("http://localhost:%s/%s", cfg.port, path)
	mData.ThumbnailURL = &tnURL
	mData.UpdatedAt = time.Now()

	err = cfg.db.UpdateVideo(mData)
	if err != nil {
		respondWithError(w, 500, "failed to update video", err)
		return
	}

	fmt.Println(fileType)
	fmt.Println(path)
	fmt.Println(tnURL)
	respondWithJSON(w, http.StatusOK, mData)
}
