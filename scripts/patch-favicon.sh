#!/bin/sh
# Swap toolgui's built-in favicon for ours in a toolgui-wasm build.
set -eu
index="${1:-dist}/index.html"
sed -i 's|^\( *\)<link rel="icon" .*$|\1<link rel="icon" href="assets/favicon.ico" sizes="any" />\n\1<link rel="apple-touch-icon" href="assets/apple-touch-icon.png" />|' "$index"
grep -q 'href="assets/favicon.ico"' "$index" || { echo "favicon link not found in $index" >&2; exit 1; }
