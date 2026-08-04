// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package favaadapter (P3, phase 1) implements the private, loopback-only
// Fava-shaped contract layer that the transplanted Fava frontend consumes
// (ADR-0033, docs/fava-contract-map.md).
//
// Ownership boundary (see docs/fava-frontend-transplant-plan.md):
//   - This package owns DTOs, the contract registry, and pure projections only.
//   - It never performs accounting calculation, mutable ledger state, source
//     file authorization, import parsing, or snapshot publication. Those stay
//     in internal/ledger, internal/report, internal/query, internal/snapshot,
//     and internal/source.
//   - No public Fava API compatibility is promised or documented here. The
//     envelope shape ({"data": ..., "mtime": "..."}) is modeled on the wire
//     contract "ledger_data" documented in docs/fava-contract-map.md and is
//     intended for the embedded client only.
//
// Provenance: this package contains no Fava source code. Every type and
// function is an independent Go mapping; the contract rows in
// docs/fava-contract-map.md list the Fava source module and the exact
// frontend validator for each shape.
//
// Wire compatibility rules observed in this phase:
//   - Exact values: ledger.Decimal values are marshalled as their canonical
//     exact string (internal/ledger/decimal.go), so JSON never mutates an
//     amount into a float.
//   - Stable ordering: all derived slices and maps are emitted in a sorted
//     order documented per DTO; no map iteration order may reach the wire.
//   - Redaction: only display-safe identifiers (source.Graph.DisplayPath,
//     source.SafeDisplayPath) and sanitized fixture data are ever emitted.
//   - The `fava_options` map intentionally returns an OPT-IN subset of the
//     Fava options (locale, theme, conversion/interval defaults); plugin-only
//     options (ADR-0024) are never projected.
package favaadapter

import (
	"time"
)

// AdapterError is the standardized error payload for the private adapter.
// It mirrors the {"error": "..."} wire shape documented in the contract map
// (row "envelope", frontend lib/fetch.ts error_response_validator), and is
// intentionally NOT a public API: the JSON tag is the only serialization
// contract, and no status code table is ever exposed to outside clients.
type AdapterError struct {
	// Error is a human-readable message. It is never derived from private
	// ledger data; adapter handlers must supply a fixed localizable message.
	Error string `json:"error"`
}

// Envelope wraps every adapter payload. Fava's JSON API wraps responses as
// {"data": ..., "mtime": "..."} and the frontend's fetch wrapper
// (api/index.ts fetch_and_handle_api_call) reads both fields, feeding mtime
// into the change-polling store. The adapter keeps the same shape for the
// transplanted client only.
type Envelope struct {
	// Data is the typed payload (Bootstrap, MetadataProjection, ...).
	Data any `json:"data"`
	// Mtime is a monotonically increasing snapshot timestamp string; the
	// frontend compares it numerically (stores/mtime.ts set_mtime).
	Mtime string `json:"mtime"`
}

// NewEnvelope builds an envelope with the given payload and snapshot
// fingerprint. Mtime is derived from the immutable snapshot's built-at
// timestamp (RFC3339Nano, UTC), not from wall-clock at request time, so a
// stable snapshot yields a stable mtime.
func NewEnvelope(data any, snapshotBuiltAt time.Time) Envelope {
	mtime := ""
	if !snapshotBuiltAt.IsZero() {
		mtime = snapshotBuiltAt.UTC().Format(time.RFC3339Nano)
	}
	return Envelope{Data: data, Mtime: mtime}
}

// FavaOption is one supported UI preference projected into fava_options.
// The frontend reads these keys by name (stores/fava_options.ts); the keys
// here are the Fava names retained for the transplanted client.
type FavaOption struct {
	// Key is the Fava option name, e.g. "conversion-currencies".
	Key string `json:"key"`
	// Value is the projected value; exactly one is set.
	Value string `json:"value,omitempty"`
}

// SupportedFavaOption declares a single opt-in Fava option projection. Only
// options listed here may appear in a bootstrap; plugin-only options are
// excluded per ADR-0024.
type SupportedFavaOption struct {
	Key    string
	Kind   string // "bool", "int", "string", "list"
	Header bool   // persisted or session-only UI preference (contract row: stores/fava_options.ts)
}

// SupportedFavaOptions is the registry of fava-option projections. It is an
// explicit allowlist so an unknown option name can never be projected.
var SupportedFavaOptions = [...]SupportedFavaOption{
	{Key: "locale", Kind: "string", Header: false},
	{Key: "currency-column", Kind: "int", Header: false},
	{Key: "indent", Kind: "int", Header: false},
}
