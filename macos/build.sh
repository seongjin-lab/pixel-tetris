#!/bin/sh
set -eu

project=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd -P)
cd "$project"
app="$project/build/픽셀 테트리스.app"
resources="$app/Contents/Resources"
iconset="$project/build/PixelTetris.iconset"
mkdir -p "$resources" "$app/Contents/MacOS" "$iconset"

for size in 16 32 128 256 512; do
    double=$((size * 2))
    sips -z "$size" "$size" macos/PixelTetris.png --out "$iconset/icon_${size}x${size}.png" >/dev/null
    sips -z "$double" "$double" macos/PixelTetris.png --out "$iconset/icon_${size}x${size}@2x.png" >/dev/null
done
iconutil -c icns "$iconset" -o "$resources/PixelTetris.icns"

cp macos/Info.plist "$app/Contents/Info.plist"
"${GO:-go}" build -trimpath -o "$resources/pixel-tetris" .
swiftc -O macos/Window.swift -o "$app/Contents/MacOS/PixelTetrisLauncher"
codesign --force --sign - "$app"
codesign --verify --deep --strict "$app"
printf '%s\n' "$app"
