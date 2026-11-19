#!/bin/bash
# Builds the Windows and Linux binaries into ../dist (pure Go, no cgo: cross-compiles from any OS)
set -euo pipefail
cd "$(dirname "$0")"
mkdir -p ../dist
for t in linux/amd64 linux/arm64 windows/amd64 windows/arm64; do
  os=${t%/*}; arch=${t#*/}; ext=""; ldf="-s -w"
  [ "$os" = windows ] && ext=".exe" && ldf="$ldf -H windowsgui"
  CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath -ldflags "$ldf" -o "../dist/brightcontrol-$os-$arch$ext" .
done
echo "✅ dist/brightcontrol-{linux,windows}-{amd64,arm64}"
