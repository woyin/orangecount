// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Validate candidate-only output from the controlled Fava reference runner.
// This is deliberately an evidence-completeness check, not a visual approval
// rule: it verifies the pinned environment, fixture hash, and four-cell
// screenshot matrix without comparing screenshots to OrangeCount output.

import { createHash } from "node:crypto";
import { readFile, readdir, realpath } from "node:fs/promises";
import path from "node:path";

const repoRoot = path.resolve(import.meta.dirname, "../..");
const fixtureBase = path.resolve(repoRoot, "testdata/fixtures");
const requestedFixture = path.resolve(repoRoot, process.env.FAVA_REFERENCE_FIXTURE || "testdata/fixtures/fava-reference");
const fixtureBaseReal = await realpath(fixtureBase);
const fixtureRoot = await realpath(requestedFixture);
const fixtureRelative = path.relative(fixtureBaseReal, fixtureRoot);
if (!fixtureRelative || fixtureRelative.startsWith("..") || path.isAbsolute(fixtureRelative)) {
  throw new Error("FAVA_REFERENCE_FIXTURE must resolve below testdata/fixtures");
}
const outputRoot = path.resolve(repoRoot, process.env.FAVA_REFERENCE_OUTPUT || "testdata/visual-candidates/fava-reference");
const lockPath = path.join(outputRoot, "environment-lock.json");
const routeNames = ["shell-journal", "income-statement", "balance-sheet", "trial-balance", "account-detail"];
const projects = ["desktop-light", "desktop-dark", "narrow-light", "narrow-dark"];

async function files(dir, prefix = "") {
  const result = [];
  for (const entry of (await readdir(dir, { withFileTypes: true })).sort((a, b) => a.name.localeCompare(b.name))) {
    const relative = path.posix.join(prefix, entry.name);
    if (entry.isDirectory()) result.push(...await files(path.join(dir, entry.name), relative));
    else result.push([relative, await readFile(path.join(dir, entry.name))]);
  }
  return result;
}

function fixtureHash(entries) {
  const hash = createHash("sha256");
  for (const [name, data] of entries) {
    hash.update(name); hash.update("\0"); hash.update(data); hash.update("\0");
  }
  return hash.digest("hex");
}

function pngSize(data) {
  const signature = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]);
  if (data.length < 24 || !data.subarray(0, 8).equals(signature) || data.toString("ascii", 12, 16) !== "IHDR") return null;
  return { width: data.readUInt32BE(16), height: data.readUInt32BE(20) };
}

function requireValue(errors, condition, message) {
  if (!condition) errors.push(message);
}

async function main() {
  const errors = [];
  let lock;
  try {
    lock = JSON.parse(await readFile(lockPath, "utf8"));
  } catch (error) {
    console.error(`reference-output: unable to read ${lockPath}: ${error.message}`);
    process.exit(1);
  }
  const fixtureEntries = await files(fixtureRoot);
  requireValue(errors, lock.schema_version === 1, "environment lock schema_version must be 1");
  requireValue(errors, lock.image_id && lock.image_id.startsWith("sha256:"), "environment lock must record an immutable image id");
  requireValue(errors, lock.fava?.version === "1.30.12", "environment lock Fava version is not 1.30.12");
  requireValue(errors, lock.fava?.commit === "aa7538e8971252c9efc52c8a516a3a77d604553f", "environment lock Fava commit is not pinned");
  requireValue(errors, /^v22\./.test(lock.node || ""), "environment lock must use Node 22");
  requireValue(errors, lock.playwright === "1.52.0", "environment lock Playwright version is not 1.52.0");
  requireValue(errors, /^Chromium /.test(lock.chromium || ""), "environment lock must record Chromium");
  requireValue(errors, lock.locale === "en-US" && lock.timezone === "UTC", "environment lock locale/timezone must be en-US/UTC");
  requireValue(errors, lock.reduced_motion === "reduce" && lock.device_scale_factor === 1, "environment lock browser settings are not deterministic");
  requireValue(errors, lock.capture?.candidate_only === true, "reference output must remain candidate-only");
  requireValue(errors, lock.capture?.formal_baseline_directory !== lock.capture?.output_directory, "candidate and formal baseline directories must differ");
  requireValue(errors, lock.fixture_content_sha256 === fixtureHash(fixtureEntries), "fixture content hash does not match the captured lock");

  for (const project of projects) {
    const [width, height] = project.startsWith("desktop") ? [1280, 800] : [520, 800];
    for (const route of routeNames) {
      const screenshot = path.join(outputRoot, "screenshots", project, "visual", "fava-reference.spec.mjs", `${route}.png`);
      try {
        const data = await readFile(screenshot);
        const size = pngSize(data);
        requireValue(errors, size?.width === width && size?.height >= height, `${project}/${route}.png has unexpected dimensions`);
      } catch (error) {
        errors.push(`missing candidate screenshot '${path.relative(repoRoot, screenshot)}': ${error.message}`);
      }
    }
  }

  if (errors.length > 0) {
    for (const error of errors) console.error(`reference-output: ${error}`);
    console.error(`reference-output: ${errors.length} problem(s)`);
    process.exit(1);
  }
  console.log(`reference-output: complete candidate matrix (${projects.length * routeNames.length} screenshots, fixture hash verified)`);
}

await main();
