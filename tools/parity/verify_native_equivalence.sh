#!/bin/sh
# Native contract and build checks; GUI acceptance is recorded separately.
set -eu
REPO_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$REPO_ROOT"
go test ./internal/desktop ./internal/bridge
make mac-test app
echo "Native contracts, state tests and compilation passed. GUI acceptance is separate."
