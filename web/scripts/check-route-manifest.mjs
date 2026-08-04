// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Prerequisite Phase 0 machine-checkable route/state manifest completeness
// checker (Deliverable A).
//
// It reads docs/fava-route-state-manifest.md and fails (non-zero exit) when
// any of the following holds:
//   - an ID is not unique across all registry tables;
//   - the R / G / M / D / X category is not present;
//   - a core route ID (R-*) defined by CORE_ROUTE_IDS is missing;
//   - a status-table row (R/G/M/D) lacks a Wave or Current status;
//   - an exclusion-table row (X) lacks a Reason.
//
// The core check is exported as a pure function so node:test unit tests can
// exercise both passing and failing manifests without touching the docs.

import { readFile } from "node:fs/promises";
import { realpathSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";

const MANIFEST_PATH = path.resolve(import.meta.dirname, "../../docs/fava-route-state-manifest.md");

// Core standard-surface route IDs the manifest must cover. These are the
// R-* rows of the route registry; a missing one is a coverage gap.
export const CORE_ROUTE_IDS = Object.freeze([
  "R-ROOT", "R-IS", "R-BS", "R-TB", "R-JOURNAL", "R-ACCOUNT",
  "R-HOLD-ACCOUNT", "R-HOLD-CURRENCY", "R-HOLD-ROOT", "R-HOLD-COMMODITY",
  "R-COMMODITIES", "R-DOCUMENTS", "R-EVENTS", "R-STATISTICS", "R-ERRORS",
  "R-HELP", "R-QUERY", "R-EDITOR", "R-IMPORT", "R-OPTIONS",
]);

const CATEGORIES = ["R", "G", "M", "D", "X"];
const ID_FORMAT = /^[A-Z]-[A-Z0-9-]+$/;

function splitCells(line) {
  const cells = line.trim().split("|").map((c) => c.trim());
  // First and last cells are empty (around the leading/trailing |).
  return cells.slice(1, Math.max(1, cells.length - 1));
}

function isSeparator(cells) {
  return cells.length > 0 && cells.every((c) => /^:?-+:?$/.test(c));
}

export function parseTables(markdown) {
  const lines = markdown.split("\n");
  const tables = [];
  let i = 0;
  while (i < lines.length) {
    if (!lines[i].trim().startsWith("|")) {
      i++;
      continue;
    }
    const block = [];
    while (i < lines.length && lines[i].trim().startsWith("|")) {
      block.push(lines[i]);
      i++;
    }
    if (block.length === 0) continue;
    const header = splitCells(block[0]);
    const rows = [];
    for (let j = 1; j < block.length; j++) {
      const cells = splitCells(block[j]);
      if (isSeparator(cells)) continue;
      // A repeated header row (e.g. when a table continues across
      // subsections) is not a data row.
      if (cells.length === header.length && cells.every((c, k) => c === header[k])) continue;
      rows.push(cells);
    }
    tables.push({ header, rows });
  }
  return tables;
}

function isStatusTable(header) {
  return header.includes("Wave") && header.includes("Current status");
}

function isExclusionTable(header) {
  return header.includes("Reason");
}

/**
 * checkRouteManifest verifies completeness of a route/state manifest.
 * Returns a list of human-readable problems (empty when the manifest is
 * complete). Does not throw.
 */
export function checkRouteManifest(markdown, coreRouteIds = CORE_ROUTE_IDS) {
  const errors = [];
  const tables = parseTables(markdown);
  const seen = new Map(); // id -> table type
  const categories = new Set();

  for (const table of tables) {
    const isStatus = isStatusTable(table.header);
    const isExclusion = isExclusionTable(table.header);
    if (!isStatus && !isExclusion) continue; // ignore non-registry tables

    const waveIdx = table.header.indexOf("Wave");
    const statusIdx = table.header.indexOf("Current status");
    const reasonIdx = table.header.indexOf("Reason");

    for (const row of table.rows) {
      const id = row[0];
      if (!id) continue; // blank row
      const cell = (idx) => (idx >= 0 && idx < row.length ? row[idx].trim() : "");

      if (!ID_FORMAT.test(id)) {
        errors.push(`invalid ID format: '${id}'`);
        continue;
      }
      const category = id[0];
      categories.add(category);

      if (isStatus && !["R", "G", "M", "D"].includes(category)) {
        errors.push(`status-table row '${id}' has category '${category}' (expected R/G/M/D)`);
      }
      if (isExclusion && category !== "X") {
        errors.push(`exclusion-table row '${id}' has category '${category}' (expected X)`);
      }

      if (seen.has(id)) {
        errors.push(`duplicate ID: '${id}'`);
      } else {
        seen.set(id, isStatus ? "status" : "exclusion");
      }

      if (isStatus) {
        if (!cell(waveIdx) || !cell(statusIdx)) {
          errors.push(`row '${id}' missing Wave or Current status`);
        }
      } else if (isExclusion) {
        if (!cell(reasonIdx)) {
          errors.push(`exclusion '${id}' missing Reason`);
        }
      }
    }
  }

  for (const category of CATEGORIES) {
    if (!categories.has(category)) {
      errors.push(`missing category '${category}'`);
    }
  }

  for (const id of coreRouteIds) {
    if (!seen.has(id)) {
      errors.push(`missing core route ID '${id}'`);
    }
  }

  return errors;
}

async function main() {
  const markdown = await readFile(MANIFEST_PATH, "utf8");
  const errors = checkRouteManifest(markdown);
  if (errors.length > 0) {
    for (const error of errors) {
      console.error(`route-manifest: ${error}`);
    }
    console.error(`route-manifest: ${errors.length} problem(s); manifest incomplete`);
    process.exit(1);
  }
  console.log("route-manifest: complete (unique IDs, all categories, core set, statuses/reasons present)");
}

if (process.argv[1]) {
  const invoked = process.argv[1];
  let resolved;
  try {
    resolved = realpathSync(invoked);
  } catch {
    resolved = path.resolve(invoked);
  }
  if (resolved === realpathSync(fileURLToPath(import.meta.url))) {
    main().catch((error) => {
      console.error(`route-manifest: ${error.message}`);
      process.exit(1);
    });
  }
}