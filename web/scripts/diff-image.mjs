import { PNG } from "pngjs";
import { readFileSync, writeFileSync } from "node:fs";
const [aFile, bFile, outFile] = process.argv.slice(2);
const a = PNG.sync.read(readFileSync(aFile));
const b = PNG.sync.read(readFileSync(bFile));
const w = Math.min(a.width, b.width);
const h = Math.min(a.height, b.height);
const out = new PNG({ width: w, height: h });
for (let y = 0; y < h; y++) {
  for (let x = 0; x < w; x++) {
    const ia = (a.width * y + x) << 2;
    const ib = (b.width * y + x) << 2;
    const io = (w * y + x) << 2;
    const d = Math.abs(a.data[ia] - b.data[ib]) + Math.abs(a.data[ia+1] - b.data[ib+1]) + Math.abs(a.data[ia+2] - b.data[ib+2]);
    if (d > 24) {
      out.data[io] = 255; out.data[io+1] = 0; out.data[io+2] = 0; out.data[io+3] = 255;
    } else {
      const g = (a.data[ia] + a.data[ia+1] + a.data[ia+2]) / 3;
      out.data[io] = g; out.data[io+1] = g; out.data[io+2] = g; out.data[io+3] = 90;
    }
  }
}
writeFileSync(outFile, PNG.sync.write(out));
console.log("wrote", outFile, w + "x" + h);
