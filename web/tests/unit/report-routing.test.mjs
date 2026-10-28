// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Route-to-report wiring guard.
//
// Fava renders a dedicated report component per route. When ReportOutlet only
// had the generic table fallback, /commodities silently degraded to
// GenericReport: the price line chart never mounted and the commodity metadata
// table disappeared. This guard pins the routing table at the source level, so
// a future refactor cannot drop a specialised report back to the generic one
// without a failing test.

import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import path from "node:path";
import test from "node:test";

const repoRoot = path.resolve(import.meta.dirname, "..", "..", "..");
const outletPath = path.join(repoRoot, "web", "src", "fava", "components", "ReportOutlet.svelte");

/**
 * Every specialised report route and the component that must render it.
 * `branch` is the exact table-branch condition text; journal and statistics are
 * discriminated by their own loaded state instead of the table branch.
 */
const SPECIALISED = [
  { route: "commodities", component: "CommoditiesReport", branch: 'table && route === "commodities"' },
  { route: "holdings", component: "HoldingsReport", branch: 'table && (route === "holdings" || route.startsWith("holdings_by_"))' },
  { route: "events", component: "EventsReport", branch: 'table && route === "events"' },
  { route: "documents", component: "DocumentsReport", branch: 'table && route === "documents"' },
  { route: "errors", component: "ErrorsReport", branch: 'table && route === "errors"' },
  { route: "journal", component: "JournalReport" },
  { route: "statistics", component: "StatisticsReport" },
];

test("each specialised route imports its own report component", async () => {
  const source = await readFile(outletPath, "utf8");
  for (const { component } of SPECIALISED) {
    const expected = 'import ' + component + ' from "../reports/' + component + '.svelte";';
    assert.ok(source.includes(expected), "ReportOutlet must import " + component);
  }
});

test("every specialised route mounts its component before the generic fallback", async () => {
  const source = await readFile(outletPath, "utf8");
  // The generic fallback is the bare `{:else if table}` branch. A specialised
  // route handled after it would never render.
  const genericIndex = source.indexOf("{:else if table}");
  assert.ok(genericIndex > 0, "ReportOutlet must keep a generic table fallback");
  for (const { route, component, branch } of SPECIALISED) {
    if (!branch) continue; // journal and statistics are discriminated by their loaded state
    const marker = "{:else if " + branch + "}";
    const branchIndex = source.indexOf(marker);
    assert.ok(branchIndex > 0, "route " + route + " must have its own table branch");
    assert.ok(branchIndex < genericIndex, "route " + route + " must be handled before the generic fallback");
    const body = source.slice(branchIndex, branchIndex + 400);
    assert.ok(body.includes("<" + component), "route " + route + " must render <" + component + ">");
  }
  // journal and statistics are discriminated by their own loaded state.
  assert.ok(source.includes("<JournalReport report={journal}"), "journal must render JournalReport");
  assert.ok(source.includes("<StatisticsReport "), "statistics must render StatisticsReport");
});

test("commodities passes the price history into CommoditiesReport", async () => {
  const source = await readFile(outletPath, "utf8");
  // Fava's commodities page shows one price chart per base/quote pair. The
  // adapter serves prices as a separate report, so the route must request and
  // forward them; dropping either half leaves the chart area empty.
  const marker = '{:else if table && route === "commodities"}';
  const branchIndex = source.indexOf(marker);
  assert.ok(branchIndex > 0, "commodities branch must exist");
  const branch = source.slice(branchIndex, branchIndex + 400);
  assert.ok(branch.includes("prices={commodityPrices}"), "commodities must pass the price history");
  assert.ok(source.includes('adapter.load("prices"'), "ReportOutlet must request the prices report for commodities");
});

test("the embedded bundle contains the commodities report surfaces", async () => {
  const bundle = await readFile(
    path.join(repoRoot, "internal", "web", "assets", "transplanted", "app.js"),
    "utf8",
  );
  for (const marker of ["prices-table", "chart-switcher", "commodities-table"]) {
    assert.ok(bundle.includes(marker), "embedded bundle must contain " + marker);
  }
});
