UNAME_S ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
UNAME_M ?= $(shell uname -m)
ifeq ($(UNAME_M),aarch64)
	UNAME_M = arm64
else ifeq ($(UNAME_M),x86_64)
	UNAME_M = amd64
endif
TARGET ?= $(UNAME_S)
CC ?= $(shell which gcc)
FFMPEG_PREFIX ?= $(shell pwd)/FFmpeg/$(TARGET)-$(UNAME_M)
FFMPEG_OPTS ?= --prefix=$(FFMPEG_PREFIX) \
               --enable-static \
               --disable-shared \
               --disable-programs \
               --disable-doc \
               --disable-everything \
               --enable-decoder=h264 \
               --enable-decoder=hevc \
               --enable-network \
               --enable-demuxer=rtsp \
               --enable-parser=h264 \
               --enable-parser=hevc

CGO_LDFLAGS := -L$(FFMPEG_PREFIX)/lib

ifeq ($(TARGET),android)
	NDK_ROOT ?= $(HOME)/Library/Android/Sdk/ndk/26.1.10909125
	CC = $(NDK_ROOT)/toolchains/llvm/prebuilt/$(UNAME_S)-x86_64/bin/aarch64-linux-android$(API_LEVEL)-clang
	API_LEVEL ?= 30
	FFMPEG_OPTS += --target-os=android \
	               --arch=aarch64 \
				   --cpu=armv8-a \
	               --enable-cross-compile \
	               --sysroot=$(NDK_ROOT)/toolchains/llvm/prebuilt/$(UNAME_S)-x86_64/sysroot \
				   --cc=$(NDK_ROOT)/toolchains/llvm/prebuilt/$(UNAME_S)-x86_64/bin/aarch64-linux-android$(API_LEVEL)-clang \
				   --cxx=$(NDK_ROOT)/toolchains/llvm/prebuilt/$(UNAME_S)-x86_64/bin/aarch64-linux-android$(API_LEVEL)-clang++
	CGO_CFLAGS += -I$(NDK_ROOT)/toolchains/llvm/prebuilt/$(UNAME_S)-x86_64/sysroot/usr/include \
                  -I$(NDK_ROOT)/toolchains/llvm/prebuilt/$(UNAME_S)-x86_64/sysroot/usr/include/aarch64-linux-android
	CGO_LDFLAGS += -L$(NDK_ROOT)/toolchains/llvm/prebuilt/$(UNAME_S)-x86_64/sysroot/usr/lib
endif

ifeq ($(UNAME_S),linux)
	CGO_LDFLAGS += -l:libjpeg.a
endif

.PHONY: build-ffmpeg test lint updaterdk module clean

# go mod edit -replace=go.viam.com/rdk=github.com/abe-winter/rdk@droid-apk
bin/viamrtsp: build-ffmpeg *.go cmd/module/*.go
ifeq ($(TARGET),android)
	go mod edit -replace=go.viam.com/rdk=go.viam.com/rdk@v0.20.1-0.20240209172210-8dc034cf4d2a
	go mod tidy
endif
	PKG_CONFIG_PATH=$(FFMPEG_PREFIX)/lib/pkgconfig \
		CGO_LDFLAGS="$(CGO_LDFLAGS)" \
		CC=$(CC) \
		CGO_ENABLED=1 GOOS=$(TARGET) GOARCH=$(UNAME_M) go build -v -tags no_cgo,no_media -o bin/viamrtsp cmd/module/cmd.go

test:
	go test

lint:
	gofmt -w -s .

updaterdk:
	go get go.viam.com/rdk@latest
	go mod tidy

FFmpeg:
	git clone https://github.com/FFmpeg/FFmpeg.git --depth 1 --branch n6.1

$(FFMPEG_PREFIX): FFmpeg
	cd FFmpeg && ./configure $(FFMPEG_OPTS) && $(MAKE) -j$(shell nproc) && $(MAKE) install

build-ffmpeg:
ifeq ($(UNAME_S),linux)
ifeq ($(UNAME_M),amd64)
	which nasm || (sudo apt update && sudo apt install -y nasm)
endif
endif
	$(MAKE) $(FFMPEG_PREFIX)

module: bin/viamrtsp
	tar czf module.tar.gz bin/viamrtsp

clean:
	rm -rf FFmpeg bin/viamrtsp module.tar.gz
