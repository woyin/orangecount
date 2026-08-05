<!--
  @component
  An autocomplete input for fuzzy selection of suggestions.

  Adapted from Fava's frontend/src/AutocompleteInput.svelte (MIT license,
  pinned reference revision). The upstream component uses Svelte 5 runes
  ($props, $state, $bindable, $props.id) that are unavailable in the pinned
  Svelte version, so it is expressed with the classic component API. The
  combobox behavior (ARIA APG editable combobox with list autocomplete),
  fuzzy filtering, keyboard handling, and markup match upstream.
-->
<script lang="ts">
  import { createEventDispatcher } from "svelte";

  import { keyboardShortcut, type KeySpec } from "../keyboard-shortcuts";
  import { fuzzyfilter, fuzzywrap } from "../lib/fuzzy";

  export let value: string;
  export let placeholder: string;
  export let suggestions: readonly string[];
  export let valueExtractor: ((val: string, input: HTMLInputElement) => string) | undefined = undefined;
  export let valueSelector: ((val: string, input: HTMLInputElement) => string) | undefined = undefined;
  export let setSize = false;
  export let key: KeySpec | undefined = undefined;
  export let clearButton = false;
  export let onBlur: ((el: HTMLInputElement) => void) | undefined = undefined;
  export let onEnter: ((el: HTMLInputElement) => void) | undefined = undefined;
  export let onSelect: ((el: HTMLInputElement) => void) | undefined = undefined;

  const dispatch = createEventDispatcher<{ change: string }>();

  let hidden = true;
  let index = -1;
  let input: HTMLInputElement | undefined = undefined;
  let uid = Math.random().toString(36).slice(2);
  const autocompleteId = `combobox-autocomplete-${uid}`;

  $: size = setSize ? Math.max(value.length, placeholder.length) + 1 : undefined;
  $: extractedValue = input && valueExtractor ? valueExtractor(value, input) : value;
  $: filteredSuggestions = (() => {
    const filtered = fuzzyfilter(extractedValue, suggestions)
      .slice(0, 30)
      .map((suggestion) => ({
        suggestion,
        fuzzywrapped: fuzzywrap(extractedValue, suggestion),
      }));
    return filtered.length === 1 && filtered[0]?.suggestion === extractedValue
      ? []
      : filtered;
  })();
  $: index = Math.min(index, filteredSuggestions.length - 1);
  $: expanded = !hidden && filteredSuggestions.length > 0;

  function setValue(next: string) {
    value = next;
    dispatch("change", value);
  }

  function select(suggestion: string) {
    setValue(input && valueSelector ? valueSelector(suggestion, input) : suggestion);
    if (input) {
      onSelect?.(input);
    }
    hidden = true;
  }

  function mousedown(event: MouseEvent, suggestion: string) {
    if (event.button === 0) {
      select(suggestion);
    }
  }

  function keydown(event: KeyboardEvent) {
    if (event.key === "Enter") {
      const suggestion = filteredSuggestions[index]?.suggestion;
      if (index > -1 && !hidden && suggestion != null) {
        event.preventDefault();
        select(suggestion);
      } else if (input) {
        onEnter?.(input);
      }
    } else if (event.key === " " && event.ctrlKey) {
      hidden = false;
    } else if (event.key === "Escape") {
      event.stopPropagation();
      if (expanded) {
        index = -1;
        hidden = true;
      } else {
        setValue("");
      }
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      index = index === 0 ? filteredSuggestions.length - 1 : index - 1;
    } else if (event.key === "ArrowDown") {
      event.preventDefault();
      index = index === filteredSuggestions.length - 1 ? 0 : index + 1;
    }
  }
</script>

<span>
  <input
    type="text"
    autocomplete="off"
    role="combobox"
    aria-expanded={expanded}
    aria-controls={autocompleteId}
    {value}
    bind:this={input}
    use:keyboardShortcut={key}
    on:blur={(event) => {
      hidden = true;
      onBlur?.(event.currentTarget);
    }}
    on:focus={() => {
      hidden = false;
    }}
    on:input={(event) => {
      setValue((event.currentTarget as HTMLInputElement).value);
      hidden = false;
    }}
    on:keydown={keydown}
    {placeholder}
    {size}
  />
  {#if clearButton && value}
    <button
      type="button"
      tabindex={-1}
      class="muted round"
      on:click={() => {
        setValue("");
        if (input) {
          onSelect?.(input);
        }
      }}
    >
      ×
    </button>
  {/if}
  {#if filteredSuggestions.length}
    <ul {hidden} role="listbox" id={autocompleteId}>
      {#each filteredSuggestions as { fuzzywrapped, suggestion }, i (suggestion)}
        <li
          role="option"
          aria-selected={i === index}
          class:selected={i === index}
          on:mousedown={(event) => {
            mousedown(event, suggestion);
          }}
        >
          {#each fuzzywrapped as [type, text], j (j)}
            {#if type === "text"}
              {text}
            {:else}
              <span>{text}</span>
            {/if}
          {/each}
        </li>
      {/each}
    </ul>
  {/if}
</span>

<style>
  span {
    position: relative;
    display: inline-block;
    flex: var(--autocomplete-wrapper-flex, initial);
  }

  input {
    width: 100%;
  }

  ul {
    position: var(--autocomplete-list-position, absolute);
    z-index: var(--z-index-autocomplete);
    overflow: hidden auto;
    background-color: var(--background);
    border: 1px solid var(--border-darker);
    box-shadow: var(--box-shadow-dropdown);
  }

  li {
    min-width: 8rem;
    padding: 0 0.5em;
    white-space: nowrap;
    cursor: pointer;
  }

  li.selected,
  li:hover {
    color: var(--background);
    background-color: var(--link-color);
  }

  button {
    position: absolute;
    top: 8px;
    right: 4px;
    background: transparent;
  }

  li span {
    height: 1.2em;
    padding: 0 0.05em;
    margin: 0 -0.05em;
    background-color: var(--autocomplete-match);
    border-radius: 2px;
  }

  @media print {
    button {
      display: none;
    }
  }
</style>
