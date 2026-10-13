#!/bin/sh
set -eu

APP_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$APP_DIR"
BUNDLE="$APP_DIR/build/OrangeCount Go UI QA.app"
EXECUTABLE="$BUNDLE/Contents/MacOS/orangecount-goui"
mkdir -p "$(dirname -- "$EXECUTABLE")"
GOWORK=off CGO_ENABLED=0 go build -trimpath -ldflags='-buildid= -X main.qaBuild=true' -o "$EXECUTABLE" ./cmd/orangecount-goui
cp "$APP_DIR/qa/Info.plist" "$BUNDLE/Contents/Info.plist"
open -n "$BUNDLE"
