package viamrtsp

import (
	"bytes"
	"image"
	"image/jpeg"

	"go.viam.com/rdk/components/camera"
	rutils "go.viam.com/rdk/utils"
)

func encodeToJPEG(img image.Image) ([]byte, camera.ImageMetadata, error) {
	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, img, nil); err != nil {
		return nil, camera.ImageMetadata{}, err
	}
	return buf.Bytes(), camera.ImageMetadata{
		MimeType: rutils.MimeTypeJPEG,
	}, nil
}
