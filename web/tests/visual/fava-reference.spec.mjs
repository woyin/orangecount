import { test, expect } from "./fixtures.mjs";

const referenceURL = process.env.FAVA_BASE_URL || "";
const safeReference = referenceURL && !/:5000(?:\/|$)/.test(referenceURL) && !/financebook|private|profile/i.test(referenceURL);

test.describe("optional isolated Fava 1.30.12 reference", () => {
  test.skip(!safeReference, "reference capture requires the newly started isolated OCI Fava process");
  test.use({ baseURL: referenceURL });

  for (const [name, route] of [
    ["shell-journal", "journal"],
    ["income-statement", "income_statement"],
    ["balance-sheet", "balance_sheet"],
    ["trial-balance", "trial_balance"],
    ["account-detail", "account/Assets:Cash:Wallet01"],
  ]) {
    test(`captures sanitized ${name} reference`, async ({ page }) => {
      // Fava serves the SPA below a ledger-specific slug. Start at `/` so
      // the server supplies that slug, then navigate to the client route.
      await page.goto("/", { waitUntil: "networkidle" });
      const redirectedPath = new URL(page.url()).pathname;
      const marker = "/income_statement/";
      const markerIndex = redirectedPath.indexOf(marker);
      if (markerIndex < 1) throw new Error(`Fava did not provide a ledger route prefix: ${redirectedPath}`);
      const ledgerPrefix = redirectedPath.slice(0, markerIndex + 1);
      const response = await page.goto(`${ledgerPrefix}${route}/`, { waitUntil: "networkidle" });
      expect(response?.status()).toBe(200);
      await expect(page.locator("body")).toBeVisible();
      await expect(page.locator("body")).not.toContainText("Not Found");
      await expect(page).toHaveScreenshot(`${name}.png`, { fullPage: true });
    });
  }
});
