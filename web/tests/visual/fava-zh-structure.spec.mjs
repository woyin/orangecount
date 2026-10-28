// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

// The plan requires a Simplified Chinese structural suite: every route must
// reproduce the English structure under zh-CN. "Structure" here means the
// rendered element tree and control order, never pixels — the approved visual
// bar is the four Chromium cells only.
//
// The check compares a locale-independent fingerprint of the route's own
// container between en and zh-CN. Text is expected to differ; if the element
// tree, class identity, or control order changes, the two fingerprints diverge
// and the route fails. The route list comes from router.mjs so a new route
// cannot silently drop out of the suite.

import { test, expect } from "./fixtures.mjs";
import { ROUTES } from "../../src/fava/router.mjs";

const ROUTE_PATHS = {
  income_statement: "/income_statement",
  balance_sheet: "/balance_sheet",
  trial_balance: "/trial_balance",
  journal: "/journal",
  query: "/query",
  holdings: "/holdings",
  holdings_by_account: "/holdings/by_account",
  holdings_by_currency: "/holdings/by_currency",
  holdings_by_cost_currency: "/holdings/by_cost_currency",
  holdings_by_root_account: "/holdings/by_root_account",
  holdings_by_commodity: "/holdings/by_commodity",
  commodities: "/commodities",
  documents: "/documents",
  events: "/events",
  statistics: "/statistics",
  editor: "/editor",
  import: "/import",
  options: "/options",
  help: "/help",
  source: "/source",
  diagnostics: "/diagnostics",
  errors: "/errors",
  "quick-profile": "/quick-profile",
  planning: "/planning",
};

// The shell takes its locale from localStorage (the sidebar switch writes this
// key); it does not read a `?locale=` query parameter. Driving the URL would
// therefore compare English against English and prove nothing, which is exactly
// what an earlier version of this suite did.
const LOCALE_KEY = "orangecount-locale";

async function applyLocale(page, locale) {
  await page.addInitScript(
    ([key, value]) => {
      try {
        window.localStorage.setItem(key, value);
      } catch {
        /* storage is optional in the shell */
      }
    },
    [LOCALE_KEY, locale],
  );
}

// A structural fingerprint: element tag plus its sorted class list, in document
// order, for the route container. No text, no geometry, no styling values.
async function structureFingerprint(page, selector) {
  return page.locator(selector).evaluate((root) => {
    const rows = [];
    const walk = (node, depth) => {
      for (const child of node.children) {
        const classes = [...child.classList].sort().join(".");
        rows.push(`${depth}:${child.tagName.toLowerCase()}.${classes}`);
        walk(child, depth + 1);
      }
    };
    walk(root, 0);
    return rows;
  });
}

async function sidebarOrder(page) {
  return page.locator("#sidebar a").evaluateAll((links) => links.map((link) => link.getAttribute("href")));
}

async function openRoute(page, route, locale) {
  await applyLocale(page, locale);
  await page.goto(route, { waitUntil: "networkidle" });
  await expect(page.locator("#page-title")).not.toHaveText("");
  await expect(page.locator("article")).toBeVisible();
  await expect(page.locator("article")).not.toContainText("The report could not be loaded");
  await expect(page.locator("article")).not.toContainText("报告无法加载");
}

test.describe("Simplified Chinese structural parity", () => {
  for (const name of ROUTES) {
    const route = ROUTE_PATHS[name];
    test(`${name} keeps the English structure under zh-CN`, async ({ page }) => {
      await openRoute(page, route, "en");
      const englishStructure = await structureFingerprint(page, "article");
      const englishSidebar = await sidebarOrder(page);

      await openRoute(page, route, "zh-CN");
      const chineseStructure = await structureFingerprint(page, "article");
      const chineseSidebar = await sidebarOrder(page);

      expect(englishStructure.length, `${name} rendered an empty article`).toBeGreaterThan(0);
      // Element identity and control order must survive the locale switch.
      expect(chineseStructure, `${name} structure changed under zh-CN`).toEqual(englishStructure);
      expect(chineseSidebar, `${name} navigation order changed under zh-CN`).toEqual(englishSidebar);
    });
  }
});

// A structural suite that passes because the locale never applied would prove
// nothing, so at least one route must visibly change its text.
// A suite that passes because the locale never applied would prove nothing.
// The page title is a route label, not a translated string, so the probe reads
// the article text and the document language instead.
test("zh-CN is actually applied to the rendered shell", async ({ page }) => {
  const snapshot = async (locale) => {
    await openRoute(page, "/income_statement", locale);
    return {
      lang: await page.evaluate(() => document.documentElement.lang),
      text: (await page.locator("article").innerText()).replace(/\\s+/g, " ").trim(),
    };
  };
  const english = await snapshot("en");
  const chinese = await snapshot("zh-CN");
  expect(chinese.lang, "zh-CN did not reach the document language").toBe("zh-CN");
  expect(chinese.text, "zh-CN rendered identical text to en").not.toBe(english.text);
});
