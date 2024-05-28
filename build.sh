#!/bin/bash
set -e

sudo apt-get update
sudo apt-get install -y libfuse2 ffmpeg pkg-config

make module

UNAME_S=$(uname -s)
UNAME_M=$(uname -m)

artifact_dir="./bin/${UNAME_S}-${UNAME_M}"
artifact_name="module.tar.gz"

if [ -f "${artifact_dir}/${artifact_name}" ]; then 
    mkdir -p ./dist
    output_path="./dist/archive.tar.gz"
    mv "${artifact_dir}/${artifact_name}" "${output_path}"
    echo "Moved ${artifact_name} to the current directory."
else
    echo "Error: ${artifact_name} not found in ${artifact_dir}." >&2
    exit 1
fi
