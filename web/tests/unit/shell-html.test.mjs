import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import path from "node:path";
import test from "node:test";

import { ROUTES, routeHref } from "../../src/fava/router.mjs";

const repoRoot = path.resolve(import.meta.dirname, "..", "..", "..");
const shellPath = path.join(repoRoot, "internal", "web", "assets", "transplanted", "index.html");

// The Go server returns this one shell for every client route, so a relative
// asset URL resolves against the route's own directory: under /income_statement/
// a "./app.js" would request /income_statement/app.js, which the catch-all
// handler answers with the shell HTML itself. The module then fails its MIME
// check and the page renders blank on every deep link and every refresh.
test("shell references its assets with root-absolute URLs", async () => {
  const html = await readFile(shellPath, "utf8");
  const references = [...html.matchAll(/(?:src|href)="([^"]+)"/g)].map(([, value]) => value);
  assert.ok(references.length > 0, "shell should reference at least one asset");
  for (const reference of references) {
    assert.ok(
      reference.startsWith("/") && !reference.startsWith("//"),
      `asset reference ${reference} must be root-absolute so nested routes resolve it`,
    );
  }
  assert.ok(references.includes("/app.js"), "shell should load /app.js");
  assert.ok(references.includes("/app.css"), "shell should load /app.css");
});

// Guards the same failure from the routing side: every known route is served
// the shell, so each one must sit at a path where those absolute URLs work.
test("every route is a nested path served by the same shell", () => {
  for (const route of ROUTES) {
    const href = routeHref(route);
    assert.ok(href.startsWith("/"), `${route} should resolve to an absolute path`);
  }
  assert.equal(routeHref("income_statement"), "/income_statement");
  assert.equal(routeHref("holdings_by_currency"), "/holdings/by_currency");
});
