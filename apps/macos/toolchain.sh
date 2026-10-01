#!/bin/sh
# Shared native build settings. Source after REPO_ROOT has been set.
set -eu
[ "$(uname -s)" = Darwin ] || { echo 'Native builds require macOS' >&2; exit 1; }
[ "$(uname -m)" = arm64 ] || { echo 'This development package currently targets Apple Silicon only' >&2; exit 1; }
OC_GO=${ORANGECOUNT_GO:-go}
OC_SWIFT=${ORANGECOUNT_SWIFTC:-$(xcrun --find swiftc)}
OC_MIN_OS=${ORANGECOUNT_MIN_MACOS:-13.0}
case "$OC_MIN_OS" in *[!0-9.]*|'') echo 'Invalid deployment target' >&2; exit 1;; esac
OC_TARGET="arm64-apple-macosx$OC_MIN_OS"
export MACOSX_DEPLOYMENT_TARGET="$OC_MIN_OS"
export CGO_ENABLED=1 GOOS=darwin GOARCH=arm64
export SDKROOT="$(xcrun --show-sdk-path)"
export CC="$(xcrun --find clang)"
export CGO_CFLAGS="${CGO_CFLAGS:-} -mmacosx-version-min=$OC_MIN_OS -isysroot \"$SDKROOT\""
export CGO_LDFLAGS="${CGO_LDFLAGS:-} -mmacosx-version-min=$OC_MIN_OS -isysroot \"$SDKROOT\""
