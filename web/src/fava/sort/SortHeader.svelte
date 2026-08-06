<script lang="ts">
  import { Sorter, UnsortedColumn, type SortColumn } from "./index";

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  export let sorter: Sorter<any>;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  export let column: SortColumn<any>;
  export let numeric = false;

  $: sortable = !(column instanceof UnsortedColumn);
  $: active = column === sorter.column;
</script>

{#if sortable}
  <th
    class:num={numeric}
    aria-sort={active ? (sorter.order === "asc" ? "ascending" : "descending") : undefined}
  >
    <button type="button" class="sort-toggle" on:click={() => (sorter = sorter.switchColumn(column))}>
      {column.name}{#if active}<span aria-hidden="true">{sorter.order === "asc" ? " ▲" : " ▼"}</span>{/if}
    </button>
  </th>
{:else}
  <th class:num={numeric}>{column.name}</th>
{/if}

<style>
  .num {
    text-align: right;
  }

  .sort-toggle {
    padding: 0;
    font: inherit;
    color: inherit;
    cursor: pointer;
    background: transparent;
    border: 0;
  }
</style>
