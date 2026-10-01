// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import "C"

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"orangecount/internal/desktop"
	"orangecount/internal/query"
	"orangecount/internal/report"
)

const protocolVersion = 2

var sessions = struct {
	sync.Mutex
	next   uint64
	values map[uint64]*desktop.Session
}{values: make(map[uint64]*desktop.Session)}

type wireError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type envelope struct {
	ProtocolVersion int        `json:"protocol_version"`
	Data            any        `json:"data,omitempty"`
	Error           *wireError `json:"error,omitempty"`
}

func sessionResponse(data any, err error) *C.char {
	response := envelope{ProtocolVersion: protocolVersion, Data: data}
	if err != nil {
		var typed query.TypedError
		code, message := "internal", "Unable to complete the ledger request"
		switch {
		case errors.Is(err, desktop.ErrClosed):
			code, message = "session-closed", err.Error()
		case errors.Is(err, desktop.ErrRequest):
			code, message = "invalid-request", err.Error()
		case errors.Is(err, desktop.ErrStale):
			code, message = "snapshot-changed", err.Error()
		case errors.Is(err, desktop.ErrResultBudget):
			code, message = "result-budget-exceeded", err.Error()
		case errors.Is(err, desktop.ErrNoSnapshot):
			code, message = "no-snapshot", err.Error()
		case errors.As(err, &typed):
			code, message = typed.Code, typed.Message
		case errors.Is(err, report.ErrAccountUnknown):
			code, message = "account-not-found", err.Error()
		case errors.Is(err, report.ErrCommodityUnknown):
			code, message = "commodity-not-found", err.Error()
		}
		response.Data, response.Error = nil, &wireError{Code: code, Message: message}
	}
	encoded, encodeErr := json.Marshal(response)
	if encodeErr != nil {
		encoded = []byte(`{"protocol_version":2,"error":{"code":"encoding","message":"Unable to encode ledger response"}}`)
	}
	return C.CString(string(encoded))
}

//export OC_ExecuteQuery
func OC_ExecuteQuery(handle C.ulonglong, requestJSON *C.char) *C.char {
	session, err := sessionFor(uint64(handle))
	if err != nil {
		return sessionResponse(nil, err)
	}
	if requestJSON == nil {
		return sessionResponse(nil, desktop.ErrRequest)
	}
	raw := C.GoString(requestJSON)
	if len(raw) > query.QueryTextLimit*6+4096 {
		return sessionResponse(nil, query.TypedError{Code: "text-budget-exceeded", Message: "Query request exceeds the text budget"})
	}
	var request desktop.QueryRequest
	if err := json.Unmarshal([]byte(raw), &request); err != nil {
		return sessionResponse(nil, desktop.ErrRequest)
	}
	result, err := session.ExecuteQuery(context.Background(), request)
	return sessionResponse(result, err)
}

//export OC_CancelQuery
func OC_CancelQuery(handle C.ulonglong, requestID C.ulonglong) *C.char {
	session, err := sessionFor(uint64(handle))
	if err != nil {
		return sessionResponse(nil, err)
	}
	err = session.CancelQuery(uint64(requestID))
	return sessionResponse(struct {
		Accepted bool `json:"accepted"`
	}{true}, err)
}

func sessionFor(handle uint64) (*desktop.Session, error) {
	sessions.Lock()
	defer sessions.Unlock()
	value := sessions.values[handle]
	if value == nil {
		return nil, desktop.ErrClosed
	}
	return value, nil
}

//export OC_OpenLedger
func OC_OpenLedger(path *C.char) *C.char {
	return openLedger(path, desktop.OpenOptions{})
}

//export OC_OpenLedgerWithOptions
func OC_OpenLedgerWithOptions(path *C.char, optionsJSON *C.char) *C.char {
	if optionsJSON == nil {
		return sessionResponse(nil, desktop.ErrRequest)
	}
	var options desktop.OpenOptions
	if err := json.Unmarshal([]byte(C.GoString(optionsJSON)), &options); err != nil {
		return sessionResponse(nil, desktop.ErrRequest)
	}
	return openLedger(path, options)
}

func openLedger(path *C.char, options desktop.OpenOptions) *C.char {
	if path == nil {
		return sessionResponse(nil, desktop.ErrRequest)
	}
	session, err := desktop.OpenWithOptions(C.GoString(path), options)
	if err != nil {
		return sessionResponse(nil, err)
	}
	sessions.Lock()
	sessions.next++
	handle := sessions.next
	sessions.values[handle] = session
	sessions.Unlock()
	state, err := session.Poll()
	return sessionResponse(struct {
		Handle uint64        `json:"handle"`
		State  desktop.State `json:"state"`
	}{handle, state}, err)
}

//export OC_PollState
func OC_PollState(handle C.ulonglong, after C.ulonglong) *C.char {
	session, err := sessionFor(uint64(handle))
	if err != nil {
		return sessionResponse(nil, err)
	}
	state, err := session.Poll()
	return sessionResponse(struct {
		Changed bool          `json:"changed"`
		State   desktop.State `json:"state"`
	}{state.Revision != uint64(after), state}, err)
}

//export OC_ReadView
func OC_ReadView(handle C.ulonglong, requestJSON *C.char) *C.char {
	session, err := sessionFor(uint64(handle))
	if err != nil {
		return sessionResponse(nil, err)
	}
	if requestJSON == nil {
		return sessionResponse(nil, desktop.ErrRequest)
	}
	var request desktop.Request
	if err := json.Unmarshal([]byte(C.GoString(requestJSON)), &request); err != nil {
		return sessionResponse(nil, desktop.ErrRequest)
	}
	view, err := session.Read(request)
	return sessionResponse(view, err)
}

//export OC_ReloadLedger
func OC_ReloadLedger(handle C.ulonglong) *C.char {
	session, err := sessionFor(uint64(handle))
	if err != nil {
		return sessionResponse(nil, err)
	}
	if err := session.Reload(); err != nil {
		return sessionResponse(nil, err)
	}
	state, err := session.Poll()
	return sessionResponse(state, err)
}

//export OC_CloseLedger
func OC_CloseLedger(handle C.ulonglong) *C.char {
	sessions.Lock()
	session := sessions.values[uint64(handle)]
	delete(sessions.values, uint64(handle))
	sessions.Unlock()
	if session != nil {
		session.Close()
	}
	return sessionResponse(struct {
		Closed bool `json:"closed"`
	}{true}, nil)
}
