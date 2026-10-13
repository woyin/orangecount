// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"orangecount/apps/goui/internal/charts"
	"orangecount/apps/goui/internal/widgets"
	"orangecount/internal/desktop"
	"orangecount/internal/report"
)

func (m *Model) holdingsPage(c *ui.Context) {
	t := c.Theme()
	sectionIndex := map[string]int{"groups": 0, "events": 1, "history": 2, "lots": 3}[m.holdingSection]
	if ui.Segmented(c, &sectionIndex, m.tr("持仓"), m.tr("已实现"), m.tr("平均成本"), m.tr("批次详情")).Label(m.tr("持仓页面")).Changed() {
		m.holdingSection = []string{"groups", "events", "history", "lots"}[sectionIndex]
		m.holdingOffset = 0
		m.holdingSelection, m.holdingSelectedID = -1, ""
		m.requestView("holdings")
		return
	}
	m.holdingsFilters(c)
	sortIndex := 0
	if m.holdingDescending {
		sortIndex = 1
	}
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, m.tr("排序")).TextColor(t.TextMuted)
		if ui.Segmented(c, &sortIndex, m.tr("正序"), m.tr("倒序")).Label(m.tr("持仓排序")).Changed() {
			m.holdingDescending = sortIndex == 1
			m.holdingOffset = 0
			m.holdingSelection, m.holdingSelectedID = -1, ""
			m.requestView("holdings")
		}
	})
	response := m.responseFor("holdings")
	if response == nil || response.Holdings == nil || !m.holdingResponseMatchesFilters(response) {
		message := "正在读取持仓与成本…"
		switch {
		case m.workspaceState.Busy:
		case response != nil && response.Holdings != nil:
			message = "请应用持仓筛选以读取新结果"
		default:
			message = "当前筛选没有持仓或活动"
		}
		ui.Text(c, m.tr(message)).TextColor(t.TextMuted)
		return
	}
	payload := response.Holdings
	switch m.holdingSection {
	case "groups":
		m.holdingGroups(c, payload)
	case "events":
		m.holdingEvents(c, payload)
	case "history":
		m.holdingHistory(c, payload)
	case "lots":
		m.holdingLots(c, payload)
	}
}

func (m *Model) holdingsFilters(c *ui.Context) {
	t := c.Theme()
	groupBy := map[string]string{"account": "按账户", "commodity": "按商品", "cost_currency": "按成本币种"}[m.holdingGroupBy]
	if groupBy == "" {
		groupBy = "按账户"
	}
	scope := "含子账户"
	if m.holdingScope == "direct" {
		scope = "仅此账户"
	}
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Gap(3).Children(func() {
			ui.Text(c, m.tr("账户范围")).FontSize(11).TextColor(t.TextMuted)
			search := ui.TextInput(c, &m.holdingAccountDraft).Placeholder(m.tr("留空显示全部资产与负债")).Label(m.tr("持仓账户"))
			m.focusSearchInput(pageHoldings, search)
		})
		ui.Column(c).Gap(3).Children(func() {
			ui.Text(c, m.tr("包含范围")).FontSize(11).TextColor(t.TextMuted)
			if m.selectLocalized(c, &scope, []string{"含子账户", "仅此账户"}).Label(m.tr("持仓账户范围")).Changed() {
				if scope == "仅此账户" {
					m.holdingScope = "direct"
				} else {
					m.holdingScope = "subtree"
				}
			}
		})
		ui.Column(c).Gap(3).Children(func() {
			ui.Text(c, m.tr("分组方式")).FontSize(11).TextColor(t.TextMuted)
			if m.selectLocalized(c, &groupBy, []string{"按账户", "按商品", "按成本币种"}).Label(m.tr("持仓分组")).Changed() {
				switch groupBy {
				case "按商品":
					m.holdingGroupBy = "commodity"
				case "按成本币种":
					m.holdingGroupBy = "cost_currency"
				default:
					m.holdingGroupBy = "account"
				}
			}
		})
		if ui.Button(c, m.tr("应用持仓范围")).Clicked() {
			m.holdingAccount = strings.TrimSpace(m.holdingAccountDraft)
			if m.holdingSection == "lots" {
				if m.holdingEventID != "" {
					m.holdingSection = "events"
				} else {
					m.holdingSection = "groups"
				}
			}
			m.holdingOffset = 0
			m.holdingGroupID, m.holdingEventID = "", ""
			m.holdingSelection, m.holdingSelectedID = -1, ""
			m.requestView("holdings")
		}
	})
}

func (m *Model) holdingGroups(c *ui.Context, payload *report.HoldingsProjection) {
	t := c.Theme()
	page := payload.GroupsPage
	ui.Text(c, m.pageDescription(page, "个持仓分组")).TextColor(t.TextMuted)
	items := payload.Groups
	if m.holdingSelectedID == "" && len(items) > 0 {
		m.holdingSelectedID = items[0].ID
	}
	textCompare := func(a, b string) int { return strings.Compare(a, b) }
	m.holdingGroupTable.Sort = &m.holdingGroupSort
	m.holdingGroupTable.Selected = &m.holdingSelection
	table, err := widgets.NewStableTable(&m.holdingGroupTable, items,
		func(item report.HoldingGroup) string { return item.ID },
		func(item report.HoldingGroup) string { return item.Account + " " + item.Commodity },
		[]widgets.TableColumn[report.HoldingGroup]{
			{Column: ui.TableColumn{Title: m.tr("账户/域"), ID: "account", Width: 200, Fixed: true, Sortable: true}, Compare: func(a, b report.HoldingGroup) int {
				aAccount, bAccount := a.Account, b.Account
				if aAccount == "" {
					aAccount = strings.Join(a.Accounts, "\x00")
				}
				if bAccount == "" {
					bAccount = strings.Join(b.Accounts, "\x00")
				}
				return textCompare(aAccount, bAccount)
			}, Cell: func(c *ui.Context, item report.HoldingGroup) {
				if item.Account == "" {
					ui.Text(c, strings.Join(item.Accounts, ", ")).SingleLine()
				} else {
					ui.Text(c, item.Account).SingleLine()
				}
			}},
			{Column: ui.TableColumn{Title: m.tr("商品"), ID: "commodity", Width: 100, Sortable: true}, Compare: func(a, b report.HoldingGroup) int { return textCompare(a.Commodity, b.Commodity) }, Cell: func(c *ui.Context, item report.HoldingGroup) { ui.Text(c, item.Commodity) }},
			{Column: ui.TableColumn{Title: m.tr("库存类型"), ID: "kind", Width: 100, Sortable: true}, Compare: func(a, b report.HoldingGroup) int { return textCompare(a.InventoryKind, b.InventoryKind) }, Cell: func(c *ui.Context, item report.HoldingGroup) { ui.Text(c, m.tr(holdingKindLabel(item.InventoryKind))) }},
			{Column: ui.TableColumn{Title: m.tr("数量"), ID: "quantity", Width: 110, Align: ui.End, Sortable: true}, Compare: func(a, b report.HoldingGroup) int { return widgets.CompareAmounts(a.Quantity, b.Quantity) }, Cell: func(c *ui.Context, item report.HoldingGroup) { ui.Text(c, item.Quantity.Exact+" "+item.Commodity) }},
			{Column: ui.TableColumn{Title: m.tr("成本基数"), ID: "basis", Width: 130, Align: ui.End, Sortable: true}, Compare: func(a, b report.HoldingGroup) int { return widgets.CompareOptionalAmounts(a.CostBasis, b.CostBasis) }, Cell: func(c *ui.Context, item report.HoldingGroup) {
				ui.Text(c, holdingAmount(item.CostBasis, item.CostCurrency))
			}},
			{Column: ui.TableColumn{Title: m.tr("平均成本"), ID: "average", Width: 130, Align: ui.End, Sortable: true}, Compare: func(a, b report.HoldingGroup) int {
				return widgets.CompareOptionalAmounts(a.AverageCost, b.AverageCost)
			}, Cell: func(c *ui.Context, item report.HoldingGroup) {
				ui.Text(c, holdingAmount(item.AverageCost, item.CostCurrency))
			}},
			{Column: ui.TableColumn{Title: m.tr("未实现损益"), ID: "gain", Width: 150, Align: ui.End, Sortable: true}, Compare: func(a, b report.HoldingGroup) int {
				return widgets.CompareOptionalAmounts(a.UnrealizedGain, b.UnrealizedGain)
			}, Cell: func(c *ui.Context, item report.HoldingGroup) {
				ui.Text(c, holdingAmount(item.UnrealizedGain, holdingGroupMarketCurrency(item))).TextColor(t.Text)
			}},
		})
	if err != nil {
		ui.Text(c, m.tr("持仓表格不可用：")+err.Error()).TextColor(t.Danger)
		return
	}
	selected := table.IndexOf(m.holdingSelectedID)
	if selected < 0 && len(items) > 0 {
		selected = 0
		m.holdingSelectedID = table.IDAt(selected)
	}
	m.holdingSelection = selected
	rows := table.Render(c).Grow(1)
	if rows.Changed() {
		if item, ok := table.At(m.holdingSelection); ok {
			m.holdingSelectedID = item.ID
		}
	}
	selected = table.IndexOf(m.holdingSelectedID)
	if selected >= 0 {
		item, _ := table.At(selected)
		ui.Column(c).Padding(12).Gap(8).Background(t.Surface).Border(1, t.Border).Radius(8).Children(func() {
			ui.Row(c).Gap(10).Children(func() {
				ui.Text(c, item.Commodity+" · "+m.tr(holdingKindLabel(item.InventoryKind))).FontSize(16).Bold()
				ui.Spacer(c)
				if ui.Button(c, m.tr("查看当前批次")).Clicked() {
					m.holdingSection, m.holdingGroupID, m.holdingEventID = "lots", item.ID, ""
					m.holdingOffset = 0
					m.holdingSelection, m.holdingSelectedID = -1, ""
					m.requestView("holdings")
				}
				if hasAverageHistory(payload.AverageHistory, item.ID) && ui.Button(c, m.tr("查看平均成本曲线")).Clicked() {
					m.holdingSection, m.holdingGroupID, m.holdingEventID = "history", item.ID, ""
					m.holdingOffset = 0
					m.holdingSelection, m.holdingSelectedID = -1, ""
					m.requestView("holdings")
				}
			})
			accounts := item.Account
			if accounts == "" {
				accounts = strings.Join(item.Accounts, ", ")
			}
			ui.Text(c, m.tr("账户 · ")+accounts+m.tr("　数量 · ")+item.Quantity.Exact+" "+item.Commodity).TextColor(t.TextMuted)
			ui.Text(c, m.tr("成本基数 · ")+holdingAmount(item.CostBasis, item.CostCurrency)+m.tr("　平均成本 · ")+holdingAmount(item.AverageCost, item.CostCurrency))
			ui.Text(c, m.tr("成本估值 · ")+m.holdingValuationText(item.CostValuation)+m.tr("　市价估值 · ")+m.holdingValuationText(item.MarketValuation))
			gain := holdingAmount(item.UnrealizedGain, holdingGroupMarketCurrency(item))
			if item.UnrealizedGain == nil && item.GainReason != "" {
				gain += " · " + m.tr(holdingReasonLabel(item.GainReason))
			}
			ui.Text(c, m.tr("未实现损益 · ")+gain)
			renderHoldingQuotes(c, m, payload, item.MarketValuation)
		})
	}
	m.holdingPageControls(c, page)
	if len(items) == 0 {
		ui.Text(c, m.tr("当前日期与账户范围内没有剩余持仓。 ")).TextColor(t.TextMuted)
	}
}

func (m *Model) holdingEvents(c *ui.Context, payload *report.HoldingsProjection) {
	t := c.Theme()
	page := payload.EventsPage
	ui.Text(c, m.pageDescription(page, "笔减仓/实现事件")).TextColor(t.TextMuted)
	items := payload.Events
	columns := []ui.TableColumn{
		{Title: m.tr("日期"), ID: "date", Width: 110, Fixed: true},
		{Title: m.tr("账户"), ID: "account", Width: 180},
		{Title: m.tr("商品"), ID: "commodity", Width: 85},
		{Title: m.tr("类型"), ID: "kind", Width: 110},
		{Title: m.tr("数量"), ID: "quantity", Width: 100, Align: ui.End},
		{Title: m.tr("已实现损益"), ID: "gain", Width: 150, Align: ui.End},
	}
	selected := selectedHoldingIndex(items, m.holdingSelectedID, func(item report.ReductionEvent) string { return item.ID })
	if selected < 0 && m.holdingSelectedID == "" && len(items) > 0 {
		selected = 0
		m.holdingSelectedID = items[0].ID
	}
	m.holdingSelection = selected
	m.holdingEventTable.Selected = &m.holdingSelection
	m.holdingEventTable.Key = func(row int) any { return items[row].ID }
	m.holdingEventTable.Label = func(row int) string { return items[row].Date + " " + items[row].Account + " " + items[row].Commodity }
	rows := ui.Table(c, &m.holdingEventTable, columns, len(items), func(row, col int) {
		item := items[row]
		switch col {
		case 0:
			ui.Text(c, item.Date)
		case 1:
			ui.Text(c, item.Account).SingleLine()
		case 2:
			ui.Text(c, item.Commodity)
		case 3:
			ui.Text(c, m.tr(holdingKindLabel(item.Kind)))
		case 4:
			ui.Text(c, item.Quantity.Exact+" "+item.Commodity)
		case 5:
			ui.Text(c, eventGainText(item, m.tr)).TextColor(t.Text)
		}
	}).Grow(1)
	if rows.Changed() && m.holdingSelection >= 0 && m.holdingSelection < len(items) {
		m.holdingSelectedID = items[m.holdingSelection].ID
	}
	selected = selectedHoldingIndex(items, m.holdingSelectedID, func(item report.ReductionEvent) string { return item.ID })
	if selected >= 0 {
		item := items[selected]
		ui.Column(c).Padding(12).Gap(8).Background(t.Surface).Border(1, t.Border).Radius(8).Children(func() {
			ui.Row(c).Gap(10).Children(func() {
				ui.Text(c, item.Date+" · "+m.tr(holdingKindLabel(item.Kind))+" · "+item.Narration).Bold().Grow(1)
				if ui.Button(c, m.tr("查看消耗批次")).Disabled(item.LotsPage.TotalCount == 0).Clicked() {
					m.holdingSection, m.holdingGroupID, m.holdingEventID, m.holdingLotKind = "lots", "", item.ID, "consumed"
					m.holdingOffset = 0
					m.holdingSelection, m.holdingSelectedID = -1, ""
					m.requestView("holdings")
				}
				if item.MergeLotsPage.TotalCount > 0 && ui.Button(c, m.tr("查看合并来源")).Clicked() {
					m.holdingSection, m.holdingGroupID, m.holdingEventID, m.holdingLotKind = "lots", "", item.ID, "merged"
					m.holdingOffset = 0
					m.holdingSelection, m.holdingSelectedID = -1, ""
					m.requestView("holdings")
				}
			})
			ui.Text(c, m.tr("数量 ")+item.Quantity.Exact+" "+item.Commodity+m.tr("　消耗数量 ")+item.ConsumedQuantity.Exact+" "+item.Commodity).TextColor(t.TextMuted)
			ui.Text(c, m.tr("成本基数 · ")+holdingAmount(item.CostBasis, item.CostCurrency)+m.tr("　卖出收入 · ")+holdingAmount(item.Proceeds, &item.PriceCurrency))
			ui.Text(c, m.tr("按成本币种的收入 · ")+holdingAmount(item.ProceedsAtCost, item.CostCurrency)+m.tr("　交易费用 · ")+holdingAmount(item.Fees, item.CostCurrency))
			ui.Text(c, m.tr("已实现损益 · ")+eventGainText(item, m.tr))
			if item.NetReason != "" {
				ui.Text(c, m.tr(holdingReasonLabel(item.NetReason))).TextColor(t.TextMuted)
			}
			ui.Row(c).Gap(8).Children(func() {
				ui.Text(c, m.tr("来源 · ")+sourceLabel(item.Source)).TextColor(t.TextMuted).Grow(1)
				if item.Source.Path != "" && ui.Button(c, m.tr("显示事件来源")).Clicked() {
					m.revealLocation(item.Source.Path)
				}
				m.sourceEditorButton(c, item.Source.Path)
			})
		})
	}
	m.holdingPageControls(c, page)
	if len(items) == 0 {
		ui.Text(c, m.tr("当前范围内没有已实现或减仓事件。 ")).TextColor(t.TextMuted)
	}
}

func (m *Model) holdingHistory(c *ui.Context, payload *report.HoldingsProjection) {
	t := c.Theme()
	if m.holdingGroupID == "" {
		page := payload.HistoryPage
		ui.Text(c, m.pageDescription(page, "条平均成本序列")).TextColor(t.TextMuted)
		items := payload.AverageHistory
		columns := []ui.TableColumn{
			{Title: m.tr("账户"), ID: "account", Width: 200, Fixed: true},
			{Title: m.tr("商品"), ID: "commodity", Width: 90},
			{Title: m.tr("成本币种"), ID: "currency", Width: 100},
			{Title: m.tr("记录数"), ID: "points", Width: 90, Align: ui.End},
			{Title: m.tr("最新均价"), ID: "average", Width: 150, Align: ui.End},
		}
		selected := selectedHoldingIndex(items, m.holdingSelectedID, func(item report.AverageCostSeries) string { return item.ID })
		if selected < 0 && m.holdingSelectedID == "" && len(items) > 0 {
			selected = 0
			m.holdingSelectedID = items[0].ID
		}
		m.holdingSelection = selected
		m.holdingHistoryTable.Selected = &m.holdingSelection
		m.holdingHistoryTable.Key = func(row int) any { return items[row].ID }
		m.holdingHistoryTable.Label = func(row int) string { return items[row].Account + " " + items[row].Commodity }
		rows := ui.Table(c, &m.holdingHistoryTable, columns, len(items), func(row, col int) {
			item := items[row]
			switch col {
			case 0:
				if item.Account == "" {
					ui.Text(c, strings.Join(item.Accounts, ", ")).SingleLine()
				} else {
					ui.Text(c, item.Account).SingleLine()
				}
			case 1:
				ui.Text(c, item.Commodity)
			case 2:
				ui.Text(c, item.CostCurrency)
			case 3:
				ui.Text(c, fmt.Sprint(item.PointsPage.TotalCount))
			case 4:
				if len(item.Points) == 0 || item.Points[len(item.Points)-1].AverageCost == nil {
					ui.Text(c, "—")
				} else {
					ui.Text(c, item.Points[len(item.Points)-1].AverageCost.Exact+" "+item.CostCurrency)
				}
			}
		}).Grow(1)
		if rows.Changed() && m.holdingSelection >= 0 && m.holdingSelection < len(items) {
			m.holdingSelectedID = items[m.holdingSelection].ID
		}
		selected = selectedHoldingIndex(items, m.holdingSelectedID, func(item report.AverageCostSeries) string { return item.ID })
		if selected >= 0 {
			item := items[selected]
			ui.Row(c).Gap(8).Children(func() {
				ui.Text(c, item.Commodity+" · "+item.CostCurrency+" · "+strings.Join(item.Accounts, ", ")).TextColor(t.TextMuted).Grow(1)
				if ui.PrimaryButton(c, m.tr("查看完整曲线")).Clicked() {
					m.holdingGroupID = item.ID
					m.holdingOffset = 0
					m.holdingSelection, m.holdingSelectedID = -1, ""
					m.requestView("holdings")
				}
			})
		}
		m.holdingPageControls(c, page)
		if len(items) == 0 {
			ui.Text(c, m.tr("当前范围没有按成本计价的平均成本历史。 ")).TextColor(t.TextMuted)
		}
		return
	}
	if len(payload.AverageHistory) == 0 {
		ui.Text(c, m.tr("所选平均成本序列已不在当前快照中。 ")).TextColor(t.Danger)
		return
	}
	series := payload.AverageHistory[0]
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, series.Commodity+" · "+series.CostCurrency+" · "+strings.Join(series.Accounts, ", ")).Bold().Grow(1)
		if ui.Button(c, m.tr("返回曲线列表")).Clicked() {
			m.holdingGroupID = ""
			m.holdingOffset = 0
			m.requestView("holdings")
		}
	})
	points := make([]charts.Point, 0, len(series.Points))
	for _, point := range series.Points {
		exact, available := m.tr("空仓"), point.AverageCost != nil && point.Value != nil
		value := 0.0
		if point.AverageCost != nil {
			exact = point.AverageCost.Exact + " " + series.CostCurrency
		}
		if point.Value != nil {
			value = *point.Value
		}
		points = append(points, charts.Point{Label: point.Date, Value: value, Exact: exact, Available: available})
	}
	graph := charts.Series{Label: series.Commodity + " / " + series.CostCurrency, Color: ui.RGB(42, 119, 214), Points: points}
	LineChart(c, graph, m.tr)
	ExactTable(c, &m.holdingDetailTable, []charts.Series{graph}, m.tr)
	ui.Text(c, fmt.Sprintf(m.tr("历史点 · 显示 %d–%d"), series.PointsPage.Offset+1, series.PointsPage.Offset+series.PointsPage.ReturnedCount)).TextColor(t.TextMuted)
	m.holdingPageControls(c, series.PointsPage)
}

func (m *Model) holdingLots(c *ui.Context, payload *report.HoldingsProjection) {
	t := c.Theme()
	var lots []report.HoldingLot
	var page report.ProjectionPage
	if m.holdingEventID != "" {
		if len(payload.Events) == 0 {
			ui.Text(c, m.tr("所选事件不在当前快照中。 ")).TextColor(t.Danger)
			return
		}
		event := payload.Events[0]
		if event.MergeLotsPage.TotalCount > 0 {
			lotKind := 0
			if m.holdingLotKind == "merged" {
				lotKind = 1
			}
			if ui.Segmented(c, &lotKind, m.tr("被消耗批次"), m.tr("合并前来源")).Label(m.tr("事件批次类型")).Changed() {
				if lotKind == 1 {
					m.holdingLotKind = "merged"
				} else {
					m.holdingLotKind = "consumed"
				}
				m.holdingOffset = 0
				m.holdingSelection, m.holdingSelectedID = -1, ""
				m.requestView("holdings")
				return
			}
		}
		if m.holdingLotKind == "merged" {
			lots, page = event.MergeLots, event.MergeLotsPage
		} else {
			lots, page = event.Lots, event.LotsPage
		}
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, event.Date+" · "+event.Narration+" · "+m.tr(holdingKindLabel(m.holdingLotKind))).Bold().Grow(1)
			if ui.Button(c, m.tr("返回已实现事件")).Clicked() {
				m.holdingSection = "events"
				m.holdingEventID, m.holdingGroupID = "", ""
				m.holdingOffset = 0
				m.requestView("holdings")
			}
		})
	} else if len(payload.Groups) > 0 {
		group := payload.Groups[0]
		lots, page = group.Lots, group.LotsPage
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, group.Commodity+" · "+group.Account+m.tr(" · 当前批次")).Bold().Grow(1)
			if ui.Button(c, m.tr("返回持仓")).Clicked() {
				m.holdingSection = "groups"
				m.holdingGroupID = ""
				m.holdingOffset = 0
				m.requestView("holdings")
			}
		})
	} else {
		ui.Text(c, m.tr("先从持仓或已实现事件选择一个批次详情。 ")).TextColor(t.TextMuted)
		return
	}
	ui.Text(c, m.pageDescription(page, "个批次")).TextColor(t.TextMuted)
	columns := []ui.TableColumn{
		{Title: m.tr("账户"), ID: "account", Width: 180, Fixed: true},
		{Title: m.tr("商品"), ID: "commodity", Width: 85},
		{Title: m.tr("数量"), ID: "quantity", Width: 100, Align: ui.End},
		{Title: m.tr("单位成本"), ID: "unit-cost", Width: 120, Align: ui.End},
		{Title: m.tr("成本基数"), ID: "basis", Width: 120, Align: ui.End},
		{Title: m.tr("成本日期/标签"), ID: "key", Width: 170},
		{Title: m.tr("来源类型"), ID: "origin", Width: 110},
		{Title: m.tr("来源"), ID: "source"},
	}
	selected := selectedHoldingIndex(lots, m.holdingSelectedID, func(item report.HoldingLot) string { return item.ID })
	if selected < 0 && m.holdingSelectedID == "" && len(lots) > 0 {
		selected = 0
		m.holdingSelectedID = lots[0].ID
	}
	m.holdingSelection = selected
	m.holdingLotTable.Selected = &m.holdingSelection
	m.holdingLotTable.Key = func(row int) any { return lots[row].ID }
	m.holdingLotTable.Label = func(row int) string {
		return lots[row].Account + " " + lots[row].Commodity + " " + lots[row].Quantity.Exact
	}
	rows := ui.Table(c, &m.holdingLotTable, columns, len(lots), func(row, col int) {
		item := lots[row]
		switch col {
		case 0:
			ui.Text(c, item.Account).SingleLine()
		case 1:
			ui.Text(c, item.Commodity)
		case 2:
			ui.Text(c, item.Quantity.Exact+" "+item.Commodity)
		case 3:
			ui.Text(c, holdingAmount(item.CostNumber, item.CostCurrency))
		case 4:
			ui.Text(c, holdingAmount(item.CostBasis, item.CostCurrency))
		case 5:
			key := "—"
			if item.CostDate != nil {
				key = *item.CostDate
			}
			if item.CostLabel != nil {
				key += " · " + *item.CostLabel
			}
			ui.Text(c, key).SingleLine()
		case 6:
			ui.Text(c, item.Origin)
		case 7:
			ui.Text(c, sourceLabel(item.Source)).FontSize(11).SingleLine()
		}
	}).Grow(1)
	if rows.Changed() && m.holdingSelection >= 0 && m.holdingSelection < len(lots) {
		m.holdingSelectedID = lots[m.holdingSelection].ID
	}
	selected = selectedHoldingIndex(lots, m.holdingSelectedID, func(item report.HoldingLot) string { return item.ID })
	if selected >= 0 {
		source := lots[selected].Source
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, m.tr("批次来源 · ")+sourceLabel(source)).TextColor(t.TextMuted).Grow(1)
			if source.Path != "" && ui.Button(c, m.tr("显示批次来源")).Clicked() {
				m.revealLocation(source.Path)
			}
			m.sourceEditorButton(c, source.Path)
		})
	}
	m.holdingPageControls(c, page)
}

func (m *Model) holdingPageControls(c *ui.Context, page report.ProjectionPage) {
	ui.Row(c).Gap(8).Children(func() {
		if ui.Button(c, m.tr("上一页")).Disabled(page.Offset == 0 || m.workspaceState.Busy).Clicked() {
			m.holdingOffset = max(0, page.Offset-page.Limit)
			m.holdingSelection, m.holdingSelectedID = -1, ""
			m.requestView("holdings")
		}
		ui.Spacer(c)
		if ui.Button(c, m.tr("下一页")).Disabled(!page.HasMore || m.workspaceState.Busy).Clicked() {
			m.holdingOffset = page.Offset + page.Limit
			m.holdingSelection, m.holdingSelectedID = -1, ""
			m.requestView("holdings")
		}
	})
}

func (m *Model) holdingResponseMatchesFilters(response *desktop.ViewResponse) bool {
	r := response.Request
	sortOrder := "ascending"
	if m.holdingDescending {
		sortOrder = "descending"
	}
	if r.View != "holdings" || r.HoldingSection != m.holdingSection || r.Account != m.holdingAccount || r.Scope != m.holdingScope || r.GroupBy != m.holdingGroupBy || r.Sort != sortOrder || r.Offset != m.holdingOffset || r.Limit != 100 || r.DateFrom != m.filterDateFrom || r.DateTo != m.filterDateTo || r.Window != m.filterWindow || r.Currency != m.filterCurrency || r.Valuation != m.filterValuation {
		return false
	}
	groupID, eventID, lotKind := "", "", ""
	if m.holdingSection == "history" {
		groupID = m.holdingGroupID
	}
	if m.holdingSection == "lots" {
		groupID, eventID = m.holdingGroupID, m.holdingEventID
		if eventID != "" {
			lotKind = m.holdingLotKind
		}
	}
	if r.GroupID != groupID || r.EventID != eventID || r.LotKind != lotKind {
		return false
	}
	return (r.SnapshotID == "" || r.SnapshotID == response.State.SnapshotID) &&
		(m.workspaceState.Ledger.SnapshotID == "" || r.SnapshotID == m.workspaceState.Ledger.SnapshotID)
}

func selectedHoldingIndex[T any](items []T, id string, identity func(T) string) int {
	for index, item := range items {
		if identity(item) == id {
			return index
		}
	}
	return -1
}

func holdingKindLabel(kind string) string {
	switch kind {
	case "long":
		return "多头"
	case "short":
		return "空头"
	case "mixed":
		return "多空混合"
	case "plain":
		return "无成本库存"
	case "sale":
		return "卖出"
	case "transfer-or-exchange":
		return "转移/兑换"
	case "short-or-mixed":
		return "空头/混合减仓"
	case "merged":
		return "合并前来源"
	case "consumed":
		return "消耗批次"
	default:
		return kind
	}
}

func holdingReasonLabel(reason string) string {
	switch reason {
	case "missing-price", "missing-proceeds-quote", "missing-fee-or-settlement-quote":
		return "缺少本地报价"
	case "missing-cost", "missing-cost-quote":
		return "成本资料不完整"
	case "mixed-cost-currencies":
		return "成本币种不同，不能合并计算"
	case "missing-sale-price":
		return "没有卖出价格"
	case "transfer-or-exchange":
		return "转移或兑换，不按卖出损益计算"
	case "short-position-opening":
		return "该交易包含新增空头"
	case "negative-sale-price":
		return "卖出价格为负数"
	case "unattributed-expenses":
		return "费用没有明确关联到该账户"
	case "proceeds-not-reconciled":
		return "现金收入与费用未能核对"
	case "ambiguous-trade-fees":
		return "一笔交易包含多个减仓，无法归属费用"
	default:
		return reason
	}
}

func holdingAmount(value *report.PresentedDecimal, currency *string) string {
	if value == nil {
		return "—"
	}
	code := ""
	if currency != nil && *currency != "" {
		code = *currency
	}
	return widgets.ExactAmountText(*value, code)
}

func (m *Model) holdingValuationText(value *report.AccountValuation) string {
	if value == nil {
		return m.tr("未知")
	}
	if value.CompleteTotal != nil {
		return holdingAmount(value.CompleteTotal, &value.TargetCurrency)
	}
	if value.ValuedPortion != nil {
		text := m.tr("部分 ") + holdingAmount(value.ValuedPortion, &value.TargetCurrency) + m.tr(" · 缺 ") + fmt.Sprint(value.MissingItemCount)
		if len(value.Missing) > 0 {
			text += m.tr(" · ") + m.tr(valuationReasonText(value.Missing[0].Reason))
		}
		return text
	}
	text := m.tr("不可用 · ") + m.tr(valuationCompletenessText(value.Completeness))
	if len(value.Missing) > 0 {
		text += m.tr(" · ") + m.tr(valuationReasonText(value.Missing[0].Reason))
	}
	return text
}

func holdingGroupMarketCurrency(group report.HoldingGroup) *string {
	if group.MarketValuation == nil || group.MarketValuation.TargetCurrency == "" {
		return nil
	}
	currency := group.MarketValuation.TargetCurrency
	return &currency
}

func eventGainText(event report.ReductionEvent, translate func(string) string) string {
	if event.NetGain != nil && event.CostCurrency != nil {
		return event.NetGain.Exact + " " + *event.CostCurrency
	}
	if event.GrossGain != nil && event.CostCurrency != nil {
		return event.GrossGain.Exact + " " + *event.CostCurrency + translate(" · 净值") + translate(holdingReasonLabel(event.NetReason))
	}
	return translate(holdingReasonLabel(event.GrossReason))
}

func (m *Model) pageDescription(page report.ProjectionPage, suffix string) string {
	start, end := 0, 0
	if page.ReturnedCount > 0 {
		start, end = page.Offset+1, page.Offset+page.ReturnedCount
	}
	return fmt.Sprintf(m.tr("%d%s · 显示 %d–%d"), page.TotalCount, m.tr(suffix), start, end)
}

func hasAverageHistory(items []report.AverageCostSeries, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func renderHoldingQuotes(c *ui.Context, m *Model, payload *report.HoldingsProjection, value *report.AccountValuation) {
	if value == nil {
		return
	}
	for _, quote := range value.Quotes {
		path := payload.SourceFiles[quote.Source.File]
		source := report.ProjectionSource{Path: path, Line: quote.Source.StartLine, Column: quote.Source.StartColumn}
		ui.Row(c).Gap(8).Children(func() {
			kind := m.tr("直接报价")
			if quote.Inverse {
				kind = m.tr("反向报价")
			}
			ui.Text(c, m.tr("市价来源 · ")+quote.Date+" "+quote.Rate.Exact+" "+quote.Quote+" · "+kind+" · "+sourceLabel(source)).TextColor(c.Theme().TextMuted).Grow(1)
			if path != "" && ui.Button(c, m.tr("显示估值报价来源")).Clicked() {
				m.revealLocation(path)
			}
			m.sourceEditorButton(c, path)
		})
	}
}
