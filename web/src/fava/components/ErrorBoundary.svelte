<script lang="ts">
  export let message: string | null = null;
  export let onRetry: () => void;
</script>

{#if message}
  <section class="state-panel error-panel" role="alert">
    <h2>Unable to load this local view</h2>
    <p>{message}</p>
    <button type="button" on:click={onRetry}>Try again</button>
  </section>
{:else}
  <slot />
{/if}
