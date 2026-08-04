import { build } from "esbuild";
import svelte from "esbuild-svelte";
import { mkdir, rm, writeFile } from "node:fs/promises";
import path from "node:path";

const webRoot = import.meta.dirname;
const staging = path.join(webRoot, "staging", "fava-shell");

const indexHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>OrangeCount</title>
    <link rel="stylesheet" href="./app.css">
    <style>#app { display: contents; }</style>
  </head>
  <body>
    <div id="app"></div>
    <script type="module" src="./app.js"></script>
  </body>
</html>
`;

await rm(staging, { recursive: true, force: true });
await mkdir(staging, { recursive: true });
await build({
  entryPoints: [path.join(webRoot, "src", "fava", "main.ts")],
  bundle: true,
  format: "esm",
  target: "es2022",
  outfile: path.join(staging, "app.js"),
  metafile: false,
  sourcemap: false,
  minify: false,
  legalComments: "eof",
  loader: { ".woff": "dataurl", ".woff2": "dataurl" },
  plugins: [svelte({ compilerOptions: { dev: false } })],
});
await writeFile(path.join(staging, "index.html"), indexHTML);
await writeFile(path.join(staging, "manifest.json"), `${JSON.stringify({
  name: "orangecount-fava-shell",
  artifact: "development-only",
  adapter: "private-orange-count-fava-shaped",
  entry: "index.html",
}, null, 2)}\n`);
console.log(`built sanitized Fava shell into web/staging/fava-shell`);
