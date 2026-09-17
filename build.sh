#!/usr/bin/env bash
# 跨平台编译脚本 - 探海 (Tanhai)
# 生成 Windows / Linux / macOS 等多平台二进制

set -e

VERSION="1.0.0"
APP_NAME="tanhai"
LDFLAGS="-s -w"

echo "[*] 探海 v$VERSION 跨平台编译"
echo ""

# 创建输出目录
mkdir -p dist

# 平台列表 (GOOS/GOARCH)
PLATFORMS=(
    "windows/amd64:tanhai.exe"
    "windows/386:tanhai-x86.exe"
    "windows/arm64:tanhai-arm64.exe"
    "linux/amd64:tanhai-linux"
    "linux/386:tanhai-linux-x86"
    "linux/arm64:tanhai-linux-arm64"
    "darwin/amd64:tanhai-darwin-amd64"
    "darwin/arm64:tanhai-darwin-arm64"
)

for PLATFORM in "${PLATFORMS[@]}"; do
    IFS=':' read -r GOOS_GOARCH OUTPUT <<< "$PLATFORM"
    IFS='/' read -r GOOS GOARCH <<< "$GOOS_GOARCH"

    echo "[*] 编译 $GOOS/$GOARCH -> dist/$OUTPUT"
    GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=0 \
        go build -trimpath -ldflags="$LDFLAGS" \
            -o "dist/$OUTPUT" main.go
done

echo ""
echo "[*] 编译完成，产物在 dist/ 目录："
ls -lh dist/ | tail -n +2

echo ""
echo "[*] 使用示例："
echo "  Windows: dist/tanhai.exe -f urls.txt"
echo "  Linux:   ./dist/tanhai-linux -f urls.txt"
echo "  macOS:   ./dist/tanhai-darwin-arm64 -f urls.txt"