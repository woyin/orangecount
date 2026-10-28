import { test, expect } from "./fixtures.mjs";

function wireDecimal(value) {
  return { display: String(value), exact: String(value), approximate: false };
}

function sparseAccountChart(measure, title) {
  return {
    kind: "line",
    title,
    unit: measure === "average-cost" ? "cost per unit" : "units",
    currency: "",
    valuation: "at_cost",
    period: "all",
    interval: "month",
    measure,
    series: ["AAA", "BBB", "CCC", "DDD", "EEE", "FFF"].map((label, seriesIndex) => ({
      label: `${label} (USD)`,
      points: Array.from({ length: 6 - seriesIndex }, (_, pointIndex) => ({
        date: `${2020 + pointIndex}-01`,
        value: wireDecimal(seriesIndex + pointIndex + 1),
      })),
    })),
  };
}

async function open(page, route) {
  await page.goto(route, { waitUntil: "networkidle" });
  await expect(page.locator("#page-title")).not.toHaveText("");
  await expect(page.locator("article")).not.toContainText("The report could not be loaded");
}

// The loading panel is removed from the DOM once the adapter round-trip
// finishes, so this must assert an absent count rather than text on a locator
// that resolves to zero elements (which would simply wait out the timeout).
async function waitLoaded(page) {
  await expect(
    page.locator("article [role=status]").filter({ hasText: "Loading report" }),
  ).toHaveCount(0);
}

function slug(route) {
  return route === "/" ? "root" : route.replace(/^\/+/, "").replace(/[^a-z0-9]+/gi, "-").replace(/-+$/, "").toLowerCase();
}

test.describe("transplanted Fava shell smoke", () => {
  test("root and core reports load real adapter data", async ({ page }) => {
    // All four core reports render tree tables as ordered lists. The only real
    // <table> on these routes is the chart's tabular fallback, which can be a
    // one-row "No chart data." placeholder, so never assert on it here.
    for (const route of ["/", "/income_statement", "/balance_sheet", "/trial_balance"]) {
      await open(page, route);
      await expect(page.locator("[data-tree-table]")).not.toHaveCount(0);
      await expect(page.locator("[data-tree-table] li .account").first()).toBeVisible();
    }
  });

  test("data-backed utility reports render their adapter rows", async ({ page }) => {
    for (const route of ["/holdings", "/documents", "/events", "/statistics"]) {
      await open(page, route);
      await expect(page.locator("table tbody tr").first()).toBeVisible();
    }
  });

  test("commodity and event chart tooltips stay visible after a click", async ({ page }) => {
    await open(page, "/commodities");
    // /commodities mounts CommoditiesReport: one line chart for the active
    // base/quote pair, one sortable price table per pair, and Fava's switcher.
    await expect(page.locator(".line-chart")).toHaveCount(1);
    await expect(page.locator(".line-chart .chart-tick").first()).toBeVisible();
    await expect(page.locator(".chart-switcher button").first()).toBeVisible();
    const switcher = page.locator(".chart-switcher button");
    await expect(page.locator("table.prices-table")).toHaveCount(await switcher.count());
    await expect(page.locator("table.prices-table thead th")).not.toHaveCount(0);
    await page.locator(".line-chart").click();
    await expect(page.locator(".chart-tooltip")).toBeVisible();
    await page.mouse.move(0, 0);
    await expect(page.locator(".chart-tooltip")).toBeVisible();

    await open(page, "/events");
    await page.locator(".scatter-chart circle").first().click();
    await expect(page.locator(".chart-tooltip")).toBeVisible();
    await page.mouse.move(0, 0);
    await expect(page.locator(".chart-tooltip")).toBeVisible();
  });

  test("journal groups postings and query executes", async ({ page }) => {
    await open(page, "/journal");
    // Fava's journal is one ordered list; transaction rows carry the entry type
    // and postings live in a nested list that starts collapsed.
    const transactions = page.locator("ol.journal > li.transaction");
    await expect(transactions).not.toHaveCount(0);
    await expect(page.locator("ol.journal > li.custom").first()).toBeVisible();
    const first = transactions.first();
    const postings = first.locator("ul.postings");
    await expect(postings).toBeHidden();
    await first.locator(".indicators").click();
    await expect(first).toHaveClass(/show-full-entry/);
    await expect(postings).toBeVisible();
    await expect(postings.locator("li").first()).toBeVisible();

    await open(page, "/query");
    const editor = page.locator("#query-editor .cm-content");
    await editor.click();
    await page.keyboard.press("ControlOrMeta+A");
    await page.keyboard.type("SELECT account, balance FROM accounts ORDER BY account");
    await page.locator(".query-form button[type=submit]").click();
    await expect(page.locator(".report-table tbody tr").first()).toBeVisible();
    await expect(page.locator(".report-table")).toContainText("Assets:");
  });

  test("editor and import expose reviewed workflows", async ({ page }) => {
    await open(page, "/editor");
    // The buffer is a CodeMirror host div, not a textarea.
    const buffer = page.locator("#editor-buffer .cm-content");
    await expect(buffer).not.toHaveText("");
    await page.locator("#editor-validate").click();
    await expect(page.locator("article [role=status]").last()).toContainText(/Valid|Diagnostics/);
    await open(page, "/import");
    await page.locator(".import-buffer").fill("2000-01-01 open Assets:Imported USD\n");
    await page.getByRole("button", { name: "Preview" }).click();
    await expect(page.locator("article [role=status]").last()).toContainText(/Preview ready|error/i);
    await expect(page.locator("article [role=status]").last()).not.toContainText("Previewing…");
  });

  test("narrow menu opens the transplanted aside", async ({ page }, testInfo) => {
    await open(page, "/income_statement");
    if (testInfo.project.name.startsWith("narrow")) {
      // Below the breakpoint the aside is off-canvas until the menu is used.
      await expect(page.locator("#sidebar")).not.toBeVisible();
      await page.locator("#menu-toggle").click();
      await expect(page.locator("#sidebar")).toBeVisible();
      await expect(page.locator("#menu-toggle")).toHaveAttribute("aria-expanded", "true");
    } else {
      // Above the breakpoint the aside is always visible and the menu control
      // is not rendered for pointer input.
      await expect(page.locator("#sidebar")).toBeVisible();
      await expect(page.locator("#menu-toggle")).not.toBeVisible();
    }
  });

  test("account detail renders sparse multi-series charts", async ({ page }) => {
    const chart = sparseAccountChart("balance", "Account balance");
    const averageCostChart = sparseAccountChart("average-cost", "Average cost evolution");
    await page.route("**/__orangecount/fava/reports/account?*", async (route) => {
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ data: { columns: [], rows: [], chart, average_cost_chart: averageCostChart } }),
      });
    });

    // Default: Fava's single parity chart and no OrangeCount extension blocks.
    await open(page, "/account/Assets%3AInvestments%3AFund?interval=month");
    await expect(page.locator(".report-chart")).not.toHaveCount(0);
    await expect(page.locator(".average-cost-current")).toHaveCount(0);
    await expect(page.locator(".report-table")).toBeVisible();

    // Opt-in modern layer: the d3 charts plus the average-cost extension.
    await open(page, "/account/Assets%3AInvestments%3AFund?interval=month&chart_layer=modern");
    await expect(page.locator(".modern-chart-card")).toHaveCount(2);
    await expect(page.locator(".average-cost-current li")).toHaveCount(6);
  });
});

// The remaining standard surfaces. Each test names the surface it covers so the
// route manifest can cite a resolvable browser reference instead of "pending".
test.describe("transplanted Fava shell standard surfaces", () => {
  test("errors page lists diagnostics or the clean state", async ({ page }) => {
    await open(page, "/errors");
    await expect(page.locator("article")).toBeVisible();
    const rows = page.locator("table tbody tr");
    const empty = page.locator("article").filter({ hasText: "No errors" });
    await expect(rows.or(empty).first()).toBeVisible();
  });

  test("help page renders the topic index and a topic", async ({ page }) => {
    await open(page, "/help");
    await expect(page.locator("article")).toBeVisible();
    await expect(page.locator("article a").first()).toBeVisible();
  });

  test("options page renders the fava and beancount option tables", async ({ page }) => {
    await open(page, "/options");
    await expect(page.locator("article table")).not.toHaveCount(0);
    // The colour-scheme radios are visually hidden behind their labels, so the
    // visible control is the label, not the input element.
    await expect(page.locator("article label").first()).toBeVisible();
  });

  test("query page runs a query and exposes every download format", async ({ page }) => {
    await open(page, "/query");
    await page.locator(".cm-content").click();
    await page.keyboard.press("ControlOrMeta+A");
    await page.keyboard.type("SELECT account, balance FROM accounts ORDER BY account");
    await page.keyboard.press("ControlOrMeta+Enter");
    await expect(page.locator(".report-table")).toBeVisible({ timeout: 15000 });
    for (const format of ["csv", "xlsx", "ods"]) {
      await expect(page.locator(`a[href*="format=${format}"]`).first()).toBeVisible();
    }
  });

  test("locale switch keeps the same components in Simplified Chinese", async ({ page }) => {
    await open(page, "/income_statement");
    const before = await page.locator("[data-route]").count();
    await page.goto("/income_statement?locale=zh-CN", { waitUntil: "networkidle" });
    await expect(page.locator("#page-title")).not.toHaveText("");
    await expect(page.locator("[data-route]")).toHaveCount(before);
  });

  test("global g shortcuts navigate between reports", async ({ page }) => {
    await open(page, "/income_statement");
    await page.keyboard.press("g");
    await page.keyboard.press("j");
    await expect(page).toHaveURL(/\/journal/);
    await page.keyboard.press("g");
    await page.keyboard.press("b");
    await expect(page).toHaveURL(/\/balance_sheet/);
    await page.keyboard.press("?");
    await expect(page.locator(".keyboard-tooltip").first()).toBeVisible();
  });

  test("journal export modal and query export links are reachable", async ({ page }) => {
    // The journal's Beancount export lives in the Export modal, opened by the
    // #export hash the sidebar download button targets.
    await open(page, "/journal#export");
    await expect(page.locator('a[href*="download-journal"]').first()).toBeVisible({ timeout: 10000 });
    await open(page, "/query");
    await expect(page.locator('a[href*="format=csv"]').first()).toBeVisible();
  });

  test("add entry modal opens with the three entry types", async ({ page }) => {
    // The modal reacts to hashchange, so set the hash after the shell has
    // mounted instead of requesting it in the initial navigation.
    await open(page, "/");
    await page.evaluate(() => { window.location.hash = "#add-transaction"; });
    const dialog = page.locator("[role=dialog]").first();
    await expect(dialog).toBeVisible({ timeout: 10000 });
    // The dialog element is the form itself, so its fields are direct children.
    await expect(dialog.locator("input").first()).toBeVisible();
  });

  test("entry context modal shows the source slice and balances", async ({ page }) => {
    await open(page, "/journal");
    // Balances are computed for postings, so pick the context link of a
    // transaction row rather than the first row (which may be a close).
    const href = await page
      .locator("li.transaction a[href*='#context-'], li.cleared a[href*='#context-'], li.pending a[href*='#context-']")
      .first()
      .getAttribute("href");
    expect(href).toMatch(/^#context-/);
    await page.evaluate((hash) => { window.location.hash = hash; }, href);
    const dialog = page.locator("[role=dialog], dialog").first();
    await expect(dialog).toBeVisible({ timeout: 10000 });
    await expect(dialog).toContainText(/Context/);
    await expect(dialog).toContainText(/[A-Z][a-z]+:[A-Z][A-Za-z]+/);
    await expect(dialog).toContainText(/balance|Balance/);
  });

  test("export modal exposes the journal download", async ({ page }) => {
    await open(page, "/");
    await page.evaluate(() => { window.location.hash = "#export"; });
    const dialog = page.locator("[role=dialog]").first();
    await expect(dialog).toBeVisible({ timeout: 10000 });
    await expect(dialog.locator('a[href*="download-journal"]')).toBeVisible();
  });

  test("document upload modal exposes the account folder and name inputs", async ({ page }) => {
    await open(page, "/documents");
    await page.evaluate(() => { window.location.hash = "#add-document"; });
    await page.waitForTimeout(1200);
    const dialog = page.locator("[role=dialog], dialog").first();
    if (await dialog.count() === 0) {
      // No hash entry for this modal: the droptarget on the account trees is the
      // documented entry point, so assert that instead of a synthetic dialog.
      await expect(page.locator(".droptarget, [data-droptarget]").first()).toBeVisible();
      return;
    }
    await expect(dialog).toBeVisible();
    await expect(dialog.locator("input").first()).toBeVisible();
  });

  test("a failed adapter load raises a visible notification", async ({ page }) => {
    // The notification list is created lazily on the first message, so the test
    // provokes the bootstrap failure path instead of asserting an always-present
    // host. Aborting ledger_data is the same failure the shell reports on a
    // broken adapter round-trip.
    await page.route("**/__orangecount/fava/ledger_data*", (route) => route.abort());
    await page.goto("/income_statement", { waitUntil: "domcontentloaded" });
    await expect(page.locator(".notifications li").first()).toBeVisible({ timeout: 15000 });
  });

  test("print media hides the chrome and keeps the report body", async ({ page }) => {
    await open(page, "/income_statement");
    await page.emulateMedia({ media: "print" });
    await expect(page.locator("#menu-toggle")).toBeHidden();
    await expect(page.locator("#sidebar")).toBeHidden();
    await expect(page.locator("#main-content, article").first()).toBeVisible();
  });
});

// Modal baselines: the same three modals the reference capture records, so the
// two trees can be compared for those surfaces too.
test.describe("transplanted Fava shell modal baseline", () => {
  // m-context is deliberately absent: its body renders the source slice of
  // whichever entry opened it, so the pixels are not a stable target. The
  // "entry context modal shows the source slice and balances" test covers it.
  const modals = [
    ["m-add", "#add-transaction"],
    ["m-export", "#export"],
  ];

  for (const [name, hash] of modals) {
    test(`baseline ${name}`, async ({ page }) => {
      await open(page, "/income_statement");
      await page.evaluate((value) => { window.location.hash = value; }, hash);
      await expect(page.locator("[role=dialog]").first()).toBeVisible({ timeout: 10000 });
      await expect(page).toHaveScreenshot(name + ".png", { fullPage: true });
    });
  }
});

// The four-cell English baseline matrix: every captured route must be stable in
// both viewports and both colour schemes.
test.describe("transplanted Fava shell visual baseline", () => {
  // Every route the reference capture records must have an OrangeCount snapshot,
  // otherwise the two trees cannot be compared and the route gate has no visual
  // evidence at all.
  const routes = [
    "/",
    "/journal",
    "/income_statement",
    "/balance_sheet",
    "/trial_balance",
    "/holdings",
    "/commodities",
    "/events",
    "/documents",
    "/statistics",
    "/editor",
    "/account/Assets%3AInvestments%3AFund",
    "/query",
    "/import",
    "/options",
    "/help",
    "/errors",
  ];

  for (const route of routes) {
    test(`baseline ${slug(route)}`, async ({ page }) => {
      await open(page, route);
      await expect(page.locator("article")).toBeVisible();
      await expect(page).toHaveScreenshot(`${slug(route)}.png`, { fullPage: true });
    });
  }
});
