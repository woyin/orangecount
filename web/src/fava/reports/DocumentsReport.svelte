<script lang="ts">
  import { translations, type Locale } from "../../translations";
  import DocumentTable from "./DocumentTable.svelte";
  import type { TableReport } from "./types";

  export let report: TableReport;
  export let locale = "en";

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  $: documents = report.rows.map((row) => ({
    date: typeof row.date === "string" ? row.date : "",
    account: typeof row.account === "string" ? row.account : "",
    filename: typeof row.filename === "string" ? row.filename : "",
  }));
</script>

{#if documents.length}
  <DocumentTable {documents} {locale} />
{:else}
  <p>{t("noDocuments")}</p>
{/if}

