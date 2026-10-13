// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// importDateFixture opens the account the CSV rows post to and the default
// offset account, so a preview failure can only come from the date column.
const importDateFixture = "2000-01-01 open Assets:Cash USD\n" +
	"2000-01-01 open Equity:Imported USD\n"

// csvPreviewRows decodes the previewed rows of a successful import preview.
func csvPreviewRows(t *testing.T, body string) []map[string]any {
	t.Helper()
	var payload struct {
		Rows struct {
			Rows []map[string]any `json:"rows"`
		} `json:"rows"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("preview payload=%q err=%v", body, err)
	}
	return payload.Rows.Rows
}

// TestCSVImportDateLayoutPinsRequestContract drives POST /api/v1/import/preview
// with and without the optional date_layout field. The absent field must keep
// the historical ISO YYYY-MM-DD contract, an explicit ISO layout must render the
// same date, and a week layout must resolve to the calendar date the declared
// fields agree on (2020-W01-1 is 2019-12-30). A rejected request must name the
// offending row and datefmt's stable failure code, and must never publish a
// preview.
func TestCSVImportDateLayoutPinsRequestContract(t *testing.T) {
	server, _ := newAdapterServer(t, importDateFixture)
	preview := func(t *testing.T, body string) (int, string) {
		t.Helper()
		response := post(t, server, "/api/v1/import/preview", body)
		return response.Code, response.Body.String()
	}
	csv := func(dateLayout, content string) string {
		layout := ""
		if dateLayout != "" {
			layout = fmt.Sprintf(`,"date_layout":%q`, dateLayout)
		}
		contentJSON, err := json.Marshal(content)
		if err != nil {
			t.Fatal(err)
		}
		return fmt.Sprintf(`{"path":"bank.csv","adapter":"csv"%s,"content":%s}`, layout, contentJSON)
	}
	for _, tc := range []struct {
		name        string
		dateLayout  string
		content     string
		wantStatus  int
		wantDate    string
		wantMessage []string
	}{
		{
			name:       "absent layout keeps ISO default",
			content:    "date,account,amount\n2000-01-03,Assets:Cash,5\n",
			wantStatus: http.StatusOK,
			wantDate:   "2000-01-03",
		},
		{
			name:       "explicit ISO layout renders the same date",
			dateLayout: "%Y-%m-%d",
			content:    "date,account,amount\n2000-01-03,Assets:Cash,5\n",
			wantStatus: http.StatusOK,
			wantDate:   "2000-01-03",
		},
		{
			name:       "ISO week layout resolves the calendar date",
			dateLayout: "%G-W%V-%u",
			content:    "date,account,amount\n2020-W01-1,Assets:Cash,5\n",
			wantStatus: http.StatusOK,
			wantDate:   "2019-12-30",
		},
		{
			name:       "ISO week 53 rolls into the next calendar year",
			dateLayout: "%G-W%V-%u",
			content:    "date,account,amount\n2020-W53-7,Assets:Cash,5\n",
			wantStatus: http.StatusOK,
			wantDate:   "2021-01-03",
		},
		{
			name:       "unsupported layout directive is rejected before the rows",
			dateLayout: "%Q",
			content:    "date,account,amount\n2000-01-03,Assets:Cash,5\n",
			wantStatus: http.StatusBadRequest,
			wantMessage: []string{
				"invalid date layout", "unsupported CSV DateFormat directive '%Q'",
			},
		},
		{
			name:        "incompatible layout directives are rejected with a syntax code",
			dateLayout:  "%V-%u",
			content:     "date,account,amount\n01-1,Assets:Cash,5\n",
			wantStatus:  http.StatusBadRequest,
			wantMessage: []string{"CSV row 2", "date-format-syntax"},
		},
		{
			name:        "nonexistent ISO week 53 is rejected with the row number",
			dateLayout:  "%G-W%V-%u",
			content:     "date,account,amount\n2021-W53-1,Assets:Cash,5\n",
			wantStatus:  http.StatusBadRequest,
			wantMessage: []string{"CSV row 2", "2021-W53-1", "%G-W%V-%u", "date-value-out-of-range"},
		},
		{
			name:        "ISO week 00 is rejected as a value mismatch",
			dateLayout:  "%G-W%V-%u",
			content:     "date,account,amount\n2020-W00-1,Assets:Cash,5\n",
			wantStatus:  http.StatusBadRequest,
			wantMessage: []string{"CSV row 2", "date-value-mismatch"},
		},
		{
			name:        "week layout with an ISO date cell is a mismatch",
			dateLayout:  "%G-W%V-%u",
			content:     "date,account,amount\n2000-01-03,Assets:Cash,5\n",
			wantStatus:  http.StatusBadRequest,
			wantMessage: []string{"CSV row 2", "date-value-mismatch"},
		},
		{
			name:        "calendar and week fields that disagree are a conflict",
			dateLayout:  "%Y-%m-%d %G-W%V-%u",
			content:     "date,account,amount\n2000-01-03 2020-W01-1,Assets:Cash,5\n",
			wantStatus:  http.StatusBadRequest,
			wantMessage: []string{"CSV row 2", "date-value-conflict"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := preview(t, csv(tc.dateLayout, tc.content))
			if status != tc.wantStatus {
				t.Fatalf("status=%d body=%q", status, body)
			}
			for _, want := range tc.wantMessage {
				if !strings.Contains(body, want) {
					t.Fatalf("body=%q missing %q", body, want)
				}
			}
			if tc.wantStatus != http.StatusOK {
				if strings.Contains(body, "preview_id") {
					t.Fatalf("rejected import published a preview: %q", body)
				}
				return
			}
			if !strings.Contains(body, "preview_id") {
				t.Fatalf("body=%q missing preview_id", body)
			}
			rows := csvPreviewRows(t, body)
			if len(rows) == 0 {
				t.Fatalf("body=%q has no previewed rows", body)
			}
			for _, row := range rows {
				if date := fmt.Sprintf("%v", row["date"]); date != tc.wantDate {
					t.Fatalf("row=%v dated %s, want %s", row, date, tc.wantDate)
				}
			}
		})
	}
}

// TestCSVImportDateLayoutAppliesOnlyToCSV pins the adapter contract: a layout
// on a non-CSV adapter is refused instead of silently ignored.
func TestCSVImportDateLayoutAppliesOnlyToCSV(t *testing.T) {
	server, _ := newAdapterServer(t, importDateFixture)
	response := post(t, server, "/api/v1/import/preview",
		`{"path":"bank.bean","adapter":"beancount","date_layout":"%Y-%m-%d","content":"2000-01-03 * \"x\"\n  Assets:Cash 1 USD\n  Equity:Imported -1 USD\n"}`)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "date_layout applies only to the csv adapter") {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
}

// TestCSVToBeancountDateLayoutKeepsRenderingAndAllOrNothing pins the conversion
// contract below the HTTP layer: a blank layout renders exactly the historical
// ISO output, a declared week layout renders the resolved calendar date, and one
// bad row discards every already-rendered row.
func TestCSVToBeancountDateLayoutKeepsRenderingAndAllOrNothing(t *testing.T) {
	const header = "date,payee,account,amount,currency,narration\n"
	const isoOutput = "2000-01-03 * \"Cafe\" \"Lunch\"\n  Assets:Cash 5 USD\n  Equity:Imported -5 USD\n"
	for _, tc := range []struct {
		name       string
		layout     string
		content    string
		wantOutput string
		wantErr    []string
	}{
		{name: "blank layout keeps the ISO rendering", content: header + "2000-01-03,Cafe,Assets:Cash,5,USD,Lunch\n", wantOutput: isoOutput},
		{name: "explicit ISO layout matches the blank layout", layout: "%Y-%m-%d", content: header + "2000-01-03,Cafe,Assets:Cash,5,USD,Lunch\n", wantOutput: isoOutput},
		{
			name:       "week layout renders the resolved date",
			layout:     "%G-W%V-%u",
			content:    header + "2020-W01-1,Cafe,Assets:Cash,5,USD,Lunch\n",
			wantOutput: "2019-12-30 * \"Cafe\" \"Lunch\"\n  Assets:Cash 5 USD\n  Equity:Imported -5 USD\n",
		},
		{
			name:    "invalid layout fails before the header is read",
			layout:  "%Q",
			content: header + "2000-01-03,Cafe,Assets:Cash,5,USD,Lunch\n",
			wantErr: []string{"invalid date layout", "unsupported CSV DateFormat directive '%Q'"},
		},
		{
			name:    "trailing percent is a syntax failure",
			layout:  "%Y-%m-%d%",
			content: header + "2000-01-03,Cafe,Assets:Cash,5,USD,Lunch\n",
			wantErr: []string{"invalid date layout", "bare '%'"},
		},
		{
			name:    "invalid layout wins over an empty file",
			layout:  "%Q",
			content: "",
			wantErr: []string{"invalid date layout"},
		},
		{
			name:    "one bad week row discards the whole batch",
			layout:  "%G-W%V-%u",
			content: header + "2020-W01-1,Cafe,Assets:Cash,5,USD,Lunch\n2021-W53-1,Cafe,Assets:Cash,6,USD,Dinner\n",
			wantErr: []string{"CSV row 3", "date-value-out-of-range"},
		},
		{
			name:    "blank layout still rejects a non-ISO date",
			content: header + "2020-W01-1,Cafe,Assets:Cash,5,USD,Lunch\n",
			wantErr: []string{"CSV row 2", "invalid date"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output, err := csvToBeancount(tc.content, nil, tc.layout)
			if len(tc.wantErr) == 0 {
				if err != nil || output != tc.wantOutput {
					t.Fatalf("output=%q err=%v want %q", output, err, tc.wantOutput)
				}
				return
			}
			for _, want := range tc.wantErr {
				if err == nil || !strings.Contains(err.Error(), want) {
					t.Fatalf("output=%q err=%v missing %q", output, err, want)
				}
			}
			if output != "" {
				t.Fatalf("failed conversion returned partial output %q", output)
			}
		})
	}
}
