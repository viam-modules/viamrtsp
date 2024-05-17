SOURCE_OS ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
SOURCE_ARCH ?= $(shell uname -m)
TARGET_OS ?= $(SOURCE_OS)
TARGET_ARCH ?= $(SOURCE_ARCH)
ifeq ($(TARGET_ARCH),aarch64)
    TARGET_ARCH = arm64
else ifeq ($(TARGET_ARCH),x86_64)
    TARGET_ARCH = amd64
endif
BIN_OUTPUT_PATH = bin/$(TARGET_OS)-$(TARGET_ARCH)
TOOL_BIN = bin/gotools/$(shell uname -s)-$(shell uname -m)

FFMPEG_TAG ?= n6.1
FFMPEG_VERSION ?= $(shell pwd)/FFmpeg/$(FFMPEG_TAG)
FFMPEG_VERSION_PLATFORM ?= $(FFMPEG_VERSION)/$(TARGET_OS)-$(TARGET_ARCH)
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
CGO_LDFLAGS := -L$(FFMPEG_BUILD)/lib
export PKG_CONFIG_PATH=$(FFMPEG_BUILD)/lib/pkgconfig

# If we are building for android, we need to set the correct flags
# and toolchain paths for FFMPEG and go binary cross-compilation.
# Make sure to install android SDK and NDK before building.
# If you are using a different version of NDK, please set the NDK_VERSION variable.
ifeq ($(TARGET_OS),android)
# x86 android targets have not been tested, so we do not support them for now.
ifeq ($(TARGET_ARCH),arm64)
    # Android build doesn't support most of our cgo libraries, so we use the no_cgo flag.
    GO_TAGS ?= -tags no_cgo
    # We need the go build command to think it's in cgo mode to support android NDK cross-compilation.
    export CGO_ENABLED = 1
    NDK_VERSION ?= 26.1.10909125
	ifeq ($(SOURCE_OS),darwin)
        NDK_ROOT ?= $(HOME)/Library/Android/Sdk/ndk/$(NDK_VERSION)
    else ifeq ($(SOURCE_OS),linux)
        NDK_ROOT ?= $(HOME)/android-ndk-r26
	else
		$(error Error: We do not support the source OS: $(SOURCE_OS) for Android)
    endif
    # We do not need to handle source arch for toolchain paths.
    # On darwin host, android toolchain binaries and libs are mach-O universal
    # with 2 architecture targets: x86_64 and arm64.
	CC = $(NDK_ROOT)/toolchains/llvm/prebuilt/$(SOURCE_OS)-x86_64/bin/aarch64-linux-android$(API_LEVEL)-clang
    export CC
    API_LEVEL ?= 30
    FFMPEG_OPTS += --target-os=android \
                   --arch=aarch64 \
                   --cpu=armv8-a \
                   --enable-cross-compile \
                   --cc=$(CC)
else
    $(error Error: We do not support the target combination: TARGET_OS=$(TARGET_OS), TARGET_ARCH=$(TARGET_ARCH))
endif
endif

ifeq ($(TARGET_OS),linux)
	CGO_LDFLAGS := "$(CGO_LDFLAGS) -l:libjpeg.a"
endif

.PHONY: build-ffmpeg tool-install gofmt lint update-rdk module clean clean-all

# We set GOOS, GOARCH, and GO_TAGS to support cross-compilation for android targets
$(BIN_OUTPUT_PATH)/viamrtsp: build-ffmpeg *.go cmd/module/*.go
	CGO_LDFLAGS=$(CGO_LDFLAGS) \
	GOOS=$(TARGET_OS) GOARCH=$(TARGET_ARCH) go build $(GO_TAGS) -o $(BIN_OUTPUT_PATH)/viamrtsp cmd/module/cmd.go

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
# Only need nasm to build assembly kernels for x86 targets.
ifeq ($(SOURCE_OS),linux)
ifeq ($(SOURCE_ARCH),x86_64)
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
