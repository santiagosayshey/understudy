#!/bin/bash
# Makes Plex trust Understudy's certificate authority. Mount this file into the
# Plex container's /custom-cont-init.d (linuxserver image) and ca.crt at
# /understudy/ca.crt. It runs on every start, so an image upgrade is trusted
# again before Plex serves anything.
CA=${UNDERSTUDY_CA:-/understudy/ca.crt}
if [ ! -r "$CA" ]; then
    echo "[understudy] $CA is missing; Plex will not trust the proxy"
    exit 0
fi
cp "$CA" /usr/local/share/ca-certificates/understudy.crt
if update-ca-certificates >/dev/null 2>&1; then
    echo "[understudy] certificate authority installed"
else
    echo "[understudy] update-ca-certificates failed; Plex will not trust the proxy"
fi
