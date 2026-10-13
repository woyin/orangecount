// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package workspace

import (
	"context"

	"orangecount/internal/desktop"
)

type querySession interface {
	ExecuteQuery(context.Context, desktop.QueryRequest) (desktop.QueryResponse, error)
	CancelQuery(uint64) error
}

type queryCancellation struct {
	session querySession
	id      uint64
	cancel  context.CancelFunc
}

// executeQuery has a runner independent from report reads. Session-level
// request generations reject stale completions even if cancellation is late.
func (c *Controller) executeQuery(text string) error {
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
	session, ok := c.session.(querySession)
	if !ok {
		c.mu.Unlock()
		return ErrNoQuery
	}
	if !c.state.Ledger.HasSnapshot || c.state.Ledger.SnapshotID == "" {
		c.mu.Unlock()
		return desktop.ErrNoSnapshot
	}
	if c.querySequence == ^uint64(0) {
		c.mu.Unlock()
		return ErrQueryID
	}
	previous := c.invalidateQueryLocked()
	c.querySequence++
	requestID := c.querySequence
	ctx, cancel := context.WithCancel(context.Background())
	c.queryCancel = cancel
	c.activeQueryID = requestID
	c.state.QueryRequestID = requestID
	c.state.QueryBusy = true
	c.state.QueryCancelling = false
	c.state.QueryResponse = nil
	c.state.QueryError = ""
	epoch, generation, snapshotID := c.state.Epoch, c.sessionGeneration, c.state.Ledger.SnapshotID
	c.work.Add(1)
	state := c.commitLocked()
	c.mu.Unlock()
	c.cancelQueryGeneration(previous)
	c.emit(state)
	go c.runQuery(ctx, session, desktop.QueryRequest{SnapshotID: snapshotID, RequestID: requestID, Text: text}, epoch, generation)
	return nil
}

func (c *Controller) runQuery(ctx context.Context, session querySession, request desktop.QueryRequest, epoch, generation uint64) {
	defer c.work.Done()
	response, err := session.ExecuteQuery(ctx, request)
	c.mu.Lock()
	if c.closed || c.activeQueryID != request.RequestID || c.state.Epoch != epoch || c.sessionGeneration != generation || c.state.Ledger.SnapshotID != request.SnapshotID {
		c.mu.Unlock()
		return
	}
	c.activeQueryID = 0
	if c.queryCancel != nil {
		c.queryCancel()
		c.queryCancel = nil
	}
	c.state.QueryBusy = false
	c.state.QueryCancelling = false
	c.state.QueryResponse = &response
	if err != nil {
		c.state.QueryError = err.Error()
	} else if response.Error != nil {
		c.state.QueryError = response.Error.Message
	} else {
		c.state.QueryError = ""
	}
	state := c.commitLocked()
	c.mu.Unlock()
	c.emit(state)
}

func (c *Controller) cancelQuery() error {
	c.mu.Lock()
	if err := c.canOperateLocked(); err != nil {
		c.mu.Unlock()
		return err
	}
	if c.activeQueryID == 0 {
		c.mu.Unlock()
		return nil
	}
	session, ok := c.session.(querySession)
	if !ok {
		c.mu.Unlock()
		return ErrNoQuery
	}
	id := c.activeQueryID
	cancel := c.queryCancel
	c.state.QueryCancelling = true
	state := c.commitLocked()
	c.mu.Unlock()
	c.emit(state)
	err := session.CancelQuery(id)
	if cancel != nil {
		cancel()
	}
	if err != nil {
		c.mu.Lock()
		if !c.closed && c.activeQueryID == id {
			c.state.QueryError = err.Error()
			c.state.QueryCancelling = false
			state := c.commitLocked()
			c.mu.Unlock()
			c.emit(state)
			return err
		}
		c.mu.Unlock()
	}
	return err
}

func (c *Controller) invalidateQueryLocked() *queryCancellation {
	if c.activeQueryID == 0 && c.queryCancel == nil {
		return nil
	}
	cancellation := &queryCancellation{id: c.activeQueryID, cancel: c.queryCancel}
	if session, ok := c.session.(querySession); ok {
		cancellation.session = session
	}
	if cancellation.cancel != nil {
		cancellation.cancel()
	}
	c.activeQueryID = 0
	c.queryCancel = nil
	c.state.QueryBusy = false
	c.state.QueryCancelling = false
	c.state.QueryResponse = nil
	c.state.QueryError = ""
	return cancellation
}

func (c *Controller) cancelQueryGeneration(cancellation *queryCancellation) {
	if cancellation == nil {
		return
	}
	if cancellation.session != nil && cancellation.id != 0 {
		_ = cancellation.session.CancelQuery(cancellation.id)
	}
	if cancellation.cancel != nil {
		cancellation.cancel()
	}
}
