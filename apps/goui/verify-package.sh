#!/bin/sh
set -eu

APP_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$APP_DIR/../.." && pwd)
OUT_DIR=${1:-"$REPO_ROOT/dist/goui"}
VERSION=${2:-${VERSION:-0.0.0-dev}}
TARGET_ARCH=${3:-${GOARCH:-$(go env GOARCH)}}
APP_NAME="OrangeCount Go UI Development"
APP_PATH="$OUT_DIR/$APP_NAME.app"
ZIP_NAME="OrangeCount-Go-UI-Development-$VERSION-$TARGET_ARCH.zip"
ZIP_PATH="$OUT_DIR/$ZIP_NAME"
SHA_PATH="$ZIP_PATH.sha256"

case "$TARGET_ARCH" in
	arm64|amd64) ;;
	*) printf 'unsupported macOS architecture: %s\n' "$TARGET_ARCH" >&2; exit 2 ;;
esac
if [ "$(uname -s)" != Darwin ]; then
	printf 'verify-package.sh requires macOS (ditto, plutil, and Mach-O tools)\n' >&2
	exit 2
fi

CONTENTS="$APP_PATH/Contents"
EXECUTABLE="$CONTENTS/MacOS/orangecount-goui"
LICENSES="$CONTENTS/Resources/licenses"
TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/orangecount-goui-verify.XXXXXX")
trap 'rm -rf "$TMP_ROOT"' EXIT
for required in "$APP_PATH" "$EXECUTABLE" "$ZIP_PATH" "$SHA_PATH"; do
	if [ ! -e "$required" ]; then
		printf 'required package artifact is missing: %s\n' "$required" >&2
		exit 1
	fi
done

plutil -lint "$CONTENTS/Info.plist"
if [ "$(plutil -extract CFBundleIdentifier raw -o - "$CONTENTS/Info.plist")" != "org.orangecount.desktop.goui.dev" ]; then
	printf 'unexpected development bundle identifier\n' >&2
	exit 1
fi
if [ "$(plutil -extract CFBundleShortVersionString raw -o - "$CONTENTS/Info.plist")" != "$VERSION" ]; then
	printf 'bundle version does not match requested version %s\n' "$VERSION" >&2
	exit 1
fi
if [ "$(plutil -extract LSMinimumSystemVersion raw -o - "$CONTENTS/Info.plist")" != "13.0" ]; then
	printf 'bundle minimum macOS version is not 13.0\n' >&2
	exit 1
fi

FILE_KIND=$(file -b "$EXECUTABLE")
case "$FILE_KIND" in
	*Mach-O*) ;;
	*) printf 'expected a Mach-O executable, got: %s\n' "$FILE_KIND" >&2; exit 1 ;;
esac
MACH_ARCH="$TARGET_ARCH"
if [ "$TARGET_ARCH" = amd64 ]; then
	MACH_ARCH=x86_64
fi
lipo -verify_arch "$MACH_ARCH" "$EXECUTABLE"
MACHO_MINIMUM=$(otool -l "$EXECUTABLE" | awk '/cmd LC_BUILD_VERSION/ { build=1; next } build && /minos/ { print $2; exit }')
if [ "$MACHO_MINIMUM" != "13.0" ]; then
	printf 'Mach-O minimum macOS version is %s, expected 13.0\n' "${MACHO_MINIMUM:-unknown}" >&2
	exit 1
fi
BUILD_SETTINGS=$(go version -m "$EXECUTABLE")
printf '%s\n' "$BUILD_SETTINGS" | grep -q 'CGO_ENABLED=0' || {
	printf 'packaged binary was not built with CGO_ENABLED=0\n' >&2
	exit 1
}
printf '%s\n' "$BUILD_SETTINGS" | grep -q 'GOOS=darwin' || {
	printf 'packaged binary was not built for macOS\n' >&2
	exit 1
}
printf '%s\n' "$BUILD_SETTINGS" | grep -q "GOARCH=$TARGET_ARCH" || {
	printf 'packaged binary does not match requested architecture %s\n' "$TARGET_ARCH" >&2
	exit 1
}
LINKED_LIBRARIES=$(otool -L "$EXECUTABLE")
case "$LINKED_LIBRARIES" in
	*libswift*) printf 'unexpected Swift runtime dependency in Go UI bundle\n' >&2; exit 1 ;;
esac

for required in \
	"$LICENSES/LICENSE-OrangeCount.txt" \
	"$LICENSES/NOTICE-OrangeCount.txt" \
	"$LICENSES/LICENSE-Go-toolchain.txt" \
	"$LICENSES/UPSTREAM-mygo.md" \
	"$LICENSES/manifest.txt" \
	"$LICENSES/module-licenses.txt"; do
	if [ ! -s "$required" ]; then
		printf 'required license inventory file is missing or empty: %s\n' "$required" >&2
		exit 1
	fi
done
EXPECTED_MODULES="$TMP_ROOT/modules.expected"
ACTUAL_MODULES="$TMP_ROOT/modules.actual"
awk -F ' — ' 'NF == 2 && $2 ~ /^LICENSE-/ && $1 !~ /^Go toolchain:/ { print $1 }' "$LICENSES/module-licenses.txt" | sort > "$EXPECTED_MODULES"
(cd "$APP_DIR" && GOWORK=off go list -m -f '{{if not .Main}}{{.Path}} {{.Version}}{{end}}' all) | sed '/^$/d' | sort > "$ACTUAL_MODULES"
if ! diff -u "$EXPECTED_MODULES" "$ACTUAL_MODULES"; then
	printf 'bundled license manifest does not match the resolved Go module graph\n' >&2
	exit 1
fi
awk -F ' — ' 'NF == 2 && $2 ~ /^LICENSE-/ { print $2 }' "$LICENSES/module-licenses.txt" | while IFS= read -r license_file; do
	if [ ! -s "$LICENSES/$license_file" ]; then
		printf 'license manifest entry is missing: %s\n' "$license_file" >&2
		exit 1
	fi
done

BUILD_INFO="$CONTENTS/Resources/build-info.json"
if [ ! -s "$BUILD_INFO" ]; then
	printf 'build information is missing\n' >&2
	exit 1
fi
grep -Fq '"version": "'"$VERSION"'"' "$BUILD_INFO"
grep -Fq '"goarch": "'"$TARGET_ARCH"'"' "$BUILD_INFO"
grep -Fq '"minimum_macos": "13.0"' "$BUILD_INFO"
grep -Fq '"cgo_enabled": false' "$BUILD_INFO"
grep -Fq '"signed": false' "$BUILD_INFO"
grep -Fq '"notarized": false' "$BUILD_INFO"

(
	cd "$OUT_DIR"
	shasum -a 256 -c "$(basename -- "$SHA_PATH")"
)
EXTRACT_ROOT="$TMP_ROOT/extracted"
mkdir -p "$EXTRACT_ROOT"
ditto -x -k "$ZIP_PATH" "$EXTRACT_ROOT"
EXTRACTED_APP="$EXTRACT_ROOT/$APP_NAME.app"
if [ ! -x "$EXTRACTED_APP/Contents/MacOS/orangecount-goui" ]; then
	printf 'ZIP does not contain the expected runnable application bundle\n' >&2
	exit 1
fi
cmp "$CONTENTS/Info.plist" "$EXTRACTED_APP/Contents/Info.plist"
cmp "$EXECUTABLE" "$EXTRACTED_APP/Contents/MacOS/orangecount-goui"
cmp "$LICENSES/module-licenses.txt" "$EXTRACTED_APP/Contents/Resources/licenses/module-licenses.txt"

printf 'verified %s (%s, macOS 13.0+, unsigned development bundle)\n' "$ZIP_PATH" "$TARGET_ARCH"
