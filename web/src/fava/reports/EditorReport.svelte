<script lang="ts">
  import { onMount } from "svelte";
  import type { AdapterClient } from "../adapter-client";

  export let adapter: AdapterClient;

  let paths: string[] = [];
  let selected = "";
  let content = "";
  let snapshotID = "";
  let status = "";
  let diagnostics: any[] = [];
  let loading = true;

  async function loadIndex() {
    const value = await adapter.load("editor") as { paths: string[]; entry: string; snapshot_id: string };
    paths = value.paths ?? [];
    snapshotID = value.snapshot_id ?? "";
    selected = value.entry || paths[0] || "";
    if (selected) await loadFile();
  }

  async function loadFile() {
    if (!selected) return;
    loading = true;
    try {
      const value = await adapter.load("editor", { path: selected }) as { path: string; content: string; snapshot_id: string };
      content = value.content;
      snapshotID = value.snapshot_id;
      diagnostics = [];
      status = "";
    } catch (value) {
      status = value instanceof Error ? value.message : "Unable to load source file.";
    } finally {
      loading = false;
    }
  }

  async function send(path: string, body: unknown) {
    const response = await fetch(`/api/v1/editor/${path}`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
    const value = await response.json() as any;
    if (!response.ok) throw new Error(value.error || `Editor request failed (${response.status})`);
    return value;
  }

  async function validate() {
    status = "Validating…";
    try {
      const value = await send("validate", { path: selected, content, expected_snapshot_id: snapshotID });
      diagnostics = value.diagnostics ?? [];
      status = value.valid ? "Valid" : "Diagnostics found";
    } catch (value) {
      status = value instanceof Error ? value.message : "Validation failed.";
    }
  }

  async function save() {
    status = "Saving…";
    try {
      const value = await send("save", { path: selected, content, expected_snapshot_id: snapshotID });
      diagnostics = value.diagnostics ?? [];
      if (value.published) {
        snapshotID = value.snapshot_id;
        status = `Saved; backup: ${value.backup}`;
      } else {
        status = "Save rejected; the previous snapshot remains active.";
      }
    } catch (value) {
      status = value instanceof Error ? value.message : "Save failed.";
    }
  }

  onMount(() => { void loadIndex(); });
</script>

<div class="headerline"><h2>Editor</h2><span class="muted">Reviewed writes only</span></div>
<div class="editor-layout">
  <aside class="editor-files">
    <label for="editor-file">Files</label>
    <select id="editor-file" size={Math.min(Math.max(paths.length, 2), 12)} bind:value={selected} on:change={loadFile}>
      {#each paths as path (path)}<option value={path}>{path}</option>{/each}
    </select>
  </aside>
  <section class="editor-pane">
    <div class="toolbar">
      <button type="button" on:click={validate} disabled={loading}>Validate</button>
      <button type="button" on:click={save} disabled={loading}>Save</button>
      <span class="muted" role="status">{status}</span>
    </div>
    <textarea id="editor-buffer" bind:value={content} spellcheck="false" aria-label="Ledger source"></textarea>
    {#if diagnostics.length}
      <ul class="diagnostics">
        {#each diagnostics as diagnostic (diagnostic.code + diagnostic.line + diagnostic.message)}
          <li>{diagnostic.path}:{diagnostic.line}: {diagnostic.message}</li>
        {/each}
      </ul>
    {/if}
  </section>
</div>

<style>
  .muted { color: var(--text-color-lightest); }
  .editor-layout { display: grid; grid-template-columns: minmax(10rem, 16rem) minmax(0, 1fr); gap: 1rem; }
  .editor-files { display: grid; gap: 0.5rem; align-content: start; }
  .editor-files select { min-height: 16rem; }
  .editor-pane { min-width: 0; }
  .toolbar { display: flex; flex-wrap: wrap; gap: 0.5rem; align-items: center; margin-bottom: 0.5rem; }
  #editor-buffer { width: 100%; min-height: 28rem; font-family: var(--font-family-editor); white-space: pre; }
  .diagnostics { padding: 0.5rem 1.5rem; color: var(--error); border: 1px solid var(--error); }
  @media (width <= 767px) { .editor-layout { grid-template-columns: 1fr; } .editor-files select { min-height: 6rem; } }
</style>
