  <script lang="ts">
  import type { SourceNode } from "../../lib/sources";
  import { expanded_directories, toggle_directory } from "./stores";
  export let is_root: boolean = false;
  export let node: SourceNode;
  export let on_select: (source: string) => void;
  export let selected: string;

  $: is_directory = node.children.length > 0;
  // Even though root is always expanded, treat it as collapsed by default. This
  // allows for expanding everything with one Ctrl-/Meta-Click. The subsequent
  // click would then collapse everything.
  $: is_expanded = $expanded_directories.get(node.path) ?? (!is_root && selected.startsWith(node.path));
  // Show where the selected file would be, if directories are collapsed
  $: is_selected = is_directory && !is_expanded && !is_root
    ? selected.startsWith(node.path)
    : selected === node.path;

  function action(event: MouseEvent) {
    if (is_directory) {
      toggle_directory(node.path, !is_expanded, event);
    } else {
      on_select(node.path);
    }
    event.stopPropagation();
  }
</script>

<li class:selected={is_selected} role="menuitem">
  {#if is_root}
    <button
      type="button"
      title={"Beancount data root directory\nShift-Click to expand/collapse immediate directories\nCtrl-/Cmd-/Meta-Click to expand/collapse all directories."}
      class="unset root"
      on:click={action}
    >
      {node.name}
    </button>
  {:else}
    <p>
      {#if is_directory}
        <button type="button" class="unset toggle" on:click={action}>{is_expanded ? "▾" : "▸"}</button>
      {/if}
      <button type="button" class="unset leaf" on:click={action}>{node.name}</button>
    </p>
  {/if}
  {#if is_directory && (is_expanded || is_root)}
    <ul>
      {#each node.children as child (child.path)}
        <svelte:self node={child} {on_select} {selected} />
      {/each}
    </ul>
  {/if}
</li>

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

  .root {
    margin: 0 0.25rem;
  }
</style>
