import { test, expect } from "./fixtures.mjs";

async function openRoute(page, route) {
  await page.goto(route, { waitUntil: "networkidle" });
  await expect(page.locator("#page-title")).not.toHaveText("Overview");
  await expect(page.locator("#app")).not.toContainText("Loading");
}

test.describe("sanitized OrangeCount Fava visual baseline", () => {
  test("shell and responsive navigation", async ({ page }, testInfo) => {
    await openRoute(page, "/journal");
    await expect(page.locator("#page-title")).toHaveText("Journal");
    await expect(page.locator("#navigation button[aria-current='page']")).toHaveAttribute("data-view", "journal");
    await expect(page.locator("#global-time")).toHaveValue("all");
    await expect(page.locator("#locale")).toHaveValue("en");

    if (testInfo.project.name === "narrow") {
      await expect(page.locator("#sidebar")).not.toBeVisible();
      await page.locator("#menu-toggle").click();
      await expect(page.locator("#sidebar")).toBeVisible();
      await expect(page.locator("#menu-toggle")).toHaveAttribute("aria-expanded", "true");
      await page.locator("#menu-toggle").click();
    } else {
      await expect(page.locator("#sidebar")).toBeVisible();
    }

    await expect(page).toHaveScreenshot("shell-journal.png", { fullPage: true });
  });

  test("Journal keeps transaction groups and URL-backed filters", async ({ page }) => {
    await openRoute(page, "/journal");
    const groups = page.locator(".journal-transaction-row");
    const postings = page.locator(".journal-posting-row");
    await expect(groups).not.toHaveCount(0);
    await expect(postings).not.toHaveCount(0);

    const firstGroup = groups.first();
    const firstPosting = firstGroup.locator("xpath=following-sibling::tr[contains(@class, 'journal-posting-row')][1]");
    const firstToggle = firstGroup.locator(".journal-transaction-toggle");
    await expect(firstToggle).toHaveAttribute("aria-expanded", "true");
    await firstToggle.click();
    await expect(firstToggle).toHaveAttribute("aria-expanded", "false");
    await expect(firstPosting).toBeHidden();

    await page.locator("#journal-from").fill("2024-01-07");
    await page.locator("#journal-to").fill("2024-01-12");
    await page.locator("#journal-apply").click();
    await expect(page).toHaveURL(/[?&]from=2024-01-07/);
    await expect(page).toHaveURL(/[?&]to=2024-01-12/);
    await expect(page).toHaveScreenshot("journal-filtered.png", { fullPage: true });
  });

  test("standalone reports ignore stale global filters", async ({ page }) => {
    for (const route of ["/holdings", "/commodities", "/events", "/documents", "/statistics"]) {
      await openRoute(page, `${route}?account=Assets%3AWallet%3APrimary&filter=not-present&time=year`);
      await expect(page.locator("#report-result tbody tr").first()).toBeVisible();
    }
  });

  test("balance sheet and trial balance expose tree and hierarchy states", async ({ page }) => {
    for (const route of ["/balance_sheet", "/trial_balance"]) {
      await openRoute(page, route);
      await expect(page.locator("#report-result table")).toBeVisible();
      await expect(page.locator("#report-result [data-tree-table] tbody tr")).not.toHaveCount(0);
      await expect(page.locator("#report-result th")).not.toHaveCount(0);
      await expect(page.locator(".chart-card")).toBeVisible();
      await expect(page).toHaveScreenshot(`${route.slice(1)}.png`, { fullPage: true });
    }

    await page.locator("#chart-hierarchy-layout").selectOption("sunburst");
    await expect(page.locator(".report-hierarchy-chart")).toBeVisible();
  });

  test("account detail provides account-scoped journal and running balance", async ({ page }) => {
    await openRoute(page, "/account/Assets%3AWallet%3APrimary");
    await expect(page.locator(".account-detail-head h3")).toHaveText("Assets:Wallet:Primary");
    await expect(page.locator("#account-journal table")).toBeVisible();
    await expect(page.locator("#account-journal")).toContainText("Running balance");
    await expect(page).toHaveScreenshot("account-detail.png", { fullPage: true });
  });
});
