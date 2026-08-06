<script lang="ts">
  import { translations, type Locale } from "../../translations";
  import type { TableReport } from "./types";

  export let report: TableReport;
  export let locale = "en";

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  // Fava strips a leading date from document basenames
  // ("2025-01-01-invoice.pdf" -> "invoice.pdf") and sorts by date descending.
  function displayName(date: string, filename: string): string {
    const base = filename.split("/").pop() ?? filename;
    return base.startsWith(date) ? base.slice(date.length + 1) : base;
  }

  $: documents = report.rows
    .map((row) => ({
      date: typeof row.date === "string" ? row.date : "",
      account: typeof row.account === "string" ? row.account : "",
      filename: typeof row.filename === "string" ? row.filename : "",
    }))
    .sort((a, b) => b.date.localeCompare(a.date) || a.account.localeCompare(b.account));
</script>

{#if documents.length}
  <table class="documents-table">
    <thead>
      <tr><th>{t("date")}</th><th>{t("account")}</th><th>{t("name")}</th></tr>
    </thead>
    <tbody>
      {#each documents as doc (doc.filename)}
        <tr title={doc.filename}>
          <td>{doc.date}</td>
          <td>{doc.account}</td>
          <td>{displayName(doc.date, doc.filename)}</td>
        </tr>
      {/each}
    </tbody>
  </table>
{:else}
  <p>{t("noDocuments")}</p>
{/if}

<style>
  .documents-table {
    max-width: 60rem;
  }
</style>
