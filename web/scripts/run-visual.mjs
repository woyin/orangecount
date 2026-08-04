import { spawn } from "node:child_process";
import { once } from "node:events";
import path from "node:path";

const nodeMajor = Number(process.versions.node.split(".")[0]);
if (nodeMajor === 24) {
  console.error("OrangeCount visual harness blocked: Node 24 with @playwright/test 1.52.0 hangs before test discovery in this environment. Use Node 22 LTS (without changing the system Node) and rerun npm --prefix web run visual:test.");
  process.exit(78);
}

const repoRoot = path.resolve(import.meta.dirname, "../..");
const webRoot = path.join(repoRoot, "web");
const fixtureRoot = path.join(repoRoot, "testdata", "fixtures", "fava-visual");
const entry = path.join(fixtureRoot, "main.bean");
const documents = path.join(fixtureRoot, "documents");
const runName = `orangecount-fava-visual-${process.pid}`;

const binary = process.env.ORANGECOUNT_BIN;
const command = binary || "go";
const args = binary
  ? ["serve", "--addr", "127.0.0.1:0", "--document-root", documents, entry]
  : ["run", "./cmd/orangecount", "serve", "--addr", "127.0.0.1:0", "--document-root", documents, entry];

const server = spawn(command, args, {
  cwd: repoRoot,
  env: { ...process.env, ORANGECOUNT_VISUAL_RUN: runName },
  detached: true,
  stdio: ["ignore", "pipe", "pipe"],
});

let stderr = "";
server.stderr.setEncoding("utf8");
server.stderr.on("data", (chunk) => { stderr += chunk; });

const baseURL = await new Promise((resolve, reject) => {
  let stdout = "";
  const onData = (chunk) => {
    stdout += chunk;
    const match = stdout.match(/serving on (http:\/\/127\.0\.0\.1:\d+)/);
    if (match) {
      server.stdout.off("data", onData);
      resolve(match[1]);
    }
  };
  server.stdout.setEncoding("utf8");
  server.stdout.on("data", onData);
  server.once("error", reject);
  server.once("exit", (code) => {
    reject(new Error(`OrangeCount visual server exited before readiness (${code}): ${stderr.replace(/\s+/g, " ").trim()}`));
  });
});

const cli = path.join(webRoot, "node_modules", "@playwright", "test", "cli.js");
const playwright = spawn(process.execPath, [cli, "test", "--config", path.join(webRoot, "playwright.config.mjs"), ...process.argv.slice(2)], {
  cwd: repoRoot,
  env: { ...process.env, ORANGECOUNT_BASE_URL: baseURL, ORANGECOUNT_VISUAL_RUN: runName },
  stdio: "inherit",
});

const stop = () => {
  if (server.exitCode === null) {
    try { process.kill(-server.pid, "SIGTERM"); } catch (_) { server.kill("SIGTERM"); }
  }
  if (playwright.exitCode === null) {
    playwright.kill("SIGTERM");
  }
};
process.once("SIGINT", stop);
process.once("SIGTERM", stop);

const [result] = await once(playwright, "exit");
stop();
process.exitCode = result ?? 1;
