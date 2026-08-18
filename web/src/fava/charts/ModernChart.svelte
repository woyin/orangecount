<script lang="ts">
 import { scaleBand, scaleLinear, scaleOrdinal } from "d3-scale";
 import { interpolateHcl } from "d3-interpolate";
 import { max, min } from "d3-array";
  import { stack, stackOffsetDiverging, area, line, curveMonotoneX } from "d3-shape";
  import { format } from "d3-format";
  import { onMount, onDestroy } from "svelte";
  import { translations, type Locale } from "../../translations";
  import { formatAmount, type ReportChart } from "../reports/types";

  // Modern chart layer (Approved Fava deviation, owner presentation preference).
  // See ADR-0040 and CONTEXT.md "Modern chart layer". It consumes the same
  // adapter data contract as the parity ReportChart and never changes ledger
  // semantics: valuation, currency aggregation, and stacked-vs-single are pure
  // presentation. d3 owns scales, axes, stack offsets, and tick formatting.

  export let chart: ReportChart;
  export let locale = "en";

  $: catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
  function label(key: string): string { return catalog[key] ?? translations.en[key] ?? key; }

  function numberValue(value: { display: string } | undefined): number {
    if (!value) return 0;
    if (value.display.includes("/")) {
      const [numerator, denominator] = value.display.split("/").map(Number);
      return denominator ? numerator / denominator : 0;
    }
    const parsed = Number(value.display);
    return Number.isFinite(parsed) ? parsed : 0;
  }

  $: isBar = chart.kind === "bar" || chart.kind === "stacked-bar";

  // Bar/line mode is shared with the parity layer via the same localStorage
  // keys, so a choice made in one skin carries across skins and pages.
  let barMode: "stacked" | "single" = "stacked";
  let lineMode: "line" | "area" = "line";
  try {
    const storedBar = localStorage.getItem("bar-chart-mode");
    if (storedBar === "stacked" || storedBar === "single") barMode = storedBar;
    const storedLine = localStorage.getItem("line-chart-mode");
    if (storedLine === "line" || storedLine === "area") lineMode = storedLine;
  } catch { /* storage optional */ }
  function setBarMode(mode: "stacked" | "single") { barMode = mode; try { localStorage.setItem("bar-chart-mode", mode); } catch { /* storage optional */ } }
  function setLineMode(mode: "line" | "area") { lineMode = mode; try { localStorage.setItem("line-chart-mode", mode); } catch { /* storage optional */ } }

  // Currency-dot legend doubles as a per-series on/off selector (same UX as the
  // parity layer and Fava's ChartLegend). Hidden series stay out of the plot
  // and the y-scale re-fits to whatever remains visible.
  let hidden: string[] = [];
  function toggleSeries(seriesLabel: string) {
    hidden = hidden.includes(seriesLabel) ? hidden.filter((v) => v !== seriesLabel) : [...hidden, seriesLabel];
  }
  $: visibleSeries = chart.series.filter((series) => !hidden.includes(series.label));

  // HCL ordinal palette anchored on the same blue/green the parity layer falls
  // back to. Interpolating in HCL keeps lightness roughly constant so series
  // stay distinguishable for color-blind viewers and never collide for ledgers
  // that hold more currencies than the parity layer's fixed four.
  const paletteStart = "#2563eb";
  const paletteEnd = "#16a34a";
  function colorFor(seriesLabel: string): string {
    const index = Math.max(0, chart.series.findIndex((s) => s.label === seriesLabel));
    const total = Math.max(1, chart.series.length);
    return interpolateHcl(paletteStart, paletteEnd)(total === 1 ? 0 : index / (total - 1));
  }

  // Responsive width: the SVG redraws to the container's pixel width so wide
  // screens fill the space instead of centering a tiny viewBox. The container
  // is always rendered so the observer keeps a stable target.
  let containerEl: HTMLElement;
  let width = 0;
  const height = 280;
  const margin = { top: 16, right: 24, bottom: 32, left: 60 };
  let resizeObserver: ResizeObserver | null = null;
  onMount(() => {
    resizeObserver = new ResizeObserver((entries) => {
      for (const entry of entries) width = Math.max(0, entry.contentRect.width);
    });
    if (containerEl) resizeObserver.observe(containerEl);
  });
  onDestroy(() => { resizeObserver?.disconnect(); });

  $: innerWidth = Math.max(0, width - margin.left - margin.right);
  $: innerHeight = height - margin.top - margin.bottom;

  // A series may begin later than another (for example, when a commodity is
  // first purchased years after the account opened). Use the union of dates,
  // rather than the first series' index positions, so sparse series neither
  // crash the chart nor appear at the wrong point in time.
  $: periods = [...new Set(chart.series.flatMap((series) => series.points.map((point) => point.date)))].sort();
  function pointAt(series: typeof visibleSeries[number], date: string | undefined) {
    return date === undefined ? undefined : series.points.find((point) => point.date === date);
  }
  $: hasMultipleSeries = visibleSeries.length > 1;
  $: showStacked = isBar && barMode === "stacked" && hasMultipleSeries;

  // Per-period datum for d3 stack: one row per period keyed by series label.
  $: stackData = periods.map((_, i) => {
    const row: Record<string, number> = { __i: i };
    for (const series of visibleSeries) row[series.label] = numberValue(pointAt(series, periods[i])?.value);
    return row;
  });
  $: stackedLayers = showStacked
    ? stack<string, Record<string, number>>()
        .keys(visibleSeries.map((s) => s.label))
        .value((d, key) => d[key] ?? 0)
        .offset(stackOffsetDiverging)(stackData)
    : [];

  // Y extent follows the stacked totals in stacked mode; otherwise raw points.
  $: yExtentValues = showStacked
    ? stackedLayers.flatMap((layer) => layer.map((seg) => [seg[0], seg[1]]).flat()) as number[]
    : visibleSeries.flatMap((s) => s.points.map((p) => numberValue(p.value)));
  $: yMin = min(yExtentValues.concat([0])) ?? 0;
  $: yMax = max(yExtentValues.concat([0])) ?? 0;

  $: y = scaleLinear([innerHeight, 0]).domain([yMin, yMax]).nice();
  $: xBand = scaleBand([0, innerWidth]).domain(periods).padding(0.2);

 const compactTick = format("~s");
 function tickFormat(value: number): string { try { return compactTick(value); } catch { return String(value); } }
 // Axes are rendered declaratively ({#each} over computed tick values). An
 // earlier imperative d3-axis version bound to <g> refs was cleared by
 // Svelte's legacy patch pass, leaving empty axes; rendering the ticks as
 // first-class template elements avoids that race entirely.
 $: yTickValues = innerHeight > 0 ? y.ticks(Math.max(2, Math.floor(innerHeight / 40))) : [];
 $: xTickPeriods = (() => {
   if (!innerWidth || !periods.length) return [];
   const every = Math.max(1, Math.ceil(periods.length / Math.max(2, Math.floor(innerWidth / 80))));
   return periods.filter((_, i) => i % every === 0 || i === periods.length - 1);
 })();
 function xTickLabel(date: string): string {
    if (chart.interval === "year") return date.slice(0, 4);
    if (chart.interval === "day") return date;
    return date.slice(0, 7);
  }

  // Single-mode bars are placed side-by-side within each period's band.
  $: singleBars = (() => {
    if (!isBar || showStacked || !periods.length) return [];
    const groupWidth = xBand.bandwidth();
    const barWidth = Math.max(1, groupWidth / Math.max(1, visibleSeries.length));
    return visibleSeries.flatMap((series, s) => periods.flatMap((date, i) => {
      const point = pointAt(series, date);
      if (!point) return [];
      const value = numberValue(point.value);
      return [{ seriesLabel: series.label, date: point.date, display: point.value.display, value, x: (xBand(periods[i]) ?? 0) + s * barWidth, w: Math.max(0.5, barWidth - 1), y: y(Math.max(0, value)), h: Math.abs(y(value) - y(0)) }];
    }));
  })();

  // Crosshair: the pointer's nearest period drives a vertical guide, a dot on
  // every visible series, and a card listing each series' value there. Hovering
  // a bar/series dims the others (linked series highlight).
  let hoverIndex: number | null = null;
  let hoverPinned = false;
  let highlighted: string | null = null;
  function onMove(event: MouseEvent) {
    const svg = event.currentTarget as SVGSVGElement;
    const rect = svg.getBoundingClientRect();
    if (!rect.width || !innerWidth) return;
    const ratio = (event.clientX - rect.left - margin.left) / innerWidth;
    if (ratio < -0.02 || ratio > 1.02) { hoverIndex = null; return; }
    hoverIndex = periods.length > 1 ? Math.max(0, Math.min(periods.length - 1, Math.round(ratio * (periods.length - 1)))) : 0;
  }
  function onLeave() { if (!hoverPinned) hoverIndex = null; }
  function onClick(event: MouseEvent) { onMove(event); hoverPinned = hoverIndex !== null; }
  function clearPinnedHover() { hoverPinned = false; hoverIndex = null; }
  function onKeydown(event: KeyboardEvent) { if (event.key === "Escape") clearPinnedHover(); }
  $: hoverX = hoverIndex !== null ? (xBand(periods[hoverIndex]) ?? 0) + (isBar ? xBand.bandwidth() / 2 : 0) : 0;

  function linePath(series: typeof visibleSeries[number]): string {
    return line<(typeof series.points)[number]>()
      .x((point) => (xBand(point.date) ?? 0) + xBand.bandwidth() / 2)
      .y((p) => y(numberValue(p.value)))
      .curve(curveMonotoneX)(series.points as any);
  }
  function areaPath(series: typeof visibleSeries[number]): string {
    return area<(typeof series.points)[number]>()
      .x((point) => (xBand(point.date) ?? 0) + xBand.bandwidth() / 2)
      .y0(Math.min(innerHeight, y(0)))
      .y1((p) => y(numberValue(p.value)))
      .curve(curveMonotoneX)(series.points as any);
  }
</script>

<section class="modern-chart-card" aria-label={chart.title}>
  <header class="modern-chart-header">
    <h3>{chart.title}</h3>
    {#if isBar && hasMultipleSeries}
      <span class="mode-switch">
        <label class="button" class:muted={barMode !== "stacked"}>
          <input type="radio" name={"modern-bar-" + chart.title} value="stacked" checked={barMode === "stacked"} on:change={() => setBarMode("stacked")} />
          {label("stackedBars")}
        </label>
        <label class="button" class:muted={barMode !== "single"}>
          <input type="radio" name={"modern-bar-" + chart.title} value="single" checked={barMode === "single"} on:change={() => setBarMode("single")} />
          {label("singleBars")}
        </label>
      </span>
    {:else if !isBar && visibleSeries.length}
      <span class="mode-switch">
        <label class="button" class:muted={lineMode !== "line"}>
          <input type="radio" name={"modern-line-" + chart.title} value="line" checked={lineMode === "line"} on:change={() => setLineMode("line")} />
          {label("lineChart")}
        </label>
        <label class="button" class:muted={lineMode !== "area"}>
          <input type="radio" name={"modern-line-" + chart.title} value="area" checked={lineMode === "area"} on:change={() => setLineMode("area")} />
          {label("areaChart")}
        </label>
      </span>
    {/if}
  </header>

  <div class="modern-chart-wrap" bind:this={containerEl}>
    {#if width > 0}
      <svg class="modern-chart" viewBox={`0 0 ${width} ${height}`} width={width} height={height} role="button" aria-label={`${chart.title}; click to keep the value visible`} tabindex="0" on:mousemove={onMove} on:mouseleave={onLeave} on:click={onClick} on:keydown={onKeydown}>
        <g transform={`translate(${margin.left},${margin.top})`}>
         {#each yTickValues as tick (tick)}
           <line class="grid-line" x1={0} x2={innerWidth} y1={y(tick)} y2={y(tick)} />
           <text class="y-tick" x={-8} y={y(tick)} dy="0.32em" text-anchor="end">{tickFormat(tick)}</text>
         {/each}
         {#each xTickPeriods as period (period)}
           <text class="x-tick" x={(xBand(period) ?? 0) + xBand.bandwidth() / 2} y={innerHeight + 18} text-anchor="middle">{xTickLabel(period)}</text>
         {/each}
         <line class="zero-baseline" x1={0} x2={innerWidth} y1={y(0)} y2={y(0)} />

          {#if isBar && showStacked}
            {#each stackedLayers as layer (layer.key)}
              {#each layer as segment, i (i)}
                <rect
                  x={xBand(periods[i]) ?? 0}
                  y={y(segment[1])}
                  width={Math.max(0.5, xBand.bandwidth() - 1)}
                  height={Math.max(0, y(segment[0]) - y(segment[1]))}
                  fill={colorFor(String(layer.key))}
                  class:dimmed={highlighted !== null && highlighted !== String(layer.key)}
                  on:mouseenter={() => (highlighted = String(layer.key))}
                  on:mouseleave={() => (highlighted = null)}
                />
              {/each}
            {/each}
          {:else if isBar}
            {#each singleBars as bar (bar.seriesLabel + "-" + bar.date)}
              <rect
                x={bar.x}
                y={bar.y}
                width={bar.w}
                height={bar.h}
                fill={colorFor(bar.seriesLabel)}
                class:dimmed={highlighted !== null && highlighted !== bar.seriesLabel}
                on:mouseenter={() => (highlighted = bar.seriesLabel)}
                on:mouseleave={() => (highlighted = null)}
              />
            {/each}
          {:else}
            {#each visibleSeries as series (series.label)}
              <g class:dimmed={highlighted !== null && highlighted !== series.label} on:mouseenter={() => (highlighted = series.label)} on:mouseleave={() => (highlighted = null)}>
                {#if lineMode === "area"}
                  <path class="area-fill" d={areaPath(series)} fill={colorFor(series.label)} />
                {/if}
                <path class="series-line" d={linePath(series)} stroke={colorFor(series.label)} />
              </g>
            {/each}
          {/if}

          {#if hoverIndex !== null}
            <line class="crosshair" x1={hoverX} x2={hoverX} y1={0} y2={innerHeight} />
            {#each visibleSeries as series (series.label)}
              {#if pointAt(series, periods[hoverIndex])}
                <circle class="hover-dot" cx={hoverX} cy={y(numberValue(pointAt(series, periods[hoverIndex])?.value))} r={3.5} fill={colorFor(series.label)} />
              {/if}
            {/each}
          {/if}
        </g>
      </svg>
    {/if}
  </div>

  {#if hoverIndex !== null}
    <div class="hover-card">
      <span class="hover-date">{periods[hoverIndex]}</span>
      {#each visibleSeries as series (series.label)}
        <span class="hover-row"><i style={`background:${colorFor(series.label)}`}></i>{series.label}: {formatAmount(pointAt(series, periods[hoverIndex])?.value)}{chart.currency ? ` ${chart.currency}` : ""}</span>
      {/each}
    </div>
  {/if}

  <p class="chart-meta">{chart.interval} · {chart.valuation}{chart.currency ? ` · ${chart.currency}` : ""}</p>
  {#each chart.series as series (series.label)}
    <button type="button" class="legend" class:inactive={hidden.includes(series.label)} aria-pressed={!hidden.includes(series.label)} on:click={() => toggleSeries(series.label)}>
      <i style={`background:${colorFor(series.label)}`}></i><span>{series.label}</span>
    </button>
  {/each}
</section>

<style>
  .modern-chart-card { position: relative; margin-bottom: 1rem; }
  .modern-chart-header { display: flex; align-items: center; gap: 0.75rem; flex-wrap: wrap; margin-bottom: 0.25rem; }
  .modern-chart-header h3 { margin: 0; }
  .modern-chart-wrap { width: 100%; min-height: 280px; }
  .modern-chart { display: block; width: 100%; height: 280px; background: var(--background-darker); border: 1px solid var(--border); }
 .y-tick, .x-tick { font-size: 11px; fill: var(--text-color-lighter); font-variant-numeric: tabular-nums; }
 .grid-line { stroke: var(--border); stroke-width: 1px; opacity: 0.15; }
  .zero-baseline { stroke: var(--chart-axis, var(--text-color-lightest)); stroke-width: 1px; opacity: 0.4; }
  .series-line { fill: none; stroke-width: 2; }
  .area-fill { opacity: 0.22; }
  .dimmed { opacity: 0.25; transition: opacity 120ms; }
  .crosshair { stroke: var(--text-color-lighter); stroke-width: 1px; stroke-dasharray: 3 3; opacity: 0.6; pointer-events: none; }
  .hover-dot { stroke: var(--background-darker); stroke-width: 1.5; pointer-events: none; }
  .hover-card { position: absolute; z-index: 10; pointer-events: none; left: 50%; transform: translateX(-50%); top: 0.25rem; display: flex; flex-direction: column; gap: 0.1rem; padding: 0.35rem 0.6rem; background: var(--background-darker); border: 1px solid var(--border); color: var(--text-color-lighter); font-size: 0.8rem; font-variant-numeric: tabular-nums; box-shadow: 0 1px 3px rgb(0 0 0 / .25); }
  .hover-date { font-weight: 600; }
  .hover-row { display: inline-flex; gap: 0.3rem; align-items: center; }
  .hover-row i { display: inline-block; width: 0.6rem; height: 0.6rem; border-radius: 50%; }
  .chart-meta { color: var(--text-color-lightest); }
  .mode-switch { display: inline-flex; margin-left: auto; }
  .mode-switch label.button { display: inline-flex; align-items: center; padding: 0 0.5rem; font-size: 0.85em; color: var(--text-color); cursor: pointer; }
  .mode-switch label.button + label.button { margin-left: 0.125rem; }
  .mode-switch label.button.muted { color: var(--text-color-lightest); }
  .mode-switch input { display: none; }
  .legend { display: inline-flex; gap: 0.3rem; align-items: center; margin: 0 0.75rem 0.5rem 0; background: none; border: none; padding: 0; font: inherit; color: inherit; cursor: pointer; font-variant-numeric: tabular-nums; }
  .legend i { display: inline-block; width: 0.7rem; height: 0.7rem; border-radius: 50%; }
  .legend.inactive span { text-decoration: line-through; }
  .legend.inactive i { filter: grayscale(); }
</style>
