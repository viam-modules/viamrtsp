package viamrtsp

/*
#cgo pkg-config: libavutil libavcodec
#include <libavcodec/avcodec.h>
#include <libavutil/error.h>
*/
import "C"

import (
	"fmt"
	"unsafe"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	rutils "go.viam.com/rdk/utils"
)

type MimeHandler struct {
	logger     logging.Logger
	jpegSwsCtx *C.struct_SwsContext
	jpegEnc    *C.AVCodecContext
	currentPTS int
}

func newMimeHandler(logger logging.Logger) *MimeHandler {
	return &MimeHandler{
		logger:     logger,
		currentPTS: 0,
	}
}

func (mh *MimeHandler) convertJPEG(frame *avFrameWrapper) ([]byte, camera.ImageMetadata, error) {
	if mh.jpegEnc == nil || frame.frame.width != mh.jpegEnc.width || frame.frame.height != mh.jpegEnc.height {
		mh.logger.Info("creating MJPEG encoder with frame size: ", frame.frame.width, "x", frame.frame.height)
		// Tear down jpeg encoder if we're changing frame size
		if mh.jpegEnc != nil {
			C.avcodec_free_context(&mh.jpegEnc)
		}
		codec := C.avcodec_find_encoder(C.AV_CODEC_ID_MJPEG)
		if codec == nil {
			return nil, camera.ImageMetadata{}, fmt.Errorf("failed to find MJPEG encoder")
		}
		mh.jpegEnc = C.avcodec_alloc_context3(codec)
		mh.jpegEnc.width = frame.frame.width
		mh.jpegEnc.height = frame.frame.height
		mh.jpegEnc.pix_fmt = C.AV_PIX_FMT_YUVJ420P
		mh.jpegEnc.time_base = C.AVRational{num: 1, den: 1} // We don't care about time base for still images
		if res := C.avcodec_open2(mh.jpegEnc, codec, nil); res < 0 {
			return nil, camera.ImageMetadata{}, fmt.Errorf("failed to open codec: %d", res)
		}
	}
	if mh.jpegEnc == nil {
		return nil, camera.ImageMetadata{}, fmt.Errorf("failed to create encoder or destination frame")
	}
	pkt := C.av_packet_alloc()
	if pkt == nil {
		return nil, camera.ImageMetadata{}, fmt.Errorf("failed to allocate packet")
	}
	frame.frame.pts = C.int64_t(mh.currentPTS)
	mh.currentPTS++
	defer C.av_packet_free(&pkt)
	res := C.avcodec_send_frame(mh.jpegEnc, frame.frame)
	if res < 0 {
		return nil, camera.ImageMetadata{}, fmt.Errorf("failed to send frame: %d", res)
	}
	res = C.avcodec_receive_packet(mh.jpegEnc, pkt)
	if res < 0 {
		return nil, camera.ImageMetadata{}, fmt.Errorf("failed to receive packet: %d", res)
	}
	dataGo := C.GoBytes(unsafe.Pointer(pkt.data), pkt.size)

	return dataGo, camera.ImageMetadata{
		MimeType: rutils.MimeTypeJPEG,
	}, nil
}

func (mh *MimeHandler) close() {
	if mh.jpegEnc != nil {
		C.avcodec_free_context(&mh.jpegEnc)
	}
}
