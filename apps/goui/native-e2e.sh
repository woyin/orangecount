#!/bin/sh
set -eu

APP_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
if [ "$(uname -s)" != Darwin ]; then
	printf 'native-e2e.sh requires macOS and opens real MyGo windows\n' >&2
	exit 2
fi

cd "$APP_DIR"
exec env GOWORK=off CGO_ENABLED=1 ORANGECOUNT_GOUI_NATIVE_E2E=1 \
	go test -tags orangecount_e2e ./internal/app -count=1 "$@"
