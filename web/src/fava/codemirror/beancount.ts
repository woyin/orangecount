import { indentUnit, syntaxHighlighting } from "@codemirror/language";
import type { EditorState } from "@codemirror/state";
import type { KeyBinding } from "@codemirror/view";
import { EditorView, keymap } from "@codemirror/view";

import { base_extensions } from "./base-extensions";
import { beancount_highlight } from "./beancount-highlight";
import { beancount_language_support } from "./beancount-language";
import { ruler_plugin } from "./ruler";

export {
  replace_contents,
  scroll_to_line,
  set_errors,
} from "./editor-transactions";

/**
 * A Beancount editor.
 */
export function init_beancount_editor(
  value: string,
  onDocChanges: (s: EditorState) => void,
  commands: KeyBinding[],
  indent: number,
  currency_column: number,
): EditorView {
  return new EditorView({
    doc: value,
    extensions: [
      beancount_language_support,
      indentUnit.of(" ".repeat(indent)),
      ...(currency_column ? [ruler_plugin(currency_column - 1)] : []),
      keymap.of(commands),
      EditorView.updateListener.of((update) => {
        if (update.docChanged) {
          onDocChanges(update.state);
        }
      }),
      base_extensions,
      syntaxHighlighting(beancount_highlight),
    ],
  });
}
