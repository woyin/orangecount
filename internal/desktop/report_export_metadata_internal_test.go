// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package desktop

import (
	"bytes"
	"testing"

	"orangecount/internal/presentation"
	"orangecount/internal/query"
)

func TestJournalMetadataValueJSONOmitsNestedSourcesAndRetainsExactAmounts(t *testing.T) {
	sourcePath := "/private/ledger/include.bean"
	value := DirectiveValue{
		Type: "map",
		Value: []DirectiveMetadata{{
			Key: "fee",
			Value: DirectiveValue{
				Type: "amount",
				Value: query.PresentedAmount{
					Number:   presentation.Decimal{Display: "1.23", Exact: "1.2300"},
					Currency: "USD",
				},
			},
			Source: SourceLocation{Path: sourcePath, Line: 4, Column: 3},
		}},
	}
	encoded, err := journalMetadataValueJSON(value, 0)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte(sourcePath)) || bytes.Contains(encoded, []byte(`"source"`)) {
		t.Fatal("nested metadata serialization exposed its source location")
	}
	if !bytes.Contains(encoded, []byte(`"type":"amount"`)) || !bytes.Contains(encoded, []byte(`"exact":"1.2300"`)) || !bytes.Contains(encoded, []byte(`"currency":"USD"`)) {
		t.Fatal("nested metadata lost its typed exact amount")
	}
}
