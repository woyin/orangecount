<script lang="ts">
  import type { EditorView } from "@codemirror/view";
  import type { SourceNode } from "../../lib/sources";
  import { modKey } from "../../keyboard-shortcuts";
  import { toggleComment } from "@codemirror/commands";
  import { foldAll, unfoldAll } from "@codemirror/language";
  import AppMenu from "./AppMenu.svelte";
  import AppMenuItem from "./AppMenuItem.svelte";
  import AppMenuSubItem from "./AppMenuSubItem.svelte";
  import Key from "./Key.svelte";
  import Sources from "./Sources.svelte";

  export let sources_tree: SourceNode;
  export let file_path: string;
  export let editor: EditorView | null;
  export let on_open_file: (path: string) => void;
</script>

<div>
  <AppMenu>
    <AppMenuItem name="File">
      <Sources
        is_root
        node={sources_tree}
        on_select={on_open_file}
        selected={file_path}
      ></Sources>
    </AppMenuItem>
    {#if editor}
      <AppMenuItem name="Edit">
        <AppMenuSubItem action={() => toggleComment(editor!)}>
          Toggle Comment (selection)
          <span slot="right"><Key key={[modKey, "/"]} /></span>
        </AppMenuSubItem>
        <AppMenuSubItem action={() => unfoldAll(editor!)}>
          Open all folds
          <span slot="right"><Key key={["Ctrl", "Alt", "]"]} /></span>
        </AppMenuSubItem>
        <AppMenuSubItem action={() => foldAll(editor!)}>
          Close all folds
          <span slot="right"><Key key={["Ctrl", "Alt", "["]} /></span>
        </AppMenuSubItem>
      </AppMenuItem>
    {/if}
  </AppMenu>
  <slot></slot>
</div>

<style>
  div {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    background: var(--sidebar-background);
    border-bottom: 1px solid var(--sidebar-border);
  }
</style>