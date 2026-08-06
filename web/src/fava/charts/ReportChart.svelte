<script lang="ts">
  import { translations, type Locale } from "../../translations";
  import { formatAmount, type HierarchyNode, type ReportChart } from "../reports/types";

  export let chart: ReportChart;
  export let locale = "en";

  $: catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
  function label(key: string): string { return catalog[key] ?? translations.en[key] ?? key; }

  // The adapter reports availability as a machine status; render the same
  // human-readable copy the rest of the shell uses instead of the raw enum.
  //
  // "native-multi" is deliberately absent: in this shell every chart plots one
  // series per currency, so holding several currencies is the normal case and
  // nothing has been dropped. Reporting it as "Not valued" told the user their
  // data was missing while it was on screen.
  const availabilityKeys: Record<string, string> = {
    "unavailable-price": "unavailablePrice",
    "unavailable-currency": "unavailableCurrency",
  };
  $: availabilityKey = chart.availability ? availabilityKeys[chart.availability] ?? "" : "";
  $: availabilityText = availabilityKey ? label(availabilityKey) : "";

  const colors = ["var(--series-0, #2563eb)", "var(--series-1, #d97706)", "var(--series-2, #16a34a)", "var(--series-3, #9333ea)"];

  function numberValue(value: { display: string }): number {
    if (value.display.includes("/")) {
      const [numerator, denominator] = value.display.split("/").map(Number);
      return denominator ? numerator / denominator : 0;
    }
    const parsed = Number(value.display);
    return Number.isFinite(parsed) ? parsed : 0;
  }

  $: pointValues = chart.series.flatMap((series) => series.points.map((point) => numberValue(point.value)));
  $: min = Math.min(0, ...(pointValues.length ? pointValues : [0]));
  $: max = Math.max(0, ...(pointValues.length ? pointValues : [0]));
  $: range = max - min || 1;
  $: width = Math.max(1, chart.series[0]?.points.length ?? 1);

  // Plot area: a left gutter carries the value labels, the bottom strip the
  // period labels.
  const X0 = 14;
  const X1 = 98;
  function x(index: number): number { return width === 1 ? (X0 + X1) / 2 : X0 + (index / (width - 1)) * (X1 - X0); }
  function y(value: number): number { return 44 - ((value - min) / range) * 42; }
  function linePath(points: { value: { display: string } }[]): string {
    return points.map((point, index) => `${index ? "L" : "M"}${x(index).toFixed(2)},${y(numberValue(point.value)).toFixed(2)}`).join(" ");
  }
  function barHeight(value: number): number { return Math.max(0.5, Math.abs(y(value) - y(0))); }
  function barY(value: number): number { return value >= 0 ? y(value) : y(0); }

  // "Nice" value ticks: step rounds to 1/2/5 times a power of ten, like the
  // d3 axis the upstream charts use.
  function niceTicks(lo: number, hi: number, count = 4): number[] {
    if (!(hi > lo)) return [lo];
    const step0 = (hi - lo) / count;
    const magnitude = Math.pow(10, Math.floor(Math.log10(step0)));
    const residual = step0 / magnitude;
    const step = (residual >= 5 ? 5 : residual >= 2 ? 2 : 1) * magnitude;
    const ticks: number[] = [];
    for (let value = Math.ceil(lo / step) * step; value <= hi + step / 1e6; value += step) {
      ticks.push(value);
    }
    return ticks;
  }
  $: yTicks = niceTicks(min, max);
  $: tickFormat = new Intl.NumberFormat(locale === "zh-CN" ? "zh-CN" : "en", { notation: "compact", maximumFractionDigits: 1 });
  function tickLabel(value: number): string { return tickFormat.format(Math.abs(value) < 1e-9 ? 0 : value); }

  // Sparse period labels: always the first and last point, plus evenly spaced
  // stops between when the series is long.
  $: xTickIndices = (() => {
    const points = chart.series[0]?.points ?? [];
    if (points.length <= 6) return points.map((_, index) => index);
    const count = 5;
    const indices: number[] = [];
    for (let stop = 0; stop < count; stop += 1) {
      indices.push(Math.round((stop * (points.length - 1)) / (count - 1)));
    }
    return indices;
  })();
  function xTickLabel(date: string): string {
    if (chart.interval === "year") return date.slice(0, 4);
    if (chart.interval === "day") return date;
    return date.slice(0, 7);
  }

  // --- Treemap -------------------------------------------------------------
  // Only leaf accounts are laid out: an aggregate's area is the sum of its
  // children, so drawing both would double-count the canvas.
  interface Leaf { name: string; root: string; value: number; display: string }
  interface Tile extends Leaf { x: number; y: number; w: number; h: number }

  function collectLeaves(nodes: HierarchyNode[], root = ""): Leaf[] {
    const out: Leaf[] = [];
    for (const node of nodes) {
      const top = root || node.name.split(":")[0];
      if (node.children?.length) {
        out.push(...collectLeaves(node.children, top));
      } else {
        out.push({ name: node.name, root: top, value: Math.abs(numberValue(node.value)), display: node.value.display });
      }
    }
    return out;
  }

  // Recursive bisection: split the items into two value-balanced groups, cut
  // the rectangle along its longer side, and recurse. This keeps every tile
  // proportional to its value while staying two-dimensional, where a single
  // row of rectangles degenerates into slivers as soon as values are skewed.
  function tile(items: Leaf[], x0: number, y0: number, w: number, h: number): Tile[] {
    if (!items.length || w <= 0 || h <= 0) return [];
    if (items.length === 1) return [{ ...items[0], x: x0, y: y0, w, h }];
    const total = items.reduce((sum, item) => sum + item.value, 0);
    if (!total) return [];
    let running = 0;
    let split = 1;
    for (let index = 0; index < items.length; index += 1) {
      running += items[index].value;
      if (running >= total / 2) { split = index + 1; break; }
    }
    split = Math.min(Math.max(split, 1), items.length - 1);
    const head = items.slice(0, split);
    const tail = items.slice(split);
    const share = head.reduce((sum, item) => sum + item.value, 0) / total;
    if (w >= h) {
      const headWidth = w * share;
      return [...tile(head, x0, y0, headWidth, h), ...tile(tail, x0 + headWidth, y0, w - headWidth, h)];
    }
    const headHeight = h * share;
    return [...tile(head, x0, y0, w, headHeight), ...tile(tail, x0, y0 + headHeight, w, h - headHeight)];
  }

  $: leaves = (chart.nodes ?? [])
    .flatMap((node) => collectLeaves([node]))
    .filter((leaf) => leaf.value > 0)
    .sort((left, right) => right.value - left.value);
  $: roots = [...new Set(leaves.map((leaf) => leaf.root))].sort();
  $: tiles = tile(leaves, 0, 0, 100, 52);
  function tileColor(leaf: Leaf): string { return colors[roots.indexOf(leaf.root) % colors.length]; }

  // --- Icicle --------------------------------------------------------------
  // Fava offers treemap, sunburst, and icicle views of the same hierarchy.
  // The icicle lays each level out as one row of rectangles spanning the
  // parent above, which keeps every aggregate visible at once.
  interface IceRect { name: string; root: string; value: number; display: string; x: number; y: number; w: number; h: number }

  function icicleRects(nodes: HierarchyNode[]): IceRect[] {
    const rects: IceRect[] = [];
    const rowHeight = 6;
    function walk(list: HierarchyNode[], x0: number, span: number, depth: number, root: string) {
      const total = list.reduce((sum, node) => sum + Math.abs(numberValue(node.value)), 0);
      if (!total || span <= 0) return;
      let cursor = x0;
      for (const node of list) {
        const value = Math.abs(numberValue(node.value));
        const width = (value / total) * span;
        const top = root || node.name.split(":")[0];
        if (value > 0) {
          rects.push({ name: node.name, root: top, value, display: node.value.display, x: cursor, y: depth * rowHeight, w: width, h: rowHeight });
        }
        if (node.children?.length) {
          walk(node.children, cursor, width, depth + 1, top);
        }
        cursor += width;
      }
    }
    walk(nodes, 0, 100, 0, "");
    return rects;
  }

  let hierarchyView: "treemap" | "icicle" | "sunburst" = "treemap";
  $: iceRects = icicleRects(chart.nodes ?? []);
  $: iceDepth = iceRects.reduce((max, rect) => Math.max(max, rect.y + rect.h), 0);
  function iceColor(rect: IceRect): string {
    const rootsForIce = [...new Set((chart.nodes ?? []).map((node) => node.name.split(":")[0]))].sort();
    return colors[rootsForIce.indexOf(rect.root) % colors.length];
  }

  // --- Sunburst ------------------------------------------------------------
  // Polar partition: the root accounts sit in the center circle and each
  // deeper level is a ring of annular sectors around them.
  interface ArcSegment { name: string; root: string; value: number; display: string; a0: number; a1: number; depth: number }

  function sunburstSegments(nodes: HierarchyNode[]): ArcSegment[] {
    const segments: ArcSegment[] = [];
    function walk(list: HierarchyNode[], a0: number, span: number, depth: number, root: string) {
      const total = list.reduce((sum, node) => sum + Math.abs(numberValue(node.value)), 0);
      if (!total || span <= 0) return;
      let cursor = a0;
      for (const node of list) {
        const value = Math.abs(numberValue(node.value));
        const angle = (value / total) * span;
        const top = root || node.name.split(":")[0];
        if (value > 0) {
          segments.push({ name: node.name, root: top, value, display: node.value.display, a0: cursor, a1: cursor + angle, depth });
        }
        if (node.children?.length) {
          walk(node.children, cursor, angle, depth + 1, top);
        }
        cursor += angle;
      }
    }
    walk(nodes, 0, Math.PI * 2, 0, "");
    return segments;
  }

  $: sunSegments = sunburstSegments(chart.nodes ?? []);
  $: sunDepth = sunSegments.reduce((max, segment) => Math.max(max, segment.depth), 0) + 1;
  function sunColor(segment: ArcSegment): string {
    const rootsForSun = [...new Set((chart.nodes ?? []).map((node) => node.name.split(":")[0]))].sort();
    return colors[rootsForSun.indexOf(segment.root) % colors.length];
  }
  // Annular sector path. Angles run clockwise from 12 o'clock.
  function arcPath(segment: ArcSegment): string {
    const ring = 46 / sunDepth;
    const r0 = segment.depth * ring + 1;
    const r1 = r0 + ring - 0.5;
    // A full-circle arc collapses (start == end); leave a hairline gap so the
    // renderer still has a direction to sweep.
    const a1 = segment.a1 - segment.a0 >= Math.PI * 2 - 1e-4 ? segment.a0 + Math.PI * 2 - 1e-4 : segment.a1;
    const large = a1 - segment.a0 > Math.PI ? 1 : 0;
    const px = (radius: number, angle: number) => (50 + radius * Math.sin(angle)).toFixed(2);
    const py = (radius: number, angle: number) => (50 - radius * Math.cos(angle)).toFixed(2);
    return `M${px(r1, segment.a0)},${py(r1, segment.a0)} A${r1.toFixed(2)},${r1.toFixed(2)} 0 ${large} 1 ${px(r1, a1)},${py(r1, a1)} L${px(r0, a1)},${py(r0, a1)} A${r0.toFixed(2)},${r0.toFixed(2)} 0 ${large} 0 ${px(r0, segment.a0)},${py(r0, segment.a0)} Z`;
  }
</script>

<section class="chart-card" aria-label={chart.title}>
  <h3>{chart.title}</h3>
  {#if chart.kind === "hierarchy" && (tiles.length || iceRects.length || sunSegments.length)}
    <div class="hierarchy-picker">
      <button type="button" class="unset" class:selected={hierarchyView === "treemap"} on:click={() => (hierarchyView = "treemap")}>{label("treemap")}</button>
      <button type="button" class="unset" class:selected={hierarchyView === "sunburst"} on:click={() => (hierarchyView = "sunburst")}>{label("sunburst")}</button>
      <button type="button" class="unset" class:selected={hierarchyView === "icicle"} on:click={() => (hierarchyView = "icicle")}>{label("icicle")}</button>
    </div>
    {#if hierarchyView === "sunburst" && sunSegments.length}
      <svg class="report-chart report-hierarchy-chart report-sunburst-chart" viewBox="0 0 100 100" role="img" aria-label={chart.title}>
        {#each sunSegments as item (item.name)}
          <path
            d={arcPath(item)}
            style={`fill:${sunColor(item)}`}
            opacity=".8"
          ><title>{item.name}: {item.display} {chart.currency}</title></path>
        {/each}
      </svg>
    {:else if hierarchyView === "icicle" && iceRects.length}
      <svg class="report-chart report-hierarchy-chart" viewBox="0 0 100 {Math.max(6, iceDepth)}" preserveAspectRatio="none" role="img" aria-label={chart.title}>
        {#each iceRects as item (item.name)}
          <rect
            x={item.x + 0.1}
            y={item.y + 0.15}
            width={Math.max(0.2, item.w - 0.2)}
            height={Math.max(0.3, item.h - 0.3)}
            style={`fill:${iceColor(item)}`}
            opacity=".8"
          ><title>{item.name}: {item.display} {chart.currency}</title></rect>
        {/each}
      </svg>
    {:else}
      <svg class="report-chart report-hierarchy-chart" viewBox="0 0 100 52" preserveAspectRatio="none" role="img" aria-label={chart.title}>
        {#each tiles as item (item.name)}
          <rect
            x={item.x + 0.15}
            y={item.y + 0.15}
            width={Math.max(0.3, item.w - 0.3)}
            height={Math.max(0.3, item.h - 0.3)}
            style={`fill:${tileColor(item)}`}
            opacity=".8"
          ><title>{item.name}: {item.display} {chart.currency}</title></rect>
        {/each}
      </svg>
    {/if}
  {:else if chart.kind === "stacked-bar" || chart.kind === "bar"}
    <svg class="report-chart report-bar-chart" viewBox="0 0 100 52" role="img" aria-label={chart.title}>
      {#each yTicks as tick (tick)}
        <line x1={X0} y1={y(tick)} x2={X1} y2={y(tick)} class="chart-grid" />
        <text x={X0 - 1} y={y(tick) + 1} class="chart-tick" text-anchor="end">{tickLabel(tick)}</text>
      {/each}
      <line x1={X0} y1={y(0)} x2={X1} y2={y(0)} class="chart-axis" />
      {#each chart.series as series, seriesIndex (series.label)}
        {#each series.points as point, index (point.date)}
          {@const value = numberValue(point.value)}
          {@const barWidth = Math.max(1, (X1 - X0) / Math.max(1, width) / Math.max(1, chart.series.length))}
          <rect x={x(index) - (X1 - X0) / (2 * Math.max(1, width)) + seriesIndex * barWidth} y={barY(value)} width={barWidth - .25} height={barHeight(value)} style={`fill:${colors[seriesIndex % colors.length]}`} />
        {/each}
      {/each}
      {#each xTickIndices as index (index)}
        {@const point = chart.series[0]?.points[index]}
        {#if point}
          <text x={x(index)} y="51" class="chart-tick" text-anchor="middle">{xTickLabel(point.date)}</text>
        {/if}
      {/each}
    </svg>
  {:else}
    <svg class="report-chart report-line-chart" viewBox="0 0 100 52" role="img" aria-label={chart.title}>
      {#each yTicks as tick (tick)}
        <line x1={X0} y1={y(tick)} x2={X1} y2={y(tick)} class="chart-grid" />
        <text x={X0 - 1} y={y(tick) + 1} class="chart-tick" text-anchor="end">{tickLabel(tick)}</text>
      {/each}
      <line x1={X0} y1={y(0)} x2={X1} y2={y(0)} class="chart-axis" />
      {#each chart.series as series, index (series.label)}
        <path d={linePath(series.points)} style={`stroke:${colors[index % colors.length]}`} />
      {/each}
      {#each xTickIndices as index (index)}
        {@const point = chart.series[0]?.points[index]}
        {#if point}
          <text x={x(index)} y="51" class="chart-tick" text-anchor="middle">{xTickLabel(point.date)}</text>
        {/if}
      {/each}
    </svg>
  {/if}
  <p class="chart-meta">{chart.interval} · {chart.valuation}{chart.currency ? ` · ${chart.currency}` : ""}</p>
  {#each chart.series as series, index (series.label)}
    <span class="legend"><i style={`background:${colors[index % colors.length]}`}></i>{series.label}</span>
  {/each}
  {#if availabilityText}
    <p class="chart-availability">{availabilityText}</p>
  {/if}
  <!-- The tabular fallback stays reachable for keyboard and screen-reader use,
       but starts collapsed: an unbounded per-period dump above the account
       trees pushed the actual report off the first screen. -->
  <details class="chart-data">
    <summary>{label("chartData")}</summary>
    <div class="chart-scroll">
    <table>
      <thead>
        <tr>
          <th scope="col">Period</th>
          {#each chart.series as series (series.label)}
            <th scope="col" class="num">{series.label}</th>
          {/each}
        </tr>
      </thead>
      <tbody>
        {#each (chart.series[0]?.points ?? []) as point, index (point.date)}
          <tr>
            <th scope="row">{point.date}</th>
            {#each chart.series as series (series.label)}
              <td class="num">{formatAmount(series.points[index]?.value)}</td>
            {/each}
          </tr>
        {:else}
          <tr><td colspan={chart.series.length + 1}>No chart data.</td></tr>
        {/each}
      </tbody>
    </table>
    </div>
  </details>
</section>

<style>
  .chart-card { margin-bottom: 1rem; }
  .hierarchy-picker { margin-bottom: .5rem; color: var(--text-color-lightest); text-align: center; }
  .hierarchy-picker button { padding: 0 .5em; }
  .hierarchy-picker button + button { border-left: 1px solid var(--text-color-lighter); }
  .hierarchy-picker button.selected, .hierarchy-picker button:hover { color: var(--text-color-lighter); }
  .chart-meta, .chart-availability { color: var(--text-color-lightest); }
  .chart-data { margin-top: .5rem; }
  .chart-data summary { color: var(--text-color-lightest); cursor: pointer; }
  .chart-scroll { overflow-x: auto; max-height: 22rem; overflow-y: auto; }
  .report-chart { display: block; width: min(100%, 52rem); height: 14rem; margin-bottom: .5rem; background: var(--background-darker); border: 1px solid var(--border); }
  .report-sunburst-chart { width: 14rem; }
  .report-chart path { fill: none; stroke-width: .8; vector-effect: non-scaling-stroke; }
  .chart-axis { stroke: var(--chart-axis); stroke-width: .25; vector-effect: non-scaling-stroke; }
  .chart-grid { stroke: var(--border); stroke-width: .2; vector-effect: non-scaling-stroke; opacity: .5; }
  .chart-tick { font-size: 2.6px; fill: var(--text-color-lighter); }
  .legend { display: inline-flex; gap: .3rem; align-items: center; margin: 0 .75rem .5rem 0; }
  .legend i { display: inline-block; width: .7rem; height: .7rem; border-radius: 50%; }
  table { min-width: 32rem; }
</style>
