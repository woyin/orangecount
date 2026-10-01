#!/bin/sh
# Copyright 2026 OrangeCount contributors
# Licensed under the Apache License, Version 2.0.
# Complete v3 state plus actual Swift/C reads; GUI acceptance is separate.
set -eu
if [ "$#" -gt 1 ]; then echo 'Expected at most one external absolute ledger path' >&2; exit 2; fi
if [ "$#" -eq 1 ]; then export ORANGECOUNT_PRIVATE_LEDGER="$1"; fi
REPO_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
exec sh "$REPO_ROOT/apps/macos/private-read.sh"
