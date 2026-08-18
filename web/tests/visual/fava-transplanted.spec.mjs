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

test.describe("transplanted Fava shell smoke", () => {
  test("root and core reports load real adapter data", async ({ page }) => {
    for (const route of ["/", "/income_statement", "/balance_sheet", "/trial_balance"]) {
      await open(page, route);
      await expect(page.locator("[data-tree-table]")).not.toHaveCount(0);
      await expect(page.locator("table")).not.toHaveCount(0);
    }
  });

  test("data-backed utility reports render their adapter rows", async ({ page }) => {
    for (const route of ["/holdings", "/commodities", "/documents", "/events", "/statistics"]) {
      await open(page, route);
      await expect(page.locator("table tbody tr").first()).toBeVisible();
    }
  });

  test("commodity and event chart tooltips stay visible after a click", async ({ page }) => {
    await open(page, "/commodities");
    await expect(page.locator(".line-chart .chart-tick")).toHaveCount(6);
    await page.locator(".line-chart").click();
    await expect(page.locator(".chart-tooltip")).toBeVisible();
    await page.mouse.move(0, 0);
    await expect(page.locator(".chart-tooltip")).toBeVisible();

    await open(page, "/events");
    await page.locator(".scatter-chart circle").click();
    await expect(page.locator(".chart-tooltip")).toBeVisible();
    await page.mouse.move(0, 0);
    await expect(page.locator(".chart-tooltip")).toBeVisible();
  });

  test("journal groups postings and query executes", async ({ page }) => {
    await open(page, "/journal");
    await expect(page.locator(".journal-transaction-row")).not.toHaveCount(0);
    const toggle = page.locator(".journal-transaction-toggle").first();
    await expect(toggle).toHaveAttribute("aria-expanded", "false");
    await toggle.click();
    await expect(toggle).toHaveAttribute("aria-expanded", "true");
    await open(page, "/query");
    await page.locator("#query-editor").fill("SELECT account, balance FROM accounts ORDER BY account");
    await page.locator(".query-form button[type=submit]").click();
    await expect(page.locator(".report-table")).toContainText("Assets:");
  });

  test("editor and import expose reviewed workflows", async ({ page }) => {
    await open(page, "/editor");
    await expect(page.locator("#editor-buffer")).not.toHaveValue("");
    await page.locator("#editor-validate").click();
    await expect(page.locator("[role=status]")).toContainText(/Valid|Diagnostics/);
    await open(page, "/import");
    await page.locator(".import-buffer").fill("2000-01-01 open Assets:Imported USD\n");
    await page.locator("button", { hasText: "Preview" }).click();
    await expect(page.locator("[role=status]")).not.toHaveText("Previewing…");
  });

  test("narrow menu opens the transplanted aside", async ({ page }) => {
    await open(page, "/income_statement");
    await expect(page.locator("#sidebar")).not.toBeVisible();
    await page.locator("#menu-toggle").click();
    await expect(page.locator("#sidebar")).toBeVisible();
    await expect(page.locator("#menu-toggle")).toHaveAttribute("aria-expanded", "true");
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

    await open(page, "/account/Assets%3AInvestments%3AFund?interval=month");
    await expect(page.locator(".modern-chart-card")).toHaveCount(2);
    await expect(page.locator(".average-cost-current li")).toHaveCount(6);
    await expect(page.locator(".report-table")).toBeVisible();
  });
});
