# MyGo editor selection API gap

- Dependency: `github.com/egoist/mygo` pseudo-version `v0.2.16-0.20261006170038-869768d02d91`.
- Origin commit: `869768d02d9154d5b2f9f3a6ebb39767f8f96983`.
- Audit scope: exported `ui.TextArea`, `ui.TextInput`, and `*ui.Element` APIs in `ui/editor.go` and `ui/element.go` at that commit.
- Observed gap: text can be edited and IME composition is exposed with `Composing()`, but callers have no public API to set a selection/caret or scroll an error range into view.
- Application rule: do not reach into `ui` private editor state. Keep diagnostics visible and retain the byte-to-rune converter until a supported API exists.
- Required patch before query-error navigation: expose selection setting and reveal/scroll behavior on the text element; preserve undo history and marked text; add UTF-8 and IME regression tests upstream or in a maintained minimal fork.
- Patch state: a workspace-local source snapshot is maintained at `apps/goui/third_party/mygo`, selected by the nested `go.mod` replace. Provenance, MIT notice and omitted unrelated upstream trees are recorded in `third_party/mygo/UPSTREAM.md`.
- API: `(*ui.Element).SetTextSelection(start, end)` accepts a half-open rune range, focuses the editor and reveals the trailing caret. It does not mutate text or undo history and returns false without changing a marked IME composition.
- Tests: `ui/textarea_test.go` verifies Unicode selection, undo preservation, marked-text preservation and scrolling the selected caret into view. The app's UTF-8 diagnostic byte-range converter remains the boundary adapter.
- Upstream status: no upstream PR or commit is claimed; the patch remains local until equivalent behavior ships in a pinned upstream release.
