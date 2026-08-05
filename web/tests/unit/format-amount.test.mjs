import assert from "node:assert/strict";
import test from "node:test";

import { formatAmount } from "../../src/fava/reports/types.ts";

const wire = (display) => ({ display, exact: display, approximate: false });

test("leaves amounts ungrouped unless the ledger asks for commas", () => {
  assert.equal(formatAmount(wire("1622607.76")), "1622607.76");
  assert.equal(formatAmount(wire("1622607.76"), false), "1622607.76");
  assert.equal(formatAmount(undefined), "");
});

test("groups thousands only in the integer part", () => {
  assert.equal(formatAmount(wire("1622607.76"), true), "1,622,607.76");
  assert.equal(formatAmount(wire("100"), true), "100");
  assert.equal(formatAmount(wire("1000"), true), "1,000");
  assert.equal(formatAmount(wire("260000"), true), "260,000");
  assert.equal(formatAmount(wire("0.123456"), true), "0.123456");
});

test("keeps the sign and never regroups the fraction", () => {
  assert.equal(formatAmount(wire("-410242.97"), true), "-410,242.97");
  assert.equal(formatAmount(wire("-0.41"), true), "-0.41");
  assert.equal(formatAmount(wire("1234512345.123"), true), "1,234,512,345.123");
});

test("leaves a non-terminating rational untouched", () => {
  // Decimal.String emits a slash only for a non-terminating rational; grouping
  // it would corrupt a value the user must be able to read exactly.
  assert.equal(formatAmount(wire("1000/3"), true), "1000/3");
});
