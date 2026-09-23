#!/bin/sh
set -eu

REPO_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
BUILD_DIR="$REPO_ROOT/build/macos"
APP_BUNDLE="$REPO_ROOT/build/OrangeCount.app"
MACOS_DIR="$APP_BUNDLE/Contents/MacOS"
RESOURCES_DIR="$APP_BUNDLE/Contents/Resources"

echo "=== [1/3] Building Go C-Archive Bridge ==="
mkdir -p "$BUILD_DIR"
go build -buildmode=c-archive -o "$BUILD_DIR/liborangecount.a" "$REPO_ROOT/internal/bridge"

echo "=== [2/3] Compiling Native SwiftUI Application (via swiftc) ==="
cp "$REPO_ROOT/apps/macos/src/Bridge.h" "$BUILD_DIR/Bridge.h"
cp "$REPO_ROOT/apps/macos/src/module.modulemap" "$BUILD_DIR/module.modulemap"

swiftc -O \
  -parse-as-library \
  -I "$BUILD_DIR" \
  -L "$BUILD_DIR" \
  -lorangecount \
  -framework SwiftUI \
  -framework AppKit \
  -framework Foundation \
  -framework Charts \
  -framework UniformTypeIdentifiers \
  "$REPO_ROOT/apps/macos/src/Models.swift" \
  "$REPO_ROOT/apps/macos/src/BridgeClient.swift" \
  "$REPO_ROOT/apps/macos/src/AccountTreeView.swift" \
  "$REPO_ROOT/apps/macos/src/JournalView.swift" \
  "$REPO_ROOT/apps/macos/src/DashboardView.swift" \
  "$REPO_ROOT/apps/macos/src/DiagnosticsSheetView.swift" \
  "$REPO_ROOT/apps/macos/src/main.swift" \
  -o "$BUILD_DIR/OrangeCount"

echo "=== [3/3] Assembling macOS .app Bundle ==="
rm -rf "$APP_BUNDLE"
mkdir -p "$MACOS_DIR" "$RESOURCES_DIR"
cp "$BUILD_DIR/OrangeCount" "$MACOS_DIR/OrangeCount"

cat > "$APP_BUNDLE/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key>
    <string>OrangeCount</string>
    <key>CFBundleDisplayName</key>
    <string>OrangeCount</string>
    <key>CFBundleIdentifier</key>
    <string>org.orangecount.desktop</string>
    <key>CFBundleVersion</key>
    <string>1.0.0</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleExecutable</key>
    <string>OrangeCount</string>
    <key>LSMinimumSystemVersion</key>
    <string>13.0</string>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
EOF

echo ""
echo "✅ SUCCESS! OrangeCount.app built successfully without Xcode.app!"
echo "Bundle location: $APP_BUNDLE"
ls -lh "$MACOS_DIR/OrangeCount"
