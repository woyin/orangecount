import { PNG } from "pngjs";
import { readFileSync, readdirSync, existsSync } from "node:fs";
import path from "node:path";

const root = "/Users/breestealth/Development/OrangeCount";
const cells = ["desktop-light", "desktop-dark", "narrow-light", "narrow-dark"];
const baseDir = path.join(root, "testdata/visual-baselines");
const ocDir = path.join(root, "testdata/visual-regression");

function load(file) {
  return PNG.sync.read(readFileSync(file));
}

// Comparing only when the sizes match would report a 19px height difference as a
// total mismatch, which hides whether the pages are otherwise close. Compare the
// overlapping region and report the size delta separately.
function diffRatio(a, b) {
  if (a.width !== b.width) {
    return { ratio: 1, reason: "width " + a.width + " vs " + b.width, sizeDelta: 1 };
  }
  const h = Math.min(a.height, b.height);
  let diff = 0;
  const total = a.width * h;
  for (let y = 0; y < h; y++) {
    for (let x = 0; x < a.width; x++) {
      const ia = (a.width * y + x) * 4;
      const ib = (b.width * y + x) * 4;
      if (Math.abs(a.data[ia] - b.data[ib]) > 8 ||
          Math.abs(a.data[ia+1] - b.data[ib+1]) > 8 ||
          Math.abs(a.data[ia+2] - b.data[ib+2]) > 8) {
        diff++;
      }
    }
  }
  const taller = Math.max(a.height, b.height);
  const sizeDelta = (taller - h) / taller;
  return {
    ratio: diff / total,
    sizeDelta,
    reason: sizeDelta > 0 ? "height " + a.height + " vs " + b.height + " (overlap compared)" : null,
  };
}

const results = [];
for (const cell of cells) {
  const favaPath = path.join(baseDir, cell, "visual", "fava-reference.spec.mjs");
  const ocPath = path.join(ocDir, cell, "visual", "fava-transplanted.spec.mjs");
  if (!existsSync(favaPath) || !existsSync(ocPath)) continue;
  for (const name of readdirSync(favaPath).filter((f) => f.endsWith(".png")).sort()) {
    const ocFile = path.join(ocPath, name);
    if (!existsSync(ocFile)) {
      results.push({ cell, name, ratio: null, reason: "no OrangeCount snapshot" });
      continue;
    }
    const d = diffRatio(load(path.join(favaPath, name)), load(ocFile));
    results.push({ cell, name, ratio: d.ratio, sizeDelta: d.sizeDelta, reason: d.reason });
  }
}

results.sort((a, b) => (b.ratio ?? 2) - (a.ratio ?? 2));
const lines = results.map((r) =>
  (r.ratio === null ? "  n/a  " : (r.ratio * 100).toFixed(1).padStart(6) + "%") +
  "  " + r.cell.padEnd(14) + "  " + r.name + (r.reason ? "  (" + r.reason + ")" : ""));
console.log(lines.join("\n"));
const ratios = results.filter((r) => r.ratio !== null).map((r) => r.ratio).sort((a, b) => a - b);
const below1 = ratios.filter((r) => r < 0.01).length;
const below5 = ratios.filter((r) => r < 0.05).length;
const sizeMismatch = results.filter((r) => (r.sizeDelta ?? 0) > 0.01).length;
console.log("\nsummary: " + ratios.length + " compared, " + below1 + " under 1% pixel diff, " +
  below5 + " under 5%, median " + (ratios[Math.floor(ratios.length / 2)] * 100).toFixed(1) + "%, " +
  sizeMismatch + " with a height difference");
