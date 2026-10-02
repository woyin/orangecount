// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package desktop

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"orangecount/internal/ledger"
	"orangecount/internal/query"
)

var ErrNoSnapshot = errors.New("no valid ledger snapshot is available for querying")

type QueryRequest struct {
	SnapshotID string `json:"snapshot_id"`
	RequestID  uint64 `json:"request_id"`
	Text       string `json:"text"`
}

type QueryContext struct {
	SnapshotID         string                  `json:"snapshot_id"`
	SemanticsProfile   ledger.SemanticsProfile `json:"semantics_profile"`
	EvaluationRevision string                  `json:"evaluation_revision"`
	RequestID          uint64                  `json:"request_id"`
	Text               string                  `json:"text"`
	Fingerprint        string                  `json:"fingerprint"`
}

type QueryFailure struct {
	Code    string           `json:"code"`
	Message string           `json:"message"`
	Phase   string           `json:"phase"`
	Range   *query.ByteRange `json:"range"`
}

type QueryResponse struct {
	Context   QueryContext     `json:"context"`
	Execution *query.Execution `json:"execution"`
	Error     *QueryFailure    `json:"error"`
}

// ExecuteQuery captures an immutable publication and never applies shared
// reading filters to the visible BQL. Per-session work is serialized; a new
// execution cancels both an older running request and any older waiting one.
func (s *Session) ExecuteQuery(ctx context.Context, request QueryRequest) (QueryResponse, error) {
	if request.RequestID == 0 {
		return QueryResponse{}, ErrRequest
	}
	if len(request.Text) > query.QueryTextLimit {
		return QueryResponse{}, query.TypedError{Code: "text-budget-exceeded", Message: "Query text exceeds 64 KiB of UTF-8"}
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return QueryResponse{}, ErrClosed
	}
	state, current := s.state, s.current
	if current == nil {
		s.mu.Unlock()
		return QueryResponse{}, ErrNoSnapshot
	}
	if request.SnapshotID == "" {
		s.mu.Unlock()
		return QueryResponse{}, ErrRequest
	}
	if request.SnapshotID != state.SnapshotID {
		s.mu.Unlock()
		return QueryResponse{}, ErrStale
	}
	if request.RequestID <= s.queryCancelledThrough || request.RequestID < s.queryLastID {
		s.mu.Unlock()
		result := capturedQueryResponse(state, request)
		result.Error = queryFailure(context.Canceled)
		return result, nil
	}
	if request.RequestID == s.queryLastID {
		s.mu.Unlock()
		return QueryResponse{}, ErrRequest
	}
	if s.queryCancel != nil && s.queryRequestID == request.RequestID {
		s.mu.Unlock()
		return QueryResponse{}, ErrRequest
	}
	if s.queryCancel != nil {
		s.queryCancel()
	}
	run, cancel := context.WithTimeout(ctx, 5*time.Second)
	s.queryCancel, s.queryRequestID = cancel, request.RequestID
	s.queryLastID = request.RequestID
	s.queryExport = nil
	s.queryExportProtectedPaths = nil
	s.mu.Unlock()
	defer func() {
		cancel()
		s.mu.Lock()
		if s.queryRequestID == request.RequestID {
			s.queryCancel = nil
			s.queryRequestID = 0
		}
		s.mu.Unlock()
	}()
	result := capturedQueryResponse(state, request)
	select {
	case s.queryGate <- struct{}{}:
		defer func() { <-s.queryGate }()
	case <-run.Done():
		result.Error = queryFailure(run.Err())
		return result, nil
	}
	if err := run.Err(); err != nil {
		result.Error = queryFailure(err)
		return result, nil
	}
	contextJSON, _ := json.Marshal(result.Context)
	execution, err := query.ExecuteTyped(run, request.Text, current.evaluation, query.ExecutionLimits{Bytes: query.QueryByteLimit - len(contextJSON) - 256})
	if err != nil {
		result.Error = queryFailure(err)
		return result, nil
	}
	result.Execution = &execution
	encoded, err := json.Marshal(result)
	if err != nil {
		return QueryResponse{}, err
	}
	if len(encoded) > query.QueryByteLimit-64 {
		result.Execution = nil
		result.Error = queryFailure(ErrResultBudget)
	}
	if err := run.Err(); err != nil {
		result.Execution = nil
		result.Error = queryFailure(err)
	}
	s.mu.Lock()
	closed, changed := s.closed, s.state.SnapshotID != state.SnapshotID
	if !closed && !changed && result.Execution != nil {
		if s.queryLastID != request.RequestID || request.RequestID <= s.queryCancelledThrough {
			result.Execution = nil
			result.Error = queryFailure(context.Canceled)
		} else {
			captured := cloneQueryResponse(result)
			s.queryExport = &captured
			s.queryExportProtectedPaths = current.protectedPaths
		}
	}
	s.mu.Unlock()
	if closed {
		return QueryResponse{}, ErrClosed
	}
	if changed {
		return QueryResponse{}, ErrStale
	}
	return result, nil
}

func capturedQueryResponse(state State, request QueryRequest) QueryResponse {
	result := QueryResponse{Context: QueryContext{SnapshotID: state.SnapshotID, SemanticsProfile: state.SemanticsProfile, EvaluationRevision: state.EvaluationRevision, RequestID: request.RequestID, Text: request.Text}}
	// Request generation identifies delivery, not the query's financial view.
	fingerprintContext := result.Context
	fingerprintContext.RequestID = 0
	identity, _ := json.Marshal(fingerprintContext)
	hash := sha256.Sum256(identity)
	result.Context.Fingerprint = hex.EncodeToString(hash[:])
	return result
}

func queryFailure(err error) *QueryFailure {
	failure := &QueryFailure{Code: "execution-error", Message: "Unable to execute this query"}
	var typed query.TypedError
	switch {
	case errors.Is(err, context.Canceled):
		failure.Code, failure.Message = "cancelled", "Query execution was cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		failure.Code, failure.Message = "deadline-exceeded", "Query execution exceeded its time budget"
	case errors.Is(err, ErrResultBudget):
		failure.Code, failure.Message = "result-budget-exceeded", err.Error()
	case errors.As(err, &typed):
		failure.Code, failure.Message, failure.Phase = typed.Code, typed.Message, typed.Phase
		if typed.Range != nil {
			location := *typed.Range
			failure.Range = &location
		}
	}
	return failure
}

// CancelQuery remembers a monotonically increasing request generation even
// before dispatch, and cancels that generation and older work in this window.
func (s *Session) CancelQuery(requestID uint64) error {
	if requestID == 0 {
		return ErrRequest
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	s.queryCancelledThrough = max(s.queryCancelledThrough, requestID)
	if s.queryCancel != nil && s.queryRequestID <= requestID {
		s.queryCancel()
	}
	return nil
}
