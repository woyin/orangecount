<script lang="ts">
  export let active = false;
</script>

{#if active}
  <div class="state-panel loading-panel" role="status" aria-live="polite">
    <span class="spinner" aria-hidden="true"></span>
    <span>Loading local view…</span>
  </div>
{:else}
  <slot />
{/if}
