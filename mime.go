package viamrtsp

/*
#cgo pkg-config: libavutil libavcodec
#include <libavcodec/avcodec.h>
#include <libavutil/error.h>
#include <libavutil/opt.h>
#include <libavutil/dict.h>
#include <stdlib.h>
*/
import "C"

import (
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
	currentPTS int
	mu         sync.Mutex
}

func newMimeHandler(logger logging.Logger) *mimeHandler {
	return &mimeHandler{
		logger:     logger,
		currentPTS: 0,
	}
}

func (mh *mimeHandler) convertJPEG(frame *avFrameWrapper) ([]byte, camera.ImageMetadata, error) {
	if mh.jpegEnc == nil || frame.frame.width != mh.jpegEnc.width || frame.frame.height != mh.jpegEnc.height {
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
	frame.frame.pts = C.int64_t(mh.currentPTS)
	// If this reaches max int, it will wrap around to 0
	mh.currentPTS++
	res := C.avcodec_send_frame(mh.jpegEnc, frame.frame)
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

func (mh *mimeHandler) initJPEGEncoder(frame *avFrameWrapper) error {
	// Lock to prevent modifying encoder while it is being used concurrently.
	// Frame param changes are rare, so we can afford to block here.
	mh.mu.Lock()
	defer mh.mu.Unlock()
	mh.logger.Info("creating MJPEG encoder with frame size: ", frame.frame.width, "x", frame.frame.height)
	if mh.jpegEnc != nil {
		C.avcodec_free_context(&mh.jpegEnc)
	}
	codec := C.avcodec_find_encoder(C.AV_CODEC_ID_MJPEG)
	if codec == nil {
		return errors.New("failed to find MJPEG encoder")
	}
	mh.jpegEnc = C.avcodec_alloc_context3(codec)
	mh.jpegEnc.width = frame.frame.width
	mh.jpegEnc.height = frame.frame.height
	mh.jpegEnc.pix_fmt = C.AV_PIX_FMT_YUVJ420P
	// We don't care about accurate timestamps still frames
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

func (mh *mimeHandler) close() {
	if mh.jpegEnc != nil {
		C.avcodec_free_context(&mh.jpegEnc)
	}
}
