#!/bin/sh
# Copyright 2026 OrangeCount contributors
# Licensed under the Apache License, Version 2.0.
# Read-only private check. No GUI, source edits, benchmark or lifecycle soak.
set -eu
[ -n "${ORANGECOUNT_PRIVATE_LEDGER:-}" ] || { echo 'Set ORANGECOUNT_PRIVATE_LEDGER to an external absolute ledger path' >&2; exit 2; }
REPO_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
. "$REPO_ROOT/apps/macos/toolchain.sh"
cd "$REPO_ROOT"
sh apps/macos/test.sh
BUILD_DIR="$REPO_ROOT/build/macos-tests"
"$OC_SWIFT" -O -parse-as-library -swift-version 6 -target "$OC_TARGET" -sdk "$SDKROOT" \
  -I "$BUILD_DIR" -L "$BUILD_DIR" -lorangecount \
  apps/macos/src/Models.swift apps/macos/src/BridgeClient.swift apps/macos/tests/PrivateReadProbe.swift \
  -o "$BUILD_DIR/PrivateReadProbe"
export ORANGECOUNT_NATIVE_PROBE="$BUILD_DIR/PrivateReadProbe"
exec sh tools/reference/differential.sh
