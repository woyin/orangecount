<script lang="ts">
  // Mirrors Fava's commodities LineChart semantics (time-scaled single price
  // series, padded value extent, pointer tooltip) in the shell's
  // dependency-free SVG chart style used by ReportChart/ScatterPlot.
  export let points: { date: string; display: string; value: number }[] = [];
  export let formatTip: (point: { date: string; display: string; value: number }) => string = (point) => `${point.date}: ${point.display}`;
  /** "line" draws a stroked line; "area" fills beneath it to the zero baseline. */
  export let mode: "line" | "area" = "line";

  const X0 = 14;
  const X1 = 98;
  const Y0 = 4;
  const Y1 = 44;

  $: data = points
    .filter((point) => Number.isFinite(point.value) && point.date)
    .map((point) => ({ ...point, timestamp: Date.parse(point.date) }))
    .filter((point) => Number.isFinite(point.timestamp))
    .sort((left, right) => left.timestamp - right.timestamp);
  $: timeExtent = data.length
    ? [data[0].timestamp, data[data.length - 1].timestamp]
    : [0, 1];
  // Upstream pads the value extent instead of forcing zero; price series
  // would flatten against a zero baseline.
  $: valueExtent = (() => {
    if (!data.length) return [0, 1];
    let lo = Math.min(...data.map((point) => point.value));
    let hi = Math.max(...data.map((point) => point.value));
    if (mode === "area") {
      lo = Math.min(lo, 0);
      hi = Math.max(hi, 0);
    }
    if (lo === hi) {
      lo -= 1;
      hi += 1;
    }
    const pad = (hi - lo) * 0.03;
    return [lo - pad, hi + pad];
  })();

  function x(timestamp: number): number {
    const span = timeExtent[1] - timeExtent[0];
    if (!span) return (X0 + X1) / 2;
    return X0 + ((timestamp - timeExtent[0]) / span) * (X1 - X0);
  }

  function y(value: number): number {
    const span = valueExtent[1] - valueExtent[0];
    if (!span) return (Y0 + Y1) / 2;
    return Y1 - ((value - valueExtent[0]) / span) * (Y1 - Y0);
  }

  $: path = data
    .map((point, index) => `${index ? "L" : "M"}${x(point.timestamp).toFixed(2)},${y(point.value).toFixed(2)}`)
    .join(" ");

  // Area mode closes the stroke back to the zero baseline so the region under
  // the line is filled.
  $: areaPath = mode === "area" && data.length
    ? `${path} L${x(data[data.length - 1].timestamp).toFixed(2)},${y(0).toFixed(2)} L${x(data[0].timestamp).toFixed(2)},${y(0).toFixed(2)} Z`
    : "";

  // "Nice" value ticks: step rounds to 1/2/5 times a power of ten, like the
  // d3 axis the upstream charts use.
  function niceTicks(lo: number, hi: number, count = 4): number[] {
    if (!(hi > lo)) return [lo];
    const step0 = (hi - lo) / count;
    const magnitude = Math.pow(10, Math.floor(Math.log10(step0)));
    const residual = step0 / magnitude;
    const step = (residual >= 5 ? 5 : residual >= 2 ? 2 : 1) * magnitude;
    const ticks: number[] = [];
    for (let value = Math.ceil(lo / step) * step; value <= hi + step / 1e-6; value += step) {
      ticks.push(value);
    }
    return ticks;
  }
  $: yTicks = niceTicks(valueExtent[0], valueExtent[1]);
  $: tickFormat = new Intl.NumberFormat("en", { notation: "compact", maximumFractionDigits: 2 });
  function tickLabel(value: number): string { return tickFormat.format(Math.abs(value) < 1e-9 ? 0 : value); }

  $: xTicks = (() => {
    if (!data.length) return [];
    const count = Math.min(5, data.length);
    const ticks: typeof data = [];
    for (let stop = 0; stop < count; stop += 1) {
      const timestamp = timeExtent[0] + ((timeExtent[1] - timeExtent[0]) * stop) / Math.max(1, count - 1);
      let nearest = data[0];
      for (const point of data) {
        if (Math.abs(point.timestamp - timestamp) < Math.abs(nearest.timestamp - timestamp)) nearest = point;
      }
      if (!ticks.some((tick) => tick.date === nearest.date)) ticks.push(nearest);
    }
    return ticks;
  })();

  function xTickLabel(date: string): string {
    const spanDays = (timeExtent[1] - timeExtent[0]) / 86400000;
    return spanDays > 366 ? date.slice(0, 4) : date.slice(0, 7);
  }

  let tooltipText = "";
  let tooltipLeft = 0;
  let tooltipTop = 0;
  let tooltipVisible = false;

  function showTip(event: MouseEvent) {
    const svg = event.currentTarget as SVGSVGElement;
    const box = svg.getBoundingClientRect();
    if (!box.width || !data.length) return;
    const vx = ((event.clientX - box.left) / box.width) * 100;
    let best = data[0];
    for (const point of data) {
      if (Math.abs(x(point.timestamp) - vx) < Math.abs(x(best.timestamp) - vx)) best = point;
    }
    tooltipText = formatTip(best);
    tooltipVisible = true;
    const card = svg.closest(".line-card");
    const cardBox = card?.getBoundingClientRect();
    if (cardBox) {
      tooltipLeft = event.clientX - cardBox.left + 12;
      tooltipTop = event.clientY - cardBox.top + 12;
    }
  }

  function hideTip() { tooltipVisible = false; }
</script>

<section class="line-card">
  {#if data.length}
    <svg class="line-chart" viewBox="0 0 100 52" role="img" aria-label="Price line chart" on:mousemove={showTip} on:mouseleave={hideTip}>
      {#each yTicks as tick (tick)}
        <line x1={X0} y1={y(tick)} x2={X1} y2={y(tick)} class="chart-grid" />
        <text x={X0 - 1} y={y(tick) + 1} class="chart-tick" text-anchor="end">{tickLabel(tick)}</text>
      {/each}
      {#if areaPath}
        <path d={areaPath} class="area-path" />
      {/if}
      <path d={path} class="line-path" />
      {#each xTicks as tick (tick.date)}
        <text x={x(tick.timestamp)} y="51" class="chart-tick" text-anchor="middle">{xTickLabel(tick.date)}</text>
      {/each}
    </svg>
  {:else}
    <p class="chart-empty">No prices.</p>
  {/if}
  {#if tooltipVisible}
    <div class="chart-tooltip" role="status" style={`left:${tooltipLeft}px;top:${tooltipTop}px`}>{tooltipText}</div>
  {/if}
</section>

<style>
  .line-card { position: relative; margin-bottom: 1rem; }
  .line-chart { display: block; width: min(100%, 52rem); height: 14rem; margin-bottom: .5rem; background: var(--background-darker); border: 1px solid var(--border); }
  .line-path { fill: none; stroke: var(--series-0, #2563eb); stroke-width: .8; vector-effect: non-scaling-stroke; }
  .area-path { fill: var(--series-0, #2563eb); fill-opacity: .18; stroke: none; }
  .chart-grid { stroke: var(--border); stroke-width: .2; vector-effect: non-scaling-stroke; opacity: .5; }
  .chart-tick { font-size: 2.6px; fill: var(--text-color-lighter); }
  .chart-empty { color: var(--text-color-lightest); }
  .chart-tooltip { position: absolute; z-index: 10; pointer-events: none; max-width: 24rem; padding: .3rem .5rem; background: var(--background-darker); border: 1px solid var(--border); color: var(--text-color-lighter); font-size: .8rem; line-height: 1.35; white-space: pre-line; box-shadow: 0 1px 3px rgb(0 0 0 / .25); }
</style>
