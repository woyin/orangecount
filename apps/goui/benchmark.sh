#!/bin/sh
set -eu

APP_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$APP_DIR/../.." && pwd)
STAMP=$(date -u '+%Y%m%dT%H%M%SZ')
OUT=${OUT:-"$APP_DIR/build/benchmarks/$STAMP"}
case "$OUT" in
	/*) ;;
	*) OUT="$REPO_ROOT/$OUT" ;;
esac
mkdir -p "$OUT"

LEDGER=${GOUI_BENCH_LEDGER:-"$OUT/transactions-52500.bean"}
case "$LEDGER" in
	/*) ;;
	*) LEDGER="$PWD/$LEDGER" ;;
esac
if [ -z "${GOUI_BENCH_LEDGER:-}" ]; then
	cd "$REPO_ROOT"
	go run ./tools/perfbench -transactions 52500 -skip-compare -out "$LEDGER" > "$OUT/fixture-generation.txt" 2>&1
fi

if command -v shasum >/dev/null 2>&1; then
	LEDGER_SHA=$(shasum -a 256 "$LEDGER" | awk '{print $1}')
else
	LEDGER_SHA=$(sha256sum "$LEDGER" | awk '{print $1}')
fi
MYGO_VERSION=$(sed -n 's/^[[:space:]]*github.com\/egoist\/mygo[[:space:]]\{1,\}//p' "$APP_DIR/go.mod" | head -n 1)
{
	echo "captured_utc=$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
	echo "machine=$(uname -a)"
	if command -v sw_vers >/dev/null 2>&1; then
		echo "macos=$(sw_vers -productVersion)"
		echo "macos_build=$(sw_vers -buildVersion)"
	fi
	if command -v sysctl >/dev/null 2>&1; then
		echo "hardware_model=$(sysctl -n hw.model 2>/dev/null || true)"
		echo "logical_cpus=$(sysctl -n hw.logicalcpu 2>/dev/null || true)"
	fi
	echo "go=$(go version)"
	echo "go_flags=$(go env GOFLAGS)"
	echo "goos=$(go env GOOS)"
	echo "goarch=$(go env GOARCH)"
	echo "goamd64=$(go env GOAMD64)"
	echo "goarm64=$(go env GOARM64)"
	echo "cgo_enabled=0"
	echo "gowork=off"
	echo "benchmark_build=go test default optimized compiler with -trimpath and -ldflags=-buildid=; race detector disabled"
	echo "mygo_version=$MYGO_VERSION"
	echo "semantics_profile=v3"
	echo "ledger=$LEDGER"
	echo "ledger_sha256=$LEDGER_SHA"
	echo "window=1180x780 DIP, scale=1 (welcome/journal); chart=900x600 DIP, scale=1"
	echo "journal_page_size=100; chart_raw_points=10000"
	echo "frame_samples=${FRAME_SAMPLES:-100}"
	echo "count=${COUNT:-10}"
	echo "core_read_samples=${READ_SAMPLES:-50}"
	echo "core_open_samples=${OPEN_SAMPLES:-10}"
	echo "limitations=headless layout and CPU raster only; no native input-to-present latency, Metal/GPU, idle CPU, process RSS, cancellation-to-feedback, or SwiftUI baseline"
} > "$OUT/metadata.txt"

cd "$APP_DIR"
GOUI_BENCH_LEDGER="$LEDGER" GOWORK=off CGO_ENABLED=0 go test ./internal/app \
	-run '^$' \
	-bench '^BenchmarkGouiHeadless' \
	-benchmem \
	-trimpath \
	-ldflags=-buildid= \
	-benchtime="${FRAME_SAMPLES:-100}x" \
	-count="${COUNT:-10}" > "$OUT/results.txt" 2>&1
GOUI_BENCH_LEDGER="$LEDGER" GOWORK=off CGO_ENABLED=0 go test ./internal/app \
	-run '^$' \
	-bench '^BenchmarkGouiCoreReadJournalPage10052500$' \
	-benchmem \
	-trimpath \
	-ldflags=-buildid= \
	-benchtime="${READ_SAMPLES:-50}x" \
	-count=1 >> "$OUT/results.txt" 2>&1
GOUI_BENCH_LEDGER="$LEDGER" GOWORK=off CGO_ENABLED=0 go test ./internal/app \
	-run '^$' \
	-bench '^BenchmarkGouiCoreOpenLedger52500$' \
	-benchmem \
	-trimpath \
	-ldflags=-buildid= \
	-benchtime="${OPEN_SAMPLES:-10}x" \
	-count=1 >> "$OUT/results.txt" 2>&1

echo "Benchmark results: $OUT"
cat "$OUT/metadata.txt"
cat "$OUT/results.txt"
