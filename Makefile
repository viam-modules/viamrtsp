GOOS=linux
GOARCH=arm64
TARGET_IP ?= 127.0.0.1

.Phony : build

# Build go binary
build:
	go build -v -o bin/viamrtsp-$(GOOS)-$(GOARCH) cmd/module/cmd.go
	
# Create AppImage bundle
package:
	cd etc && sudo appimage-builder --recipe viam-rtsp-arm64.yml

# Push binary to target
push-bin:
	scp bin/viamrtsp-$(GOOS)-$(GOARCH) viam@$(TARGET_IP):~/viamrtsp-$(GOOS)-$(GOARCH)

# Fake cam for testing
fake-cam:
	ffmpeg -re -f lavfi -i testsrc=size=640x480:rate=30 -vcodec libx264 -b:v 900k -f rtsp -rtsp_transport tcp rtsp://localhost:8554/live.stream

# RTSP server for testing
rtsp-server:
	cd etc && docker run --rm -it -v rtsp-simple-server.yml:/rtsp-simple-server.yml -p 8554:8554 aler9/rtsp-simple-server:v1.3.0

# Install dependencies
linux-dep:
	sudo apt install libswscale-dev libavcodec-dev
	
test:
	go test

lint:
	gofmt -w -s .

updaterdk:
	go get go.viam.com/rdk@latest
	go mod tidy

module: bin/viamrtsp
	tar czf module.tar.gz bin/viamrtsp
