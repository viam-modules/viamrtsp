
Build
===

The binary is statically linked with [FFmpeg v6.1](https://github.com/FFmpeg/FFmpeg/tree/release/6.1), eliminating the need for separate FFmpeg installation on target machines.

We support building this module using the Makefile for the following targets:
|        | Linux | Android  | Darwin |
|--------|-------|----------|--------|
| arm64  | ✅    | ✅       | ✅     |
| amd64  | ✅    | ❌       | ✅     |


* Build for Linux targets:
    * Install canon: `go install github.com/viamrobotics/canon@latest`
    * Startup canon dev container.
        * Linux/Arm64: `canon -profile viam-rtsp-antique -arch arm64`
        * Linux/Amd64: `canon -profile viam-rtsp-antique -arch amd64`
    * Build binary: `make`
* Build for MacOS target:
    * Build binary: `make`
* Build for Android target:
    * Cross-compile from Linux or Darwin host.
    * Build binary: `TARGET_OS=android TARGET_ARCH=arm64 make`
* Binary will be in `bin/<OS>-<CPU>/viamrtsp`
* Clean up build artifacts: `make clean`
* Clean up all files not tracked in git: `make clean-all`

Sample Config
===
```
 {
  "name": "rtsp-1",
  "namespace": "rdk",
  "type": "camera",
  "model": "erh:viamrtsp:rtsp",
  "attributes": {
    "rtp_passthrough": true,
    "rtsp_address": "rtsp://foo:bar@192.168.10.10:554/stream"
  }
}
```

Models:
===
* `erh:viamrtsp:rtsp` - Codec agnostic. Will auto detect the codec of the `rtsp_address`.
* `erh:viamrtsp:rtsp-h264` - Only supports H264 codec.
* `erh:viamrtsp:rtsp-h265` - Only supports H265 codec.
* `erh:viamrtsp:rtsp-mjpeg` - Only supports M-JPEG codec.

Notes
===
* `rtp_passthrough` (which improves video streaming efficiency) is supported with the H264 codec if the `rtp_passthrough` attrbute is set to `true`
* Non fatal LibAV errors are suppressed unles the module is run in debug mode.
* Heavily cribbed from [gortsplib](https://github.com/bluenviron/gortsplib) examples:
    * [H264 stream to JPEG](https://github.com/bluenviron/gortsplib/blob/main/examples/client-play-format-h264-convert-to-jpeg/main.go)
    * [H265 stream to JPEG](https://github.com/bluenviron/gortsplib/blob/main/examples/client-play-format-h265-convert-to-jpeg/main.go)
