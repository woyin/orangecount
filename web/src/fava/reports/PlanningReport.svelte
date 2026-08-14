<!-- OrangeCount original component. Conservative liquidity planning report. -->
<script lang="ts">
  import { onMount } from "svelte";
  import type { AdapterClient } from "../adapter-client";
  import { PRIVATE_ADAPTER_BASE } from "../adapter-client";
  import { translations, type Locale } from "../../translations";

  export let adapter: AdapterClient;
  export let locale = "en";

  type Point = { date: string; headroom: string; committed_outflow: string; adjustable_outflow: string; ignored_inflows: string };
  type Result = { currency: string; primary_safe_to_spend: string; funding_shortfall: string; liquidity_low_point: Point; timeline: Point[]; current_planning: boolean };
  type PlanningState = { snapshot_id: string; readiness_error?: string; profile: { currency: string; timezone: string; recorded_through: string; problems: { code: string; message: string }[]; plans: { id: string; revision: number; name: string; date: string; amount: string; direction: string; commitment: string; account: string }[] }; result?: Result };
  type Review = { snapshot_id: string; completed: boolean; stale: boolean; patterns: { key: string; name: string; account: string; cadence_days: number; minimum_amount: string; maximum_amount: string; occurrences: unknown[] }[]; matches: { plan_id: string; occurrence_id: string; same_name: boolean; same_account: boolean; same_amount: boolean; date_distance: number }[] };

  let state: PlanningState | null = null;
  let error = "";
  let scenarioError = "";
  let scenario: Result | null = null;
  let horizon = new Date().toISOString().slice(0, 10);
  let purchaseName = "";
  let purchaseAmount = "";
  let purchaseDate = horizon;
  let includeInflows = false;
  let releaseAdjustable = false;
  let review: Review | null = null;
  let reviewError = "";
  let selectedMatches: string[] = [];
  let reviewPreview: { token: string; content: string; snapshot_id: string } | null = null;
  let generatedPreview: { token: string; content: string; snapshot_id: string } | null = null;
  let setupCurrency = "CNY"; let setupTimezone = "Asia/Singapore"; let setupReserve = "0"; let setupFunds = ""; let setupDebts = ""; let setupRecorded = horizon;
  let planID = ""; let planName = ""; let planAmount = ""; let planDate = horizon; let planDirection = "outflow"; let planCommitment = "committed"; let planAccount = "";
  let resolvePlanID = ""; let resolvePlanStatus = "fulfilled"; let resolvePlanDate = horizon;
  $: t = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale];

  async function load() {
    try {
      state = await adapter.load("planning", { horizon_end: horizon }) as PlanningState;
      purchaseDate = horizon;
      error = "";
      scenario = null;
    } catch (e) {
      error = e instanceof Error ? e.message : "Planning could not load";
    }
  }

  async function evaluatePurchase() {
    scenarioError = "";
    scenario = null;
    try {
      const response = await fetch(`${PRIVATE_ADAPTER_BASE}/planning-scenario`, {
        method: "POST",
        headers: { "Content-Type": "application/json", Accept: "application/json" },
        body: JSON.stringify({ horizon_end: horizon, name: purchaseName, amount: purchaseAmount, date: purchaseDate, include_future_inflows: includeInflows, release_adjustable: releaseAdjustable }),
      });
      const payload = await response.json() as { error?: string; result?: Result };
      if (!response.ok || !payload.result) throw new Error(payload.error || "Scenario could not be evaluated");
      scenario = payload.result;
    } catch (e) {
      scenarioError = e instanceof Error ? e.message : "Scenario could not be evaluated";
    }
  }

  async function loadReview() {
    try { review = await adapter.load("planning-review") as Review; reviewError = ""; selectedMatches = []; reviewPreview = null; }
    catch (e) { reviewError = e instanceof Error ? e.message : "Review evidence could not load"; }
  }

  async function previewGenerated(kind: "profile" | "plan") {
    if (!state) return;
    try {
      const profile = { id: "primary", currency: setupCurrency, timezone: setupTimezone, minimum_reserve: setupReserve, recorded_through: setupRecorded, spendable_accounts: setupFunds.split(","), short_term_debt_accounts: setupDebts.split(",") };
      const existing = state.profile.plans.find(value => value.id === resolvePlanID);
      const plan = kind === "plan" && existing && resolvePlanStatus ? { id: existing.id, revision: existing.revision + 1, status: resolvePlanStatus, name: existing.name, amount: existing.amount, date: resolvePlanStatus === "active" ? resolvePlanDate : existing.date, direction: existing.direction, commitment: existing.commitment, account: existing.account } : { id: planID, name: planName, amount: planAmount, date: planDate, direction: planDirection, commitment: planCommitment, account: planAccount };
      const response = await fetch(`${PRIVATE_ADAPTER_BASE}/planning-generate-preview`, { method: "POST", headers: { "Content-Type": "application/json", Accept: "application/json" }, body: JSON.stringify({ kind, expected_snapshot_id: state.snapshot_id, profile, plan }) });
      const payload = await response.json() as { token?: string; content?: string; snapshot_id?: string; error?: string };
      if (!response.ok || !payload.token || !payload.content || !payload.snapshot_id) throw new Error(payload.error || "Preview could not be created");
      generatedPreview = { token: payload.token, content: payload.content, snapshot_id: payload.snapshot_id }; error = "";
    } catch (e) { error = e instanceof Error ? e.message : "Preview could not be created"; }
  }
  async function commitGenerated() {
    if (!generatedPreview) return;
    try {
      const response = await fetch(`${PRIVATE_ADAPTER_BASE}/planning-commit`, { method: "POST", headers: { "Content-Type": "application/json", Accept: "application/json" }, body: JSON.stringify({ token: generatedPreview.token, expected_snapshot_id: generatedPreview.snapshot_id }) });
      const payload = await response.json() as { published?: boolean; error?: string };
      if (!response.ok || !payload.published) throw new Error(payload.error || "Planning change could not be saved");
      generatedPreview = null; await load();
    } catch (e) { error = e instanceof Error ? e.message : "Planning change could not be saved"; }
  }

  function toggleMatch(key: string, checked: boolean) { selectedMatches = checked ? [...selectedMatches, key] : selectedMatches.filter(value => value !== key); reviewPreview = null; }
  function useCandidate(name: string, amount: string) { planName = name; planAmount = amount; planDirection = "outflow"; planCommitment = "committed"; planID = `${name.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "") || "plan"}-${planDate}`; }
  async function previewCompletion() {
    if (!review) return;
    try {
      const matches = selectedMatches.map(key => { const [plan_id, occurrence_id] = key.split("\u0000"); return { plan_id, occurrence_id }; });
      const response = await fetch(`${PRIVATE_ADAPTER_BASE}/planning-review-preview`, { method: "POST", headers: { "Content-Type": "application/json", Accept: "application/json" }, body: JSON.stringify({ expected_snapshot_id: review.snapshot_id, matches }) });
      const payload = await response.json() as { token?: string; content?: string; snapshot_id?: string; error?: string };
      if (!response.ok || !payload.token || !payload.content || !payload.snapshot_id) throw new Error(payload.error || "Review preview could not be created");
      reviewPreview = { token: payload.token, content: payload.content, snapshot_id: payload.snapshot_id };
    } catch (e) { reviewError = e instanceof Error ? e.message : "Review preview could not be created"; }
  }
  async function commitCompletion() {
    if (!reviewPreview) return;
    try {
      const response = await fetch(`${PRIVATE_ADAPTER_BASE}/planning-commit`, { method: "POST", headers: { "Content-Type": "application/json", Accept: "application/json" }, body: JSON.stringify({ token: reviewPreview.token, expected_snapshot_id: reviewPreview.snapshot_id }) });
      const payload = await response.json() as { published?: boolean; error?: string };
      if (!response.ok || !payload.published) throw new Error(payload.error || "Review could not be saved");
      reviewPreview = null; await load(); await loadReview();
    } catch (e) { reviewError = e instanceof Error ? e.message : "Review could not be saved"; }
  }

  onMount(() => { void load(); });
</script>

<section class="planning-report">
  <div class="headerline">
    <h2>{t.planning || "Planning"}</h2>
    <label>{t.to || "To"} <input type="date" bind:value={horizon} onchange={load}></label>
  </div>
  {#if error}
    <p class="error-panel">{error}</p>
  {:else if !state}
    <p>{t.loading || "Loading…"}</p>
  {:else if state.readiness_error}
    <div class="state-panel">
      <p>{state.readiness_error}</p>
      {#each state.profile.problems as problem}<p><code>{problem.code}</code> — {problem.message}</p>{/each}
      <h3>{t.planningSetup || "Planning setup required"}</h3>
      <label>{t.currency || "Currency"} <input bind:value={setupCurrency}></label>
      <label>{t.timezone || "Timezone"} <input bind:value={setupTimezone}></label>
      <label>{t.minimumReserve || "Minimum reserve"} <input inputmode="decimal" bind:value={setupReserve}></label>
      <label>{t.spendableAccounts || "Spendable accounts (comma separated)"} <input bind:value={setupFunds}></label>
      <label>{t.debtAccounts || "Short-term debt accounts (comma separated)"} <input bind:value={setupDebts}></label>
      <label>{t.recordedThrough || "Recorded through"} <input type="date" bind:value={setupRecorded}></label>
      <button type="button" onclick={() => previewGenerated("profile")}>{t.preview || "Preview"}</button>
    </div>
  {:else if state.result}
    <div class="state-panel">
      <h3>{t.safeToSpend || "Safe to spend"}: {state.result.primary_safe_to_spend} {state.result.currency}</h3>
      <p>{t.lowPoint || "Lowest point"}: {state.result.liquidity_low_point.date} — {state.result.liquidity_low_point.headroom}</p>
      {#if !state.result.current_planning}<p>{t.planningStale || "Recorded-through date is not today; this is a stale estimate."}</p>{/if}
      {#if state.result.funding_shortfall !== "0"}<p>{t.fundingShortfall || "Funding shortfall"}: {state.result.funding_shortfall} {state.result.currency}</p>{/if}
    </div>
    <h3>{t.addPlan || "Add a planned flow"}</h3>
    <div class="state-panel">
      <label>{t.planID || "Plan ID"} <input bind:value={planID}></label><label>{t.name || "Name"} <input bind:value={planName}></label><label>{t.amount || "Amount"} <input inputmode="decimal" bind:value={planAmount}></label><label>{t.date || "Date"} <input type="date" bind:value={planDate}></label>
      <label>{t.direction || "Direction"} <select bind:value={planDirection}><option value="outflow">{t.outflow || "Outflow"}</option><option value="inflow">{t.inflow || "Inflow"}</option></select></label>
      {#if planDirection === "outflow"}<label>{t.commitment || "Commitment"} <select bind:value={planCommitment}><option value="committed">{t.committed || "Committed"}</option><option value="adjustable">{t.adjustable || "Adjustable"}</option></select></label>{/if}
      <label>{t.account || "Account"} <input bind:value={planAccount}></label><button type="button" onclick={() => previewGenerated("plan")}>{t.preview || "Preview"}</button>
    </div>
    {#if state.profile.plans.length > 0}<h3>{t.resolvePlan || "Resolve an active plan"}</h3><div class="state-panel"><label>{t.planID || "Plan ID"} <select bind:value={resolvePlanID}><option value="">{t.choose || "Choose"}</option>{#each state.profile.plans as plan}<option value={plan.id}>{plan.name} — {plan.date} ({plan.amount})</option>{/each}</select></label><label>{t.resolution || "Resolution"} <select bind:value={resolvePlanStatus}><option value="fulfilled">{t.fulfilled || "Fulfilled"}</option><option value="cancelled">{t.cancelled || "Cancelled"}</option><option value="active">{t.reschedule || "Reschedule"}</option></select></label>{#if resolvePlanStatus === "active"}<label>{t.date || "Date"} <input type="date" bind:value={resolvePlanDate}></label>{/if}<button type="button" onclick={() => previewGenerated("plan")}>{t.preview || "Preview"}</button></div>{/if}

    <h3>{t.purchaseScenario || "Test a large purchase"}</h3>
    <div class="state-panel">
      <label>{t.name || "Name"} <input bind:value={purchaseName} placeholder={t.purchaseName || "Proposed purchase"}></label>
      <label>{t.amount || "Amount"} <input inputmode="decimal" bind:value={purchaseAmount}></label>
      <label>{t.date || "Date"} <input type="date" bind:value={purchaseDate}></label>
      <label><input type="checkbox" bind:checked={includeInflows}> {t.includeInflows || "Include future inflows"}</label>
      <label><input type="checkbox" bind:checked={releaseAdjustable}> {t.releaseAdjustable || "Release adjustable outflows"}</label>
      <button type="button" onclick={evaluatePurchase}>{t.evaluate || "Evaluate"}</button>
      {#if scenarioError}<p class="error-panel">{scenarioError}</p>{/if}
      {#if scenario}<p>{t.scenarioLowPoint || "Scenario low point"}: {scenario.liquidity_low_point.date} — {scenario.liquidity_low_point.headroom} {scenario.currency}</p>{#if scenario.funding_shortfall !== "0"}<p>{t.fundingShortfall || "Funding shortfall"}: {scenario.funding_shortfall} {scenario.currency}</p>{/if}<p>{t.scenarioEphemeral || "This trial is not saved to the ledger."}</p>{/if}
    </div>

    <table>
      <thead><tr><th>{t.date || "Date"}</th><th>{t.safeToSpend || "Headroom"}</th><th>{t.committed || "Committed"}</th><th>{t.adjustable || "Adjustable"}</th><th>{t.inflows || "Future inflows"}</th></tr></thead>
      <tbody>{#each state.result.timeline as point}<tr><td>{point.date}</td><td>{point.headroom}</td><td>{point.committed_outflow}</td><td>{point.adjustable_outflow}</td><td>{point.ignored_inflows}</td></tr>{/each}</tbody>
    </table>
    <h3>{t.cycleReview || "Cycle review"}</h3>
    <div class="state-panel">
      <button type="button" onclick={loadReview}>{t.inspectReview || "Inspect review evidence"}</button>
      {#if reviewError}<p class="error-panel">{reviewError}</p>{/if}
      {#if review}
        <p>{t.reviewConfirmation || "Suggestions are evidence only; nothing is fulfilled automatically."}</p>
        {#if review.completed}<p>{review.stale ? (t.reviewStale || "The completed review is stale and needs reconfirmation.") : (t.reviewComplete || "This review is completed.")}</p>{/if}
        <h4>{t.matchSuggestions || "Plan-to-actual suggestions"}</h4>
        {#if review.matches.length === 0}<p>{t.none || "None"}</p>{:else}<ul>{#each review.matches as match}<li><label><input type="checkbox" checked={selectedMatches.includes(`${match.plan_id}\u0000${match.occurrence_id}`)} onchange={(event) => toggleMatch(`${match.plan_id}\u0000${match.occurrence_id}`, event.currentTarget.checked)}> {match.plan_id} ↔ {match.occurrence_id} ({match.date_distance}d; {match.same_name ? t.name || "name" : ""} {match.same_account ? t.account || "account" : ""} {match.same_amount ? t.amount || "amount" : ""})</label></li>{/each}</ul>{/if}
        <h4>{t.recurringCandidates || "Recurring candidates"}</h4>
        {#if review.patterns.length === 0}<p>{t.none || "None"}</p>{:else}<ul>{#each review.patterns as pattern}<li>{pattern.name} — {pattern.account}; {pattern.cadence_days}d; {pattern.minimum_amount}–{pattern.maximum_amount}; {pattern.occurrences.length} {t.occurrences || "occurrences"} <button type="button" onclick={() => useCandidate(pattern.name, pattern.maximum_amount)}>{t.useCandidate || "Use as a plan draft"}</button></li>{/each}</ul>{/if}
        <button type="button" onclick={previewCompletion}>{t.previewReview || "Preview review completion"}</button>
        {#if reviewPreview}<pre>{reviewPreview.content}</pre><button type="button" onclick={commitCompletion}>{t.commitReview || "Save reviewed completion"}</button>{/if}
      {/if}
    </div>
  {/if}
  {#if generatedPreview}<div class="state-panel"><h3>{t.reviewedWrite || "Review change"}</h3><pre>{generatedPreview.content}</pre><button type="button" onclick={commitGenerated}>{t.commit || "Commit"}</button></div>{/if}
</section>
