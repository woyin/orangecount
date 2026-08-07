<script lang="ts">
  import { translations, type Locale } from "../../translations";
  import type { AdapterClient } from "../adapter-client";
  import { notify } from "../notifications";
  import DocumentAccounts from "./DocumentAccounts.svelte";
  import DocumentMoveModal from "../modals/DocumentMoveModal.svelte";
  import DocumentPreview from "./DocumentPreview.svelte";
  import DocumentTable from "./DocumentTable.svelte";
  import { parseTableReport, type TableReport } from "./types";

  export let report: TableReport;
  export let locale = "en";
  export let adapter: AdapterClient | null = null;
  export let query: Record<string, string> = {};
  export let accounts: string[] = [];

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  interface DocumentRow {
    date: string;
    account: string;
    filename: string;
  }

  interface AccountNode {
    name: string;
    count: number;
    children: AccountNode[];
  }

  interface MoveDetails {
    account: string;
    filename: string;
    newName: string;
  }

  $: documents = report.rows.map((row) => ({
    date: typeof row.date === "string" ? row.date : "",
    account: typeof row.account === "string" ? row.account : "",
    filename: typeof row.filename === "string" ? row.filename : "",
  }));

  // Mirrors upstream's stratifyAccounts over documents grouped by account:
  // implicit intermediate accounts are inserted so every document account
  // hangs under its full ancestor chain.
  $: tree = buildTree(documents);

  function buildTree(rows: DocumentRow[]): AccountNode {
    const counts = new Map<string, number>();
    for (const row of rows) counts.set(row.account, (counts.get(row.account) ?? 0) + 1);
    const root: AccountNode = { name: "", count: 0, children: [] };
    const map = new Map<string, AccountNode>([["", root]]);
    const addNode = (name: string): AccountNode => {
      const existing = map.get(name);
      if (existing) return existing;
      const node: AccountNode = { name, count: counts.get(name) ?? 0, children: [] };
      map.set(name, node);
      const parentName = name.slice(0, Math.max(0, name.lastIndexOf(":")));
      const parent = map.get(parentName) ?? addNode(parentName);
      parent.children.push(node);
      return node;
    };
    [...counts.keys()].sort((a, b) => a.localeCompare(b)).forEach(addNode);
    return root;
  }

  let selected: DocumentRow | null = null;
  let accountFilter = "";
  let toggled = new Set<string>();
  let moving: MoveDetails | null = null;

  function basename(path: string): string {
    return path.split("/").pop() ?? path;
  }

  function onSelectAccount(account: string) {
    accountFilter = account;
    selected = null;
  }

  // Upstream opens the move/rename dialog with <F2> while a row is selected.
  function onKeyup(event: KeyboardEvent) {
    if (event.key === "F2" && selected && !moving) {
      moving = { account: selected.account, filename: selected.filename, newName: basename(selected.filename) };
    }
  }

  function onStartMove(detail: { account: string; filename: string }) {
    moving = { ...detail, newName: basename(detail.filename) };
  }

  async function refresh() {
    if (!adapter) return;
    try {
      const payload = await adapter.load("documents", query);
      report = parseTableReport(payload);
    } catch {
      notify("The documents list could not be reloaded.");
    }
  }

  function onMoved(message: string) {
    notify(message);
    selected = null;
    void refresh();
  }

  $: visible = accountFilter
    ? documents.filter((doc) => doc.account === accountFilter || doc.account.startsWith(`${accountFilter}:`))
    : documents;
</script>

<svelte:window on:keyup={onKeyup} />
<DocumentMoveModal {locale} {accounts} {moving} onMoved={onMoved} onClose={() => (moving = null)} />

{#if documents.length}
  <div class="documents-layout" class:with-preview={selected != null}>
    <div class="accounts">
      {#each tree.children as child (child.name)}
        <DocumentAccounts node={child} bind:toggled selectedAccount={accountFilter} on:select={(event) => onSelectAccount(event.detail)} on:move={(event) => onStartMove(event.detail)} />
      {/each}
    </div>
    <div>
      <DocumentTable documents={visible} {locale} {selected} onSelect={(doc) => (selected = doc)} />
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
  .documents-layout {
    display: grid;
    grid-template-columns: 1fr 2fr;
    height: 70vh;
  }

  .documents-layout.with-preview {
    grid-template-columns: 1fr 2fr 3fr;
  }

  .documents-layout > :global(*) {
    overflow: auto;
  }

  .documents-layout > :global(* + *) {
    border-left: thin solid var(--sidebar-border);
    padding-left: 0.75rem;
  }
</style>
