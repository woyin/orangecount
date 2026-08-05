import assert from "node:assert/strict";
import test from "node:test";

import { fuzzyfilter, fuzzytest, fuzzywrap } from "../../src/fava/lib/fuzzy.ts";

test("fuzzytest scores exact substrings quadratically", () => {
  assert.equal(fuzzytest("cash", "Assets:Cash"), 16);
  assert.ok(fuzzytest("acs", "Assets:Cash:Reserve01") > 0);
  assert.equal(fuzzytest("zzz", "Assets:Cash"), 0);
});

test("fuzzytest treats lowercase patterns as case-insensitive", () => {
  assert.ok(fuzzytest("cash", "ASSETS:CASH") > 0);
});

test("fuzzyfilter sorts better matches first and drops misses", () => {
  const suggestions = ["Assets:Cash", "Expenses:Food", "Assets:Cash:Reserve01"];
  assert.deepEqual(fuzzyfilter("cash", suggestions), [
    "Assets:Cash",
    "Assets:Cash:Reserve01",
  ]);
  assert.deepEqual(fuzzyfilter("", suggestions), suggestions);
});

test("fuzzywrap marks exact substring matches", () => {
  assert.deepEqual(fuzzywrap("cash", "Assets:Cash"), [
    ["text", "Assets:"],
    ["match", "Cash"],
  ]);
  assert.deepEqual(fuzzywrap("", "Assets:Cash"), [["text", "Assets:Cash"]]);
});

test("fuzzywrap falls back to plain text when not all characters match", () => {
  assert.deepEqual(fuzzywrap("xyz", "Assets:Cash"), [["text", "Assets:Cash"]]);
});
