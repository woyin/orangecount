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

  const dispatch = createEventDispatcher<{ select: string; move: { account: string; filename: string } }>();

  $: account = node.name;
  $: leafName = account.split(":").pop() || account;
  $: hasChildren = node.children.length > 0;
  $: isToggled = toggled.has(account);
  $: selected = selectedAccount === account;

  let drag = false;

  function dragenter(event: DragEvent) {
    const types = event.dataTransfer?.types ?? [];
    if (types.includes("fava/filename")) {
      event.preventDefault();
      drag = true;
    }
  }

  function drop(event: DragEvent) {
    event.preventDefault();
    const filename = event.dataTransfer?.getData("fava/filename");
    if (filename != null && filename !== "") {
      dispatch("move", { account, filename });
      drag = false;
    }
  }

  function toggle() {
    toggled = isToggled ? new Set([...toggled].filter((name) => name !== account)) : new Set([...toggled, account]);
  }

  function select() {
    dispatch("select", selected ? "" : account);
  }
</script>

{#if account}
  <p
    title={account}
    class="droptarget"
    data-account-name={account}
    class:selected
    class:drag
    on:dragenter={dragenter}
    on:dragover={dragenter}
    on:dragleave={() => {
      drag = false;
    }}
    on:drop={drop}
  >
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
        <svelte:self node={child} bind:toggled {selectedAccount} on:select on:move />
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

  .selected,
  .drag {
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
