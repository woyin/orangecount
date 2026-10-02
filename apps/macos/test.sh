#!/bin/sh
set -eu

REPO_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
. "$REPO_ROOT/apps/macos/toolchain.sh"
BUILD_DIR="$REPO_ROOT/build/macos-tests"
mkdir -p "$BUILD_DIR"
cd "$REPO_ROOT"
"$OC_SWIFT" -O -parse-as-library -swift-version 6 -target "$OC_TARGET" -sdk "$SDKROOT" \
    apps/macos/src/AnalysisFileExport.swift apps/macos/tests/ExportTests.swift -o "$BUILD_DIR/ExportTests"
"$BUILD_DIR/ExportTests"
"$OC_GO" build -trimpath -ldflags=-buildid= -buildmode=c-archive -o "$BUILD_DIR/liborangecount.a" ./internal/bridge
cp apps/macos/src/Bridge.h apps/macos/src/module.modulemap "$BUILD_DIR/"
"$OC_SWIFT" -O -parse-as-library -swift-version 6 -target "$OC_TARGET" -sdk "$SDKROOT" -I "$BUILD_DIR" -L "$BUILD_DIR" -lorangecount \
    apps/macos/src/Models.swift apps/macos/src/BridgeClient.swift \
    apps/macos/src/AnalysisFileExport.swift apps/macos/tests/ContractTests.swift -o "$BUILD_DIR/ContractTests"
"$BUILD_DIR/ContractTests" "$REPO_ROOT"
"$OC_SWIFT" -O -parse-as-library -swift-version 6 -target "$OC_TARGET" -sdk "$SDKROOT" -I "$BUILD_DIR" -L "$BUILD_DIR" -lorangecount \
    apps/macos/src/Models.swift apps/macos/src/BridgeClient.swift \
    apps/macos/src/AnalysisFileExport.swift apps/macos/src/AttachmentRootPreferences.swift apps/macos/src/LedgerStore.swift apps/macos/tests/StateTests.swift -o "$BUILD_DIR/StateTests"
"$BUILD_DIR/StateTests"
"$OC_SWIFT" -O -parse-as-library -swift-version 6 -target "$OC_TARGET" -sdk "$SDKROOT" -I "$BUILD_DIR" -L "$BUILD_DIR" -lorangecount \
    apps/macos/src/Models.swift apps/macos/src/BridgeClient.swift \
    apps/macos/src/AnalysisFileExport.swift apps/macos/src/AttachmentRootPreferences.swift apps/macos/src/LedgerStore.swift apps/macos/tests/NativeClientTests.swift -o "$BUILD_DIR/NativeClientTests"
"$BUILD_DIR/NativeClientTests" "$REPO_ROOT"
