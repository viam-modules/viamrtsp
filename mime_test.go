package viamrtsp

import (
	"encoding/binary"
	"testing"

	"go.viam.com/rdk/logging"
	"go.viam.com/test"
)

//nolint:dupl
func TestJPEGConvert(t *testing.T) {
	t.Run("valid YUV420P frame succeeds", func(t *testing.T) {
		width, height := 640, 480
		frame := createTestYUV420PFrame(width, height)
		test.That(t, frame, test.ShouldNotBeNil)
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
		test.That(t, frame, test.ShouldNotBeNil)
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
		frame := createInvalidFrame()
		test.That(t, frame, test.ShouldNotBeNil)
		defer freeFrame(frame)
		logger := logging.NewDebugLogger("mime_test")
		mh := newMimeHandler(logger)
		bytes, metadata, err := mh.convertJPEG(frame)
		test.That(t, err, test.ShouldNotBeNil)
		test.That(t, err.Error(), test.ShouldContainSubstring, "failed to open MJPEG encoder")
		test.That(t, bytes, test.ShouldBeNil)
		test.That(t, metadata.MimeType, test.ShouldBeEmpty)
	})
}

//nolint:dupl
func TestYUYVConvert(t *testing.T) {
	t.Run("valid YUV420P frame succeeds", func(t *testing.T) {
		width, height := 640, 480
		frame := createTestYUV420PFrame(width, height)
		test.That(t, frame, test.ShouldNotBeNil)
		defer freeFrame(frame)
		fillDummyYUV420PData(frame)
		logger := logging.NewDebugLogger("mime_test")
		mh := newMimeHandler(logger)
		bytes, metadata, err := mh.convertYUYV(frame)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, bytes, test.ShouldNotBeNil)
		test.That(t, len(bytes), test.ShouldBeGreaterThan, 0)
		test.That(t, metadata, test.ShouldNotBeEmpty)
	})

	t.Run("valid YUVJ420P frame succeeds", func(t *testing.T) {
		width, height := 640, 480
		frame := createTestYUVJ420PFrame(width, height)
		test.That(t, frame, test.ShouldNotBeNil)
		defer freeFrame(frame)
		fillDummyYUV420PData(frame)
		logger := logging.NewDebugLogger("mime_test")
		mh := newMimeHandler(logger)
		bytes, metadata, err := mh.convertYUYV(frame)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, bytes, test.ShouldNotBeNil)
		test.That(t, len(bytes), test.ShouldBeGreaterThan, 0)
		test.That(t, metadata, test.ShouldNotBeEmpty)
	})

	t.Run("invalid frame fails", func(t *testing.T) {
		frame := createInvalidFrame()
		test.That(t, frame, test.ShouldNotBeNil)
		defer freeFrame(frame)
		logger := logging.NewDebugLogger("mime_test")
		mh := newMimeHandler(logger)
		bytes, metadata, err := mh.convertYUYV(frame)
		test.That(t, err, test.ShouldNotBeNil)
		test.That(t, err.Error(), test.ShouldContainSubstring, "failed to allocate buffer for YUYV")
		test.That(t, bytes, test.ShouldBeNil)
		test.That(t, metadata.MimeType, test.ShouldBeEmpty)
	})

	t.Run("test yuyv magic header", func(t *testing.T) {
		origWidth := 640
		origHeight := 480
		header := packYUYVHeader(origWidth, origHeight)
		test.That(t, header, test.ShouldNotBeNil)
		test.That(t, len(header), test.ShouldEqual, 12)
		test.That(t, header[0], test.ShouldEqual, 'Y')
		test.That(t, header[1], test.ShouldEqual, 'U')
		test.That(t, header[2], test.ShouldEqual, 'Y')
		test.That(t, header[3], test.ShouldEqual, 'V')
		parsedWidth := int(binary.BigEndian.Uint32(header[4:8]))
		test.That(t, parsedWidth, test.ShouldEqual, origWidth)
		parsedHeight := int(binary.BigEndian.Uint32(header[8:12]))
		test.That(t, parsedHeight, test.ShouldEqual, origHeight)
	})
}