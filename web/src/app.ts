// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

import { translations, type Locale } from "./translations";

// The checked-in dist/app.js is intentionally dependency-free. This source
// module documents the typed UI boundary used by the build in development;
// the Go package embeds the generated bundle and never invokes Node at run
// time.
export type View = "overview" | "accounts" | "journal" | "trial-balance" | "balance-sheet" |
  "income-statement" | "holdings" | "commodities" | "prices" | "events" | "documents" | "statistics" |
  "source" | "diagnostics" | "query";

export interface QueryResult {
  columns: string[];
  rows: Record<string, unknown>[];
}

export interface TreeRowMetadata {
  account: string;
  parent: string;
  depth: number;
  hasChild: boolean;
}

export interface ChartPoint {
  date: string;
  value: PresentedDecimal | string;
}

export interface ChartSeries {
  label: string;
  points: ChartPoint[];
}

export interface ReportChart {
  kind: "line" | "bar" | string;
  title: string;
  unit: string;
  currency: string;
  valuation: string;
  period: string;
  series: ChartSeries[];
}

/**
 * Resolve tree presentation metadata from report-declared relationships.
 * Account punctuation is deliberately not interpreted here: only a parent
 * that is present as another row can affect indentation or visibility.
 */
export function deriveTreeMetadata(rows: Record<string, unknown>[]): Map<Record<string, unknown>, TreeRowMetadata> {
  const accounts = new Set(rows.map((row) => typeof row.account === "string" ? row.account : "").filter(Boolean));
  const parents = new Map<string, string>();
  for (const row of rows) {
    const account = typeof row.account === "string" ? row.account : "";
    const parent = typeof row._tree_parent === "string" ? row._tree_parent : "";
    if (account && parent && parent !== account && accounts.has(parent)) parents.set(account, parent);
  }
  const depth = (account: string): number => {
    let value = 0;
    const seen = new Set([account]);
    let parent = parents.get(account) ?? "";
    while (parent && !seen.has(parent)) {
      value += 1;
      seen.add(parent);
      parent = parents.get(parent) ?? "";
    }
    return value;
  };
  const children = new Set(parents.values());
  return new Map(rows.map((row) => {
    const account = typeof row.account === "string" ? row.account : "";
    return [row, { account, parent: parents.get(account) ?? "", depth: account ? depth(account) : 0, hasChild: children.has(account) }];
  }));
}

export interface PresentedDecimal {
  display: string;
  exact: string;
  approximate?: boolean;
}

export type SortKind = "date" | "decimal" | "text";

export interface JournalRange {
  from: string;
  to: string;
}

export interface JournalFilters {
  flag?: string;
  tag?: string;
  link?: string;
  payee?: string;
  narration?: string;
}

export interface SourceListing {
  paths: string[];
}

export function label(locale: Locale, key: string): string {
  return translations[locale][key] ?? translations.en[key] ?? key;
}

export function displayValue(value: unknown): string {
  if (isPresentedDecimal(value)) return value.display;
  if (Array.isArray(value)) return value.join(", ");
  if (value && typeof value === "object") return JSON.stringify(value);
  return value == null ? "" : String(value);
}

export function isPresentedDecimal(value: unknown): value is PresentedDecimal {
  return !!value && typeof value === "object" &&
    typeof (value as PresentedDecimal).display === "string" &&
    typeof (value as PresentedDecimal).exact === "string";
}

export function journalQuery(range: JournalRange): string {
  const params = new URLSearchParams();
  if (range.from) params.set("from", range.from);
  if (range.to) params.set("to", range.to);
  const query = params.toString();
  return `/api/v1/reports/journal${query ? `?${query}` : ""}`;
}

export function compareTableValues(left: string, right: string, kind: SortKind): number {
  if (left === right) return 0;
  if (kind === "date") return left < right ? -1 : 1;
  if (kind === "decimal") {
    const a = rationalParts(left);
    const b = rationalParts(right);
    if (a && b) {
      const difference = a.numerator * b.denominator - b.numerator * a.denominator;
      if (difference !== 0n) return difference < 0n ? -1 : 1;
      return 0;
    }
  }
  return left < right ? -1 : 1;
}

function rationalParts(value: string): { numerator: bigint; denominator: bigint } | undefined {
  const fraction = value.match(/^([+-]?\d+)\/([+-]?\d+)$/);
  if (fraction && fraction[2] !== "0") {
    const numerator = BigInt(fraction[1]);
    const denominator = BigInt(fraction[2]);
    return denominator < 0n ? { numerator: -numerator, denominator: -denominator } : { numerator, denominator };
  }
  const decimal = value.match(/^([+-]?)(\d+)(?:\.(\d+))?$/);
  if (!decimal) return undefined;
  const digits = decimal[3] ?? "";
  const sign = decimal[1] === "-" ? -1n : 1n;
  return { numerator: sign * BigInt(`${decimal[2]}${digits}`), denominator: 10n ** BigInt(digits.length) };
}

export async function request<T>(path: string): Promise<T> {
  const response = await fetch(path, { headers: { Accept: "application/json" } });
  const body = await response.json() as T & { error?: string };
  if (!response.ok) throw new Error(body.error ?? `${response.status} ${response.statusText}`);
  return body;
}
