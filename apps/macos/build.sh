#!/bin/sh
set -eu
REPO_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
. "$REPO_ROOT/apps/macos/toolchain.sh"
cd "$REPO_ROOT"
BUILD_DIR="$REPO_ROOT/build/macos"
APP_BUNDLE="$REPO_ROOT/build/OrangeCount.app"
MACOS_DIR="$APP_BUNDLE/Contents/MacOS"
RESOURCES_DIR="$APP_BUNDLE/Contents/Resources"
OC_VERSION=${ORANGECOUNT_VERSION:-0.1.0}
case "$OC_VERSION" in *[!0-9.]*|'') echo 'Version must contain digits and dots only' >&2; exit 1;; esac
mkdir -p "$BUILD_DIR"
"$OC_GO" build -trimpath -ldflags=-buildid= -buildmode=c-archive -o "$BUILD_DIR/liborangecount.a" ./internal/bridge
cp apps/macos/src/Bridge.h apps/macos/src/module.modulemap "$BUILD_DIR/"
"$OC_SWIFT" -O -parse-as-library -swift-version 6 -target "$OC_TARGET" -sdk "$SDKROOT" \
  -I "$BUILD_DIR" -L "$BUILD_DIR" -lorangecount \
  -framework SwiftUI -framework AppKit -framework Foundation -framework Charts \
  -framework UniformTypeIdentifiers apps/macos/src/*.swift -o "$BUILD_DIR/OrangeCount"
rm -rf "$APP_BUNDLE"
mkdir -p "$MACOS_DIR" "$RESOURCES_DIR"
cp "$BUILD_DIR/OrangeCount" "$MACOS_DIR/OrangeCount"
cp LICENSE NOTICE "$RESOURCES_DIR/"
cat > "$APP_BUNDLE/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleName</key><string>OrangeCount</string>
<key>CFBundleDisplayName</key><string>OrangeCount</string>
<key>CFBundleIdentifier</key><string>org.orangecount.desktop</string>
<key>CFBundleVersion</key><string>$OC_VERSION</string>
<key>CFBundleShortVersionString</key><string>$OC_VERSION</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>CFBundleExecutable</key><string>OrangeCount</string>
<key>LSMinimumSystemVersion</key><string>$OC_MIN_OS</string>
<key>NSHighResolutionCapable</key><true/>
<key>CFBundleDocumentTypes</key><array><dict>
<key>CFBundleTypeName</key><string>Beancount Ledger</string>
<key>CFBundleTypeRole</key><string>Viewer</string>
<key>LSHandlerRank</key><string>Alternate</string>
<key>CFBundleTypeExtensions</key><array><string>bean</string><string>beancount</string></array>
</dict></array>
</dict></plist>
PLIST
{
  echo "version=$OC_VERSION"
  echo "architecture=arm64"
  echo "deployment_target=$OC_MIN_OS"
  echo "revision=${ORANGECOUNT_REVISION:-$(git rev-parse HEAD 2>/dev/null || echo unknown)}"
  if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    if [ -n "$(git status --porcelain)" ]; then echo 'working_tree=dirty'; else echo 'working_tree=clean'; fi
  else echo 'working_tree=source-export'; fi
  "$OC_GO" version
  "$OC_SWIFT" --version
  xcrun --show-sdk-version
} > "$RESOURCES_DIR/build-info.txt"
codesign --force --sign - --timestamp=none "$APP_BUNDLE"
sh apps/macos/check-bundle.sh
printf 'Development app: %s\n' "$APP_BUNDLE"
