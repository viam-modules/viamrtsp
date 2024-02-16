
Prep linux
===

* sudo apt install libswscale-dev libavcodec-dev

Build Mac
===

* Start canon `canon -arch arm64`
* Install deps `make linux-deps`
* Create golang binary `make build`
* Create appimage `make package`

Notes
===
* Heavily cribbed from https://github.com/bluenviron/gortsplib/blob/main/examples/client-read-format-h264-convert-to-jpeg/main.go


