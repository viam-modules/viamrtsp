
Build
===
The binary is statically linked with [FFmpeg v6.1](https://github.com/FFmpeg/FFmpeg/tree/release/6.1), eliminating the need for separate FFmpeg installation on target machines.
* Build for Linux targets:
    * Install canon: `go install github.com/viamrobotics/canon@latest`
    * Startup canon dev container.
        * Linux/Arm64: `canon -profile viam-rdk-antique -arch arm64`
        * Linux/Amd64: `canon -profile viam-rdk-antique -arch amd64`
    * Build binary: `make bin/viamrtsp`
* Build for MacOS target:
    * Build binary: `make bin/viamrtsp`
* Clean up build artifacts: `make clean`


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
