// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

/**
 * Deterministic regression test for the embedded Journal table.
 *
 * The bundle is intentionally dependency-free and is not a JavaScript module,
 * so this test evaluates only the table helpers in a tiny DOM-shaped harness.
 * It exercises the public interaction seam (header click and next-page click)
 * rather than asserting implementation-only row arrays.
 */
import assert from "node:assert/strict";
import fs from "node:fs";
import vm from "node:vm";

const sourcePath = new URL("./app.js", import.meta.url);
const source = fs.readFileSync(sourcePath, "utf8");

function extractFunction(name) {
  const marker = `function ${name}(`;
  const start = source.indexOf(marker);
  assert.notEqual(start, -1, `missing ${name} in app.js`);
  let parentheses = 0;
  let open = -1;
  for (let index = source.indexOf("(", start); index < source.length; index += 1) {
    if (source[index] === "(") parentheses += 1;
    if (source[index] === ")" && --parentheses === 0) {
      open = source.indexOf("{", index);
      break;
    }
  }
  assert.notEqual(open, -1, `missing body for ${name}`);
  let depth = 0;
  let quote = "";
  let escaped = false;
  for (let index = open; index < source.length; index += 1) {
    const character = source[index];
    if (quote) {
      if (escaped) escaped = false;
      else if (character === "\\") escaped = true;
      else if (character === quote) quote = "";
      continue;
    }
    if (character === "'" || character === '"' || character === "`") {
      quote = character;
      continue;
    }
    if (character === "{") depth += 1;
    if (character === "}" && --depth === 0) return source.slice(start, index + 1);
  }
  throw new Error(`unterminated ${name}`);
}

class FakeElement {
  constructor(tagName) {
    this.tagName = tagName;
    this.dataset = {};
    this.children = [];
    this.listeners = new Map();
    this.hidden = false;
    this.disabled = false;
    this.textContent = "";
    this.parentElement = null;
  }

  addEventListener(type, listener) {
    this.listeners.set(type, listener);
  }

  dispatchEvent(event) {
    this.listeners.get(event.type)?.(event);
  }

  click() {
    this.listeners.get("click")?.({ type: "click", target: this });
  }

  append(...children) {
    this.children.push(...children);
    children.forEach((child) => { child.parentElement = this; });
  }

  setAttribute(name, value) {
    this[name] = String(value);
  }
}

class FakeCell extends FakeElement {
  constructor(value, kind = "text") {
    super("TD");
    this.dataset.sortValue = value;
    this.dataset.sortKind = kind;
  }
}

class FakeRow extends FakeElement {
  constructor(value) {
    super("TR");
    this.cells = [new FakeCell(value, "decimal")];
    this.textContent = value;
  }
}

class FakeBody extends FakeElement {
  constructor(rows) {
    super("TBODY");
    this.rows = rows;
    rows.forEach((row) => { row.parentElement = this; });
  }

  appendChild(row) {
    const index = this.rows.indexOf(row);
    if (index >= 0) this.rows.splice(index, 1);
    this.rows.push(row);
    row.parentElement = this;
    return row;
  }
}

class FakeTable extends FakeElement {
  constructor(rows, sortButton) {
    super("TABLE");
    this.tBodies = [new FakeBody(rows)];
    this.sortButton = sortButton;
    this.parentElement = new FakeElement("DIV");
    this.parentElement.parentElement = new FakeElement("DIV");
  }

  querySelectorAll(selector) {
    if (selector === "button.table-sort") return [this.sortButton];
    return [];
  }
}

class FakeContainer extends FakeElement {
  constructor(table) {
    super("MAIN");
    this.table = table;
    this.pagination = null;
  }

  querySelector(selector) {
    if (selector === "table") return this.table;
    return null;
  }

  querySelectorAll(selector) {
    if (selector === "table[data-sortable]") return [this.table];
    return [];
  }
}

const context = {
  console,
  CustomEvent: class CustomEvent {
    constructor(type) { this.type = type; }
  },
  document: {
    createElement(tagName) { return new FakeElement(tagName.toUpperCase()); },
  },
  t(key) { return key; },
};
vm.runInNewContext([
  extractFunction("decimalParts"),
  extractFunction("compareTyped"),
  extractFunction("tableSortKind"),
  extractFunction("sortTableRows"),
  extractFunction("wireTables"),
  extractFunction("wirePagination"),
].join("\n"), context);

const values = Array.from({ length: 51 }, (_, index) => String(50 - index));
const rows = values.map((value) => new FakeRow(value));
const sortButton = new FakeElement("BUTTON");
sortButton.dataset.columnIndex = "0";
const table = new FakeTable(rows, sortButton);
const container = new FakeContainer(table);
table.parentElement.parentElement.insertBefore = (controls) => { container.pagination = controls; };

context.wireTables(container);
context.wirePagination(container, 50);

// The source rows are descending. Sorting ascending must reorder all 51 rows,
// then pagination must slice that sorted sequence rather than stale row indexes.
sortButton.click();
assert.deepEqual(
  table.tBodies[0].rows.filter((row) => !row.hidden).map((row) => row.cells[0].dataset.sortValue),
  Array.from({ length: 50 }, (_, index) => String(index)),
  "first page should contain the first 50 globally sorted rows",
);

const next = container.pagination?.children.at(-1);
assert.ok(next, "pagination should expose a next-page control");
next.click();
assert.deepEqual(
  table.tBodies[0].rows.filter((row) => !row.hidden).map((row) => row.cells[0].dataset.sortValue),
  ["50"],
  "second page should contain the remaining globally sorted row",
);

// Decimal sorting must remain numeric, not lexical, for journal amount cells.
const decimalRows = [new FakeRow("10"), new FakeRow("2"), new FakeRow("1")];
const decimalButton = new FakeElement("BUTTON");
decimalButton.dataset.columnIndex = "0";
const decimalTable = new FakeTable(decimalRows, decimalButton);
const decimalContainer = new FakeContainer(decimalTable);
context.wireTables(decimalContainer);
decimalButton.click();
assert.deepEqual(
  decimalTable.tBodies[0].rows.map((row) => row.cells[0].dataset.sortValue),
  ["1", "2", "10"],
  "decimal columns should sort by exact numeric value",
);

const stableRows = [new FakeRow("2"), new FakeRow("2"), new FakeRow("1")];
stableRows.forEach((row, index) => { row.dataset.rowId = String(index); });
const stableButton = new FakeElement("BUTTON");
stableButton.dataset.columnIndex = "0";
const stableTable = new FakeTable(stableRows, stableButton);
const stableContainer = new FakeContainer(stableTable);
context.wireTables(stableContainer);
stableButton.click();
assert.deepEqual(
  stableTable.tBodies[0].rows.map((row) => row.dataset.rowId),
  ["2", "0", "1"],
  "equal values should retain their original relative order",
);

console.log("journal sorting pagination regression: ok");
