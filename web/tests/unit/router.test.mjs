import assert from "node:assert/strict";
import test from "node:test";

import { parseRoute, routeHref, updateQuery } from "../../src/fava/router.mjs";

test("uses Fava's default income statement route at the shell root", () => {
  assert.equal(parseRoute("https://orange-count.invalid/").route, "income_statement");
});

test("parses standard report routes and supported query state", () => {
  const parsed = parseRoute("https://orange-count.invalid/balance_sheet?time=year&filter=%23seed");
  assert.equal(parsed.route, "balance_sheet");
  assert.deepEqual(parsed.query, { time: "year", filter: "#seed" });
});

test("keeps Fava holdings variants and help slugs on their route families", () => {
  assert.equal(parseRoute("https://orange-count.invalid/holdings/by_currency").route, "holdings_by_currency");
  assert.equal(parseRoute("https://orange-count.invalid/help/features").route, "help");
  assert.equal(routeHref("holdings_by_currency"), "/holdings/by_currency");
  assert.deepEqual(parseRoute("https://orange-count.invalid/source?path=activity.bean").query, { path: "activity.bean" });
});

test("parses and serializes account detail without losing the account", () => {
  const parsed = parseRoute("https://orange-count.invalid/account/Assets%3AWallet%3APrimary?conversion=at_cost");
  assert.equal(parsed.route, "account");
  assert.equal(parsed.account, "Assets:Wallet:Primary");
  assert.equal(routeHref(parsed.route, parsed), "/account/Assets%3AWallet%3APrimary?conversion=at_cost");
});

test("updates one query parameter while retaining route state", () => {
  const next = updateQuery("https://orange-count.invalid/journal?time=year&account=Assets%3AWallet", { filter: "payee:sample", time: "" });
  assert.equal(next, "/journal?account=Assets%3AWallet&filter=payee%3Asample");
});
