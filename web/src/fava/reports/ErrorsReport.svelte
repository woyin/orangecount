<script lang="ts">
  import { translations, type Locale } from "../../translations";
  import type { TableReport } from "./types";

  export let report: TableReport;
  export let locale = "en";

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  interface ErrorRow {
    code: string;
    severity: string;
    path: string;
    line: number;
    message: string;
  }

  $: errors = report.rows.map((row): ErrorRow => ({
    code: typeof row.code === "string" ? row.code : "",
    severity: typeof row.severity === "string" ? row.severity : "",
    path: typeof row.path === "string" ? row.path : "",
    line: typeof row.line === "number" ? row.line : 0,
    message: typeof row.message === "string" ? row.message : String(row.message ?? ""),
  }));

  // Fava sorts the errors table by file, newest first, and allows cycling the
  // direction; the generic three-state toggle mirrors the sortable tables.
  let sortColumn = "path";
  let sortDirection: "ascending" | "descending" | null = "descending";

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

  function sortValue(row: ErrorRow): string | number {
    if (sortColumn === "line") return row.line;
    if (sortColumn === "message") return row.message;
    return row.path;
  }

  $: sortedRows = (() => {
    if (!sortDirection) return errors;
    const factor = sortDirection === "ascending" ? 1 : -1;
    return [...errors].sort((a, b) => {
      const left = sortValue(a);
      const right = sortValue(b);
      if (typeof left === "number" && typeof right === "number") return factor * (left - right);
      return factor * String(left).localeCompare(String(right));
    });
  })();
</script>

{#if errors.length}
  <table class="errors-table">
    <thead>
      <tr>
        <th aria-sort={sortColumn === "path" ? sortDirection : undefined}>
          <button type="button" class="sort-toggle" onclick={() => toggleSort("path")}>
            {t("file")}{#if sortColumn === "path"}<span aria-hidden="true">{sortDirection === "ascending" ? " ▲" : " ▼"}</span>{/if}
          </button>
        </th>
        <th class="num" aria-sort={sortColumn === "line" ? sortDirection : undefined}>
          <button type="button" class="sort-toggle" onclick={() => toggleSort("line")}>
            {t("line")}{#if sortColumn === "line"}<span aria-hidden="true">{sortDirection === "ascending" ? " ▲" : " ▼"}</span>{/if}
          </button>
        </th>
        <th aria-sort={sortColumn === "message" ? sortDirection : undefined}>
          <button type="button" class="sort-toggle" onclick={() => toggleSort("message")}>
            {t("error")}{#if sortColumn === "message"}<span aria-hidden="true">{sortDirection === "ascending" ? " ▲" : " ▼"}</span>{/if}
          </button>
        </th>
      </tr>
    </thead>
    <tbody>
      {#each sortedRows as error, index (index)}
        <tr class={error.severity}>
          {#if error.path}
            <td><a href={`/source?path=${encodeURIComponent(error.path)}`}>{error.path}</a></td>
            <td class="num"><a href={`/source?path=${encodeURIComponent(error.path)}`}>{error.line}</a></td>
          {:else}
            <td></td>
            <td class="num"></td>
          {/if}
          <td class="pre">{#if error.code}<span class="code">{error.code}</span> {/if}{error.message}</td>
        </tr>
      {/each}
    </tbody>
  </table>
{:else}
  <p>{t("noErrors")}</p>
{/if}

<style>
  .errors-table {
    min-width: 100%;
  }

  .errors-table .num {
    text-align: right;
  }

  .errors-table .pre {
    white-space: pre-wrap;
    font-family: var(--font-family-monospace, monospace);
  }

  .errors-table tr.warning td {
    color: var(--text-color-light);
  }

  .errors-table .code {
    color: var(--text-color-lightest);
  }

  .sort-toggle {
    padding: 0;
    font: inherit;
    color: inherit;
    cursor: pointer;
    background: transparent;
    border: 0;
  }
</style>
