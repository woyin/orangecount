// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"orangecount/internal/desktop"
	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/snapshot"
)

func TestLateOpenIsClosedAndCannotReplaceNewerLedger(t *testing.T) {
	engine := newFakeEngine()
	c := NewController("window-1", engine, nil)
	defer closeController(t, c)

	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "first.bean"}); err != nil {
		t.Fatal(err)
	}
	first := <-engine.calls
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "second.bean"}); err != nil {
		t.Fatal(err)
	}
	second := <-engine.calls

	secondSession := newFakeSession("second.bean")
	second.result <- openResult{session: secondSession}
	waitForState(t, c, func(s State) bool { return s.ActivePath == "second.bean" && s.Phase == PhaseReady })

	firstSession := newFakeSession("first.bean")
	first.result <- openResult{session: firstSession}
	waitFor(t, "late session close", func() bool { return firstSession.isClosed() })
	if state := c.State(); state.ActivePath != "second.bean" {
		t.Fatalf("late first open replaced the newer ledger: %+v", state)
	}
}

func TestFailedReplacementOpenKeepsThePreviousWorkspaceAndResponse(t *testing.T) {
	engine := newFakeEngine()
	c := NewController("window-open-failure", engine, nil)
	defer closeController(t, c)
	options := OpenOptions{Watch: snapshot.WatchOptions{PollInterval: time.Hour}}
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "stable.bean", Options: options}); err != nil {
		t.Fatal(err)
	}
	initialOpen := <-engine.calls
	session := newFakeSession("stable.bean")
	initialOpen.result <- openResult{session: session}
	waitForState(t, c, func(s State) bool { return s.Phase == PhaseReady && !s.Busy })
	if err := c.Dispatch(Intent{Kind: IntentRead, Request: desktop.Request{View: "dashboard", Text: "old"}}); err != nil {
		t.Fatal(err)
	}
	<-session.readStarted
	session.reads["old"] <- readResult{response: desktop.ViewResponse{State: validDesktopState("stable.bean")}}
	previous := waitForState(t, c, func(s State) bool { return s.Response != nil && !s.Busy })

	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "unavailable.bean", Options: options}); err != nil {
		t.Fatal(err)
	}
	replacement := <-engine.calls
	replacement.result <- openResult{err: errors.New("ledger could not be opened")}
	failed := waitForState(t, c, func(s State) bool { return s.Phase == PhaseFailed && !s.Busy })
	if failed.ActivePath != "stable.bean" || failed.Response == nil || failed.Response.Request.Text != previous.Response.Request.Text || failed.Ledger.SnapshotID != previous.Ledger.SnapshotID {
		t.Fatalf("failed replacement open discarded the last usable workspace: %+v", failed)
	}
}

func TestRapidReadsUseOneRunningAndOnlyTheLatestWaitingRequest(t *testing.T) {
	engine := newFakeEngine()
	c := NewController("window-2", engine, nil)
	defer closeController(t, c)

	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "cashflow.bean"}); err != nil {
		t.Fatal(err)
	}
	opened := <-engine.calls
	session := newFakeSession("cashflow.bean")
	opened.result <- openResult{session: session}
	waitForState(t, c, func(s State) bool { return s.Phase == PhaseReady })

	if err := c.Dispatch(Intent{Kind: IntentRead, Request: desktop.Request{View: "dashboard", Text: "old"}}); err != nil {
		t.Fatal(err)
	}
	if got := receiveString(t, session.readStarted, "first read start"); got != "old" {
		t.Fatalf("first running read = %q, want old", got)
	}
	if err := c.Dispatch(Intent{Kind: IntentRead, Request: desktop.Request{View: "dashboard", Text: "new"}}); err != nil {
		t.Fatal(err)
	}
	if err := c.Dispatch(Intent{Kind: IntentRead, Request: desktop.Request{View: "dashboard", Text: "middle"}}); err != nil {
		t.Fatal(err)
	}
	if err := c.Dispatch(Intent{Kind: IntentRead, Request: desktop.Request{View: "dashboard", Text: "new"}}); err != nil {
		t.Fatal(err)
	}
	oldResponse := desktop.ViewResponse{Request: desktop.Request{Text: "old"}, State: validDesktopState("cashflow.bean")}
	session.reads["old"] <- readResult{response: oldResponse}
	if got := receiveString(t, session.readFinished, "first read completion"); got != "old" {
		t.Fatalf("first completed read = %q, want old", got)
	}
	if got := receiveString(t, session.readStarted, "latest queued read start"); got != "new" {
		t.Fatalf("queued read = %q, want latest request new", got)
	}
	newResponse := desktop.ViewResponse{Request: desktop.Request{Text: "new"}, State: validDesktopState("cashflow.bean")}
	session.reads["new"] <- readResult{response: newResponse}
	if got := receiveString(t, session.readFinished, "latest read completion"); got != "new" {
		t.Fatalf("latest completed read = %q, want new", got)
	}
	waitForState(t, c, func(s State) bool { return s.Response != nil && s.Response.Request.Text == "new" && !s.Busy })
	if state := c.State(); state.Response == nil || state.Response.Request.Text != "new" {
		t.Fatalf("superseded read replaced the newest response: %+v", state)
	}
	select {
	case started := <-session.readStarted:
		t.Fatalf("superseded intermediate read %q was executed", started)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestReadNormalizesAndPinsTheRequestToItsLedgerSnapshot(t *testing.T) {
	engine := newFakeEngine()
	c := NewController("window-read-normalize", engine, nil)
	defer closeController(t, c)
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "read-normalize.bean"}); err != nil {
		t.Fatal(err)
	}
	opened := <-engine.calls
	session := newFakeSession("read-normalize.bean")
	opened.result <- openResult{session: session}
	waitForState(t, c, func(s State) bool { return s.Phase == PhaseReady })
	if err := c.Dispatch(Intent{Kind: IntentRead, Request: desktop.Request{View: "account", Account: "Assets:Cash", Text: "old"}}); err != nil {
		t.Fatal(err)
	}
	request := receiveRequest(t, session.requests, "normalized request")
	if request.View != "account" || request.Locale != "zh-CN" || request.Scope != "subtree" || request.SnapshotID != "snapshot-read-normalize.bean" {
		t.Fatalf("read did not apply core defaults and bind the current snapshot: %+v", request)
	}
	<-session.readStarted
	session.reads["old"] <- readResult{response: desktop.ViewResponse{State: validDesktopState("read-normalize.bean")}}
	state := waitForState(t, c, func(s State) bool { return s.Response != nil && !s.Busy })
	if state.Response.Request != request {
		t.Fatalf("published normalized request changed: got %+v want %+v", state.Response.Request, request)
	}
}

func TestReadRejectsRequestsWhileLedgerIsReloading(t *testing.T) {
	engine := newFakeEngine()
	c := NewController("window-read-reloading", engine, nil)
	defer closeController(t, c)
	options := OpenOptions{Watch: snapshot.WatchOptions{PollInterval: time.Hour}}
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "read-reloading.bean", Options: options}); err != nil {
		t.Fatal(err)
	}
	opened := <-engine.calls
	session := newFakeSession("read-reloading.bean")
	session.state.Reloading = true
	opened.result <- openResult{session: session}
	waitForState(t, c, func(s State) bool { return s.Phase == PhaseReady && s.Ledger.Reloading })
	if err := c.Dispatch(Intent{Kind: IntentRead, Request: desktop.Request{View: "dashboard"}}); !errors.Is(err, ErrBusy) {
		t.Fatalf("read during ledger reload returned %v, want ErrBusy", err)
	}
	select {
	case started := <-session.readStarted:
		t.Fatalf("read %q reached the stale publication during reload", started)
	default:
	}
}

func TestReadRejectsAResponseFromAnOlderLedgerRevision(t *testing.T) {
	engine := newFakeEngine()
	c := NewController("window-read-revision", engine, nil)
	defer closeController(t, c)
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "read-revision.bean"}); err != nil {
		t.Fatal(err)
	}
	opened := <-engine.calls
	session := newFakeSession("read-revision.bean")
	session.state.Revision = 10
	opened.result <- openResult{session: session}
	waitForState(t, c, func(s State) bool { return s.Phase == PhaseReady })
	if err := c.Dispatch(Intent{Kind: IntentRead, Request: desktop.Request{View: "dashboard", Text: "old"}}); err != nil {
		t.Fatal(err)
	}
	<-session.readStarted
	old := validDesktopState("read-revision.bean")
	old.Revision = 9
	session.reads["old"] <- readResult{response: desktop.ViewResponse{State: old}}
	state := waitForState(t, c, func(s State) bool { return s.Phase == PhaseStale && s.Error != "" && !s.Busy })
	if state.Response != nil || state.Ledger.Revision != 10 || state.Error != desktop.ErrStale.Error() {
		t.Fatalf("older response replaced current ledger state: %+v", state)
	}
}

func TestAttachmentRootsRestoreBeforePublishAndConfigureWithCAS(t *testing.T) {
	fixture := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "cashflow.bean")
	fixture, err := filepath.Abs(fixture)
	if err != nil {
		t.Fatal(err)
	}
	additional := t.TempDir()
	c := NewController("window-attachment-roots", DesktopEngine{}, nil)
	defer closeController(t, c)
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: fixture, Options: OpenOptions{
		Watch:                     snapshot.WatchOptions{PollInterval: time.Hour},
		AdditionalAttachmentRoots: []string{additional},
	}}); err != nil {
		t.Fatal(err)
	}
	state := waitForState(t, c, func(s State) bool { return s.Phase == PhaseReady && s.AttachmentRootsLoaded && !s.Busy })
	canonicalAdditional, err := filepath.EvalSymlinks(additional)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.AttachmentRoots.AdditionalRoots) != 1 || state.AttachmentRoots.AdditionalRoots[0] != canonicalAdditional {
		t.Fatalf("opening a restored workspace omitted its roots: %+v", state.AttachmentRoots)
	}

	next := t.TempDir()
	canonicalNext, err := filepath.EvalSymlinks(next)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := c.ConfigureAttachmentRoots(desktop.AttachmentRootsRequest{
		PolicyRevision: state.AttachmentRoots.PolicyRevision, AdditionalRoots: []string{next},
	})
	if err != nil {
		t.Fatal(err)
	}
	state = waitForState(t, c, func(s State) bool {
		return !s.AttachmentRootsBusy && s.AttachmentRoots.PolicyRevision == updated.PolicyRevision
	})
	if len(state.AttachmentRoots.AdditionalRoots) != 1 || state.AttachmentRoots.AdditionalRoots[0] != canonicalNext {
		t.Fatalf("configured attachment roots were not published: %+v", state.AttachmentRoots)
	}
	if _, err := c.ConfigureAttachmentRoots(desktop.AttachmentRootsRequest{
		PolicyRevision: state.AttachmentRoots.PolicyRevision - 1, AdditionalRoots: []string{additional},
	}); !errors.Is(err, desktop.ErrAttachmentPolicy) {
		t.Fatalf("stale attachment policy update = %v, want %v", err, desktop.ErrAttachmentPolicy)
	}
}

func TestReadFailureIsPublishedWithoutDroppingTheCurrentResponse(t *testing.T) {
	engine := newFakeEngine()
	c := NewController("window-read-error", engine, nil)
	defer closeController(t, c)
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "read-error.bean"}); err != nil {
		t.Fatal(err)
	}
	opened := <-engine.calls
	session := newFakeSession("read-error.bean")
	opened.result <- openResult{session: session}
	waitForState(t, c, func(s State) bool { return s.Phase == PhaseReady })
	if err := c.Dispatch(Intent{Kind: IntentRead, Request: desktop.Request{View: "dashboard", Text: "new"}}); err != nil {
		t.Fatal(err)
	}
	<-session.readStarted
	first := desktop.ViewResponse{Request: desktop.Request{Text: "new"}, State: validDesktopState("read-error.bean")}
	session.reads["new"] <- readResult{response: first}
	waitForState(t, c, func(s State) bool { return s.Response != nil && !s.Busy })
	if err := c.Dispatch(Intent{Kind: IntentRead, Request: desktop.Request{View: "dashboard", Text: "old"}}); err != nil {
		t.Fatal(err)
	}
	<-session.readStarted
	session.reads["old"] <- readResult{err: errors.New("projection failed")}
	state := waitForState(t, c, func(s State) bool { return s.Error == "projection failed" && !s.Busy })
	if state.Response == nil || state.Response.Request.Text != "new" || state.Phase != PhaseReady {
		t.Fatalf("read failure discarded the current projection or changed phase: %+v", state)
	}
}

func TestCloseWaitsForLateOpenAndClosesItsSessionWithoutObserver(t *testing.T) {
	engine := newFakeEngine()
	c := NewController("window-3", engine, func(State) {})
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "late.bean"}); err != nil {
		t.Fatal(err)
	}
	opening := <-engine.calls
	c.Close()
	select {
	case <-c.Done():
		t.Fatal("workspace finished closing before the in-flight open released its result")
	case <-time.After(20 * time.Millisecond):
	}

	lateSession := newFakeSession("late.bean")
	opening.result <- openResult{session: lateSession}
	waitFor(t, "late session cleanup", func() bool { return lateSession.isClosed() })
	select {
	case <-c.Done():
	case <-time.After(time.Second):
		t.Fatal("workspace did not finish closing")
	}
	if state := c.State(); state.Phase != PhaseClosed || state.Response != nil {
		t.Fatalf("closed state retained active UI data: %+v", state)
	}
	if err := c.Dispatch(Intent{Kind: IntentReload}); !errors.Is(err, ErrClosed) {
		t.Fatalf("dispatch after close = %v, want ErrClosed", err)
	}
}

func TestWindowControllersKeepIndependentIdentityAndSession(t *testing.T) {
	engineA, engineB := newFakeEngine(), newFakeEngine()
	a := NewController("window-A", engineA, nil)
	b := NewController("window-B", engineB, nil)
	defer closeController(t, a)
	defer closeController(t, b)

	optionsA := OpenOptions{SemanticsProfile: ledger.Legacy, Watch: snapshot.WatchOptions{PollInterval: time.Hour}}
	optionsB := OpenOptions{SemanticsProfile: ledger.V3, Watch: snapshot.WatchOptions{PollInterval: time.Hour}}
	if err := a.Dispatch(Intent{Kind: IntentOpen, Path: "a.bean", Options: optionsA}); err != nil {
		t.Fatal(err)
	}
	if err := b.Dispatch(Intent{Kind: IntentOpen, Path: "b.bean", Options: optionsB}); err != nil {
		t.Fatal(err)
	}
	callA, callB := <-engineA.calls, <-engineB.calls
	if callA.options.SemanticsProfile != ledger.Legacy || callB.options.SemanticsProfile != ledger.V3 {
		t.Fatalf("open options crossed between windows: A=%+v B=%+v", callA.options, callB.options)
	}
	sessionA, sessionB := newFakeSession("a.bean"), newFakeSession("b.bean")
	sessionA.state.SemanticsProfile = ledger.Legacy
	sessionB.state.SemanticsProfile = ledger.V3
	callA.result <- openResult{session: sessionA}
	callB.result <- openResult{session: sessionB}
	waitForState(t, a, func(s State) bool { return s.Phase == PhaseReady })
	waitForState(t, b, func(s State) bool { return s.Phase == PhaseReady })
	if a.State().WindowID != "window-A" || a.State().ActivePath != "a.bean" {
		t.Fatalf("window A state crossed: %+v", a.State())
	}
	if b.State().WindowID != "window-B" || b.State().ActivePath != "b.bean" {
		t.Fatalf("window B state crossed: %+v", b.State())
	}
	if a.State().Ledger.SemanticsProfile != ledger.Legacy || b.State().Ledger.SemanticsProfile != ledger.V3 {
		t.Fatalf("profiles crossed between windows: A=%s B=%s", a.State().Ledger.SemanticsProfile, b.State().Ledger.SemanticsProfile)
	}
	if err := a.Dispatch(Intent{Kind: IntentRead, Request: desktop.Request{View: "dashboard", DateFrom: "2026-01-01", Currency: "USD", Valuation: "at-cost", Text: "old"}}); err != nil {
		t.Fatal(err)
	}
	if err := b.Dispatch(Intent{Kind: IntentRead, Request: desktop.Request{View: "dashboard", DateFrom: "2026-02-01", Currency: "EUR", Valuation: "market-value", Text: "new"}}); err != nil {
		t.Fatal(err)
	}
	requestA := receiveRequest(t, sessionA.requests, "window A reading controls")
	requestB := receiveRequest(t, sessionB.requests, "window B reading controls")
	if requestA.Currency != "USD" || requestA.DateFrom != "2026-01-01" || requestA.Valuation != "at-cost" || requestB.Currency != "EUR" || requestB.DateFrom != "2026-02-01" || requestB.Valuation != "market-value" {
		t.Fatalf("reading controls crossed between windows: A=%+v B=%+v", requestA, requestB)
	}
	<-sessionA.readStarted
	<-sessionB.readStarted
	sessionA.reads["old"] <- readResult{response: desktop.ViewResponse{State: sessionA.state}}
	sessionB.reads["new"] <- readResult{response: desktop.ViewResponse{State: sessionB.state}}
	stateA := waitForState(t, a, func(s State) bool { return s.Response != nil && !s.Busy })
	stateB := waitForState(t, b, func(s State) bool { return s.Response != nil && !s.Busy })
	if stateA.Response.Request.Currency != "USD" || stateB.Response.Request.Currency != "EUR" || stateA.Ledger.SemanticsProfile != ledger.Legacy || stateB.Ledger.SemanticsProfile != ledger.V3 {
		t.Fatalf("window publications crossed: A=%+v B=%+v", stateA, stateB)
	}
	b.Close()
	select {
	case <-b.Done():
	case <-time.After(time.Second):
		t.Fatal("closing second window did not finish")
	}
	if sessionA.isClosed() || a.State().ActivePath != "a.bean" {
		t.Fatal("closing the second window affected the first window session")
	}
}

func TestQueryCancelBypassesBlockedReportRead(t *testing.T) {
	engine := newFakeEngine()
	c := NewController("window-query-cancel", engine, nil)
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "query-cancel.bean"}); err != nil {
		t.Fatal(err)
	}
	opened := <-engine.calls
	session := newFakeQuerySession("query-cancel.bean", true)
	opened.result <- openResult{session: session}
	waitForState(t, c, func(s State) bool { return s.Phase == PhaseReady })

	if err := c.Dispatch(Intent{Kind: IntentRead, Request: desktop.Request{View: "dashboard", Text: "old"}}); err != nil {
		t.Fatal(err)
	}
	receiveString(t, session.readStarted, "blocked report read")
	if err := c.Dispatch(Intent{Kind: IntentExecuteQuery, QueryText: "select 1"}); err != nil {
		t.Fatal(err)
	}
	call := receiveQueryCall(t, session.calls, "query execution")
	if call.request.RequestID != 1 || call.request.SnapshotID != "snapshot-query-cancel.bean" {
		t.Fatalf("query request did not capture its generation and snapshot: %+v", call.request)
	}
	if err := c.Dispatch(Intent{Kind: IntentCancelQuery}); err != nil {
		t.Fatal(err)
	}
	if got := receiveUint64(t, session.cancelled, "direct query cancellation"); got != call.request.RequestID {
		t.Fatalf("cancelled request id = %d, want %d", got, call.request.RequestID)
	}
	state := waitForState(t, c, func(s State) bool { return s.QueryResponse != nil && !s.QueryBusy })
	if state.QueryResponse.Error == nil || state.QueryResponse.Error.Code != "cancelled" {
		t.Fatalf("cancelled query did not publish its typed outcome: %+v", state.QueryResponse)
	}
	select {
	case <-session.readFinished:
		t.Fatal("query cancellation waited for or completed the independent report read")
	default:
	}

	session.reads["old"] <- readResult{response: desktop.ViewResponse{State: validDesktopState("query-cancel.bean")}}
	receiveString(t, session.readFinished, "released report read")
	closeController(t, c)
}

func TestQueryGenerationRejectsLateSupersededResult(t *testing.T) {
	engine := newFakeEngine()
	c := NewController("window-query-generation", engine, nil)
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "query-generation.bean"}); err != nil {
		t.Fatal(err)
	}
	opened := <-engine.calls
	session := newFakeQuerySession("query-generation.bean", false)
	opened.result <- openResult{session: session}
	waitForState(t, c, func(s State) bool { return s.Phase == PhaseReady })

	if err := c.Dispatch(Intent{Kind: IntentExecuteQuery, QueryText: "select 'old'"}); err != nil {
		t.Fatal(err)
	}
	old := receiveQueryCall(t, session.calls, "older query")
	if err := c.Dispatch(Intent{Kind: IntentExecuteQuery, QueryText: "select 'new'"}); err != nil {
		t.Fatal(err)
	}
	newer := receiveQueryCall(t, session.calls, "newer query")
	if old.request.RequestID != 1 || newer.request.RequestID != 2 {
		t.Fatalf("query ids are not strictly increasing: old=%d new=%d", old.request.RequestID, newer.request.RequestID)
	}
	if got := receiveUint64(t, session.cancelled, "superseded query cancellation"); got != old.request.RequestID {
		t.Fatalf("superseded id = %d, want %d", got, old.request.RequestID)
	}
	newer.result <- queryResult{response: successfulQueryResponse(newer.request)}
	waitForState(t, c, func(s State) bool {
		return !s.QueryBusy && s.QueryResponse != nil && s.QueryResponse.Context.RequestID == newer.request.RequestID
	})
	old.result <- queryResult{response: successfulQueryResponse(old.request)}
	receiveUint64(t, session.finished, "late superseded query completion")
	state := c.State()
	if state.QueryResponse == nil || state.QueryResponse.Context.RequestID != newer.request.RequestID {
		t.Fatalf("late old query replaced the latest result: %+v", state)
	}
	closeController(t, c)
}

func TestQueryRequestIDOverflowDoesNotReuseGeneration(t *testing.T) {
	engine := newFakeEngine()
	c := NewController("window-query-overflow", engine, nil)
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "query-overflow.bean"}); err != nil {
		t.Fatal(err)
	}
	opened := <-engine.calls
	session := newFakeQuerySession("query-overflow.bean", true)
	opened.result <- openResult{session: session}
	waitForState(t, c, func(s State) bool { return s.Phase == PhaseReady })
	c.mu.Lock()
	c.querySequence = ^uint64(0)
	c.mu.Unlock()
	if err := c.Dispatch(Intent{Kind: IntentExecuteQuery, QueryText: "select 1"}); !errors.Is(err, ErrQueryID) {
		t.Fatalf("query ID overflow = %v, want ErrQueryID", err)
	}
	if state := c.State(); state.QueryBusy || state.QueryRequestID != 0 {
		t.Fatalf("overflow reused or started a request generation: %+v", state)
	}
	closeController(t, c)
}

func TestReportCaptureCancellationBypassesBlockedReportRead(t *testing.T) {
	engine := newFakeEngine()
	c := NewController("window-capture-cancel", engine, nil)
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "capture-cancel.bean"}); err != nil {
		t.Fatal(err)
	}
	opened := <-engine.calls
	session := newFakeCaptureSession("capture-cancel.bean")
	opened.result <- openResult{session: session}
	waitForState(t, c, func(s State) bool { return s.Phase == PhaseReady })
	setCurrentReportIdentity(c, "capture-cancel.bean", "projection-1")
	if err := c.Dispatch(Intent{Kind: IntentRead, Request: desktop.Request{View: "dashboard", Text: "old"}}); err != nil {
		t.Fatal(err)
	}
	receiveString(t, session.readStarted, "blocked report read during capture")
	if err := c.Dispatch(Intent{Kind: IntentCaptureReport, Report: desktop.ReportExportRequest{
		ProjectionID: "projection-1", SnapshotID: "snapshot-capture-cancel.bean", Format: "csv", CaptureScope: "current-page",
	}}); err != nil {
		t.Fatal(err)
	}
	capture := receiveCaptureCall(t, session.calls, "report capture")
	if capture.request.CaptureID == "" || len(capture.request.CaptureID) > 128 {
		t.Fatalf("controller did not assign a bounded cancellation id: %+v", capture.request)
	}
	if err := c.Dispatch(Intent{Kind: IntentCancelReportCapture}); err != nil {
		t.Fatal(err)
	}
	if got := receiveString(t, session.cancelled, "direct report capture cancellation"); got != capture.request.CaptureID {
		t.Fatalf("cancelled capture id = %q, want %q", got, capture.request.CaptureID)
	}
	state := waitForState(t, c, func(s State) bool { return !s.ReportCaptureBusy && s.ReportCaptureError != "" })
	if state.ReportCaptureError != "Report capture was cancelled" {
		t.Fatalf("cancel result = %q", state.ReportCaptureError)
	}
	select {
	case <-session.readFinished:
		t.Fatal("capture cancellation waited for or completed the independent report read")
	default:
	}
	session.reads["old"] <- readResult{response: desktop.ViewResponse{State: validDesktopState("capture-cancel.bean")}}
	receiveString(t, session.readFinished, "released report read")
	closeController(t, c)
}

func TestReportCaptureRejectsProjectionOutsideCurrentView(t *testing.T) {
	engine := newFakeEngine()
	c := NewController("window-report-expired", engine, nil)
	defer closeController(t, c)
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "report-expired.bean"}); err != nil {
		t.Fatal(err)
	}
	opened := <-engine.calls
	session := newFakeCaptureSession("report-expired.bean")
	opened.result <- openResult{session: session}
	waitForState(t, c, func(state State) bool { return state.Phase == PhaseReady && !state.Busy })
	setCurrentReportIdentity(c, "report-expired.bean", "projection-current")
	err := c.Dispatch(Intent{Kind: IntentCaptureReport, Report: desktop.ReportExportRequest{
		ProjectionID: "projection-old", SnapshotID: "snapshot-report-expired.bean", Format: "csv", CaptureScope: "current-page",
	}})
	if !errors.Is(err, desktop.ErrExpiredCapture) {
		t.Fatalf("capture outside the visible report projection = %v, want ErrExpiredCapture", err)
	}
	select {
	case call := <-session.calls:
		t.Fatalf("expired report unexpectedly reached the session: %+v", call.request)
	default:
	}
}

func TestProfileReplacementCancelsAndClearsReportCapture(t *testing.T) {
	engine := newFakeEngine()
	c := NewController("profile-capture-lifecycle", engine, nil)
	defer closeController(t, c)
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "profile-capture.bean", Options: OpenOptions{SemanticsProfile: ledger.Legacy}}); err != nil {
		t.Fatal(err)
	}
	firstOpen := <-engine.calls
	first := newFakeCaptureSession("profile-capture.bean")
	firstOpen.result <- openResult{session: first}
	waitForState(t, c, func(state State) bool { return state.Phase == PhaseReady && !state.Busy })
	setCurrentReportIdentity(c, "profile-capture.bean", "projection-1")
	if err := c.Dispatch(Intent{Kind: IntentCaptureReport, Report: desktop.ReportExportRequest{
		ProjectionID: "projection-1", SnapshotID: "snapshot-profile-capture.bean", Format: "csv", CaptureScope: "current-page",
	}}); err != nil {
		t.Fatal(err)
	}
	capture := receiveCaptureCall(t, first.calls, "capture before profile replacement")
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "profile-capture.bean", Options: OpenOptions{SemanticsProfile: ledger.V3}}); err != nil {
		t.Fatal(err)
	}
	if got := receiveString(t, first.cancelled, "profile-replacement capture cancellation"); got != capture.request.CaptureID {
		t.Fatalf("profile replacement cancelled capture %q, want %q", got, capture.request.CaptureID)
	}
	opening := c.State()
	if opening.ReportCaptureBusy || opening.ReportCapture != nil || opening.ReportCaptureID != "" || opening.ReportCaptureError != "" {
		t.Fatalf("old capture state survived profile replacement: %+v", opening)
	}
	secondOpen := <-engine.calls
	if secondOpen.options.SemanticsProfile != ledger.V3 {
		t.Fatalf("replacement profile = %q, want V3", secondOpen.options.SemanticsProfile)
	}
	secondOpen.result <- openResult{session: newFakeCaptureSession("profile-capture.bean")}
	ready := waitForState(t, c, func(state State) bool { return state.Phase == PhaseReady && !state.Busy })
	if ready.ReportCaptureBusy || ready.ReportCapture != nil || ready.ReportCaptureID != "" || ready.ReportCaptureError != "" {
		t.Fatalf("old profile capture leaked into the replacement session: %+v", ready)
	}
}

func TestProfileReplacementDropsLateQueryCapture(t *testing.T) {
	engine := newFakeEngine()
	c := NewController("profile-query-capture-lifecycle", engine, nil)
	defer closeController(t, c)
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "profile-query-capture.bean", Options: OpenOptions{SemanticsProfile: ledger.Legacy}}); err != nil {
		t.Fatal(err)
	}
	firstOpen := <-engine.calls
	first := newFakeQueryExportSession("profile-query-capture.bean")
	firstOpen.result <- openResult{session: first}
	waitForState(t, c, func(state State) bool { return state.Phase == PhaseReady && !state.Busy })
	if err := c.Dispatch(Intent{Kind: IntentExecuteQuery, QueryText: "SELECT 1"}); err != nil {
		t.Fatal(err)
	}
	query := receiveQueryCall(t, first.calls, "query before profile replacement")
	response := successfulTypedQueryResponse(query.request)
	query.result <- queryResult{response: response}
	receiveUint64(t, first.finished, "query before profile replacement")
	state := waitForState(t, c, func(state State) bool { return !state.QueryBusy && state.QueryResponse != nil })
	request := desktop.QueryExportRequest{
		SnapshotID:  state.QueryResponse.Context.SnapshotID,
		RequestID:   state.QueryResponse.Context.RequestID,
		Fingerprint: state.QueryResponse.Context.Fingerprint,
		Format:      "csv",
	}
	if err := c.Dispatch(Intent{Kind: IntentCaptureQuery, QueryExport: request}); err != nil {
		t.Fatal(err)
	}
	capture := receiveQueryExportCall(t, first.captures, "query capture before profile replacement")
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "profile-query-capture.bean", Options: OpenOptions{SemanticsProfile: ledger.V3}}); err != nil {
		t.Fatal(err)
	}
	opening := c.State()
	if opening.Phase != PhaseOpening || !opening.Busy || opening.QueryCaptureBusy || opening.QueryCapture != nil || opening.QueryCaptureID != "" || opening.QueryCaptureError != "" {
		t.Fatalf("old query capture state survived profile replacement: %+v", opening)
	}
	secondOpen := <-engine.calls
	if secondOpen.options.SemanticsProfile != ledger.V3 {
		t.Fatalf("replacement profile = %q, want V3", secondOpen.options.SemanticsProfile)
	}
	secondOpen.result <- openResult{session: newFakeQueryExportSession("profile-query-capture.bean")}
	ready := waitForState(t, c, func(state State) bool { return state.Phase == PhaseReady && !state.Busy })
	if ready.QueryCaptureBusy || ready.QueryCapture != nil || ready.QueryCaptureID != "" || ready.QueryCaptureError != "" {
		t.Fatalf("old query capture leaked into the replacement session: %+v", ready)
	}

	capture.result <- queryExportResult{capture: desktop.QueryExportCapture{
		Context: response.Context, Format: request.Format, Content: []byte("late old-profile bytes"),
	}}
	select {
	case <-first.captureFinished:
	case <-time.After(time.Second):
		t.Fatal("old query capture did not finish after profile replacement")
	}
	final := c.State()
	if final.Epoch != ready.Epoch || final.Phase != PhaseReady || final.QueryCaptureBusy || final.QueryCapture != nil || final.QueryCaptureID != "" || final.QueryCaptureError != "" {
		t.Fatalf("late old-profile query capture mutated the replacement session: %+v", final)
	}
}

func setCurrentReportIdentity(c *Controller, path, projectionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	identity := desktop.ReportExportIdentity{
		ProjectionID: projectionID, SnapshotID: "snapshot-" + path,
		CaptureScope: "current-page", Section: "tree", Completeness: "complete",
	}
	c.state.Response = &desktop.ViewResponse{
		State:        c.state.Ledger,
		Request:      desktop.Request{View: "dashboard"},
		ReportExport: &identity,
	}
}

func TestQueryCaptureFreezesSuccessfulBytesAcrossReloadAndLaterQuery(t *testing.T) {
	engine := newFakeEngine()
	c := NewController("window-query-capture-freeze", engine, nil)
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "query-capture-freeze.bean"}); err != nil {
		t.Fatal(err)
	}
	opened := <-engine.calls
	session := newFakeQueryExportSession("query-capture-freeze.bean")
	opened.result <- openResult{session: session}
	waitForState(t, c, func(state State) bool { return state.Phase == PhaseReady })
	if err := c.Dispatch(Intent{Kind: IntentCaptureQuery, QueryExport: desktop.QueryExportRequest{Format: "csv"}}); !errors.Is(err, ErrNoQueryCapture) {
		t.Fatalf("capture without a successful query = %v, want ErrNoQueryCapture", err)
	}
	if err := c.Dispatch(Intent{Kind: IntentExecuteQuery, QueryText: "SELECT 1"}); err != nil {
		t.Fatal(err)
	}
	firstQuery := receiveQueryCall(t, session.calls, "successful query before export capture")
	firstResponse := successfulTypedQueryResponse(firstQuery.request)
	firstQuery.result <- queryResult{response: firstResponse}
	receiveUint64(t, session.finished, "first query completion")
	state := waitForState(t, c, func(state State) bool { return !state.QueryBusy && state.QueryResponse != nil })
	request := desktop.QueryExportRequest{
		SnapshotID: state.QueryResponse.Context.SnapshotID, RequestID: state.QueryResponse.Context.RequestID,
		Fingerprint: state.QueryResponse.Context.Fingerprint, Format: "analysis",
	}
	stale := request
	stale.RequestID++
	if err := c.Dispatch(Intent{Kind: IntentCaptureQuery, QueryExport: stale}); !errors.Is(err, desktop.ErrStale) {
		t.Fatalf("capture with stale query identity = %v, want ErrStale", err)
	}
	unsupported := request
	unsupported.Format = "pdf"
	if err := c.Dispatch(Intent{Kind: IntentCaptureQuery, QueryExport: unsupported}); !errors.Is(err, desktop.ErrRequest) {
		t.Fatalf("capture with unsupported format = %v, want ErrRequest", err)
	}
	if err := c.Dispatch(Intent{Kind: IntentCaptureQuery, QueryExport: request}); err != nil {
		t.Fatal(err)
	}
	captureCall := receiveQueryExportCall(t, session.captures, "fixed query export")
	if captureCall.request != request || c.State().QueryCaptureID == "" || !c.State().QueryCaptureBusy {
		t.Fatalf("query export request was not fixed in workspace state: request=%+v state=%+v", captureCall.request, c.State())
	}
	if err := c.Dispatch(Intent{Kind: IntentCaptureReport, Report: desktop.ReportExportRequest{}}); !errors.Is(err, ErrBusy) {
		t.Fatalf("a report capture overlapped the fixed query capture: %v", err)
	}
	captureCall.result <- queryExportResult{capture: desktop.QueryExportCapture{
		Context: firstResponse.Context, Format: request.Format, Content: []byte("frozen query bytes"), ProtectedPaths: []string{"source.bean"},
	}}
	state = waitForState(t, c, func(state State) bool { return !state.QueryCaptureBusy && state.QueryCapture != nil })
	if string(state.QueryCapture.Content) != "frozen query bytes" || state.QueryCapture.Context.RequestID != request.RequestID {
		t.Fatalf("captured query bytes/context changed: %+v", state.QueryCapture)
	}
	state.QueryCapture.Content[0] = 'X'
	if string(c.State().QueryCapture.Content) != "frozen query bytes" {
		t.Fatal("caller mutation changed the controller's frozen bytes")
	}
	if err := c.Dispatch(Intent{Kind: IntentReload}); err != nil {
		t.Fatal(err)
	}
	state = waitForState(t, c, func(state State) bool { return !state.Busy })
	if state.QueryCapture == nil || string(state.QueryCapture.Content) != "frozen query bytes" {
		t.Fatalf("reload replaced bytes already captured for saving: %+v", state.QueryCapture)
	}
	if err := c.Dispatch(Intent{Kind: IntentExecuteQuery, QueryText: "SELECT 2"}); err != nil {
		t.Fatal(err)
	}
	secondQuery := receiveQueryCall(t, session.calls, "query after fixed export")
	secondQuery.result <- queryResult{response: successfulTypedQueryResponse(secondQuery.request)}
	receiveUint64(t, session.finished, "second query completion")
	state = waitForState(t, c, func(state State) bool {
		return !state.QueryBusy && state.QueryResponse != nil && state.QueryResponse.Context.RequestID == secondQuery.request.RequestID
	})
	if state.QueryCapture == nil || string(state.QueryCapture.Content) != "frozen query bytes" {
		t.Fatalf("later query replaced previously captured bytes: %+v", state.QueryCapture)
	}
	closeController(t, c)
}

func TestQueryCaptureCancellationDropsLateResult(t *testing.T) {
	engine := newFakeEngine()
	c := NewController("window-query-capture-cancel", engine, nil)
	defer closeController(t, c)
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "query-capture-cancel.bean"}); err != nil {
		t.Fatal(err)
	}
	opened := <-engine.calls
	session := newFakeQueryExportSession("query-capture-cancel.bean")
	opened.result <- openResult{session: session}
	waitForState(t, c, func(state State) bool { return state.Phase == PhaseReady })
	if err := c.Dispatch(Intent{Kind: IntentExecuteQuery, QueryText: "SELECT 1"}); err != nil {
		t.Fatal(err)
	}
	queryCall := receiveQueryCall(t, session.calls, "query before cancellable capture")
	response := successfulTypedQueryResponse(queryCall.request)
	queryCall.result <- queryResult{response: response}
	receiveUint64(t, session.finished, "query completion before cancellable capture")
	state := waitForState(t, c, func(state State) bool { return !state.QueryBusy && state.QueryResponse != nil })
	request := desktop.QueryExportRequest{SnapshotID: response.Context.SnapshotID, RequestID: response.Context.RequestID, Fingerprint: response.Context.Fingerprint, Format: "csv"}
	if err := c.Dispatch(Intent{Kind: IntentCaptureQuery, QueryExport: request}); err != nil {
		t.Fatal(err)
	}
	captureCall := receiveQueryExportCall(t, session.captures, "cancellable query capture")
	if err := c.Dispatch(Intent{Kind: IntentCancelQueryCapture}); err != nil {
		t.Fatal(err)
	}
	state = waitForState(t, c, func(state State) bool { return state.QueryCaptureBusy && state.QueryCaptureCancelling })
	if state.QueryCapture != nil || state.QueryCaptureError != "Query capture was cancelled" {
		t.Fatalf("cancel did not mark the pending capture: %+v", state)
	}
	captureCall.result <- queryExportResult{capture: desktop.QueryExportCapture{Context: response.Context, Format: request.Format, Content: []byte("late")}}
	select {
	case <-session.captureFinished:
	case <-time.After(time.Second):
		t.Fatal("late capture call did not finish")
	}
	state = waitForState(t, c, func(state State) bool { return !state.QueryCaptureBusy && !state.QueryCaptureCancelling })
	if state.QueryCapture != nil || state.QueryCaptureError != "Query capture was cancelled" {
		t.Fatalf("late completion replaced cancellation state: %+v", state)
	}
	if err := c.Dispatch(Intent{Kind: IntentCaptureQuery, QueryExport: request}); err != nil {
		t.Fatal(err)
	}
	closeCall := receiveQueryExportCall(t, session.captures, "query capture before window close")
	c.Close()
	closeCall.result <- queryExportResult{capture: desktop.QueryExportCapture{Context: response.Context, Format: request.Format, Content: []byte("closed window")}}
	select {
	case <-session.captureFinished:
	case <-time.After(time.Second):
		t.Fatal("capture did not settle after window close")
	}
	select {
	case <-c.Done():
	case <-time.After(time.Second):
		t.Fatal("workspace did not finish closing after late query capture")
	}
	state = c.State()
	if state.Phase != PhaseClosed || state.QueryCapture != nil || state.QueryCaptureBusy {
		t.Fatalf("late query capture published after window close: %+v", state)
	}
}

func TestOpenAndCloseCancelActiveQuery(t *testing.T) {
	engine := newFakeEngine()
	c := NewController("window-query-lifecycle", engine, nil)
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "first-query.bean"}); err != nil {
		t.Fatal(err)
	}
	firstOpen := <-engine.calls
	first := newFakeQuerySession("first-query.bean", false)
	firstOpen.result <- openResult{session: first}
	waitForState(t, c, func(s State) bool { return s.Phase == PhaseReady })
	if err := c.Dispatch(Intent{Kind: IntentExecuteQuery, QueryText: "select 'open'"}); err != nil {
		t.Fatal(err)
	}
	oldQuery := receiveQueryCall(t, first.calls, "query before ledger switch")
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "second-query.bean"}); err != nil {
		t.Fatal(err)
	}
	if got := receiveUint64(t, first.cancelled, "ledger-switch query cancellation"); got != oldQuery.request.RequestID {
		t.Fatalf("ledger switch cancelled id = %d, want %d", got, oldQuery.request.RequestID)
	}
	secondOpen := <-engine.calls
	second := newFakeQuerySession("second-query.bean", false)
	secondOpen.result <- openResult{session: second}
	waitForState(t, c, func(s State) bool { return s.Phase == PhaseReady && s.ActivePath == "second-query.bean" })
	oldQuery.result <- queryResult{response: successfulQueryResponse(oldQuery.request)}
	receiveUint64(t, first.finished, "late result after ledger switch")
	if state := c.State(); state.QueryResponse != nil || state.QueryBusy {
		t.Fatalf("old ledger query survived workspace switch: %+v", state)
	}

	if err := c.Dispatch(Intent{Kind: IntentExecuteQuery, QueryText: "select 'reload'"}); err != nil {
		t.Fatal(err)
	}
	reloadingQuery := receiveQueryCall(t, second.calls, "query before reload")
	if err := c.Dispatch(Intent{Kind: IntentReload}); err != nil {
		t.Fatal(err)
	}
	if got := receiveUint64(t, second.cancelled, "reload query cancellation"); got != reloadingQuery.request.RequestID {
		t.Fatalf("reload cancelled id = %d, want %d", got, reloadingQuery.request.RequestID)
	}
	waitForState(t, c, func(s State) bool { return !s.Busy })
	reloadingQuery.result <- queryResult{response: successfulQueryResponse(reloadingQuery.request)}
	receiveUint64(t, second.finished, "late query result after reload")
	if state := c.State(); state.QueryResponse != nil || state.QueryBusy {
		t.Fatalf("query result survived reload: %+v", state)
	}

	if err := c.Dispatch(Intent{Kind: IntentExecuteQuery, QueryText: "select 'close'"}); err != nil {
		t.Fatal(err)
	}
	closingQuery := receiveQueryCall(t, second.calls, "query before close")
	c.Close()
	if got := receiveUint64(t, second.cancelled, "close query cancellation"); got != closingQuery.request.RequestID {
		t.Fatalf("close cancelled id = %d, want %d", got, closingQuery.request.RequestID)
	}
	closingQuery.result <- queryResult{response: successfulQueryResponse(closingQuery.request)}
	receiveUint64(t, second.finished, "in-flight query cleanup")
	select {
	case <-c.Done():
	case <-time.After(time.Second):
		t.Fatal("controller did not close after query cancellation")
	}
}

func TestStateCopiesMutableDesktopMetadata(t *testing.T) {
	engine := newFakeEngine()
	c := NewController("window-copy", engine, nil)
	defer closeController(t, c)
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "copy.bean"}); err != nil {
		t.Fatal(err)
	}
	opened := <-engine.calls
	session := newFakeSession("copy.bean")
	session.state.ActiveExtensions = []string{"plugin-a"}
	opened.result <- openResult{session: session}
	waitForState(t, c, func(s State) bool { return s.Phase == PhaseReady })
	state := c.State()
	state.Ledger.ActiveExtensions[0] = "corrupted"
	if got := c.State().Ledger.ActiveExtensions[0]; got != "plugin-a" {
		t.Fatalf("caller mutated controller state: %q", got)
	}
}

func TestDesktopEnginePollsInvalidReloadAndRecovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.bean")
	valid := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Income:Salary USD
2026-01-02 * "Salary"
  Assets:Cash 100 USD
  Income:Salary -100 USD
`
	if err := os.WriteFile(path, []byte(valid), 0600); err != nil {
		t.Fatal(err)
	}
	c := NewController("window-real", DesktopEngine{}, nil)
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: path, Options: OpenOptions{
		SemanticsProfile: ledger.V3,
		Watch:            snapshot.WatchOptions{PollInterval: 10 * time.Millisecond, Debounce: 15 * time.Millisecond},
	}}); err != nil {
		t.Fatal(err)
	}
	waitForState(t, c, func(s State) bool { return s.Phase == PhaseReady && !s.Busy })
	if err := c.Dispatch(Intent{Kind: IntentRead, Request: desktop.Request{View: "dashboard", Valuation: "at-cost"}}); err != nil {
		t.Fatal(err)
	}
	initialRead := waitForState(t, c, func(s State) bool { return s.Response != nil && s.Response.Summary.TransactionsCount == 1 })
	if initialRead.Response.Request.SnapshotID != initialRead.Ledger.SnapshotID || initialRead.Response.Request.Locale != "zh-CN" {
		t.Fatalf("actual desktop read was not normalized and snapshot-bound: %+v", initialRead.Response.Request)
	}

	if err := os.WriteFile(path, []byte("this is not valid Beancount\n"), 0600); err != nil {
		t.Fatal(err)
	}
	waitForState(t, c, func(s State) bool { return s.Phase == PhaseStale && s.Response == nil && !s.Ledger.Reloading })
	if err := c.Dispatch(Intent{Kind: IntentRead, Request: desktop.Request{View: "dashboard", Valuation: "at-cost"}}); err != nil {
		t.Fatal(err)
	}
	failed := waitForState(t, c, func(s State) bool {
		return s.Response != nil && s.Response.State.Valid == false && s.Response.Summary.ErrorCount > 0
	})
	if failed.Phase != PhaseStale {
		t.Fatalf("invalid reload phase = %s, want stale snapshot plus diagnostics", failed.Phase)
	}
	if failed.Response.Summary.TransactionsCount != 1 {
		t.Fatalf("invalid reload discarded the previous valid analysis: %+v", failed.Response.Summary)
	}
	firstDiagnostic := failed.Response.Summary.Diagnostics[0]
	firstRevision := failed.Ledger.Revision

	if err := os.WriteFile(path, []byte(valid+"\n\nthis is a different invalid ledger\n"), 0600); err != nil {
		t.Fatal(err)
	}
	secondInvalid := waitForState(t, c, func(s State) bool {
		return s.Phase == PhaseStale && s.Response == nil && !s.Busy && !s.Ledger.Reloading && s.Ledger.Revision > firstRevision
	})
	if err := c.Dispatch(Intent{Kind: IntentRead, Request: desktop.Request{View: "dashboard", Valuation: "at-cost"}}); err != nil {
		t.Fatal(err)
	}
	secondDiagnosticState := waitForState(t, c, func(s State) bool {
		return s.Response != nil && !s.Response.State.Valid && s.Response.Summary.ErrorCount > 0
	})
	secondDiagnostic := secondDiagnosticState.Response.Summary.Diagnostics[0]
	if secondDiagnostic.Source.Line <= firstDiagnostic.Source.Line || secondDiagnosticState.Ledger.Revision < secondInvalid.Ledger.Revision {
		t.Fatalf("second parse failure did not expose the newest diagnostic revision: first=%+v second=%+v observed_revision=%d invalid_revision=%d state=%+v", firstDiagnostic, secondDiagnostic, secondDiagnosticState.Ledger.Revision, secondInvalid.Ledger.Revision, secondDiagnosticState)
	}

	repaired := valid + `2026-01-03 * "Refund"
  Assets:Cash 25 USD
  Income:Salary -25 USD
`
	if err := os.WriteFile(path, []byte(repaired), 0600); err != nil {
		t.Fatal(err)
	}
	recovered := waitForState(t, c, func(s State) bool {
		return s.Phase == PhaseReady && s.Response == nil && !s.Busy && !s.Ledger.Reloading
	})
	if recovered.Ledger.SnapshotID == secondDiagnosticState.Ledger.SnapshotID {
		t.Fatal("repaired ledger did not publish a new snapshot")
	}
	if err := c.Dispatch(Intent{Kind: IntentRead, Request: desktop.Request{View: "dashboard", Valuation: "at-cost"}}); err != nil {
		t.Fatal(err)
	}
	waitForState(t, c, func(s State) bool { return s.Response != nil && s.Response.Summary.TransactionsCount == 2 })
	closeController(t, c)
}

func TestProfileReplacementAcceptsMatchingPreviewAndRejectsStalePreview(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.bean")
	valid := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Equity:Opening-Balances USD
2026-01-02 * "Opening"
  Assets:Cash 10 USD
  Equity:Opening-Balances -10 USD
`
	if err := os.WriteFile(path, []byte(valid), 0600); err != nil {
		t.Fatal(err)
	}
	c := NewController("profile-identity", DesktopEngine{}, nil)
	defer closeController(t, c)
	watch := snapshot.WatchOptions{PollInterval: time.Hour}
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: path, Options: OpenOptions{SemanticsProfile: ledger.Legacy, Watch: watch}}); err != nil {
		t.Fatal(err)
	}
	legacy := waitForState(t, c, func(state State) bool { return state.Phase == PhaseReady && !state.Busy })
	if err := c.Dispatch(Intent{Kind: IntentRead, Request: desktop.Request{View: "migration"}}); err != nil {
		t.Fatal(err)
	}
	preview := waitForState(t, c, func(state State) bool { return state.Response != nil && state.Response.Migration != nil && !state.Busy })
	identity := preview.Response.Migration.InputIdentity
	if identity == "" || preview.Response.Request.SnapshotID != legacy.Ledger.SnapshotID || preview.Response.State.SnapshotID != legacy.Ledger.SnapshotID {
		t.Fatalf("preview was not bound to the opened snapshot: preview=%q request=%q state=%q", identity, preview.Response.Request.SnapshotID, legacy.Ledger.SnapshotID)
	}
	expectedV3SnapshotID := preview.Response.Migration.V3.SnapshotID

	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: path, Options: OpenOptions{SemanticsProfile: ledger.V3, InputIdentity: identity, Watch: watch}}); err != nil {
		t.Fatal(err)
	}
	v3 := waitForState(t, c, func(state State) bool {
		return state.Phase == PhaseReady && !state.Busy && state.Ledger.SemanticsProfile == ledger.V3
	})
	if v3.Ledger.SnapshotID != expectedV3SnapshotID || v3.Response != nil {
		t.Fatalf("matching preview did not create a clean session for the same input: %+v", v3)
	}

	if err := os.WriteFile(path, []byte(valid+"\n; changed after preview\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: path, Options: OpenOptions{SemanticsProfile: ledger.Legacy, InputIdentity: identity, Watch: watch}}); err != nil {
		t.Fatal(err)
	}
	stale := waitForState(t, c, func(state State) bool { return state.Phase == PhaseFailed && !state.Busy })
	if stale.ActivePath != path || stale.Ledger.SemanticsProfile != ledger.V3 || stale.Ledger.SnapshotID != expectedV3SnapshotID || stale.Error != desktop.ErrStale.Error() {
		t.Fatalf("stale profile apply replaced the last usable session: %+v", stale)
	}
}

func TestDesktopEngineOpensInitiallyInvalidLedgerWithFreshDiagnostics(t *testing.T) {
	path := filepath.Join(t.TempDir(), "initial-invalid.bean")
	if err := os.WriteFile(path, []byte("not valid Beancount\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c := NewController("window-initial-invalid", DesktopEngine{}, nil)
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: path, Options: OpenOptions{
		SemanticsProfile: ledger.V3,
		Watch:            snapshot.WatchOptions{PollInterval: time.Hour},
	}}); err != nil {
		t.Fatal(err)
	}
	opened := waitForState(t, c, func(s State) bool { return (s.Phase == PhaseInvalid || s.Phase == PhaseFailed) && !s.Busy })
	if opened.Phase != PhaseInvalid || opened.ActivePath != path || opened.Response != nil || opened.Ledger.HasSnapshot {
		t.Fatalf("initial invalid ledger inherited stale workspace state: %+v", opened)
	}
	if err := c.Dispatch(Intent{Kind: IntentRead, Request: desktop.Request{View: "dashboard"}}); err != nil {
		t.Fatal(err)
	}
	read := waitForState(t, c, func(s State) bool { return s.Response != nil && !s.Busy })
	if read.Response.State.Valid || read.Response.Summary.ErrorCount == 0 || read.Response.Summary.EntryPath != path {
		t.Fatalf("initial invalid ledger did not expose its own diagnostics: %+v", read.Response)
	}
	closeController(t, c)
}

func TestDesktopEngineQueryPublishesTypedSuccessAndFailure(t *testing.T) {
	c := NewController("window-real-query", DesktopEngine{}, nil)
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "../../../../testdata/fixtures/query/default-source.bean", Options: OpenOptions{
		SemanticsProfile: ledger.V3,
		Watch:            snapshot.WatchOptions{PollInterval: time.Hour},
	}}); err != nil {
		t.Fatal(err)
	}
	openedState := waitForState(t, c, func(s State) bool { return (s.Phase == PhaseReady || s.Phase == PhaseFailed) && !s.Busy })
	if openedState.Phase != PhaseReady {
		t.Fatalf("open query fixture: %+v", openedState)
	}
	if err := c.Dispatch(Intent{Kind: IntentExecuteQuery, QueryText: "SELECT 1 AS v, 2 AS v LIMIT 1"}); err != nil {
		t.Fatal(err)
	}
	success := waitForState(t, c, func(s State) bool { return s.QueryResponse != nil && !s.QueryBusy })
	if success.QueryResponse.Execution == nil || success.QueryResponse.Error != nil || len(success.QueryResponse.Execution.Result.Columns) != 2 || len(success.QueryResponse.Execution.Result.Rows) != 1 {
		t.Fatalf("real desktop query lost its typed result: %+v", success.QueryResponse)
	}
	if success.QueryResponse.Execution.Result.Columns[0].ID == success.QueryResponse.Execution.Result.Columns[1].ID {
		t.Fatal("duplicate query titles collapsed into one result column")
	}
	queryContext := success.QueryResponse.Context
	if err := c.Dispatch(Intent{Kind: IntentCaptureQuery, QueryExport: desktop.QueryExportRequest{
		SnapshotID: queryContext.SnapshotID, RequestID: queryContext.RequestID, Fingerprint: queryContext.Fingerprint, Format: "csv",
	}}); err != nil {
		t.Fatal(err)
	}
	captured := waitForState(t, c, func(s State) bool { return !s.QueryCaptureBusy && s.QueryCapture != nil })
	if captured.QueryCapture.Context != queryContext || captured.QueryCapture.Format != "csv" || string(captured.QueryCapture.Content) != "v,v\n1,2\n" {
		t.Fatalf("workspace did not freeze the real core export identity/bytes: %+v", captured.QueryCapture)
	}
	if err := c.Dispatch(Intent{Kind: IntentExecuteQuery, QueryText: "SELECT abs(missing) LIMIT 0"}); err != nil {
		t.Fatal(err)
	}
	failure := waitForState(t, c, func(s State) bool { return s.QueryResponse != nil && !s.QueryBusy && s.QueryResponse.Error != nil })
	if failure.QueryResponse.Error.Code != "unknown-column" || failure.QueryResponse.Error.Range == nil || failure.QueryError == "" {
		t.Fatalf("typed query failure was not preserved for the UI: %+v", failure)
	}
	if failure.QueryCapture == nil || string(failure.QueryCapture.Content) != "v,v\n1,2\n" {
		t.Fatal("a following failed query replaced bytes already captured for saving")
	}
	closeController(t, c)
}

func TestRefreshExportProtectionIsBoundToWindowEpoch(t *testing.T) {
	engine := newFakeEngine()
	c := NewController("window-export-protection", engine, nil)
	if err := c.Dispatch(Intent{Kind: IntentOpen, Path: "protected.bean"}); err != nil {
		t.Fatal(err)
	}
	call := <-engine.calls
	session := newFakeSession(call.path)
	call.result <- openResult{session: session}
	state := waitForState(t, c, func(state State) bool { return state.Phase == PhaseReady && !state.Busy })
	protection, err := c.RefreshExportProtection(state.Epoch)
	if err != nil || protection.SnapshotID != state.Ledger.SnapshotID || len(protection.ProtectedPaths) != 1 || protection.ProtectedPaths[0] != call.path {
		t.Fatalf("RefreshExportProtection = %+v, %v", protection, err)
	}
	if _, err := c.RefreshExportProtection(state.Epoch + 1); !errors.Is(err, desktop.ErrStale) {
		t.Fatalf("wrong epoch export protection = %v, want stale", err)
	}
	c.Close()
	if _, err := c.RefreshExportProtection(state.Epoch); !errors.Is(err, ErrClosed) {
		t.Fatalf("closed workspace export protection = %v, want closed", err)
	}
	select {
	case <-c.Done():
	case <-time.After(time.Second):
		t.Fatal("controller did not close")
	}
}

func closeController(t *testing.T, c *Controller) {
	t.Helper()
	c.Close()
	select {
	case <-c.Done():
	case <-time.After(time.Second):
		t.Fatal("controller did not close")
	}
}

func waitForState(t *testing.T, c *Controller, predicate func(State) bool) State {
	t.Helper()
	var state State
	waitFor(t, "workspace state", func() bool {
		state = c.State()
		return predicate(state)
	})
	return state
}

func waitFor(t *testing.T, what string, predicate func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if predicate() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

type openCall struct {
	path    string
	options OpenOptions
	result  chan openResult
}

type openResult struct {
	session Session
	err     error
}

type fakeEngine struct{ calls chan openCall }

func newFakeEngine() *fakeEngine { return &fakeEngine{calls: make(chan openCall, 8)} }

func (e *fakeEngine) Open(_ context.Context, path string, options OpenOptions) (Session, error) {
	call := openCall{path: path, options: options, result: make(chan openResult, 1)}
	e.calls <- call
	result := <-call.result // Deliberately ignores cancellation to model a late native open.
	return result.session, result.err
}

type readResult struct {
	response desktop.ViewResponse
	err      error
}

type fakeSession struct {
	state        desktop.State
	reads        map[string]chan readResult
	requests     chan desktop.Request
	readStarted  chan string
	readFinished chan string
	closed       chan struct{}
	closeOnce    sync.Once
}

func newFakeSession(path string) *fakeSession {
	return &fakeSession{
		state: validDesktopState(path),
		reads: map[string]chan readResult{
			"old": make(chan readResult, 1), "middle": make(chan readResult, 1), "new": make(chan readResult, 1),
		},
		requests:     make(chan desktop.Request, 8),
		readStarted:  make(chan string, 8),
		readFinished: make(chan string, 8),
		closed:       make(chan struct{}),
	}
}

func validDesktopState(path string) desktop.State {
	return desktop.State{EntryPath: path, SnapshotID: "snapshot-" + path, Valid: true, HasSnapshot: true, ActiveExtensions: []string{}}
}

func (s *fakeSession) Poll() (desktop.State, error) { return s.state, nil }

func (s *fakeSession) ExportProtection() (desktop.ExportProtection, error) {
	return desktop.ExportProtection{
		Revision: s.state.Revision, SnapshotID: s.state.SnapshotID,
		ProtectedPaths: []string{s.state.EntryPath},
	}, nil
}

func (s *fakeSession) Read(request desktop.Request) (desktop.ViewResponse, error) {
	ch, ok := s.reads[request.Text]
	if !ok {
		return desktop.ViewResponse{}, fmt.Errorf("unexpected read %q", request.Text)
	}
	s.requests <- request
	s.readStarted <- request.Text
	result := <-ch
	result.response.Request = normalizeReadRequest(request)
	if result.response.State.EntryPath == "" {
		result.response.State = s.state
	}
	s.readFinished <- request.Text
	return result.response, result.err
}

func (s *fakeSession) Reload() error { return nil }

func (s *fakeSession) Close() { s.closeOnce.Do(func() { close(s.closed) }) }

func (s *fakeSession) isClosed() bool {
	select {
	case <-s.closed:
		return true
	default:
		return false
	}
}

var _ Engine = (*fakeEngine)(nil)
var _ Session = (*fakeSession)(nil)
var _ Engine = DesktopEngine{}

type queryResult struct {
	response desktop.QueryResponse
	err      error
}

type queryCall struct {
	request desktop.QueryRequest
	result  chan queryResult
}

type fakeQuerySession struct {
	*fakeSession
	calls             chan queryCall
	cancelled         chan uint64
	finished          chan uint64
	honorCancellation bool
}

type captureResult struct {
	capture desktop.ReportExportCapture
	err     error
}

type queryExportResult struct {
	capture desktop.QueryExportCapture
	err     error
}

type captureCall struct {
	request desktop.ReportExportRequest
	result  chan captureResult
}

type queryExportCall struct {
	request desktop.QueryExportRequest
	result  chan queryExportResult
}

type fakeCaptureSession struct {
	*fakeSession
	calls      chan captureCall
	cancelled  chan string
	cancel     chan struct{}
	cancelOnce sync.Once
}

type fakeQueryExportSession struct {
	*fakeQuerySession
	captures        chan queryExportCall
	captureFinished chan struct{}
}

func newFakeCaptureSession(path string) *fakeCaptureSession {
	return &fakeCaptureSession{
		fakeSession: newFakeSession(path),
		calls:       make(chan captureCall, 4),
		cancelled:   make(chan string, 4),
		cancel:      make(chan struct{}),
	}
}

func (s *fakeCaptureSession) CaptureReportExport(request desktop.ReportExportRequest) (desktop.ReportExportCapture, error) {
	call := captureCall{request: request, result: make(chan captureResult, 1)}
	s.calls <- call
	select {
	case result := <-call.result:
		return result.capture, result.err
	case <-s.cancel:
		return desktop.ReportExportCapture{}, context.Canceled
	}
}

func (s *fakeCaptureSession) CancelReportExport(id string) error {
	s.cancelled <- id
	s.cancelOnce.Do(func() { close(s.cancel) })
	return nil
}

func newFakeQuerySession(path string, honorCancellation bool) *fakeQuerySession {
	return &fakeQuerySession{
		fakeSession:       newFakeSession(path),
		calls:             make(chan queryCall, 8),
		cancelled:         make(chan uint64, 8),
		finished:          make(chan uint64, 8),
		honorCancellation: honorCancellation,
	}
}

func newFakeQueryExportSession(path string) *fakeQueryExportSession {
	return &fakeQueryExportSession{
		fakeQuerySession: newFakeQuerySession(path, false),
		captures:         make(chan queryExportCall, 4),
		captureFinished:  make(chan struct{}, 4),
	}
}

func (s *fakeQueryExportSession) CaptureQueryExport(request desktop.QueryExportRequest) (desktop.QueryExportCapture, error) {
	call := queryExportCall{request: request, result: make(chan queryExportResult, 1)}
	s.captures <- call
	result := <-call.result
	s.captureFinished <- struct{}{}
	return result.capture, result.err
}

func (s *fakeQuerySession) ExecuteQuery(ctx context.Context, request desktop.QueryRequest) (desktop.QueryResponse, error) {
	call := queryCall{request: request, result: make(chan queryResult, 1)}
	s.calls <- call
	var result queryResult
	if s.honorCancellation {
		select {
		case result = <-call.result:
		case <-ctx.Done():
			result.response = desktop.QueryResponse{
				Context: requestContext(request),
				Error:   &desktop.QueryFailure{Code: "cancelled", Message: "Query execution was cancelled"},
			}
		}
	} else {
		result = <-call.result
	}
	s.finished <- request.RequestID
	return result.response, result.err
}

func (s *fakeQuerySession) CancelQuery(requestID uint64) error {
	s.cancelled <- requestID
	return nil
}

func successfulQueryResponse(request desktop.QueryRequest) desktop.QueryResponse {
	return desktop.QueryResponse{Context: requestContext(request)}
}

func successfulTypedQueryResponse(request desktop.QueryRequest) desktop.QueryResponse {
	response := successfulQueryResponse(request)
	response.Context.Fingerprint = fmt.Sprintf("query-%d-fingerprint", request.RequestID)
	response.Execution = &query.Execution{
		Result: query.TypedResult{Contract: query.CurrentContract,
			Columns: []query.Column{{ID: "c0", Name: "value", Type: "integer"}},
			Rows:    [][]query.Cell{{{Type: "integer", Value: "1"}}},
		},
		ReturnedRows: 1,
		RowLimit:     1,
	}
	return response
}

func requestContext(request desktop.QueryRequest) desktop.QueryContext {
	return desktop.QueryContext{SnapshotID: request.SnapshotID, RequestID: request.RequestID, Text: request.Text}
}

func receiveQueryCall(t *testing.T, ch <-chan queryCall, what string) queryCall {
	t.Helper()
	select {
	case call := <-ch:
		return call
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", what)
		return queryCall{}
	}
}

func receiveQueryExportCall(t *testing.T, ch <-chan queryExportCall, what string) queryExportCall {
	t.Helper()
	select {
	case call := <-ch:
		return call
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", what)
		return queryExportCall{}
	}
}

func receiveUint64(t *testing.T, ch <-chan uint64, what string) uint64 {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", what)
		return 0
	}
}

func receiveRequest(t *testing.T, ch <-chan desktop.Request, what string) desktop.Request {
	t.Helper()
	select {
	case request := <-ch:
		return request
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", what)
		return desktop.Request{}
	}
}

func receiveCaptureCall(t *testing.T, ch <-chan captureCall, what string) captureCall {
	t.Helper()
	select {
	case call := <-ch:
		return call
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", what)
		return captureCall{}
	}
}

func receiveString(t *testing.T, ch <-chan string, what string) string {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", what)
		return ""
	}
}
