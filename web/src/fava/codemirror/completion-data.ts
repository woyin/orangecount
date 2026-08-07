/**
 * Completion data for the Beancount editor.
 *
 * Fava feeds its Svelte stores here; OrangeCount has no equivalent global
 * stores, so the editor page sets these values explicitly.
 */

let account_names: readonly string[] = [];
let currencies_list: readonly string[] = [];
let payees_list: readonly string[] = [];
let tags_list: readonly string[] = [];
let links_list: readonly string[] = [];

export interface CompletionData {
  accounts?: readonly string[];
  currencies?: readonly string[];
  payees?: readonly string[];
  tags?: readonly string[];
  links?: readonly string[];
}

export function set_completion_data(data: CompletionData): void {
  account_names = data.accounts ?? [];
  currencies_list = data.currencies ?? [];
  payees_list = data.payees ?? [];
  tags_list = data.tags ?? [];
  links_list = data.links ?? [];
}

export const get_accounts = (): readonly string[] => account_names;
export const get_currencies = (): readonly string[] => currencies_list;
export const get_payees = (): readonly string[] => payees_list;
export const get_tags = (): readonly string[] => tags_list;
export const get_links = (): readonly string[] => links_list;
