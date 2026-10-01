#!/bin/sh
set -eu
REPO_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$REPO_ROOT"
SOURCE_APP="$REPO_ROOT/build/OrangeCount.app"
QA_APP="$REPO_ROOT/build/OrangeCount-QA.app"
case "${ORANGECOUNT_QA_APPEARANCE:-}" in
    ''|light|dark) ;;
    *) echo 'ORANGECOUNT_QA_APPEARANCE must be light or dark' >&2; exit 1 ;;
esac
case "${ORANGECOUNT_QA_FONT_SCALE:-}" in
    ''|1.0|1.25|1.5) ;;
    *) echo 'ORANGECOUNT_QA_FONT_SCALE must be 1.0, 1.25 or 1.5' >&2; exit 1 ;;
esac
[ -d "$SOURCE_APP" ] || { echo 'Run make app first' >&2; exit 1; }
rm -rf "$QA_APP"
ditto "$SOURCE_APP" "$QA_APP"
/usr/libexec/PlistBuddy -c 'Set :CFBundleIdentifier org.orangecount.desktop.qa' "$QA_APP/Contents/Info.plist"
/usr/libexec/PlistBuddy -c 'Set :CFBundleName OrangeCount QA' "$QA_APP/Contents/Info.plist"
/usr/libexec/PlistBuddy -c 'Set :CFBundleDisplayName OrangeCount QA' "$QA_APP/Contents/Info.plist"
# An isolated test app must not compete with installed viewers for file opens.
/usr/libexec/PlistBuddy -c 'Delete :CFBundleDocumentTypes' "$QA_APP/Contents/Info.plist"
/usr/libexec/PlistBuddy -c 'Add :OrangeCountTestMode bool true' "$QA_APP/Contents/Info.plist"
if [ -n "${ORANGECOUNT_QA_APPEARANCE:-}" ]; then
    /usr/libexec/PlistBuddy -c "Add :OrangeCountTestAppearance string $ORANGECOUNT_QA_APPEARANCE" "$QA_APP/Contents/Info.plist"
fi
if [ -n "${ORANGECOUNT_QA_FONT_SCALE:-}" ]; then
    /usr/libexec/PlistBuddy -c "Add :OrangeCountTestFontScale real $ORANGECOUNT_QA_FONT_SCALE" "$QA_APP/Contents/Info.plist"
fi
printf '\ntest_mode=isolated\nbundle_identifier=org.orangecount.desktop.qa\n' >> "$QA_APP/Contents/Resources/build-info.txt"
printf 'test_appearance=%s\n' "${ORANGECOUNT_QA_APPEARANCE:-system}" >> "$QA_APP/Contents/Resources/build-info.txt"
printf 'test_font_scale=%s\n' "${ORANGECOUNT_QA_FONT_SCALE:-1.0}" >> "$QA_APP/Contents/Resources/build-info.txt"
codesign --force --sign - --timestamp=none "$QA_APP"
codesign --verify --deep --strict "$QA_APP"
printf 'Isolated GUI test app: %s\n' "$QA_APP"
