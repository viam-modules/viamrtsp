package viamrtsp

/*
#cgo pkg-config: libavutil libavcodec libswscale
#include <libavcodec/avcodec.h>
#include <libavutil/error.h>
#include <libavutil/opt.h>
#include <libavutil/dict.h>
#include <libavutil/frame.h>
#include <libavutil/imgutils.h>
#include <libavutil/pixfmt.h>
#include <libswscale/swscale.h>
#include <stdlib.h>
*/
import "C"

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	rutils "go.viam.com/rdk/utils"
)

const (
	yuyvMagicString   = "YUYV"
	yuyvBytesPerPixel = 2
	rgbaMagicString   = "RGBA"
	rgbaBytesPerPixel = 4
)

type mimeHandler struct {
	logger     logging.Logger
	jpegEnc    *C.AVCodecContext
	yuyvFrame  *C.AVFrame
	yuyvSwsCtx *C.struct_SwsContext
	rgbaFrame  *C.AVFrame
	rgbaSwsCtx *C.struct_SwsContext
	currentPTS int
	jpegMu     sync.Mutex
	yuyvMu     sync.Mutex
	rgbaMu     sync.Mutex
}

func newMimeHandler(logger logging.Logger) *mimeHandler {
	return &mimeHandler{
		logger:     logger,
		currentPTS: 0,
	}
}

func (mh *mimeHandler) convertJPEG(frame *C.AVFrame) ([]byte, camera.ImageMetadata, error) {
	if frame == nil {
		return nil, camera.ImageMetadata{}, errors.New("frame input is nil, cannot convert to JPEG")
	}
	if mh.jpegEnc == nil || frame.width != mh.jpegEnc.width || frame.height != mh.jpegEnc.height {
		if err := mh.initJPEGEncoder(frame); err != nil {
			return nil, camera.ImageMetadata{}, err
		}
	}
	if mh.jpegEnc == nil {
		return nil, camera.ImageMetadata{}, errors.New("failed to create encoder or destination frame")
	}
	// Allocate a fresh packet to prevent issues with concurrent jpeg encoding
	pkt := C.av_packet_alloc()
	if pkt == nil {
		return nil, camera.ImageMetadata{}, errors.New("failed to allocate packet")
	}
	defer C.av_packet_free(&pkt)
	frame.pts = C.int64_t(mh.currentPTS)
	// If this reaches max int, it will wrap around to 0
	mh.currentPTS++
	res := C.avcodec_send_frame(mh.jpegEnc, frame)
	if res < 0 {
		return nil, camera.ImageMetadata{}, newAvError(res, "failed to send frame to MJPEG encoder")
	}
	res = C.avcodec_receive_packet(mh.jpegEnc, pkt)
	if res < 0 {
		return nil, camera.ImageMetadata{}, newAvError(res, "failed to receive packet from MJPEG encoder")
	}
	// There is no need to create a frame for the packet, as the packet already contains the data
	dataGo := C.GoBytes(unsafe.Pointer(pkt.data), pkt.size)

	return dataGo, camera.ImageMetadata{
		MimeType: rutils.MimeTypeJPEG,
	}, nil
}

func (mh *mimeHandler) initJPEGEncoder(frame *C.AVFrame) error {
	// Lock to prevent modifying encoder while it is being used concurrently.
	mh.jpegMu.Lock()
	defer mh.jpegMu.Unlock()
	mh.logger.Info("creating MJPEG encoder with frame size: ", frame.width, "x", frame.height)
	if mh.jpegEnc != nil {
		C.avcodec_free_context(&mh.jpegEnc)
	}
	codec := C.avcodec_find_encoder(C.AV_CODEC_ID_MJPEG)
	if codec == nil {
		return errors.New("failed to find MJPEG encoder")
	}
	mh.jpegEnc = C.avcodec_alloc_context3(codec)
	mh.jpegEnc.width = frame.width
	mh.jpegEnc.height = frame.height
	mh.jpegEnc.pix_fmt = C.AV_PIX_FMT_YUVJ420P
	// We don't care about accurate timestamps for still frames
	mh.jpegEnc.time_base = C.AVRational{num: 1, den: 1}
	var opts *C.AVDictionary
	// Equivalent to 75% quality
	qscaleKey := C.CString("qscale")
	qscaleValue := C.CString("8")
	defer func() {
		C.free(unsafe.Pointer(qscaleValue))
		C.free(unsafe.Pointer(qscaleKey))
		C.av_dict_free(&opts)
	}()
	res := C.av_dict_set(&opts, qscaleKey, qscaleValue, 0)
	if res < 0 {
		return newAvError(res, "failed to set qscale option")
	}
	if res := C.avcodec_open2(mh.jpegEnc, codec, &opts); res < 0 {
		C.avcodec_free_context(&mh.jpegEnc)
		return newAvError(res, "failed to open MJPEG encoder")
	}

	return nil
}

func (mh *mimeHandler) convertYUYV(frame *C.AVFrame) ([]byte, camera.ImageMetadata, error) {
	return mh.convertPixelFormat(
		frame,
		yuyvMagicString,
		&mh.yuyvSwsCtx,
		&mh.yuyvFrame,
		&mh.yuyvMu,
		mh.initYUYVContext,
		yuyvBytesPerPixel,
		mimeTypeYUYV,
	)
}

func (mh *mimeHandler) convertRGBA(frame *C.AVFrame) ([]byte, camera.ImageMetadata, error) {
	return mh.convertPixelFormat(
		frame,
		rgbaMagicString,
		&mh.rgbaSwsCtx,
		&mh.rgbaFrame,
		&mh.rgbaMu,
		mh.initRGBAContext,
		rgbaBytesPerPixel,
		rutils.MimeTypeRawRGBA,
	)
}

// convertPixelFormat handles the common logic for converting frames to different pixel formats
func (mh *mimeHandler) convertPixelFormat(
	frame *C.AVFrame,
	format string,
	swsCtx **C.struct_SwsContext,
	dstFrame **C.AVFrame,
	mu *sync.Mutex,
	initContext func(*C.AVFrame) error,
	bytesPerPixel int,
	mimeType string,
) ([]byte, camera.ImageMetadata, error) {
	if frame == nil {
		return nil, camera.ImageMetadata{}, fmt.Errorf("frame input is nil, cannot convert to %s", format)
	}

	mu.Lock()
	defer mu.Unlock()

	if *swsCtx == nil || frame.width != (*dstFrame).width || frame.height != (*dstFrame).height {
		if err := initContext(frame); err != nil {
			return nil, camera.ImageMetadata{}, err
		}
	}

	if *swsCtx == nil {
		return nil, camera.ImageMetadata{}, fmt.Errorf("failed to create %s converter", format)
	}
	if *dstFrame == nil {
		return nil, camera.ImageMetadata{}, fmt.Errorf("failed to create %s destination frame", format)
	}

	res := C.sws_scale(
		*swsCtx,
		(**C.uint8_t)(unsafe.Pointer(&frame.data[0])),
		(*C.int)(unsafe.Pointer(&frame.linesize[0])),
		0,
		frame.height,
		(**C.uint8_t)(unsafe.Pointer(&(*dstFrame).data[0])),
		(*C.int)(unsafe.Pointer(&(*dstFrame).linesize[0])),
	)
	if res < 0 {
		return nil, camera.ImageMetadata{}, newAvError(res, "failed to convert frame to "+format)
	}

	dataSize := int((*dstFrame).width) * int((*dstFrame).height) * bytesPerPixel
	header := packHeader(format, int((*dstFrame).width), int((*dstFrame).height))
	data := make([]byte, len(header)+dataSize)
	copy(data[0:], header)
	C.memcpy(unsafe.Pointer(&data[len(header)]), unsafe.Pointer((*dstFrame).data[0]), C.size_t(dataSize))

	return data, camera.ImageMetadata{
		MimeType: mimeType,
	}, nil
}

func (mh *mimeHandler) initYUYVContext(frame *C.AVFrame) error {
	return mh.initPixelFormatContext(frame, C.AV_PIX_FMT_YUYV422, &mh.yuyvSwsCtx, &mh.yuyvFrame)
}

func (mh *mimeHandler) initRGBAContext(frame *C.AVFrame) error {
	return mh.initPixelFormatContext(frame, C.AV_PIX_FMT_RGBA, &mh.rgbaSwsCtx, &mh.rgbaFrame)
}

// initPixelFormatContext is a helper function that initializes the conversion context and destination frame
// for pixel format conversions. It handles cleanup of existing contexts/frames and allocation of new ones.
//
// Parameters:
// - frame: Source AVFrame containing the input image
// - pixFmt: Target pixel format to convert to
// - swsCtxPtr: Pointer to SwsContext pointer that will be initialized
// - dstFrame: Pointer to AVFrame pointer that will be initialized
//
// Returns error if any allocation or initialization fails
func (mh *mimeHandler) initPixelFormatContext(
	frame *C.AVFrame,
	pixFmt C.enum_AVPixelFormat,
	swsCtxPtr **C.struct_SwsContext,
	dstFrame **C.AVFrame,
) error {
	mh.logger.Infof("creating sws context with frame size: %dx%d for format %d", frame.width, frame.height, pixFmt)

	if *swsCtxPtr != nil {
		C.sws_freeContext(*swsCtxPtr)
		*swsCtxPtr = nil
	}
	if *dstFrame != nil {
		C.av_frame_free(dstFrame)
		*dstFrame = nil
	}

	newFrame := C.av_frame_alloc()
	if newFrame == nil {
		return errors.New("failed to allocate frame")
	}

	newFrame.width = frame.width
	newFrame.height = frame.height
	newFrame.format = C.int(pixFmt)

	if res := C.av_frame_get_buffer(newFrame, 32); res < 0 {
		C.av_frame_free(&newFrame)
		return newAvError(res, "failed to allocate buffer for frame")
	}

	newSwsCtx := C.sws_getContext(
		frame.width, frame.height, pixFmt,
		frame.width, frame.height, pixFmt,
		C.SWS_FAST_BILINEAR, nil, nil, nil,
	)

	if newSwsCtx == nil {
		C.av_frame_free(&newFrame)
		return errors.New("failed to create converter")
	}

	*dstFrame = newFrame
	*swsCtxPtr = newSwsCtx
	return nil
}

func (mh *mimeHandler) close() {
	if mh.jpegEnc != nil {
		C.avcodec_free_context(&mh.jpegEnc)
		mh.jpegEnc = nil
	}
	if mh.yuyvSwsCtx != nil {
		C.sws_freeContext(mh.yuyvSwsCtx)
		mh.yuyvSwsCtx = nil
	}
	if mh.yuyvFrame != nil {
		C.av_frame_free(&mh.yuyvFrame)
		mh.yuyvFrame = nil
	}
	if mh.rgbaSwsCtx != nil {
		C.sws_freeContext(mh.rgbaSwsCtx)
		mh.rgbaSwsCtx = nil
	}
	if mh.rgbaFrame != nil {
		C.av_frame_free(&mh.rgbaFrame)
		mh.rgbaFrame = nil
	}
}

// packHeader creates a header for image data with the given format, width and height.
// The header structure is as follows:
// - Format string (4 bytes): A fixed string indicating the format (e.g., "YUYV" or "RGBA").
// - Width (4 bytes): The width of the image, stored in big-endian format.
// - Height (4 bytes): The height of the image, stored in big-endian format.
func packHeader(format string, width, height int) []byte {
	headerSize := 12
	headerBytes := make([]byte, headerSize)
	copy(headerBytes[0:4], []byte(format))
	binary.BigEndian.PutUint32(headerBytes[4:8], uint32(width))
	binary.BigEndian.PutUint32(headerBytes[8:12], uint32(height))
	return headerBytes
}

// packYUYVHeader creates a header for YUYV data with the given width and height.
func packYUYVHeader(width, height int) []byte {
	return packHeader(yuyvMagicString, width, height)
}

// packRGBAHeader creates a header for RGBA data with the given width and height.
func packRGBAHeader(width, height int) []byte {
	return packHeader(rgbaMagicString, width, height)
}
