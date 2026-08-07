<!--
  @component
  A sub-item in an app menu.
-->
<script lang="ts">
  export let action: () => void;
  export let title: string = "";
  export let selected: boolean = false;
</script>

<li
  class:selected
  {title}
  role="menuitem"
  on:click={action}
  on:keydown={(event) => {
    if (event.key === "Enter") {
      action();
    }
  }}
>
  <slot></slot>
  <span class="aspect">
    <slot name="right"></slot>
  </span>
</li>

<style>
  .selected::before {
    content: "›";
  }

  li {
    padding: 0.25em 0.5em;
  }

  span {
    float: right;
  }

  li:hover,
  li:focus-visible {
    background-color: var(--background-darkest);
  }
</style>