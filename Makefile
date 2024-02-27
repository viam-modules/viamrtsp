GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
ARCH ?= $(shell uname -m)
TARGET_IP ?= 127.0.0.1
API_LEVEL ?= 29

UNAME=$(shell uname)
ifeq ($(UNAME),Linux)
	NDK_ROOT ?= $(HOME)/Android/Sdk/ndk/26.1.10909125
	HOST_OS ?= linux
	CC_ARCH ?= aarch64
else
	NDK_ROOT ?= $(HOME)/Library/Android/sdk/ndk/26.1.10909125
	HOST_OS ?= darwin
	CC_ARCH ?= aarch64
endif

# FFmpeg build settings
TOOLCHAIN := $(NDK_ROOT)/toolchains/llvm/prebuilt/$(HOST_OS)-x86_64
CC := $(TOOLCHAIN)/bin/$(CC_ARCH)-linux-android$(API_LEVEL)-clang
CXX := $(TOOLCHAIN)/bin/$(CC_ARCH)-linux-android$(API_LEVEL)-clang++
AR := $(TOOLCHAIN)/bin/llvm-ar
LD := $(CC)
RANLIB := $(TOOLCHAIN)/bin/llvm-ranlib
STRIP := $(TOOLCHAIN)/bin/llvm-strip
NM := $(TOOLCHAIN)/bin/llvm-nm
SYSROOT := $(TOOLCHAIN)/sysroot

FFMPEG_SUBDIR=viamrtsp/ffmpeg-android
FFMPEG_PREFIX=$(HOME)/$(FFMPEG_SUBDIR)

# CGO settings
CGO_ENABLED := 1
CGO_CFLAGS := -I$(FFMPEG_PREFIX)/include
CGO_LDFLAGS := -L$(FFMPEG_PREFIX)/lib
OUTPUT_DIR := bin
OUTPUT := $(OUTPUT_DIR)/viamrtsp-$(GOOS)-$(GOARCH)

# Build go binary for linux
.PHONY: build-linux
build-linux:
	go build -v -o ./bin/viamrtsp-$(GOOS)-$(GOARCH) cmd/module/cmd.go

edit-android:
	# temporary command to get an android-compatible rdk branch
	# todo: dedup with rdk-droid command
	go mod edit -replace=go.viam.com/rdk=github.com/abe-winter/rdk@droid-apk
	go mod tidy

# Build go binary for android
.PHONY: build-android
build-android:
	# if this fails with Camera interfaces, run `make edit-android` first
	GOOS=android GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) \
		CGO_CFLAGS="$(CGO_CFLAGS)" \
		CGO_LDFLAGS="$(CGO_LDFLAGS)" \
		CC=$(CC) \
		go build -v -tags no_cgo \
		-o $(OUTPUT) ./cmd/module/cmd.go

module.tar.gz: $(OUTPUT) run.sh
	# todo: dedup with 'make module' command
	tar czf $@ $^ -C $(FFMPEG_PREFIX) lib

# Create linux AppImage bundle
.PHONY: package
package:
	cd etc && GOARCH=$(GOARCH) ARCH=$(ARCH) appimage-builder --recipe viam-rtsp-appimage.yml

# Push AppImage to target device
push-appimg:
	scp etc/rtsp-module-0.0.1-$(ARCH).AppImage viam@$(TARGET_IP):~/viamrtsp-mod

# Install dependencies
linux-dep:
	sudo apt-get install -qy libswscale-dev libavcodec-dev

FFmpeg:
	# clone ffmpeg in the spot we need
	# todo: maybe make this a submodule
	git clone https://github.com/FFmpeg/FFmpeg -b n6.1.1 --depth 1

# Build FFmpeg for Android
# Requires Android NDK to be installed
.PHONY: ffmpeg-android
ffmpeg-android: FFmpeg
	cd FFmpeg && \
	./configure \
		--prefix=$(FFMPEG_PREFIX) \
		--target-os=android \
		--arch=aarch64 \
		--cpu=armv8-a \
		--cc=$(CC) \
		--cxx=$(CXX) \
		--ar=$AR \
		--ld=$(CC) \
		--ranlib=$(RANLIB) \
		--strip=$(STRIP) \
		--nm=$(NM) \
		--disable-static \
		--enable-shared \
		--disable-doc \
		--disable-ffmpeg \
		--disable-ffplay \
		--disable-ffprobe \
		--disable-avdevice \
		--disable-symver \
		--enable-small \
		--enable-cross-compile \
		--sysroot=$(SYSROOT) && \
	make -j$(shell nproc) && make install

# Push FFmpeg to android device
push-ffmpeg-android:
	adb push $(FFMPEG_PREFIX) android /data/local/tmp/ffmpeg

# Push binary to android device
push-binary-android:
	adb push $(OUTPUT_DIR)/viamrtsp-android-arm64 /data/local/tmp/viamrtsp-android-arm64

# Get android compatible rdk
# Need to add 'replace go.viam.com/rdk => ./rdk-droid` to go.mod
rdk-droid:
	git clone git@github.com:abe-winter/rdk.git rdk-droid && \
		cd rdk-droid && \
		git checkout droid-apk

# Fake cam for testing (h264)
fake-cam:
	ffmpeg -re -f lavfi -i testsrc=size=640x480:rate=30 -vcodec libx264 -b:v 900k -f rtsp -rtsp_transport tcp rtsp://localhost:8554/live.stream

# RTSP server for testing
# need docker installed
rtsp-server:
	cd etc && docker run --rm -it -v rtsp-simple-server.yml:/rtsp-simple-server.yml -p 8554:8554 aler9/rtsp-simple-server:v1.3.0

test:
	go test

lint:
	gofmt -w -s .

updaterdk:
	go get go.viam.com/rdk@latest
	go mod tidy

module: bin/viamrtsp
	tar czf module.tar.gz bin/viamrtsp
