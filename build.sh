#!/bin/bash
set -e

sudo apt-get update
sudo apt-get install -y libfuse2 ffmpeg pkg-config

make module.tar.gz
