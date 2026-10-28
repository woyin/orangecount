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
const isReference = Boolean(process.env.FAVA_BASE_URL);
// Two different authorities, two different trees. The Fava reference capture is
// the visual authority a route gate compares against; OrangeCount's own
// snapshots only detect regressions and must never be mistaken for that
// authority. Keeping them apart is what lets each be reviewed on its own terms.
const snapshotRoot =
  process.env.VISUAL_SNAPSHOT_DIR ||
  path.join(repoRoot, "testdata", isReference ? "visual-baselines" : "visual-regression");
const baseURL = isReference ? process.env.FAVA_BASE_URL : process.env.ORANGECOUNT_BASE_URL;
const viewports = [
  ["desktop", { width: 1280, height: 800 }],
  ["narrow", { width: 520, height: 800 }],
];
// The four-cell English matrix is the only approved baseline shape: desktop and
// narrow viewports, each in the light and dark colour scheme.
const projects = viewports.flatMap(([size, viewport]) =>
  ["light", "dark"].map((colorScheme) => ({
    name: `${size}-${colorScheme}`,
    use: { baseURL, viewport, deviceScaleFactor: 1, colorScheme },
  })),
);

export default defineConfig({
  testDir: path.join(repoRoot, "web", "tests"),
  // One spec per mode: the reference capture only ever records Fava, and the
  // local mode only records the transplanted shell, so `visual:update` cannot
  // mix the two authorities into one baseline tree.
  testMatch: isReference ? ["**/visual/fava-reference.spec.mjs"] : ["**/visual/fava-transplanted.spec.mjs"],
  outputDir: path.join(os.tmpdir(), temporaryRun, "test-results"),
  snapshotPathTemplate: path.join(
    snapshotRoot,
    "{projectName}",
    "{testFilePath}",
    "{arg}{ext}",
  ),
  fullyParallel: false,
  workers: 1,
  retries: 0,
  timeout: 20_000,
  expect: {
    timeout: 5_000,
    toHaveScreenshot: {
      animations: "disabled",
      caret: "hide",
      scale: "css",
    },
  },
  reporter: [["list"]],
  projects,
  use: {
    locale: "en-US",
    timezoneId: "UTC",
    reducedMotion: "reduce",
    serviceWorkers: "block",
    trace: "off",
    video: "off",
    screenshot: "off",
    launchOptions: {
      args: ["--font-render-hinting=none", "--no-sandbox"],
      ...(process.env.CHROMIUM_EXECUTABLE_PATH
        ? { executablePath: process.env.CHROMIUM_EXECUTABLE_PATH }
        : {}),
    },
  },
});
