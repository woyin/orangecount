#!/bin/sh
set -eu

APP_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$APP_DIR"
OUT=${OUT:-"$APP_DIR/build/orangecount-goui"}
TARGET_ARCH=${GOARCH:-$(go env GOARCH)}
mkdir -p "$(dirname -- "$OUT")"
GOOS=darwin GOARCH="$TARGET_ARCH" GOWORK=off CGO_ENABLED=0 go build -trimpath -ldflags='-buildid=' -o "$OUT" ./cmd/orangecount-goui
printf 'built %s\n' "$OUT"
