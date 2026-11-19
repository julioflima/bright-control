#!/bin/bash
# Builds BrightControl.app (universal: Apple Silicon + Intel) and dist/BrightControl.zip
set -euo pipefail
cd "$(dirname "$0")"
VERSION="${VERSION:-1.0.0}"
APP=build/BrightControl.app
rm -rf build dist && mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources" dist

for arch in arm64 x86_64; do
  swiftc -O -target "$arch-apple-macos12.0" Sources/main.swift -o "build/BrightControl-$arch"
done
lipo -create build/BrightControl-arm64 build/BrightControl-x86_64 -output "$APP/Contents/MacOS/BrightControl"

sed "s/__VERSION__/$VERSION/" Resources/Info.plist > "$APP/Contents/Info.plist"
[ -f Resources/AppIcon.icns ] && cp Resources/AppIcon.icns "$APP/Contents/Resources/"
codesign --force --sign - "$APP"
(cd build && ditto -c -k --keepParent BrightControl.app ../dist/BrightControl.zip)
echo "✅ $APP  ·  dist/BrightControl.zip"
