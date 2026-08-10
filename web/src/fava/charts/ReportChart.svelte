<!-- This file is derived from Fava 1.30.12 (commit #aa7538e8971252c9efc52c8a516a3a77d604553f),
which is Copyright (c) 2015-2016 Dominik Aumayr <dominik@aumayr.name> and
distributed under the MIT License. Adapted for OrangeCount; see NOTICE and
web/provenance-manifest.json. The MIT notice is reproduced here:

  Copyright (c) 2015-2016 Dominik Aumayr <dominik@aumayr.name>

  Permission is hereby granted, free of charge, to any person obtaining a copy
  of this software and associated documentation files (the "Software"), to deal
  in the Software without restriction, including without limitation the rights
  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
  copies of the Software, and to permit persons to whom the Software is
  furnished to do so, subject to the following conditions:

  The above copyright notice and this permission notice shall be included in all
  copies or substantial portions of the Software.

  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
  SOFTWARE. -->

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
 
  // Fava's chart mode switch lives in two localStorage-synced stores
  // (stores/chart.ts): barChartMode ("stacked" | "single", default "stacked")
  // shown when a bar chart has stacked data, and lineChartMode ("line" |
  // "area", default "line") shown for every line chart. OC keeps the same
  // keys, defaults, and gating so the toggle reads with Fava semantics.
  let barMode: "stacked" | "single" = "stacked";
  let lineMode: "line" | "area" = "line";
  try {
    const storedBar = localStorage.getItem("bar-chart-mode");
    if (storedBar === "stacked" || storedBar === "single") barMode = storedBar;
    const storedLine = localStorage.getItem("line-chart-mode");
    if (storedLine === "line" || storedLine === "area") lineMode = storedLine;
  } catch { /* storage is optional */ }
  function setBarMode(mode: "stacked" | "single") {
    barMode = mode;
    try { localStorage.setItem("bar-chart-mode", mode); } catch { /* storage optional */ }
  }
  function setLineMode(mode: "line" | "area") {
    lineMode = mode;
    try { localStorage.setItem("line-chart-mode", mode); } catch { /* storage optional */ }
  }
  $: isBarChart = chart.kind === "stacked-bar" || chart.kind === "bar";
  $: isLineChart = chart.kind !== "stacked-bar" && chart.kind !== "bar" && chart.kind !== "hierarchy";
  // Fava only renders the stacked/single switch when hasStackedData is true,
  // i.e. more than one account contributes. OC's bar series are currencies,
  // so the same gate is "more than one visible series".
  $: showBarMode = isBarChart && visibleSeries.length > 1;
  $: showLineMode = isLineChart && visibleSeries.length > 0;

  function numberValue(value: { display: string }): number {
    if (value.display.includes("/")) {
      const [numerator, denominator] = value.display.split("/").map(Number);
      return denominator ? numerator / denominator : 0;
    }
    const parsed = Number(value.display);
    return Number.isFinite(parsed) ? parsed : 0;
  }

  // Fava's chart legend doubles as a per-currency selector: clicking an entry
  // toggles that series out of the plot (and back in). Hidden labels live here;
  // the value extent below re-derives from whatever stays visible so the
  // y-scale re-fits.
  let hidden: string[] = [];
  function toggleSeries(seriesLabel: string) {
    hidden = hidden.includes(seriesLabel) ? hidden.filter((value) => value !== seriesLabel) : [...hidden, seriesLabel];
  }
  $: visibleSeries = chart.series.filter((series) => !hidden.includes(series.label));
  function colorFor(seriesLabel: string): string {
    const index = chart.series.findIndex((series) => series.label === seriesLabel);
    return colors[(index < 0 ? 0 : index) % colors.length];
  }

  // Fava's hierarchy charts make every node a link into that account's page.
  // Reuse the tree-table convention for the target URL.
  function accountHref(name: string): string {
    return `/account/${encodeURIComponent(name)}`;
  }

 $: pointValues = visibleSeries.flatMap((series) => series.points.map((point) => numberValue(point.value)));
 // Stacked bars total each period's visible series separately for positives
 // and negatives (d3's stackOffsetDiverging places bands around zero), so the
 // value extent has to follow the stack totals rather than raw point values.
 $: stackedTotals = (() => {
   const counts = chart.series[0]?.points.length ?? 0;
   const positiveTotals = new Array(counts).fill(0);
   const negativeTotals = new Array(counts).fill(0);
   for (const series of visibleSeries) {
     series.points.forEach((point, index) => {
       if (index >= counts) return;
       const value = numberValue(point.value);
       if (value >= 0) positiveTotals[index] += value;
       else negativeTotals[index] += value;
     });
   }
   return { positiveTotals, negativeTotals };
 })();
 $: extentValues = (isBarChart && barMode === "stacked")
   ? [...stackedTotals.positiveTotals, ...stackedTotals.negativeTotals]
   : pointValues;
 $: min = Math.min(0, ...(extentValues.length ? extentValues : [0]));
 $: max = Math.max(0, ...(extentValues.length ? extentValues : [0]));
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
 
  // One rectangle per (period, visible series) for stacked mode. Positives
  // stack upward from zero and negatives downward, mirroring Fava's diverging
  // stack offset; every series in a period shares the same band, unlike the
  // side-by-side layout single mode uses.
  interface StackedBarRect { seriesLabel: string; date: string; display: string; x: number; y: number; w: number; h: number; color: string }
  $: stackedBarRects = (() => {
    const counts = chart.series[0]?.points.length ?? 0;
    const periodBand = width <= 1 ? Math.min(8, X1 - X0) : (X1 - X0) / width;
    const barW = Math.max(0.5, periodBand * 0.8);
    const rects: StackedBarRect[] = [];
    for (let index = 0; index < counts; index += 1) {
      let positiveBase = 0;
      let negativeBase = 0;
      const centerX = x(index);
      for (const series of visibleSeries) {
        const point = series.points[index];
        if (!point) continue;
        const value = numberValue(point.value);
        const xRect = centerX - barW / 2;
        if (value >= 0) {
          const yTop = y(positiveBase + value);
          const yBottom = y(positiveBase);
          rects.push({ seriesLabel: series.label, date: point.date, display: point.value.display, x: xRect, y: yTop, w: barW, h: Math.max(0.3, yBottom - yTop), color: colorFor(series.label) });
          positiveBase += value;
        } else {
          const yTop = y(negativeBase);
          const yBottom = y(negativeBase + value);
          rects.push({ seriesLabel: series.label, date: point.date, display: point.value.display, x: xRect, y: yTop, w: barW, h: Math.max(0.3, yBottom - yTop), color: colorFor(series.label) });
          negativeBase += value;
        }
      }
    }
    return rects;
  })();
 
  // Area chart fill: the line path closed down to the zero baseline (clamped
  // to the plot bottom), the way Fava's LineChart builds its area shape.
  const baselineY = 52;
  function areaPath(points: { value: { display: string } }[]): string {
    if (!points.length) return "";
    const line = points.map((point, index) => `${index ? "L" : "M"}${x(index).toFixed(2)},${y(numberValue(point.value)).toFixed(2)}`).join(" ");
    return `${line} L${x(points.length - 1).toFixed(2)},${Math.min(baselineY, y(0)).toFixed(2)} L${x(0).toFixed(2)},${Math.min(baselineY, y(0)).toFixed(2)} Z`;
  }

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

  // --- Tooltip -------------------------------------------------------------
  // One absolutely-positioned card follows the pointer. Bars report the hovered
  // series + period; the line overlay bisects the pointer to the nearest period
  // and lists every visible series there. Hierarchy shapes keep their native
  // <title> fallback.
  let tooltipText = "";
  let tooltipLeft = 0;
  let tooltipTop = 0;
  let tooltipVisible = false;

  function moveTooltip(event: MouseEvent, anchor: Element | null) {
    const card = anchor?.closest(".chart-card");
    if (!card) return;
    const box = card.getBoundingClientRect();
    tooltipLeft = event.clientX - box.left + 12;
    tooltipTop = event.clientY - box.top + 12;
  }

  function showBarTip(event: MouseEvent, seriesLabel: string, point: { date: string; value: { display: string } }) {
    tooltipText = `${seriesLabel} · ${point.date}: ${point.value.display}${chart.currency ? ` ${chart.currency}` : ""}`;
    tooltipVisible = true;
    moveTooltip(event, event.currentTarget as Element);
  }

  function hideTip() { tooltipVisible = false; }

  function showLineTip(event: MouseEvent) {
    const svg = event.currentTarget as SVGSVGElement;
    const points = chart.series[0]?.points ?? [];
    if (!points.length) return;
    const box = svg.getBoundingClientRect();
    if (!box.width) return;
    const viewBoxX = ((event.clientX - box.left) / box.width) * 100;
    const step = width === 1 ? 1 : (X1 - X0) / (width - 1);
    const index = Math.max(0, Math.min(points.length - 1, Math.round((viewBoxX - X0) / step)));
    const lines = visibleSeries
      .map((series) => {
        const value = series.points[index]?.value.display;
        return value === undefined ? "" : `${series.label}: ${value}`;
      })
      .filter(Boolean);
    if (!lines.length) return;
    tooltipText = `${points[index].date}\n${lines.join("\n")}`;
    tooltipVisible = true;
    moveTooltip(event, svg);
  }
</script>

<section class="chart-card" aria-label={chart.title}>
 <h3>{chart.title}</h3>
   {#if showBarMode}
     <span class="mode-switch">
       <label class="button" class:muted={barMode !== "stacked"}>
         <input type="radio" name={"bar-mode-" + chart.title} value="stacked" checked={barMode === "stacked"} on:change={() => setBarMode("stacked")} />
         {label("stackedBars")}
       </label>
       <label class="button" class:muted={barMode !== "single"}>
         <input type="radio" name={"bar-mode-" + chart.title} value="single" checked={barMode === "single"} on:change={() => setBarMode("single")} />
         {label("singleBars")}
       </label>
     </span>
   {:else if showLineMode}
     <span class="mode-switch">
       <label class="button" class:muted={lineMode !== "line"}>
         <input type="radio" name={"line-mode-" + chart.title} value="line" checked={lineMode === "line"} on:change={() => setLineMode("line")} />
         {label("lineChart")}
       </label>
       <label class="button" class:muted={lineMode !== "area"}>
         <input type="radio" name={"line-mode-" + chart.title} value="area" checked={lineMode === "area"} on:change={() => setLineMode("area")} />
         {label("areaChart")}
       </label>
     </span>
   {/if}
 {#if chart.kind === "hierarchy" && (tiles.length || iceRects.length || sunSegments.length)}
    <div class="hierarchy-picker">
      <button type="button" class="unset" class:selected={hierarchyView === "treemap"} on:click={() => (hierarchyView = "treemap")}>{label("treemap")}</button>
      <button type="button" class="unset" class:selected={hierarchyView === "sunburst"} on:click={() => (hierarchyView = "sunburst")}>{label("sunburst")}</button>
      <button type="button" class="unset" class:selected={hierarchyView === "icicle"} on:click={() => (hierarchyView = "icicle")}>{label("icicle")}</button>
    </div>
    {#if hierarchyView === "sunburst" && sunSegments.length}
      <svg class="report-chart report-hierarchy-chart report-sunburst-chart" viewBox="0 0 100 100" role="img" aria-label={chart.title}>
        {#each sunSegments as item (item.name)}
          <a href={accountHref(item.name)}>
            <path
              d={arcPath(item)}
              style={`fill:${sunColor(item)}`}
              opacity=".8"
            ><title>{item.name}: {item.display} {chart.currency}</title></path>
          </a>
        {/each}
      </svg>
    {:else if hierarchyView === "icicle" && iceRects.length}
      <svg class="report-chart report-hierarchy-chart" viewBox="0 0 100 {Math.max(6, iceDepth)}" preserveAspectRatio="none" role="img" aria-label={chart.title}>
        {#each iceRects as item (item.name)}
          <a href={accountHref(item.name)}>
            <rect
              x={item.x + 0.1}
              y={item.y + 0.15}
              width={Math.max(0.2, item.w - 0.2)}
              height={Math.max(0.3, item.h - 0.3)}
              style={`fill:${iceColor(item)}`}
              opacity=".8"
            ><title>{item.name}: {item.display} {chart.currency}</title></rect>
          </a>
        {/each}
      </svg>
    {:else}
      <svg class="report-chart report-hierarchy-chart" viewBox="0 0 100 52" preserveAspectRatio="none" role="img" aria-label={chart.title}>
        {#each tiles as item (item.name)}
          <a href={accountHref(item.name)}>
            <rect
              x={item.x + 0.15}
              y={item.y + 0.15}
              width={Math.max(0.3, item.w - 0.3)}
              height={Math.max(0.3, item.h - 0.3)}
              style={`fill:${tileColor(item)}`}
              opacity=".8"
            ><title>{item.name}: {item.display} {chart.currency}</title></rect>
          </a>
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
     {#if barMode === "stacked" && visibleSeries.length > 1}
       {#each stackedBarRects as rect (rect.seriesLabel + "-" + rect.date)}
         <rect x={rect.x} y={rect.y} width={rect.w} height={rect.h} style={`fill:${rect.color}`} on:mousemove={(e) => showBarTip(e, rect.seriesLabel, { date: rect.date, value: { display: rect.display } })} on:mouseleave={hideTip} />
       {/each}
     {:else}
       {#each visibleSeries as series, seriesIndex (series.label)}
         {#each series.points as point, index (point.date)}
           {@const value = numberValue(point.value)}
           {@const barWidth = Math.max(1, (X1 - X0) / Math.max(1, width) / Math.max(1, visibleSeries.length))}
           <rect x={x(index) - (X1 - X0) / (2 * Math.max(1, width)) + seriesIndex * barWidth} y={barY(value)} width={barWidth - .25} height={barHeight(value)} style={`fill:${colorFor(series.label)}`} on:mousemove={(e) => showBarTip(e, series.label, point)} on:mouseleave={hideTip} />
         {/each}
       {/each}
     {/if}
     {#each xTickIndices as index (index)}
        {@const point = chart.series[0]?.points[index]}
        {#if point}
          <text x={x(index)} y="51" class="chart-tick" text-anchor="middle">{xTickLabel(point.date)}</text>
        {/if}
      {/each}
    </svg>
  {:else}
    <svg class="report-chart report-line-chart" viewBox="0 0 100 52" role="img" aria-label={chart.title} on:mousemove={showLineTip} on:mouseleave={hideTip}>
      {#each yTicks as tick (tick)}
        <line x1={X0} y1={y(tick)} x2={X1} y2={y(tick)} class="chart-grid" />
        <text x={X0 - 1} y={y(tick) + 1} class="chart-tick" text-anchor="end">{tickLabel(tick)}</text>
      {/each}
     <line x1={X0} y1={y(0)} x2={X1} y2={y(0)} class="chart-axis" />
     {#each visibleSeries as series (series.label)}
       {#if lineMode === "area"}
         <path class="area-fill" d={areaPath(series.points)} style={`fill:${colorFor(series.label)}`} />
       {/if}
       <path d={linePath(series.points)} style={`stroke:${colorFor(series.label)}`} />
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
  {#each chart.series as series (series.label)}
    <button
      type="button"
      class="legend"
      class:inactive={hidden.includes(series.label)}
      aria-pressed={!hidden.includes(series.label)}
      on:click={() => toggleSeries(series.label)}
    ><i style={`background:${colorFor(series.label)}`}></i><span>{series.label}</span></button>
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
  {#if tooltipVisible}
    <div class="chart-tooltip" role="status" style={`left:${tooltipLeft}px;top:${tooltipTop}px`}>{tooltipText}</div>
  {/if}
</section>

<style>
  .chart-card { position: relative; margin-bottom: 1rem; }
  .chart-tooltip { position: absolute; z-index: 10; pointer-events: none; max-width: 20rem; padding: .3rem .5rem; background: var(--background-darker); border: 1px solid var(--border); color: var(--text-color-lighter); font-size: .8rem; line-height: 1.35; white-space: pre-line; box-shadow: 0 1px 3px rgb(0 0 0 / .25); }
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
  .report-hierarchy-chart a { cursor: pointer; }
  .report-chart path { fill: none; stroke-width: .8; vector-effect: non-scaling-stroke; }
  .chart-axis { stroke: var(--chart-axis); stroke-width: .25; vector-effect: non-scaling-stroke; }
  .chart-grid { stroke: var(--border); stroke-width: .2; vector-effect: non-scaling-stroke; opacity: .5; }
  .chart-tick { font-size: 2.6px; fill: var(--text-color-lighter); }
  .legend { display: inline-flex; gap: .3rem; align-items: center; margin: 0 .75rem .5rem 0; background: none; border: none; padding: 0; font: inherit; color: inherit; cursor: pointer; }
  .legend i { display: inline-block; width: .7rem; height: .7rem; border-radius: 50%; }
  .legend.inactive span { text-decoration: line-through; }
 .legend.inactive i { filter: grayscale(); }
   /* Fava's ModeSwitch: inline label.button rows with hidden radios; the
      unselected option is muted. Sits above the chart, left-aligned with the
      title row. */
   .mode-switch { display: inline-flex; margin-left: 0.5rem; vertical-align: middle; }
   .mode-switch label.button { display: inline-flex; align-items: center; padding: 0 0.5rem; font-size: 0.85em; color: var(--text-color); cursor: pointer; }
   .mode-switch label.button + label.button { margin-left: 0.125rem; }
   .mode-switch label.button.muted { color: var(--text-color-lightest); }
   .mode-switch input { display: none; }
   .area-fill { opacity: 0.25; }
 table { min-width: 32rem; }
</style>
