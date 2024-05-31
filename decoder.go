package viamrtsp

/*
#cgo pkg-config: libavcodec libavutil libswscale
#include <libavcodec/avcodec.h>
#include <libavutil/imgutils.h>
#include <libavutil/error.h>
#include <libswscale/swscale.h>
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"image"
	"unsafe"

	"github.com/pkg/errors"
	"go.viam.com/rdk/logging"
)

// decoder is a generic FFmpeg decoder.
type decoder struct {
	logger   logging.Logger
	codecCtx *C.AVCodecContext
	srcFrame *C.AVFrame
}

type videoCodec int

const (
	// Unknown indicates an error when no available video codecs could be identified
	Unknown videoCodec = iota
	// Agnostic indicates that a discrete video codec has yet to be identified
	Agnostic
	// H264 indicates the h264 video codec
	H264
	// H265 indicates the h265 video codec
	H265
	// MJPEG indicates the mjpeg video codec
	MJPEG
)

func (vc videoCodec) String() string {
	switch vc {
	case Agnostic:
		return "Agnostic"
	case H264:
		return "H264"
	case H265:
		return "H265"
	case MJPEG:
		return "MJPEG"
	default:
		return "Unknown"
	}
}

func frameData(frame *C.AVFrame) **C.uint8_t {
	return (**C.uint8_t)(unsafe.Pointer(&frame.data[0]))
}

func frameLineSize(frame *C.AVFrame) *C.int {
	return (*C.int)(unsafe.Pointer(&frame.linesize[0]))
}

// avError converts an AV error code to a AV error message string.
func avError(avErr C.int) string {
	var errbuf [C.AV_ERROR_MAX_STRING_SIZE]C.char
	if C.av_strerror(avErr, &errbuf[0], C.AV_ERROR_MAX_STRING_SIZE) < 0 {
		return fmt.Sprintf("Unknown error with code %d", avErr)
	}
	return C.GoString(&errbuf[0])
}

// SetLibAVLogLevelFatal sets libav errors to fatal log level
// to cut down on log spam
func SetLibAVLogLevelFatal() {
	C.av_log_set_level(C.AV_LOG_FATAL)
}

// newDecoder creates a new decoder for the given codec.
func newDecoder(codecID C.enum_AVCodecID, logger logging.Logger) (*decoder, error) {
	codec := C.avcodec_find_decoder(codecID)
	if codec == nil {
		return nil, errors.New("avcodec_find_decoder() failed")
	}

	codecCtx := C.avcodec_alloc_context3(codec)
	if codecCtx == nil {
		return nil, errors.New("avcodec_alloc_context3() failed")
	}

	res := C.avcodec_open2(codecCtx, codec, nil)
	if res < 0 {
		C.avcodec_close(codecCtx)
		return nil, errors.New("avcodec_open2() failed")
	}

	srcFrame := C.av_frame_alloc()
	if srcFrame == nil {
		C.avcodec_close(codecCtx)
		return nil, errors.New("av_frame_alloc() failed")
	}

	return &decoder{
		logger:   logger,
		codecCtx: codecCtx,
		srcFrame: srcFrame,
	}, nil
}

// newH264Decoder creates a new H264 decoder.
func newH264Decoder(logger logging.Logger) (*decoder, error) {
	return newDecoder(C.AV_CODEC_ID_H264, logger)
}

// newH265Decoder creates a new H265 decoder.
func newH265Decoder(logger logging.Logger) (*decoder, error) {
	return newDecoder(C.AV_CODEC_ID_H265, logger)
}

// close closes the decoder.
func (d *decoder) close() {
	C.av_frame_free(&d.srcFrame)
	C.avcodec_close(d.codecCtx)
}

func (d *decoder) decode(nalu []byte) (image.Image, error) {
	nalu = append(H2645StartCode(), nalu...)

	// send frame to decoder
	var avPacket C.AVPacket
	avPacket.data = (*C.uint8_t)(C.CBytes(nalu))
	defer C.free(unsafe.Pointer(avPacket.data))
	avPacket.size = C.int(len(nalu))
	res := C.avcodec_send_packet(d.codecCtx, &avPacket)
	if res < 0 {
		return nil, nil
	}

	// receive frame if available
	res = C.avcodec_receive_frame(d.codecCtx, d.srcFrame)
	if res < 0 {
		return nil, nil
	}

	pixDataSize := 4 * int(d.srcFrame.width) * int(d.srcFrame.height)
	pixData := C.malloc(C.size_t(pixDataSize))
	defer C.free(pixData)

	swsCtx := C.sws_getContext(d.srcFrame.width, d.srcFrame.height, C.AV_PIX_FMT_YUV420P,
		d.srcFrame.width, d.srcFrame.height, C.AV_PIX_FMT_RGBA, C.SWS_BILINEAR, nil, nil, nil)
	if swsCtx == nil {
		return nil, errors.New("sws_getContext() err")
	}
	defer C.sws_freeContext(swsCtx)

	// convert frame from YUV420 to RGB
	lineSizes := frameLineSizeForRGBA(d.srcFrame.width)
	res = C.sws_scale(swsCtx, frameData(d.srcFrame), frameLineSize(d.srcFrame),
		0, d.srcFrame.height, (**C.uint8_t)(unsafe.Pointer(&pixData)), &lineSizes[0])
	if res < 0 {
		return nil, errors.New("sws_scale() err")
	}

	pixDataGo := C.GoBytes(pixData, C.int(pixDataSize))

	return &image.RGBA{
		Pix:    pixDataGo,
		Stride: 4 * int(d.srcFrame.width),
		Rect: image.Rectangle{
			Max: image.Point{int(d.srcFrame.width), int(d.srcFrame.height)},
		},
	}, nil
}

// frameLineSizeForRGBA returns the line size array for an RGBA frame
func frameLineSizeForRGBA(width C.int) [4]C.int {
	return [4]C.int{4 * width, 0, 0, 0}
}
