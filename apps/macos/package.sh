#!/bin/sh
set -eu
REPO_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$REPO_ROOT"
sh apps/macos/check-bundle.sh
OC_VERSION=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' build/OrangeCount.app/Contents/Info.plist)
mkdir -p dist/macos
cp docs/native-macos-support.md dist/macos/SUPPORT.md
ARCHIVE="OrangeCount-$OC_VERSION-arm64-development.zip"
ditto -c -k --sequesterRsrc --keepParent build/OrangeCount.app "dist/macos/$ARCHIVE"
(cd dist/macos && shasum -a 256 "$ARCHIVE" > "$ARCHIVE.sha256")
printf 'Development package (ad hoc signed; not notarized): dist/macos/%s\n' "$ARCHIVE"
