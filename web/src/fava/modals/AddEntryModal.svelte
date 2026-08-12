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
  import { PRIVATE_ADAPTER_BASE } from "../adapter-client";
  import { notify } from "../notifications";
  import AutocompleteInput from "../components/AutocompleteInput.svelte";
  import QuickEntryPanel from "./QuickEntryPanel.svelte";
  import { translations, type Locale } from "../../translations";
  // Locale is already imported above.

  export let locale = "en";
  export let onSaved: () => void = () => {};
  /** Known payees, used to autocomplete the payee field of a transaction. */
  export let payees: string[] = [];

  type EntryType = "transaction" | "balance" | "note" | "quick";

  interface PostingRow {
    account: string;
    amount: string;
    currency: string;
  }

  let shown = false;
  let entryType: EntryType = "transaction";
  let date = new Date().toISOString().slice(0, 10);
  let flag = "*";
  let payee = "";
  let narration = "";
  let tags = "";
  let links = "";
  let postings: PostingRow[] = [{ account: "", amount: "", currency: "" }];
  let account = "";
  let amount = "";
  let currency = "";
  let comment = "";
  let continueAdding = false;
  let saving = false;
  let error = "";
  let dateInput: HTMLInputElement | undefined;

  function t(key: string): string {
    const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];
    return catalog[key] || key;
  }

  function storedContinue(): boolean {
    try {
      return localStorage.getItem("add-entry-continue") === "true";
    } catch {
      return false;
    }
  }

  function persistContinue() {
    try {
      localStorage.setItem("add-entry-continue", String(continueAdding));
    } catch {
      /* storage is optional */
    }
  }

  function sync() {
    const hash = window.location.hash;
    shown = hash === "#add-transaction" || hash === "#add-quick";
    if (shown) {
      error = "";
      continueAdding = storedContinue();
      // If opened via the Quick hash, pre-select the Quick tab.
      if (hash === "#add-quick") {
        entryType = "quick";
      }
      setTimeout(() => dateInput?.focus(), 0);
    }
  }

  function setType(next: EntryType) {
    entryType = next;
    error = "";
  }

  function close() {
    window.history.replaceState({}, "", window.location.pathname + window.location.search);
    shown = false;
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.key === "Escape" && shown) {
      close();
    }
  }

  function splitTokens(value: string): string[] {
    return value
      .split(/[\s,]+/)
      .map((token) => token.trim())
      .filter(Boolean);
  }

  function buildEntry(): Record<string, unknown> {
    if (entryType === "transaction") {
      return {
        type: "transaction",
        date,
        flag,
        payee: payee.trim(),
        narration: narration.trim(),
        tags: splitTokens(tags),
        links: splitTokens(links),
        postings: postings
          .filter((row) => row.account.trim() || row.amount.trim() || row.currency.trim())
          .map((row) => ({
            account: row.account.trim(),
            amount: row.amount.trim(),
            currency: row.currency.trim(),
          })),
      };
    }
    if (entryType === "balance") {
      return {
        type: "balance",
        date,
        account: account.trim(),
        amount: amount.trim(),
        currency: currency.trim(),
      };
    }
    return {
      type: "note",
      date,
      account: account.trim(),
      comment: comment.trim(),
    };
  }

  function resetForm() {
    // Upstream keeps the date of the entry that was just added.
    payee = "";
    narration = "";
    tags = "";
    links = "";
    postings = [{ account: "", amount: "", currency: "" }];
    account = "";
    amount = "";
    currency = "";
    comment = "";
  }

  async function submit(event: SubmitEvent) {
    event.preventDefault();
    if (saving) return;
    saving = true;
    error = "";
    try {
      const response = await fetch(`${PRIVATE_ADAPTER_BASE}/add-entries`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ entries: [buildEntry()] }),
      });
      const payload = (await response.json()) as { error?: string };
      if (!response.ok) {
        throw new Error(payload.error || `Adding the entry failed (${response.status})`);
      }
      notify(t("entryAdded"));
      resetForm();
      onSaved();
      if (!continueAdding) close();
    } catch (value) {
      error = value instanceof Error ? value.message : "Adding the entry failed.";
    } finally {
      saving = false;
    }
  }

  function addPosting() {
    postings = [...postings, { account: "", amount: "", currency: "" }];
  }

  function removePosting(index: number) {
    postings = postings.filter((_, i) => i !== index);
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
  <div class="add-backdrop" role="presentation" on:click={close}>
    <form
      class="add-modal"
      role="dialog"
      aria-modal="true"
      aria-label={t("add")}
      on:click|stopPropagation
      on:submit={submit}
    >
      <h3>
        {t("add")}
        <button type="button" class:selected={entryType === "transaction"} on:click={() => setType("transaction")}>{t("transaction")}</button>
        <button type="button" class:selected={entryType === "balance"} on:click={() => setType("balance")}>{t("balance")}</button>
        <button type="button" class:selected={entryType === "note"} on:click={() => setType("note")}>{t("note")}</button>
        <button type="button" class:selected={entryType === "quick"} on:click={() => setType("quick")}>{t("quickEntry")}</button>
      </h3>
      {#if error}
        <p class="error" role="alert">{error}</p>
      {/if}
      <div class="field">
        <span>{t("date")}</span>
        <input type="date" bind:value={date} bind:this={dateInput} required />
      </div>
      {#if entryType === "transaction"}
        <div class="row">
          <div class="field narrow">
            <span>{t("flag")}</span>
            <select bind:value={flag}>
              <option value="*">*</option>
              <option value="!">!</option>
            </select>
          </div>
          <div class="field grow">
            <span>{t("payee")}</span>
            <AutocompleteInput bind:value={payee} suggestions={payees} onEnter={() => {}} />
          </div>
          <div class="field grow">
            <span>{t("narration")}</span>
            <input type="text" bind:value={narration} autocomplete="off" />
          </div>
        </div>
        <div class="row">
          <div class="field grow">
            <span>{t("tag")}</span>
            <input type="text" bind:value={tags} placeholder="tag1 tag2" autocomplete="off" />
          </div>
          <div class="field grow">
            <span>{t("link")}</span>
            <input type="text" bind:value={links} placeholder="link1 link2" autocomplete="off" />
          </div>
        </div>
        <table class="postings">
          <thead>
            <tr>
              <th>{t("account")}</th>
              <th>{t("amount")}</th>
              <th>{t("currency")}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {#each postings as posting, index}
              <tr>
                <td><input type="text" bind:value={posting.account} placeholder="Assets:Cash" autocomplete="off" /></td>
                <td><input type="text" bind:value={posting.amount} placeholder="10.00" autocomplete="off" /></td>
                <td><input type="text" bind:value={posting.currency} placeholder="USD" autocomplete="off" /></td>
                <td>
                  <button type="button" class="remove" aria-label="Remove posting" disabled={postings.length === 1} on:click={() => removePosting(index)}>×</button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
        <button type="button" class="add-posting" on:click={addPosting}>+ {t("add")}</button>
      {:else if entryType === "balance"}
        <div class="field">
          <span>{t("account")}</span>
          <input type="text" bind:value={account} placeholder="Assets:Cash" required autocomplete="off" />
        </div>
        <div class="row">
          <div class="field grow">
            <span>{t("amount")}</span>
            <input type="text" bind:value={amount} placeholder="100.00" required autocomplete="off" />
          </div>
          <div class="field narrow">
            <span>{t("currency")}</span>
            <input type="text" bind:value={currency} placeholder="USD" required autocomplete="off" />
          </div>
        </div>
      {:else}
        <div class="field">
          <span>{t("account")}</span>
          <input type="text" bind:value={account} placeholder="Assets:Cash" required autocomplete="off" />
        </div>
        <div class="field">
          <span>{t("comment")}</span>
          <input type="text" bind:value={comment} required autocomplete="off" />
        </div>
      {/if}
      {#if entryType === "quick"}
        <QuickEntryPanel locale={locale as Locale} {onSaved} />
      {/if}
      <div class="actions">
        <span class="spacer"></span>
        <label class="continue">
          <input type="checkbox" bind:checked={continueAdding} on:change={persistContinue} />
          <span>{t("continueAdding")}</span>
        </label>
        <button type="submit" disabled={saving}>{t("save")}</button>
      </div>
    </form>
  </div>
{/if}

<style>
  .add-backdrop {
    position: fixed;
    inset: 0;
    z-index: var(--z-index-floating-ui);
    display: flex;
    align-items: flex-start;
    justify-content: center;
    padding-top: 10vh;
    background: var(--overlay-wrapper-background);
  }

  .add-modal {
    width: min(46rem, 92vw);
    max-height: 75vh;
    padding: 1em 1.25em;
    overflow-y: auto;
    color: var(--text-color);
    background: var(--background);
    border: 1px solid var(--border);
    box-shadow: var(--box-shadow-overlay);
  }

  .add-modal h3 {
    margin: 0 0 0.5em;
    font-size: 1.1em;
    font-weight: bold;
  }

  .add-modal h3 button {
    margin-left: 0.5em;
    padding: 0.1em 0.6em;
    font: inherit;
    color: var(--text-color);
    cursor: pointer;
    background: transparent;
    border: 1px solid var(--border);
  }

  .add-modal h3 button.selected {
    color: var(--background);
    background: var(--link-color);
    border-color: var(--link-color);
  }

  .row {
    display: flex;
    gap: 0.75em;
  }

  .field {
    margin-bottom: 0.5em;
  }

  .field.grow {
    flex: 1;
  }

  .field.narrow {
    flex: 0 0 6rem;
  }

  .field span {
    display: block;
    margin-bottom: 0.15em;
    font-size: 0.85em;
    color: var(--text-color-lighter);
  }

  .field input,
  .field select,
  .postings input {
    width: 100%;
    padding: 0.25em 0.4em;
    color: var(--text-color);
    background: var(--background);
    border: 1px solid var(--border);
  }

  .postings {
    width: 100%;
    margin: 0.25em 0;
    border-collapse: collapse;
  }

  .postings th {
    padding: 0.15em 0.25em;
    font-weight: normal;
    font-size: 0.85em;
    color: var(--text-color-lighter);
    text-align: left;
  }

  .postings td {
    padding: 0.1em 0.25em 0.1em 0;
  }

  .postings td:last-child {
    width: 1.5em;
    padding-right: 0;
  }

  .remove {
    padding: 0 0.25em;
    font: inherit;
    color: var(--text-color-lighter);
    cursor: pointer;
    background: transparent;
    border: 0;
  }

  .remove:disabled {
    visibility: hidden;
  }

  .add-posting {
    padding: 0.1em 0.4em;
    font: inherit;
    color: var(--link-color);
    cursor: pointer;
    background: transparent;
    border: 0;
  }

  .actions {
    display: flex;
    align-items: center;
    margin-top: 0.75em;
  }

  .spacer {
    flex: 1;
  }

  .continue {
    display: flex;
    gap: 0.25em;
    align-items: center;
    margin-right: 1em;
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
</style>
