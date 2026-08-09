# Make modern time-series charts the standard default

Following the grilling consensus for the modern chart layer, the owner (product owner) decided to promote the modern d3-backed presentation from a switchable Approved Fava deviation to the standard-route default for time-series charts, and to remove the Fava parity fallback for them.

Bar and line time-series charts on the income-statement, balance-sheet, and account routes now render only through `ModernChart` (`ChartView` dispatches `bar`/`stacked-bar`/`line` to it with no toggle). The `?chart_layer` URL parameter, the layer toggle, and the parity bar/line path in `ReportChart` are no longer reachable for those routes. Hierarchy charts (treemap/sunburst/icicle on the trial-balance route) remain out of scope for the modern layer and still render through `ReportChart`.

This is the "B-track" decision the grilling had deferred as a larger, direction-level change. It supersedes the "switchable, default off" framing of ADR-0040 and FD-0006 for time-series charts.

## Considered options

- **(a) Keep modern as a switchable deviation (the prior state).** Rejected by the owner: the modern presentation is the desired default, and maintaining two time-series renderers doubles the maintenance surface.
- **(b) Promote modern to the default and remove parity time-series (chosen).** One time-series renderer; the parity `ReportChart` is retained only for hierarchy charts the modern layer does not cover.

## Consequences

- FD-0006 is removed: the modern time-series presentation is no longer a deviation, it is the standard.
- `ReportChart.svelte` is retained (hierarchy rendering) but its bar/line branches are no longer reachable from the standard routes.
- This decision is hard to reverse (reinstating parity time-series would require restoring the dispatch and the toggle), surprising without context (parity and modern coexisted by design before), and was a real trade-off against (a) — hence this ADR.
