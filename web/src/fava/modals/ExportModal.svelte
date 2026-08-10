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
  import { translations, type Locale } from "../../translations";

  export let locale: string;

  let shown = false;

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  function sync() {
    shown = window.location.hash === "#export";
  }

  // Recomputed when the modal opens so the download carries the filters that
  // are live at that moment.
  $: href = shown ? `/__orangecount/fava/download-journal${window.location.search}` : "";

  function close() {
    window.history.replaceState({}, "", window.location.pathname + window.location.search);
    shown = false;
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.key === "Escape") {
      close();
    }
  }

  onMount(() => {
    sync();
    window.addEventListener("hashchange", sync);
    document.addEventListener("keydown", onKeydown);
    return () => {
      window.removeEventListener("hashchange", sync);
      document.removeEventListener("keydown", onKeydown);
    };
  });
</script>

{#if shown}
  <div class="export-backdrop" role="presentation" on:click={close}>
    <div class="export-modal" role="dialog" aria-modal="true" aria-label={t("export")} on:click|stopPropagation>
      <h3>{t("export")}:</h3>
      <a {href} download="journal.bean">{t("downloadFilteredEntries")}</a>
    </div>
  </div>
{/if}

<style>
  .export-backdrop {
    position: fixed;
    inset: 0;
    z-index: var(--z-index-floating-ui);
    display: flex;
    align-items: flex-start;
    justify-content: center;
    padding-top: 15vh;
    background: var(--overlay-wrapper-background);
  }

  .export-modal {
    min-width: 18em;
    max-width: 32em;
    padding: 1em 1.25em;
    color: var(--text-color);
    background: var(--background);
    border: 1px solid var(--border);
    box-shadow: var(--box-shadow-overlay);
  }

  .export-modal h3 {
    margin: 0 0 0.5em;
    font-size: 1.1em;
    font-weight: bold;
  }

  .export-modal a {
    color: var(--link-color);
  }
</style>
