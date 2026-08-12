<!-- OrangeCount original component. This is not derived from Fava.
   Quick-entry shorthand capture panel: compiles, previews, and publishes
   two-posting transactions through the reviewed write path. -->

<script lang="ts">
  import { PRIVATE_ADAPTER_BASE } from "../adapter-client";
  import { notify, notify_err } from "../notifications";
  import { translations, type Locale } from "../../translations";

  export let locale: Locale = "en";
  export let onSaved: () => void = () => {};

  interface QuickLineError { code: string; message: string; }
  interface QuickLine {
    line: number;
    source: string;
    preview?: string;
    duplicate?: boolean;
    errors?: QuickLineError[];
  }
  interface QuickPreviewResponse {
    token: string;
    lines: QuickLine[];
    target: string;
    snapshot_id: string;
    has_errors: boolean;
  }

  let text = "";
  let date = new Date().toISOString().slice(0, 10);
  let flag = "*";
  let preview: QuickPreviewResponse | null = null;
  let saving = false;
  let error = "";
  let textArea: HTMLTextAreaElement | undefined;
  let canUndo = false;

  function t(key: string): string {
    const catalog = translations[locale === "zh-CN" ? "zh-CN" : "en"];
    return catalog[key] || key;
  }

  async function doPreview() {
    if (saving || !text.trim()) return;
    saving = true;
    error = "";
    preview = null;
    try {
      const response = await fetch(`${PRIVATE_ADAPTER_BASE}/quick-preview`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ text, date, flag }),
      });
      const payload = (await response.json()) as QuickPreviewResponse | { error?: string };
      if (!response.ok) {
        throw new Error((payload as { error?: string }).error || `Preview failed (${response.status})`);
      }
      preview = payload as QuickPreviewResponse;
    } catch (err) {
      error = err instanceof Error ? err.message : "Preview failed.";
    } finally {
      saving = false;
    }
  }

  async function doCommit() {
    if (!preview || preview.has_errors || saving) return;
    saving = true;
    error = "";
    try {
      const response = await fetch(`${PRIVATE_ADAPTER_BASE}/quick-commit`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ token: preview!.token, expected_snapshot_id: preview!.snapshot_id }),
      });
      const payload = (await response.json()) as { published?: boolean; error?: string };
      if (!response.ok || !payload.published) {
        throw new Error(payload.error || `Commit failed (${response.status})`);
      }
      notify(t("quickEntryPublished"));
      canUndo = true;
      text = "";
      preview = null;
      onSaved();
    } catch (err) {
      error = err instanceof Error ? err.message : "Commit failed.";
    } finally {
      saving = false;
    }
  }

  async function doUndo() {
    if (saving || !canUndo) return;
    saving = true;
    error = "";
    try {
      const response = await fetch(`${PRIVATE_ADAPTER_BASE}/quick-undo`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: "{}",
      });
      const payload = (await response.json()) as { undone?: boolean; error?: string };
      if (!response.ok || !payload.undone) {
        throw new Error(payload.error || `Undo failed (${response.status})`);
      }
      notify(t("quickEntryUndone"));
      canUndo = false;
      onSaved();
    } catch (err) {
      error = err instanceof Error ? err.message : "Undo failed.";
    } finally {
      saving = false;
    }
  }

  function onKeydown(event: KeyboardEvent) {
    // Ctrl/Cmd+Enter triggers preview, then commit on the second press.
    if ((event.ctrlKey || event.metaKey) && event.key === "Enter") {
      event.preventDefault();
      if (preview && !preview.has_errors) {
        doCommit();
      } else {
        doPreview();
      }
    }
  }

  $: hasErrors = preview?.has_errors ?? false;
</script>

<div class="quick-entry-panel">
  {#if error}
    <p class="error" role="alert">{error}</p>
  {/if}
  <div class="field-row">
    <label class="field">
      <span>{t("date")}</span>
      <input type="date" bind:value={date} />
    </label>
    <label class="field narrow">
      <span>{t("flag")}</span>
      <select bind:value={flag}>
        <option value="*">*</option>
        <option value="!">!</option>
      </select>
    </label>
    {#if canUndo}
      <button type="button" class="undo-btn" on:click={doUndo} disabled={saving}>
        {t("quickEntryUndo")}
      </button>
    {/if}
  </div>
  <textarea
    bind:value={text}
    bind:this={textArea}
    on:keydown={onKeydown}
    spellcheck="false"
    placeholder={t("quickEntryPlaceholder")}
    rows="5"
  ></textarea>
  <div class="actions">
    <button type="button" on:click={doPreview} disabled={saving || !text.trim()}>
      {t("preview")}
    </button>
    <button
      type="button"
      class="primary"
      on:click={doCommit}
      disabled={saving || !preview || hasErrors}
    >
      {t("commit")}
    </button>
    <span class="hint">{t("quickEntryHint")}</span>
  </div>
  {#if preview}
    <div class="preview-area">
      <h4>{t("preview")} → {preview.target}</h4>
      {#each preview.lines as line}
        <div class="preview-line" class:has-error={line.errors?.length}>
          <div class="line-source">
            <span class="line-number">{line.line}</span>
            <code>{line.source}</code>
            {#if line.duplicate}
              <span class="duplicate-badge" title={t("quickEntryDuplicate")}>⚠</span>
            {/if}
          </div>
          {#if line.errors?.length}
            <ul class="line-errors">
              {#each line.errors as err}
                <li>{err.message}</li>
              {/each}
            </ul>
          {/if}
          {#if line.preview}
            <pre class="line-preview">{line.preview}</pre>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .quick-entry-panel {
    display: flex;
    flex-direction: column;
    gap: 0.6em;
  }
  .field-row {
    display: flex;
    gap: 0.6em;
    align-items: flex-end;
  }
  .field {
    display: flex;
    flex-direction: column;
    gap: 0.2em;
    flex: 1;
  }
  .field.narrow { flex: 0 0 auto; }
  .field span { font-size: 0.85em; opacity: 0.7; }
  .undo-btn {
    flex: 0 0 auto;
    align-self: flex-end;
  }
  textarea {
    width: 100%;
    font-family: var(--mono-font, monospace);
    font-size: 0.9em;
    resize: vertical;
  }
  .actions {
    display: flex;
    gap: 0.5em;
    align-items: center;
  }
  .actions .hint {
    font-size: 0.8em;
    opacity: 0.6;
  }
  button.primary {
    font-weight: 600;
  }
  .preview-area {
    border-top: 1px solid var(--border-color, #ccc);
    padding-top: 0.6em;
    max-height: 40vh;
    overflow-y: auto;
  }
  .preview-area h4 {
    margin: 0 0 0.4em;
    font-size: 0.9em;
  }
  .preview-line {
    margin-bottom: 0.6em;
    padding: 0.4em;
    border-radius: 3px;
    background: var(--background-alt, rgba(0,0,0,0.03));
  }
  .preview-line.has-error {
    background: rgba(255, 0, 0, 0.05);
  }
  .line-source {
    display: flex;
    align-items: center;
    gap: 0.4em;
  }
  .line-number {
    font-size: 0.8em;
    opacity: 0.5;
    min-width: 1.5em;
  }
  .line-source code {
    font-family: var(--mono-font, monospace);
    font-size: 0.85em;
  }
  .duplicate-badge {
    color: var(--warning-color, #c80);
    cursor: help;
  }
  .line-errors {
    margin: 0.3em 0 0;
    padding-left: 1.5em;
    color: var(--error-color, #c00);
    font-size: 0.85em;
  }
  .line-preview {
    margin: 0.3em 0 0;
    padding: 0.3em;
    font-family: var(--mono-font, monospace);
    font-size: 0.82em;
    background: var(--background, #fff);
    border: 1px solid var(--border-color, #eee);
    border-radius: 2px;
    white-space: pre-wrap;
  }
</style>
