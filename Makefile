BIN_OUTPUT_PATH = bin/$(shell uname -s)-$(shell uname -m)
UNAME_S ?= $(shell uname -s)
UNAME_M ?= $(shell uname -m)
FFMPEG_TAG ?= n6.1
FFMPEG_VERSION ?= $(shell pwd)/FFmpeg/$(FFMPEG_TAG)
FFMPEG_VERSION_PLATFORM ?= $(FFMPEG_VERSION)/$(UNAME_S)-$(UNAME_M)
FFMPEG_BUILD ?= $(FFMPEG_VERSION_PLATFORM)/build
FFMPEG_OPTS ?= --prefix=$(FFMPEG_BUILD) \
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

CGO_LDFLAGS := -L$(FFMPEG_BUILD)/lib
ifeq ($(UNAME_S),Linux)
	CGO_LDFLAGS := "$(CGO_LDFLAGS) -l:libjpeg.a"
endif

.PHONY: build-ffmpeg test lint updaterdk module clean

$(BIN_OUTPUT_PATH)/viamrtsp: build-ffmpeg *.go cmd/module/*.go
	PKG_CONFIG_PATH=$(FFMPEG_BUILD)/lib/pkgconfig \
		CGO_LDFLAGS=$(CGO_LDFLAGS) \
		go build -o $(BIN_OUTPUT_PATH)/viamrtsp cmd/module/cmd.go

lint:
	gofmt -w -s .

updaterdk:
	go get go.viam.com/rdk@latest
	go mod tidy

$(FFMPEG_VERSION_PLATFORM):
	git clone https://github.com/FFmpeg/FFmpeg.git --depth 1 --branch $(FFMPEG_TAG) $(FFMPEG_VERSION_PLATFORM)

$(FFMPEG_BUILD): $(FFMPEG_VERSION_PLATFORM)
	cd $(FFMPEG_VERSION_PLATFORM) && ./configure $(FFMPEG_OPTS) && $(MAKE) -j$(shell nproc) && $(MAKE) install

build-ffmpeg:
ifeq ($(UNAME_S),Linux)
ifeq ($(UNAME_M),x86_64)
	which nasm || (sudo apt update && sudo apt install -y nasm)
endif
endif
	$(MAKE) $(FFMPEG_BUILD)

module: $(BIN_OUTPUT_PATH)/viamrtsp
	tar czf $(BIN_OUTPUT_PATH)/module.tar.gz $(BIN_OUTPUT_PATH)/viamrtsp

clean-ffmpeg-all:
	rm -rf FFmpeg

clean-viamrtsp:
	rm -rf $(BIN_OUTPUT_PATH)/viamrtsp $(BIN_OUTPUT_PATH)/module.tar.gz

clean-all:
	git clean -fxd
