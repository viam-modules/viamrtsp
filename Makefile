UNAME_S ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
UNAME_M ?= $(shell uname -m)
TARGET_S ?= $(UNAME_S)
TARGET_M ?= $(UNAME_M)
ifeq ($(TARGET_M),aarch64)
    TARGET_M = arm64
endif
BIN_OUTPUT_PATH = bin/$(TARGET_S)-$(TARGET_M)
TOOL_BIN = bin/gotools/$(shell uname -s)-$(shell uname -m)

FFMPEG_TAG ?= n6.1
FFMPEG_VERSION ?= $(shell pwd)/FFmpeg/$(FFMPEG_TAG)
FFMPEG_VERSION_PLATFORM ?= $(FFMPEG_VERSION)/$(TARGET_S)-$(TARGET_M)
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
               --enable-parser=h264 \
               --enable-parser=hevc

CGO_LDFLAGS := -L$(FFMPEG_PREFIX)/lib

ifeq ($(TARGET_S),android)
	export CGO_ENABLED = 1
	NDK_ROOT ?= $(HOME)/Library/Android/Sdk/ndk/26.1.10909125
	export CC = $(NDK_ROOT)/toolchains/llvm/prebuilt/$(UNAME_S)-x86_64/bin/aarch64-linux-android$(API_LEVEL)-clang
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
	GO_TAGS ?= -tags no_cgo
endif

CGO_LDFLAGS := -L$(FFMPEG_BUILD)/lib
ifeq ($(UNAME_S),linux)
	CGO_LDFLAGS := "$(CGO_LDFLAGS) -l:libjpeg.a"
endif
export PKG_CONFIG_PATH=$(FFMPEG_BUILD)/lib/pkgconfig

.PHONY: build-ffmpeg tool-install gofmt lint update-rdk module clean clean-all

$(BIN_OUTPUT_PATH)/viamrtsp: build-ffmpeg *.go cmd/module/*.go
	CGO_LDFLAGS=$(CGO_LDFLAGS) \
		GOOS=$(TARGET_S) GOARCH=$(TARGET_M) go build $(GO_TAGS) -o $(BIN_OUTPUT_PATH)/viamrtsp cmd/module/cmd.go

tool-install:
	GOBIN=`pwd`/$(TOOL_BIN) go install \
		github.com/edaniels/golinters/cmd/combined \
		github.com/golangci/golangci-lint/cmd/golangci-lint \
		github.com/rhysd/actionlint/cmd/actionlint

gofmt:
	gofmt -w -s .

lint: gofmt tool-install
	go mod tidy
	export pkgs="`go list -f '{{.Dir}}' ./...`" && echo "$$pkgs" | xargs go vet -vettool=$(TOOL_BIN)/combined
	GOGC=50 $(TOOL_BIN)/golangci-lint run -v --fix --config=./etc/.golangci.yaml

update-rdk:
	go get go.viam.com/rdk@latest
	go mod tidy

$(FFMPEG_VERSION_PLATFORM):
	git clone https://github.com/FFmpeg/FFmpeg.git --depth 1 --branch $(FFMPEG_TAG) $(FFMPEG_VERSION_PLATFORM)

$(FFMPEG_BUILD): $(FFMPEG_VERSION_PLATFORM)
	cd $(FFMPEG_VERSION_PLATFORM) && ./configure $(FFMPEG_OPTS) && $(MAKE) -j$(shell nproc) && $(MAKE) install

build-ffmpeg:
ifeq ($(UNAME_S),linux)
ifeq ($(UNAME_M),amd64)
	which nasm || (sudo apt update && sudo apt install -y nasm)
endif
endif
	$(MAKE) $(FFMPEG_BUILD)

module: $(BIN_OUTPUT_PATH)/viamrtsp
	tar czf $(BIN_OUTPUT_PATH)/module.tar.gz $(BIN_OUTPUT_PATH)/viamrtsp

clean:
	rm -rf $(BIN_OUTPUT_PATH)/viamrtsp $(BIN_OUTPUT_PATH)/module.tar.gz

clean-all:
	rm -rf FFmpeg
	git clean -fxd
