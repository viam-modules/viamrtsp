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
	"sync"
	"unsafe"

	"github.com/pkg/errors"
	"go.viam.com/rdk/logging"
)

// FrameWrapper wraps a C AVFrame and safely manages its memory in Go.
type FrameWrapper struct {
	frame *C.AVFrame
	freed bool
	mu    sync.Mutex
}

// newFrameWrapper creates a new FrameWrapper.
func newFrameWrapper(frame *C.AVFrame) *FrameWrapper {
	return &FrameWrapper{frame: frame}
}

// Data provides access to the underlying frame data
func (fw *FrameWrapper) Data() []uint8 {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.freed {
		panic("attempt to access freed memory")
	}

	size := C.av_image_get_buffer_size((int32)(fw.frame.format), fw.frame.width, fw.frame.height, 1)
	return (*[1 << 30]uint8)(unsafe.Pointer(fw.frame.data[0]))[:size:size]
}

// Close releases the memory associated with the frame.
func (fw *FrameWrapper) Close() {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.freed {
		return
	}

	C.av_frame_free(&fw.frame)
	fw.freed = true
}

// decoder is a generic FFmpeg decoder.
type decoder struct {
	logger      logging.Logger
	codecCtx    *C.AVCodecContext
	srcFrame    *FrameWrapper
	swsCtx      *C.struct_SwsContext
	dstFrame    *FrameWrapper
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
		srcFrame: newFrameWrapper(srcFrame),
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

// close closes the decoder and cleans up C resources.
func (d *decoder) close() {
	if d.dstFrame != nil {
		d.dstFrame.Close()
	}

	if d.swsCtx != nil {
		C.sws_freeContext(d.swsCtx)
	}

	if d.srcFrame != nil {
		d.srcFrame.Close()
	}

	if d.codecCtx != nil {
		C.avcodec_close(d.codecCtx)
	}
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

	res = C.avcodec_receive_frame(d.codecCtx, d.srcFrame.frame)
	if res < 0 {
		return nil, nil
	}

	if d.dstFrame == nil || d.dstFrame.frame.width != d.srcFrame.frame.width || d.dstFrame.frame.height != d.srcFrame.frame.height {
		if d.dstFrame != nil {
			d.dstFrame.Close()
		}

		if d.swsCtx != nil {
			C.sws_freeContext(d.swsCtx)
		}

		dstFrame := C.av_frame_alloc()
		dstFrame.format = C.AV_PIX_FMT_RGBA
		dstFrame.width = d.srcFrame.frame.width
		dstFrame.height = d.srcFrame.frame.height
		dstFrame.color_range = C.AVCOL_RANGE_JPEG
		res = C.av_frame_get_buffer(dstFrame, 1)
		if res < 0 {
			return nil, errors.New("av_frame_get_buffer() err")
		}

		d.dstFrame = newFrameWrapper(dstFrame)

		d.swsCtx = C.sws_getContext(d.srcFrame.frame.width, d.srcFrame.frame.height, C.AV_PIX_FMT_YUV420P,
			d.dstFrame.frame.width, d.dstFrame.frame.height, (int32)(d.dstFrame.frame.format), C.SWS_BILINEAR, nil, nil, nil)
		if d.swsCtx == nil {
			return nil, errors.New("sws_getContext() err")
		}
	}

	res = C.sws_scale(d.swsCtx, frameData(d.srcFrame.frame), frameLineSize(d.srcFrame.frame),
		0, d.srcFrame.frame.height, frameData(d.dstFrame.frame), frameLineSize(d.dstFrame.frame))
	if res < 0 {
		return nil, errors.New("sws_scale() err")
	}

	return &image.RGBA{
		Pix:    d.dstFrame.Data(),
		Stride: 4 * (int)(d.dstFrame.frame.width),
		Rect: image.Rectangle{
			Max: image.Point{(int)(d.dstFrame.frame.width), (int)(d.dstFrame.frame.height)},
		},
	}, nil
}
