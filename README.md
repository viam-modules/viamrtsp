
Prep linux
===

* sudo apt install libswscale-dev libavcodec-dev

Build for Linux on Mac
===

* Start canon `canon -arch arm64` or `canon -arch amd64`
* Install deps `make linux-deps`
* Create golang binary `make build`
* Create appimage `make package`

Virtual RTSP Cam
===

* Start rtsp server on same device as viam-server `make rtsp-server`
* Start fake input camera `make fake-cam`
* Configure rtsp component `"rtsp_address": "rtsp://localhost:8554/live.stream"`

Notes
===
* Heavily cribbed from https://github.com/bluenviron/gortsplib/blob/main/examples/client-read-format-h264-convert-to-jpeg/main.go


