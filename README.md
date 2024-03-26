
Prep linux
===

* sudo apt install libswscale-dev libavcodec-dev libavformat-dev libavutil-dev

Notes
===
* Heavily cribbed from https://github.com/bluenviron/gortsplib/blob/main/examples/client-read-format-h264-convert-to-jpeg/main.go

FFmpeg
===

This project is designed to work with FFmpeg version 6.1. During the build process of the module, platform-specific static builds of FFmpeg 6.1 will be created. This ensures that the module is always using the correct version of FFmpeg, regardless of what other versions might be installed on your system.

Sample Config
===
```
{
      "rtsp_address": "rtsp://foo:bar@192.168.10.10:554/stream"
}
```
