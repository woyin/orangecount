// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

// Package workspace owns the ledger session and asynchronous work for one
// native window. State values are immutable snapshots; observers may publish
// them to a UI thread and must ignore snapshots with an older Revision.
package workspace

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"orangecount/internal/desktop"
	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
)

var (
	ErrClosed             = errors.New("workspace is closed")
	ErrNoSession          = errors.New("workspace has no open ledger")
	ErrBusy               = errors.New("workspace is opening a ledger")
	ErrIntent             = errors.New("invalid workspace intent")
	ErrNoQuery            = errors.New("workspace session does not support queries")
	ErrNoCapture          = errors.New("workspace session does not support report capture")
	ErrNoQueryCapture     = errors.New("workspace has no current successful query to capture")
	ErrNoExportProtection = errors.New("workspace session cannot refresh export protection")
	ErrQueryID            = errors.New("query request id exhausted")
	ErrCaptureID          = errors.New("report capture id exhausted")
	ErrQueryCaptureID     = errors.New("query capture id exhausted")
)

type Phase string

const (
	PhaseEmpty   Phase = "empty"
	PhaseOpening Phase = "opening"
	PhaseReady   Phase = "ready"
	PhaseInvalid Phase = "invalid"
	PhaseStale   Phase = "stale"
	PhaseFailed  Phase = "failed"
	PhaseClosing Phase = "closing"
	PhaseClosed  Phase = "closed"
)

type IntentKind uint8

const (
	IntentOpen IntentKind = iota + 1
	IntentRead
	IntentReload
	IntentExecuteQuery
	IntentCancelQuery
	IntentCaptureReport
	IntentCancelReportCapture
	IntentCaptureQuery
	IntentCancelQueryCapture
)

type Intent struct {
	Kind        IntentKind
	Path        string
	Options     OpenOptions
	Request     desktop.Request
	QueryText   string
	Report      desktop.ReportExportRequest
	QueryExport desktop.QueryExportRequest
}

type OpenOptions struct {
	SemanticsProfile          ledger.SemanticsProfile
	InputIdentity             string
	Watch                     snapshot.WatchOptions
	AdditionalAttachmentRoots []string
}

// State is a per-window publication. Operation is a monotonically increasing
// token used to reject stale asynchronous work; Epoch changes whenever the
// selected ledger changes or the workspace starts closing.
type State struct {
	WindowID                string
	Epoch                   uint64
	Operation               uint64
	Revision                uint64
	Phase                   Phase
	Busy                    bool
	ActivePath              string
	RequestedPath           string
	Error                   string
	Ledger                  desktop.State
	Response                *desktop.ViewResponse
	QueryRequestID          uint64
	QueryBusy               bool
	QueryCancelling         bool
	QueryResponse           *desktop.QueryResponse
	QueryError              string
	QueryCaptureID          string
	QueryCaptureBusy        bool
	QueryCaptureCancelling  bool
	QueryCapture            *desktop.QueryExportCapture
	QueryCaptureError       string
	ReportCaptureID         string
	ReportCaptureBusy       bool
	ReportCaptureCancelling bool
	ReportCapture           *desktop.ReportExportCapture
	ReportCaptureError      string
	AttachmentRoots         desktop.AttachmentRoots
	AttachmentRootsLoaded   bool
	AttachmentRootsBusy     bool
	AttachmentRootsError    string
}

type Session interface {
	Poll() (desktop.State, error)
	Read(desktop.Request) (desktop.ViewResponse, error)
	Reload() error
	Close()
}

type reportCaptureSession interface {
	CaptureReportExport(desktop.ReportExportRequest) (desktop.ReportExportCapture, error)
	CancelReportExport(string) error
}

type queryCaptureSession interface {
	CaptureQueryExport(desktop.QueryExportRequest) (desktop.QueryExportCapture, error)
}

type Engine interface {
	Open(context.Context, string, OpenOptions) (Session, error)
}

type Observer func(State)

type Controller struct {
	mu                    sync.Mutex
	windowID              string
	engine                Engine
	observer              Observer
	state                 State
	session               Session
	openCancel            context.CancelFunc
	sessionCancel         context.CancelFunc
	sessionGeneration     uint64
	querySequence         uint64
	activeQueryID         uint64
	queryCancel           context.CancelFunc
	queryCaptureSequence  uint64
	activeQueryCaptureID  string
	reportCaptureSequence uint64
	activeReportCaptureID string
	pendingRead           *readJob
	readRunning           bool
	closed                bool
	done                  chan struct{}
	work                  sync.WaitGroup
}

type readJob struct {
	epoch, operation, sessionGeneration uint64
	ledgerRevision                      uint64
	snapshotID                          string
	session                             Session
	request                             desktop.Request
}

func NewController(windowID string, engine Engine, observer Observer) *Controller {
	if engine == nil {
		engine = DesktopEngine{}
	}
	return &Controller{
		windowID: windowID,
		engine:   engine,
		observer: observer,
		done:     make(chan struct{}),
		state: State{
			WindowID: windowID,
			Phase:    PhaseEmpty,
		},
	}
}

func (c *Controller) State() State {
	c.mu.Lock()
	defer c.mu.Unlock()
	return cloneState(c.state)
}

// RefreshExportProtection refreshes local write guards for a captured export
// without changing its frozen bytes. The expected epoch binds the lookup to
// the window's current ledger lifecycle; ordinary reloads may return newer
// protected paths for that same session.
func (c *Controller) RefreshExportProtection(expectedEpoch uint64) (desktop.ExportProtection, error) {
	c.mu.Lock()
	if err := c.canOperateLocked(); err != nil {
		c.mu.Unlock()
		return desktop.ExportProtection{}, err
	}
	if c.state.Epoch != expectedEpoch {
		c.mu.Unlock()
		return desktop.ExportProtection{}, desktop.ErrStale
	}
	if c.state.Phase == PhaseOpening || c.state.Busy || c.state.Ledger.Reloading {
		c.mu.Unlock()
		return desktop.ExportProtection{}, ErrBusy
	}
	session := c.session
	if session == nil {
		c.mu.Unlock()
		return desktop.ExportProtection{}, ErrNoSession
	}
	provider, ok := session.(interface {
		ExportProtection() (desktop.ExportProtection, error)
	})
	if !ok {
		c.mu.Unlock()
		return desktop.ExportProtection{}, ErrNoExportProtection
	}
	generation := c.sessionGeneration
	c.mu.Unlock()

	protection, err := provider.ExportProtection()
	if err != nil {
		return desktop.ExportProtection{}, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.canOperateLocked(); err != nil {
		return desktop.ExportProtection{}, err
	}
	if c.state.Epoch != expectedEpoch || c.sessionGeneration != generation || c.session != session {
		return desktop.ExportProtection{}, desktop.ErrStale
	}
	if c.state.Busy || c.state.Ledger.Reloading {
		return desktop.ExportProtection{}, ErrBusy
	}
	if protection.SnapshotID == "" {
		return desktop.ExportProtection{}, desktop.ErrStale
	}
	protection.ProtectedPaths = append([]string(nil), protection.ProtectedPaths...)
	return protection, nil
}

func (c *Controller) Done() <-chan struct{} { return c.done }

func (c *Controller) Dispatch(intent Intent) error {
	switch intent.Kind {
	case IntentOpen:
		return c.open(intent.Path, intent.Options)
	case IntentRead:
		return c.read(intent.Request)
	case IntentReload:
		return c.reload()
	case IntentExecuteQuery:
		return c.executeQuery(intent.QueryText)
	case IntentCancelQuery:
		return c.cancelQuery()
	case IntentCaptureReport:
		return c.captureReport(intent.Report)
	case IntentCancelReportCapture:
		return c.cancelReportCapture()
	case IntentCaptureQuery:
		return c.captureQuery(intent.QueryExport)
	case IntentCancelQueryCapture:
		return c.cancelQueryCapture()
	default:
		return ErrIntent
	}
}

func (c *Controller) open(path string, options OpenOptions) error {
	if path == "" {
		return ErrIntent
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return ErrClosed
	}
	if c.openCancel != nil {
		c.openCancel()
	}
	queryCancellation := c.invalidateQueryLocked()
	c.clearQueryCaptureLocked()
	reportCancellation := c.invalidateReportCaptureLocked()
	previousPoll := c.sessionCancel
	c.sessionCancel = nil
	c.state.Epoch++
	c.state.Operation++
	c.pendingRead = nil
	epoch, operation := c.state.Epoch, c.state.Operation
	ctx, cancel := context.WithCancel(context.Background())
	c.openCancel = cancel
	c.state.Phase = PhaseOpening
	c.state.Busy = true
	c.state.RequestedPath = path
	c.state.Error = ""
	state := c.commitLocked()
	c.work.Add(1)
	c.mu.Unlock()
	c.cancelQueryGeneration(queryCancellation)
	c.cancelReportCaptureGeneration(reportCancellation)
	if previousPoll != nil {
		previousPoll()
	}
	c.emit(state)
	pollEvery := options.Watch.PollInterval
	if pollEvery <= 0 {
		pollEvery = 500 * time.Millisecond
	}

	go func() {
		defer c.work.Done()
		session, err := c.engine.Open(ctx, path, options)
		if err == nil && ctx.Err() != nil {
			err = ctx.Err()
		}
		var ledgerState desktop.State
		var attachmentRoots desktop.AttachmentRoots
		var attachmentRootsLoaded bool
		if err == nil {
			if session == nil {
				err = errors.New("workspace engine returned a nil session")
			} else {
				ledgerState, err = session.Poll()
				if err == nil {
					if rootSession, ok := session.(attachmentRootsSession); ok {
						attachmentRoots, err = rootSession.AttachmentRoots()
						attachmentRootsLoaded = err == nil
					}
				}
			}
		}
		if err != nil && session != nil {
			session.Close()
			session = nil
		}

		c.mu.Lock()
		if c.closed || c.state.Epoch != epoch || c.state.Operation != operation {
			if c.openCancel != nil && c.state.Epoch == epoch {
				c.openCancel()
				c.openCancel = nil
			}
			c.mu.Unlock()
			if session != nil {
				session.Close()
			}
			return
		}
		cancel()
		c.openCancel = nil
		c.state.Busy = false
		c.state.RequestedPath = ""
		if err != nil {
			c.state.Phase = PhaseFailed
			c.state.Error = err.Error()
			var resume Session
			var pollCtx context.Context
			var pollCancel context.CancelFunc
			var generation uint64
			if c.session != nil {
				resume = c.session
				pollCtx, pollCancel = context.WithCancel(context.Background())
				c.sessionCancel = pollCancel
				c.sessionGeneration++
				generation = c.sessionGeneration
				c.work.Add(1)
			}
			state := c.commitLocked()
			c.mu.Unlock()
			c.emit(state)
			if resume != nil {
				go c.poll(pollCtx, resume, generation, pollEvery)
			}
			return
		} else {
			old := c.session
			oldPollCancel := c.sessionCancel
			c.session = session
			pollCtx, pollCancel := context.WithCancel(context.Background())
			c.sessionCancel = pollCancel
			c.sessionGeneration++
			generation := c.sessionGeneration
			c.work.Add(1)
			c.state.ActivePath = path
			c.state.Ledger = ledgerState
			c.state.AttachmentRoots = attachmentRoots
			c.state.AttachmentRootsLoaded = attachmentRootsLoaded
			c.state.AttachmentRootsBusy = false
			c.state.AttachmentRootsError = ""
			c.state.Response = nil
			c.state.Error = ""
			c.state.Phase = phaseFor(ledgerState)
			state := c.commitLocked()
			c.mu.Unlock()
			c.emit(state)
			if oldPollCancel != nil {
				oldPollCancel()
			}
			if old != nil {
				old.Close()
			}
			go c.poll(pollCtx, session, generation, pollEvery)
			return
		}
	}()
	return nil
}

func (c *Controller) read(request desktop.Request) error {
	c.mu.Lock()
	if err := c.canOperateLocked(); err != nil {
		c.mu.Unlock()
		return err
	}
	if c.state.Phase == PhaseOpening {
		c.mu.Unlock()
		return ErrBusy
	}
	if c.state.Ledger.Reloading {
		c.mu.Unlock()
		return ErrBusy
	}
	if c.session == nil {
		c.mu.Unlock()
		return ErrNoSession
	}
	request = normalizeReadRequest(request)
	ledgerState := c.state.Ledger
	if request.SnapshotID == "" {
		request.SnapshotID = ledgerState.SnapshotID
	}
	c.state.Operation++
	job := &readJob{
		epoch: c.state.Epoch, operation: c.state.Operation,
		sessionGeneration: c.sessionGeneration, ledgerRevision: ledgerState.Revision,
		snapshotID: ledgerState.SnapshotID, session: c.session, request: request,
	}
	c.state.Busy = true
	c.state.Error = ""
	state := c.commitLocked()
	start := !c.readRunning
	if start {
		c.readRunning = true
		c.work.Add(1)
	} else {
		c.pendingRead = job
	}
	c.mu.Unlock()
	c.emit(state)
	if start {
		go c.readLoop(job)
	}
	return nil
}

func (c *Controller) readLoop(job *readJob) {
	defer c.work.Done()
	for job != nil {
		response, err := job.session.Read(job.request)
		if err == nil && !matchesReadJob(job, response) {
			err = desktop.ErrStale
		}
		c.mu.Lock()
		var publication *State
		if c.isCurrentReadLocked(job) {
			c.state.Busy = false
			if err != nil {
				c.state.Error = err.Error()
				if errors.Is(err, desktop.ErrStale) {
					c.state.Phase = PhaseStale
				}
			} else {
				c.state.Response = &response
				c.state.Ledger = response.State
				c.state.Phase = phaseFor(response.State)
			}
			state := c.commitLocked()
			publication = &state
		}
		pending := c.pendingRead
		c.pendingRead = nil
		if pending != nil && c.isCurrentReadLocked(pending) {
			c.mu.Unlock()
			if publication != nil {
				c.emit(*publication)
			}
			job = pending
			continue
		}
		c.readRunning = false
		c.mu.Unlock()
		if publication != nil {
			c.emit(*publication)
		}
		return
	}
}

func (c *Controller) isCurrentReadLocked(job *readJob) bool {
	return !c.closed && job.epoch == c.state.Epoch && job.operation == c.state.Operation && job.sessionGeneration == c.sessionGeneration && job.snapshotID == c.state.Ledger.SnapshotID
}

func normalizeReadRequest(request desktop.Request) desktop.Request {
	if request.View == "" {
		request.View = "balance_sheet"
	}
	if request.Locale == "" {
		request.Locale = "zh-CN"
	}
	if request.View == "account" && request.Scope == "" {
		request.Scope = "subtree"
	}
	return request
}

func matchesReadJob(job *readJob, response desktop.ViewResponse) bool {
	return response.Request == job.request && response.State.SnapshotID == job.snapshotID && response.State.Revision >= job.ledgerRevision
}

func (c *Controller) reload() error {
	c.mu.Lock()
	if err := c.canOperateLocked(); err != nil {
		c.mu.Unlock()
		return err
	}
	if c.state.Phase == PhaseOpening {
		c.mu.Unlock()
		return ErrBusy
	}
	if c.session == nil {
		c.mu.Unlock()
		return ErrNoSession
	}
	queryCancellation := c.invalidateQueryLocked()
	reportCancellation := c.invalidateReportCaptureLocked()
	c.state.Operation++
	c.pendingRead = nil
	epoch, operation, session := c.state.Epoch, c.state.Operation, c.session
	c.state.Busy = true
	c.state.Error = ""
	state := c.commitLocked()
	c.work.Add(1)
	c.mu.Unlock()
	c.cancelQueryGeneration(queryCancellation)
	c.cancelReportCaptureGeneration(reportCancellation)
	c.emit(state)

	go func() {
		defer c.work.Done()
		err := session.Reload()
		var ledgerState desktop.State
		if err == nil {
			ledgerState, err = session.Poll()
		}
		c.mu.Lock()
		if c.closed || c.state.Epoch != epoch || c.state.Operation != operation {
			c.mu.Unlock()
			return
		}
		c.state.Busy = false
		if err != nil {
			c.state.Error = err.Error()
			if errors.Is(err, desktop.ErrClosed) {
				c.state.Phase = PhaseFailed
			}
		} else {
			c.state.Ledger = ledgerState
			c.state.Error = ""
			c.state.Phase = phaseFor(ledgerState)
			c.state.Response = nil
		}
		state := c.commitLocked()
		c.mu.Unlock()
		c.emit(state)
	}()
	return nil
}

// Close invalidates outstanding work immediately. Session cleanup and the
// final Closed publication happen asynchronously; Done closes after both the
// active session and every in-flight operation have released their resources.
func (c *Controller) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	if c.openCancel != nil {
		c.openCancel()
		c.openCancel = nil
	}
	pollCancel := c.sessionCancel
	c.sessionCancel = nil
	queryCancellation := c.invalidateQueryLocked()
	c.clearQueryCaptureLocked()
	reportCancellation := c.invalidateReportCaptureLocked()
	c.state.Epoch++
	c.state.Operation++
	c.pendingRead = nil
	c.state.Busy = false
	c.state.Phase = PhaseClosing
	c.state.RequestedPath = ""
	session := c.session
	c.session = nil
	state := c.commitLocked()
	c.mu.Unlock()
	c.cancelQueryGeneration(queryCancellation)
	c.cancelReportCaptureGeneration(reportCancellation)
	if pollCancel != nil {
		pollCancel()
	}
	c.emit(state)

	go func() {
		if session != nil {
			session.Close()
		}
		c.work.Wait()
		c.mu.Lock()
		c.state.Phase = PhaseClosed
		c.state.Response = nil
		c.state.QueryBusy = false
		c.state.QueryCancelling = false
		c.state.QueryResponse = nil
		c.state.QueryError = ""
		c.state.ReportCaptureID = ""
		c.state.ReportCaptureBusy = false
		c.state.ReportCaptureCancelling = false
		c.state.ReportCapture = nil
		c.state.ReportCaptureError = ""
		c.state.AttachmentRootsBusy = false
		state := c.commitLocked()
		c.mu.Unlock()
		c.emit(state)
		close(c.done)
	}()
}

func (c *Controller) poll(ctx context.Context, session Session, generation uint64, interval time.Duration) {
	defer c.work.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		ledgerState, err := session.Poll()
		if ctx.Err() != nil {
			return
		}
		c.mu.Lock()
		if c.closed || c.sessionGeneration != generation || c.session == nil {
			c.mu.Unlock()
			return
		}
		if err != nil {
			c.state.Error = err.Error()
			if errors.Is(err, desktop.ErrClosed) {
				c.state.Phase = PhaseFailed
			}
			state := c.commitLocked()
			c.mu.Unlock()
			c.emit(state)
			continue
		}
		previous := c.state.Ledger
		changed := ledgerState.Revision != previous.Revision || ledgerState.Reloading != previous.Reloading || ledgerState.Valid != previous.Valid || ledgerState.SnapshotID != previous.SnapshotID
		if !changed || c.state.Phase == PhaseOpening {
			c.mu.Unlock()
			continue
		}
		c.state.Ledger = ledgerState
		var queryCancelRequest *queryCancellation
		var reportCancelRequest *reportCaptureCancellation
		if ledgerState.Revision != previous.Revision || ledgerState.SnapshotID != previous.SnapshotID {
			queryCancelRequest = c.invalidateQueryLocked()
			reportCancelRequest = c.invalidateReportCaptureLocked()
			c.state.Operation++
			c.pendingRead = nil
			c.state.Busy = false
			c.state.Response = nil
		}
		c.state.Phase = phaseFor(ledgerState)
		c.state.Error = ""
		state := c.commitLocked()
		c.mu.Unlock()
		c.cancelQueryGeneration(queryCancelRequest)
		c.cancelReportCaptureGeneration(reportCancelRequest)
		c.emit(state)
	}
}

func (c *Controller) canOperateLocked() error {
	if c.closed {
		return ErrClosed
	}
	return nil
}

func (c *Controller) commitLocked() State {
	c.state.Revision++
	return cloneState(c.state)
}

func (c *Controller) emit(state State) {
	if c.observer != nil {
		c.observer(state)
	}
}

func phaseFor(state desktop.State) Phase {
	if !state.Valid && state.HasSnapshot {
		return PhaseStale
	}
	if !state.Valid {
		return PhaseInvalid
	}
	return PhaseReady
}

func cloneState(state State) State {
	state.Ledger.ActiveExtensions = append([]string(nil), state.Ledger.ActiveExtensions...)
	state.AttachmentRoots.AdditionalRoots = append([]string(nil), state.AttachmentRoots.AdditionalRoots...)
	if state.Response != nil {
		response := *state.Response
		response.State.ActiveExtensions = append([]string(nil), response.State.ActiveExtensions...)
		state.Response = &response
	}
	if state.ReportCapture != nil {
		capture := *state.ReportCapture
		capture.Content = append([]byte(nil), capture.Content...)
		capture.ProtectedPaths = append([]string(nil), capture.ProtectedPaths...)
		state.ReportCapture = &capture
	}
	if state.QueryCapture != nil {
		capture := *state.QueryCapture
		capture.Content = append([]byte(nil), capture.Content...)
		capture.ProtectedPaths = append([]string(nil), capture.ProtectedPaths...)
		state.QueryCapture = &capture
	}
	return state
}

type DesktopEngine struct{}

func (DesktopEngine) Open(ctx context.Context, path string, options OpenOptions) (Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{
		SemanticsProfile: options.SemanticsProfile,
		InputIdentity:    options.InputIdentity,
		Watch:            options.Watch,
	})
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		session.Close()
		return nil, fmt.Errorf("open cancelled: %w", err)
	}
	if len(options.AdditionalAttachmentRoots) != 0 {
		roots, err := session.AttachmentRoots()
		if err == nil {
			_, err = session.ConfigureAttachmentRoots(desktop.AttachmentRootsRequest{PolicyRevision: roots.PolicyRevision, AdditionalRoots: options.AdditionalAttachmentRoots})
		}
		if err != nil {
			session.Close()
			return nil, fmt.Errorf("restore attachment roots: %w", err)
		}
	}
	return session, nil
}
