/**
 * Table sorting for the report pages.
 *
 * A Sorter pairs one SortColumn (which knows how to order rows) with an
 * order; clicking a different header switches the column (resetting to
 * ascending), clicking the active header flips the direction — the same
 * contract as the upstream Fava sort module, implemented here without the
 * d3-array dependency.
 */

export type SortOrder = "asc" | "desc";
export type SortDirection = 1 | -1;

export function get_direction(order: SortOrder): SortDirection {
  return order === "asc" ? 1 : -1;
}

const collator = Intl.Collator();
const compare_strings = collator.compare.bind(collator);

export interface SortColumn<T = unknown> {
  readonly name: string;
  sort(data: readonly T[], direction: SortDirection): readonly T[];
}

export class Sorter<T = unknown> {
  readonly column: SortColumn<T>;
  readonly order: SortOrder;

  constructor(column: SortColumn<T>, order: SortOrder) {
    this.column = column;
    this.order = order;
  }

  switchColumn(column: SortColumn<T>): Sorter<T> {
    if (column === this.column) {
      return new Sorter(column, this.order === "asc" ? "desc" : "asc");
    }
    return new Sorter(column, "asc");
  }

  sort(data: readonly T[]): readonly T[] {
    return this.column.sort(data, get_direction(this.order));
  }
}

function sort_internal<T, U>(
  data: readonly T[],
  value: (row: T) => U,
  compare: (a: U, b: U) => number,
  direction: SortDirection,
): T[] {
  return [...data].sort((a, b) => direction * compare(value(a), value(b)));
}

/** A SortColumn that keeps the rows in their delivered order. */
export class UnsortedColumn<T> implements SortColumn<T> {
  readonly name: string;

  constructor(name: string) {
    this.name = name;
  }

  sort(data: readonly T[]): readonly T[] {
    return data;
  }
}

/** A SortColumn comparing a numeric value derived from each row. */
export class NumberColumn<T> implements SortColumn<T> {
  readonly name: string;
  private readonly value: (row: T) => number;

  constructor(name: string, value: (row: T) => number) {
    this.name = name;
    this.value = value;
  }

  sort(data: readonly T[], direction: SortDirection): readonly T[] {
    return sort_internal(data, this.value, (a, b) => a - b, direction);
  }
}

/** A SortColumn for rows carrying an ISO date string. */
export class DateColumn<T extends { date: string }> extends NumberColumn<T> {
  constructor(name: string) {
    super(name, (row) => new Date(row.date).valueOf());
  }
}

/** A SortColumn comparing locale-aware string values derived from each row. */
export class StringColumn<T> implements SortColumn<T> {
  readonly name: string;
  private readonly value: (row: T) => string;

  constructor(name: string, value: (row: T) => string) {
    this.name = name;
    this.value = value;
  }

  sort(data: readonly T[], direction: SortDirection): readonly T[] {
    return sort_internal(data, this.value, compare_strings, direction);
  }
}
