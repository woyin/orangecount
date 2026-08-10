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
  import { translations, type Locale } from "../../translations";

  interface MoveDetails {
    account: string;
    filename: string;
    newName: string;
  }

  export let locale = "en";
  export let accounts: string[] = [];
  export let moving: MoveDetails | null = null;
  export let onMoved: (message: string) => void = () => {};
  export let onClose: () => void = () => {};

  let account = "";
  let newName = "";
  let saving = false;
  let error = "";

  $: if (moving) {
    account = moving.account;
    newName = moving.newName;
    error = "";
    saving = false;
  }

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.key === "Escape" && moving && !saving) onClose();
  }

  async function submit(event: SubmitEvent) {
    event.preventDefault();
    if (!moving || saving) return;
    saving = true;
    error = "";
    try {
      const response = await fetch(`${PRIVATE_ADAPTER_BASE}/move-document`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ filename: moving.filename, account, new_name: newName }),
      });
      const payload = (await response.json()) as { error?: string; data?: { message?: string } };
      if (!response.ok) throw new Error(payload.error || `Moving the document failed (${response.status})`);
      onMoved(payload.data?.message || "Moved.");
      onClose();
    } catch (value) {
      error = value instanceof Error ? value.message : "Moving the document failed.";
      saving = false;
    }
  }
</script>

<svelte:document on:keydown={onKeydown} />

{#if moving}
  <div class="move-backdrop" role="presentation" on:click={() => !saving && onClose()}>
    <form
      class="move-modal"
      role="dialog"
      aria-modal="true"
      aria-label={t("moveDocument")}
      on:click|stopPropagation
      on:submit={submit}
    >
      <h3>{t("moveDocument")}</h3>
      <p><code>{moving.filename}</code></p>
      {#if error}
        <p class="error" role="alert">{error}</p>
      {/if}
      <div class="fields">
        <input type="text" bind:value={account} list="move-accounts" placeholder="Assets:Cash" required autocomplete="off" />
        <input size={40} bind:value={newName} required />
        <button type="submit" disabled={saving}>{t("move")}</button>
      </div>
      <datalist id="move-accounts">
        {#each accounts as name (name)}
          <option value={name}></option>
        {/each}
      </datalist>
    </form>
  </div>
{/if}

<style>
  .move-backdrop {
    position: fixed;
    inset: 0;
    z-index: var(--z-index-floating-ui);
    display: flex;
    align-items: flex-start;
    justify-content: center;
    padding-top: 10vh;
    background: var(--overlay-wrapper-background);
  }

  .move-modal {
    width: min(34rem, 92vw);
    padding: 1em 1.25em;
    color: var(--text-color);
    background: var(--background);
    border: 1px solid var(--border);
    box-shadow: var(--box-shadow-overlay);
  }

  .move-modal h3 {
    margin: 0 0 0.5em;
    font-size: 1.1em;
    font-weight: bold;
  }

  .move-modal code {
    word-break: break-all;
  }

  .fields {
    display: flex;
    gap: 0.4em;
    flex-wrap: wrap;
  }

  .fields input[type="text"] {
    flex: 1;
    min-width: 12rem;
    padding: 0.25em 0.4em;
    color: var(--text-color);
    background: var(--background);
    border: 1px solid var(--border);
  }

  .fields input[size] {
    padding: 0.25em 0.4em;
    color: var(--text-color);
    background: var(--background);
    border: 1px solid var(--border);
  }

  .fields button[type="submit"] {
    padding: 0.25em 0.9em;
    font: inherit;
    color: var(--background);
    cursor: pointer;
    background: var(--link-color);
    border: 1px solid var(--link-color);
  }

  .fields button[type="submit"]:disabled {
    opacity: 0.6;
  }

  .error {
    color: var(--error);
  }
</style>
