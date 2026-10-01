// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package ledger

import (
	"orangecount/internal/diagnostic"
	"orangecount/internal/source"
	"sort"
)

// Metadata stacks are lexical and local to a parsed file. Expand into a
// transaction copy before chronological evaluation, never into the source AST
// or an included file's independently parsed scope. Explicit keys, including
// null values, override the active value.
func (e *evaluator) effectiveMetadata(file *File) []Directive {
	stack := make(map[string][]Metadata)
	result := make([]Directive, 0, len(file.Directives))
	for _, directive := range file.Directives {
		switch value := directive.(type) {
		case PushMeta:
			stack[value.Key] = append(stack[value.Key], Metadata{Key: value.Key, Value: value.Value, Span: value.Span()})
		case PopMeta:
			if values := stack[value.Key]; len(values) > 0 {
				if len(values) == 1 {
					delete(stack, value.Key)
				} else {
					stack[value.Key] = values[:len(values)-1]
				}
			} else {
				e.add("E-PARSE-METADATA-STACK", diagnostic.Error, value.Span(), sourcePath(file))
			}
		case *Transaction:
			copy := *value
			copy.Meta = effectiveTransactionMetadata(value.Meta, stack)
			directive = &copy
		case Transaction:
			value.Meta = effectiveTransactionMetadata(value.Meta, stack)
			directive = value
		}
		result = append(result, directive)
	}
	keys := make([]string, 0, len(stack))
	for key := range stack {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for range keys {
		// The reference reports a file-finalization error at line zero.
		span := source.Span{}
		if file.Source != nil {
			span.File = file.Source.ID
		}
		e.add("E-PARSE-METADATA-STACK", diagnostic.Error, span, sourcePath(file))
	}
	return result
}

func effectiveTransactionMetadata(explicit []Metadata, stack map[string][]Metadata) []Metadata {
	keys := make([]string, 0, len(stack))
	for key := range stack {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	explicitKeys := make(map[string]bool, len(explicit))
	for _, item := range explicit {
		explicitKeys[item.Key] = true
	}
	result := make([]Metadata, 0, len(explicit)+len(keys))
	for _, key := range keys {
		values := stack[key]
		if !explicitKeys[key] {
			result = append(result, values[len(values)-1])
		}
	}
	return append(result, explicit...)
}
