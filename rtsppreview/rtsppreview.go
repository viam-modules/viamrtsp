// Package rtsppreview provides shared helpers for discovery services to fetch a
// preview image from an RTSP camera and format it as a data URL. It is used by
// the onvif and garmin discovery services' preview DoCommand.
package rtsppreview

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/viam-modules/viamrtsp"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	rutils "go.viam.com/rdk/utils"
)

const (
	// rtspPollTimeout bounds how long we poll a throwaway RTSP camera for a frame.
	rtspPollTimeout = 5 * time.Second
	// rtspImageInterval is how often we retry grabbing a frame during polling.
	rtspImageInterval = 100 * time.Millisecond
)

// ParsePreviewCommand extracts the rtsp_address from a preview DoCommand's attributes.
func ParsePreviewCommand(command map[string]interface{}) (string, error) {
	attributes, ok := command["attributes"].(map[string]interface{})
	if !ok {
		return "", errors.New("attributes is missing or not a map")
	}
	rtspURL, ok := attributes["rtsp_address"].(string)
	if !ok {
		return "", errors.New("rtsp_address cannot be empty")
	}
	return rtspURL, nil
}

// FormatDataURL formats the image data and content type into a data URL.
func FormatDataURL(contentType string, imageBytes []byte) string {
	base64Image := base64.StdEncoding.EncodeToString(imageBytes)
	return fmt.Sprintf("data:%s;base64,%s", contentType, base64Image)
}

// FetchImageFromRTSPURL spins up a throwaway RTSP camera, polls it for a single
// JPEG frame, and returns that frame as a base64 data URL.
func FetchImageFromRTSPURL(ctx context.Context, logger logging.Logger, rtspURL string) (string, error) {
	// Wrap viamrtsp.Config in a resource.Config.
	rtspConfig := viamrtsp.Config{
		Address: rtspURL,
	}
	resourceConfig := resource.Config{
		Name:                generateUniqueName("tmp-camera"),
		API:                 camera.API,
		Model:               viamrtsp.ModelAgnostic,
		ConvertedAttributes: &rtspConfig,
	}

	cam, err := viamrtsp.NewRTSPCamera(ctx, nil, resourceConfig, logger)
	if err != nil {
		return "", fmt.Errorf("failed to create RTSP camera: %w", err)
	}
	defer func() {
		if closeErr := cam.Close(ctx); closeErr != nil {
			logger.Warnf("failed to close camera: %v", closeErr)
		}
	}()

	ticker := time.NewTicker(rtspImageInterval)
	defer ticker.Stop()
	timeoutChan := time.After(rtspPollTimeout)
	var imageErr error
	for {
		select {
		case <-ticker.C:
			// Attempt to get an image from the RTSP camera.
			namedImages, _, err := cam.Images(ctx, nil, nil)
			if err == nil {
				if len(namedImages) != 1 {
					imageErr = fmt.Errorf("expected exactly 1 image, got %d", len(namedImages))
					continue
				}
				namedImage := namedImages[0]
				imgBytes, err := namedImage.Bytes(ctx)
				if err != nil {
					imageErr = fmt.Errorf("failed to get image bytes: %w", err)
					continue
				}
				if namedImage.MimeType() != rutils.MimeTypeJPEG {
					imageErr = fmt.Errorf("expected %s, got %s", rutils.MimeTypeJPEG, namedImage.MimeType())
					continue
				}

				logger.Debugf("Received image with metadata: %v, size: %d bytes", namedImage.SourceName, len(imgBytes))
				return FormatDataURL(rutils.MimeTypeJPEG, imgBytes), nil
			}
			imageErr = err
			logger.Debugf("Failed to get image from RTSP camera: %v", imageErr)
		case <-timeoutChan:
			return "", fmt.Errorf("timeout while trying to get image from RTSP camera: %w", imageErr)
		case <-ctx.Done():
			return "", fmt.Errorf("context canceled while fetching image from RTSP camera: %w", ctx.Err())
		}
	}
}

// generateUniqueName creates a unique name by appending a timestamp.
func generateUniqueName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
