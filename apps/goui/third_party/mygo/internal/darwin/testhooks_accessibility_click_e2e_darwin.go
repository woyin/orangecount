//go:build darwin && orangecount_e2e

package darwin

// TestClickAccessibility clicks the center of an accessibility element in a
// native window, like a user clicking its visible bounds.
func TestClickAccessibility(handle uintptr, label string) bool {
	for _, n := range TestAccessibility(handle) {
		if n.Label != label || n.Frame[2] <= 0 || n.Frame[3] <= 0 {
			continue
		}
		win := id(handle)
		point := NSPoint{X: n.Frame[0] + n.Frame[2]/2, Y: n.Frame[1] + n.Frame[3]/2}
		inWindow := msgPointToPoint(win, sel("convertPointFromScreen:"), point)
		content := msgRect(send(win, "contentView"), sel("frame"))
		TestClick(handle, inWindow.X, content.Size.Height-inWindow.Y)
		return true
	}
	return false
}

// TestClickAccessibilityID clicks the center of a native accessibility node
// by its current MyGo identity, disambiguating repeated labels.
func TestClickAccessibilityID(handle uintptr, nodeID uint64) bool {
	for _, n := range TestAccessibilityWithIDs(handle) {
		if n.ID != nodeID || n.Frame[2] <= 0 || n.Frame[3] <= 0 {
			continue
		}
		win := id(handle)
		point := NSPoint{X: n.Frame[0] + n.Frame[2]/2, Y: n.Frame[1] + n.Frame[3]/2}
		inWindow := msgPointToPoint(win, sel("convertPointFromScreen:"), point)
		content := msgRect(send(win, "contentView"), sel("frame"))
		TestClick(handle, inWindow.X, content.Size.Height-inWindow.Y)
		return true
	}
	return false
}
