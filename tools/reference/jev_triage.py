#!/usr/bin/env python3
"""Development-only differential triage using TypeSafe Jev (ADR-0008 / Plan P3).

Consumes redacted difference records emitted by `orangecount` / `tools/parity`,
evaluates each difference against the Jev decision endpoint, and produces
`differences-triaged.json` with calibrated probabilities and triage actions.

Privacy notice:
  This script submits ONLY redacted metadata (counts, error codes, directive
  kinds, and line offsets) to the external API. Private ledger text, account
  names, and amounts NEVER leave the local host (ADR-0009).

Usage:
  uv run --project tools/reference python tools/reference/jev_triage.py \
      --in /tmp/differences.json --out testdata/golden/v3-parity/differences-triaged.json
"""
from __future__ import annotations

import argparse
import json
import os
import sys
import urllib.error
import urllib.request
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent.parent
DEFAULT_IN = REPO_ROOT / "testdata" / "golden" / "v3-parity" / "differences.json"
DEFAULT_OUT = REPO_ROOT / "testdata" / "golden" / "v3-parity" / "differences-triaged.json"
ENDPOINT = os.getenv("TYPESAFE_ENDPOINT", "https://api.typesafe.ai/v1/systemone")
MODEL = os.getenv("JEV_MODEL", "jev-1.13.0")


def load_env_file(env_path: Path):
    if not env_path.is_file():
        return
    with env_path.open("r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, val = line.split("=", 1)
            key = key.strip()
            val = val.strip().strip("'\"")
            if key and key not in os.environ:
                os.environ[key] = val


def build_triage_payload(record: dict) -> dict:
    state = {
        "fixture": record.get("fixture"),
        "dimension": record.get("dimension"),
        "kind": record.get("kind"),
        "detail": record.get("detail"),
    }
    questions = {
        "classification": {
            "type": "choice",
            "options": [
                "real-semantic-divergence",
                "approved-boundary-miss",
                "display-only",
                "formatting-noise",
                "oracle-version-artifact",
            ],
            "criteria": {
                "real-semantic-divergence": "OrangeCount and Beancount assign different core accounting semantics or validity",
                "approved-boundary-miss": "A known approved architectural boundary (e.g. plugins or local-only APIs) that should be filtered",
                "display-only": "Number formatting, string whitespace, or non-semantic display difference",
                "formatting-noise": "Line number off-by-one or AST node ordering noise without semantic consequence",
                "oracle-version-artifact": "Difference caused by beancount upstream version quirks rather than OrangeCount defect",
            },
        },
        "blocks_parity": {
            "type": "noul",
            "instructions": "True if this difference blocks claiming core Beancount v3 accounting parity for production ledgers.",
        },
        "fix_priority": {
            "type": "score",
            "range": [0, 100],
            "criteria": [
                {"value": 0, "description": "cosmetic or line-attribution detail that does not change balance"},
                {"value": 50, "description": "affects non-core directive or secondary workflow"},
                {"value": 100, "description": "core accounting validity, balancing, or parsing broken"},
            ],
        },
    }
    return {
        "model": MODEL,
        "state": state,
        "questions": questions,
    }


def call_jev(payload: dict, api_key: str) -> dict:
    data = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(
        ENDPOINT,
        data=data,
        headers={
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
            "User-Agent": "OrangeCount-Parity-Harness/1.0",
        },
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=30) as resp:
        return json.loads(resp.read().decode("utf-8"))


def main() -> int:
    parser = argparse.ArgumentParser(description="Triage differential records with TypeSafe Jev.")
    parser.add_argument("--in", dest="input_path", type=Path, default=DEFAULT_IN, help="Input differences.json")
    parser.add_argument("--out", dest="output_path", type=Path, default=DEFAULT_OUT, help="Output differences-triaged.json")
    args = parser.parse_args()

    load_env_file(REPO_ROOT / ".env")
    api_key = os.getenv("TYPESAFE_API_KEY")
    if not api_key:
        print("ERROR: TYPESAFE_API_KEY environment variable not set (and not in .env)", file=sys.stderr)
        return 2

    if not args.input_path.is_file():
        print(f"ERROR: input file not found: {args.input_path}", file=sys.stderr)
        return 2

    with args.input_path.open("r", encoding="utf-8") as f:
        data = json.load(f)

    records = data.get("records", [])
    print(f"Jev triage: evaluating {len(records)} difference records with model={MODEL}...")

    triaged_records = []
    issues_count = 0
    review_count = 0
    ignore_count = 0

    for i, rec in enumerate(records):
        fixture = rec.get("fixture", "unknown")
        dim = rec.get("dimension", "")
        kind = rec.get("kind", "")
        print(f"  [{i+1}/{len(records)}] {fixture} ({dim}/{kind})...", end=" ", flush=True)

        # If already classified by an approved boundary in compatibility-ledger, skip API call
        if rec.get("boundary"):
            rec["triage"] = {
                "source": "boundary-ledger",
                "classification": "approved-boundary-miss",
                "action": "ignore",
            }
            triaged_records.append(rec)
            ignore_count += 1
            print("skipped (boundary)")
            continue

        payload = build_triage_payload(rec)
        try:
            resp = call_jev(payload, api_key)
            answers = resp.get("answers", {})
            choice = answers.get("classification", {})
            blocks = answers.get("blocks_parity", {}).get("noul", 0.0)
            score_data = answers.get("fix_priority", {})
            score = score_data.get("score", 0.0)

            # Recommendation thresholds
            if blocks >= 0.70:
                action = "open-issue"
                issues_count += 1
            elif blocks >= 0.35:
                action = "needs-review"
                review_count += 1
            else:
                action = "ignore"
                ignore_count += 1

            rec["triage"] = {
                "source": resp.get("model", MODEL),
                "classification": choice.get("choice"),
                "classification_confidence": choice.get("confidence"),
                "blocks_parity_prob": blocks,
                "fix_priority_score": score,
                "action": action,
            }
            print(f"-> {action} (blocks={blocks:.2f}, class={choice.get('choice')})")
        except Exception as exc:  # noqa: BLE001
            print(f"FAILED: {exc}")
            rec["triage"] = {"error": str(exc), "action": "needs-review"}
            review_count += 1

        triaged_records.append(rec)

    data["records"] = triaged_records
    data["triage_summary"] = {
        "model": MODEL,
        "total": len(records),
        "open_issue": issues_count,
        "needs_review": review_count,
        "ignore": ignore_count,
    }

    args.output_path.parent.mkdir(parents=True, exist_ok=True)
    with args.output_path.open("w", encoding="utf-8") as f:
        json.dump(data, f, indent=2, ensure_ascii=False)
        f.write("\n")

    print("\nTriage completed successfully:")
    print(f"  Open Issue:   {issues_count}")
    print(f"  Needs Review: {review_count}")
    print(f"  Ignore:       {ignore_count}")
    print(f"  Report written to: {args.output_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
