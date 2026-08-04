// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import assert from "node:assert/strict";
import test from "node:test";

import { CORE_ROUTE_IDS, checkRouteManifest, parseTables } from "../../scripts/check-route-manifest.mjs";

const COMPLETE = `# Registry

| ID | Surface | Required outcomes | Wave | Current status |
| --- | --- | --- | --- | --- |
| R-ROOT | / | bootstrap | 1 | legacy |
| R-IS | /income_statement | tree | 1 | prototype |

## Globals

| ID | Surface | Wave | Current status |
| --- | --- | --- | --- |
| G-SHELL | shell | 1 | prototype |
| G-KEYBOARD | keys | 1 | planned |

## Modals

| ID | Surface | Wave | Current status |
| --- | --- | --- | --- |
| M-ADD | add | 6 | planned |

## Downloads

| ID | Surface | Wave | Current status |
| --- | --- | --- | --- |
| D-STATEMENT | statement | 3 | planned |

## Exclusions

| ID | Surface | Reason |
| --- | --- | --- |
| X-EXT | extensions | outside standard surface |
`;

test("route manifest parser skips non-table text", () => {
  const tables = parseTables("# only a heading\n\nnot a table\n");
  assert.equal(tables.length, 0);
});

test("complete manifest passes with the full core ID set", () => {
  const full = COMPLETE + coreRows();
  const errors = checkRouteManifest(full);
  assert.deepEqual(errors, []);
});

test("duplicate ID is reported", () => {
  const full = COMPLETE + coreRows() + `
| ID | Surface | Wave | Current status |
| --- | --- | --- | --- |
| R-IS | duplicate | 2 | gated |
`;
  const errors = checkRouteManifest(full);
  assert.ok(errors.some((e) => e.includes("duplicate ID: 'R-IS'")), errors.join("; "));
});

test("status row missing Wave or Current status is reported", () => {
  const full = COMPLETE + `
| ID | Surface | Wave | Current status |
| --- | --- | --- | --- |
| R-TB | toolbar | 2 |  |
`;
  const errors = checkRouteManifest(full);
  assert.ok(errors.some((e) => e.includes("missing Wave or Current status")), errors.join("; "));
});

test("exclusion row missing Reason is reported", () => {
  const full = COMPLETE + `
| ID | Surface | Reason |
| --- | --- | --- |
| X-PLUGIN | plugins |  |
`;
  const errors = checkRouteManifest(full);
  assert.ok(errors.some((e) => e.includes("missing Reason")), errors.join("; "));
});

test("missing category is reported", () => {
  // Drop the Downloads table so D-* is absent.
  const withoutD = COMPLETE.replace(/## Downloads[\s\S]*?D-STATEMENT.*\n/, "").trimStart();
  const errors = checkRouteManifest(withoutD);
  assert.ok(errors.some((e) => e.includes("missing category 'D'")), errors.join("; "));
});

test("missing core route ID is reported", () => {
  // coreRows() adds R-TB; remove it so coverage fails.
  const errors = checkRouteManifest(COMPLETE + coreRows().replace("| R-TB |", "| R-XCUSTOM |"));
  assert.ok(errors.some((e) => e.includes("missing core route ID 'R-TB'")), errors.join("; "));
});

function coreRows() {
  // The base COMPLETE fixture already declares R-ROOT and R-IS; add only the
  // remaining core IDs so the full set is covered without duplicates.
  const present = new Set(["R-ROOT", "R-IS"]);
  const rows = CORE_ROUTE_IDS.filter((id) => !present.has(id)).map((id) => `| ${id} | surface | outcome | 1 | legacy |`).join("\n");
  return `\n## Core\n| ID | Surface | Required outcomes | Wave | Current status |\n| --- | --- | --- | --- | --- |\n${rows}\n`;
}
