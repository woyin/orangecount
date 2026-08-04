<script lang="ts">
  import { formatAmount, type TreeNode } from "../reports/types";

  export let node: TreeNode;
  export let currencies: string[] = [];
  export let collapsed: Set<string>;
  export let depth = 0;
  export let onToggle: (account: string) => void;

  $: isCollapsed = collapsed.has(node.account);
  $: shown = isCollapsed ? node.balance_children : node.balance;
  $: hasChildren = node.children.length > 0;
  $: leaf = node.account.includes(":") ? node.account.slice(node.account.lastIndexOf(":") + 1) : node.account;
</script>

<li>
  <p>
    <span class="account-cell" style={`--account-indent: ${depth}em`}>
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
        {formatAmount(shown[currency])}
      </span>
    {/each}
  </p>
  {#if !isCollapsed && hasChildren}
    <ol>
      {#each node.children as child (child.account)}
        <svelte:self node={child} {currencies} {collapsed} depth={depth + 1} {onToggle} />
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

  .dimmed {
    color: var(--text-color-lightest);
  }
</style>
