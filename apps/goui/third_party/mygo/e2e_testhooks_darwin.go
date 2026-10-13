//go:build darwin && orangecount_e2e

package mygo

import "github.com/egoist/mygo/internal/darwin"

// E2EAccessNode is a native accessibility element observed by an opt-in
// application integration test. This file is excluded from production builds.
type E2EAccessNode struct {
	Role, Subrole, Label, Value, Help string
	ID                                uint64
	Focused                           bool
	Frame                             [4]float64
}

// E2EPasteboardType stores one pasteboard representation without logging or
// converting its bytes, for restoring user state after native integration tests.
type E2EPasteboardType struct {
	Name string
	Data []byte
}

// E2EPasteboardItem preserves the native item grouping and type order.
type E2EPasteboardItem struct {
	Types []E2EPasteboardType
}

// E2EAccessibility reads a Content window through AppKit's native
// accessibility surface, as assistive technology does.
func E2EAccessibility(w *Window) (nodes []E2EAccessNode) {
	RunOnMain(func() {
		for _, node := range darwin.TestAccessibilityWithIDs(w.NativeHandle()) {
			nodes = append(nodes, E2EAccessNode{Role: node.Role, Subrole: node.Subrole, Label: node.Label, Value: node.Value, Help: node.Help, ID: node.ID, Focused: node.Focused, Frame: node.Frame})
		}
	})
	return nodes
}

// E2ESurfaceLifecycle reports active AppKit Content surface frame resources.
func E2ESurfaceLifecycle() (surfaces, displayLinks, runningDisplayLinks, frameTimers int) {
	RunOnMain(func() {
		surfaces, displayLinks, runningDisplayLinks, frameTimers = darwin.TestSurfaceLifecycle()
	})
	return
}

// E2EKey presses and releases a macOS virtual key code in a native window.
func E2EKey(w *Window, keyCode uint16, chars string) (ok bool) {
	RunOnMain(func() { ok = darwin.TestKey(w.NativeHandle(), keyCode, chars) })
	return ok
}

// E2EPerformAccessibility invokes a native accessibility action by label.
func E2EPerformAccessibility(w *Window, label, action, value string) (ok bool) {
	RunOnMain(func() { ok = darwin.TestAccessibilityPerform(w.NativeHandle(), label, action, value) })
	return ok
}

// E2EClickAccessibility clicks an accessible element at its native bounds.
func E2EClickAccessibility(w *Window, label string) (ok bool) {
	RunOnMain(func() { ok = darwin.TestClickAccessibility(w.NativeHandle(), label) })
	return ok
}

// E2EClickAccessibilityID clicks a node by its current native test identity,
// disambiguating accessibility elements whose labels repeat.
func E2EClickAccessibilityID(w *Window, nodeID uint64) (ok bool) {
	RunOnMain(func() { ok = darwin.TestClickAccessibilityID(w.NativeHandle(), nodeID) })
	return ok
}

// E2EActivateMenu activates a menu item through the native application menu.
func E2EActivateMenu(path ...string) (err error) {
	RunOnMain(func() { err = darwin.TestPerformMenuItem(path...) })
	return err
}

// E2EPerformKeyEquivalent sends a Command-key equivalent through the native
// application menu, as the system does when the key is pressed.
func E2EPerformKeyEquivalent(key string, shift bool) (handled bool) {
	RunOnMain(func() { handled = darwin.TestPerformKeyEquivalent(key, shift) })
	return handled
}

// E2ECancelSheet cancels a native dialog attached to a Content window.
func E2ECancelSheet(w *Window) (ok bool) {
	RunOnMain(func() { ok = darwin.TestCancelSheet(w.NativeHandle()) })
	return ok
}

// E2EClickSheetButton activates a named native button in a Content window's
// current sheet. It is limited to opt-in integration tests.
func E2EClickSheetButton(w *Window, title string) (found, clicked bool) {
	RunOnMain(func() { found, clicked = darwin.TestClickSheetButton(w.NativeHandle(), title) })
	return found, clicked
}

// E2EHasSheet reports whether a native dialog is attached to a Content window.
func E2EHasSheet(w *Window) (ok bool) {
	RunOnMain(func() { ok = darwin.TestHasSheet(w.NativeHandle()) })
	return ok
}

func E2EOpenPanelVisible() (visible bool) {
	RunOnMain(func() { visible = darwin.TestOpenPanelVisible() })
	return visible
}

func E2ESavePanelVisible() (visible bool) {
	RunOnMain(func() { visible = darwin.TestSavePanelVisible() })
	return visible
}

// E2EApplicationRunning reports whether a macOS application is running in the
// current user session.
func E2EApplicationRunning(bundleID string) (running bool) {
	RunOnMain(func() { running = darwin.TestApplicationRunning(bundleID) })
	return running
}

// E2ETerminateApplication terminates an application started by a native test.
func E2ETerminateApplication(bundleID string) (terminated bool) {
	RunOnMain(func() { terminated = darwin.TestTerminateApplication(bundleID) })
	return terminated
}

// E2EAccessibilityPress reports whether a labeled element exists, accepts an
// accessibility press, and the result returned by its press action.
func E2EAccessibilityPress(w *Window, label string) (found, allowed, performed bool) {
	RunOnMain(func() {
		found, allowed, performed = darwin.TestAccessibilityPress(w.NativeHandle(), label)
	})
	return
}

// E2EAccessibilityPressID presses the current native accessibility node with
// the supplied MyGo identity and reports whether AppKit performed the action.
func E2EAccessibilityPressID(w *Window, nodeID uint64) (found, allowed, performed bool) {
	RunOnMain(func() {
		found, allowed, performed = darwin.TestAccessibilityPressID(w.NativeHandle(), nodeID)
	})
	return
}

// E2EAccessibilityPressAllowed checks whether a labeled element currently
// accepts a press, without invoking it.
func E2EAccessibilityPressAllowed(w *Window, label string) (found, allowed bool) {
	RunOnMain(func() {
		found, allowed = darwin.TestAccessibilityPressAllowed(w.NativeHandle(), label)
	})
	return
}

// E2EClick sends a click to the native window content in points from its
// top-left corner.
func E2EClick(w *Window, x, y float64) {
	RunOnMain(func() { darwin.TestClick(w.NativeHandle(), x, y) })
}

// E2ECompose sends a marked-text update or commit through the native input
// client, without controlling another process or requiring Accessibility
// permission.
func E2ECompose(w *Window, text string, caret int, commit bool) (ok bool) {
	RunOnMain(func() { ok = darwin.TestCompose(w.NativeHandle(), text, caret, commit) })
	return ok
}

// E2EComposeOver composes text over a UTF-16 selection in the native input
// client.
func E2EComposeOver(w *Window, text string, caret int, commit bool, from, length int) (ok bool) {
	RunOnMain(func() { ok = darwin.TestComposeOver(w.NativeHandle(), text, caret, commit, from, length) })
	return ok
}

// E2EInputClient reports the native input client's selected UTF-16 range and
// current document text.
func E2EInputClient(w *Window) (selected [2]int, document string) {
	RunOnMain(func() { selected, document = darwin.TestInputClient(w.NativeHandle()) })
	return selected, document
}

// E2EPasteboardSnapshot captures all materialized general-pasteboard types.
// ok is false when the pasteboard contains a promised type that cannot be
// faithfully saved for restoration.
func E2EPasteboardSnapshot() (items []E2EPasteboardItem, ok bool) {
	RunOnMain(func() {
		nativeItems, nativeOK := darwin.TestPasteboardSnapshot()
		ok = nativeOK
		if !nativeOK {
			return
		}
		for _, nativeItem := range nativeItems {
			item := E2EPasteboardItem{}
			for _, nativeType := range nativeItem.Types {
				item.Types = append(item.Types, E2EPasteboardType{Name: nativeType.Name, Data: append([]byte(nil), nativeType.Data...)})
			}
			items = append(items, item)
		}
	})
	return items, ok
}

// E2ERestorePasteboard restores the exact item/type payloads captured by
// E2EPasteboardSnapshot.
func E2ERestorePasteboard(items []E2EPasteboardItem) (ok bool) {
	copyItems := make([]darwin.TestPasteboardItem, 0, len(items))
	for _, item := range items {
		nativeItem := darwin.TestPasteboardItem{}
		for _, typ := range item.Types {
			nativeItem.Types = append(nativeItem.Types, darwin.TestPasteboardType{Name: typ.Name, Data: append([]byte(nil), typ.Data...)})
		}
		copyItems = append(copyItems, nativeItem)
	}
	RunOnMain(func() { ok = darwin.TestRestorePasteboard(copyItems) })
	return ok
}

// E2EOpenFiles delivers OS-opened paths through MyGo's application delegate.
// It is only present in opt-in native integration tests.
func E2EOpenFiles(paths ...string) {
	opened := append([]string(nil), paths...)
	RunOnMain(func() { (appHandler{}).OpenFiles(opened) })
}

// E2EDropFiles drops Finder-style file URLs onto a native window.
func E2EDropFiles(w *Window, x, y float64, paths ...string) (over, dropped bool) {
	droppedPaths := append([]string(nil), paths...)
	RunOnMain(func() { over, dropped = darwin.TestDropFiles(w.NativeHandle(), x, y, droppedPaths) })
	return over, dropped
}
