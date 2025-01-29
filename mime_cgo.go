package viamrtsp

/*
#cgo pkg-config: libavutil libavcodec
#include <libavcodec/avcodec.h>
#include <libavutil/frame.h>
#include <libavutil/imgutils.h>
#include <stdlib.h>
*/
import "C"
import "unsafe"

// This is a CGO helper file for mime_test.go. We cannot use CGO directly in test files
// so we have to place C interop code here.

func createTestFrame(width, height int, format C.enum_AVPixelFormat) *C.AVFrame {
	frame := C.av_frame_alloc()
	if frame == nil {
		return nil
	}
	frame.format = C.int(format)
	frame.width = C.int(width)
	frame.height = C.int(height)
	if res := C.av_frame_get_buffer(frame, 32); res < 0 {
		// If this fails, free the frame and return nil
		C.av_frame_free(&frame)
		return nil
	}
	// Mark frame as writable, so we can safely fill the data arrays.
	if res := C.av_frame_make_writable(frame); res < 0 {
		C.av_frame_free(&frame)
		return nil
	}

	return frame
}

func createTestYUV420PFrame(width, height int) *C.AVFrame {
	return createTestFrame(width, height, C.AV_PIX_FMT_YUV420P)
}

func createTestYUVJ420PFrame(width, height int) *C.AVFrame {
	return createTestFrame(width, height, C.AV_PIX_FMT_YUVJ420P)
}

func createInvalidFrame() *C.AVFrame {
	frame := C.av_frame_alloc()
	if frame == nil {
		return nil
	}
	frame.format = C.AV_PIX_FMT_NONE
	frame.width = 0
	frame.height = 0

	return frame
}

func fillDummyYUV420PData(frame *C.AVFrame) {
	width := int(frame.width)
	height := int(frame.height)

	yPlane := (*[1 << 30]uint8)(unsafe.Pointer(frame.data[0]))[: width*height : width*height]
	uPlane := (*[1 << 30]uint8)(unsafe.Pointer(frame.data[1]))[: width*height/4 : width*height/4]
	vPlane := (*[1 << 30]uint8)(unsafe.Pointer(frame.data[2]))[: width*height/4 : width*height/4]

	// Fill Y plane
	for y := range height {
		for x := range width {
			yPlane[y*int(frame.linesize[0])+x] = 128
		}
	}
	// Fill U plane
	for y := range height / 2 {
		for x := range width / 2 {
			uPlane[y*int(frame.linesize[1])+x] = 64
		}
	}
	// Fill V plane
	for y := range height / 2 {
		for x := range width / 2 {
			vPlane[y*int(frame.linesize[2])+x] = 64
		}
	}
}

func freeFrame(frame *C.AVFrame) {
	C.av_frame_free(&frame)
}

func createTestRGBAFrame(width, height int) *C.AVFrame {
	return createTestFrame(width, height, C.AV_PIX_FMT_RGBA)
}

func fillDummyRGBAData(frame *C.AVFrame) {
	width := int(frame.width)
	height := int(frame.height)
	linesize := int(frame.linesize[0])

	rgbaPlane := (*[1 << 30]uint8)(unsafe.Pointer(frame.data[0]))[: height*linesize : height*linesize]

	// Top half: solid red (255, 0, 0, 255)
	// Bottom half: solid blue (0, 0, 255, 255)
	for y := range height {
		for x := range width {
			idx := y*linesize + x*rgbaBytesPerPixel
			if y < height/2 {
				rgbaPlane[idx+0] = 255 // R
				rgbaPlane[idx+1] = 0   // G
				rgbaPlane[idx+2] = 0   // B
			} else {
				rgbaPlane[idx+0] = 0   // R
				rgbaPlane[idx+1] = 0   // G
				rgbaPlane[idx+2] = 255 // B
			}
			rgbaPlane[idx+3] = 255 // A always fully opaque
		}
	}
}
