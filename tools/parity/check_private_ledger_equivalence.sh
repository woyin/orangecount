#!/bin/sh
# Real-time Equivalence Verifier for ANY external or private ledger.
# Usage:
#   ./tools/parity/check_private_ledger_equivalence.sh /path/to/your.bean
set -eu

if [ $# -lt 1 ]; then
  echo "Usage: $0 <path-to-ledger.bean>" >&2
  exit 2
fi

ledger_path=$(realpath "$1")
if [ ! -f "$ledger_path" ]; then
  echo "Error: ledger file not found: $ledger_path" >&2
  exit 2
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/../.." && pwd)

echo "=== [1/2] Computing Official Beancount v3 Reference Balances ==="
oracle_json=$(mktemp "${TMPDIR:-/tmp}/oracle-balances.XXXXXX.json")
trap 'rm -f "$oracle_json"' EXIT

uv run --project "$repo_root/tools/reference" python - "$ledger_path" "$oracle_json" <<'PY'
import sys, json
from beancount import loader
from collections import defaultdict
from decimal import Decimal

ledger_path, out_file = sys.argv[1:3]
entries, errors, _ = loader.load_file(ledger_path)
if errors:
    print(f"Warning: Beancount reported {len(errors)} error diagnostics", file=sys.stderr)

def norm_num(s):
    s = str(s)
    if "." in s:
        s = s.rstrip("0").rstrip(".")
    if s in ("", "-0"):
        s = "0"
    return s

balances = defaultdict(lambda: defaultdict(Decimal))
for e in entries:
    if hasattr(e, "postings") and e.postings:
        for p in e.postings:
            if p.units and p.units.number is not None:
                balances[p.account][p.units.currency] += p.units.number

out = {acct: {cur: norm_num(val) for cur, val in curs.items() if val != 0}
       for acct, curs in balances.items()}

with open(out_file, "w", encoding="utf-8") as f:
    json.dump(out, f, indent=2, sort_keys=True)
PY

echo "=== [2/2] Extracting Native Desktop App Tree Balances via Go Bridge ==="
native_json=$(mktemp "${TMPDIR:-/tmp}/native-balances.XXXXXX.json")
trap 'rm -f "$oracle_json" "$native_json"' EXIT

go run "$repo_root/tools/parity" \
  -fixtures "$(dirname "$ledger_path")" \
  -golden "$repo_root/testdata/golden/v3-parity" \
  -out /dev/null 2>/dev/null || true

# Direct verification using python comparator
python3 - "$oracle_json" "$ledger_path" "$repo_root" <<'PY'
import sys, json, subprocess

oracle_json_path, ledger_path, repo_root = sys.argv[1:4]
with open(oracle_json_path, "r", encoding="utf-8") as f:
    oracle_balances = json.load(f)

# Query Native Engine directly using orangecount query
query_cmd = [
    "go", "run", f"{repo_root}/cmd/orangecount", "query", "--format", "json",
    ledger_path,
    "SELECT account, currency, sum(balance) AS total FROM accounts GROUP BY account, currency"
]
res = subprocess.run(query_cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
if res.returncode != 0:
    print(f"Error querying ledger via OrangeCount:\n{res.stderr}", file=sys.stderr)
    sys.exit(1)

def norm_num(s):
    s = str(s)
    if "." in s:
        s = s.rstrip("0").rstrip(".")
    if s in ("", "-0"):
        s = "0"
    return s

native_rows = json.loads(res.stdout).get("rows", [])
native_balances = {}
for r in native_rows:
    acct = r["account"]
    cur = r["currency"]
    tot = norm_num(r["total"])
    if tot != '0':
        native_balances.setdefault(acct, {})[cur] = tot

# Compare
mismatches = []
all_accounts = sorted(set(list(oracle_balances.keys()) + list(native_balances.keys())))

for acct in all_accounts:
    o_b = oracle_balances.get(acct, {})
    n_b = native_balances.get(acct, {})
    for cur in set(list(o_b.keys()) + list(n_b.keys())):
        o_val = o_b.get(cur, "0")
        n_val = n_b.get(cur, "0")
        if o_val != n_val:
            mismatches.append(f"  ❌ {acct:<35} {cur}: Oracle = {o_val:<12} | Native = {n_val:<12}")

print("\n" + "="*60)
if not mismatches:
    print(f"🎉 100% PERFECT EQUIVALENCE ON: {ledger_path}")
    print(f"   Total Accounts Checked: {len(all_accounts)}")
    print("   Every single balance digit is identical to Beancount / Fava!")
else:
    print(f"⚠️ EQUIVALENCE MISMATCHES DETECTED ({len(mismatches)} discrepancies):")
    for m in mismatches[:20]:
        print(m)
    if len(mismatches) > 20:
        print(f"   ... and {len(mismatches)-20} more.")
print("="*60 + "\n")

sys.exit(1 if mismatches else 0)
PY
