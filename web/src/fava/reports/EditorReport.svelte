<!-- This file is derived from Fava 1.30.12 (commit #aa7538e8971252c9efc52c8a516a3a77d604553f),
which is Copyright (c) 2015-2016 Dominik Aumayr <dominik@aumayr.name> and
distributed under the MIT License. Adapted for OrangeCount; see NOTICE and
web/provenance-manifest.json. The MIT notice is reproduced here:

  Copyright (c) 2015-2016 Dominik Aumayr <dominik@aumayr.name>

  Permission is hereby granted, free of charge, to any person obtaining a copy
  of this software and associated documentation files (the "Software"), to deal
  in the Software without restriction, including without limitation the rights
  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
  copies of the Software, and to permit persons to whom the Software is
  furnished to do so, subject to the following conditions:

  The above copyright notice and this permission notice shall be included in all
  copies or substantial portions of the Software.

  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
  SOFTWARE. -->

<script lang="ts">
  import { onMount } from "svelte";
  import type { EditorState } from "@codemirror/state";
  import type { EditorView } from "@codemirror/view";
  import type { AdapterClient } from "../adapter-client";
  import { init_beancount_editor, replace_contents } from "../codemirror/beancount";
  import { set_completion_data } from "../codemirror/completion-data";
  import { notify, notify_err } from "../notifications";
  import EditorMenu from "./editor/EditorMenu.svelte";
  import { editor_sources, sources_tree } from "./editor/stores";
  export let adapter: AdapterClient;
  export let query: Record<string, string> = {};

  let paths: string[] = [];
  let selected = "";
  let content = "";
  let snapshotID = "";
  let status = "";
  let diagnostics: any[] = [];
  let loading = true;
  let editorHost: HTMLDivElement;
  let editor: EditorView | null = null;

  function onDocChanges(state: EditorState) {
    content = state.sliceDoc();
  }

  function showInEditor(value: string) {
    if (editor) {
      editor.dispatch(replace_contents(editor.state, value));
      return;
    }
    editor = init_beancount_editor(value, onDocChanges, [], 2, 0);
    editorHost.appendChild(editor.dom);
  }

  async function loadIndex() {
    const value = await adapter.load("editor") as { paths: string[]; entry: string; snapshot_id: string };
    paths = value.paths ?? [];
    editor_sources.set(new Set(paths));
    snapshotID = value.snapshot_id ?? "";
    const requested = query.path && paths.includes(query.path) ? query.path : "";
    selected = requested || value.entry || paths[0] || "";
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
      showInEditor(value.content);
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
        notify("File saved.");
      } else {
        status = "Save rejected; the previous snapshot remains active.";
        notify(status, "warning");
      }
    } catch (value) {
      status = value instanceof Error ? value.message : "Save failed.";
      notify_err(value, (error) => `Saving failed: ${error.message}`);
    }
  }

  async function openFile(path: string) {
    if (path === selected) return;
    selected = path;
    await loadFile();
  }

  onMount(() => {
    void loadIndex();
    void adapter.bootstrap().then((bootstrap) => {
      set_completion_data({
        accounts: bootstrap.accounts,
        currencies: bootstrap.currencies,
        payees: bootstrap.payees,
        tags: bootstrap.tags,
        links: bootstrap.links,
      });
    }).catch(() => {});
    return () => editor?.destroy();
  });
</script>

<div class="headerline"><h2>Editor</h2><span class="muted">Reviewed writes only</span></div>
<EditorMenu sources_tree={$sources_tree} file_path={selected} {editor} on_open_file={openFile}>
  <button id="editor-validate" type="button" on:click={validate} disabled={loading}>Validate</button>
  <button id="editor-save" type="button" on:click={save} disabled={loading}>Save</button>
  <span class="muted" role="status">{status}</span>
</EditorMenu>
<div class="editor-layout">
  <aside class="editor-files">
    <label for="editor-file">Files</label>
    <select id="editor-file" size={Math.min(Math.max(paths.length, 2), 12)} bind:value={selected} on:change={loadFile}>
      {#each paths as path (path)}<option value={path}>{path}</option>{/each}
    </select>
  </aside>
  <section class="editor-pane">
    <div id="editor-buffer" bind:this={editorHost} aria-label="Ledger source"></div>
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
  #editor-buffer :global(.cm-editor) { min-height: 28rem; border: 1px solid var(--border); }
  #editor-buffer :global(.cm-scroller) { font-family: var(--font-family-editor); }
  .diagnostics { padding: 0.5rem 1.5rem; color: var(--error); border: 1px solid var(--error); }
  @media (width <= 767px) { .editor-layout { grid-template-columns: 1fr; } .editor-files select { min-height: 6rem; } }
</style>