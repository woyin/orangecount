import { test, expect } from "./fixtures.mjs";

const referenceURL = process.env.FAVA_BASE_URL || "";
const safeReference = referenceURL && !/:5000(?:\/|$)/.test(referenceURL) && !/financebook|private|profile/i.test(referenceURL);

test.describe("optional isolated Fava 1.30.12 reference", () => {
  test.skip(!safeReference, "reference capture requires the newly started isolated OCI Fava process");
  test.use({ baseURL: referenceURL });

  // The route set and slugs mirror fava-transplanted.spec.mjs so a reference
  // capture and the OrangeCount snapshot for the same route land on the same
  // file name. Without that, the two trees could never be compared.
  for (const [name, route] of [
    ["root", ""],
    ["journal", "journal"],
    ["income-statement", "income_statement"],
    ["balance-sheet", "balance_sheet"],
    ["trial-balance", "trial_balance"],
    ["holdings", "holdings"],
    ["commodities", "commodities"],
    ["events", "events"],
    ["documents", "documents"],
    ["statistics", "statistics"],
    ["editor", "editor"],
    ["account-assets-3ainvestments-3afund", "account/Assets:Investments:Fund"],
    ["query", "query"],
    ["import", "import"],
    ["options", "options"],
    ["help", "help"],
    ["errors", "errors"],
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
      const target = route === "" ? ledgerPrefix : `${ledgerPrefix}${route}/`;
      const response = await page.goto(target, { waitUntil: "networkidle" });
      expect(response?.status()).toBe(200);
      await expect(page.locator("body")).toBeVisible();
      await expect(page.locator("body")).not.toContainText("Not Found");
      await expect(page).toHaveScreenshot(`${name}.png`, { fullPage: true });
    });
  }
});
