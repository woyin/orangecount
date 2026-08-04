import assert from "node:assert/strict";
import test from "node:test";

import { initialShellState, reduceShellState } from "../../src/fava/state.mjs";

test("shell state keeps route changes bookmarkable and closes the narrow menu", () => {
  const initial = { ...initialShellState("journal"), sidebarOpen: true, account: "" };
  const next = reduceShellState(initial, { type: "route", route: "balance_sheet", query: { time: "year" } });
  assert.equal(next.route, "balance_sheet");
  assert.deepEqual(next.query, { time: "year" });
  assert.equal(next.sidebarOpen, false);
});

test("theme and locale stores accept only supported values", () => {
  const initial = initialShellState("journal");
  assert.equal(reduceShellState(initial, { type: "locale", locale: "fr" }).locale, "en");
  assert.equal(reduceShellState(initial, { type: "theme", theme: "light" }).theme, "light");
  assert.equal(reduceShellState(initial, { type: "theme", theme: "neon" }).theme, "system");
});

test("adapter errors clear loading without discarding the route", () => {
  const initial = { ...initialShellState("journal"), loading: true };
  const next = reduceShellState(initial, { type: "error", message: "adapter unavailable" });
  assert.equal(next.route, "journal");
  assert.equal(next.loading, false);
  assert.equal(next.error, "adapter unavailable");
});
