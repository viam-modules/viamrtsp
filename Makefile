SOURCE_OS ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
SOURCE_ARCH ?= $(shell uname -m)
TARGET_OS ?= $(SOURCE_OS)
TARGET_ARCH ?= $(SOURCE_ARCH)
normalize_arch = $(if $(filter aarch64,$(1)),arm64,$(if $(filter x86_64,$(1)),amd64,$(1)))
# Normalize the source and target arch to arm64 or amd64 for compatibility with go build.
SOURCE_ARCH := $(call normalize_arch,$(SOURCE_ARCH))
TARGET_ARCH := $(call normalize_arch,$(TARGET_ARCH))

# Here we will handle error cases where the host/target combinations are not supported.
SUPPORTED_COMBINATIONS := \
    linux-arm64-linux-arm64 \
    linux-amd64-linux-amd64 \
    linux-amd64-android-arm64 \
    darwin-arm64-darwin-arm64 \
    darwin-arm64-android-arm64 \
    linux-amd64-windows-amd64
CURRENT_COMBINATION := $(SOURCE_OS)-$(SOURCE_ARCH)-$(TARGET_OS)-$(TARGET_ARCH)
ifneq (,$(filter $(CURRENT_COMBINATION),$(SUPPORTED_COMBINATIONS)))
    $(info Supported combination: $(CURRENT_COMBINATION))
else
    $(error Error: Unsupported combination: $(CURRENT_COMBINATION))
endif

ifeq ($(SOURCE_OS),linux)
    NPROC ?= $(shell nproc)
else ifeq ($(SOURCE_OS),darwin)
    NPROC ?= $(shell sysctl -n hw.ncpu)
else
    NPROC ?= 1
endif

BIN_OUTPUT_PATH = bin/$(TARGET_OS)-$(TARGET_ARCH)
ifeq ($(TARGET_OS),windows)
	BIN_SUFFIX := .exe
endif
BIN_VIAMRTSP := $(BIN_OUTPUT_PATH)/viamrtsp$(BIN_SUFFIX)
BIN_DISCOVERY := $(BIN_OUTPUT_PATH)/discovery$(BIN_SUFFIX)
TOOL_BIN = bin/gotools/$(shell uname -s)-$(shell uname -m)

FFMPEG_TAG ?= n6.1
FFMPEG_VERSION ?= $(shell pwd)/FFmpeg/$(FFMPEG_TAG)
FFMPEG_VERSION_PLATFORM ?= $(FFMPEG_VERSION)/$(TARGET_OS)-$(TARGET_ARCH)
FFMPEG_BUILD ?= $(FFMPEG_VERSION_PLATFORM)/build
FFMPEG_LIBS=    libavformat \
                libavcodec  \
                libavutil   \
                libswscale  \

FFMPEG_OPTS ?= --prefix=$(FFMPEG_BUILD) \
--enable-static \
--disable-shared \
--disable-programs \
--disable-doc \
--disable-everything \
--enable-bsf=h264_mp4toannexb \
--enable-decoder=mpeg4 \
--enable-decoder=h264 \
--enable-decoder=hevc \
--enable-decoder=mjpeg \
--enable-demuxer=concat \
--enable-demuxer=mov \
--enable-demuxer=mp4 \
--enable-demuxer=segment \
--enable-encoder=libx264 \
--enable-encoder=mjpeg \
--enable-encoder=mpeg4 \
--enable-gpl \
--enable-libx264 \
--enable-muxer=mp4 \
--enable-muxer=segment \
--enable-muxer=mov \
--enable-network \
--enable-parser=h264 \
--enable-parser=hevc \
--enable-protocol=concat \
--enable-protocol=crypto \
--enable-protocol=file \

# Add linker flag -checklinkname=0 for anet https://github.com/wlynxg/anet?tab=readme-ov-file#how-to-build-with-go-1230-or-later.
PKG_CONFIG_PATH = $(FFMPEG_BUILD)/lib/pkgconfig
CGO_CFLAGS = $(shell PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) pkg-config --cflags $(FFMPEG_LIBS))
ifeq ($(SOURCE_OS),linux)
	SUBST = -l:libx264.a
endif
ifeq ($(SOURCE_OS),darwin)
	SUBST = $(HOMEBREW_PREFIX)/Cellar/x264/r3108/lib/libx264.a
endif
CGO_LDFLAGS = $(subst -lx264, $(SUBST),$(shell PKG_CONFIG_PATH=$(PKG_CONFIG_PATH) pkg-config --libs $(FFMPEG_LIBS))) 
ifeq ($(TARGET_OS),windows)
	CGO_LDFLAGS += -static -static-libgcc -static-libstdc++
endif
ifeq ($(SOURCE_OS),darwin)
ifeq ($(shell brew list | grep -w x264 > /dev/null; echo $$?), 1)
	brew update && brew install x264
endif
endif

# If we are building for android, we need to set the correct flags
# and toolchain paths for FFMPEG and go binary cross-compilation.
ifeq ($(TARGET_OS),android)
# amd64 android targets have not been tested, so we do not support them for now.
ifeq ($(TARGET_ARCH),arm64)
    # Android build doesn't support most of our cgo libraries, so we use the no_cgo flag.
    GO_TAGS ?= -tags no_cgo
    # We need the go build command to think it's in cgo mode to support android NDK cross-compilation.
    export CGO_ENABLED = 1
    NDK_ROOT ?= $(shell pwd)/ndk/$(SOURCE_OS)/android-ndk-r26
    # We do not need to handle source arch for toolchain paths.
    # On darwin host, android toolchain binaries and libs are mach-O universal
    # with 2 architecture targets: x86_64 and arm64.
    CC = $(NDK_ROOT)/toolchains/llvm/prebuilt/$(SOURCE_OS)-x86_64/bin/aarch64-linux-android$(API_LEVEL)-clang
    export CC
    # Android API level is an integer value that uniquely identifies the revision of the Android platform framework API.
    # We use API level 30 as the default value. You can change it by setting the API_LEVEL variable.
    API_LEVEL ?= 30
    FFMPEG_OPTS += --target-os=android \
                   --arch=aarch64 \
                   --cpu=armv8-a \
                   --enable-cross-compile \
                   --cc=$(CC)
endif
endif

ifeq ($(TARGET_OS),windows)
ifeq ($(SOURCE_OS),linux)
ifeq ($(TARGET_ARCH),amd64)
    GO_TAGS ?= -tags no_cgo
    X264_ROOT ?= $(shell pwd)/x264/windows-amd64
    X264_BUILD_DIR ?= $(X264_ROOT)/build
    # We need the go build command to think it's in cgo mode
    export CGO_ENABLED = 1
    # mingw32 flags refer to 64 bit windows target
    export CC=/usr/bin/x86_64-w64-mingw32-gcc
    export CXX=/usr/bin/x86_64-w64-mingw32-g++
    export AS=x86_64-w64-mingw32-as
    export AR=x86_64-w64-mingw32-ar
    export RANLIB=x86_64-w64-mingw32-ranlib
    export LD=x86_64-w64-mingw32-ld
    export STRIP=x86_64-w64-mingw32-strip
    FFMPEG_OPTS += --target-os=mingw32 \
                   --arch=x86 \
                   --cpu=x86-64 \
                   --cross-prefix=x86_64-w64-mingw32- \
                   --enable-cross-compile \
                   --pkg-config=$(shell pwd)/etc/pkg-config-wrapper.sh
endif
endif
endif

.PHONY: build-ffmpeg tool-install gofmt lint test profile-cpu profile-memory update-rdk module clean clean-all

all: $(BIN_VIAMRTSP) $(BIN_DISCOVERY)

# We set GOOS, GOARCH, GO_TAGS, and GO_LDFLAGS to support cross-compilation for android targets.
$(BIN_VIAMRTSP): build-ffmpeg *.go cmd/module/*.go
	CGO_LDFLAGS="$(CGO_LDFLAGS)" \
	CGO_CFLAGS="$(CGO_CFLAGS)" \
	GOOS=$(TARGET_OS) \
	GOARCH=$(TARGET_ARCH) \
	go build $(GO_TAGS) -ldflags="-checklinkname=0" -o $(BIN_VIAMRTSP) cmd/module/cmd.go

$(BIN_DISCOVERY): build-ffmpeg *.go cmd/discovery/*.go
	CGO_LDFLAGS="$(CGO_LDFLAGS)" \
	CGO_CFLAGS="$(CGO_CFLAGS)" \
	GOOS=$(TARGET_OS) \
	GOARCH=$(TARGET_ARCH) go build $(GO_TAGS) -ldflags="-checklinkname=0" -o $(BIN_DISCOVERY) cmd/discovery/cmd.go

tool-install:
	GOBIN=`pwd`/$(TOOL_BIN) go install \
		github.com/edaniels/golinters/cmd/combined \
		github.com/golangci/golangci-lint/cmd/golangci-lint \
		github.com/rhysd/actionlint/cmd/actionlint

gofmt:
	gofmt -w -s .

lint: gofmt tool-install build-ffmpeg
	CGO_CFLAGS=$(CGO_CFLAGS) GOFLAGS=$(GOFLAGS) $(TOOL_BIN)/golangci-lint run -v --fix --config=./etc/.golangci.yaml --timeout=2m

test: build-ffmpeg
	CGO_CFLAGS="$(CGO_CFLAGS)" CGO_LDFLAGS="$(CGO_LDFLAGS)" go test -ldflags="-checklinkname=0" -race -v ./...

profile-cpu:
	go test -v -cpuprofile cpu.prof -run "^TestRTSPCameraPerformance$$" -bench github.com/viam-modules/viamrtsp
	go tool pprof -top cpu.prof > cpu-profile.txt
	rm cpu.prof

profile-memory:
	go test -v -memprofile mem.prof -run "^TestRTSPCameraPerformance$$" -bench github.com/viam-modules/viamrtsp
	go tool pprof -top mem.prof > mem-profile.txt
	rm mem.prof

update-rdk:
	go get go.viam.com/rdk@latest
	go mod tidy

$(FFMPEG_VERSION_PLATFORM):
	git clone https://github.com/FFmpeg/FFmpeg.git --depth 1 --branch $(FFMPEG_TAG) $(FFMPEG_VERSION_PLATFORM)

$(FFMPEG_BUILD): $(FFMPEG_VERSION_PLATFORM)
# Only need nasm to build assembly kernels for amd64 targets.
ifeq ($(SOURCE_OS),linux)
ifeq ($(TARGET_OS),linux)
ifeq ($(shell dpkg -l | grep -w x264 > /dev/null; echo $$?), 1)
	sudo apt update && sudo apt install -y libx264-dev
endif
endif
ifeq ($(SOURCE_ARCH),amd64)
	which nasm || (sudo apt update && sudo apt install -y nasm)
endif
endif
ifeq ($(SOURCE_OS),darwin)
ifeq ($(shell brew list | grep -w x264 > /dev/null; echo $$?), 1)
	brew update && brew install x264
endif
endif
	cd $(FFMPEG_VERSION_PLATFORM) && ./configure $(FFMPEG_OPTS) && $(MAKE) -j$(NPROC) && $(MAKE) install

build-ffmpeg: $(NDK_ROOT) $(X264_BUILD_DIR)
# Only need nasm to build assembly kernels for amd64 targets.
ifeq ($(SOURCE_OS),linux)
ifeq ($(SOURCE_ARCH),amd64)
	which nasm || (sudo apt update && sudo apt install -y nasm)
endif
endif
	$(MAKE) $(FFMPEG_BUILD)

# Warning: This will download a large file (1.5GB) and extract the contents. If you have 
# already downloaded the NDK, you can set the NDK_ROOT variable to the path of the NDK.
$(NDK_ROOT):
ifeq ($(TARGET_OS),android)
ifeq ($(SOURCE_OS),darwin)
	wget https://dl.google.com/android/repository/android-ndk-r26d-darwin.dmg && \
	hdiutil attach android-ndk-r26d-darwin.dmg && \
	mkdir -p $(NDK_ROOT) && \
	cp -R "/Volumes/Android NDK r26d"/AndroidNDK11579264.app/Contents/NDK/* $(NDK_ROOT) && \
	hdiutil detach "/Volumes/Android NDK r26d" && \
	rm android-ndk-r26d-darwin.dmg
endif
ifeq ($(SOURCE_OS),linux)
ifeq ($(SOURCE_ARCH),amd64)
	sudo apt update && sudo apt install -y unzip && \
	wget https://dl.google.com/android/repository/android-ndk-r26-linux.zip && \
	mkdir -p $(dir $(NDK_ROOT)) && \
	yes A | unzip android-ndk-r26-linux.zip -d $(dir $(NDK_ROOT)) && \
	rm android-ndk-r26-linux.zip
endif
endif
endif

$(X264_ROOT):
ifeq ($(TARGET_OS),windows)
ifeq ($(SOURCE_OS),linux)
ifeq ($(TARGET_ARCH),amd64)
	git clone https://code.videolan.org/videolan/x264.git $(X264_ROOT)
endif
endif
endif

$(X264_BUILD_DIR): $(X264_ROOT)
ifeq ($(TARGET_OS),windows)
ifeq ($(SOURCE_OS),linux)
ifeq ($(TARGET_ARCH),amd64)
ifeq ($(shell which x86_64-w64-mingw32-gcc > /dev/null; echo $$?), 1)
	$(info MinGW cross compiler not found, installing...)
	sudo apt-get update && sudo apt-get install -y mingw-w64
endif
	cd $(X264_ROOT) && \
	./configure \
		--host=x86_64-w64-mingw32 \
		--cross-prefix=x86_64-w64-mingw32- \
		--prefix=$(X264_BUILD_DIR) \
		--enable-static \
		--disable-opencl \
		--disable-asm && \
	make -j$(NPROC) && \
	make install
endif
endif
endif

videostore/buf.lock: videostore/buf.yaml
	cd videostore && /home/viam/go/bin/buf mod update

videostore/src/videostore_api_go/grpc/videostore.pb.go: videostore/src/proto/videostore.proto videostore/src/proto/buf.gen.yaml videostore/buf.lock
	cd videostore && /home/viam/go/bin/buf generate buf.build/googleapis/googleapis --template ./src/proto/buf.gen.yaml  -o ./src
	cd videostore && /home/viam/go/bin/buf generate --template ./src/proto/buf.gen.yaml --path ./src/proto -o ./src

generate: videostore/src/videostore_api_go/grpc/videostore.pb.go

module: $(BIN_VIAMRTSP)
	cp $(BIN_VIAMRTSP) bin/viamrtsp$(BIN_SUFFIX)
	tar czf module.tar.gz bin/viamrtsp$(BIN_SUFFIX)
	rm bin/viamrtsp$(BIN_SUFFIX)

clean:
	rm -rf $(BIN_VIAMRTSP) $(BIN_DISCOVERY) module.tar.gz

clean-all:
	rm -rf FFmpeg
	rm -rf x264
	git clean -fxd
