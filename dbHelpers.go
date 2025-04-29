package main

import (
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"strings"
	"time"
	"fmt"
)

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {

	if video.VideoURL == nil {
		return database.Video{}, fmt.Errorf("Video URL is nil")
	}

	videoParams := strings.Split(*video.VideoURL, ",")
	if len(videoParams) != 2 {
		return database.Video{}, fmt.Errorf("Cannot split video URL: %s to bucket and key", *video.VideoURL)
	}
	bucket := videoParams[0]
	key := videoParams[1]

	presignedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, 5 * time.Minute)
	if err != nil {
		return database.Video{}, err
	}

	video.VideoURL = &presignedURL
	return video, nil
}