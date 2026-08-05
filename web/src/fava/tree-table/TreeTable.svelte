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
