
Build for Linux
===

* Install canon: `go install github.com/viamrobotics/canon@latest`
* Startup canon dev container.
    * Linux/arm64: `canon -profile viam-rdk-antique -arch arm64`
    * Linux/amd64: `canon -profile viam-rdk-antique -arch amd64`
* Build binary: `make bin/viamrtsp`


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
