<script lang="ts">
  import ReportChart from "../charts/ReportChart.svelte";
  import { formatAmount, type TableReport } from "./types";

  export let report: TableReport;
  export let title: string;
  export let route = "";
  export let locale = "en";
  export let renderCommas = false;

  function display(value: unknown): string {
    if (value && typeof value === "object" && "display" in value && typeof value.display === "string") {
      return formatAmount(value as { display: string; exact: string; approximate: boolean }, renderCommas);
    }
    if (Array.isArray(value)) return value.join(", ");
    if (value && typeof value === "object") return JSON.stringify(value);
    return value == null ? "" : String(value);
  }

  function isNumberLike(value: unknown): boolean {
    return typeof value === "number" || (typeof value === "object" && value !== null && "display" in value);
  }

  function numericValue(value: unknown): number {
    if (typeof value === "number") return value;
    if (value && typeof value === "object" && "display" in value && typeof value.display === "string") {
      const parsed = Number(value.display.replace(/,/g, ""));
      return Number.isFinite(parsed) ? parsed : 0;
    }
    return 0;
  }

  let sortColumn = "";
  let sortDirection: "ascending" | "descending" | null = null;

  function toggleSort(column: string) {
    if (sortColumn !== column) {
      sortColumn = column;
      sortDirection = "ascending";
    } else if (sortDirection === "ascending") {
      sortDirection = "descending";
    } else {
      sortColumn = "";
      sortDirection = null;
    }
  }

  $: sortedRows = (() => {
    if (!sortDirection) return report.rows;
    const factor = sortDirection === "ascending" ? 1 : -1;
    return [...report.rows].sort((a, b) => {
      const left = a[sortColumn];
      const right = b[sortColumn];
      if (isNumberLike(left) || isNumberLike(right)) {
        return factor * (numericValue(left) - numericValue(right));
      }
      return factor * display(left).localeCompare(display(right));
    });
  })();
</script>

<div class="headerline">
  <h2>{title}</h2>
  <span class="muted">{report.rows.length} rows</span>
  {#if route}<a class="button" href={`/api/v1/reports/${route}?format=csv`}>Export CSV</a>{/if}
</div>
{#if report.chart}
  <ReportChart chart={report.chart} {locale} />
{/if}
<div class="table-scroll">
  <table class="report-table">
    <thead>
      <tr>
        {#each report.columns as column (column)}
          <th
            scope="col"
            class:num={report.rows.some((row) => isNumberLike(row[column]))}
            aria-sort={sortColumn === column ? sortDirection : undefined}
          >
            <button type="button" class="sort-toggle" onclick={() => toggleSort(column)}>
              {column}{#if sortColumn === column}<span aria-hidden="true">{sortDirection === "ascending" ? " ▲" : " ▼"}</span>{/if}
            </button>
          </th>
        {/each}
      </tr>
    </thead>
    <tbody>
      {#each sortedRows as row, index (index)}
        <tr>
          {#each report.columns as column (column)}
            <td class:num={isNumberLike(row[column])}>
              {#if route === "documents" && column === "filename" && typeof row[column] === "string"}
                <a href={`/documents/${encodeURIComponent(row[column] as string)}`}>{display(row[column])}</a>
              {:else if ["file", "path"].includes(column) && typeof row[column] === "string"}
                <a href={`/source?path=${encodeURIComponent(row[column] as string)}`}>{display(row[column])}</a>
              {:else}
                {display(row[column])}
              {/if}
            </td>
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

  .report-table .sort-toggle {
    padding: 0;
    font: inherit;
    color: inherit;
    cursor: pointer;
    background: transparent;
    border: 0;
  }
</style>
