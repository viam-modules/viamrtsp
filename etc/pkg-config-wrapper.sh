#!/bin/sh

# force pkg-config to look at x264 build pkgconfig dir
# ffmpeg configure overrides PKG_CONFIG_PATH and PKG_CONFIG_LIBDIR
# so we need to add a wrapper to inject the path
script_path="$0"
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
