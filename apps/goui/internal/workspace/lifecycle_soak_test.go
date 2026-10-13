// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package workspace

import (
	"path/filepath"
	"testing"
	"time"

	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
)

func TestTwentyDesktopWorkspaceOpenCloseCyclesReleaseWork(t *testing.T) {
	fixture := publicCashflowFixture(t)
	for cycle := 0; cycle < 20; cycle++ {
		controller := NewController("lifecycle-soak", DesktopEngine{}, nil)
		if err := controller.Dispatch(Intent{Kind: IntentOpen, Path: fixture, Options: OpenOptions{
			SemanticsProfile: ledger.V3,
			Watch:            snapshot.WatchOptions{PollInterval: 2 * time.Millisecond},
		}}); err != nil {
			t.Fatalf("cycle %d: open: %v", cycle+1, err)
		}
		opened := waitForLifecycleState(t, controller, func(state State) bool {
			return state.Phase == PhaseReady && !state.Busy
		})
		if opened.ActivePath != fixture || !opened.Ledger.Valid || opened.Ledger.SnapshotID == "" {
			t.Fatalf("cycle %d: unexpected opened workspace: %+v", cycle+1, opened)
		}

		controller.Close()
		closed := waitForLifecycleClosed(t, controller)
		if closed.Phase != PhaseClosed || closed.Busy || closed.Response != nil || closed.QueryResponse != nil || closed.ReportCapture != nil || closed.QueryCapture != nil {
			t.Fatalf("cycle %d: workspace retained published work after Close: %+v", cycle+1, closed)
		}
	}
}

func TestSixtyDesktopReloadsFinishBeforeWorkspaceClose(t *testing.T) {
	fixture := publicCashflowFixture(t)
	controller := NewController("reload-soak", DesktopEngine{}, nil)
	if err := controller.Dispatch(Intent{Kind: IntentOpen, Path: fixture, Options: OpenOptions{
		SemanticsProfile: ledger.V3,
		Watch:            snapshot.WatchOptions{PollInterval: time.Hour},
	}}); err != nil {
		t.Fatalf("open: %v", err)
	}
	opened := waitForLifecycleState(t, controller, func(state State) bool {
		return state.Phase == PhaseReady && !state.Busy
	})

	for reload := 0; reload < 60; reload++ {
		previousOperation := controller.State().Operation
		if err := controller.Dispatch(Intent{Kind: IntentReload}); err != nil {
			t.Fatalf("reload %d: %v", reload+1, err)
		}
		state := waitForLifecycleState(t, controller, func(state State) bool {
			return state.Operation > previousOperation && !state.Busy
		})
		if state.Phase != PhaseReady || state.ActivePath != fixture || state.Ledger.SnapshotID != opened.Ledger.SnapshotID {
			t.Fatalf("reload %d changed the workspace identity or failed: %+v", reload+1, state)
		}
	}

	controller.Close()
	closed := waitForLifecycleClosed(t, controller)
	if closed.Phase != PhaseClosed || closed.Busy || closed.Response != nil {
		t.Fatalf("workspace retained state after 60 reloads and Close: %+v", closed)
	}
}

func publicCashflowFixture(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "cashflow.bean"))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func waitForLifecycleState(t *testing.T, controller *Controller, predicate func(State) bool) State {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		state := controller.State()
		if predicate(state) {
			return state
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for lifecycle state; last state: %+v", controller.State())
	return State{}
}

func waitForLifecycleClosed(t *testing.T, controller *Controller) State {
	t.Helper()
	select {
	case <-controller.Done():
		return controller.State()
	case <-time.After(10 * time.Second):
		t.Fatalf("workspace did not release in-flight work after Close: %+v", controller.State())
		return State{}
	}
}
