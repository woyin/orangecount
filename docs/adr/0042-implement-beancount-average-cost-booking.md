# Implement Beancount's AVERAGE cost booking

Beancount v3 defines `Booking.AVERAGE` (in `core/data.py`) but its implementation in `parser/booking_method.py` is **disabled** — `booking_method_AVERAGE` returns the error "AVERAGE method is not supported" and the real algorithm lives behind an `if False:` block marked "FIXME: Future implementation here." OrangeCount will implement it, so stock accounts that declare `booking "AVERAGE"` get a self-consistent average cost basis (the displayed average equals the ledger's realized cost and remaining basis, unlike a display-only average over a FIFO ledger).

## Decision

Implement AVERAGE as a per-account opt-in booking method in `internal/ledger/evaluator.go`. The default booking (FIFO) is unchanged; only accounts whose `open` directive sets `booking "AVERAGE"` use it.

Semantics, fixed by grilling consensus (D3):

- **Lazy merge**: augmentations (buys) use strict lot matching — different-price buys coexist as separate lots. Reductions (sells) merge the matching same-(units-currency, cost-currency) lots into one weighted-average lot, then reduce it. This matches Beancount's disabled design sketch.
- **Merged-lot date**: the earliest contributing lot's date (the FIXME in Beancount's sketch left this open; earliest is the conservative cost-basis choice).
- **Explicit-cost reductions rejected**: a reduction that names its own cost (`{price}`) is diagnosed and not booked, because AVERAGE's contract is that the engine owns the cost. (FIFO accounts remain free to use explicit costs.)
- **Cross-cost-currency merge rejected**: lots held in different cost currencies cannot be averaged together; the reduction is diagnosed and not merged. (Split by cost currency — use separate accounts.)
- **Partial fills / negative positions**: reuse OrangeCount's existing `E-EVAL-INVENTORY` diagnostic, consistent with FIFO.
- **Zero-cost lots** (plain currency balances): do not participate in averaging; they flow through normal balance handling.
- **Balance assertions**: assert against the post-booking inventory, as with every booking method.

## Why not the alternatives

- **Display-only average over a FIFO ledger**: rejected because it creates two inconsistent numbers — the displayed average vs the FIFO realized P&L and remaining basis — which diverge as trades accumulate (raised and agreed in grilling D1).
- **Global forced average**: rejected; too invasive and changes every account's semantics.
- **Eager (buy-time) merge**: rejected; it rewrites lots on every augmentation and diverges from Beancount's only design sketch. Lazy is localized to the reduction path and produces the same displayed result via the holdings aggregation.

## Consequences

- OrangeCount is the first working implementation of Beancount's AVERAGE booking. Acceptance is therefore against Beancount's documented intent (the disabled sketch + the Booking enum), not against Beancount output (which errors out). This is recorded honestly as an OC-first extension of a Beancount-envisioned feature, not as v3 coverage completion.
- This is a real ledger-semantics change for opt-in accounts (it changes their balances, costs, and realized P&L vs FIFO). It is gated behind an explicit per-account directive, so ledgers that do not opt in are byte-for-byte unaffected.
- A new report path computes a per-period average-cost time series for the account page by replaying lot history with the same booking predicates, so the series' last point reconciles with the holdings average.
