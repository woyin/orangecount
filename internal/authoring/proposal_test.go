// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package authoring

import "testing"

func TestSerializeEntriesRendersTransactionBalanceAndNote(t *testing.T) {
	text, err := SerializeEntries([]Entry{
		{
			Type: "transaction", Date: "2026-08-06", Payee: `Bob's "Cafe"`, Narration: "Lunch",
			Tags: []string{"meal"}, Links: []string{"trip"},
			Postings: []Posting{
				{Account: "Assets:Bank:Cash", Amount: "-2.50", Currency: "USD"},
				{Account: "Expenses:Food"},
			},
		},
		{Type: "balance", Date: "2026-08-07", Account: "Assets:Bank:Cash", Amount: "10", Currency: "USD"},
		{Type: "note", Date: "2026-08-08", Account: "Assets:Bank:Cash", Comment: "checked"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "2026-08-06 * \"Bob's \\\"Cafe\\\"\" \"Lunch\" #meal ^trip\n" +
		"  Assets:Bank:Cash -2.50 USD\n" +
		"  Expenses:Food\n\n" +
		"2026-08-07 balance Assets:Bank:Cash 10 USD\n\n" +
		"2026-08-08 note Assets:Bank:Cash \"checked\""
	if text != want {
		t.Fatalf("text=%q want=%q", text, want)
	}
}

func TestSerializeEntriesRejectsInvalidProposal(t *testing.T) {
	cases := []Entry{
		{Type: "transaction", Date: "2026-8-6", Narration: "x", Postings: []Posting{{Account: "Assets:A:B"}}},
		{Type: "transaction", Date: "2026-08-06", Flag: "?", Narration: "x", Postings: []Posting{{Account: "Assets:A:B"}}},
		{Type: "transaction", Date: "2026-08-06", Narration: "line\nbreak", Postings: []Posting{{Account: "Assets:A:B"}}},
		{Type: "transaction", Date: "2026-08-06", Narration: "x"},
		{Type: "balance", Date: "2026-08-06", Account: "assets:bad", Amount: "1", Currency: "USD"},
		{Type: "note", Date: "2026-08-06", Account: "Assets:A:B", Comment: "line\nbreak"},
		{Type: "note", Date: "2026-08-06", Comment: "x"},
		{Type: "price", Date: "2026-08-06"},
	}
	for _, entry := range cases {
		if _, err := SerializeEntries([]Entry{entry}); err == nil {
			t.Fatalf("accepted invalid entry: %+v", entry)
		}
	}
	if _, err := SerializeEntries(nil); err == nil {
		t.Fatal("accepted empty proposal")
	}
}
