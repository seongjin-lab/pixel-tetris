#!/bin/sh
set -eu
cd "$SRCROOT/.."
case "$PLATFORM_NAME" in
    iphoneos) target="arm64-apple-ios$IPHONEOS_DEPLOYMENT_TARGET" ;;
    iphonesimulator) target="arm64-apple-ios$IPHONEOS_DEPLOYMENT_TARGET-simulator" ;;
    *) exit 1 ;;
esac
clang=$(xcrun --sdk "$PLATFORM_NAME" --find clang)
sdk=$(xcrun --sdk "$PLATFORM_NAME" --show-sdk-path)
mkdir -p "$DERIVED_FILE_DIR"
CGO_ENABLED=1 GOOS=ios GOARCH=arm64 CC="\"$clang\" -target $target -isysroot \"$sdk\"" \
    "${PIXEL_GO:-go}" build -trimpath -buildmode=c-archive -o "$DERIVED_FILE_DIR/libPixelEngine.a" game.go mobile_ios.go
