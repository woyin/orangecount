#!/bin/sh
set -eu

APP_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$APP_DIR/../.." && pwd)
OUT_DIR=${OUT_DIR:-"$REPO_ROOT/dist/goui"}
VERSION=${VERSION:-0.0.0-dev}
TARGET_ARCH=${GOARCH:-$(go env GOARCH)}
cd "$APP_DIR"
APP_NAME="OrangeCount Go UI Development"
APP_DEST="$OUT_DIR/$APP_NAME.app"
ZIP_NAME="OrangeCount-Go-UI-Development-$VERSION-$TARGET_ARCH.zip"
ZIP_DEST="$OUT_DIR/$ZIP_NAME"
SHA_DEST="$ZIP_DEST.sha256"

case "$VERSION" in
	''|*[!A-Za-z0-9._-]*)
		printf 'invalid VERSION: %s\n' "$VERSION" >&2
		exit 2
		;;
esac
case "$TARGET_ARCH" in
	arm64|amd64) ;;
	*) printf 'unsupported macOS architecture: %s\n' "$TARGET_ARCH" >&2; exit 2 ;;
esac
if [ "$(uname -s)" != Darwin ]; then
	printf 'package.sh requires macOS (ditto, plutil, and Mach-O tools)\n' >&2
	exit 2
fi
if [ -e "$APP_DEST" ] || [ -e "$ZIP_DEST" ] || [ -e "$SHA_DEST" ]; then
	printf 'package output already exists; choose another OUT_DIR or remove that generated output first\n' >&2
	exit 2
fi

TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/orangecount-goui-package.XXXXXX")
trap 'rm -rf "$TMP_ROOT"' EXIT
APP_STAGE="$TMP_ROOT/$APP_NAME.app"
CONTENTS="$APP_STAGE/Contents"
EXECUTABLE="$CONTENTS/MacOS/orangecount-goui"
LICENSES="$CONTENTS/Resources/licenses"
mkdir -p "$CONTENTS/MacOS" "$LICENSES"

OUT="$EXECUTABLE" GOARCH="$TARGET_ARCH" "$APP_DIR/build.sh"
FILE_KIND=$(file -b "$EXECUTABLE")
case "$FILE_KIND" in
	*Mach-O*) ;;
	*) printf 'expected a Mach-O executable, got: %s\n' "$FILE_KIND" >&2; exit 1 ;;
esac
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
LINKED_LIBRARIES=$(otool -L "$EXECUTABLE")
case "$LINKED_LIBRARIES" in
	*libswift*) printf 'unexpected Swift runtime dependency in Go UI bundle\n' >&2; exit 1 ;;
esac

cat > "$CONTENTS/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleDevelopmentRegion</key>
	<string>en</string>
	<key>CFBundleExecutable</key>
	<string>orangecount-goui</string>
	<key>CFBundleIdentifier</key>
	<string>org.orangecount.desktop.goui.dev</string>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
	<key>CFBundleName</key>
	<string>$APP_NAME</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>$VERSION</string>
	<key>CFBundleVersion</key>
	<string>$VERSION</string>
	<key>LSMinimumSystemVersion</key>
	<string>13.0</string>
	<key>NSHighResolutionCapable</key>
	<true/>
	<key>NSPrincipalClass</key>
	<string>NSApplication</string>
	<key>UTExportedTypeDeclarations</key>
	<array>
		<dict>
			<key>UTTypeIdentifier</key>
			<string>org.orangecount.query-analysis</string>
			<key>UTTypeDescription</key>
			<string>OrangeCount Query Analysis</string>
			<key>UTTypeConformsTo</key>
			<array><string>public.data</string></array>
			<key>UTTypeTagSpecification</key>
			<dict>
				<key>public.filename-extension</key>
				<array><string>ocanalysis</string></array>
			</dict>
		</dict>
	</array>
	<key>CFBundleDocumentTypes</key>
	<array>
		<dict>
			<key>CFBundleTypeName</key>
			<string>OrangeCount Query Analysis</string>
			<key>CFBundleTypeRole</key>
			<string>Viewer</string>
			<key>LSHandlerRank</key>
			<string>Default</string>
			<key>LSItemContentTypes</key>
			<array><string>org.orangecount.query-analysis</string></array>
			<key>CFBundleTypeExtensions</key>
			<array><string>ocanalysis</string></array>
		</dict>
	</array>
</dict>
</plist>
EOF
plutil -lint "$CONTENTS/Info.plist"

cp "$REPO_ROOT/LICENSE" "$LICENSES/LICENSE-OrangeCount.txt"
cp "$REPO_ROOT/NOTICE" "$LICENSES/NOTICE-OrangeCount.txt"
cp "$APP_DIR/resources/licenses/manifest.txt" "$LICENSES/manifest.txt"
cp "$APP_DIR/third_party/mygo/UPSTREAM.md" "$LICENSES/UPSTREAM-mygo.md"
GOROOT=$(go env GOROOT)
TOOLCHAIN_LICENSE="$GOROOT/LICENSE"
TOOLCHAIN_PATENTS="$GOROOT/PATENTS"
if [ ! -f "$TOOLCHAIN_LICENSE" ]; then
	TOOLCHAIN_ROOT=${GOROOT%/libexec}
	TOOLCHAIN_LICENSE="$TOOLCHAIN_ROOT/LICENSE"
	TOOLCHAIN_PATENTS="$TOOLCHAIN_ROOT/PATENTS"
fi
if [ ! -f "$TOOLCHAIN_LICENSE" ]; then
	printf 'Go toolchain license not found below GOROOT or its Homebrew prefix\n' >&2
	exit 1
fi
cp "$TOOLCHAIN_LICENSE" "$LICENSES/LICENSE-Go-toolchain.txt"
if [ -f "$TOOLCHAIN_PATENTS" ]; then
	cp "$TOOLCHAIN_PATENTS" "$LICENSES/PATENTS-Go-toolchain.txt"
fi

GOWORK=off go mod download all
MODULES=$(GOWORK=off go list -m -f '{{if not .Main}}{{.Path}}|{{.Version}}|{{.Dir}}{{end}}' all)
GO_VERSION=$(go version)
GENERATED_MANIFEST="$LICENSES/module-licenses.txt"
{
	cat "$APP_DIR/resources/licenses/manifest.txt"
	printf '\nGo toolchain: %s — LICENSE-Go-toolchain.txt\n' "$GO_VERSION"
	if [ -f "$LICENSES/PATENTS-Go-toolchain.txt" ]; then
		printf 'Go toolchain patent notice: PATENTS-Go-toolchain.txt\n'
	fi
	printf '\nIncluded module licenses:\n'
} > "$GENERATED_MANIFEST"
printf '%s\n' "$MODULES" | while IFS='|' read -r module version directory; do
	[ -n "$module" ] || continue
	license_path=
	for candidate in LICENSE LICENSE.txt COPYING COPYING.txt; do
		if [ -f "$directory/$candidate" ]; then
			license_path="$directory/$candidate"
			break
		fi
	done
	if [ -z "$license_path" ]; then
		printf 'no license file found for %s %s at %s\n' "$module" "$version" "$directory" >&2
		exit 1
	fi
	license_name=$(printf '%s-%s' "$module" "$version" | sed 's/[^A-Za-z0-9._-]/-/g')
	license_output="LICENSE-$license_name.txt"
	cp "$license_path" "$LICENSES/$license_output"
	printf '%s %s — %s\n' "$module" "$version" "$license_output" >> "$GENERATED_MANIFEST"
done

if git -C "$REPO_ROOT" rev-parse --verify HEAD >/dev/null 2>&1; then
	SOURCE_REVISION=$(git -C "$REPO_ROOT" rev-parse HEAD)
	if [ -n "$(git -C "$REPO_ROOT" status --porcelain)" ]; then
		SOURCE_TREE=dirty
	else
		SOURCE_TREE=clean
	fi
else
	SOURCE_REVISION=unknown
	SOURCE_TREE=unknown
fi
MYGO_VERSION=$(GOWORK=off go list -m -f '{{.Version}}' github.com/egoist/mygo)
cat > "$CONTENTS/Resources/build-info.json" <<EOF
{
  "application": "OrangeCount Go UI Development",
  "version": "$VERSION",
  "bundle_identifier": "org.orangecount.desktop.goui.dev",
  "source_revision": "$SOURCE_REVISION",
  "source_tree": "$SOURCE_TREE",
  "go_version": "$GO_VERSION",
  "goos": "darwin",
  "goarch": "$TARGET_ARCH",
  "cgo_enabled": false,
  "trimpath": true,
  "build_id": "empty",
  "minimum_macos": "13.0",
  "mygo_version": "$MYGO_VERSION",
  "signed": false,
  "notarized": false
}
EOF

mkdir -p "$OUT_DIR"
ditto --rsrc --extattr "$APP_STAGE" "$APP_DEST"
ditto -c -k --sequesterRsrc --keepParent "$APP_STAGE" "$TMP_ROOT/$ZIP_NAME"
cp "$TMP_ROOT/$ZIP_NAME" "$ZIP_DEST"
(cd "$OUT_DIR" && shasum -a 256 "$ZIP_NAME" > "$ZIP_NAME.sha256")

printf 'created %s\n' "$APP_DEST"
printf 'created %s\n' "$ZIP_DEST"
printf 'created %s\n' "$SHA_DEST"
