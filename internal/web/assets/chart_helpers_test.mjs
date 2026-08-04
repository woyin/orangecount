// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

/**
 * Deterministic regression test for the embedded chart helpers.
 *
 * The bundle is dependency-free and not a module, so this test evaluates only
 * the pure chart functions in a tiny harness. It asserts tick generation,
 * compact formatting, date sampling, tooltip content, and legend toggle state
 * without constructing a DOM or measuring SVG pixels.
 */
import assert from "node:assert/strict";
import fs from "node:fs";
import vm from "node:vm";

const sourcePath = new URL("./app.js", import.meta.url);
const source = fs.readFileSync(sourcePath, "utf8");

function extractFunction(name) {
  const marker = `function ${name}(`;
  const start = source.indexOf(marker);
  assert.notEqual(start, -1, `missing ${name} in app.js`);
  let parentheses = 0;
  let open = -1;
  for (let index = source.indexOf("(", start); index < source.length; index += 1) {
    if (source[index] === "(") parentheses += 1;
    if (source[index] === ")" && --parentheses === 0) {
      open = source.indexOf("{", index);
      break;
    }
  }
  assert.notEqual(open, -1, `missing body for ${name}`);
  let depth = 0;
  let quote = "";
  let escaped = false;
  for (let index = open; index < source.length; index += 1) {
    const character = source[index];
    if (quote) {
      if (escaped) escaped = false;
      else if (character === "\\") escaped = true;
      else if (character === quote) quote = "";
      continue;
    }
    if (character === "'" || character === '"' || character === "`") {
      quote = character;
      continue;
    }
    if (character === "{") depth += 1;
    if (character === "}" && --depth === 0) return source.slice(start, index + 1);
  }
  throw new Error(`unterminated ${name}`);
}

const context = {
  console,
  t(key) { return key; },
  escapeHTML(value) {
    return String(value == null ? "" : value).replace(/[&<>"']/g, (character) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[character]));
  },
};
vm.runInNewContext([
  extractFunction("ticks"),
  extractFunction("formatCompact"),
  extractFunction("dateTicks"),
  extractFunction("samplePoints"),
  extractFunction("buildTooltipHtml"),
  extractFunction("legendToggleStates"),
  extractFunction("presented"),
  extractFunction("display"),
  extractFunction("exactValue"),
  extractFunction("chartValue"),
  extractFunction("flattenHierarchy"),
  extractFunction("hierarchyTotal"),
  extractFunction("layoutTreemapRects"),
].join("\n"), context);

// Ticks span the data range and are monotonic.
const tickValues = context.ticks(0, 100, 4);
assert.ok(tickValues.length >= 2, `ticks(0,100) should have several ticks, got ${tickValues}`);
assert.ok(tickValues.every((value, index) => index === 0 || value > tickValues[index - 1]), "ticks should be strictly increasing");
assert.ok(tickValues[0] >= 0 && tickValues[tickValues.length - 1] <= 100, "ticks should stay within the data range");

// Compact formatting abbreviates large magnitudes.
assert.equal(context.formatCompact(1234), "1.2k", "1.2k for thousands");
assert.equal(context.formatCompact(2_500_000), "2.5M", "2.5M for millions");
assert.equal(context.formatCompact(123), "123", "plain value unchanged");

// Date ticks respect the requested count, always keep the last date, and may
// add one extra tick for that final date when it lands outside the stride.
const dates = Array.from({ length: 68 }, (_, i) => `20${String(20 + Math.floor(i / 12)).padStart(2, "0")}-${String((i % 12) + 1).padStart(2, "0")}-01`);
const dateTicks = context.dateTicks(dates, 6);
assert.ok(dateTicks.length <= 7, `dateTicks should cap near 6, got ${dateTicks.length}`);
assert.equal(dateTicks[dateTicks.length - 1], dates[dates.length - 1], "dateTicks should keep the last date");

// Sampling keeps first and last points and downsamples the dense middle.
const points = Array.from({ length: 100 }, (_, i) => ({ date: `d${i}`, value: i }));
const sampled = context.samplePoints(points, 60);
assert.equal(sampled.length, 60, "samplePoints should return exactly maxPoints");
assert.equal(sampled[0], points[0], "samplePoints keeps the first point");
assert.equal(sampled[sampled.length - 1], points[points.length - 1], "samplePoints keeps the last point");
assert.equal(context.samplePoints(points.slice(0, 10)).length, 10, "short series are not sampled");

// Tooltip includes the value and unit and escapes markup.
const tooltip = context.buildTooltipHtml("Assets", "2000-01", "110", "USD", "at-cost");
assert.ok(tooltip.includes("110 USD"), "tooltip should include value and unit");
assert.ok(tooltip.includes("at-cost"), "tooltip should include valuation");
assert.ok(tooltip.includes("Assets") && tooltip.includes("2000-01"), "tooltip should include label and date");

// Legend toggle state reflects the hidden set.
const states = context.legendToggleStates([{ label: "A" }, { label: "B" }, { label: "C" }], new Set([1]));
assert.deepEqual(states.map((s) => s.hidden), [false, true, false], "legend toggle should hide only the selected series");

// chartValue normalizes a serialized PresentedDecimal ({display, exact}) into
// a number, and falls back to a plain number/string.
assert.equal(context.chartValue({ display: "15", exact: "15", approximate: false }), 15, "chartValue should read a PresentedDecimal display");
assert.equal(context.chartValue(7), 7, "chartValue passes through plain numbers");
assert.equal(context.chartValue("2.5"), 2.5, "chartValue parses numeric strings");
assert.ok(Number.isNaN(context.chartValue(null)), "chartValue of null is NaN");

// flattenHierarchy must normalize PresentedDecimal node values so the layout
// math never sees a NaN (which previously made the whole hierarchy card vanish).
const hierarchy = {
  nodes: [
    { name: "Assets", value: { display: "1114.59", exact: "1114.59", approximate: false }, depth: 0, children: [
      { name: "Assets:Bank", value: { display: "1114.59", exact: "1114.59", approximate: false }, depth: 1, children: [
        { name: "Assets:Bank:ZA", value: { display: "15", exact: "15", approximate: false }, depth: 2 },
      ] },
    ] },
  ],
};
const flat = context.flattenHierarchy(hierarchy);
assert.equal(flat.length, 3, "flattenHierarchy should flatten all nodes");
assert.ok(flat.every((node) => typeof node.value === "number"), "flattenHierarchy should normalize values to numbers");
assert.equal(flat[0].value, 1114.59, "normalized aggregate value");
assert.equal(flat[2].value, 15, "normalized leaf value");

// hierarchyTotal is the sum of absolute node values and must be finite/positive.
const total = context.hierarchyTotal(flat);
assert.ok(Number.isFinite(total) && total > 0, `hierarchyTotal should be a positive finite number, got ${total}`);
assert.ok(context.hierarchyTotal([]) === 0, "empty hierarchy totals zero");

// layoutTreemapRects must place every item exactly once, keep every rectangle
// two-dimensional (non-zero width and height), and exactly tile the given box
// even when values are wildly skewed (one dominant leaf plus many tiny ones,
// the real-world shape that made the previous single-row layout degenerate
// into a sliver of near-invisible rectangles).
const skewedItems = [
  { node: { name: "dominant" }, value: 1000000 },
  { node: { name: "tiny-1" }, value: 1 },
  { node: { name: "tiny-2" }, value: 1 },
  { node: { name: "tiny-3" }, value: 1 },
  { node: { name: "tiny-4" }, value: 1 },
  { node: { name: "mid" }, value: 500 },
];
const rects = context.layoutTreemapRects(skewedItems, 0, 4, 100, 44);
assert.equal(rects.length, skewedItems.length, "every item should produce exactly one rectangle");
// rects was built inside the vm context, so it (and Array.prototype.map's
// result) carries that context's Array constructor; compare the names as a
// joined string rather than via assert.deepEqual, which also checks
// prototype identity and would spuriously fail on a cross-realm array even
// when the contents match.
const names = rects.map((r) => r.node.name).sort().join(",");
const expectedNames = skewedItems.map((i) => i.node.name).sort().join(",");
assert.equal(names, expectedNames, "every leaf must appear exactly once");
rects.forEach((rect) => {
  assert.ok(rect.w > 0.001, `rectangle for ${rect.node.name} has non-positive width ${rect.w}`);
  assert.ok(rect.h > 0.001, `rectangle for ${rect.node.name} has non-positive height ${rect.h}`);
});
const totalArea = rects.reduce((sum, rect) => sum + rect.w * rect.h, 0);
assert.ok(Math.abs(totalArea - 100 * 44) < 0.01, `rectangles should exactly tile the box, got area ${totalArea}`);

console.log("chart helpers regression: ok");