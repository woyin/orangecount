// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package workspace

import (
	"context"
	"errors"
	"fmt"

	"orangecount/internal/desktop"
)

type reportCaptureCancellation struct {
	session reportCaptureSession
	id      string
}

// captureReport starts projection capture independently from report reads and
// query execution. The core capture id gives cancellation its own direct path.
func (c *Controller) captureReport(request desktop.ReportExportRequest) error {
	c.mu.Lock()
	if err := c.canOperateLocked(); err != nil {
		c.mu.Unlock()
		return err
	}
	if c.state.Phase == PhaseOpening {
		c.mu.Unlock()
		return ErrBusy
	}
	if c.state.QueryCaptureBusy {
		c.mu.Unlock()
		return ErrBusy
	}
	if request.SnapshotID != c.state.Ledger.SnapshotID {
		c.mu.Unlock()
		return desktop.ErrStale
	}
	response := c.state.Response
	if response == nil || response.ReportExport == nil {
		c.mu.Unlock()
		return desktop.ErrExpiredCapture
	}
	identity := response.ReportExport
	if request.ProjectionID != identity.ProjectionID {
		c.mu.Unlock()
		return desktop.ErrExpiredCapture
	}
	if request.SnapshotID != identity.SnapshotID || request.CaptureScope != identity.CaptureScope {
		c.mu.Unlock()
		return desktop.ErrRequest
	}
	if c.session == nil {
		c.mu.Unlock()
		return ErrNoSession
	}
	session, ok := c.session.(reportCaptureSession)
	if !ok {
		c.mu.Unlock()
		return ErrNoCapture
	}
	if c.reportCaptureSequence == ^uint64(0) {
		c.mu.Unlock()
		return ErrCaptureID
	}
	previous := c.invalidateReportCaptureLocked()
	c.reportCaptureSequence++
	captureID := fmt.Sprintf("goui-report-%d", c.reportCaptureSequence)
	request.CaptureID = captureID
	c.activeReportCaptureID = captureID
	c.state.ReportCaptureID = captureID
	c.state.ReportCaptureBusy = true
	c.state.ReportCaptureCancelling = false
	c.state.ReportCapture = nil
	c.state.ReportCaptureError = ""
	epoch, generation := c.state.Epoch, c.sessionGeneration
	c.work.Add(1)
	state := c.commitLocked()
	c.mu.Unlock()
	c.cancelReportCaptureGeneration(previous)
	c.emit(state)
	go c.runReportCapture(session, request, epoch, generation)
	return nil
}

func (c *Controller) runReportCapture(session reportCaptureSession, request desktop.ReportExportRequest, epoch, generation uint64) {
	defer c.work.Done()
	capture, err := session.CaptureReportExport(request)
	c.mu.Lock()
	if c.closed || c.activeReportCaptureID != request.CaptureID || c.state.Epoch != epoch || c.sessionGeneration != generation || c.state.Ledger.SnapshotID != request.SnapshotID {
		c.mu.Unlock()
		return
	}
	c.activeReportCaptureID = ""
	c.state.ReportCaptureBusy = false
	c.state.ReportCaptureCancelling = false
	if err != nil {
		if errors.Is(err, context.Canceled) {
			c.state.ReportCaptureError = "Report capture was cancelled"
		} else {
			c.state.ReportCaptureError = err.Error()
		}
		c.state.ReportCapture = nil
	} else {
		capture.Content = append([]byte(nil), capture.Content...)
		capture.ProtectedPaths = append([]string(nil), capture.ProtectedPaths...)
		c.state.ReportCapture = &capture
		c.state.ReportCaptureError = ""
	}
	state := c.commitLocked()
	c.mu.Unlock()
	c.emit(state)
}

func (c *Controller) cancelReportCapture() error {
	c.mu.Lock()
	if err := c.canOperateLocked(); err != nil {
		c.mu.Unlock()
		return err
	}
	if c.activeReportCaptureID == "" {
		c.mu.Unlock()
		return nil
	}
	session, ok := c.session.(reportCaptureSession)
	if !ok {
		c.mu.Unlock()
		return ErrNoCapture
	}
	id := c.activeReportCaptureID
	c.state.ReportCaptureCancelling = true
	state := c.commitLocked()
	c.mu.Unlock()
	c.emit(state)
	err := session.CancelReportExport(id)
	if err != nil {
		c.mu.Lock()
		if !c.closed && c.activeReportCaptureID == id {
			c.state.ReportCaptureError = err.Error()
			c.state.ReportCaptureCancelling = false
			state := c.commitLocked()
			c.mu.Unlock()
			c.emit(state)
			return err
		}
		c.mu.Unlock()
	}
	return err
}

func (c *Controller) invalidateReportCaptureLocked() *reportCaptureCancellation {
	if c.activeReportCaptureID == "" {
		return nil
	}
	cancellation := &reportCaptureCancellation{id: c.activeReportCaptureID}
	if session, ok := c.session.(reportCaptureSession); ok {
		cancellation.session = session
	}
	c.activeReportCaptureID = ""
	c.state.ReportCaptureID = ""
	c.state.ReportCaptureBusy = false
	c.state.ReportCaptureCancelling = false
	c.state.ReportCapture = nil
	c.state.ReportCaptureError = ""
	return cancellation
}

func (c *Controller) cancelReportCaptureGeneration(cancellation *reportCaptureCancellation) {
	if cancellation == nil || cancellation.session == nil || cancellation.id == "" {
		return
	}
	_ = cancellation.session.CancelReportExport(cancellation.id)
}
