import { createHash } from "node:crypto";
import { readdir, readFile } from "node:fs/promises";
import { execFile } from "node:child_process";
import { promisify } from "node:util";
import path from "node:path";

const run = promisify(execFile);
const webRoot = import.meta.dirname.replace(/\/scripts$/, "");
const staging = path.join(webRoot, "staging", "fava-shell");
const build = path.join(webRoot, "build.mjs");

async function filesAndHashes() {
  const names = (await readdir(staging)).sort();
  const entries = [];
  for (const name of names) {
    const data = await readFile(path.join(staging, name));
    entries.push([name, createHash("sha256").update(data).digest("hex")]);
  }
  return entries;
}

await run(process.execPath, [build], { cwd: webRoot });
const first = await filesAndHashes();
await run(process.execPath, [build], { cwd: webRoot });
const second = await filesAndHashes();
if (JSON.stringify(first) !== JSON.stringify(second)) {
  console.error("frontend build is not deterministic");
  process.exitCode = 1;
} else {
  console.log(`deterministic frontend build: ${first.length} staging files`);
}
