# Local MyGo fork

- Upstream: https://github.com/egoist/mygo
- Source module version: `v0.2.16-0.20261006170038-869768d02d91`
- Source commit: `869768d02d9154d5b2f9f3a6ebb39767f8f96983`
- Upstream license: MIT, retained in `LICENSE`
- Local patches:
  - `ui.Element.SetTextSelection(start, end)` accepts half-open Unicode rune offsets, focuses the editor, and reveals the trailing caret. It leaves the editor unchanged while an input method composition is active.
  - `ui.Context.SetTextScale(scale)` applies an app-level text scale to both the frame theme and explicit element font sizes, composing with desktop accessibility text scaling.

This fork is a source snapshot of the pinned upstream revision. Documentation, examples, website, plugin bundles, and their unrelated assets are omitted; imported Go packages and their required embedded resources are retained. The query editor change is covered by tests in `ui/textarea_test.go`; text scaling is covered by `ui/preferences_test.go`. Rebase each patch onto an upstream release that provides equivalent public behavior before removing this local replacement.
