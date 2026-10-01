#!/bin/sh
set -eu
REPO_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
. "$REPO_ROOT/apps/macos/toolchain.sh"
cd "$REPO_ROOT"
BUILD_DIR="$REPO_ROOT/build/macos-tests"
[ -f "$BUILD_DIR/liborangecount.a" ] || { echo 'Run make mac-test first' >&2; exit 1; }
"$OC_SWIFT" -O -parse-as-library -swift-version 6 -target "$OC_TARGET" -sdk "$SDKROOT" \
  -I "$BUILD_DIR" -L "$BUILD_DIR" -lorangecount \
  apps/macos/src/Models.swift apps/macos/src/BridgeClient.swift \
  apps/macos/tests/SessionBenchmark.swift -o "$BUILD_DIR/SessionBenchmark"
if [ "$#" -eq 0 ]; then
  mkdir -p build/macos-benchmark
  "$OC_GO" run ./tools/perfbench -skip-compare -out build/macos-benchmark/large.bean
  echo 'Scenario: mixed currencies without exchange rates'
  "$BUILD_DIR/SessionBenchmark" "$REPO_ROOT/build/macos-benchmark/large.bean"
  # Keep the generated transaction/lot counts; make all cash units USD so the
  # supported at-cost view returns populated charts. This is a separate input,
  # not a currency conversion or a modification of a user's ledger.
  sed -e 's/USD,EUR,CNY/USD/g' -e 's/EUR/USD/g' -e 's/CNY/USD/g' \
    build/macos-benchmark/large.bean > build/macos-benchmark/usd.bean
  echo 'Scenario: same transaction count with USD cash units and USD costs'
  set -- "$REPO_ROOT/build/macos-benchmark/usd.bean"
fi
"$BUILD_DIR/SessionBenchmark" "$1"
