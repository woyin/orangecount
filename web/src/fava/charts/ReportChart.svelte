<script lang="ts">
  import { formatAmount, type ReportChart } from "../reports/types";

  export let chart: ReportChart;

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

  function x(index: number): number { return width === 1 ? 50 : (index / (width - 1)) * 96 + 2; }
  function y(value: number): number { return 48 - ((value - min) / range) * 44; }
  function linePath(points: { value: { display: string } }[]): string {
    return points.map((point, index) => `${index ? "L" : "M"}${x(index).toFixed(2)},${y(numberValue(point.value)).toFixed(2)}`).join(" ");
  }
  function barHeight(value: number): number { return Math.max(0.5, Math.abs(y(value) - y(0))); }
  function barY(value: number): number { return value >= 0 ? y(value) : y(0); }
</script>

<section class="chart-card" aria-label={chart.title}>
  <h3>{chart.title}</h3>
  {#if chart.kind === "hierarchy" && chart.nodes?.length}
    <svg class="report-chart report-hierarchy-chart" viewBox="0 0 100 52" role="img" aria-label={chart.title}>
      {#each chart.nodes.slice(0, 24) as node, index (node.name + index)}
        {@const nodeWidth = 96 / Math.min(chart.nodes.length, 24)}
        <rect x={2 + index * nodeWidth} y={8 + node.depth * 5} width={Math.max(0.5, nodeWidth - 0.5)} height={Math.max(2, 40 - node.depth * 5)} style={`fill:${colors[index % colors.length]}`} opacity=".75" />
      {/each}
    </svg>
  {:else if chart.kind === "stacked-bar" || chart.kind === "bar"}
    <svg class="report-chart report-bar-chart" viewBox="0 0 100 52" role="img" aria-label={chart.title}>
      <line x1="2" y1={y(0)} x2="98" y2={y(0)} class="chart-axis" />
      {#each chart.series as series, seriesIndex (series.label)}
        {#each series.points as point, index (point.date)}
          {@const value = numberValue(point.value)}
          {@const barWidth = Math.max(1, 90 / Math.max(1, width) / Math.max(1, chart.series.length))}
          <rect x={x(index) - 45 / Math.max(1, width) + seriesIndex * barWidth} y={barY(value)} width={barWidth - .25} height={barHeight(value)} style={`fill:${colors[seriesIndex % colors.length]}`} />
        {/each}
      {/each}
    </svg>
  {:else}
    <svg class="report-chart report-line-chart" viewBox="0 0 100 52" role="img" aria-label={chart.title}>
      <line x1="2" y1={y(0)} x2="98" y2={y(0)} class="chart-axis" />
      {#each chart.series as series, index (series.label)}
        <path d={linePath(series.points)} style={`stroke:${colors[index % colors.length]}`} />
      {/each}
    </svg>
  {/if}
  <p class="chart-meta">{chart.interval} · {chart.valuation}{chart.currency ? ` · ${chart.currency}` : ""}</p>
  {#each chart.series as series, index (series.label)}
    <span class="legend"><i style={`background:${colors[index % colors.length]}`}></i>{series.label}</span>
  {/each}
  {#if chart.availability}
    <p class="chart-availability">{chart.availability}</p>
  {/if}
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
</section>

<style>
  .chart-card { margin-bottom: 1rem; }
  .chart-meta, .chart-availability { color: var(--text-color-lightest); }
  .chart-scroll { overflow-x: auto; }
  .report-chart { display: block; width: min(100%, 52rem); height: 14rem; margin-bottom: .5rem; background: var(--background-darker); border: 1px solid var(--border); }
  .report-chart path { fill: none; stroke-width: .8; vector-effect: non-scaling-stroke; }
  .chart-axis { stroke: var(--chart-axis); stroke-width: .25; vector-effect: non-scaling-stroke; }
  .legend { display: inline-flex; gap: .3rem; align-items: center; margin: 0 .75rem .5rem 0; }
  .legend i { display: inline-block; width: .7rem; height: .7rem; border-radius: 50%; }
  table { min-width: 32rem; }
</style>
