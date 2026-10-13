# MyGo per-window text scale API gap

- Dependency: `github.com/egoist/mygo` pseudo-version `v0.2.16-0.20261006170038-869768d02d91`.
- Origin commit: `869768d02d9154d5b2f9f3a6ebb39767f8f96983`.
- Observed gap: MyGo follows desktop accessibility text sizing, but an application could not apply a per-window or per-document scale to its own theme and explicit `FontSize` calls through a public API.
- Application rule: each ledger owns a supported scale (`100%`, `125%`, or `150%`); changes are stored in that ledger's preferences. The application calls one context API at the start of each frame rather than rewriting every widget.
- API: `(*ui.Context).SetTextScale(scale)` scales the frame theme and explicit element font sizes. It composes with the desktop accessibility scale, validates non-finite and out-of-range values, and is applied before the app builds elements.
- Tests: `ui/preferences_test.go` checks default and explicit font-size reflow plus composition with desktop text sizing. The app tests validate per-ledger persistence, restoration, and isolation. A native macOS test checks visible reflow at `900×600`, menu and key-equivalent commands, and persistence after reopening.
- Patch state: this API is a local change in `apps/goui/third_party/mygo`; provenance and the pinned upstream commit are recorded in `third_party/mygo/UPSTREAM.md`.
- Upstream status: no upstream PR or commit is claimed; retain the local patch until an equivalent API ships in a pinned upstream release.
