package viamrtsp

import (
	"testing"

	"go.viam.com/rdk/logging"
	"go.viam.com/test"
)

func TestJPEGConvert(t *testing.T) {
	t.Run("valid YUV420P frame succeeds", func(t *testing.T) {
		width, height := 640, 480
		frame := createTestYUV420PFrame(width, height)
		if frame == nil {
			t.Fatalf("failed to allocate YUV420P frame")
		}
		defer freeFrame(frame)
		fillDummyYUV420PData(frame)
		logger := logging.NewDebugLogger("mime_test")
		mh := newMimeHandler(logger)
		bytes, metadata, err := mh.convertJPEG(frame)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, bytes, test.ShouldNotBeNil)
		test.That(t, len(bytes), test.ShouldBeGreaterThan, 0)
		test.That(t, metadata, test.ShouldNotBeEmpty)
	})

	t.Run("valid YUVJ420P frame succeeds", func(t *testing.T) {
		width, height := 640, 480
		frame := createTestYUVJ420PFrame(width, height)
		if frame == nil {
			t.Fatalf("failed to allocate YUV420P frame")
		}
		defer freeFrame(frame)
		fillDummyYUV420PData(frame)
		logger := logging.NewDebugLogger("mime_test")
		mh := newMimeHandler(logger)
		bytes, metadata, err := mh.convertJPEG(frame)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, bytes, test.ShouldNotBeNil)
		test.That(t, len(bytes), test.ShouldBeGreaterThan, 0)
		test.That(t, metadata, test.ShouldNotBeEmpty)
	})

	t.Run("invalid frame fails", func(t *testing.T) {
		frame := createBadFrame()
		if frame == nil {
			t.Fatalf("failed to allocate bad frame")
		}
		defer freeFrame(frame)
		logger := logging.NewDebugLogger("mime_test")
		mh := newMimeHandler(logger)
		bytes, metadata, err := mh.convertJPEG(frame)
		test.That(t, err, test.ShouldNotBeNil)
		test.That(t, bytes, test.ShouldBeNil)
		test.That(t, metadata.MimeType, test.ShouldBeEmpty)
	})
}
