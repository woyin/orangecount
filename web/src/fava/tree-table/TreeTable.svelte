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
  import { currenciesInTree, type TreeNode } from "../reports/types";
  import TreeTableNode from "./TreeTableNode.svelte";

  export let tree: TreeNode;
  export let end: string | null = null;
  export let operatingCurrencies: string[] = [];
  export let renderCommas = false;

  let collapsed = new Set<string>();
  $: present = currenciesInTree(tree);
  // Fava gives each declared operating currency its own numeric column and
  // collapses every remaining currency into a single "Other" column, so a
  // ledger holding a dozen commodities still renders a readable table instead
  // of a dozen mostly-empty columns.
  $: columns = operatingCurrencies.filter((currency) => present.includes(currency));
  $: other = present.filter((currency) => !columns.includes(currency));
  $: roots = tree.account === "" ? tree.children : [tree];

  function toggle(account: string) {
    const next = new Set(collapsed);
    if (next.has(account)) next.delete(account);
    else next.add(account);
    collapsed = next;
  }
</script>

<ol class="flex-table tree-table-new" data-tree-table data-end={end ?? undefined}>
  <li class="head">
    <p>
      <span class="account-cell">{tree.account || "Accounts"}</span>
      {#each columns as currency (currency)}
        <span class="num" title={currency}>{currency}</span>
      {/each}
      {#if other.length}
        <span class="other">Other</span>
      {/if}
    </p>
  </li>
  {#each roots as node (node.account)}
    <TreeTableNode node={node} currencies={columns} otherCurrencies={other} {renderCommas} {collapsed} onToggle={toggle} />
  {/each}
</ol>

<style>
  .account-cell {
    display: block;
    flex: 1;
    min-width: 14em;
  }
</style>
