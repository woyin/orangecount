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
 * Adapted from Fava's frontend/src/lib/errors.ts (MIT license, pinned
 * reference revision). The upstream module logs through Fava's log store;
 * here errors are logged to the console at the OrangeCount adapter boundary.
 */

/** Render the message of an error, with causes if set. */
export function errorWithCauses(error: Error): string {
  const msg = error.message;
  return error.cause instanceof Error
    ? `${msg}\n  Caused by: ${errorWithCauses(error.cause)}`
    : error.message;
}

class InvalidErrorType extends Error {
  constructor() {
    super("INTERNAL ERROR: error of invalid type.");
  }
}

export function assert_is_error(error: unknown): asserts error is Error {
  if (error instanceof Error) {
    return;
  }
  console.error(error);
  throw new InvalidErrorType();
}
