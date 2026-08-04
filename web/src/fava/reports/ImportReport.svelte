<script lang="ts">
  import { onMount } from "svelte";
  import type { AdapterClient } from "../adapter-client";

  export let adapter: AdapterClient;

  let paths: string[] = [];
  let target = "";
  let importPath = "import.bean";
  let adapterID = "beancount";
  let content = "";
  let snapshotID = "";
  let previewID = "";
  let valid = false;
  let diagnostics: any[] = [];
  let rows: Record<string, unknown>[] = [];
  let status = "";

  async function loadTargets() {
    const value = await adapter.load("import") as { paths: string[]; entry: string; snapshot_id: string };
    paths = value.paths ?? [];
    target = value.entry || paths[0] || "";
    snapshotID = value.snapshot_id ?? "";
  }

  async function request(path: string, body: unknown) {
    const response = await fetch(`/api/v1/import/${path}`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
    const value = await response.json() as any;
    if (!response.ok) throw new Error(value.error || `Import request failed (${response.status})`);
    return value;
  }

  async function preview() {
    status = "Previewing…";
    previewID = "";
    try {
      const value = await request("preview", { path: importPath, content, adapter: adapterID, mapping: {} });
      previewID = value.preview_id ?? "";
      valid = Boolean(value.valid);
      diagnostics = value.diagnostics ?? [];
      rows = value.rows?.rows ?? [];
      status = valid ? `Preview ready: ${previewID}` : "Preview has diagnostics";
    } catch (value) {
      status = value instanceof Error ? value.message : "Preview failed.";
    }
  }

  async function commit() {
    if (!previewID) return;
    status = "Committing…";
    try {
      const value = await request("commit", { preview_id: previewID, target, expected_snapshot_id: snapshotID });
      snapshotID = value.snapshot_id ?? snapshotID;
      status = `Committed; backup: ${value.backup}`;
      previewID = "";
    } catch (value) {
      status = value instanceof Error ? value.message : "Commit failed.";
    }
  }

  onMount(() => { void loadTargets(); });
</script>

<div class="headerline"><h2>Import</h2><span class="muted">Preview before commit</span></div>
<div class="toolbar">
  <label>Source path <input bind:value={importPath}></label>
  <label>Adapter <select bind:value={adapterID}><option value="beancount">Beancount</option><option value="csv">CSV</option></select></label>
  <label>Target <select bind:value={target}>{#each paths as path (path)}<option value={path}>{path}</option>{/each}</select></label>
</div>
<textarea class="import-buffer" bind:value={content} placeholder="Paste Beancount or CSV content" spellcheck="false"></textarea>
<div class="toolbar">
  <button type="button" on:click={preview}>Preview</button>
  <button type="button" on:click={commit} disabled={!previewID || !valid}>Commit</button>
  <span class="muted" role="status">{status}</span>
</div>
{#if diagnostics.length}
  <ul class="diagnostics">{#each diagnostics as diagnostic (diagnostic.code + diagnostic.line + diagnostic.message)}<li>{diagnostic.path}:{diagnostic.line}: {diagnostic.message}</li>{/each}</ul>
{/if}
{#if rows.length}
  <table><thead><tr><th>Date</th><th>Account</th><th>Units</th><th>Currency</th></tr></thead><tbody>
    {#each rows as row, index (index)}<tr><td>{String(row.date ?? "")}</td><td>{String(row.account ?? "")}</td><td>{String(row.units ?? "")}</td><td>{String(row.currency ?? "")}</td></tr>{/each}
  </tbody></table>
{/if}

<style>
  .muted { color: var(--text-color-lightest); }
  .toolbar { display:flex; flex-wrap:wrap; gap:.5rem; align-items:center; margin-bottom:.75rem; }
  .import-buffer { width:100%; min-height:18rem; font-family:var(--font-family-editor); }
  .diagnostics { padding:.5rem 1.5rem; color:var(--error); border:1px solid var(--error); }
</style>
