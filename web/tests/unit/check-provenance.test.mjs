// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import assert from "node:assert/strict";
import { mkdtemp, mkdir, writeFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import test from "node:test";

import { checkProvenance, ALLOWED_KINDS } from "../../scripts/check-provenance.mjs";

async function scratch() {
  const dir = await mkdtemp(path.join(tmpdir(), "oc-prov-"));
  return dir;
}

const BASE_MANIFEST = {
  schema_version: 1,
  managed_roots: ["web/src", "internal/web/assets"],
  excluded_dirs: ["node_modules", "staging", "test-results", "playwright-report", "dist"],
  entries: {},
};

async function writeManifest(repoRoot, manifest) {
  const target = path.join(repoRoot, "web", "provenance-manifest.json");
  await mkdir(path.dirname(target), { recursive: true });
  await writeFile(target, JSON.stringify(manifest, null, 2));
  return target;
}

test("allowed kinds include prototype/legacy/original but the guard rejects derived without provenance", async () => {
  assert.ok(ALLOWED_KINDS.includes("prototype"));
  assert.ok(ALLOWED_KINDS.includes("legacy"));
  assert.ok(ALLOWED_KINDS.includes("original"));
  assert.ok(ALLOWED_KINDS.includes("derived"));

  const repoRoot = await scratch();
  try {
    const src = path.join(repoRoot, "web", "src", "fava");
    await mkdir(src, { recursive: true });
    await writeFile(path.join(src, "App.svelte"), "export {};\n");
    await writeManifest(repoRoot, {
      ...BASE_MANIFEST,
      entries: {
        "web/src/fava/App.svelte": { current_kind: "derived", upstream: "frontend/src/App.svelte" },
      },
    });
    const errors = await checkProvenance(
      path.join(repoRoot, "web", "provenance-manifest.json"),
      repoRoot,
    );
    assert.ok(
      errors.some((e) => e.includes("missing required field 'upstream_revision'")),
      errors.join("; "),
    );
  } finally {
    await rm(repoRoot, { recursive: true, force: true });
  }
});

test("unmanaged file on disk under a managed root is reported", async () => {
  const repoRoot = await scratch();
  try {
    const src = path.join(repoRoot, "web", "src", "fava");
    await mkdir(src, { recursive: true });
    await writeFile(path.join(src, "App.svelte"), "export {};\n");
    await writeFile(path.join(src, "new-unlisted.ts"), "export {};\n");
    await writeManifest(repoRoot, {
      ...BASE_MANIFEST,
      entries: {
        "web/src/fava/App.svelte": { current_kind: "prototype" },
      },
    });
    const errors = await checkProvenance(
      path.join(repoRoot, "web", "provenance-manifest.json"),
      repoRoot,
    );
    assert.ok(errors.some((e) => e.includes("unmanaged file on disk: 'web/src/fava/new-unlisted.ts'")), errors.join("; "));
  } finally {
    await rm(repoRoot, { recursive: true, force: true });
  }
});

test("manifest-configured generated directories are never scanned", async () => {
  const repoRoot = await scratch();
  try {
    const src = path.join(repoRoot, "web", "src", "fava");
    await mkdir(path.join(src, "node_modules", "esbuild"), { recursive: true });
    await mkdir(path.join(src, "staging"), { recursive: true });
    await mkdir(path.join(src, "generated-cache"), { recursive: true });
    await writeFile(path.join(src, "node_modules", "esbuild", "lib.js"), "export {};\n");
    await writeFile(path.join(src, "staging", "out.js"), "export {};\n");
    await writeFile(path.join(src, "generated-cache", "out.js"), "export {};\n");
    await writeFile(path.join(src, "App.svelte"), "export {};\n");
    await writeManifest(repoRoot, {
      ...BASE_MANIFEST,
      excluded_dirs: [...BASE_MANIFEST.excluded_dirs, "generated-cache"],
      entries: {
        "web/src/fava/App.svelte": { current_kind: "prototype" },
      },
    });
    const errors = await checkProvenance(
      path.join(repoRoot, "web", "provenance-manifest.json"),
      repoRoot,
    );
    assert.deepEqual(errors, []);
  } finally {
    await rm(repoRoot, { recursive: true, force: true });
  }
});

test("clean tree with honest classification passes", async () => {
  const repoRoot = await scratch();
  try {
    const src = path.join(repoRoot, "web", "src", "fava");
    await mkdir(src, { recursive: true });
    await writeFile(path.join(src, "App.svelte"), "export {};\n");
    await writeManifest(repoRoot, {
      ...BASE_MANIFEST,
      entries: {
        "web/src/fava/App.svelte": { current_kind: "prototype" },
      },
    });
    const errors = await checkProvenance(
      path.join(repoRoot, "web", "provenance-manifest.json"),
      repoRoot,
    );
    assert.deepEqual(errors, []);
  } finally {
    await rm(repoRoot, { recursive: true, force: true });
  }
});

test("legacy and original kinds pass without upstream fields", async () => {
  const repoRoot = await scratch();
  try {
    const src = path.join(repoRoot, "internal", "web", "assets");
    await mkdir(src, { recursive: true });
    await writeFile(path.join(src, "app.js"), "export {};\n");
    await writeManifest(repoRoot, {
      schema_version: 1,
      managed_roots: ["internal/web/assets"],
      excluded_dirs: [],
      entries: {
        "internal/web/assets/app.js": { current_kind: "legacy" },
      },
    });
    const errors = await checkProvenance(
      path.join(repoRoot, "web", "provenance-manifest.json"),
      repoRoot,
    );
    assert.deepEqual(errors, []);
  } finally {
    await rm(repoRoot, { recursive: true, force: true });
  }
});
