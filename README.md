
Prep linux
===

* sudo apt install libswscale-dev libavcodec-dev

Build for Linux
===

* Start canon `canon -arch arm64` or `canon -arch amd64`
* Install deps `make linux-dep`
* Create golang binary `make build-linux`
* Create appimage `make package`

Build for Android
===
* Install android specific RDK branch `make rdk-droid`
* Build Android specific FFmpeg `make android-ffmpeg`
* Build Android specific golang binary `make build-android`
* Move FFmpeg lib onto device `make push-ffmpeg-android`
* Move golang binary onto device `make push-binary-android`
* Include `"env": {"LD_LIBRARY_PATH": "/data/local/tmp/ffmpeg/lib"}` in module config

Test RTSP Cam
===

* Start rtsp server on same device as viam-server `make rtsp-server`
* Start fake input camera `make fake-cam`
* Configure rtsp component `"rtsp_address": "rtsp://localhost:8554/live.stream"`

Notes
===
* Heavily cribbed from https://github.com/bluenviron/gortsplib/blob/main/examples/client-read-format-h264-convert-to-jpeg/main.go


