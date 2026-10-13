// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package workspace

import (
	"fmt"

	"orangecount/internal/desktop"
)

// captureQuery freezes the current successful query before a save panel is
// opened. The core validates the request identity again at capture time.
func (c *Controller) captureQuery(request desktop.QueryExportRequest) error {
	c.mu.Lock()
	if err := c.canOperateLocked(); err != nil {
		c.mu.Unlock()
		return err
	}
	if c.state.Phase == PhaseOpening || c.state.Busy || c.state.ReportCaptureBusy || c.state.QueryCaptureBusy {
		c.mu.Unlock()
		return ErrBusy
	}
	if c.session == nil {
		c.mu.Unlock()
		return ErrNoSession
	}
	session, ok := c.session.(queryCaptureSession)
	if !ok {
		c.mu.Unlock()
		return ErrNoQueryCapture
	}
	response := c.state.QueryResponse
	if c.state.QueryBusy || response == nil || response.Error != nil || response.Execution == nil || !c.state.Ledger.HasSnapshot {
		c.mu.Unlock()
		return ErrNoQueryCapture
	}
	if request.SnapshotID != response.Context.SnapshotID || request.SnapshotID != c.state.Ledger.SnapshotID || request.RequestID != response.Context.RequestID || request.Fingerprint != response.Context.Fingerprint {
		c.mu.Unlock()
		return desktop.ErrStale
	}
	if request.Format != "csv" && request.Format != "analysis" {
		c.mu.Unlock()
		return desktop.ErrRequest
	}
	if c.queryCaptureSequence == ^uint64(0) {
		c.mu.Unlock()
		return ErrQueryCaptureID
	}
	c.queryCaptureSequence++
	captureID := fmt.Sprintf("goui-query-%d", c.queryCaptureSequence)
	c.activeQueryCaptureID = captureID
	c.state.QueryCaptureID = captureID
	c.state.QueryCaptureBusy = true
	c.state.QueryCaptureCancelling = false
	c.state.QueryCapture = nil
	c.state.QueryCaptureError = ""
	epoch, generation := c.state.Epoch, c.sessionGeneration
	c.work.Add(1)
	state := c.commitLocked()
	c.mu.Unlock()
	c.emit(state)
	go c.runQueryCapture(session, request, captureID, epoch, generation)
	return nil
}

func (c *Controller) runQueryCapture(session queryCaptureSession, request desktop.QueryExportRequest, captureID string, epoch, generation uint64) {
	defer c.work.Done()
	capture, err := session.CaptureQueryExport(request)
	c.mu.Lock()
	if c.closed || c.activeQueryCaptureID != captureID || c.state.Epoch != epoch || c.sessionGeneration != generation {
		c.mu.Unlock()
		return
	}
	if c.state.QueryCaptureCancelling {
		c.activeQueryCaptureID = ""
		c.state.QueryCaptureID = ""
		c.state.QueryCaptureBusy = false
		c.state.QueryCaptureCancelling = false
		c.state.QueryCapture = nil
		c.state.QueryCaptureError = "Query capture was cancelled"
		state := c.commitLocked()
		c.mu.Unlock()
		c.emit(state)
		return
	}
	c.activeQueryCaptureID = ""
	c.state.QueryCaptureID = ""
	c.state.QueryCaptureBusy = false
	c.state.QueryCaptureCancelling = false
	if err != nil {
		c.state.QueryCaptureError = err.Error()
		c.state.QueryCapture = nil
	} else if capture.Context.SnapshotID != request.SnapshotID || capture.Context.RequestID != request.RequestID || capture.Context.Fingerprint != request.Fingerprint || capture.Format != request.Format {
		c.state.QueryCaptureError = "query capture identity changed while it was being frozen"
		c.state.QueryCapture = nil
	} else {
		capture.Content = append([]byte(nil), capture.Content...)
		capture.ProtectedPaths = append([]string(nil), capture.ProtectedPaths...)
		c.state.QueryCapture = &capture
		c.state.QueryCaptureError = ""
	}
	state := c.commitLocked()
	c.mu.Unlock()
	c.emit(state)
}

// cancelQueryCapture stops publication of an in-flight capture. The core's
// bounded serializer has no cancellation hook; any late bytes are discarded.
func (c *Controller) cancelQueryCapture() error {
	c.mu.Lock()
	if err := c.canOperateLocked(); err != nil {
		c.mu.Unlock()
		return err
	}
	if c.activeQueryCaptureID == "" {
		c.mu.Unlock()
		return nil
	}
	if c.state.QueryCaptureCancelling {
		c.mu.Unlock()
		return nil
	}
	c.state.QueryCaptureCancelling = true
	c.state.QueryCapture = nil
	c.state.QueryCaptureError = "Query capture was cancelled"
	state := c.commitLocked()
	c.mu.Unlock()
	c.emit(state)
	return nil
}

func (c *Controller) clearQueryCaptureLocked() {
	c.activeQueryCaptureID = ""
	c.state.QueryCaptureID = ""
	c.state.QueryCaptureBusy = false
	c.state.QueryCaptureCancelling = false
	c.state.QueryCapture = nil
	c.state.QueryCaptureError = ""
}
