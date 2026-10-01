#!/bin/sh
set -eu
REPO_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
APP_BUNDLE=${1:-"$REPO_ROOT/build/OrangeCount.app"}
PLIST_MIN=$(/usr/libexec/PlistBuddy -c 'Print :LSMinimumSystemVersion' "$APP_BUNDLE/Contents/Info.plist")
BINARY_MIN=$(xcrun vtool -show-build "$APP_BUNDLE/Contents/MacOS/OrangeCount" | awk '/minos/ {print $2}')
[ "$PLIST_MIN" = "$BINARY_MIN" ] || { echo "Deployment mismatch: plist=$PLIST_MIN Mach-O=$BINARY_MIN" >&2; exit 1; }
[ "$(lipo -archs "$APP_BUNDLE/Contents/MacOS/OrangeCount")" = arm64 ]
plutil -lint "$APP_BUNDLE/Contents/Info.plist"
codesign --verify --deep --strict "$APP_BUNDLE"
test -s "$APP_BUNDLE/Contents/Resources/LICENSE"
test -s "$APP_BUNDLE/Contents/Resources/NOTICE"
test -s "$APP_BUNDLE/Contents/Resources/build-info.txt"
echo "Bundle verified: arm64; deployment target $BINARY_MIN (runtime compatibility requires separate testing)"
