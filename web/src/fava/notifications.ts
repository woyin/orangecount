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
 * Adapted from Fava's frontend/src/notifications.ts (MIT license, pinned
 * reference revision). The upstream module logs through Fava's log store;
 * here errors are logged to the console at the OrangeCount adapter boundary.
 */

import { errorWithCauses } from "./lib/errors";

/** The notification list div, lazily created. */
const notificationList = (() => {
  let value: HTMLDivElement | null = null;
  return () => {
    if (value == null) {
      value = document.createElement("div");
      value.className = "notifications";
      value.style.right = "10px";
      document.body.appendChild(value);
    }
    // always update the distance to top to account for the current header height
    const headerHeight =
      document.querySelector("header")?.getBoundingClientRect().height ?? 50;
    value.style.top = `${(headerHeight + 10).toString()}px`;
    return value;
  };
})();

type NotificationType = "info" | "warning" | "error";

/**
 * Show a notification containing the given `msg` text and having class `cls`.
 * The notification is automatically removed after 5 seconds and on click
 * `callback` is called.
 *
 * @param msg - The message to diplay
 * @param cls - The message type.
 * @param callback - The callback to execute on click..
 */
export function notify(
  msg: string,
  cls: NotificationType = "info",
  callback?: () => void,
): void {
  const notification = document.createElement("li");
  notification.classList.add(cls);
  notification.appendChild(document.createTextNode(msg));
  notificationList().append(notification);
  notification.addEventListener("click", () => {
    notification.remove();
    callback?.();
  });
  setTimeout(() => {
    notification.remove();
  }, 5000);
}

/**
 * Notify the user about an warning and log to console.
 */
export function notify_warn(msg: string): void {
  notify(msg, "warning");

  console.warn(msg);
}

/**
 * Notify the user about an error and log to console.
 */
export function notify_err(
  error: unknown,
  msg: (e: Error) => string = errorWithCauses,
): void {
  if (error instanceof Error) {
    notify(msg(error), "error");
  }
  console.error(error);
}
