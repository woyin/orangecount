<script lang="ts">
  // Mirrors Fava's events ScatterPlot semantics (time x event-type point
  // scale, one dot per event, pointer tooltip with description and date,
  // per-type colors, future dates desaturated) in the shell's dependency-free
  // SVG chart style used by ReportChart.
  export let events: { date: string; type: string; description: string }[] = [];

  interface Datum {
    timestamp: number;
    date: string;
    type: string;
    description: string;
  }

  const X0 = 16;
  const X1 = 98;
  const Y0 = 4;
  const Y1 = 44;

  // Upstream colors come from an HCL wheel (offset 270, 45/70); HSL at fixed
  // saturation/lightness approximates the same "equal brightness" intent
  // without pulling in d3-color.
  function colorForType(index: number, count: number): string {
    const hue = ((index * (360 / Math.max(1, count))) + 270) % 360;
    return `hsl(${hue.toFixed(0)}, 45%, 60%)`;
  }

  $: data = events
    .map((event) => ({ timestamp: Date.parse(event.date), date: event.date, type: event.type, description: event.description }))
    .filter((datum) => Number.isFinite(datum.timestamp)) as Datum[];
  $: types = [...new Set(data.map((datum) => datum.type))];
  $: extent = data.length
    ? [Math.min(...data.map((datum) => datum.timestamp)), Math.max(...data.map((datum) => datum.timestamp))]
    : [0, 1];

  function x(timestamp: number): number {
    const span = extent[1] - extent[0];
    if (!span) return (X0 + X1) / 2;
    return X0 + ((timestamp - extent[0]) / span) * (X1 - X0);
  }

  // scalePoint with padding 1 over the range [bottom, top]: the first type
  // sits at the bottom, matching the upstream scatterplot.
  function y(type: string): number {
    const index = types.indexOf(type);
    const count = types.length;
    if (!count) return Y0;
    if (count === 1) return (Y0 + Y1) / 2;
    const padding = 1;
    const step = (Y0 - Y1) / (count - 1 + 2 * padding);
    return Y1 + step * (padding + index);
  }

  $: today = new Date().toISOString().slice(0, 10);

  // Sparse date ticks: first, last, and evenly spaced stops between.
  $: xTicks = (() => {
    if (!data.length) return [];
    const count = Math.min(5, data.length);
    const ticks: Datum[] = [];
    for (let stop = 0; stop < count; stop += 1) {
      const timestamp = extent[0] + ((extent[1] - extent[0]) * stop) / Math.max(1, count - 1);
      let nearest = data[0];
      for (const datum of data) {
        if (Math.abs(datum.timestamp - timestamp) < Math.abs(nearest.timestamp - timestamp)) nearest = datum;
      }
      if (!ticks.some((tick) => tick.date === nearest.date)) ticks.push(nearest);
    }
    return ticks;
  })();

  function tickLabel(date: string): string {
    const spanDays = (extent[1] - extent[0]) / 86400000;
    return spanDays > 366 ? date.slice(0, 4) : date.slice(0, 7);
  }

  function typeLabel(type: string): string {
    return type.length > 12 ? `${type.slice(0, 11)}…` : type;
  }

  let tooltipText = "";
  let tooltipLeft = 0;
  let tooltipTop = 0;
  let tooltipVisible = false;

  function showTip(event: MouseEvent) {
    const svg = event.currentTarget as SVGSVGElement;
    const box = svg.getBoundingClientRect();
    if (!box.width) return;
    const vx = ((event.clientX - box.left) / box.width) * 100;
    const vy = ((event.clientY - box.top) / box.height) * 52;
    let best: Datum | null = null;
    let bestDistance = 3.5;
    for (const datum of data) {
      const distance = Math.hypot(x(datum.timestamp) - vx, y(datum.type) - vy);
      if (distance < bestDistance) {
        bestDistance = distance;
        best = datum;
      }
    }
    if (!best) {
      tooltipVisible = false;
      return;
    }
    tooltipText = `${best.description}\n${best.date}`;
    tooltipVisible = true;
    const card = svg.closest(".scatter-card");
    const cardBox = card?.getBoundingClientRect();
    if (cardBox) {
      tooltipLeft = event.clientX - cardBox.left + 12;
      tooltipTop = event.clientY - cardBox.top + 12;
    }
  }

  function hideTip() { tooltipVisible = false; }
</script>

<section class="scatter-card">
  {#if data.length}
    <svg class="scatter-chart" viewBox="0 0 100 52" role="img" aria-label="Events scatterplot" on:mousemove={showTip} on:mouseleave={hideTip}>
      {#each types as type (type)}
        <line x1={X0} y1={y(type)} x2={X1} y2={y(type)} class="chart-grid" />
        <text x={X0 - 1} y={y(type) + 1} class="chart-tick" text-anchor="end">{typeLabel(type)}</text>
      {/each}
      {#each xTicks as tick (tick.date)}
        <text x={x(tick.timestamp)} y="51" class="chart-tick" text-anchor="middle">{tickLabel(tick.date)}</text>
      {/each}
      {#each data as dot (`${dot.date}-${dot.type}`)}
        <circle
          cx={x(dot.timestamp)}
          cy={y(dot.type)}
          r="1.3"
          style={`fill:${colorForType(types.indexOf(dot.type), types.length)}`}
          class:desaturate={dot.date > today}
        />
      {/each}
    </svg>
  {:else}
    <p class="chart-empty">No events.</p>
  {/if}
  {#if tooltipVisible}
    <div class="chart-tooltip" role="status" style={`left:${tooltipLeft}px;top:${tooltipTop}px`}>{tooltipText}</div>
  {/if}
</section>

<style>
  .scatter-card { position: relative; margin-bottom: 1rem; }
  .scatter-chart { display: block; width: min(100%, 52rem); height: 14rem; margin-bottom: .5rem; background: var(--background-darker); border: 1px solid var(--border); }
  .chart-grid { stroke: var(--border); stroke-width: .2; vector-effect: non-scaling-stroke; opacity: .5; }
  .chart-tick { font-size: 2.6px; fill: var(--text-color-lighter); }
  .desaturate { filter: saturate(50%); }
  .chart-empty { color: var(--text-color-lightest); }
  .chart-tooltip { position: absolute; z-index: 10; pointer-events: none; max-width: 20rem; padding: .3rem .5rem; background: var(--background-darker); border: 1px solid var(--border); color: var(--text-color-lighter); font-size: .8rem; line-height: 1.35; white-space: pre-line; box-shadow: 0 1px 3px rgb(0 0 0 / .25); }
</style>
