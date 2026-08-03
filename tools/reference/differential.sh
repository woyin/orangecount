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

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/orangecount-differential.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM
orange_json="$tmp_dir/orange.json"
orange_stderr="$tmp_dir/orange.stderr"
orange_status=0

if [ -x "$repo_root/bin/orangecount" ]; then
  "$repo_root/bin/orangecount" check --json "$ledger_real" >"$orange_json" 2>"$orange_stderr" || orange_status=$?
else
  (cd "$repo_root" && go run ./cmd/orangecount check --json "$ledger_real") >"$orange_json" 2>"$orange_stderr" || orange_status=$?
fi

# The Python process reads only normalized counts/codes from OrangeCount's
# JSON and Beancount's error classes. It never prints paths, account names,
# amounts, transaction text, metadata, or raw diagnostic strings.
exec uv run --project "$script_dir" python - "$ledger_real" "$orange_json" "$orange_status" <<'PY'
import json
import sys
from beancount import loader

ledger_path, orange_path, orange_status = sys.argv[1:]
with open(orange_path, encoding="utf-8") as handle:
    diagnostics = json.load(handle)
orange_errors = [item for item in diagnostics if item.get("severity") == "error"]
entries, errors, _options = loader.load_file(ledger_path)
summary = {
    "beancount": {
        "entry_count": len(entries),
        "error_count": len(errors),
        "error_types": sorted({type(error).__name__ for error in errors}),
    },
    "orangecount": {
        "diagnostic_count": len(diagnostics),
        "error_count": len(orange_errors),
        "error_codes": sorted({item.get("code", "") for item in orange_errors}),
        "exit_status": int(orange_status),
    },
}
print(json.dumps(summary, ensure_ascii=True, sort_keys=True))
raise SystemExit(1 if errors or orange_errors or int(orange_status) else 0)
PY
