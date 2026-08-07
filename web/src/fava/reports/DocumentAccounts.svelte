<script lang="ts">
  import { createEventDispatcher } from "svelte";

  interface AccountNode {
    name: string;
    count: number;
    children: AccountNode[];
  }

  export let node: AccountNode;
  export let toggled: Set<string>;
  export let selectedAccount: string;

  const dispatch = createEventDispatcher<{ select: string }>();

  $: account = node.name;
  $: leafName = account.split(":").pop() || account;
  $: hasChildren = node.children.length > 0;
  $: isToggled = toggled.has(account);
  $: selected = selectedAccount === account;

  function toggle() {
    toggled = isToggled ? new Set([...toggled].filter((name) => name !== account)) : new Set([...toggled, account]);
  }

  function select() {
    dispatch("select", selected ? "" : account);
  }
</script>

{#if account}
  <p title={account} class:selected>
    {#if hasChildren}
      <button type="button" class="unset toggle" on:click|stopPropagation={toggle}>{isToggled ? "▸" : "▾"}</button>
    {/if}
    <button type="button" class="unset leaf" on:click={select}>{leafName}</button>
    {#if node.count > 0}
      <span class="count">{node.count}</span>
    {/if}
  </p>
{/if}
{#if hasChildren && !isToggled}
  <ul>
    {#each node.children as child (child.name)}
      <li>
        <svelte:self node={child} bind:toggled {selectedAccount} on:select />
      </li>
    {/each}
  </ul>
{/if}

<style>
  ul {
    padding: 0 0 0 0.5em;
    margin: 0;
    list-style: none;
  }

  p {
    position: relative;
    display: flex;
    padding-right: 0.5em;
    margin: 0;
    overflow: hidden;
    border-bottom: 1px solid var(--table-border);
    border-left: 1px solid var(--table-border);
  }

  p > * {
    padding: 1px;
  }

  .count {
    opacity: 0.6;
  }

  .selected {
    background-color: var(--table-header-background);
  }

  .leaf {
    flex-grow: 1;
    margin-left: 1em;
  }

  .toggle {
    position: absolute;
    margin: 0 0.25rem;
    color: var(--treetable-expander);
  }
</style>
