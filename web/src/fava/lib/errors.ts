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
