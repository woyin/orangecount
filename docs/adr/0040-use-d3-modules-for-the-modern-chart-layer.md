# Use d3 modules for the modern chart layer

The switchable modern chart layer (`?charts=modern`, an owner-presentation-preference Approved Fava deviation) needs per-pixel responsive redrawing, nice axis ticks, diverging stacked bars, and crosshair series linking that the hand-written parity layer (`ReportChart.svelte`) does not provide. The parity layer stays hand-written SVG with zero dependencies and is not touched.

We will use `d3-scale`, `d3-array`, `d3-shape`, `d3-format`, and `d3-interpolate` in the modern layer only. (`d3-axis` was evaluated but dropped: rendering axes imperatively into bound `<g>` refs was cleared by Svelte's legacy patch pass, leaving empty axes; ticks and grid lines are now rendered declaratively via `{#each}` over `scale.ticks()` instead.) SVG nodes remain OC-authored so the charts inherit Fava's CSS variables and spacing; d3 owns scales, stack offsets, tick computation, and the HCL color interpolation. Axes (tick marks, grid lines, labels) are authored as template elements.

Fava 1.30.12's upstream charts already use these same d3 modules, so this aligns the modern layer with upstream rather than introducing an unrelated dependency. Hand-writing nice ticks, diverging stacks, and time scales a second time is error-prone, and a full chart library (ECharts/Chart.js) would violate ADR-0030/0032 by importing a visual system that fights the Fava shell.

## Considered options

- **(a) Keep hand-writing SVG, zero dependencies.** Cleanest for project discipline, but the modern layer's whole point is axis/scale work that d3 already does and that we have already hand-written once fragilly in `ReportChart.svelte`.
- **(b) d3 submodules (chosen).** Upstream-aligned, tree-shakeable (~30-50 KB), SVG stays ours so visuals stay controllable.
- **(c) Full chart library.** Rejected: violates ADR-0030/0032 and clashes with the Fava CSS variable/typography system.

## Consequences

- New dev-dependency, but isolated to the modern layer; the parity layer remains zero-dependency and is the default for standard routes.
- The dependency lives in `web/package.json` `dependencies` (d3-scale@4, d3-array@3, d3-shape@3, d3-format@3, d3-interpolate@3); these are ESM and tree-shakeable under esbuild (d3 v3/v4 line, not v7).
- Two chart implementations coexist until the modern layer is promoted (a future B-track decision), which is acceptable because the modern layer is switchable off via `?chart_layer=modern` (default off).
- Build (`build:embedded`) must continue to pass; verified: the modern layer + d3 bundle into the embedded `app.js`.
- Axes are rendered declaratively (template `{#each}` over `scale.ticks()`), not via imperative `d3-axis`, to avoid Svelte's legacy patch pass clearing d3-attached DOM nodes.
