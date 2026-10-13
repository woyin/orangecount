// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"orangecount/apps/goui/internal/preferences"
	"orangecount/apps/goui/internal/workspace"
)

func TestColumnLayoutsPersistAndRestorePerLedger(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first.bean")
	second := filepath.Join(root, "second.bean")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("option \"title\" \"test\"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store := preferences.NewStore(filepath.Join(root, "support", "preferences.json"))
	if err := store.UpdateLedger(first, func(item *preferences.LedgerPreferences) error {
		item.Currency = "EUR"
		item.Columns = map[string][]string{"journal": {"payee", "date"}}
		item.ColumnWidths = map[string]map[string]float32{"journal": {"date": 126, "payee": 340}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateLedger(second, func(item *preferences.LedgerPreferences) error {
		item.Columns = map[string][]string{"journal": {"date", "payee"}}
		item.ColumnWidths = map[string]map[string]float32{"journal": {"date": 150}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	model := NewModel(true)
	model.preferenceStore = store
	model.workspaceState.ActivePath = first
	model.journalTable.Columns = ui.TableLayout{
		Order:  []string{"amount", "date"},
		Widths: map[string]float32{"date": 132, "amount": 210},
	}
	model.reportTree.List.Columns = ui.TableLayout{Order: []string{"account-detail", "account"}}
	model.persistColumnLayouts()
	model.preferenceWork.Wait()

	firstPreferences, err := store.ForLedger(first)
	if err != nil {
		t.Fatal(err)
	}
	if firstPreferences.Currency != "EUR" {
		t.Fatalf("layout update overwrote reading preferences: %+v", firstPreferences)
	}
	if !reflect.DeepEqual(firstPreferences.Columns["journal"], []string{"amount", "date"}) || firstPreferences.ColumnWidths["journal"]["amount"] != 210 {
		t.Fatalf("persisted journal layout = order:%v widths:%v", firstPreferences.Columns["journal"], firstPreferences.ColumnWidths["journal"])
	}
	if !reflect.DeepEqual(firstPreferences.Columns["statement-accounts"], []string{"account-detail", "account"}) {
		t.Fatalf("persisted statement layout = %v", firstPreferences.Columns["statement-accounts"])
	}

	restored := NewModel(true)
	restored.pendingOpen = true
	restored.pendingPath = first
	restored.pendingPreferences = firstPreferences
	restored.applyPendingOpenState(workspace.State{ActivePath: first, Phase: workspace.PhaseReady})
	if !reflect.DeepEqual(restored.journalTable.Columns.Order, []string{"amount", "date"}) || restored.journalTable.Columns.Widths["amount"] != 210 {
		t.Fatalf("restored first-ledger journal layout = %+v", restored.journalTable.Columns)
	}
	if !reflect.DeepEqual(restored.reportTree.List.Columns.Order, []string{"account-detail", "account"}) {
		t.Fatalf("restored statement layout = %+v", restored.reportTree.List.Columns)
	}

	secondPreferences, err := store.ForLedger(second)
	if err != nil {
		t.Fatal(err)
	}
	restored.pendingOpen = true
	restored.pendingPath = second
	restored.pendingPreferences = secondPreferences
	restored.applyPendingOpenState(workspace.State{ActivePath: second, Phase: workspace.PhaseReady})
	if !reflect.DeepEqual(restored.journalTable.Columns.Order, []string{"date", "payee"}) || restored.journalTable.Columns.Widths["date"] != 150 {
		t.Fatalf("second-ledger journal layout = %+v", restored.journalTable.Columns)
	}
	if len(restored.reportTree.List.Columns.Order) != 0 || len(restored.reportTree.List.Columns.Widths) != 0 {
		t.Fatalf("first ledger's statement layout leaked to second ledger: %+v", restored.reportTree.List.Columns)
	}
}

func TestQueryColumnLayoutIsolatedByResultSchema(t *testing.T) {
	firstSchema, secondSchema := "[{\"id\":\"c0\",\"name\":\"account\"}]", "[{\"id\":\"c0\",\"name\":\"amount\"}]"
	firstKey, secondKey := queryColumnLayoutKey(firstSchema), queryColumnLayoutKey(secondSchema)
	model := NewModel(true)
	model.columnLayoutPreferences = map[string]ui.TableLayout{
		firstKey:  {Order: []string{"c0"}, Widths: map[string]float32{"c0": 220}},
		secondKey: {Order: []string{"c1", "c0"}, Widths: map[string]float32{"c0": 180}},
	}
	model.restoreQueryColumnLayout(firstSchema)
	if model.queryLayoutPreferenceKey != firstKey || model.queryTable.Columns.Widths["c0"] != 220 {
		t.Fatalf("first schema layout = key:%q layout:%+v", model.queryLayoutPreferenceKey, model.queryTable.Columns)
	}
	model.restoreQueryColumnLayout(secondSchema)
	if model.queryLayoutPreferenceKey != secondKey || !reflect.DeepEqual(model.queryTable.Columns.Order, []string{"c1", "c0"}) || model.queryTable.Columns.Widths["c0"] != 180 {
		t.Fatalf("second schema layout = key:%q layout:%+v", model.queryLayoutPreferenceKey, model.queryTable.Columns)
	}
}

func TestQueryLayoutPersistsSchemaDigestWithoutColumnTitle(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "ledger.bean")
	if err := os.WriteFile(path, []byte("option \"title\" \"test\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := preferences.NewStore(filepath.Join(root, "preferences.json"))
	schema := `[{"id":"c0","name":"private account alias","type":"string"}]`
	key := queryColumnLayoutKey(schema)
	model := NewModel(true)
	model.preferenceStore = store
	model.workspaceState.ActivePath = path
	model.querySchemaKey = schema
	model.queryLayoutPreferenceKey = key
	model.queryTable.Columns = ui.TableLayout{Order: []string{"c0"}, Widths: map[string]float32{"c0": 240}}
	model.persistColumnLayouts()
	model.preferenceWork.Wait()

	data, err := os.ReadFile(filepath.Join(root, "preferences.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "private account alias") || !strings.Contains(string(data), key) {
		t.Fatalf("query preference stored schema text or omitted its digest: %s", data)
	}
	item, err := store.ForLedger(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(item.Columns[key], []string{"c0"}) || item.ColumnWidths[key]["c0"] != 240 {
		t.Fatalf("persisted query layout = order:%v widths:%v", item.Columns[key], item.ColumnWidths[key])
	}
}
