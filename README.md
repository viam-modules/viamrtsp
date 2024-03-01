
Prep linux
===

* sudo apt install libswscale-dev libavcodec-dev

Build for Linux
===

* Start canon `canon -arch arm64` or `canon -arch amd64`
* Install deps `make linux-dep`
* Create golang binary `make build-linux`
* Create appimage `make package`

Notes
===
* Heavily cribbed from https://github.com/bluenviron/gortsplib/blob/main/examples/client-read-format-h264-convert-to-jpeg/main.go


Sample Config
===
```
{
      "rtsp_address": "rtsp://foo:bar@192.168.10.10:554/stream"
}
```
