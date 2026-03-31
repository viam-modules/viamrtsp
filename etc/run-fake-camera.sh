#!/bin/bash
# Run fake RTSP camera using ffmpeg
# Usage: ./run-fake-camera.sh <codec> <pix_fmt> <transport> [extra_ffmpeg_args]

set -e

CODEC="${1:-libx264}"
PIX_FMT="${2:-yuv420p}"
TRANSPORT="${3:-tcp}"
EXTRA_ARGS="${4:-}"

ffmpeg -re -f lavfi -i testsrc=size=640x480:rate=30 \
    -vcodec "$CODEC" $EXTRA_ARGS \
    -pix_fmt "$PIX_FMT" \
    -f rtsp -rtsp_transport "$TRANSPORT" \
    rtsp://0.0.0.0:8554/live.stream &

sleep 3
echo "Fake RTSP camera started with codec=$CODEC pix_fmt=$PIX_FMT transport=$TRANSPORT"
