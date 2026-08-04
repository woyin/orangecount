import os from "node:os";
import path from "node:path";
import { defineConfig } from "@playwright/test";

const nodeMajor = Number(process.versions.node.split(".")[0]);
if (nodeMajor === 24) {
  const message = "OrangeCount visual harness blocked: Node 24 with @playwright/test 1.52.0 hangs before test discovery in this environment. Use Node 22 LTS (without changing the system Node) and rerun npm --prefix web run visual:test.";
  console.error(message);
  throw new Error(message);
}

const repoRoot = path.resolve(import.meta.dirname, "..");
const temporaryRun = `orangecount-fava-visual-${process.pid}`;

export default defineConfig({
  testDir: path.join(repoRoot, "web", "tests"),
  outputDir: path.join(os.tmpdir(), temporaryRun, "test-results"),
  snapshotPathTemplate: path.join(
    repoRoot,
    "testdata",
    "visual-baselines",
    "{projectName}",
    "{testFilePath}",
    "{arg}{ext}",
  ),
  fullyParallel: false,
  workers: 1,
  retries: 0,
  timeout: 15_000,
  expect: {
    timeout: 5_000,
    toHaveScreenshot: {
      animations: "disabled",
      caret: "hide",
      scale: "css",
    },
  },
  reporter: [["list"]],
  projects: [
    {
      name: "desktop",
      use: {
        baseURL: process.env.ORANGECOUNT_BASE_URL,
        viewport: { width: 1280, height: 800 },
        deviceScaleFactor: 1,
      },
    },
    {
      name: "narrow",
      use: {
        baseURL: process.env.ORANGECOUNT_BASE_URL,
        viewport: { width: 520, height: 800 },
        deviceScaleFactor: 1,
      },
    },
  ],
  use: {
    locale: "en-US",
    timezoneId: "UTC",
    colorScheme: "dark",
    reducedMotion: "reduce",
    serviceWorkers: "block",
    trace: "off",
    video: "off",
    screenshot: "off",
    launchOptions: {
      args: ["--font-render-hinting=none"],
    },
  },
});
