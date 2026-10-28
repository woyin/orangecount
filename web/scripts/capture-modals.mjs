import { chromium } from "@playwright/test";
import path from "node:path";

// The modal capture writes into the reference baseline tree, so it needs the
// same isolation guard as the Playwright reference suite. Without it the script
// happily screenshots OrangeCount's own server and files the result as a Fava
// reference capture.
const base = process.env.FAVA_BASE_URL;
const out = process.env.MODAL_OUT;
const safeReference =
  Boolean(base) &&
  !/:5000(?:\/|$)/.test(base) &&
  !/financebook|private|profile/i.test(base);
if (!safeReference) {
  console.error(
    "capture-modals: refusing to capture. Set FAVA_BASE_URL to the isolated " +
      "Fava 1.30.12 reference server; port 5000 and any financebook/private/profile URL are rejected.",
  );
  process.exit(1);
}
if (!out) {
  console.error("capture-modals: MODAL_OUT is required.");
  process.exit(1);
}
const cells = [
  ["desktop-light", { width: 1280, height: 800 }, "light"],
  ["desktop-dark", { width: 1280, height: 800 }, "dark"],
  ["narrow-light", { width: 520, height: 800 }, "light"],
  ["narrow-dark", { width: 520, height: 800 }, "dark"],
];
const targets = [
  ["m-add", "income_statement/#add-transaction"],
  ["m-export", "income_statement/#export"],
  ["m-context", "journal/#context-"],
];

for (const [cell, viewport, scheme] of cells) {
  const browser = await chromium.launch();
  const context = await browser.newContext({
    viewport,
    colorScheme: scheme,
    reducedMotion: "reduce",
    locale: "en-US",
    timezoneId: "UTC",
  });
  const page = await context.newPage();
  await page.goto(base + "/", { waitUntil: "networkidle" });
  await page.waitForTimeout(700);
  const prefix = new URL(page.url()).pathname.replace(/income_statement\/?$/, "");
  for (const [name, route] of targets) {
    await page.goto(base + prefix + route, { waitUntil: "networkidle" });
    await page.waitForTimeout(1300);
    // The modal capture declares its own producer segment. Writing into
    // fava-reference.spec.mjs/ would make the baseline tree attribute these
    // files to the reference suite, which cannot be checked from the path alone.
    const target = path.join(out, cell, "visual", "capture-modals.mjs", name + ".png");
    await page.screenshot({ path: target, fullPage: true });
    console.log(cell, name, "captured");
  }
  await browser.close();
}
