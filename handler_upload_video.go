package main

import (
	"net/http"
	"github.com/google/uuid"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"fmt"
	"mime"
	"os"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"io"
	"encoding/base64"
	"crypto/rand"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	const maxMemory = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, maxMemory)

	// Grab the videoID
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	// Getting user and validating
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

	// Getting video metadata
	videoMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Unable to get video metadata for video ID %s", videoID), err)
		return
	} else if videoMetadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, fmt.Sprintf("User is not authorized to get video data"), err)
	}

	// Parsing to multiform from request to get video data
	multiParseErr := r.ParseMultipartForm(maxMemory)
	if multiParseErr != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse to multi part form", multiParseErr)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to get video", err)
		return
	}

	defer file.Close()

	// Ensure that the file type is valid
	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to get media type", err)
		return
	} else if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid file format for videos, attempted to upload file of format %s", mediaType), err)
		return
	}

	// Saving file to temporary location on disk
	tempFileName := "tubely-upload.mp4"
	temp, err := os.CreateTemp("", tempFileName)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not create temporary file for upload", err)
		return
	}

	
	defer os.Remove(tempFileName)
	defer temp.Close()
	
	_, err = io.Copy(temp, file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not copy to temp file", err)
		return
	}

	
	// Using the temporary file as upload object
	_, err = temp.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Resetting file resulted in err for temp file", err)
		return
	}

	// Reading aspect ratio from file when saved
	aspect, err := getVideoAspectRatio(temp.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to get aspect ratio from file", err)
		return
	}

	videoKeyBytes := make([]byte, 32)
	rand.Read(videoKeyBytes)
	videoKey := base64.RawURLEncoding.EncodeToString(videoKeyBytes)
	videoKey = fmt.Sprintf("%s/%s", aspect, videoKey)

	// Processing the file to have fast start
	processedName, err := processVideoForFastStart(temp.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to process video for fast start", err)
		return
	}

	processedFile, err := os.OpenFile(processedName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Unable to open processed file %s", processedName), err)
	}

	defer os.Remove(processedName)
	defer processedFile.Close()

	putObjectParams := s3.PutObjectInput {
		Bucket: &cfg.s3Bucket,
		Key: &videoKey,
		Body: processedFile,
		ContentType: &mediaType,
	}

	_, err = cfg.s3Client.PutObject(r.Context(), &putObjectParams)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Put Object failed", err)
		return
	}

	// Uploading to database the new data
	// videoURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, videoKey)

	// Using presigned URLs for url, so we only pass the bucket and key
	videoURL := fmt.Sprintf("%s,%s", cfg.s3Bucket, videoKey)
	videoMetadata.VideoURL = &videoURL
	presignedVideoData, err := cfg.dbVideoToSignedVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Cannot obtain presigned data", err)
		return
	}

	videoUploadErr := cfg.db.UpdateVideo(presignedVideoData)
	if videoUploadErr != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to update video data", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoMetadata)
}




