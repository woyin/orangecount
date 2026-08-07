import { derived, get, writable } from "svelte/store";

import { build_compressed_sources_tree, is_directory, parent } from "../../lib/sources";
import { all_matching } from "../../lib/tree";

// The set of beancount source files available to the editor. Populated by
// EditorReport from the flat adapter `paths` list (upstream derives this from
// the sources store/API, which OrangeCount replaces with the adapter paths).
export const editor_sources = writable<Set<string>>(new Set());

export const sources_tree = derived(editor_sources, build_compressed_sources_tree);

// The directories which have been explicitly expanded (true) or collapsed (false).
export const expanded_directories = writable<ReadonlyMap<string, boolean>>(
  new Map(),
);

/**
 * Toggle a directory.
 *
 * If Shift-Click, deeply expand/collapse all descendants.
 * If Ctrl- or Meta-Click, expand/collapse direct children.
 */
export function toggle_directory(
  directory: string,
  expand: boolean,
  event: MouseEvent,
): void {
  const $sources_tree = get(sources_tree);

  expanded_directories.update(($expanded_directories) => {
    const new_expanded_directories = new Map($expanded_directories);
    new_expanded_directories.set(directory, expand);
    if (event.shiftKey) {
      const descendants = all_matching(
        $sources_tree,
        (node) => is_directory(node) && node.path.startsWith(directory),
      );

      for (const node of descendants) {
        new_expanded_directories.set(node.path, expand);
      }
    } else if (event.ctrlKey || event.metaKey) {
      const direct_children = all_matching(
        $sources_tree,
        (node) => is_directory(node) && parent(node.path) === directory,
      );

      for (const node of direct_children) {
        new_expanded_directories.set(node.path, expand);
      }
    }
    return new_expanded_directories;
  });
}
