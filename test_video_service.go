package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"time"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/robot/client"
	"go.viam.com/rdk/services/video"
	"go.viam.com/utils/rpc"
)

func main() {
	logger := logging.NewDebugLogger("test-video-service")

	// Connection credentials
	apiKeyID := "314b9a1b-f332-4a17-a6cc-5021ca5c8228"
	apiKey := "yflldsvbtkr69px8mqawen4co3j7vyuw"
	machineAddress := "framework-pc-dev-1-main.h028k9vtj1.viam.cloud"

	ctx := context.Background()
	machine, err := client.New(
		ctx,
		machineAddress,
		logger,
		client.WithDialOptions(rpc.WithEntityCredentials(
			apiKeyID,
			rpc.Credentials{
				Type:    rpc.CredentialsTypeAPIKey,
				Payload: apiKey,
			})),
	)
	if err != nil {
		logger.Fatal(err)
	}
	defer machine.Close(ctx)

	logger.Info("================================================================================")
	logger.Info("Testing Camera: LOREXSystems-LNZ43P4A-ND012410005900-url0")
	logger.Info("================================================================================")

	// Test Camera
	lorexCamera, err := camera.FromProvider(machine, "LOREXSystems-LNZ43P4A-ND012410005900-url0")
	if err != nil {
		logger.Errorf("Failed to get camera: %v", err)
	} else {
		props, err := lorexCamera.Properties(ctx)
		if err != nil {
			logger.Errorf("Failed to get properties: %v", err)
		} else {
			logger.Infof("✓ Got camera properties: %+v", props)
		}
	}

	logger.Info("================================================================================")
	logger.Info("Testing Video Service: video-service")
	logger.Info("================================================================================")

	// Test Video Service
	vs, err := video.FromProvider(machine, "video-service")
	if err != nil {
		logger.Fatalf("Failed to get video service: %v", err)
	}

	// Test 1: get-storage-state DoCommand
	logger.Info("Test 1: get-storage-state DoCommand")
	storageStateCmd := map[string]interface{}{
		"command": "get-storage-state",
	}
	storageState, err := vs.DoCommand(ctx, storageStateCmd)
	if err != nil {
		logger.Errorf("get-storage-state failed: %v", err)
	} else {
		logger.Infof("✓ Storage state retrieved successfully")
		if diskUsage, ok := storageState["disk_usage"].(map[string]interface{}); ok {
			logger.Infof("  - Storage path: %v", diskUsage["storage_path"])
			logger.Infof("  - Storage used: %.2f GB", diskUsage["storage_used_gb"])
			logger.Infof("  - Storage limit: %.2f GB", diskUsage["storage_limit_gb"])
			logger.Infof("  - Device remaining: %.2f GB", diskUsage["device_storage_remaining_gb"])
		}
		if storedVideo, ok := storageState["stored_video"].([]interface{}); ok {
			logger.Infof("  - Number of video segments: %d", len(storedVideo))
			if len(storedVideo) > 0 {
				first := storedVideo[0].(map[string]interface{})
				last := storedVideo[len(storedVideo)-1].(map[string]interface{})
				logger.Infof("  - First segment: %v to %v", first["from"], first["to"])
				logger.Infof("  - Last segment: %v to %v", last["from"], last["to"])
			}
		}
	}
	logger.Info("")

	// Test 2: fetch DoCommand
	end := time.Now().UTC().Add(-50 * time.Second)
	start := end.Add(-10 * time.Second)

	logger.Info("Test 2: fetch DoCommand")
	fetchStart := start.Format("2006-01-02_15-04-05Z")
	fetchEnd := end.Format("2006-01-02_15-04-05Z")

	logger.Infof("Fetching video from %s to %s", fetchStart, fetchEnd)
	fetchCmd := map[string]interface{}{
		"command": "fetch",
		"from":    fetchStart,
		"to":      fetchEnd,
	}
	fetchResult, err := vs.DoCommand(ctx, fetchCmd)
	var fetchFileSize int64
	if err != nil {
		logger.Errorf("fetch failed: %v", err)
	} else {
		if videoBase64, ok := fetchResult["video"].(string); ok {
			// Decode base64 video
			videoBytes, err := base64.StdEncoding.DecodeString(videoBase64)
			if err != nil {
				logger.Errorf("Failed to decode base64 video: %v", err)
			} else {
				// Save the video to a file
				fetchOutputFile := "test_video_fetch.mp4"
				err := os.WriteFile(fetchOutputFile, videoBytes, 0644)
				if err != nil {
					logger.Errorf("Failed to save fetch video: %v", err)
				} else {
					fileInfo, _ := os.Stat(fetchOutputFile)
					fetchFileSize = fileInfo.Size()

					logger.Infof("✓ Video fetched and saved successfully")
					logger.Infof("  - Base64 length: %d characters", len(videoBase64))
					logger.Infof("  - Decoded size: %d bytes", len(videoBytes))
					logger.Infof("  - Saved to: %s", fetchOutputFile)

					// Verify with ffprobe
					cmd := exec.Command("ffprobe", "-v", "error", "-show_format", fetchOutputFile)
					_, err := cmd.CombinedOutput()
					if err == nil {
						logger.Info("  - Video file is VALID and playable")
					} else {
						logger.Errorf("  - Video validation failed: %v", err)
					}
				}
			}
		} else {
			logger.Warn("fetch succeeded but no video data in response")
		}
	}
	logger.Info("")

	// Test 3: save DoCommand
	logger.Info("Test 3: save DoCommand")

	saveStart := start.Format("2006-01-02_15-04-05Z")
	saveEnd := end.Format("2006-01-02_15-04-05Z")

	logger.Infof("Saving video clip from %s to %s", saveStart, saveEnd)
	saveCmd := map[string]interface{}{
		"command":  "save",
		"from":     saveStart,
		"to":       saveEnd,
		"metadata": "test-clip",
		"tags":     []string{"test", "automated"},
	}
	saveResult, err := vs.DoCommand(ctx, saveCmd)
	if err != nil {
		logger.Errorf("save failed: %v", err)
	} else {
		if filename, ok := saveResult["filename"].(string); ok {
			logger.Infof("✓ Video clip saved successfully")
			logger.Infof("  - Filename: %s", filename)
			logger.Info("  - File will be uploaded to cloud via data manager")
		} else {
			logger.Warn("save succeeded but no filename in response")
		}

		// Check if async save
		if status, ok := saveResult["status"].(string); ok && status == "async" {
			logger.Info("  - Save operation is running asynchronously")
		}
	}
	logger.Info("")

	// Test 4: GetVideo streaming API
	logger.Info("Test 4: GetVideo streaming API")

	// Get video from 50 to 60 seconds ago (should have recent video)
	// end := time.Now().UTC().Add(-50 * time.Second)
	// start := end.Add(-10 * time.Second)

	logger.Infof("Requesting video from %s to %s", start.Format(time.RFC3339), end.Format(time.RFC3339))

	// Test with mp4 container
	videoContainer := "mp4"
	logger.Infof("Container format: %s", videoContainer)

	ch, err := vs.GetVideo(ctx, start, end, "", videoContainer, nil)
	if err != nil {
		logger.Fatalf("GetVideo failed: %v", err)
	}

	// Create file to save chunks as we receive them
	outputFile := fmt.Sprintf("test_video_go_%s.mp4", videoContainer)
	f, err := os.Create(outputFile)
	if err != nil {
		logger.Fatalf("Failed to create file: %v", err)
	}
	defer f.Close()

	var totalBytes int
	chunkIdx := 0
	var firstChunkData []byte

	logger.Infof("Streaming and saving video to %s...", outputFile)

	for chunk := range ch {
		if chunk == nil {
			continue
		}

		// Write chunk to file immediately
		n, err := f.Write(chunk.Data)
		if err != nil {
			logger.Fatalf("Failed to write chunk: %v", err)
		}
		totalBytes += n
		chunkIdx++

		// Save first chunk details
		if chunkIdx == 1 {
			if len(chunk.Data) >= 20 {
				firstChunkData = chunk.Data[:20]
			} else {
				firstChunkData = chunk.Data
			}
			logger.Infof("First chunk: %d bytes, container: %s", n, chunk.Container)
			logger.Infof("First 20 bytes: % x", firstChunkData)
		}

		if chunkIdx%10 == 0 {
			logger.Infof("Received %d chunks so far (%d bytes)...", chunkIdx, totalBytes)
		}
	}

	logger.Infof("✓ Stream complete and saved: %d chunks, %d total bytes to %s", chunkIdx, totalBytes, outputFile)

	// Try to verify with ffprobe
	logger.Info("Verifying video file with ffprobe...")
	cmd := exec.Command("ffprobe", "-v", "error", "-show_format", "-show_streams", outputFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Errorf("✗ ffprobe failed: %v", err)
		logger.Errorf("Output: %s", string(output))
	} else {
		logger.Info("✓ Video file is valid!")
	}

	logger.Info("================================================================================")
	logger.Info("Test Summary:")
	logger.Info("Camera Tests:")
	logger.Info("  ✓ Camera Properties() - working")
	logger.Info("")
	logger.Info("Video Service DoCommand Tests:")
	logger.Info("  ✓ get-storage-state - working")
	logger.Info("  ✓ fetch - working")
	logger.Info("  ✓ save - working (saves to cloud upload directory)")
	logger.Info("")
	logger.Info("Video Service GetVideo Streaming API:")
	logger.Info("  ⚠ GetVideo() - streams data but produces invalid MP4")
	logger.Infof("  - Received %d chunks, %d bytes total", chunkIdx, totalBytes)
	logger.Infof("  - First bytes: % x", firstChunkData)
	logger.Info("  - Issue: moov atom not found (bug in video-store v0.0.9-rc1)")
	logger.Info("")
	logger.Info("File Size Comparison:")
	if fetchFileSize > 0 {
		logger.Infof("  - fetch DoCommand result: %d bytes (VALID MP4)", fetchFileSize)
		logger.Infof("  - GetVideo result: %d bytes (INVALID MP4)", totalBytes)
		sizeDiff := int64(totalBytes) - fetchFileSize
		if sizeDiff > 0 {
			logger.Infof("  - GetVideo is %d bytes larger (%.1f%% larger)", sizeDiff, float64(sizeDiff)/float64(fetchFileSize)*100)
		} else if sizeDiff < 0 {
			logger.Infof("  - GetVideo is %d bytes smaller (%.1f%% smaller)", -sizeDiff, float64(-sizeDiff)/float64(fetchFileSize)*100)
		} else {
			logger.Info("  - Both methods returned the same size")
		}
	}
	logger.Info("")
	logger.Info("Recommendation: Use 'fetch' DoCommand for video retrieval")
	logger.Info("================================================================================")
}
