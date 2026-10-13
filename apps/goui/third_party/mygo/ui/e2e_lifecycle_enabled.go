//go:build orangecount_e2e

package ui

import "sync"

var e2eHosts struct {
	sync.Mutex
	hosts map[*windowHost]struct{}
}

// E2EHostLifecycleStats reports opt-in window-host timer state. It is
// available only in native integration-test builds.
type E2EHostLifecycleStats struct {
	Hosts            int
	DetachedHosts    int
	ActiveIdleTimers int
}

func e2eHostAttached(h *windowHost) {
	e2eHosts.Lock()
	if e2eHosts.hosts == nil {
		e2eHosts.hosts = make(map[*windowHost]struct{})
	}
	e2eHosts.hosts[h] = struct{}{}
	e2eHosts.Unlock()
}

func e2eHostDetached(h *windowHost) {
	e2eHosts.Lock()
	delete(e2eHosts.hosts, h)
	e2eHosts.Unlock()
}

// E2EHostLifecycle returns counts for hosts attached to MyGo Content windows.
// Call it on the main thread, where windowHost timer state is changed.
func E2EHostLifecycle() E2EHostLifecycleStats {
	e2eHosts.Lock()
	defer e2eHosts.Unlock()
	var stats E2EHostLifecycleStats
	for h := range e2eHosts.hosts {
		stats.Hosts++
		if h.detached {
			stats.DetachedHosts++
		}
		if h.idleArmed && h.idleTimer != nil {
			stats.ActiveIdleTimers++
		}
	}
	return stats
}
