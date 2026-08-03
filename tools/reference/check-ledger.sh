#!/bin/sh
# Development-only differential harness. Never use this script at runtime.
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

# The Python snippet intentionally prints no paths, account names, amounts,
# narration, metadata, or raw diagnostic text.
exec uv run --project "$script_dir" python - "$ledger_real" <<'PY'
import json
import sys
from beancount import loader

entries, errors, _options = loader.load_file(sys.argv[1])
summary = {
    "entry_count": len(entries),
    "error_count": len(errors),
    "error_types": sorted({type(error).__name__ for error in errors}),
}
print(json.dumps(summary, ensure_ascii=True, sort_keys=True))
raise SystemExit(1 if errors else 0)
PY
