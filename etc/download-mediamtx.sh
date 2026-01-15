#!/bin/bash
# Download and extract mediamtx for the current platform

set -e

VERSION="${MEDIAMTX_VERSION:-v1.9.0}"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Normalize arch
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64v8" ;;
    arm64) ARCH="arm64" ;; # Darwin uses arm64
esac

# Determine URL based on OS
case "$OS" in
    linux)
        if [ "$ARCH" = "arm64" ]; then
            ARCH="arm64v8"
        fi
        URL="https://github.com/bluenviron/mediamtx/releases/download/${VERSION}/mediamtx_${VERSION}_linux_${ARCH}.tar.gz"
        ;;
    darwin)
        URL="https://github.com/bluenviron/mediamtx/releases/download/${VERSION}/mediamtx_${VERSION}_darwin_${ARCH}.tar.gz"
        ;;
    mingw*|msys*|cygwin*)
        URL="https://github.com/bluenviron/mediamtx/releases/download/${VERSION}/mediamtx_${VERSION}_windows_amd64.zip"
        ;;
    *)
        # Check for Windows environment variables
        if [ -n "$WINDIR" ]; then
            URL="https://github.com/bluenviron/mediamtx/releases/download/${VERSION}/mediamtx_${VERSION}_windows_amd64.zip"
        else
            echo "ERROR: Unsupported OS: $OS"
            exit 1
        fi
        ;;
esac

echo "Downloading mediamtx from: $URL"

FILENAME=$(basename "$URL")

if command -v wget &> /dev/null; then
    wget "$URL"
elif command -v curl &> /dev/null; then
    curl -L -o "$FILENAME" "$URL"
else
    echo "ERROR: Neither wget nor curl found"
    exit 1
fi

# Extract based on file extension
case "$FILENAME" in
    *.tar.gz)
        tar -xzf "$FILENAME"
        ;;
    *.zip)
        unzip -o "$FILENAME"
        ;;
esac

echo "mediamtx downloaded and extracted"
