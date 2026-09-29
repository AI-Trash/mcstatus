#!/usr/bin/env bash
set -e

# Resolve real go binary if not provided
if [ -z "$GO_BIN" ]; then
    GO_BIN=$(command -v go || which go || echo "go")
fi

# Execute original go command
"$GO_BIN" "$@"

# If this is a build command, find the output binary and pack it
if [ "$1" = "build" ]; then
    prev=""
    out=""
    for arg in "$@"; do
        if [ "$prev" = "-o" ]; then
            out="$arg"
            break
        fi
        prev="$arg"
    done

    if [ -n "$out" ] && [ -f "$out" ] && command -v upx >/dev/null 2>&1; then
        echo "[go-upx] Compressing $out with UPX..."
        upx --best --lzma "$out" || true
    fi
fi
