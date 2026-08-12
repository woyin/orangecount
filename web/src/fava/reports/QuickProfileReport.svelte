<!-- OrangeCount original component. Quick-entry profile manager. -->
<script lang="ts">
  import { onMount } from "svelte";
  import { PRIVATE_ADAPTER_BASE } from "../adapter-client";
  import { notify_err } from "../notifications";
  import { translations, type Locale } from "../../translations";

  export let locale: Locale = "en";
  export let onSaved: () => void = () => {};

  interface ProfileRule {
    type: string;
    name: string;
    account?: string;
    source?: string;
    destination?: string;
    currency?: string;
    payee?: string;
    narration?: string;
  }
  interface ProfileProblem {
    code: string;
    message: string;
    source?: string;
  }

  let rules: ProfileRule[] = [];
  let problems: ProfileProblem[] = [];
  let loading = true;

  // New rule form state
  let newType = "account";
  let newName = "";
  let newAccount = "";
  let newSource = "";
  let newDest = "";
  let newCurrency = "";
  let newDate = new Date().toISOString().slice(0, 10);
  let saving = false;

  function t(key: string): string {
    const catalog = translations[locale === "zh-CN" ? "zh-CN" : "en"];
    return catalog[key] || key;
  }

  async function load() {
    loading = true;
    try {
      const response = await fetch(`${PRIVATE_ADAPTER_BASE}/quick-profile`);
      const payload = (await response.json()) as { rules: ProfileRule[]; problems: ProfileProblem[] };
      rules = payload.rules || [];
      problems = payload.problems || [];
    } catch (err) {
      notify_err(err, () => "Failed to load profile");
    } finally {
      loading = false;
    }
  }

  async function save() {
    if (saving || !newName.trim()) return;
    saving = true;
    try {
      const response = await fetch(`${PRIVATE_ADAPTER_BASE}/quick-profile-save`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          date: newDate,
          type: newType,
          name: newName,
          account: newAccount,
          source: newSource,
          destination: newDest,
          currency: newCurrency,
        }),
      });
      if (!response.ok) {
        const payload = (await response.json()) as { error?: string };
        throw new Error(payload.error || `Save failed (${response.status})`);
      }
      newName = "";
      newAccount = "";
      newSource = "";
      newDest = "";
      newCurrency = "";
      await load();
      onSaved();
    } catch (err) {
      notify_err(err, () => "Failed to save profile rule");
    } finally {
      saving = false;
    }
  }

  onMount(() => load());
</script>

<div class="headerline">
  <h2>{t("quickProfileTitle")}</h2>
  <span class="muted">{t("quickProfileAdd")}</span>
</div>

{#if loading}
  <p class="muted">{t("loading")}</p>
{:else}
  <div class="profile-section">
    {#if problems.length}
      <div class="problems">
        {#each problems as problem}
          <div class="problem" role="alert">
            <strong>{problem.code}</strong>: {problem.message}
          </div>
        {/each}
      </div>
    {/if}

    {#if rules.length}
      <table class="rules-table">
        <thead>
          <tr>
            <th>{t("quickProfileType")}</th>
            <th>{t("quickProfileName")}</th>
            <th>{t("account")}</th>
            <th>{t("source")}</th>
            <th>{t("quickProfileType")}</th>
            <th>{t("currency")}</th>
          </tr>
        </thead>
        <tbody>
          {#each rules as rule}
            <tr>
              <td>{rule.type === "account" ? t("quickProfileAccount") : t("quickProfileTemplate")}</td>
              <td>{rule.name}</td>
              <td>{rule.account || ""}</td>
              <td>{rule.source || ""}</td>
              <td>{rule.destination || ""}</td>
              <td>{rule.currency || ""}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {:else}
      <p class="muted">{t("empty")}</p>
    {/if}
  </div>

  <div class="add-rule-form">
    <h3>{t("quickProfileAdd")}</h3>
    <div class="form-row">
      <label>
        {t("date")}
        <input type="date" bind:value={newDate} />
      </label>
      <label>
        {t("quickProfileType")}
        <select bind:value={newType}>
          <option value="account">{t("quickProfileAccount")}</option>
          <option value="template">{t("quickProfileTemplate")}</option>
        </select>
      </label>
      <label>
        {t("quickProfileName")}
        <input type="text" bind:value={newName} placeholder="微信" />
      </label>
    </div>
    {#if newType === "account"}
      <div class="form-row">
        <label>
          {t("account")}
          <input type="text" bind:value={newAccount} placeholder="Assets:WeChat" />
        </label>
      </div>
    {:else}
      <div class="form-row">
        <label>
          {t("source")}
          <input type="text" bind:value={newSource} placeholder="微信" />
        </label>
        <label>
          {t("quickProfileType")}
          <input type="text" bind:value={newDest} placeholder="Expenses:Food" />
        </label>
        <label>
          {t("currency")}
          <input type="text" bind:value={newCurrency} placeholder="CNY" />
        </label>
      </div>
    {/if}
    <button type="button" on:click={save} disabled={saving || !newName.trim()}>
      {t("save")}
    </button>
  </div>
{/if}

<style>
  .profile-section { margin-bottom: 1.5em; }
  .problems { margin-bottom: 1em; }
  .problem {
    padding: 0.4em 0.6em;
    margin-bottom: 0.3em;
    background: rgba(255, 140, 0, 0.1);
    border-radius: 3px;
    font-size: 0.9em;
  }
  .rules-table { width: 100%; border-collapse: collapse; }
  .rules-table th, .rules-table td {
    text-align: left;
    padding: 0.3em 0.5em;
    border-bottom: 1px solid var(--border-color, #eee);
    font-size: 0.9em;
  }
  .add-rule-form {
    border-top: 1px solid var(--border-color, #ccc);
    padding-top: 1em;
  }
  .add-rule-form h3 { margin: 0 0 0.5em; font-size: 1em; }
  .form-row {
    display: flex;
    gap: 0.6em;
    margin-bottom: 0.5em;
    flex-wrap: wrap;
  }
  .form-row label {
    display: flex;
    flex-direction: column;
    gap: 0.2em;
    font-size: 0.85em;
    flex: 1;
  }
  .form-row input, .form-row select {
    min-width: 8em;
  }
</style>
