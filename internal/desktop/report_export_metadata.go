// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package desktop

import (
	"encoding/json"

	"orangecount/internal/analysis"
	"orangecount/internal/query"
)

func journalMetadataColumn(id, name string) analysis.Column {
	fields := []analysis.Column{
		analysis.NewColumn("key", "Key", "string", false),
		analysis.NewColumn("type", "Type", "string", false),
		analysis.NewColumn("value_json", "Typed value JSON", "string", false),
	}
	return analysis.Column{ID: id, Name: name, Type: "list<record>", Fields: &fields}
}

func journalMetadataCell(metadata []DirectiveMetadata) (query.Cell, error) {
	rows := make([][]query.Cell, 0, len(metadata))
	for _, item := range metadata {
		value, err := journalMetadataValueJSON(item.Value, 0)
		if err != nil {
			return query.Cell{}, err
		}
		rows = append(rows, []query.Cell{
			stringCell(item.Key),
			stringCell(item.Value.Type),
			stringCell(string(value)),
		})
	}
	return query.Cell{Type: "list<record>", Value: rows}, nil
}

// journalMetadataValueJSON retains the typed value while deliberately omitting
// DirectiveMetadata.Source, which contains absolute filesystem paths.
func journalMetadataValueJSON(value DirectiveValue, depth int) (json.RawMessage, error) {
	if depth > 64 {
		return nil, analysis.ErrBudget
	}
	var encodedValue json.RawMessage
	switch value.Type {
	case "map":
		metadata, ok := value.Value.([]DirectiveMetadata)
		if !ok {
			return nil, analysis.ErrInvalid
		}
		entries := make([]json.RawMessage, 0, len(metadata))
		for _, item := range metadata {
			child, err := journalMetadataValueJSON(item.Value, depth+1)
			if err != nil {
				return nil, err
			}
			entry, err := json.Marshal(struct {
				Key   string          `json:"key"`
				Value json.RawMessage `json:"value"`
			}{item.Key, child})
			if err != nil {
				return nil, err
			}
			entries = append(entries, entry)
		}
		var err error
		encodedValue, err = json.Marshal(entries)
		if err != nil {
			return nil, err
		}
	case "list":
		values, ok := value.Value.([]DirectiveValue)
		if !ok {
			return nil, analysis.ErrInvalid
		}
		entries := make([]json.RawMessage, 0, len(values))
		for _, item := range values {
			encoded, err := journalMetadataValueJSON(item, depth+1)
			if err != nil {
				return nil, err
			}
			entries = append(entries, encoded)
		}
		var err error
		encodedValue, err = json.Marshal(entries)
		if err != nil {
			return nil, err
		}
	default:
		encoded, err := json.Marshal(value.Value)
		if err != nil {
			return nil, err
		}
		encodedValue = encoded
	}
	return json.Marshal(struct {
		Type  string          `json:"type"`
		Value json.RawMessage `json:"value"`
	}{value.Type, encodedValue})
}
