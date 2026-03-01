package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	buff := bytes.Buffer{}
	cmd.Stdout = &buff
	cmd.Run()
	type Aspect struct {
		Width  float32 `json:"width"`
		Height float32 `json:"height"`
	}
	type JsonData struct {
		Streams []Aspect `json:"streams"`
	}

	var asp JsonData
	err := json.Unmarshal(buff.Bytes(), &asp)
	if err != nil {
		fmt.Printf("\n%s", err)
		return "", err
	}

	if len(asp.Streams) < 1 {
		return "", errors.New("no file data received from path")
	}

	var ratioStr string
	ratio := asp.Streams[0].Width / asp.Streams[0].Height
	if ratio > 1.76 && ratio < 1.79 {
		ratioStr = "16:9"
	} else if ratio > 0.54 && ratio < 0.57 {
		ratioStr = "9:16"
	} else {
		ratioStr = "other"
	}

	fmt.Printf("\nvideo width: %f, video height: %f", asp.Streams[0].Width, asp.Streams[0].Height)
	fmt.Printf("\nVideo aspect measured: %s\n", ratioStr)
	return ratioStr, nil
}
