// Package encoder returns encoded video bytes from an image
package encoder

/*
#include <libavcodec/avcodec.h>
#include <libavutil/opt.h>
#include <libavutil/dict.h>
#include <libavutil/frame.h>
#include <libavutil/imgutils.h>
#include <libswscale/swscale.h>
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"errors"
	"sync"
	"unsafe"

	"go.viam.com/rdk/components/generic"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
)

// Model is videostore's Viam model.
var Model = resource.ModelNamespace("viam").WithFamily("viamrtsp").WithModel("x264-encoder")

type Config struct {
	// empty for now
}

func (cfg *Config) Validate(path string) ([]string, []string, error) {
	return nil, nil, nil
}

func init() {
	resource.RegisterComponent(generic.API, Model, resource.Registration[resource.Resource, *Config]{
		Constructor: New,
	})
}

type service struct {
	resource.AlwaysRebuild
	name    resource.Name
	logger  logging.Logger
	encoder *x264Encoder
}

// New creates a new encoder.
func New(ctx context.Context, deps resource.Dependencies, conf resource.Config, logger logging.Logger) (resource.Resource, error) {
	_, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return nil, err
	}
	return &service{
		name:   conf.ResourceName(),
		logger: logger,
	}, nil
}

func (s *service) Name() resource.Name {
	return s.name
}

func (s *service) Close(_ context.Context) error {
	if s.encoder != nil {
		s.encoder.close()
		s.encoder = nil
	}
	return nil
}

// Receives a JPEG image from the command map key "image" and returns the encoded, packetizable x264 video bytes in
// response map key "payload".
func (s *service) DoCommand(ctx context.Context, command map[string]interface{}) (map[string]interface{}, error) {
	image, ok := command["image"].([]byte)
	if !ok {
		return nil, errors.New("image is not a []byte")
	}
	if s.encoder == nil {
		s.encoder = newX264Encoder(s.logger)
	}
	payload, err := s.encoder.Encode(image)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"payload": payload}, nil
}

type x264Encoder struct {
	logger     logging.Logger
	mu         sync.Mutex
	enc        *C.AVCodecContext
	mjpegDec   *C.AVCodecContext
	srcFrame   *C.AVFrame
	yuvFrame   *C.AVFrame
	swsCtx     *C.struct_SwsContext
	currentPTS int
}

func newX264Encoder(logger logging.Logger) *x264Encoder {
	return &x264Encoder{logger: logger}
}

func (e *x264Encoder) close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.enc != nil {
		C.avcodec_free_context(&e.enc)
		e.enc = nil
	}
	if e.mjpegDec != nil {
		C.avcodec_free_context(&e.mjpegDec)
		e.mjpegDec = nil
	}
	if e.srcFrame != nil {
		C.av_frame_free(&e.srcFrame)
		e.srcFrame = nil
	}
	if e.yuvFrame != nil {
		C.av_frame_free(&e.yuvFrame)
		e.yuvFrame = nil
	}
	if e.swsCtx != nil {
		C.sws_freeContext(e.swsCtx)
		e.swsCtx = nil
	}
}

func (e *x264Encoder) ensureEncoder(width, height int) error {
	if width%2 != 0 || height%2 != 0 {
		return errors.New("image dimensions must be even for yuv420p")
	}
	if e.enc != nil && int(e.enc.width) == width && int(e.enc.height) == height {
		return nil
	}

	if e.enc != nil {
		C.avcodec_free_context(&e.enc)
		e.enc = nil
	}
	if e.yuvFrame != nil {
		C.av_frame_free(&e.yuvFrame)
		e.yuvFrame = nil
	}
	if e.swsCtx != nil {
		C.sws_freeContext(e.swsCtx)
		e.swsCtx = nil
	}

	codec := C.avcodec_find_encoder(C.AV_CODEC_ID_H264)
	if codec == nil {
		return errors.New("failed to find libx264 encoder")
	}
	e.enc = C.avcodec_alloc_context3(codec)
	if e.enc == nil {
		return errors.New("failed to allocate encoder context")
	}
	e.enc.width = C.int(width)
	e.enc.height = C.int(height)
	e.enc.time_base = C.AVRational{num: 1, den: 30}
	e.enc.pix_fmt = C.AV_PIX_FMT_YUV420P
	e.enc.max_b_frames = 0
	e.enc.gop_size = 1

	var opts *C.AVDictionary
	presetKey := C.CString("preset")
	presetVal := C.CString("ultrafast")
	tuneKey := C.CString("tune")
	tuneVal := C.CString("zerolatency")
	profileKey := C.CString("profile")
	profileVal := C.CString("baseline")
	crfKey := C.CString("crf")
	crfVal := C.CString("23")
	defer func() {
		C.free(unsafe.Pointer(presetKey))
		C.free(unsafe.Pointer(presetVal))
		C.free(unsafe.Pointer(tuneKey))
		C.free(unsafe.Pointer(tuneVal))
		C.free(unsafe.Pointer(profileKey))
		C.free(unsafe.Pointer(profileVal))
		C.free(unsafe.Pointer(crfKey))
		C.free(unsafe.Pointer(crfVal))
		C.av_dict_free(&opts)
	}()
	if res := C.av_dict_set(&opts, presetKey, presetVal, 0); res < 0 {
		return errors.New("failed to set preset option")
	}
	if res := C.av_dict_set(&opts, tuneKey, tuneVal, 0); res < 0 {
		return errors.New("failed to set tune option")
	}
	if res := C.av_dict_set(&opts, profileKey, profileVal, 0); res < 0 {
		return errors.New("failed to set profile option")
	}
	if res := C.av_dict_set(&opts, crfKey, crfVal, 0); res < 0 {
		return errors.New("failed to set crf option")
	}

	if res := C.avcodec_open2(e.enc, codec, &opts); res < 0 {
		C.avcodec_free_context(&e.enc)
		e.enc = nil
		return errors.New("avcodec_open2 failed for libx264")
	}

	e.yuvFrame = C.av_frame_alloc()
	if e.yuvFrame == nil {
		return errors.New("failed to allocate yuv frame")
	}
	e.yuvFrame.format = C.int(C.AV_PIX_FMT_YUV420P)
	e.yuvFrame.width = C.int(width)
	e.yuvFrame.height = C.int(height)
	if res := C.av_frame_get_buffer(e.yuvFrame, 32); res < 0 {
		C.av_frame_free(&e.yuvFrame)
		e.yuvFrame = nil
		return errors.New("failed to allocate yuv frame buffer")
	}

	// swsCtx will be created on demand in Encode based on src pix_fmt
	e.currentPTS = 0
	return nil
}

func (e *x264Encoder) Encode(jpegBytes []byte) ([]byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(jpegBytes) == 0 {
		return nil, errors.New("image is empty")
	}

	if err := e.ensureMJPEGDecoder(); err != nil {
		return nil, err
	}

	var pkt C.AVPacket
	pkt.data = (*C.uint8_t)(C.CBytes(jpegBytes))
	defer C.free(unsafe.Pointer(pkt.data))
	pkt.size = C.int(len(jpegBytes))

	if res := C.avcodec_send_packet(e.mjpegDec, &pkt); res < 0 {
		return nil, errors.New("failed to send packet to MJPEG decoder")
	}
	if res := C.avcodec_receive_frame(e.mjpegDec, e.srcFrame); res < 0 {
		return nil, errors.New("failed to receive frame from MJPEG decoder")
	}

	width := int(e.srcFrame.width)
	height := int(e.srcFrame.height)
	if err := e.ensureEncoder(width, height); err != nil {
		return nil, err
	}

	// If source is already YUV420P, copy to yuvFrame; otherwise, swscale to YUV420P
	if e.srcFrame.format == C.int(C.AV_PIX_FMT_YUV420P) || e.srcFrame.format == C.int(C.AV_PIX_FMT_YUVJ420P) {
		if res := C.av_frame_copy(e.yuvFrame, e.srcFrame); res < 0 {
			return nil, errors.New("failed to copy YUV frame")
		}
		if res := C.av_frame_copy_props(e.yuvFrame, e.srcFrame); res < 0 {
			return nil, errors.New("failed to copy YUV frame props")
		}
	} else {
		if e.swsCtx == nil || e.srcFrame.width != e.enc.width || e.srcFrame.height != e.enc.height {
			if e.swsCtx != nil {
				C.sws_freeContext(e.swsCtx)
			}
			e.swsCtx = C.sws_getContext(
				e.srcFrame.width, e.srcFrame.height, (C.enum_AVPixelFormat)(e.srcFrame.format),
				e.enc.width, e.enc.height, C.AV_PIX_FMT_YUV420P,
				C.SWS_FAST_BILINEAR, nil, nil, nil,
			)
			if e.swsCtx == nil {
				return nil, errors.New("failed to create sws context")
			}
		}
		if res := C.sws_scale(
			e.swsCtx,
			(**C.uint8_t)(unsafe.Pointer(&e.srcFrame.data[0])),
			(*C.int)(unsafe.Pointer(&e.srcFrame.linesize[0])),
			0,
			e.srcFrame.height,
			(**C.uint8_t)(unsafe.Pointer(&e.yuvFrame.data[0])),
			(*C.int)(unsafe.Pointer(&e.yuvFrame.linesize[0])),
		); res < 0 {
			return nil, errors.New("failed to convert to yuv420p")
		}
	}

	e.yuvFrame.pts = C.int64_t(e.currentPTS)
	e.currentPTS++

	outPkt := C.av_packet_alloc()
	if outPkt == nil {
		return nil, errors.New("failed to allocate packet")
	}
	defer C.av_packet_free(&outPkt)

	if res := C.avcodec_send_frame(e.enc, e.yuvFrame); res < 0 {
		return nil, errors.New("failed to send frame to x264 encoder")
	}
	if res := C.avcodec_receive_packet(e.enc, outPkt); res < 0 {
		return nil, errors.New("failed to receive packet from x264 encoder")
	}

	return C.GoBytes(unsafe.Pointer(outPkt.data), outPkt.size), nil
}

func (e *x264Encoder) ensureMJPEGDecoder() error {
	if e.mjpegDec != nil && e.srcFrame != nil {
		return nil
	}
	codec := C.avcodec_find_decoder(C.AV_CODEC_ID_MJPEG)
	if codec == nil {
		return errors.New("failed to find MJPEG decoder")
	}
	e.mjpegDec = C.avcodec_alloc_context3(codec)
	if e.mjpegDec == nil {
		return errors.New("failed to allocate MJPEG decoder context")
	}
	if res := C.avcodec_open2(e.mjpegDec, codec, nil); res < 0 {
		C.avcodec_free_context(&e.mjpegDec)
		e.mjpegDec = nil
		return errors.New("failed to open MJPEG decoder")
	}
	e.srcFrame = C.av_frame_alloc()
	if e.srcFrame == nil {
		return errors.New("failed to allocate source frame")
	}
	return nil
}
