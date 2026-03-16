#!/bin/bash
# Run mediamtx in background

set -e

OS=$(uname -s | tr '[:upper:]' '[:lower:]')

if [[ "$OS" == "mingw"* ]] || [[ "$OS" == "msys"* ]] || [[ "$OS" == "cygwin"* ]] || [[ -n "$WINDIR" ]]; then
    # Windows
    ./mediamtx.exe &
else
    # Linux/macOS
    ./mediamtx &
fi

sleep 2
echo "mediamtx started"
