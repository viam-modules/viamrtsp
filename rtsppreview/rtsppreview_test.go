package rtsppreview

import (
	"strings"
	"testing"

	"go.viam.com/test"
)

func TestParsePreviewCommand(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		rtspURL, err := ParsePreviewCommand(map[string]interface{}{
			"command": "preview",
			"attributes": map[string]interface{}{
				"rtsp_address": "rtsp://host:8554/stream",
			},
		})
		test.That(t, err, test.ShouldBeNil)
		test.That(t, rtspURL, test.ShouldEqual, "rtsp://host:8554/stream")
	})

	t.Run("missing attributes", func(t *testing.T) {
		_, err := ParsePreviewCommand(map[string]interface{}{"command": "preview"})
		test.That(t, err, test.ShouldNotBeNil)
	})

	t.Run("missing rtsp_address", func(t *testing.T) {
		_, err := ParsePreviewCommand(map[string]interface{}{
			"attributes": map[string]interface{}{},
		})
		test.That(t, err, test.ShouldNotBeNil)
	})
}

func TestFormatDataURL(t *testing.T) {
	// "hello" base64-encodes to aGVsbG8=
	dataURL := FormatDataURL("image/jpeg", []byte("hello"))
	test.That(t, dataURL, test.ShouldEqual, "data:image/jpeg;base64,aGVsbG8=")
}

func TestGenerateUniqueName(t *testing.T) {
	prefix := "tmp-camera"
	n1 := generateUniqueName(prefix)
	n2 := generateUniqueName(prefix)
	test.That(t, strings.HasPrefix(n1, prefix), test.ShouldBeTrue)
	test.That(t, strings.HasPrefix(n2, prefix), test.ShouldBeTrue)
	test.That(t, n1, test.ShouldNotEqual, n2)
}
