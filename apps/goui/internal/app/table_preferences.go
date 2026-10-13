// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
	"orangecount/apps/goui/internal/preferences"
)

type columnLayoutWrite struct {
	store    *preferences.Store
	path     string
	identity string
	layouts  map[string]ui.TableLayout
}

// columnTableStates gives persisted tables stable names independent of page
// order or the data currently visible in them.
func (m *Model) columnTableStates() map[string]*ui.ListState {
	return map[string]*ui.ListState{
		"account-amounts":        &m.accountAmountTable,
		"account-journal":        &m.accountJournalTable,
		"commodities":            &m.commodityTable,
		"holding-events":         &m.holdingEventTable,
		"holding-groups":         &m.holdingGroupTable,
		"holding-history":        &m.holdingHistoryTable,
		"holding-lots":           &m.holdingLotTable,
		"holding-history-detail": &m.holdingDetailTable,
		"journal":                &m.journalTable,
		"postings":               &m.postingTable,
		"price-history":          &m.priceExactTable,
		"prices":                 &m.priceTable,
		"statement-accounts":     &m.reportTree.List,
	}
}

func (m *Model) restoreColumnPreferences(path string, item preferences.LedgerPreferences) {
	m.columnLayoutPreferences = columnLayoutsFromPreferences(item)
	for key, state := range m.columnTableStates() {
		state.Columns = cloneColumnLayout(m.columnLayoutPreferences[key])
	}
	m.querySchemaKey = ""
	m.queryLayoutPreferenceKey = ""
	m.queryTable.Columns = ui.TableLayout{}
	m.columnLayoutPath = path
	m.columnLayoutIdentity = item.Identity
	m.columnLayoutRequested = columnLayoutDigest(item.Identity, m.currentColumnLayouts())
}

func (m *Model) currentColumnLayouts() map[string]ui.TableLayout {
	layouts := cloneColumnLayouts(m.columnLayoutPreferences)
	for key, state := range m.columnTableStates() {
		if layout := boundedColumnLayout(state.Columns); hasColumnLayout(layout) {
			layouts[key] = layout
		} else {
			delete(layouts, key)
		}
	}
	if m.queryLayoutPreferenceKey != "" {
		if layout := boundedColumnLayout(m.queryTable.Columns); hasColumnLayout(layout) {
			layouts[m.queryLayoutPreferenceKey] = layout
		} else {
			delete(layouts, m.queryLayoutPreferenceKey)
		}
	}
	return boundedColumnLayouts(layouts, m.queryLayoutPreferenceKey)
}

func columnLayoutsFromPreferences(item preferences.LedgerPreferences) map[string]ui.TableLayout {
	layouts := make(map[string]ui.TableLayout)
	for key, order := range item.Columns {
		layout := layouts[key]
		layout.Order = append([]string(nil), order...)
		layout = boundedColumnLayout(layout)
		if hasColumnLayout(layout) {
			layouts[key] = layout
		}
	}
	for key, widths := range item.ColumnWidths {
		layout := layouts[key]
		layout.Widths = make(map[string]float32, len(widths))
		for column, width := range widths {
			layout.Widths[column] = width
		}
		layout = boundedColumnLayout(layout)
		if hasColumnLayout(layout) {
			layouts[key] = layout
		}
	}
	return boundedColumnLayouts(layouts, "")
}

func cloneColumnLayouts(layouts map[string]ui.TableLayout) map[string]ui.TableLayout {
	if len(layouts) == 0 {
		return make(map[string]ui.TableLayout)
	}
	clone := make(map[string]ui.TableLayout, len(layouts))
	for key, layout := range layouts {
		clone[key] = cloneColumnLayout(layout)
	}
	return clone
}

func cloneColumnLayout(layout ui.TableLayout) ui.TableLayout {
	clone := ui.TableLayout{Order: append([]string(nil), layout.Order...)}
	if len(layout.Widths) > 0 {
		clone.Widths = make(map[string]float32, len(layout.Widths))
		for column, width := range layout.Widths {
			clone.Widths[column] = width
		}
	}
	return clone
}

func boundedColumnLayout(layout ui.TableLayout) ui.TableLayout {
	const maxColumnsPerTable = 128
	if len(layout.Order) > maxColumnsPerTable {
		layout.Order = append([]string(nil), layout.Order[:maxColumnsPerTable]...)
	} else {
		layout.Order = append([]string(nil), layout.Order...)
	}
	if len(layout.Widths) > maxColumnsPerTable {
		keys := make([]string, 0, len(layout.Widths))
		for key := range layout.Widths {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		widths := make(map[string]float32, maxColumnsPerTable)
		for _, key := range keys[:maxColumnsPerTable] {
			widths[key] = layout.Widths[key]
		}
		layout.Widths = widths
	} else if len(layout.Widths) > 0 {
		widths := make(map[string]float32, len(layout.Widths))
		for key, width := range layout.Widths {
			widths[key] = width
		}
		layout.Widths = widths
	}
	return layout
}

func hasColumnLayout(layout ui.TableLayout) bool {
	return len(layout.Order) > 0 || len(layout.Widths) > 0
}

func boundedColumnLayouts(layouts map[string]ui.TableLayout, activeQueryKey string) map[string]ui.TableLayout {
	const maxLayouts = 32
	if len(layouts) <= maxLayouts {
		return layouts
	}
	bounded := make(map[string]ui.TableLayout, maxLayouts)
	keys := make([]string, 0, len(layouts))
	for key := range layouts {
		if !strings.HasPrefix(key, "query-schema-") {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	for _, key := range keys {
		if len(bounded) == maxLayouts {
			return bounded
		}
		bounded[key] = layouts[key]
	}
	queryKeys := make([]string, 0, len(layouts)-len(keys))
	for key := range layouts {
		if strings.HasPrefix(key, "query-schema-") && key != activeQueryKey {
			queryKeys = append(queryKeys, key)
		}
	}
	slices.Sort(queryKeys)
	if activeQueryKey != "" && len(bounded) < maxLayouts {
		if layout, ok := layouts[activeQueryKey]; ok {
			bounded[activeQueryKey] = layout
		}
	}
	for _, key := range queryKeys {
		if len(bounded) == maxLayouts {
			break
		}
		bounded[key] = layouts[key]
	}
	return bounded
}

func queryColumnLayoutKey(schema string) string {
	sum := sha256.Sum256([]byte(schema))
	return "query-schema-" + hex.EncodeToString(sum[:12])
}

func (m *Model) restoreQueryColumnLayout(schema string) {
	key := queryColumnLayoutKey(schema)
	if key == m.queryLayoutPreferenceKey {
		return
	}
	m.queryLayoutPreferenceKey = key
	m.queryTable.Columns = cloneColumnLayout(m.columnLayoutPreferences[key])
}

func columnLayoutDigest(identity string, layouts map[string]ui.TableLayout) string {
	data, _ := json.Marshal(struct {
		Identity string
		Layouts  map[string]ui.TableLayout
	}{Identity: identity, Layouts: layouts})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// persistColumnLayouts snapshots after UI construction because MyGo updates a
// table's layout while it processes the current frame's pointer input.
func (m *Model) persistColumnLayouts() {
	if m.closed.Load() || m.preferenceStore == nil || m.workspaceState.ActivePath == "" {
		return
	}
	path := m.workspaceState.ActivePath
	if m.columnLayoutPath != path {
		m.columnLayoutPath = path
		m.columnLayoutIdentity = ""
		m.columnLayoutRequested = ""
	}
	if m.columnLayoutIdentity == "" {
		identity, err := preferences.Identity(path)
		if err != nil {
			return
		}
		m.columnLayoutIdentity = identity
	}
	if m.columnLayoutIdentity == "" {
		return
	}
	layouts := m.currentColumnLayouts()
	if len(layouts) == 0 {
		return
	}
	digest := columnLayoutDigest(m.columnLayoutIdentity, layouts)
	if digest == m.columnLayoutRequested {
		return
	}
	m.columnLayoutRequested = digest
	m.columnLayoutPreferences = cloneColumnLayouts(layouts)
	write := &columnLayoutWrite{
		store: m.preferenceStore, path: path, identity: m.columnLayoutIdentity,
		layouts: layouts,
	}
	m.columnLayoutMu.Lock()
	if m.closed.Load() {
		m.columnLayoutMu.Unlock()
		return
	}
	m.columnLayoutPending = write
	if m.columnLayoutWorker {
		m.columnLayoutMu.Unlock()
		return
	}
	m.columnLayoutWorker = true
	m.preferenceWork.Add(1)
	m.columnLayoutMu.Unlock()
	go m.writeColumnLayouts()
}

func (m *Model) writeColumnLayouts() {
	defer m.preferenceWork.Done()
	for {
		m.columnLayoutMu.Lock()
		write := m.columnLayoutPending
		m.columnLayoutPending = nil
		if write == nil {
			m.columnLayoutWorker = false
			m.columnLayoutMu.Unlock()
			return
		}
		m.columnLayoutMu.Unlock()

		err := write.store.UpdateLedger(write.path, func(item *preferences.LedgerPreferences) error {
			if item.Identity != "" && item.Identity != write.identity {
				return fmt.Errorf("ledger preference identity changed")
			}
			item.Columns = make(map[string][]string, len(write.layouts))
			item.ColumnWidths = make(map[string]map[string]float32, len(write.layouts))
			for key, layout := range write.layouts {
				if len(layout.Order) > 0 {
					item.Columns[key] = append([]string(nil), layout.Order...)
				}
				if len(layout.Widths) > 0 {
					item.ColumnWidths[key] = make(map[string]float32, len(layout.Widths))
					for column, width := range layout.Widths {
						item.ColumnWidths[key][column] = width
					}
				}
			}
			return nil
		})
		if err != nil {
			m.showColumnPreferenceError(err)
		}
	}
}

func (m *Model) showColumnPreferenceError(err error) {
	window := m.window
	if window == nil || m.closed.Load() {
		return
	}
	message := "列布局未能保存：" + err.Error()
	window.Update(func() {
		if !m.closed.Load() {
			m.preferenceError = message
		}
	})
}
