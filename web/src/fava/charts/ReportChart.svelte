<script lang="ts">
  import { formatAmount, type ReportChart } from "../reports/types";

  export let chart: ReportChart;
</script>

<section class="chart-card" aria-label={chart.title}>
  <h3>{chart.title}</h3>
  <p class="chart-meta">{chart.interval} · {chart.valuation}{chart.currency ? ` · ${chart.currency}` : ""}</p>
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
        {/each}
      </tbody>
    </table>
  </div>
  {#if chart.availability}
    <p class="chart-availability">{chart.availability}</p>
  {/if}
</section>

<style>
  .chart-card {
    margin-bottom: 1rem;
  }

  .chart-meta,
  .chart-availability {
    color: var(--text-color-lightest);
  }

  .chart-scroll {
    overflow-x: auto;
  }

  table {
    min-width: 32rem;
  }
</style>
