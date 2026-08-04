import { test as base, expect } from "@playwright/test";

const motionReset = `
  *, *::before, *::after {
    animation: none !important;
    transition: none !important;
    caret-color: transparent !important;
  }
`;

export const test = base.extend({
  page: async ({ page, baseURL }, use) => {
    if (!baseURL) {
      throw new Error("A visual base URL is required; use visual:test or visual:reference");
    }
    const origin = new URL(baseURL).origin;
    page.on("request", (request) => {
      const url = new URL(request.url());
      if (url.origin !== origin && !["about:", "data:", "blob:"].includes(url.protocol)) {
        throw new Error(`unexpected network origin: ${url.origin}`);
      }
    });
    await page.emulateMedia({ reducedMotion: "reduce" });
    await page.addStyleTag({ content: motionReset });
    await use(page);
  },
});

export { expect };
