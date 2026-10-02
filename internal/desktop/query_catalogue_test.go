// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package desktop_test

import (
	"context"
	"errors"
	"orangecount/internal/desktop"
	"orangecount/internal/ledger"
	"os"
	"path/filepath"
	"testing"
)

func TestQueryCataloguePreservesDuplicatesAndIncludedSource(t *testing.T) {
	session, err := desktop.OpenWithOptions("../../testdata/fixtures/native/queries.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, _ := session.Poll()
	view, err := session.Read(desktop.Request{View: "query", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	catalogue := view.QueryCatalogue
	if catalogue == nil || catalogue.SnapshotID != state.SnapshotID || catalogue.Page.TotalCount != 3 || len(catalogue.Queries) != 1 || !catalogue.Page.HasMore {
		t.Fatalf("invalid catalogue: %+v", catalogue)
	}
	first := catalogue.Queries[0]
	if first.Name != "Included" || first.Text != "SELECT unknown(number) LIMIT 0" || first.Source.Line != 1 {
		t.Fatalf("include or invalid saved query discarded: %+v", first)
	}
	next, err := session.Read(desktop.Request{View: "query", SnapshotID: state.SnapshotID, Offset: 1, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	rows := next.QueryCatalogue.Queries
	if len(rows) != 2 || rows[0].Name != "同名" || rows[1].Name != "同名" || rows[0].ID == rows[1].ID || rows[0].Text == rows[1].Text {
		t.Fatalf("duplicate query identities lost: %+v", rows)
	}
}

func TestQueryCatalogueReloadRejectsPreviousPageIdentity(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"queries.bean", "queries-included.bean"} {
		input, err := os.ReadFile(filepath.Join("../../testdata/fixtures/native", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, name), input, 0600); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(directory, "queries.bean")
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	first, err := session.Read(desktop.Request{View: "query", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	input, _ := os.ReadFile(path)
	if err := os.WriteFile(path, append(input, []byte("\n2000-01-04 price USD 1 EUR\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	if err := session.Reload(); err != nil {
		t.Fatal(err)
	}
	current, err := session.Read(desktop.Request{View: "query", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if current.QueryCatalogue.SnapshotID == first.QueryCatalogue.SnapshotID || current.QueryCatalogue.Queries[0].ID == first.QueryCatalogue.Queries[0].ID {
		t.Fatal("old query identity relabelled after reload")
	}
	_, err = session.Read(desktop.Request{View: "query", SnapshotID: first.QueryCatalogue.SnapshotID, Offset: 1, Limit: 1})
	if !errors.Is(err, desktop.ErrStale) {
		t.Fatalf("old catalogue page accepted: %v", err)
	}
}

func TestQueryCatalogueProvidesExecutablePresets(t *testing.T) {
	session, err := desktop.OpenWithOptions("../../testdata/fixtures/query/presets.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	view, err := session.Read(desktop.Request{View: "query"})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.QueryCatalogue.Presets) != 4 {
		t.Fatal("preset catalogue missing")
	}
	for i, preset := range view.QueryCatalogue.Presets {
		response, err := session.ExecuteQuery(context.Background(), desktop.QueryRequest{SnapshotID: view.QueryCatalogue.SnapshotID, RequestID: uint64(i + 1), Text: preset.Text})
		if err != nil || response.Error != nil {
			t.Fatalf("preset %s failed: %v %v", preset.ID, err, response.Error)
		}
		if response.Execution == nil || len(response.Execution.Result.Rows) == 0 {
			t.Fatal("preset unexpectedly empty")
		}
	}
}

func TestQueryCataloguePeriodPresetsUseCapturedWindowWithoutExecuting(t *testing.T) {
	session, err := desktop.OpenWithOptions("../../testdata/fixtures/query/presets.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	view, err := session.Read(desktop.Request{View: "query", Window: "1m"})
	if err != nil {
		t.Fatal(err)
	}
	period := view.QueryCatalogue.Period
	if period == nil || period.DateFrom != "2000-02-01" || period.DateTo != "2000-02-21" || len(period.Presets) != 4 {
		t.Fatal("resolved period missing")
	}
	for i, preset := range period.Presets {
		response, err := session.ExecuteQuery(context.Background(), desktop.QueryRequest{SnapshotID: view.State.SnapshotID, RequestID: uint64(i + 1), Text: preset.Text})
		if err != nil || response.Error != nil {
			t.Fatalf("period preset failed: %v %v", err, response.Error)
		}
		if preset.ID == "monthly-income" || preset.ID == "monthly-expenses" {
			if response.Execution.ReturnedRows != 1 || response.Execution.Result.Rows[0][1].Value != "2" {
				t.Fatal("explicit period did not restrict months")
			}
		}
	}
	if view.Request.DateFrom != "" || view.Request.DateTo != "" {
		t.Fatal("catalogue rewrote request")
	}
}
