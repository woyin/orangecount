<!--
  @component
  An application menu.
-->
<script lang="ts">
</script>

<div role="menubar">
  <slot></slot>
</div>

<style>
  div {
    display: flex;
    gap: 0.5em;
    align-items: stretch;
    margin-right: 0.5em;
  }
</style>