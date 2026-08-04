import { createHash } from "node:crypto";
import { readFile, readdir } from "node:fs/promises";
import { execFileSync } from "node:child_process";
import path from "node:path";

const [output, imageId, fixtureRoot] = process.argv.slice(2);
async function files(dir, prefix = "") {
  const result = [];
  for (const entry of (await readdir(dir, { withFileTypes: true })).sort((a, b) => a.name.localeCompare(b.name))) {
    const relative = path.posix.join(prefix, entry.name);
    if (entry.isDirectory()) result.push(...await files(path.join(dir, entry.name), relative));
    else result.push([relative, await readFile(path.join(dir, entry.name))]);
  }
  return result;
}
const fixtureHash = createHash("sha256");
for (const [name, data] of await files(fixtureRoot)) {
  fixtureHash.update(name); fixtureHash.update("\0"); fixtureHash.update(data); fixtureHash.update("\0");
}
const command = (name, args) => { try { return execFileSync(name, args, { encoding: "utf8" }).trim(); } catch { return "unavailable"; } };
const lock = {
  schema_version: 1,
  image_id: imageId,
  fava: { version: "1.30.12", commit: "aa7538e8971252c9efc52c8a516a3a77d604553f" },
  python: command("python", ["--version"]),
  beancount: command("python", ["-c", "import beancount; print(beancount.__version__)"]),
  bison: command("bison", ["--version"]).split("\n")[0],
  node: command("node", ["--version"]),
  npm: command("npm", ["--version"]),
  playwright: command("node", ["-e", "console.log(require('/app/web/node_modules/@playwright/test/package.json').version)"]),
  chromium: command(process.env.CHROMIUM_EXECUTABLE_PATH || "chromium", ["--version"]),
  fonts: { fira_sans: "Fava-pinned @fontsource/fira-sans", fira_mono: "Fava-pinned @fontsource/fira-mono", source_code_pro: "Fava-pinned @fontsource/source-code-pro", cjk_fallback: "Debian package fonts-noto-cjk" },
  locale: "en-US", timezone: "UTC", reduced_motion: "reduce", device_scale_factor: 1,
  viewports: { desktop: [1280, 800], narrow: [520, 800] },
  fixture_content_sha256: fixtureHash.digest("hex"),
  capture: { candidate_only: true, formal_baseline_directory: "testdata/visual-baselines", output_directory: "testdata/visual-candidates/fava-reference" }
};
await (await import("node:fs/promises")).writeFile(output, `${JSON.stringify(lock, null, 2)}\n`);
