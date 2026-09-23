#!/bin/sh
# Development-only Beancount v3 parity corpus differential runner (ADR-0008).
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/../.." && pwd)
golden_dir="$repo_root/testdata/golden/v3-parity"
diff_file="$golden_dir/differences.json"
triage_file="$golden_dir/differences-triaged.json"

if ! command -v uv >/dev/null 2>&1; then
  echo "uv is required for the Beancount v3 reference environment" >&2
  exit 2
fi

echo "=== [1/3] Generating oracle golden snapshots ==="
uv run --project "$script_dir" python "$script_dir/generate_golden.py"

echo "=== [2/3] Running OrangeCount vs Oracle differential engine ==="
go run "$repo_root/tools/parity" \
  -fixtures "$repo_root/testdata/fixtures/v3-parity" \
  -golden "$golden_dir" \
  -out "$diff_file"

echo "=== [3/3] Checking Jev Triage readiness ==="
if [ -f "$repo_root/.env" ] && grep -q '^TYPESAFE_API_KEY=' "$repo_root/.env" 2>/dev/null; then
  echo "Running TypeSafe Jev triage..."
  uv run --project "$script_dir" python "$script_dir/jev_triage.py" \
    --in "$diff_file" \
    --out "$triage_file"
else
  echo "TYPESAFE_API_KEY not found in .env; skipping automated Jev triage."
  echo "You can run triage manually with: make parity-triage"
fi

echo ""
echo "=== Parity Scorecard Summary ==="
python3 - <<EOF
import json

diff_data = json.load(open("$diff_file", "r", encoding="utf-8"))
fc = diff_data.get("fixture_count", 0)
cc = diff_data.get("clean_count", 0)
records = diff_data.get("records", [])

print(f"Total Fixtures: {fc}")
print(f"Clean Fixtures: {cc}/{fc} ({cc*100//fc if fc else 0}%)")

unregistered = [r for r in records if not r.get("boundary")]
approved = [r for r in records if r.get("boundary")]

print(f"Unregistered Differences (Must be 0 for Parity): {len(unregistered)}")
print(f"Approved Boundary Differences                  : {len(approved)}")

if unregistered:
    print("\nUnregistered differences by dimension:")
    by_dim = {}
    for r in unregistered:
        d = r.get("dimension", "unknown")
        by_dim[d] = by_dim.get(d, 0) + 1
    for d, count in sorted(by_dim.items()):
        print(f"  {d:<18}: {count}")
else:
    print("\n🎉 ALL FIXTURES PASS BEANCOUNT v3 PARITY CONTRACT!")

try:
    triage_data = json.load(open("$triage_file", "r", encoding="utf-8"))
    s = triage_data.get("triage_summary", {})
    print("\nJev Triage Recommendations:")
    print(f"  Open Issue  : {s.get('open_issue', 0)}")
    print(f"  Needs Review: {s.get('needs_review', 0)}")
    print(f"  Ignore      : {s.get('ignore', 0)}")
except Exception:
    pass
EOF
