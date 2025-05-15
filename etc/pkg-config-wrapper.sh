#!/bin/sh

# force pkg-config to look in your x264 build’s pkgconfig dir
# ffmpeg configure overrides PKG_CONFIG_PATH and PKG_CONFIG_LIBDIR
# so we need to add wrapper to inject our path
# SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
# PARENT_DIR="$(dirname "$SCRIPT_DIR")"
script_path="$0"
# Handle relative path
case "$script_path" in
    /*) ;; # Absolute path, do nothing
    *) script_path="$(pwd)/$script_path" ;; # Relative path, make absolute
esac
SCRIPT_DIR="$(dirname "$script_path")"
PARENT_DIR="$(dirname "$SCRIPT_DIR")"
export PKG_CONFIG_PATH="$PARENT_DIR/x264/windows-amd64/build/lib/pkgconfig"
echo "Using PKG_CONFIG_PATH=$PKG_CONFIG_PATH" >&2
# now invoke the real pkg-config
exec /usr/bin/pkg-config "$@"
