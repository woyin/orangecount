# AGENTS.md — OrangeCount

## Project Skills

Daily ledger work is covered by three project skills in `.agents/skills/`.
Load the matching one instead of re-deriving the commands:

| Skill | Use for |
| --- | --- |
| `orangecount-ledger` | validate, explain a diagnostic, query, compare semantics (read-only) |
| `orangecount-authoring` | add or edit entries, quick entry, import, hand-edit a `.bean` |
| `orangecount-workbench` | serve, report pages, HTTP API, XLSX/ODS, export a v3 snapshot |

The user-facing manual is [docs/user-guide.md](docs/user-guide.md); keep it
accurate when a command or flag changes.

## Project Constraints

- UI must match the Fava reference implementation as closely as practical
  ("尽量 1:1 对齐 Fava"). When transplanting Fava UI behavior, aim for visual
  and behavioral parity; where parity is infeasible, register the deviation in
  `docs/fava-approved-deviations.md` and get product-owner approval rather than
  silently diverging.