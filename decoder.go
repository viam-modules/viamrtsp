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

	// We allocate a new destination frame for every frame we decode.
	dstFrame := C.av_frame_alloc()
	defer C.av_frame_free(&dstFrame)
	dstFrame.format = C.AV_PIX_FMT_RGBA
	dstFrame.width = d.srcFrame.width
	dstFrame.height = d.srcFrame.height
	dstFrame.color_range = C.AVCOL_RANGE_JPEG
	res = C.av_frame_get_buffer(dstFrame, 1)
	if res < 0 {
		return nil, errors.New("av_frame_get_buffer() err")
	}

	swsCtx := C.sws_getContext(d.srcFrame.width, d.srcFrame.height, C.AV_PIX_FMT_YUV420P,
		dstFrame.width, dstFrame.height, (int32)(dstFrame.format), C.SWS_BILINEAR, nil, nil, nil)
	if swsCtx == nil {
		return nil, errors.New("sws_getContext() err")
	}
	defer C.sws_freeContext(swsCtx)

	// convert frame from YUV420 to RGB
	res = C.sws_scale(swsCtx, frameData(d.srcFrame), frameLineSize(d.srcFrame),
		0, d.srcFrame.height, frameData(dstFrame), frameLineSize(dstFrame))
	if res < 0 {
		return nil, errors.New("sws_scale() err")
	}

	// Copy the frame data into a Go byte slice. This avoids filling the go image with a dangling
	// pointer to C memory and allows the go garbage collector to manage the memory for us.
	dstFrameSize := C.av_image_get_buffer_size((int32)(dstFrame.format), dstFrame.width, dstFrame.height, 1)
	dataGo := C.GoBytes(unsafe.Pointer(dstFrame.data[0]), dstFrameSize)

	return &image.RGBA{
		Pix:    dataGo,
		Stride: 4 * (int)(dstFrame.width),
		Rect: image.Rectangle{
			Max: image.Point{(int)(dstFrame.width), (int)(dstFrame.height)},
		},
	}, nil
}
