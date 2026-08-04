<script lang="ts">
  import ReportChart from "../charts/ReportChart.svelte";
  import { formatAmount, type TableReport } from "./types";

  export let report: TableReport;
  export let title: string;

  function display(value: unknown): string {
    if (value && typeof value === "object" && "display" in value && typeof value.display === "string") {
      return formatAmount(value as { display: string; exact: string; approximate: boolean });
    }
    if (Array.isArray(value)) return value.join(", ");
    if (value && typeof value === "object") return JSON.stringify(value);
    return value == null ? "" : String(value);
  }

  function isNumberLike(value: unknown): boolean {
    return typeof value === "number" || (typeof value === "object" && value !== null && "display" in value);
  }
</script>

<div class="headerline">
  <h2>{title}</h2>
  <span class="muted">{report.rows.length} rows</span>
</div>
{#if report.chart}
  <ReportChart chart={report.chart} />
{/if}
<div class="table-scroll">
  <table class="report-table">
    <thead>
      <tr>
        {#each report.columns as column (column)}
          <th scope="col" class:num={report.rows.some((row) => isNumberLike(row[column]))}>{column}</th>
        {/each}
      </tr>
    </thead>
    <tbody>
      {#each report.rows as row, index (index)}
        <tr>
          {#each report.columns as column (column)}
            <td class:num={isNumberLike(row[column])}>{display(row[column])}</td>
          {/each}
        </tr>
      {:else}
        <tr><td colspan={report.columns.length}>No rows.</td></tr>
      {/each}
    </tbody>
  </table>
</div>

<style>
  .muted {
    color: var(--text-color-lightest);
  }

  .table-scroll {
    max-width: 100%;
    overflow-x: auto;
  }

  .report-table {
    width: max-content;
    min-width: 100%;
  }

  .report-table .num {
    text-align: right;
  }
</style>
