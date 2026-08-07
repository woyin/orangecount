<script lang="ts">
  import { translations, type Locale } from "../../translations";
  import DocumentPreview from "./DocumentPreview.svelte";
  import DocumentTable from "./DocumentTable.svelte";
  import type { TableReport } from "./types";

  export let report: TableReport;
  export let locale = "en";

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  interface DocumentRow {
    date: string;
    account: string;
    filename: string;
  }

  $: documents = report.rows.map((row) => ({
    date: typeof row.date === "string" ? row.date : "",
    account: typeof row.account === "string" ? row.account : "",
    filename: typeof row.filename === "string" ? row.filename : "",
  }));

  let selected: DocumentRow | null = null;
</script>

{#if documents.length}
  <div class="documents-layout" class:with-preview={selected != null}>
    <div>
      <DocumentTable {documents} {locale} {selected} onSelect={(doc) => (selected = doc)} />
    </div>
    {#if selected}
      <div class="preview">
        <DocumentPreview filename={selected.filename} {locale} />
      </div>
    {/if}
  </div>
{:else}
  <p>{t("noDocuments")}</p>
{/if}

<style>
  .documents-layout.with-preview {
    display: grid;
    grid-template-columns: 1fr 2fr;
    height: 70vh;
  }

  .documents-layout.with-preview > :global(*) {
    overflow: auto;
  }

  .documents-layout.with-preview .preview {
    border-left: thin solid var(--sidebar-border);
    padding-left: 0.75rem;
  }
</style>

