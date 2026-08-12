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
  import { translations, type Locale } from "../../translations";
  import AutocompleteInput from "./AutocompleteInput.svelte";
  import { keyboardShortcut, type KeySpec } from "../keyboard-shortcuts";
  import { ROUTES, pageLabel, routeHref } from "../router.mjs";

  export let route: string;
  export let open = false;
  export let errors: unknown[] = [];
  export let locale = "en";
  export let accounts: string[] = [];
  export let userQueries: { name: string; query_string: string }[] = [];
  export let onMenu: () => void;
  export let onNavigate: (href: string) => void;

  // Long saved-query names are shortened in the submenu; the slice offset
  // mirrors the pinned AsideContents behaviour.
  function truncateQueryName(name: string): string {
    return name.length < 25 ? name : `${name.slice(25)}…`;
  }

  let gotoAccount = "";
  function selectAccount(el: HTMLInputElement) {
    if (gotoAccount) {
      onNavigate(`/account/${encodeURIComponent(gotoAccount)}`);
      el.blur();
      gotoAccount = "";
    }
  }

  const sections = [
    ["", ["income_statement", "balance_sheet", "trial_balance", "journal", "query"]],
    ["", ["holdings", "commodities", "documents", "events", "statistics"]],
    ["", ["editor", "import", "options", "help"]],
    ["OrangeCount", ["account", "quick-profile"]],
  ];
  const known = new Set([...ROUTES, "account"]);
  const shortcuts: Record<string, KeySpec> = {
    income_statement: "g i",
    balance_sheet: "g b",
    trial_balance: "g t",
    journal: "g j",
    query: "g q",
    holdings: "g h",
    commodities: "g c",
    documents: "g d",
    events: "g E",
    statistics: "g s",
    editor: "g e",
    import: "g n",
    options: "g o",
    help: "g H",
  };
  const keys: Record<string, string> = { income_statement: "incomeStatement", balance_sheet: "balanceSheet", trial_balance: "trialBalance", journal: "journal", query: "query", holdings: "holdings", commodities: "commodities", documents: "documents", events: "events", statistics: "statistics", editor: "editor", import: "import", options: "options", help: "help", diagnostics: "diagnostics", account: "accounts" };
  function label(routeName: string): string { const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale]; return catalog[keys[routeName] || ""] || pageLabel(routeName); }
  function t(key: string): string { const catalog = translations[(locale === "zh-CN" ? "zh-CN" : "en") as Locale]; return catalog[key] || key; }
</script>

{#if open}
  <div class="overlay" onclick={onMenu} aria-hidden="true"></div>
{/if}
<div class:active={open} class="aside-buttons">
  <button id="menu-toggle" type="button" aria-controls="sidebar" aria-expanded={open} aria-label="Menu" onclick={onMenu}>☰</button>
  <a class="button" href="#add-transaction" aria-label="Add transaction">+</a>
  <a class="button quick-btn" href="#add-quick" aria-label="Quick entry" title="Quick entry (a q)" use:keyboardShortcut={"a q"}>⚡</a>
</div>
<aside id="sidebar" class:active={open} aria-label="Primary navigation">
  {#each sections as [heading, items], sectionIndex}
    <ul class="navigation" aria-label={heading || "Reports"}>
      {#if heading}
        <li class="navigation-heading" aria-hidden="true">{heading}</li>
      {/if}
      {#each items as item}
        {#if known.has(item)}
          <li>
            <a
              href={routeHref(item)}
              class:selected={route === item}
              aria-current={route === item ? "page" : undefined}
              data-route={item}
              use:keyboardShortcut={shortcuts[item]}
              onclick={(event) => { event.preventDefault(); onNavigate(routeHref(item)); }}
            >{label(item)}</a>
            {#if item === "import"}
              <a href="#export" class="secondary" title={t("export")} aria-label={t("export")}>&#11015;</a>
            {/if}
            {#if item === "query" && userQueries.length}
              <ul class="submenu">
                {#each userQueries as saved (saved.query_string)}
                  <li>
                    <a
                      href={`/query?query_string=${encodeURIComponent(saved.query_string)}`}
                      onclick={(event) => { event.preventDefault(); onNavigate(`/query?query_string=${encodeURIComponent(saved.query_string)}`); }}
                    >{truncateQueryName(saved.name)}</a>
                  </li>
                {/each}
              </ul>
            {/if}
          </li>
        {/if}
      {/each}
      {#if sectionIndex === 0}
        <li class="account-selector">
          <AutocompleteInput
            bind:value={gotoAccount}
            placeholder={t("goToAccount")}
            suggestions={accounts}
            key="g a"
            onSelect={selectAccount}
            onEnter={selectAccount}
          />
        </li>
      {/if}
    </ul>
    {#if sectionIndex === sections.length - 1 && errors.length}
      <ul class="navigation">
        <li><a href={routeHref("diagnostics")} class:selected={route === "diagnostics"} aria-current={route === "diagnostics" ? "page" : undefined} onclick={(event) => { event.preventDefault(); onNavigate(routeHref("diagnostics")); }}>{label("diagnostics")} ({errors.length})</a></li>
        <li><a href={routeHref("errors")} class:selected={route === "errors"} aria-current={route === "errors" ? "page" : undefined} onclick={(event) => { event.preventDefault(); onNavigate(routeHref("errors")); }}>Errors ({errors.length})</a></li>
      </ul>
    {/if}
  {/each}
</aside>

<style>
  aside {
    grid-area: aside;
    padding-top: 0.5rem;
    margin: 0;
    overflow-y: auto;
    color: var(--sidebar-color);
    background-color: var(--sidebar-background);
    border-right: 1px solid var(--sidebar-border);
  }

  .aside-buttons {
    display: none;
  }

  @media (width <= 767px) {
    :global(:root) {
      --aside-width: 200px;
    }

    aside {
      position: fixed;
      top: 0;
      bottom: 0;
      visibility: hidden;
      z-index: var(--z-index-floating-ui);
      width: 200px;
      margin-left: -200px;
      transition: var(--transitions);
    }

    .overlay {
      position: fixed;
      inset: 0;
      z-index: var(--z-index-floating-ui);
      cursor: pointer;
      background: var(--overlay-wrapper-background);
      transition: var(--transitions);
    }

    aside.active {
      margin-left: 0;
      visibility: visible;
    }

    .aside-buttons {
      position: fixed;
      top: 0;
      left: 0;
      z-index: var(--z-index-floating-ui);
      display: flex;
      flex-direction: column;
      transition: var(--transitions);
    }

    .active.aside-buttons {
      left: 200px;
    }

    .aside-buttons > * {
      width: 42px;
      height: 42px;
      color: var(--mobile-button-text);
      text-align: center;
      background-color: var(--sidebar-background);
      border: 1px solid var(--sidebar-border);
    }

    .aside-buttons a {
      font-size: 28px;
    }
  }

  @media print {
    aside,
    .aside-buttons {
      display: none;
    }
  }

  .navigation {
    padding-bottom: 0.5rem;
    margin: 0;
  }

  .navigation + .navigation {
    padding-top: 0.5rem;
    border-top: 1px solid var(--sidebar-border);
  }

  .navigation a.selected,
  .navigation a:hover {
    color: var(--sidebar-hover-color);
    background-color: var(--sidebar-border);
  }

  .navigation li {
    display: flex;
    flex-wrap: wrap;
  }

  .navigation-heading {
    padding: 0.25em 0.5em 0 1em;
    font-size: 0.75em;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    opacity: 0.7;
  }

  .navigation a {
    display: block;
    flex: 1;
    padding: 0.25em 0.5em 0.25em 1em;
    color: inherit;
  }

  .navigation a.secondary {
    flex: none;
    width: 30px;
    padding: 3px 9px;
    line-height: 22px;
    color: inherit;
    background-color: var(--sidebar-background);
  }

  .submenu {
    width: 100%;
    padding: 0;
    margin: 0 0 0.5em;
    list-style: none;
  }

  .submenu a {
    width: 100%;
    padding-left: 35px;
  }

  .submenu li {
    display: block;
    font-size: 0.9em;
  }

  .account-selector {
    --input-border: none;
    --input-padding: 0.25em 0.5em 0.25em 1em;
    --autocomplete-list-position: fixed;
  }

  .account-selector :global(span) {
    flex: 1;
  }

  .aside-buttons button {
    font-size: 1.2rem;
  }
</style>
