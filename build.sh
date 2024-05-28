#!/bin/bash
set -e

sudo apt-get update
sudo apt-get install -y libfuse2 ffmpeg pkg-config

make module

UNAME_S=$(uname -s)
UNAME_M=$(uname -m)

artifact_path="./bin/${UNAME_S}-${UNAME_M}/module.tar.gz"
output_path="./dist/archive.tar.gz"

if [ -f "${artifact_path}" ]; then 
    mkdir -p ./dist
    mv "${artifact_path}" "${output_path}"
    echo "Successfully moved artifact to the expected output path."
else
    echo "Error: artifact not found in expected path: ${artifact_path}." >&2
    exit 1
fi
