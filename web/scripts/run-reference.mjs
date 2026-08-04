import { spawn } from "node:child_process";
import { mkdir, realpath, stat, unlink, writeFile } from "node:fs/promises";
import path from "node:path";

const repoRoot = path.resolve(import.meta.dirname, "../..");
const image = process.env.FAVA_REFERENCE_IMAGE || "orangecount-fava-reference:1.30.12";
const fixtureRoot = path.join(repoRoot, "testdata", "fixtures");
const fixture = path.resolve(repoRoot, process.env.FAVA_REFERENCE_FIXTURE || "testdata/fixtures/fava-reference");
const output = path.join(repoRoot, "testdata", "visual-candidates", "fava-reference");
const relativeFixture = path.relative(fixtureRoot, fixture);
if (!relativeFixture || relativeFixture.startsWith("..") || path.isAbsolute(relativeFixture)) {
  throw new Error("FAVA_REFERENCE_FIXTURE must select a directory below testdata/fixtures");
}
const fixtureReal = await realpath(fixture);
const fixtureRootReal = await realpath(fixtureRoot);
const realRelativeFixture = path.relative(fixtureRootReal, fixtureReal);
if (!realRelativeFixture || realRelativeFixture.startsWith("..") || path.isAbsolute(realRelativeFixture)) {
  throw new Error("FAVA_REFERENCE_FIXTURE must resolve below testdata/fixtures");
}
if (!(await stat(fixtureReal)).isDirectory()) {
  throw new Error(`Fava reference fixture is not a directory: ${fixture}`);
}
await mkdir(output, { recursive: true });
await unlink(path.join(output, "capture-failure.txt")).catch(() => {});

function run(command, args) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, { cwd: repoRoot, stdio: "inherit" });
    child.once("error", reject); child.once("exit", (code, signal) => resolve(code ?? (signal ? 1 : 0)));
  });
}
const build = await run("docker", ["build", "--tag", image, "--file", "tools/fava-reference/Dockerfile", "."]);
if (build !== 0) {
  await writeFile(path.join(output, "capture-failure.txt"), [
    "Fava reference capture did not start.",
    `command: docker build --tag ${image} --file tools/fava-reference/Dockerfile .`,
    `exit_code: ${build}`,
    "The command is intentionally fail-closed; no private service, ledger, or browser profile is inspected.",
    "Retry after Docker daemon access is restored.",
    "",
  ].join("\n"));
  process.exit(build);
}
const inspect = await new Promise((resolve) => {
  const child = spawn("docker", ["image", "inspect", "--format", "{{.Id}}", image], { cwd: repoRoot });
  let stdout = ""; child.stdout.on("data", (chunk) => { stdout += chunk; });
  child.once("exit", (code) => resolve(code === 0 ? stdout.trim() : "unknown"));
});
const args = ["run", "--rm", "--init", "--network", "none", "--read-only", "--tmpfs", "/tmp:rw,noexec,nosuid,size=256m", "--env", `ORANGECOUNT_REFERENCE_IMAGE_ID=${inspect}`, "--volume", `${fixtureReal}:/fixtures:ro`, "--volume", `${output}:/output:rw`, image, ...process.argv.slice(2)];
process.exitCode = await run("docker", args);
