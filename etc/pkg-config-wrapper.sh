#!/bin/sh

# Force pkg-config to look at x264 build pkgconfig dir.
# FFmpeg configure overrides PKG_CONFIG_PATH and PKG_CONFIG_LIBDIR
# so we need to add a wrapper to inject the path at runtime.
PARENT_DIR="$(dirname "$(dirname "$0")")"
export PKG_CONFIG_PATH="$PARENT_DIR/x264/windows-amd64/build/lib/pkgconfig"

# Invoke the real pkg-config with the provided arguments
exec /usr/bin/pkg-config "$@"
