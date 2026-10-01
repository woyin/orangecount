#!/bin/sh
# Development-only Beancount v3 differential harness. Never use at runtime.
set -eu

ledger_path=${ORANGECOUNT_PRIVATE_LEDGER:-}
if [ -z "$ledger_path" ]; then
  echo "ORANGECOUNT_PRIVATE_LEDGER must point to an absolute ledger path" >&2
  exit 2
fi
case "$ledger_path" in
  /*) ;;
  *) echo "refusing a relative ledger path" >&2; exit 2 ;;
esac

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/../.." && pwd)
ledger_real=$(realpath "$ledger_path")
case "$ledger_real" in
  "$repo_root"|"$repo_root"/*)
    echo "refusing to inspect a ledger inside the OrangeCount repository" >&2
    exit 2
    ;;
esac

if ! command -v uv >/dev/null 2>&1; then
  echo "uv is required only for the optional development reference harness" >&2
  exit 2
fi

# The Python verifier owns and removes its temporary graph/state capture.
# It compares complete v3 inventories/entries and explicit monthly BQL rows.
exec uv run --project "$script_dir" python "$script_dir/verify_full_ledger.py"
