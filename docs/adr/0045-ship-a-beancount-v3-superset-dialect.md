# Ship a Beancount v3 superset dialect with bidirectional compilation

OrangeCount will accept a dialect superset of Beancount v3 in ledger source files: every valid v3 file stays valid, and additional dialect lines offer a terse two-posting shorthand. Dialect lines are compiled, never rewritten in place: the dialect file is the canonical source of truth, and a separate `export` command emits a pure v3 snapshot for external tools. This is an experiment on `feature/dialect-superset`; main keeps the strict v3 contract until the experiment graduates.

## Grammar and detection

A dialect line is:

```
[DATE] [!] AMOUNT [CURRENCY] @source -> @destination ["payee"] [: narration] [#tag] [^link]...
```

- Detection is a natural syntactic gap: a date whose second token is a number, or a top-level line beginning with a number, is invalid v3 today. No valid or standard-parseable line can be misread as dialect.
- The arrow carries value flow: a positive amount moves from the left account to the right account; the compiler emits the signs.
- Account endpoints resolve through three levels: full account name, then declared `orangecount.quick-account.v1` aliases (date-effective per ADR-0043), then a unique tail-segment match across all opened accounts. Ambiguity or no match is a per-line error listing candidates; the compiler never guesses.
- An omitted currency uses the ledger's single `operating_currency`, else errors. An omitted narration defaults to "消费". A quoted string after the arrow segment is the payee. The flag may be `*` (default) or `!`.
- A missing date uses block anchoring: the nearest preceding dialect line that carries a date, within the same file. Anchoring never reads "today" and never inherits from non-dialect directives, so every compile of the same source yields the same ledger. No anchor is an error.
- Template invocation stays out of the dialect (template expansion depends on ledger state and would break reproducible compilation).

## Compilation and export

- In-process builds compile dialect lines into ordinary transactions during snapshot construction, so all reports, queries, and validation behave exactly as on a standard ledger (E3).
- `orangecount export source.bean -o out.bean` writes a pure v3 snapshot for Fava, bean-query, and future tools (E1). Export output is a disposable artifact: it must never be hand-edited, and changes go only to the dialect source. A synchronized on-disk export file (E2) is rejected — it would create a second truth and a sync war.
- Dialect line failures are first-class `E-DIALECT-*` diagnostics pointing at the exact line. A ledger whose dialect lines fail to compile does not publish a snapshot (FD-0004 semantics).

## Reverse conversion (dialectize) is a filter, not a translation

`orangecount dialectize v3.bean -o dialect.bean` converts a v3 ledger into the dialect by filtering: a transaction becomes dialect syntax only when its legs are plain amounts in one currency with no cost, no price, no metadata, flag `*` or `!`, and a non-empty narration (an empty narration would be re-materialized as the "消费" default and mutate the record). Everything else is preserved byte-for-byte in standard syntax. The dialect is a shorthand for the common shape, not a second full language; cost, price, and metadata transactions never enter it.

Two-posting transactions collapse to a single dialect line. Transactions with one source and several destinations (a mortgage split into principal and interest), or several sources and one destination (a meal paid partly from WeChat and partly from a coupon), become a **dialect block**: a standard v3 transaction header followed by one indented leg per counterparty:

```
2026-01-20 * "我" "还房贷"
  705.10 CNY @华夏0139 -> @房贷本金
  346.51 CNY @华夏0139 -> @房贷利息
```

A leg is `AMOUNT [CURRENCY] @source -> @destination` with the amount always positive (the compiler assigns signs). Legs are detected by their amount-first shape, which no standard posting line can take. The header owns date, flag, payee, narration, and tags; legs only move money, so a block is rejected if a leg follows standard postings (`E-DIALECT-LEG-ORDER`). Compilation merges same-account postings so a block that fans one source recompiles to the original single source posting (华夏 −2747.91 rather than two negative postings), keeping exports byte-stable.

## Investment buys

A buy is a dialect leg whose head is a securities lot instead of a plain amount, in either of two shapes:

```
QUANTITY SECURITY {COST} @cash -> @securities        explicit quantity (stock)
AMOUNT CURRENCY SECURITY {COST} @cash -> @securities  explicit cash, auto quantity (fund)
```

- The cost batch follows beancount's own posting grammar (`{1.5010 CNY}`), so compiled output reads exactly like hand-written v3.
- Explicit-quantity legs derive the cash side as quantity × unit cost; explicit-cash legs leave the securities quantity empty and let the evaluator infer the share count (1000 ÷ 1.5010), matching how the ledger already records fund buys.
- The leg's cash currency falls back to the cost's currency (a stock bought in CNY carries CNY even with two operating currencies), then the single operating currency.
- Dialectize converts a two-posting buy (cash + securities with cost) into a buy leg. A fee suffix (`手续费 AMOUNT CURRENCY @account`) extends buys and sells with an explicit expense posting: with no explicit amount the cash posting stays elided so the residual absorbs cost plus fee, exactly how the ledger records stock buys.

## Sells, bonus shares, and the narration gate

A sell is an investment leg with a sale price after the cost batch:

```
[CASH CURRENCY] QUANTITY SECURITY {COST} @ PRICE CURRENCY @securities -> @cash [-> @gain] [手续费 FEE CURRENCY @fee]
```

- The price (`@ 21.46 CNY`) marks the sell: the source endpoint posts the securities reduction (empty `{}` matches FIFO lots), the destination receives the cash — explicit when written (the ledger records both gross and net-of-fee conventions; the leg preserves whichever was written), elided to absorb the residual otherwise.
- The optional gain endpoint (`-> @Income:…`) receives the realized P&L as an elided posting, mirroring how the ledger books sells. A gain endpoint requires explicit cash (two elided postings cannot balance).
- A bonus share (红股) is a buy whose source is the income account: `240 STOCKA_300059 {37.62 CNY} @Income:Passive:投资收益 -> @持股`.
- Investment headers quote the narration, so lexer-significant characters (the `(QDII)A` fund names) convert; only the bare single-line form still requires reparse-safe narration.


## Transaction metadata and TODO comments

Dialect blocks carry beancount transaction metadata between the header and the legs — no new syntax, the native v3 form:

```
2026-07-31 * "我" "购买华宝纳斯达克" #fund-investment
  todo: "净值为 7.30 占位，待更新为实际净值"
  50 CNY FUND_017436 {2.1569 CNY} @小金库 -> @基金
```

Metadata is a first-class v3 citizen (queryable via `meta()`, rendered by Fava, never stripped), so the block honors it rather than inventing a parallel concept. Posting-level metadata still keeps a block standard: a dialect leg maps to two postings and cannot carry per-posting data.

Dialectize promotes `; TODO:` and `; FIXME:` comments to a `todo:` metadata pair (multiple comments join with `"; "`), so tracked work items survive conversion as data instead of being deleted. Free-text comments still keep the block standard — prose has no metadata form. A block that already defines `todo` stays standard rather than colliding.

With this the reference ledger converts 288 of 290 investment transactions; the two QDII buys with placeholder-NAV TODO notes now convert, and every investment transaction in the reference ledger now converts.

In the reference ledger 286 of 290 investment transactions convert


## Multi-asset buys

A purchase of several assets in one transaction converts to parallel derived legs — no new syntax:

```
2026-01-16 * "我" "购买2026马年纪念币和纪念钞" #collection-investment
  20 COIN_2026_HORSE {10.00 CNY} @微信 -> @收藏品
  60 NOTE_2026_HORSE {20.00 CNY} @微信 -> @收藏品
```

The rule: one cash leg (explicit or elided) plus two or more securities lots into the same account, every lot a plain single-cost lot in one currency, no price, fee, or gain, and the explicit cash — when present — exactly the sum of quantity × unit cost. Each leg derives its own cash side; the export merges them back into one posting per account, reproducing the original text byte-for-byte. Fees stay standard: a single expense cannot be attributed across legs without inventing an allocation rule. (286 fund/stock buys including fee buys and bonus shares, all 11 sells in both cash conventions); the four remaining legs are one multi-asset collectible purchase that no single leg can express.

Two property tests lock round-trip safety mechanically: for any v3 ledger, `dialectize` then `export` must build a snapshot with identical account balances to the original; and re-running the round trip must reach a fixpoint (the second export is byte-stable). Layout of converted lines is rewritten, which forfeits git blame for those lines; accepted for the experiment.

## Consequences

The canonical ledger is now readable only by OrangeCount; ecosystem tools consume exports. The compatibility contract gains a second surface: OrangeCount must remain a correct v3 reader (unchanged) and a correct dialect compiler (new). The superset keeps every existing write path valid: editor saves, quick entry, and standard syntax inside dialect files all continue to work unchanged.
