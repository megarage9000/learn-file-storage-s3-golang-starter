package main

import (
	"os/exec"
	"bytes"
	"fmt"
	"encoding/json"
	"math"
)

func getVideoAspectRatio(filePath string) (string, error) {

	// Creating ffprobe command and setting output to a buffer
	command := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	var output bytes.Buffer
	command.Stdout = &output

	// Executing the ffprobe command
	err := command.Run()
	if err != nil {
		return "", err
	}

	// Marshalling the command to a struct
	var ffProbeData FFProbeData
	
	if err := json.Unmarshal(output.Bytes(), &ffProbeData); err != nil {
		return "", fmt.Errorf("Error in unmarshalling json data ino FFProbeData struct: %s\n", err)
	}

	// Returning division properties
	ratio := float64(ffProbeData.Streams[0].Width) / float64(ffProbeData.Streams[0].Height)

    // Use thresholds to determine the ratio
    // For 16:9 ratio = 1.77777...
    // For 9:16 ratio = 0.5625
    const (
        landscapeTarget = 16.0 / 9.0  // approximately 1.778
        portraitTarget = 9.0 / 16.0   // approximately 0.563
        threshold = 0.1               // Adjust this as needed
    )
    
    if math.Abs(ratio - landscapeTarget) < threshold {
        return "landscape", nil
    } else if math.Abs(ratio - portraitTarget) < threshold {
        return "portrait", nil
    } else {
        return "other", nil
    }
}


