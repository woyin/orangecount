// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package authoring

import (
	"strings"
)

type changeMode uint8

const (
	appendChange changeMode = iota + 1
	replaceChange
)

// Change is an immutable proposal to alter one source file in the ledger graph.
type Change struct {
	target  string
	mode    changeMode
	content []byte
}

// Append creates a change that appends one canonical blank-line-separated block.
func Append(target, block string) (Change, error) {
	if strings.TrimSpace(target) == "" {
		return Change{}, invalidProposalError("target is required")
	}
	if strings.TrimSpace(block) == "" {
		return Change{}, invalidProposalError("append block is required")
	}
	return Change{target: target, mode: appendChange, content: []byte(block)}, nil
}

// Replace creates a change that preserves the submitted bytes exactly.
func Replace(target string, content []byte) (Change, error) {
	if strings.TrimSpace(target) == "" {
		return Change{}, invalidProposalError("target is required")
	}
	return Change{target: target, mode: replaceChange, content: append([]byte(nil), content...)}, nil
}

// Target returns the graph display path this change addresses.
func (c Change) Target() string { return c.target }

func (c Change) apply(current []byte) []byte {
	if c.mode == replaceChange {
		return append([]byte(nil), c.content...)
	}
	prefix := strings.TrimRight(string(current), "\r\n")
	block := strings.TrimSpace(string(c.content))
	if prefix == "" {
		return []byte(block + "\n")
	}
	return []byte(prefix + "\n\n" + block + "\n")
}
