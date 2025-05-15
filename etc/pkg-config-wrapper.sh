#!/bin/sh

# force pkg-config to look in your x264 build’s pkgconfig dir
# ffmpeg configure overrides PKG_CONFIG_PATH and PKG_CONFIG_LIBDIR
# so we need to add wrapper to inject our path
export PKG_CONFIG_PATH=/host/x264/windows-amd64/build/lib/pkgconfig
# now invoke the real pkg-confi
exec /usr/bin/pkg-config "$@"
