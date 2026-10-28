import { chromium } from "@playwright/test";
import path from "node:path";

const base = process.env.FAVA_BASE_URL;
const out = process.env.MODAL_OUT;
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
    const target = path.join(out, cell, "visual", "fava-reference.spec.mjs", name + ".png");
    await page.screenshot({ path: target, fullPage: true });
    console.log(cell, name, "captured");
  }
  await browser.close();
}
