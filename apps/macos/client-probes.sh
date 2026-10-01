#!/bin/sh
set -eu
REPO_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
. "$REPO_ROOT/apps/macos/toolchain.sh"
cd "$REPO_ROOT"
BUILD_DIR="$REPO_ROOT/build/macos-tests"
[ -f "$BUILD_DIR/liborangecount.a" ] || { echo 'Run make mac-test first' >&2; exit 1; }
[ "$#" -ge 1 ] || { echo 'Usage: client-probes.sh synthetic-ledger [cycles]' >&2; exit 1; }
for probe in ClientBurstBenchmark ClientLifecycleProbe; do
  "$OC_SWIFT" -O -parse-as-library -swift-version 6 -target "$OC_TARGET" -sdk "$SDKROOT" \
    -I "$BUILD_DIR" -L "$BUILD_DIR" -lorangecount \
    apps/macos/src/Models.swift apps/macos/src/BridgeClient.swift apps/macos/src/LedgerStore.swift \
    "apps/macos/tests/$probe.swift" -o "$BUILD_DIR/$probe"
done
"$BUILD_DIR/ClientBurstBenchmark" "$1"
"$BUILD_DIR/ClientLifecycleProbe" "$1" "${2:-20}"
