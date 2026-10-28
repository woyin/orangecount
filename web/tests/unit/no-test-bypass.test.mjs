// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// The browser harness starts a real OrangeCount server, so it must not depend on
// a test-only switch inside that server. These checks pin the contract at the
// source level: the harness exports only the loopback URL, and production Go
// code reads no environment variable at all.

import assert from "node:assert/strict";
import { readFile, readdir } from "node:fs/promises";
import path from "node:path";
import test from "node:test";

const repoRoot = path.resolve(import.meta.dirname, "..", "..", "..");

async function read(rel) {
  return readFile(path.join(repoRoot, ...rel.split("/")), "utf8");
}

async function goSources(dir) {
  const out = [];
  for (const entry of await readdir(path.join(repoRoot, dir), { withFileTypes: true })) {
    const rel = `${dir}/${entry.name}`;
    if (entry.isDirectory()) out.push(...(await goSources(rel)));
    else if (entry.name.endsWith(".go") && !entry.name.endsWith("_test.go")) out.push(rel);
  }
  return out;
}

test("the visual harness exports only the loopback base URL", async () => {
  const source = await read("web/scripts/run-visual.mjs");
  const exported = [...source.matchAll(/env:\s*\{([^}]*)\}/g)]
    .flatMap(([, body]) => [...body.matchAll(/ORANGECOUNT_[A-Z_]+/g)].map(([name]) => name));
  assert.deepEqual(exported, ["ORANGECOUNT_BASE_URL"]);
  // The retired migration flag must not come back through a script alias.
  const pkg = JSON.parse(await read("web/package.json"));
  assert.equal(
    Object.values(pkg.scripts).some((script) => script.includes("ORANGECOUNT_TRANSPLANTED_UI")),
    false,
    "no npm script may set a UI-selection switch",
  );
});

// The released binary links ./cmd/orangecount plus the packages it imports; the
// HTTP/asset surface lives in internal/web. `internal/compat` is an oracle tool
// package that the binary does not link (`go list -deps ./cmd/orangecount` has
// no internal/compat), so it is out of scope here and its INVENTORY_DEBUG
// switch can only affect tool output, never a served response.
test("the released binary's packages read no environment variable", async () => {
  const files = [...(await goSources("cmd")), ...(await goSources("internal/web"))];
  assert.ok(files.length > 0, "expected production Go sources");
  const offenders = [];
  for (const rel of files) {
    const source = await read(rel);
    for (const call of ["os.Getenv", "os.LookupEnv", "os.Environ"]) {
      if (source.includes(call)) offenders.push(`${rel}: ${call}`);
    }
  }
  assert.deepEqual(offenders, []);
});

test("the oracle tool package is not part of the served surface", async () => {
  const files = [...(await goSources("cmd")), ...(await goSources("internal/web"))];
  const importers = [];
  for (const rel of files) {
    const source = await read(rel);
    if (source.includes("orangecount/internal/compat")) importers.push(rel);
  }
  assert.deepEqual(importers, [], "internal/compat must stay out of the binary");
});

test("the transplanted bundle reads no harness environment variable", async () => {
  const bundle = await read("internal/web/assets/transplanted/app.js");
  for (const name of ["ORANGECOUNT_VISUAL_RUN", "ORANGECOUNT_BASE_URL", "ORANGECOUNT_BIN", "ORANGECOUNT_SYNC_EMBEDDED", "ORANGECOUNT_TRANSPLANTED_UI"]) {
    assert.equal(bundle.includes(name), false, `bundle must not reference ${name}`);
  }
});
