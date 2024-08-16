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
	"runtime"
	"sync"
	"unsafe"

	"github.com/pkg/errors"
	"go.viam.com/rdk/logging"
)

// decoder is a generic FFmpeg decoder.
type decoder struct {
	logger      logging.Logger
	codecCtx    *C.AVCodecContext
	src         *avFrameWrapper
	swsCtx      *C.struct_SwsContext
	dst         *avFrameWrapper
	avFramePool *sync.Pool
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

type avFrameWrapper struct {
	frame *C.AVFrame
}

// allocateAVFrame allocates a new AVFrame using C code with safety checks and returns the Go wrapper of it.
func allocateAVFrame() (*avFrameWrapper, error) {
	avFrame := C.av_frame_alloc()
	if avFrame == nil {
		return nil, errors.New("failed to allocate AVFrame: out of memory or C libav internal error")
	}
	wrapper := &avFrameWrapper{frame: avFrame}
	// Set a finalizer on the wrapper to ensure the C memory is freed
	runtime.SetFinalizer(wrapper, func(w *avFrameWrapper) {
		if w.frame != nil {
			C.av_frame_free(&w.frame)
		}
	})
	return wrapper, nil
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

	srcFrame, err := allocateAVFrame()
	if err != nil {
		return nil, errors.Errorf("AV Frame allocation error during decoder init: %v", err)
	}
	if srcFrame == nil {
		C.avcodec_close(codecCtx)
		return nil, errors.New("av_frame_alloc() failed")
	}

	return &decoder{
		logger:      logger,
		codecCtx:    codecCtx,
		src:         srcFrame,
		avFramePool: avFramePool,
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
	if d.dst != nil {
		d.dst = nil
	}

	if d.swsCtx != nil {
		C.sws_freeContext(d.swsCtx)
	}

	if d.src != nil {
		d.src = nil
	}

	if d.codecCtx != nil {
		C.avcodec_close(d.codecCtx)
		d.codecCtx = nil
	}
}

func (d *decoder) decode(nalu []byte) (*imageAndPoolItem, error) {
	nalu = append(H2645StartCode(), nalu...)

	// send frame to decoder
	var avPacket C.AVPacket
	avPacket.data = (*C.uint8_t)(C.CBytes(nalu))
	defer C.free(unsafe.Pointer(avPacket.data))
	avPacket.size = C.int(len(nalu))
	res := C.avcodec_send_packet(d.codecCtx, &avPacket)
	if res < 0 {
		//nolint:nilnil // TODO RSDK-8575: change to not nil, nil
		return nil, nil
	}

	// receive frame if available
	res = C.avcodec_receive_frame(d.codecCtx, d.src.frame)
	if res < 0 {
		//nolint:nilnil // TODO RSDK-8575: change to not nil, nil
		return nil, nil
	}

	// Get a frame from the pool. This frame will be in one of three states:
	// - The frame is uninitialized. The width/height will be set to 0 and the frame's byte buffer
	//   will be empty.
	// - The frame is initialized with a height/width/buffer, all of the desired values/size.
	// - The frame is initialized with an old height/width/buffer that no longer matches the
	//   `src.frame`
	d.dst = d.avFramePool.Get().(*avFrameWrapper)
	if d.dst == nil {
		return nil, errors.New("failed to obtain AVFrame from pool")
	}

	// If the frame from the pool has the wrong size, (re-)initialize it.
	if d.dst.frame.width != d.src.frame.width || d.dst.frame.height != d.src.frame.height {
		if d.swsCtx != nil {
			// When the resolution changes, we must also free+reallocate the `swsCtx`.
			C.sws_freeContext(d.swsCtx)
		}

		// We didn't like the frame we got from the pool, so we'll throw the old one away and create
		// a fresh frame.
		dstFrame, err := allocateAVFrame()
		if err != nil {
			return nil, errors.Errorf("AV frame allocation error while decoding: %v", err)
		}
		d.dst = dstFrame
		d.dst.frame.format = C.AV_PIX_FMT_RGBA
		d.dst.frame.width = d.src.frame.width
		d.dst.frame.height = d.src.frame.height
		d.dst.frame.color_range = C.AVCOL_RANGE_JPEG
		// This allocates the underlying byte array to contain the image data.
		res = C.av_frame_get_buffer(d.dst.frame, 1)
		if res < 0 {
			return nil, errors.New("av_frame_get_buffer() err")
		}

		// Create a scratch space for converting YUV420 to RGB. In our use-case, the src + dst
		// resolutions always match.
		d.swsCtx = C.sws_getContext(d.src.frame.width, d.src.frame.height, C.AV_PIX_FMT_YUV420P,
			d.dst.frame.width, d.dst.frame.height, (int32)(d.dst.frame.format), C.SWS_BILINEAR, nil, nil, nil)
		if d.swsCtx == nil {
			return nil, errors.New("sws_getContext() err")
		}
	}

	dstFrameSize := C.av_image_get_buffer_size((int32)(d.dst.frame.format), d.dst.frame.width, d.dst.frame.height, 1)
	dstFramePtr := (*[1 << 30]uint8)(unsafe.Pointer(d.dst.frame.data[0]))[:dstFrameSize:dstFrameSize]

	// convert frame from YUV420 to RGB
	res = C.sws_scale(d.swsCtx, frameData(d.src.frame), frameLineSize(d.src.frame),
		0, d.src.frame.height, frameData(d.dst.frame), frameLineSize(d.dst.frame))
	if res < 0 {
		return nil, errors.New("sws_scale() err")
	}

	// embed frame into an image.Image
	img := &image.RGBA{
		Pix:    dstFramePtr,
		Stride: 4 * (int)(d.dst.frame.width),
		Rect: image.Rectangle{
			Max: image.Point{(int)(d.dst.frame.width), (int)(d.dst.frame.height)},
		},
	}

	return &imageAndPoolItem{
		img:      img,
		poolItem: d.dst,
	}, nil
}
