import { test, expect } from "./fixtures.mjs";

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
});
