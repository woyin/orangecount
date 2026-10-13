// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package workspace

import (
	"errors"

	"orangecount/internal/desktop"
)

var ErrAttachmentRootsBusy = errors.New("attachment roots are already being updated")

type attachmentRootsSession interface {
	AttachmentRoots() (desktop.AttachmentRoots, error)
	ConfigureAttachmentRoots(desktop.AttachmentRootsRequest) (desktop.AttachmentRoots, error)
}

// ConfigureAttachmentRoots applies a compare-and-swap policy update to this
// window's Session. It blocks its caller, so UI code should call it from a
// worker goroutine. Close waits for the operation; results from a replaced
// Session are discarded.
func (c *Controller) ConfigureAttachmentRoots(request desktop.AttachmentRootsRequest) (desktop.AttachmentRoots, error) {
	c.mu.Lock()
	if err := c.canOperateLocked(); err != nil {
		c.mu.Unlock()
		return desktop.AttachmentRoots{}, err
	}
	if c.state.Phase == PhaseOpening || c.state.AttachmentRootsBusy {
		c.mu.Unlock()
		return desktop.AttachmentRoots{}, ErrAttachmentRootsBusy
	}
	if c.session == nil {
		c.mu.Unlock()
		return desktop.AttachmentRoots{}, ErrNoSession
	}
	session, ok := c.session.(attachmentRootsSession)
	if !ok {
		c.mu.Unlock()
		return desktop.AttachmentRoots{}, desktop.ErrRequest
	}
	if c.state.AttachmentRootsLoaded && request.PolicyRevision != c.state.AttachmentRoots.PolicyRevision {
		c.mu.Unlock()
		return desktop.AttachmentRoots{}, desktop.ErrAttachmentPolicy
	}
	generation := c.sessionGeneration
	c.state.AttachmentRootsBusy = true
	c.state.AttachmentRootsError = ""
	state := c.commitLocked()
	c.work.Add(1)
	c.mu.Unlock()
	c.emit(state)
	defer c.work.Done()

	updated, err := session.ConfigureAttachmentRoots(request)
	c.mu.Lock()
	if c.closed || c.sessionGeneration != generation || c.session == nil {
		c.mu.Unlock()
		return desktop.AttachmentRoots{}, desktop.ErrStale
	}
	c.state.AttachmentRootsBusy = false
	if err != nil {
		c.state.AttachmentRootsError = err.Error()
	} else {
		c.state.AttachmentRoots = updated
		c.state.AttachmentRootsLoaded = true
		c.state.AttachmentRootsError = ""
	}
	state = c.commitLocked()
	c.mu.Unlock()
	c.emit(state)
	return updated, err
}
