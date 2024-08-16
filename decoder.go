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

// decoder is a generic FFmpeg decoder.
type decoder struct {
	logger   logging.Logger
	codecCtx *C.AVCodecContext
	srcFrame *C.AVFrame
	swsCtx   *C.struct_SwsContext
	dstFrame *C.AVFrame
	pool     *sync.Pool
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
func newDecoder(codecID C.enum_AVCodecID, avFramePool *sync.Pool, logger logging.Logger) (*decoder, error) {
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
		return nil, errors.New("av_frame_alloc() failed for source frame")
	}

	return &decoder{
		logger:   logger,
		codecCtx: codecCtx,
		srcFrame: srcFrame,
		pool:     avFramePool,
	}, nil
}

// newH264Decoder creates a new H264 decoder.
func newH264Decoder(avFramePool *sync.Pool, logger logging.Logger) (*decoder, error) {
	return newDecoder(C.AV_CODEC_ID_H264, avFramePool, logger)
}

// newH265Decoder creates a new H265 decoder.
func newH265Decoder(avFramePool *sync.Pool, logger logging.Logger) (*decoder, error) {
	return newDecoder(C.AV_CODEC_ID_H265, avFramePool, logger)
}

// close closes the decoder.
func (d *decoder) close() {
	if d.dstFrame != nil {
		C.av_frame_free(&d.dstFrame)
	}

	if d.swsCtx != nil {
		C.sws_freeContext(d.swsCtx)
	}

	C.av_frame_free(&d.srcFrame)
	C.avcodec_close(d.codecCtx)
}

// allocateFrame allocates a bytes slice using frame info.
func (d *decoder) allocateFrame() *[]byte {
	dstFrameSize := int(C.av_image_get_buffer_size((int32)(d.dstFrame.format), d.dstFrame.width, d.dstFrame.height, 1))
	dataGo := make([]byte, dstFrameSize)
	return &dataGo
}

func (d *decoder) decode(nalu []byte) (decoderOutput, error) {
	nalu = append(H2645StartCode(), nalu...)

	// send frame to decoder
	var avPacket C.AVPacket
	avPacket.data = (*C.uint8_t)(C.CBytes(nalu))
	defer C.free(unsafe.Pointer(avPacket.data))
	avPacket.size = C.int(len(nalu))
	res := C.avcodec_send_packet(d.codecCtx, &avPacket)
	if res < 0 {
		return decoderOutput{}, nil
	}

	// receive frame if available
	res = C.avcodec_receive_frame(d.codecCtx, d.srcFrame)
	if res < 0 {
		return decoderOutput{}, nil
	}

	// if frame size has changed, allocate needed objects
	if d.dstFrame == nil || d.dstFrame.width != d.srcFrame.width || d.dstFrame.height != d.srcFrame.height {
		if d.dstFrame != nil {
			C.av_frame_free(&d.dstFrame)
		}

		if d.swsCtx != nil {
			C.sws_freeContext(d.swsCtx)
		}

		d.dstFrame = C.av_frame_alloc()
		if d.dstFrame == nil {
			return decoderOutput{}, errors.New("av_frame_alloc() failed for destination frame")
		}
		d.dstFrame.format = C.AV_PIX_FMT_RGBA
		d.dstFrame.width = d.srcFrame.width
		d.dstFrame.height = d.srcFrame.height
		d.dstFrame.color_range = C.AVCOL_RANGE_JPEG
		res = C.av_frame_get_buffer(d.dstFrame, 1)
		if res < 0 {
			return decoderOutput{}, errors.New("av_frame_get_buffer() err")
		}

		d.swsCtx = C.sws_getContext(d.srcFrame.width, d.srcFrame.height, C.AV_PIX_FMT_YUV420P,
			d.dstFrame.width, d.dstFrame.height, (int32)(d.dstFrame.format), C.SWS_BILINEAR, nil, nil, nil)
		if d.swsCtx == nil {
			return decoderOutput{}, errors.New("sws_getContext() err")
		}
	}

	// convert frame from YUV420 to RGB
	res = C.sws_scale(d.swsCtx, frameData(d.srcFrame), frameLineSize(d.srcFrame),
		0, d.srcFrame.height, frameData(d.dstFrame), frameLineSize(d.dstFrame))
	if res < 0 {
		return decoderOutput{}, errors.New("sws_scale() err")
	}
	dataGoPtr := d.pool.Get().(*[]byte)
	dstFrameSize := C.av_image_get_buffer_size((int32)(d.dstFrame.format), d.dstFrame.width, d.dstFrame.height, 1)
	if dstFrameSize < 0 {
		return decoderOutput{}, errors.New("failed to get image buffer size")
	}

	// ensure the slice from the pool is large enough
	if cap(*dataGoPtr) < int(dstFrameSize) {
		newSize := int(dstFrameSize)
		*dataGoPtr = make([]byte, newSize)
	} else {
		*dataGoPtr = (*dataGoPtr)[:dstFrameSize]
	}

	// Copy the data from C memory to Go memory
	C.memcpy(unsafe.Pointer(&(*dataGoPtr)[0]), unsafe.Pointer(d.dstFrame.data[0]), C.size_t(dstFrameSize))

	img := &image.RGBA{
		Pix:    *dataGoPtr,
		Stride: 4 * int(d.dstFrame.width),
		Rect: image.Rectangle{
			Max: image.Point{int(d.dstFrame.width), int(d.dstFrame.height)},
		},
	}

	return decoderOutput{
		img:      img,
		poolItem: dataGoPtr,
	}, nil
}
