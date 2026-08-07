import {
  defineLanguageFacet,
  Language,
  languageDataProp,
  LanguageSupport,
  syntaxHighlighting,
} from "@codemirror/language";
import { highlightTrailingWhitespace } from "@codemirror/view";
import { styleTags, tags } from "@lezer/highlight";
import { Language as TSLanguage, Parser as TSParser } from "web-tree-sitter";
import ts_wasm from "web-tree-sitter/web-tree-sitter.wasm";

import { beancount_completion } from "./beancount-autocomplete";
import { beancount_fold } from "./beancount-fold";
import { beancount_highlight } from "./beancount-highlight";
import { beancount_indent } from "./beancount-indent";
// WASM build of tree-sitter grammar from https://github.com/yagebu/tree-sitter-beancount
import ts_beancount_wasm from "./tree-sitter-beancount.wasm";
import { LezerTSParser } from "./tree-sitter-parser";

/** Import the tree-sitter and Beancount language WASM files and initialise the parser. */
function dataUrlBytes(url: string): Uint8Array {
  // The wasm imports below are inlined by esbuild as base64 data URLs. web-tree-sitter's
  // TSParser.init and Language.load accept raw bytes (bypassing their fetch path, which
  // cannot load a data: URL), so decode the URL payload and hand it over directly.
  const base64 = url.slice(url.indexOf(",") + 1);
  const bin = atob(base64);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) {
    bytes[i] = bin.charCodeAt(i);
  }
  return bytes;
}

const parse_js_wasm = dataUrlBytes(ts_wasm);
const parse_beancount_wasm = dataUrlBytes(ts_beancount_wasm);

async function load_beancount_parser(): Promise<TSParser> {
  await TSParser.init({ wasmBinary: parse_js_wasm });
  const lang = await TSLanguage.load(parse_beancount_wasm);
  const parser = new TSParser();
  parser.setLanguage(lang);
  return parser;
}

const beancount_language_facet = defineLanguageFacet();
const beancount_language_support_extensions = [
  beancount_fold,
  syntaxHighlighting(beancount_highlight),
  beancount_indent,
  beancount_language_facet.of({
    autocomplete: beancount_completion,
    commentTokens: { line: ";" },
    indentOnInput: /^\s+\d\d\d\d/,
  }),
  highlightTrailingWhitespace(),
];

/** The node props that allow for highlighting/coloring of the code. */
const props = [
  styleTags({
    account: tags.className,
    currency: tags.unit,
    date: tags.special(tags.number),
    string: tags.string,
    "BALANCE CLOSE COMMODITY CUSTOM DOCUMENT EVENT NOTE OPEN PAD PRICE TRANSACTION QUERY":
      tags.keyword,
    "tag link": tags.labelName,
    number: tags.number,
    key: tags.propertyName,
    bool: tags.bool,
    "PUSHTAG POPTAG PUSHMETA POPMETA OPTION PLUGIN INCLUDE": tags.standard(
      tags.string,
    ),
  }),
  languageDataProp.add((type) =>
    type.isTop ? beancount_language_facet : undefined,
  ),
];

const ts_parser = await load_beancount_parser();

export const beancount_language_support = new LanguageSupport(
  new Language(
    beancount_language_facet,
    new LezerTSParser(ts_parser, props, "beancount_file"),
    [],
    "beancount",
  ),
  beancount_language_support_extensions,
);
