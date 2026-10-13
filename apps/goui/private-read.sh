#!/bin/sh
# Copyright 2026 OrangeCount contributors
# Licensed under the Apache License, Version 2.0.
set -eu

case "${ORANGECOUNT_PRIVATE_LEDGER:-}" in
	/*) ;;
	*) printf 'Set ORANGECOUNT_PRIVATE_LEDGER to an external absolute ledger path.\n' >&2; exit 2 ;;
esac
if [ ! -f "$ORANGECOUNT_PRIVATE_LEDGER" ]; then
	printf 'The selected ledger entry is not a readable regular file.\n' >&2
	exit 2
fi

APP_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$APP_DIR"
GOWORK=off CGO_ENABLED=0 go test ./internal/app -run '^TestPrivateLedgerUIReadOnlyProjection$' -count=1 -v
