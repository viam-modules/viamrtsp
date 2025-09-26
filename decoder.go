package viamrtsp

/*
#include <libavcodec/avcodec.h>
#include <libavutil/imgutils.h>
#include <libavutil/error.h>
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"image"
	"sync/atomic"
	"unsafe"

	"go.viam.com/rdk/logging"
)

const (
	yuv420SubsampleRatio = 2
	mimeTypeYUYV         = "image/yuyv422"
)

// decoder is a generic FFmpeg decoder.
type decoder struct {
	logger   logging.Logger
	codecCtx *C.AVCodecContext
	// The source yuv420 frame buffer we are decoding from
	src         *C.AVFrame
	avFramePool *framePool
}

type videoCodec int

const (
	// Number of bytes per pixel for RGBA format
	bytesPerPixel = 4

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
	// MPEG4 indicates the mpeg4 video codec
	MPEG4
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
	case MPEG4:
		return "MPEG4"
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

// toImage maps the underlying AVFrame (in YUV420P format) to a Go image.YCbCr.
func (w *avFrameWrapper) toImage() image.Image {
	if w.frame.format != C.AV_PIX_FMT_YUV420P && w.frame.format != C.AV_PIX_FMT_YUVJ420P {
		return nil
	}

	width := int(w.frame.width)
	height := int(w.frame.height)
	if width <= 0 || height <= 0 {
		return nil
	}

	// For YUV420P:
	//   - data[0] = Y plane, linesize[0] = stride for Y
	//   - data[1] = U plane, linesize[1] = stride for U
	//   - data[2] = V plane, linesize[2] = stride for V
	yStride := int(w.frame.linesize[0])
	cStride := int(w.frame.linesize[1])

	// Number of bytes in each plane.
	// Y plane is full resolution: height * yStride
	// U and V planes are half resolution in both dimensions: (height/2) * cStride
	yPlaneSize := yStride * height
	cPlaneSize := cStride * (height / yuv420SubsampleRatio)

	yDataPtr := unsafe.Pointer(w.frame.data[0])
	ySlice := (*[1 << 30]byte)(yDataPtr)[:yPlaneSize:yPlaneSize]

	cbDataPtr := unsafe.Pointer(w.frame.data[1])
	cbSlice := (*[1 << 30]byte)(cbDataPtr)[:cPlaneSize:cPlaneSize]

	crDataPtr := unsafe.Pointer(w.frame.data[2])
	crSlice := (*[1 << 30]byte)(crDataPtr)[:cPlaneSize:cPlaneSize]

	return &image.YCbCr{
		Y:              ySlice,
		Cb:             cbSlice,
		Cr:             crSlice,
		YStride:        yStride,
		CStride:        cStride,
		SubsampleRatio: image.YCbCrSubsampleRatio420,
		Rect:           image.Rect(0, 0, width, height),
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

// SetLibAVLogLevelError sets libav errors to error log level
func SetLibAVLogLevelError() {
	C.av_log_set_level(C.AV_LOG_ERROR)
}

// SetLibAVLogLevelDebug sets libav errors to debug log level
func SetLibAVLogLevelDebug() {
	C.av_log_set_level(C.AV_LOG_DEBUG)
}

// SetLibAVLogLevelTrace sets libav errors to trace log level
func SetLibAVLogLevelTrace() {
	C.av_log_set_level(C.AV_LOG_TRACE)
}

// newDecoder creates a new decoder for the given codec, including any extra configuration data.
func newDecoder(codecID C.enum_AVCodecID, avFramePool *framePool, logger logging.Logger, extraData []byte) (*decoder, error) {
	codec := C.avcodec_find_decoder(codecID)
	if codec == nil {
		return nil, errors.New("avcodec_find_decoder() failed")
	}

	codecCtx := C.avcodec_alloc_context3(codec)
	if codecCtx == nil {
		return nil, errors.New("avcodec_alloc_context3() failed")
	}
	// Set the codec context to decode YUV420P frames. The decoder can still
	// output JPEG color range frames YUVJ420P.
	codecCtx.pix_fmt = C.AV_PIX_FMT_YUV420P

	// Set extradata if provided
	if len(extraData) > 0 {
		codecCtx.extradata_size = C.int(len(extraData))
		codecCtx.extradata = (*C.uint8_t)(C.av_malloc(C.size_t(codecCtx.extradata_size)))
		if codecCtx.extradata == nil {
			C.avcodec_close(codecCtx)
			return nil, errors.New("av_malloc() failed for extradata")
		}
		C.memcpy(unsafe.Pointer(codecCtx.extradata), unsafe.Pointer(&extraData[0]), C.size_t(codecCtx.extradata_size))
	}

	res := C.avcodec_open2(codecCtx, codec, nil)
	if res < 0 {
		C.avcodec_close(codecCtx)
		return nil, newAvError(res, "avcodec_open2() failed")
	}

	// Log codec context details
	logger.Infof("Initialized codec: %s, width: %d, height: %d", C.GoString(codec.name), codecCtx.width, codecCtx.height)

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
	return newDecoder(C.AV_CODEC_ID_H264, avFramePool, logger, nil)
}

// newH265Decoder creates a new H265 decoder.
func newH265Decoder(avFramePool *framePool, logger logging.Logger) (*decoder, error) {
	return newDecoder(C.AV_CODEC_ID_H265, avFramePool, logger, nil)
}

// newMPEG4Decoder creates a new MPEG4 decoder with the provided configuration data as extra data.
func newMPEG4Decoder(avFramePool *framePool, logger logging.Logger, extraData []byte) (*decoder, error) {
	return newDecoder(C.AV_CODEC_ID_MPEG4, avFramePool, logger, extraData)
}

// close closes the decoder.
func (d *decoder) close() {
	if d.src != nil {
		C.av_frame_free(&d.src)
	}

	if d.codecCtx != nil {
		C.avcodec_close(d.codecCtx)
	}
}

func (d *decoder) decode(nalu []byte) (*avFrameWrapper, error) {
	if d.codecCtx.codec_id == C.AV_CODEC_ID_H264 || d.codecCtx.codec_id == C.AV_CODEC_ID_H265 {
		nalu = append(H2645StartCode(), nalu...)
	}

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
	if dst.frame.width != d.src.width || dst.frame.height != d.src.height || dst.frame.format != d.src.format {
		d.logger.Debugf("(re)making frame due to AVFrame discrepancy: "+
			"Dst (width: %d, height: %d, format: %d) vs Src (width: %d, height: %d, format: %d)",
			dst.frame.width, dst.frame.height, dst.frame.format,
			d.src.width, d.src.height, d.src.format)
		// Handle size changes while having previously initialized frames to avoid https://github.com/erh/viamrtsp/pull/41#discussion_r1719998891
		frameWasPreviouslyInitialized := dst.frame.width > 0 && dst.frame.height > 0
		if frameWasPreviouslyInitialized {
			// Release previously initialized frames, and block old prev gen frames from returning to pool
			dst.free()
			generation := d.avFramePool.clearAndStartNewGeneration()
			// Make new frame to be initialized with new size
			newDst, err := newAVFrameWrapper(generation)
			if err != nil {
				return nil, fmt.Errorf("AV frame allocation error while decoding: %w", err)
			}
			dst = newDst
		}
		// Prepare the fresh frame
		dst.frame.format = d.src.format
		dst.frame.width = d.src.width
		dst.frame.height = d.src.height

		res = C.av_frame_get_buffer(dst.frame, 32)
		if res < 0 {
			return nil, newAvError(res, "av_frame_get_buffer() err")
		}
	}

	// We need to copy the frame data from the source frame to the destination frame
	// because the source frame will be overwritten by the next frame that is decoded.
	if res := C.av_frame_copy(dst.frame, d.src); res < 0 {
		return nil, newAvError(res, "av_frame_copy() failed")
	}

	// Copy the frame properties from the source frame to the destination frame.
	// This will fill fields not explicitly set in the initial dst frame allocation.
	if res := C.av_frame_copy_props(dst.frame, d.src); res < 0 {
		// We should never reach this point if av_frame_copy() succeeded.
		return nil, newAvError(res, "av_frame_copy_props() failed")
	}

	return dst, nil
}
