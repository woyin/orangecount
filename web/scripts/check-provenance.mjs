// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Prerequisite Phase 0 provenance guard v1 (Deliverable B).
//
// The guard keeps one canonical managed inventory at web/provenance-manifest.json
// and verifies two directions:
//
//  1. Completeness — every managed source/asset on disk under the manifest's
//     managed_roots is listed in the manifest. Nothing with a current_kind is
//     allowed to be missing from the inventory.
//  2. Honest classification — every entry must carry one of the allowed
//     current_kind values. The current clean-room prototype and legacy shell
//     must be classified as prototype/legacy/original — never derived. A
//     "derived" entry that carries no upstream revision, upstream hash, and
//     notice location fails, because a Fava-derived file cannot be added
//     without full provenance evidence (ADR-0030).
//
// Excluded directories (node_modules, staging, test-results,
// playwright-report, dist) are never scanned. Files in managed_roots that
// appear on disk but have no manifest entry fail, so adding an unmanaged
// source/asset under web/src or internal/web/assets is an error.
//
// The manifest itself is committed; entries are added by the owning
// implementation-wave agents when they import or adapt real Fava units.

import { readFile, readdir, stat } from "node:fs/promises";
import { realpathSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";

const WEB_ROOT = path.resolve(import.meta.dirname, "..");
const MANIFEST_PATH = path.join(WEB_ROOT, "provenance-manifest.json");
const REPO_ROOT = path.resolve(WEB_ROOT, "..");

export const ALLOWED_KINDS = Object.freeze([
  "prototype",
  "legacy",
  "original",
  "derived",
]);

// A derived entry (Fava-influenced) must carry these fields; a missing one
// means the file cannot be proven derivative-safe.
const DERIVED_REQUIRED_FIELDS = ["upstream", "upstream_revision", "upstream_hash", "notice"];
const MANAGEABLE_EXTENSIONS = new Set([
  ".ts", ".mts", ".cts", ".js", ".mjs", ".cjs", ".css", ".svelte", ".html",
  ".json", ".md", ".map", ".svg", ".woff", ".woff2", ".ttf", ".eot",
  ".ico", ".png", ".jpg", ".gif", ".webp", ".txt",
]);

function extensionOf(name) {
  const dot = name.lastIndexOf(".");
  if (dot < 0) return "";
  return name.slice(dot).toLowerCase();
}

async function collectFiles(root, excludedDirs, relative = "") {
  const out = [];
  const entries = await readdir(root, { withFileTypes: true });
  entries.sort((a, b) => a.name.localeCompare(b.name));
  for (const entry of entries) {
    if (entry.name.startsWith(".git")) continue;
    const full = path.join(root, entry.name);
    const rel = relative ? `${relative}/${entry.name}` : entry.name;
    if (entry.isDirectory()) {
      if (excludedDirs.has(entry.name)) continue;
      out.push(...(await collectFiles(full, excludedDirs, rel)));
    } else if (entry.isFile() && MANAGEABLE_EXTENSIONS.has(extensionOf(entry.name))) {
      out.push(rel);
    }
  }
  return out;
}

function resolveRoot(rel, repoRoot) {
  // Entries in the manifest are slash-separated relative to REPO_ROOT
  // (e.g. web/src/fava/App.svelte, internal/web/assets/app.js).
  return path.join(repoRoot, ...rel.split("/"));
}

async function collectFromManifestRoots(roots, excludedDirs, repoRoot) {
  const files = [];
  for (const root of roots) {
    const abs = resolveRoot(root, repoRoot);
    try {
      const info = await stat(abs);
      if (info.isDirectory()) {
        files.push(...(await collectFiles(abs, excludedDirs, root)));
      } else if (info.isFile() && MANAGEABLE_EXTENSIONS.has(extensionOf(root))) {
        files.push(root);
      }
    } catch {
      // Missing roots are detected as stale entries when the inventory names
      // files beneath them; an empty optional root contributes no files.
    }
  }
  return files;
}

/**
 * checkProvenance verifies the managed inventory. Returns a list of
 * human-readable problems (empty when all is well). Pure with respect to
 * filesystem I/O; used by the CLI and by unit tests.
 *
 * The optional manifestPath and repoRoot parameters let tests point the
 * guard at a scratch inventory and tree without touching the real repo.
 */
export async function checkProvenance(manifestPath = MANIFEST_PATH, repoRoot = REPO_ROOT) {
  const errors = [];
  const raw = await readFile(manifestPath, "utf8");
  let manifest;
  try {
    manifest = JSON.parse(raw);
  } catch (error) {
    return [`provenance-manifest.json is not valid JSON: ${error.message}`];
  }
  if (manifest.schema_version !== 1) {
    return [`provenance-manifest.json schema_version=${manifest.schema_version}; expected 1`];
  }
  const roots = manifest.managed_roots || [];
  const excludedDirs = new Set(manifest.excluded_dirs || []);
  const entries = manifest.entries || {};

  // 1. Syntax/classification checks on every entry.
  const known = new Map();
  for (const [rel, meta] of Object.entries(entries)) {
    const kind = meta.current_kind;
    if (!ALLOWED_KINDS.includes(kind)) {
      errors.push(`entry '${rel}' has invalid current_kind '${kind}'`);
    }
    if (kind === "derived") {
      for (const field of DERIVED_REQUIRED_FIELDS) {
        if (!meta[field]) {
          errors.push(`entry '${rel}' is derived but missing required field '${field}'`);
        }
      }
    } else if (meta.upstream) {
      errors.push(`entry '${rel}' declares upstream '${meta.upstream}' but is not derived`);
    }
    known.set(rel, kind);
  }

  // 2. Completeness: every managed file on disk must be listed.
  const onDisk = new Set(await collectFromManifestRoots(roots, excludedDirs, repoRoot));
  for (const rel of onDisk) {
    if (!known.has(rel)) {
      errors.push(`unmanaged file on disk: '${rel}' (add a row to web/provenance-manifest.json)`);
    }
  }

  // 3. Staleness: every entry must still exist on disk (no orphans).
  for (const rel of known.keys()) {
    if (!onDisk.has(rel) && MANAGEABLE_EXTENSIONS.has(extensionOf(rel))) {
      errors.push(`manifest entry no longer exists on disk: '${rel}'`);
    }
  }

  return errors;
}

async function main() {
  try {
    const errors = await checkProvenance();
    if (errors.length > 0) {
      for (const error of errors) {
        console.error(`provenance: ${error}`);
      }
      console.error(`provenance: ${errors.length} problem(s)`);
      process.exit(1);
    }
    console.log("provenance: managed inventory complete and honestly classified");
  } catch (error) {
    console.error(`provenance: ${error.message}`);
    process.exit(1);
  }
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
    main();
  }
}
