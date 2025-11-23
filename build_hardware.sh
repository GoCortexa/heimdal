#!/bin/bash
set -e

echo "Building hardware binary for ARM64 Linux..."

export GOOS=linux
export GOARCH=arm64
export CGO_ENABLED=1
export CC=aarch64-linux-gnu-gcc

mkdir -p bin

go build -trimpath \
  -o bin/heimdal-hardware-arm64 \
  -ldflags="-s -w -extldflags '-static'" \
  ./cmd/heimdal

echo "✓ Hardware binary built successfully: bin/heimdal-hardware-arm64"
ls -lh bin/heimdal-hardware-arm64
