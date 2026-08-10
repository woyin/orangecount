/* This file is derived from Fava 1.30.12 (commit #aa7538e8971252c9efc52c8a516a3a77d604553f),
which is Copyright (c) 2015-2016 Dominik Aumayr <dominik@aumayr.name> and
distributed under the MIT License. Adapted for OrangeCount; see NOTICE and
web/provenance-manifest.json. The MIT notice is reproduced here:

  Copyright (c) 2015-2016 Dominik Aumayr <dominik@aumayr.name>

  Permission is hereby granted, free of charge, to any person obtaining a copy
  of this software and associated documentation files (the "Software"), to deal
  in the Software without restriction, including without limitation the rights
  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
  copies of the Software, and to permit persons to whom the Software is
  furnished to do so, subject to the following conditions:

  The above copyright notice and this permission notice shall be included in all
  copies or substantial portions of the Software.

  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
  SOFTWARE. */

/**
 * Adapted from Fava's frontend/src/keyboard-shortcuts.ts (MIT license,
 * pinned reference revision). The upstream module uses Svelte 5 attachment
 * APIs that are unavailable in the pinned Svelte version, so the exported
 * `keyboardShortcut` is exposed as a Svelte action instead of an attachment
 * factory. All shortcut semantics (key map handling, two-key sequences,
 * editable-element filtering, tooltip behavior) match upstream.
 */

/**
 * Add a tooltip showing the keyboard shortcut over the target element.
 * @param target - The target element to show the tooltip on.
 * @returns A function to remove event handler.
 */
function showTooltip(target: HTMLElement, description: string): () => void {
  const { hidden } = target;
  if (hidden) {
    target.hidden = false;
  }
  const tooltip = document.createElement("div");
  tooltip.className = "keyboard-tooltip";
  tooltip.textContent = description;
  document.body.appendChild(tooltip);
  const targetRect = target.getBoundingClientRect();
  // Padded 10px to the left if there is space or centered otherwise
  const left =
    targetRect.left +
    Math.min((target.offsetWidth - tooltip.offsetWidth) / 2, 10);
  const top = targetRect.top + (target.offsetHeight - tooltip.offsetHeight) / 2;
  tooltip.style.left = `${left.toString()}px`;
  tooltip.style.top = `${(top + window.scrollY).toString()}px`;
  return () => {
    tooltip.remove();
    if (hidden) {
      target.hidden = true;
    }
  };
}

/**
 * Show all keyboard shortcut tooltips.
 */
function showTooltips(): () => void {
  const removes: (() => void)[] = [];
  document.querySelectorAll("[data-key]").forEach((el) => {
    const key = el.getAttribute("data-key");
    if (el instanceof HTMLElement && key != null) {
      removes.push(showTooltip(el, key));
    }
  });
  return () => {
    removes.forEach((r) => {
      r();
    });
  };
}

/**
 * Ignore events originating from editable elements.
 * @param element - The element to check.
 * @returns true if the element is one of input/select/textarea or a
 *          contentEditable element.
 */
function isEditableElement(element: EventTarget | null): boolean {
  return (
    element instanceof HTMLElement &&
    (element instanceof HTMLInputElement ||
      element instanceof HTMLSelectElement ||
      element instanceof HTMLTextAreaElement ||
      element.isContentEditable)
  );
}

type UppercaseLetter =
  | "A"
  | "B"
  | "C"
  | "D"
  | "E"
  | "F"
  | "G"
  | "H"
  | "I"
  | "J"
  | "L"
  | "M"
  | "N"
  | "O"
  | "P"
  | "Q"
  | "R"
  | "S"
  | "T"
  | "U"
  | "V"
  | "W"
  | "X"
  | "Y"
  | "Z";
type LowercaseLetter = Lowercase<UppercaseLetter>;
type Letter = UppercaseLetter | LowercaseLetter;
// This type can be extended as needed to support all the desired
// key combinations
type KeyCombo =
  | "?"
  | Letter
  | `${"Control" | "Meta"}+${"d" | "s" | "Enter"}`
  // d,s,t - journal filters; f - filters; g - reports
  | `${"d" | "f" | "g" | "s" | "t"} ${Letter}`;
/** A handler function or an element to click. */
type KeyboardShortcutAction = ((event: KeyboardEvent) => void) | HTMLElement;
const keyboardShortcuts = new Map<string, KeyboardShortcutAction>();
// The last typed character to check for sequences of two keys.
let lastChar = "";

/**
 * Handle a `keydown` event on the document.
 *
 * Dispatch to the relevant handler.
 */
function keydown(event: KeyboardEvent): void {
  if (isEditableElement(event.target)) {
    // ignore events in editable elements.
    return;
  }
  let eventKey = event.key;
  if (event.metaKey) {
    eventKey = `Meta+${eventKey}`;
  }
  if (event.altKey) {
    eventKey = `Alt+${eventKey}`;
  }
  if (event.ctrlKey) {
    eventKey = `Control+${eventKey}`;
  }
  const lastTwoKeys = `${lastChar} ${eventKey}`;
  const handler =
    keyboardShortcuts.get(lastTwoKeys) ?? keyboardShortcuts.get(eventKey);
  if (handler) {
    if (handler instanceof HTMLInputElement) {
      event.preventDefault();
      handler.focus();
    } else if (handler instanceof HTMLElement) {
      event.preventDefault();
      handler.click();
    } else {
      handler(event);
    }
  }
  if (event.key !== "Alt" && event.key !== "Control" && event.key !== "Shift") {
    lastChar = eventKey;
  }
}

/** A type to specify a platform-dependent keyboard shortcut. */
export type KeySpec =
  | KeyCombo
  | { key: KeyCombo; mac?: KeyCombo; note?: string };

const isMac =
  // eslint-disable-next-line @typescript-eslint/no-deprecated
  navigator.platform.startsWith("Mac") || navigator.platform === "iPhone";

export const modKey = isMac ? "Cmd" : "Ctrl";

/**
 * Get the keyboard key specifier string for the current platform.
 * @param spec - The key spec.
 */
function getKeySpecKey(spec: KeySpec): KeyCombo {
  if (typeof spec === "string") {
    return spec;
  }
  return isMac ? (spec.mac ?? spec.key) : spec.key;
}

/**
 * Get the keyboard key description.
 * @param spec - The key spec.
 */
function getKeySpecDescription(spec: KeySpec): string {
  if (typeof spec === "string") {
    return spec;
  }
  const key = isMac ? (spec.mac ?? spec.key) : spec.key;
  return spec.note != null ? `${key} - ${spec.note}` : key;
}

/**
 * Bind an event handler to a key.
 * @param spec - The key to bind.
 * @param handler - The callback to run on key press.
 * @returns A function to unbind the keyboard handler.
 */
function bindKey(spec: KeySpec, handler: KeyboardShortcutAction): () => void {
  const key = getKeySpecKey(spec);
  const sequence = key.split(" ");
  if (sequence.length > 2) {
    console.error("Only key sequences of length <=2 are supported: ", key);
  }
  if (keyboardShortcuts.has(key)) {
    console.warn("Duplicate keyboard shortcut: ", key, handler);
  }
  keyboardShortcuts.set(key, handler);
  return () => {
    keyboardShortcuts.delete(key);
  };
}

/**
 * Svelte action attaching a global keyboard shortcut to an element.
 *
 * This registers the given key (or key sequence of length 2). The listener
 * focuses the node if it is an <input> element and triggers a click on it
 * otherwise.
 */
export function keyboardShortcut(
  node: HTMLElement,
  spec?: KeySpec,
): { destroy: () => void } | undefined {
  if (spec == null) {
    return undefined;
  }
  node.setAttribute("data-key", getKeySpecDescription(spec));
  const unbind = bindKey(spec, node);
  return {
    destroy: () => {
      unbind();
      node.removeAttribute("data-key");
    },
  };
}

/**
 * Register the keys to show/hide the tooltips and register the global keydown handler.
 */
export function initGlobalKeyboardShortcuts(): void {
  document.addEventListener("keydown", keydown);

  bindKey("?", () => {
    const hide = showTooltips();
    const once = () => {
      hide();
      document.removeEventListener("mousedown", once);
      document.removeEventListener("keydown", once);
      document.removeEventListener("scroll", once);
    };
    document.addEventListener("mousedown", once);
    document.addEventListener("keydown", once);
    document.addEventListener("scroll", once);
  });
}
