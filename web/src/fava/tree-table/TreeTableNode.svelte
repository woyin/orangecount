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
  import { formatAmount, type TreeNode } from "../reports/types";

  export let node: TreeNode;
  export let currencies: string[] = [];
  export let otherCurrencies: string[] = [];
  export let renderCommas = false;
  export let collapsed: Set<string>;
  export let depth = 0;
  export let onToggle: (account: string) => void;

  $: isCollapsed = collapsed.has(node.account);
  // Fava shows a node's own balance only when it actually has postings, and
  // otherwise falls back to the subtree total; a collapsed node always reports
  // its subtree. Showing node.balance unconditionally left every pure-parent
  // account (Income, Expenses, Health, …) displaying its own empty 0 balance.
  $: shown = isCollapsed || !node.has_txns ? node.balance_children : node.balance;
  $: hasChildren = node.children.length > 0;
  $: leaf = node.account.includes(":") ? node.account.slice(node.account.lastIndexOf(":") + 1) : node.account;
  // Non-operating currencies share one cell, each on its own line and suffixed
  // with its currency code, the way Fava renders its "Other" column.
  $: otherAmounts = otherCurrencies
    .filter((currency) => shown[currency])
    .map((currency) => `${formatAmount(shown[currency], renderCommas)} ${currency}`);
</script>

<li>
  <p>
    <span class="account-cell droptarget" data-account-name={node.account} style={`--account-indent: ${depth}em`}>
      {#if hasChildren}
        <button
          type="button"
          class="unset expander"
          aria-label={isCollapsed ? `Expand ${node.account}` : `Collapse ${node.account}`}
          aria-expanded={!isCollapsed}
          onclick={() => onToggle(node.account)}
        >{isCollapsed ? "▸" : "▾"}</button>
      {/if}
      <a href={`/account/${encodeURIComponent(node.account)}`} class="account">{leaf}</a>
    </span>
    {#each currencies as currency (currency)}
      <span class="num" class:dimmed={!shown[currency]} title={shown[currency]?.exact ?? ""}>
        {formatAmount(shown[currency], renderCommas)}
      </span>
    {/each}
    {#if otherCurrencies.length}
      <span class="other num">
        {#each otherAmounts as amount (amount)}<span class="other-line">{amount}</span>{/each}
      </span>
    {/if}
  </p>
  {#if !isCollapsed && hasChildren}
    <ol>
      {#each node.children as child (child.account)}
        <svelte:self node={child} {currencies} {otherCurrencies} {renderCommas} {collapsed} depth={depth + 1} {onToggle} />
      {/each}
    </ol>
  {/if}
</li>

<style>
  .account-cell {
    display: flex;
    flex: 1;
    align-items: center;
    min-width: 14em;
    padding-left: var(--account-indent, 0em);
  }

  .account {
    margin-left: 1em;
  }

  .expander {
    position: absolute;
    padding: 0 3px;
    color: var(--treetable-expander);
  }

  .num {
    min-width: 7em;
  }

  .other-line {
    display: block;
    white-space: nowrap;
  }

  .dimmed {
    color: var(--text-color-lightest);
  }
</style>
