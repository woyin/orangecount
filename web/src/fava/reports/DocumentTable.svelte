<script lang="ts">
  import { translations, type Locale } from "../../translations";
  import { DateColumn, Sorter, StringColumn, type SortColumn } from "../sort/index";
  import SortHeader from "../sort/SortHeader.svelte";

  interface DocumentRow {
    date: string;
    account: string;
    filename: string;
  }

  export let documents: DocumentRow[];
  export let locale = "en";

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  // Fava strips a leading date from document basenames
  // ("2025-01-01-invoice.pdf" -> "invoice.pdf").
  function displayName(doc: DocumentRow): string {
    const base = doc.filename.split("/").pop() ?? doc.filename;
    return base.startsWith(doc.date) ? base.slice(doc.date.length + 1) : base;
  }

  $: columns = [
    new DateColumn<DocumentRow>(t("date")),
    new StringColumn<DocumentRow>(t("account"), (doc) => doc.account),
    new StringColumn<DocumentRow>(t("name"), displayName),
  ] as SortColumn<DocumentRow>[];
  $: sorter = new Sorter<DocumentRow>(columns[0], "desc");
  $: sorted = sorter.sort(documents);
</script>

<table class="documents-table">
  <thead>
    <tr>
      {#each columns as column (column.name)}
        <SortHeader bind:sorter {column} />
      {/each}
    </tr>
  </thead>
  <tbody>
    {#each sorted as doc (doc.filename)}
      <tr title={doc.filename}>
        <td>{doc.date}</td>
        <td>{doc.account}</td>
        <td>{displayName(doc)}</td>
      </tr>
    {/each}
  </tbody>
</table>

<style>
  .documents-table {
    max-width: 60rem;
  }
</style>
