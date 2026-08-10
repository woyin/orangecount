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
  import { PRIVATE_ADAPTER_BASE } from "../adapter-client";
  import { notify } from "../notifications";
  import { translations, type Locale } from "../../translations";

  export let locale = "en";
  export let documentRoots: string[] = [];
  export let accounts: string[] = [];
  export let onUploaded: () => void = () => {};

  let account = "";
  let files: FileList | null = null;
  let overrides: Record<number, string> = {};
  let folder = "";
  let saving = false;
  let error = "";

  $: if (documentRoots.length && (folder === "" || !documentRoots.includes(folder))) {
    folder = documentRoots[0];
  }
  $: shown = files != null;

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  function today(): string {
    const now = new Date();
    const month = `${now.getMonth() + 1}`.padStart(2, "0");
    const day = `${now.getDate()}`.padStart(2, "0");
    return `${now.getFullYear()}-${month}-${day}`;
  }

  // Upstream prefixes names without a leading date with the entry date (or
  // today) so documents sort chronologically inside the account folder.
  function getName(file: File, index: number): string {
    const override = overrides[index];
    if (override !== undefined && override.trim() !== "") return override.trim();
    return /^\d{4}-\d{2}-\d{2}/.test(file.name) ? file.name : `${today()} ${file.name}`;
  }

  function setOverride(index: number, value: string) {
    overrides = { ...overrides, [index]: value };
  }

  function supported(transfer: DataTransfer | null): boolean {
    return transfer != null && transfer.types.includes("Files");
  }

  function closestDragTarget(event: DragEvent, selector: string): Element | null {
    return event.target instanceof Element ? event.target.closest(selector) : null;
  }

  function onDragEnter(event: DragEvent) {
    if (!supported(event.dataTransfer)) return;
    const droptarget = closestDragTarget(event, ".droptarget");
    if (droptarget) {
      droptarget.classList.add("dragover");
      event.preventDefault();
    }
  }

  function onDragOver(event: DragEvent) {
    if (closestDragTarget(event, ".dragover")) event.preventDefault();
  }

  function onDragLeave(event: DragEvent) {
    closestDragTarget(event, ".dragover")?.classList.remove("dragover");
  }

  function onDrop(event: DragEvent) {
    const droptarget = closestDragTarget(event, ".dragover");
    const transfer = event.dataTransfer;
    if (!droptarget || !transfer) return;
    droptarget.classList.remove("dragover");
    event.preventDefault();
    if (!transfer.types.includes("Files")) return;
    account = droptarget.getAttribute("data-account-name") || "";
    overrides = {};
    error = "";
    files = transfer.files;
  }

  function close() {
    account = "";
    files = null;
    overrides = {};
    error = "";
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.key === "Escape" && shown) close();
  }

  async function submit(event: SubmitEvent) {
    event.preventDefault();
    if (!files || saving) return;
    saving = true;
    error = "";
    const failures: string[] = [];
    for (const [index, file] of Array.from(files).entries()) {
      const name = getName(file, index);
      const body = new FormData();
      body.set("account", account);
      if (folder) body.set("folder", folder);
      body.set("file", file, name);
      try {
        const response = await fetch(`${PRIVATE_ADAPTER_BASE}/document`, { method: "POST", body });
        const payload = (await response.json()) as { error?: string; data?: { message?: string } };
        if (!response.ok) {
          throw new Error(payload.error || `Uploading ${name} failed (${response.status})`);
        }
        notify(payload.data?.message || `Uploaded ${name}.`);
      } catch (value) {
        failures.push(value instanceof Error ? value.message : `Uploading ${name} failed.`);
      }
    }
    saving = false;
    if (failures.length) {
      error = failures.join(" ");
      return;
    }
    close();
    onUploaded();
  }
</script>

<svelte:document on:dragenter={onDragEnter} on:dragover={onDragOver} on:dragleave={onDragLeave} on:drop={onDrop} on:keydown={onKeydown} />

{#if shown}
  <div class="upload-backdrop" role="presentation" on:click={close}>
    <form
      class="upload-modal"
      role="dialog"
      aria-modal="true"
      aria-label={t("uploadFiles")}
      on:click|stopPropagation
      on:submit={submit}
    >
      <h3>{t("uploadFiles")}</h3>
      {#if error}
        <p class="error" role="alert">{error}</p>
      {/if}
      <div class="field">
        <span>{t("files")}</span>
        <input type="file" multiple bind:files />
      </div>
      {#if files}
        {#each Array.from(files) as file, index (index)}
          <input
            class="file"
            type="text"
            value={getName(file, index)}
            on:input={(event) => setOverride(index, event.currentTarget.value)}
          />
        {/each}
      {/if}
      {#if documentRoots.length}
        <div class="field">
          <span>{t("documentsFolder")}</span>
          <select bind:value={folder}>
            {#each documentRoots as root (root)}
              <option value={root}>{root}</option>
            {/each}
          </select>
        </div>
      {:else}
        <p class="hint">{t("noDocumentRoot")}</p>
      {/if}
      <div class="field">
        <span>{t("account")}</span>
        <input type="text" bind:value={account} list="upload-accounts" placeholder="Assets:Cash" required autocomplete="off" />
        <datalist id="upload-accounts">
          {#each accounts as name (name)}
            <option value={name}></option>
          {/each}
        </datalist>
      </div>
      <div class="actions">
        <span class="spacer"></span>
        <button type="submit" disabled={saving || !files || !files.length}>{t("upload")}</button>
      </div>
    </form>
  </div>
{/if}

<style>
  .upload-backdrop {
    position: fixed;
    inset: 0;
    z-index: var(--z-index-floating-ui);
    display: flex;
    align-items: flex-start;
    justify-content: center;
    padding-top: 10vh;
    background: var(--overlay-wrapper-background);
  }

  .upload-modal {
    width: min(34rem, 92vw);
    max-height: 75vh;
    padding: 1em 1.25em;
    overflow-y: auto;
    color: var(--text-color);
    background: var(--background);
    border: 1px solid var(--border);
    box-shadow: var(--box-shadow-overlay);
  }

  .upload-modal h3 {
    margin: 0 0 0.5em;
    font-size: 1.1em;
    font-weight: bold;
  }

  .field {
    margin-bottom: 0.5em;
  }

  .field span {
    display: block;
    margin-bottom: 0.15em;
    font-size: 0.85em;
    color: var(--text-color-lighter);
  }

  .field input[type="text"],
  .field select {
    width: 100%;
    padding: 0.25em 0.4em;
    color: var(--text-color);
    background: var(--background);
    border: 1px solid var(--border);
  }

  .file {
    width: 100%;
    margin-bottom: 0.5rem;
    padding: 0.25em 0.4em;
    color: var(--text-color);
    background: var(--background);
    border: 1px solid var(--border);
  }

  .hint {
    font-size: 0.85em;
    color: var(--text-color-lighter);
  }

  .actions {
    display: flex;
    align-items: center;
    margin-top: 0.75em;
  }

  .spacer {
    flex: 1;
  }

  .actions button[type="submit"] {
    padding: 0.25em 0.9em;
    font: inherit;
    color: var(--background);
    cursor: pointer;
    background: var(--link-color);
    border: 1px solid var(--link-color);
  }

  .actions button[type="submit"]:disabled {
    opacity: 0.6;
  }

  .error {
    color: var(--error);
  }

  :global(.dragover) {
    outline: 2px dashed var(--link-color);
    outline-offset: 2px;
  }
</style>
