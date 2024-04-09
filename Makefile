UNAME_S ?= $(shell uname -s)
UNAME_M ?= $(shell uname -m)
TARGET ?= $(UNAME_S)
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
ifeq ($(TARGET),Android)
	NDK_ROOT ?= $(HOME)/Library/Android/Sdk/ndk/26.1.10909125
	API_LEVEL ?= 30
	FFMPEG_OPTS += --target-os=android \
	               --arch=aarch64 \
				   --cpu=armv8-a \
	               --enable-cross-compile \
	               --sysroot=$(NDK_ROOT)/toolchains/llvm/prebuilt/$(UNAME_S)-x86_64/sysroot \
				   --cc=$(NDK_ROOT)/toolchains/llvm/prebuilt/$(UNAME_S)-x86_64/bin/aarch64-linux-android$(API_LEVEL)-clang \
				   --cxx=$(NDK_ROOT)/toolchains/llvm/prebuilt/$(UNAME_S)-x86_64/bin/aarch64-linux-android$(API_LEVEL)-clang++
endif

CGO_LDFLAGS := -L$(FFMPEG_PREFIX)/lib
ifeq ($(UNAME_S),Linux)
	CGO_LDFLAGS := "$(CGO_LDFLAGS) -l:libjpeg.a"
endif

.PHONY: build-ffmpeg test lint updaterdk module clean

bin/viamrtsp: build-ffmpeg *.go cmd/module/*.go
	PKG_CONFIG_PATH=$(FFMPEG_PREFIX)/lib/pkgconfig \
		CGO_LDFLAGS=$(CGO_LDFLAGS) \
		go build -o bin/viamrtsp cmd/module/cmd.go

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
ifeq ($(UNAME_S),Linux)
ifeq ($(UNAME_M),x86_64)
	which nasm || (sudo apt update && sudo apt install -y nasm)
endif
endif
	$(MAKE) $(FFMPEG_PREFIX)

module: bin/viamrtsp
	tar czf module.tar.gz bin/viamrtsp

clean:
	rm -rf FFmpeg bin/viamrtsp module.tar.gz


TOOLCHAIN := $(NDK_ROOT)/toolchains/llvm/prebuilt/Darwin-x86_64
CC := $(TOOLCHAIN)/bin/aarch64-linux-android30-clang
CXX := $(TOOLCHAIN)/bin/aarch64-linux-android30-clang++
AR := $(TOOLCHAIN)/bin/llvm-ar
LD := $(CC)
RANLIB := $(TOOLCHAIN)/bin/llvm-ranlib
STRIP := $(TOOLCHAIN)/bin/llvm-strip
NM := $(TOOLCHAIN)/bin/llvm-nm
SYSROOT := $(TOOLCHAIN)/sysroot
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
