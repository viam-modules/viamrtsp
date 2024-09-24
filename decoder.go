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
	"sync/atomic"
	"unsafe"

	"github.com/pkg/errors"
	"go.viam.com/rdk/logging"
)

// decoder is a generic FFmpeg decoder.
type decoder struct {
	logger   logging.Logger
	codecCtx *C.AVCodecContext
	// The source yuv420 frame buffer we are decoding from
	src         *C.AVFrame
	swsCtx      *C.struct_SwsContext
	avFramePool *framePool
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

// avFrameWrapper wraps the libav AVFrame.
type avFrameWrapper struct {
	frame *C.AVFrame
	// generation indicates which generation of frame formats the frame is on. It should only be set once and read from.
	// It determines whether the frame can return to the pool or not.
	// If the generation matches that of the pool, then it can return, else it cannot. We need this because otherwise, when resolution changes
	// occur, we would observe undefinied behavior. See https://github.com/erh/viamrtsp/pull/41#discussion_r1719998891
	generation int
	// isFreed indicates whether or not the underlying C memory is freed
	isFreed atomic.Bool
	// isInPool indicates whether or not the frame wrapper is currently an item in the avFramePool
	isInPool atomic.Bool
	// refCount counts how many times the frame is being referenced
	refCount atomic.Int64
}

// incrementRefs increments the ref count by 1.
func (w *avFrameWrapper) incrementRefs() {
	w.refCount.Add(1)
}

// decrementRefs decrements ref count by 1 and returns the new ref count.
func (w *avFrameWrapper) decrementRefs() int64 {
	refCount := w.refCount.Add(-1)
	if refCount < 0 {
		panic("ref count became negative")
	}
	return refCount
}

// free frees the underlying avFrame if it hasn't already been freed.
func (w *avFrameWrapper) free() {
	if w.isFreed.CompareAndSwap(false, true) {
		C.av_frame_free(&w.frame)
	} else {
		panic("av frame was double freed")
	}
}

// toImage takes the underlying av frame and embeds it into a RGBA image struct.
func (w *avFrameWrapper) toImage() image.Image {
	dstFrameSize := C.av_image_get_buffer_size((int32)(w.frame.format), w.frame.width, w.frame.height, 1)
	dstFramePtr := (*[1 << 30]uint8)(unsafe.Pointer(w.frame.data[0]))[:dstFrameSize:dstFrameSize]

	return &image.RGBA{
		Pix:    dstFramePtr,
		Stride: 4 * (int)(w.frame.width),
		Rect: image.Rectangle{
			Max: image.Point{(int)(w.frame.width), (int)(w.frame.height)},
		},
	}
}

// newAVFrameWrapper allocates a new AVFrame using C code with safety checks and returns the Go wrapper of it.
func newAVFrameWrapper(generation int) (*avFrameWrapper, error) {
	avFrame := C.av_frame_alloc()
	if avFrame == nil {
		return nil, errors.New("failed to allocate AVFrame: out of memory or C libav internal error")
	}
	wrapper := &avFrameWrapper{frame: avFrame, generation: generation}
	return wrapper, nil
}

func frameData(frame *C.AVFrame) **C.uint8_t {
	return (**C.uint8_t)(unsafe.Pointer(&frame.data[0]))
}

func frameLineSize(frame *C.AVFrame) *C.int {
	return (*C.int)(unsafe.Pointer(&frame.linesize[0]))
}

// getAvErrorStr converts an AV error code to a AV error message string.
func getAvErrorStr(avErr C.int) string {
	var errbuf [C.AV_ERROR_MAX_STRING_SIZE]C.char
	if C.av_strerror(avErr, &errbuf[0], C.AV_ERROR_MAX_STRING_SIZE) < 0 {
		return fmt.Sprintf("Unknown error with code %d", avErr)
	}
	return C.GoString(&errbuf[0])
}

// avError represents an C AV error in Golang.
type avError struct {
	message  string
	code     int
	avErrStr string
}

// Error implements the standard Error method that string-ifies the error.
func (e *avError) Error() string {
	return fmt.Sprintf("%s: av_error (code %d): %s", e.message, e.code, e.avErrStr)
}

// newAvError is a constructor that creates an avError and gets the underlying av error message.
func newAvError(code C.int, message string) *avError {
	avErrorMsg := getAvErrorStr(code)
	return &avError{
		code:     int(code),
		message:  message,
		avErrStr: avErrorMsg,
	}
}

// recoverableError is used when the decoder should return an error, but it can be ignored or recovered from.
type recoverableError struct {
	err error
}

func (e *recoverableError) Error() string {
	return e.err.Error()
}

// newRecoverableError creates a wrapped error that can be caught and recovered from.
func newRecoverableError(err error) *recoverableError {
	return &recoverableError{
		err: err,
	}
}

// SetLibAVLogLevelFatal sets libav errors to fatal log level
// to cut down on log spam
func SetLibAVLogLevelFatal() {
	C.av_log_set_level(C.AV_LOG_FATAL)
}

// newDecoder creates a new decoder for the given codec.
func newDecoder(codecID C.enum_AVCodecID, avFramePool *framePool, logger logging.Logger) (*decoder, error) {
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
		return nil, newAvError(res, "avcodec_open2() failed")
	}

	src := C.av_frame_alloc()
	if src == nil {
		C.avcodec_close(codecCtx)
		return nil, errors.New("av_frame_alloc() failed")
	}

	return &decoder{
		logger:      logger,
		codecCtx:    codecCtx,
		src:         src,
		avFramePool: avFramePool,
	}, nil
}

// newH264Decoder creates a new H264 decoder.
func newH264Decoder(avFramePool *framePool, logger logging.Logger) (*decoder, error) {
	return newDecoder(C.AV_CODEC_ID_H264, avFramePool, logger)
}

// newH265Decoder creates a new H265 decoder.
func newH265Decoder(avFramePool *framePool, logger logging.Logger) (*decoder, error) {
	return newDecoder(C.AV_CODEC_ID_H265, avFramePool, logger)
}

// close closes the decoder.
func (d *decoder) close() {
	if d.swsCtx != nil {
		C.sws_freeContext(d.swsCtx)
	}

	if d.src != nil {
		C.av_frame_free(&d.src)
	}

	if d.codecCtx != nil {
		C.avcodec_close(d.codecCtx)
	}
}

func (d *decoder) decode(nalu []byte) (*avFrameWrapper, error) {
	nalu = append(H2645StartCode(), nalu...)

	// send frame to decoder
	var avPacket C.AVPacket
	avPacket.data = (*C.uint8_t)(C.CBytes(nalu))
	defer C.free(unsafe.Pointer(avPacket.data))
	avPacket.size = C.int(len(nalu))
	res := C.avcodec_send_packet(d.codecCtx, &avPacket)
	if res < 0 {
		return nil, newRecoverableError(newAvError(res, "error sending packet to the decoder"))
	}

	// receive frame if available
	res = C.avcodec_receive_frame(d.codecCtx, d.src)
	if res < 0 {
		return nil, newRecoverableError(newAvError(res, "error receiving decoded frame from the decoder"))
	}

	// Get a frame from the pool. This frame will be in one of three states:
	// - The frame is uninitialized. The width/height will be set to 0 and the frame's byte buffer
	//   will be empty.
	// - The frame is initialized with a height/width/buffer, all of the desired values/size.
	// - The frame is initialized with an old height/width/buffer that no longer matches the
	//   source yuv frame.
	dst := d.avFramePool.get()

	if dst == nil {
		return nil, errors.New("failed to obtain AVFrame from pool")
	}
	if dst.isFreed.Load() {
		return nil, errors.New("got frame from pool that was already freed")
	}

	// If the frame from the pool has the wrong size, (re-)initialize it.
	if dst.frame.width != d.src.width || dst.frame.height != d.src.height {
		d.logger.Debugf("(re)making frame due to AVFrame dimension discrepancy: Dst (width: %d, height: %d) vs Src (width: %d, height: %d)",
			dst.frame.width, dst.frame.height, d.src.width, d.src.height)

		// Handle size changes while having previously initialized frames to avoid https://github.com/erh/viamrtsp/pull/41#discussion_r1719998891
		frameWasPreviouslyInitialized := dst.frame.width > 0 && dst.frame.height > 0
		if frameWasPreviouslyInitialized {
			// Release previously initialized frames, and block old prev gen frames from returning to pool
			dst.free()
			generation := d.avFramePool.clearAndStartNewGeneration()
			// Make new frame to be initialized with new size
			newDst, err := newAVFrameWrapper(generation)
			if err != nil {
				return nil, errors.Errorf("AV frame allocation error while decoding: %v", err)
			}
			dst = newDst
		}
		if d.swsCtx != nil {
			// When the resolution changes, we must also free+reallocate the `swsCtx`.
			C.sws_freeContext(d.swsCtx)
		}

		// Prepare the fresh frame
		dst.frame.format = C.AV_PIX_FMT_RGBA
		dst.frame.width = d.src.width
		dst.frame.height = d.src.height
		dst.frame.color_range = C.AVCOL_RANGE_JPEG

		// This allocates the underlying byte array to contain the image data.
		res = C.av_frame_get_buffer(dst.frame, 1)
		if res < 0 {
			return nil, newAvError(res, "av_frame_get_buffer() err")
		}

		// Create a scratch space for converting YUV420 to RGB. In our use-case, the yuv source + dst
		// resolutions always match.
		d.swsCtx = C.sws_getContext(d.src.width, d.src.height, C.AV_PIX_FMT_YUV420P,
			dst.frame.width, dst.frame.height, (int32)(dst.frame.format), C.SWS_BILINEAR, nil, nil, nil)
		if d.swsCtx == nil {
			return nil, errors.New("sws_getContext() err")
		}
	}

	// convert frame from YUV420 to RGB
	res = C.sws_scale(d.swsCtx, frameData(d.src), frameLineSize(d.src),
		0, d.src.height, frameData(dst.frame), frameLineSize(dst.frame))
	if res < 0 {
		return nil, newAvError(res, "sws_scale() err")
	}

	return dst, nil
}
