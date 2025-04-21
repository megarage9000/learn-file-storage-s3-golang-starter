package main

import (
	"fmt"
	"net/http"
	"io"
	"os"
	"mime"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
	"path/filepath"
	"strings"
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

	// TODO: implement the upload here
	// max memory set to 10 mb
	const maxMemory = 10 << 20

	// Parsing to multiform from request
	multiParseErr := r.ParseMultipartForm(maxMemory)
	if multiParseErr != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse to multi part form", multiParseErr)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unable to parse form file", err)
		return
	}

	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to get media type", err)
		return
	} else if mediaType != "image/jpeg" || mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid file format for thumbnails, attempted to upload file of format %s", mediaType), err)
		return
	}
	// Reading thumbnail data
	// data, readErr := io.ReadAll(file)
	// if readErr != nil {
	// 	respondWithError(w, http.StatusBadRequest, "Unable to read data", err)
	// 	return
	// }

	// Obtaining video meta data
	videoMetaData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Unable to get video metadata for video ID %s", videoID), err)
		return
	} else if videoMetaData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized user", err)
		return
	}

	// // Saving as thumbnail
	// videoThumbnail := thumbnail {
	// 	data: data,
	// 	mediaType: mediaType,
	// }

	// // Updating the video thumbnail URL
	// videoThumbnails[videoID] = videoThumbnail
	
	// Encoding data into an encoder
	// encodedThumbnailStr := base64.StdEncoding.EncodeToString(data)
	// thumbnailURL := fmt.Sprintf("data:%s;base64,%s", mediaType, encodedThumbnailStr)

	// Saving the file on our own file system
	thumbnailName := fmt.Sprintf("%s.%s", videoID, strings.ReplaceAll(mediaType, "image/", ""))
	filePath := filepath.Join(cfg.assetsRoot, thumbnailName)
	thumbnailData, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Unable to create file for %s", thumbnailName), err)
		return
	}
	defer thumbnailData.Close()

	_, err = io.Copy(thumbnailData, file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Unable to copy file for %s", thumbnailName), err)
		return
	}

	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, thumbnailName)
	videoMetaData.ThumbnailURL = &thumbnailURL
	cfg.db.UpdateVideo(videoMetaData)

	respondWithJSON(w, http.StatusOK, videoMetaData)
}
