import { test, expect } from "./fixtures.mjs";

const referenceURL = process.env.FAVA_BASE_URL || "";
const safeReference = referenceURL && !/:5000(?:\/|$)/.test(referenceURL) && !/financebook|private|profile/i.test(referenceURL);

test.describe("optional isolated Fava 1.30.12 reference", () => {
  test.skip(!safeReference, "reference capture requires the newly started isolated OCI Fava process");
  test.use({ baseURL: referenceURL });

  for (const [name, route] of [
    ["shell-journal", "/journal/"],
    ["income-statement", "/income_statement/"],
    ["balance-sheet", "/balance_sheet/"],
    ["trial-balance", "/trial_balance/"],
    ["account-detail", "/account/Assets%3AWallet%3APrimary/"],
  ]) {
    test(`captures sanitized ${name} reference`, async ({ page }) => {
      await page.goto(route, { waitUntil: "networkidle" });
      await expect(page.locator("body")).toBeVisible();
      await expect(page).toHaveScreenshot(`${name}.png`, { fullPage: true });
    });
  }
});
