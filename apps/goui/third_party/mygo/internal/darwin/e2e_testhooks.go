//go:build darwin && orangecount_e2e

package darwin

// TestAccessNodeWithID is a native accessibility node with its current MyGo
// runtime identity; it is only available to opt-in integration tests.
type TestAccessNodeWithID struct {
	TestAccessNode
	ID uint64
}

// TestAccessibilityWithIDs reads native accessibility nodes and their MyGo
// runtime identities for transition-safe integration assertions.
func TestAccessibilityWithIDs(handle uintptr) []TestAccessNodeWithID {
	nodes := TestAccessibility(handle)
	withIDs := make([]TestAccessNodeWithID, 0, len(nodes))
	for _, node := range nodes {
		var nodeID uint64
		if element := theBackend.accessElementOf(node.obj); element != nil {
			nodeID = element.node.ID
		}
		withIDs = append(withIDs, TestAccessNodeWithID{TestAccessNode: node, ID: nodeID})
	}
	return withIDs
}

// TestHasSheet reports whether a Content window currently owns a native sheet.
func TestHasSheet(handle uintptr) (hasSheet bool) {
	withPool(func() { hasSheet = send(id(handle), "attachedSheet") != 0 })
	return hasSheet
}

// TestClickSheetButton activates a titled NSButton in a Content window's
// attached sheet. It walks public NSView subviews and checks selectors before
// sending them, so it works for NSAlert without assuming the sheet implements
// NSAlert's buttons selector.
func TestClickSheetButton(handle uintptr, title string) (found, clicked bool) {
	withPool(func() {
		sheet := send(id(handle), "attachedSheet")
		if sheet == 0 {
			return
		}
		content := send(sheet, "contentView")
		if content == 0 {
			return
		}
		titleSelector := sel("title")
		subviewsSelector := sel("subviews")
		performSelector := sel("performClick:")
		var walk func(id)
		walk = func(view id) {
			if view == 0 || clicked {
				return
			}
			if sendBool(view, "respondsToSelector:", uintptr(titleSelector)) && goString(send(view, "title")) == title {
				found = true
				if sendBool(view, "respondsToSelector:", uintptr(performSelector)) {
					send(view, "performClick:", 0)
					clicked = true
					return
				}
			}
			if !sendBool(view, "respondsToSelector:", uintptr(subviewsSelector)) {
				return
			}
			for _, child := range arrayItems(send(view, "subviews")) {
				walk(child)
				if clicked {
					return
				}
			}
		}
		walk(content)
	})
	return found, clicked
}

func TestOpenPanelVisible() (visible bool) {
	withPool(func() {
		panelClass := class("NSOpenPanel")
		for _, window := range arrayItems(send(theBackend.app, "windows")) {
			if sendBool(window, "isKindOfClass:", uintptr(panelClass)) && sendBool(window, "isVisible") {
				visible = true
				return
			}
		}
	})
	return visible
}

func TestSavePanelVisible() (visible bool) {
	withPool(func() {
		panelClass := class("NSSavePanel")
		for _, window := range arrayItems(send(theBackend.app, "windows")) {
			if sendBool(window, "isKindOfClass:", uintptr(panelClass)) && sendBool(window, "isVisible") {
				visible = true
				return
			}
		}
	})
	return visible
}

// TestApplicationRunning reports whether a native application with bundleID
// is present in the current user session.
func TestApplicationRunning(bundleID string) (running bool) {
	withPool(func() {
		workspace := send(class("NSWorkspace"), "sharedWorkspace")
		for _, app := range arrayItems(send(workspace, "runningApplications")) {
			if goString(send(app, "bundleIdentifier")) == bundleID {
				running = true
				return
			}
		}
	})
	return
}

// TestTerminateApplication asks the current user session to terminate the
// named application. It is only used to clean up an app launched by a test.
func TestTerminateApplication(bundleID string) (terminated bool) {
	withPool(func() {
		workspace := send(class("NSWorkspace"), "sharedWorkspace")
		for _, app := range arrayItems(send(workspace, "runningApplications")) {
			if goString(send(app, "bundleIdentifier")) == bundleID {
				terminated = sendBool(app, "terminate")
				return
			}
		}
	})
	return
}

// TestAccessibilityPress reports whether label exists, allows an accessibility
// press, and what accessibilityPerformPress returned.
func TestAccessibilityPress(handle uintptr, label string) (found, allowed, performed bool) {
	for _, n := range TestAccessibility(handle) {
		if n.Label != label {
			continue
		}
		found = true
		withPool(func() {
			selector := sel("accessibilityPerformPress")
			allowed = sendBool(n.obj, "isAccessibilitySelectorAllowed:", uintptr(selector))
			if allowed {
				performed = sendBool(n.obj, "accessibilityPerformPress")
			}
		})
		return
	}
	return
}

// TestAccessibilityPressID performs a press on the current accessibility node
// identified by its MyGo identity.
func TestAccessibilityPressID(handle uintptr, nodeID uint64) (found, allowed, performed bool) {
	for _, n := range TestAccessibilityWithIDs(handle) {
		if n.ID != nodeID {
			continue
		}
		found = true
		withPool(func() {
			selector := sel("accessibilityPerformPress")
			allowed = sendBool(n.obj, "isAccessibilitySelectorAllowed:", uintptr(selector))
			if allowed {
				performed = sendBool(n.obj, "accessibilityPerformPress")
			}
		})
		return
	}
	return
}

// TestAccessibilityPressAllowed reports whether a labeled accessibility
// element currently accepts a press without invoking the action.
func TestAccessibilityPressAllowed(handle uintptr, label string) (found, allowed bool) {
	for _, n := range TestAccessibility(handle) {
		if n.Label != label {
			continue
		}
		found = true
		withPool(func() {
			allowed = sendBool(n.obj, "isAccessibilitySelectorAllowed:", uintptr(sel("accessibilityPerformPress")))
		})
		return
	}
	return
}
