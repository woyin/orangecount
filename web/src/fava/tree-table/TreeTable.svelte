<script lang="ts">
  import { currenciesInTree, type TreeNode } from "../reports/types";
  import TreeTableNode from "./TreeTableNode.svelte";

  export let tree: TreeNode;
  export let end: string | null = null;

  let collapsed = new Set<string>();
  $: currencies = currenciesInTree(tree);
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
      {#each currencies as currency (currency)}
        <span class="num" title={currency}>{currency}</span>
      {/each}
    </p>
  </li>
  {#each roots as node (node.account)}
    <TreeTableNode node={node} {currencies} {collapsed} onToggle={toggle} />
  {/each}
</ol>

<style>
  .account-cell {
    display: block;
    flex: 1;
    min-width: 14em;
  }

  .num {
    min-width: 7em;
  }
</style>
