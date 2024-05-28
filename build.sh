#!/bin/bash
set -e

sudo apt-get update
sudo apt-get install -y libfuse2 ffmpeg pkg-config

make module

UNAME_S=$(uname -s)
UNAME_M=$(uname -m)

output_dir="./bin/${UNAME_S}-${UNAME_M}"
archive_name="module.tar.gz"

if [ -f "${output_dir}/${archive_name}" ]; then 
    mkdir -p ./dist
    mv "${output_dir}/${archive_name}" "./dist/archive.tar.gz"
    echo "Moved ${archive_name} to the current directory."
else
    echo "Error: ${archive_name} not found in ${output_dir}." >&2
    exit 1
fi
