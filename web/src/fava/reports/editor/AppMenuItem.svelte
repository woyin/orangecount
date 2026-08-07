<!--
  @component
  A single top level menu item in an app menu.

  The default slot should filled with its vertically arranged sub-items.
-->
<script lang="ts">
  export let name: string;

  let open = false;
</script>

<span
  class:open
  tabindex="0"
  role="menuitem"
  on:blur={() => {
    open = false;
  }}
  on:keydown={(event) => {
    if (event.key === "Escape") {
      open = false;
    } else if (event.key === "ArrowDown") {
      open = true;
    }
  }}
>
  {name}
  <ul role="menu">
    <slot></slot>
  </ul>
</span>

<style>
  span {
    padding: 0.7em 0.5em;
    cursor: pointer;
  }

  span.open,
  span:hover {
    background-color: var(--background-darkest);
  }

  span::after {
    content: "▾";
  }

  ul {
    position: absolute;
    z-index: var(--z-index-floating-ui);
    display: none;
    width: 500px;
    max-height: 400px;
    margin: 0.7em 0 0 -0.5em; /* The top margin should match the (bottom) padding of the span above. */
    overflow-y: auto;
    background-color: var(--background);
    border: 1px solid var(--border);
    box-shadow: var(--box-shadow-dropdown);
  }

  span.open > ul,
  span:hover > ul {
    display: block;
  }
</style>