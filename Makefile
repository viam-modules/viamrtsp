BIN_OUTPUT_PATH = bin/$(shell uname -s)-$(shell uname -m)
TOOL_BIN = bin/gotools/$(shell uname -s)-$(shell uname -m)
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
               --enable-parser=h264 \
               --enable-parser=hevc

CGO_LDFLAGS := -L$(FFMPEG_BUILD)/lib
ifeq ($(UNAME_S),Linux)
	CGO_LDFLAGS := "$(CGO_LDFLAGS) -l:libjpeg.a"
endif
export PKG_CONFIG_PATH=$(FFMPEG_BUILD)/lib/pkgconfig

.PHONY: build-ffmpeg tool-install gofmt lint update-rdk module clean clean-all

$(BIN_OUTPUT_PATH)/viamrtsp: build-ffmpeg *.go cmd/module/*.go
	CGO_LDFLAGS=$(CGO_LDFLAGS) \
	go build -o $(BIN_OUTPUT_PATH)/viamrtsp cmd/module/cmd.go

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
ifeq ($(UNAME_S),Linux)
ifeq ($(UNAME_M),x86_64)
	which nasm || (sudo apt update && sudo apt install -y nasm)
endif
endif
	$(MAKE) $(FFMPEG_BUILD)

module: $(BIN_OUTPUT_PATH)/viamrtsp
	cd $(BIN_OUTPUT_PATH) && tar czf module.tar.gz viamrtsp

clean:
	rm -rf $(BIN_OUTPUT_PATH)/viamrtsp $(BIN_OUTPUT_PATH)/module.tar.gz

clean-all:
	rm -rf FFmpeg
	git clean -fxd
