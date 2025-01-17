package viamrtsp

/*
#cgo pkg-config: libavutil libavcodec libswscale
#include <libavcodec/avcodec.h>
#include <libavutil/error.h>
#include <libavutil/opt.h>
#include <libavutil/dict.h>
#include <libavutil/frame.h>
#include <libavutil/imgutils.h>
#include <libswscale/swscale.h>
#include <stdlib.h>
*/
import "C"

import (
	"bytes"
	"encoding/binary"
	"errors"
	"sync"
	"unsafe"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	rutils "go.viam.com/rdk/utils"
)

type mimeHandler struct {
	logger     logging.Logger
	jpegEnc    *C.AVCodecContext
	yuyvFrame  *C.AVFrame
	yuyvSwsCtx *C.struct_SwsContext
	currentPTS int
	mu         sync.Mutex
}

func newMimeHandler(logger logging.Logger) *mimeHandler {
	return &mimeHandler{
		logger:     logger,
		currentPTS: 0,
	}
}

func (mh *mimeHandler) convertJPEG(frame *C.AVFrame) ([]byte, camera.ImageMetadata, error) {
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
	// Frame param changes are rare, so we can afford to block here.
	mh.mu.Lock()
	defer mh.mu.Unlock()
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
	if res := C.avcodec_open2(mh.jpegEnc, codec, nil); res < 0 {
		return newAvError(res, "failed to open MJPEG encoder")
	}

	return nil
}

func (mh *mimeHandler) convertYUYV(frame *C.AVFrame) ([]byte, camera.ImageMetadata, error) {
	if mh.yuyvSwsCtx == nil || frame.width != mh.yuyvFrame.width || frame.height != mh.yuyvFrame.height {
		if err := mh.initYUYVContext(frame); err != nil {
			return nil, camera.ImageMetadata{}, err
		}
	}

	// perform the conversion
	res := C.sws_scale(
		mh.yuyvSwsCtx,
		(**C.uint8_t)(unsafe.Pointer(&frame.data[0])),
		(*C.int)(unsafe.Pointer(&frame.linesize[0])),
		0,
		frame.height,
		(**C.uint8_t)(unsafe.Pointer(&mh.yuyvFrame.data[0])),
		(*C.int)(unsafe.Pointer(&mh.yuyvFrame.linesize[0])),
	)
	if res < 0 {
		return nil, camera.ImageMetadata{}, newAvError(res, "failed to convert frame to YUYV")
	}

	yuyvBytes := C.GoBytes(unsafe.Pointer(mh.yuyvFrame.data[0]), C.int(mh.yuyvFrame.width*mh.yuyvFrame.height*2))
	header := packYUYVHeader(int(mh.yuyvFrame.width), int(mh.yuyvFrame.height))
	// TODO(seanp): Figure out if this is memory efficient
	yuyvPacket := append(header, yuyvBytes...)

	return yuyvPacket, camera.ImageMetadata{
		MimeType: mimeTypeYUYV,
	}, nil
}

func (mh *mimeHandler) initYUYVContext(frame *C.AVFrame) error {
	mh.mu.Lock()
	defer mh.mu.Unlock()
	mh.logger.Info("creating YUYV sws cosntext with frame size: ", frame.width, "x", frame.height)
	if mh.yuyvSwsCtx != nil {
		C.sws_freeContext(mh.yuyvSwsCtx)
	}
	if mh.yuyvFrame != nil {
		C.av_frame_free(&mh.yuyvFrame)
	}
	mh.yuyvFrame = C.av_frame_alloc()
	if mh.yuyvFrame == nil {
		return errors.New("failed to allocate YUYV frame")
	}
	mh.yuyvFrame.width = frame.width
	mh.yuyvFrame.height = frame.height
	mh.yuyvFrame.format = C.AV_PIX_FMT_YUYV422
	if res := C.av_frame_get_buffer(mh.yuyvFrame, 32); res < 0 {
		C.av_frame_free(&mh.yuyvFrame)
		return newAvError(res, "failed to allocate buffer for YUYV frame")
	}
	mh.yuyvSwsCtx = C.sws_getContext(
		frame.width, frame.height, C.AV_PIX_FMT_YUV420P,
		frame.width, frame.height, C.AV_PIX_FMT_YUYV422,
		C.SWS_FAST_BILINEAR, nil, nil, nil,
	)
	if mh.yuyvSwsCtx == nil {
		C.av_frame_free(&mh.yuyvFrame)
		return errors.New("failed to create YUYV converter")
	}

	return nil
}

func (mh *mimeHandler) close() {
	if mh.jpegEnc != nil {
		C.avcodec_free_context(&mh.jpegEnc)
	}
}

func packYUYVHeader(width, height int) []byte {
	var header bytes.Buffer
	header.Write([]byte("YUYV"))
	tmp := make([]byte, 4)
	binary.BigEndian.PutUint32(tmp, uint32(width))
	header.Write(tmp)
	binary.BigEndian.PutUint32(tmp, uint32(height))
	header.Write(tmp)

	return header.Bytes()
}
