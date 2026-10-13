// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import "orangecount/apps/goui/internal/workspace"

const maxPageHistory = 64

func pageForView(view string) (page, bool) {
	switch view {
	case "dashboard":
		return pageDashboard, true
	case "balance_sheet":
		return pageBalance, true
	case "income_statement":
		return pageIncome, true
	case "journal":
		return pageJournal, true
	case "account":
		return pageAccount, true
	case "prices":
		return pagePrices, true
	case "holdings":
		return pageHoldings, true
	case "directives":
		return pageDirectives, true
	case "migration":
		return pageSemantics, true
	case "query":
		return pageQuery, true
	case "attachments":
		return pageAttachments, true
	default:
		return pageWelcome, false
	}
}

func viewForPage(target page) (string, bool) {
	switch target {
	case pageDashboard:
		return "dashboard", true
	case pageBalance:
		return "balance_sheet", true
	case pageIncome:
		return "income_statement", true
	case pageJournal:
		return "journal", true
	case pageAccount:
		return "account", true
	case pagePrices:
		return "prices", true
	case pageHoldings:
		return "holdings", true
	case pageDirectives:
		return "directives", true
	case pageSemantics:
		return "migration", true
	case pageQuery:
		return "query", true
	case pageAttachments:
		return "attachments", true
	default:
		return "", false
	}
}

func (m *Model) viewForNavigationPage(target page) (string, bool) {
	if target == pageQuery && (m.offlineAnalysis != nil || m.offlineAnalysisLoading || m.offlineAnalysisError != "") {
		return "", false
	}
	return viewForPage(target)
}

func pushPage(history []page, target page) []page {
	if len(history) > 0 && history[len(history)-1] == target {
		return history
	}
	history = append(history, target)
	if len(history) > maxPageHistory {
		copy(history, history[len(history)-maxPageHistory:])
		history = history[:maxPageHistory]
	}
	return history
}

func (m *Model) navigateToPage(target page, view string, remember bool) bool {
	if view == "directives" && len(m.directiveSearch) > 65536 {
		m.directiveError = "搜索文字超过 64 KiB 上限。"
		return false
	}
	if view == "journal" && len(m.journalSearch) > 65536 {
		m.journalError = "流水搜索文字超过 64 KiB 上限。"
		return false
	}

	previous := m.page
	if previous != target {
		if remember && (view == "" || m.workspaceState.ActivePath != "") {
			m.pageHistoryBack = pushPage(m.pageHistoryBack, previous)
			m.pageHistoryForward = nil
		}
		m.searchFocusPending = false
		m.cancelPendingExport()
	}
	m.page = target

	if view == "" {
		m.status = ""
		return true
	}
	if m.workspace == nil || m.workspaceState.ActivePath == "" {
		m.status = "请先打开账本"
		return true
	}
	if err := m.workspace.Dispatch(workspace.Intent{Kind: workspace.IntentRead, Request: m.requestForView(view)}); err != nil {
		m.status = "无法读取页面：" + err.Error()
		return true
	}
	if view == "directives" {
		m.directiveError = ""
	}
	m.dashboardRequestOperation = m.workspace.State().Operation
	m.status = ""
	return true
}

func (m *Model) navigateBack() {
	if len(m.pageHistoryBack) == 0 {
		return
	}
	last := len(m.pageHistoryBack) - 1
	target := m.pageHistoryBack[last]
	view, hasView := m.viewForNavigationPage(target)
	if !hasView {
		view = ""
	}
	current := m.page
	if !m.navigateToPage(target, view, false) {
		return
	}
	m.pageHistoryBack = m.pageHistoryBack[:last]
	m.pageHistoryForward = pushPage(m.pageHistoryForward, current)
}

func (m *Model) navigateForward() {
	if len(m.pageHistoryForward) == 0 {
		return
	}
	last := len(m.pageHistoryForward) - 1
	target := m.pageHistoryForward[last]
	view, hasView := m.viewForNavigationPage(target)
	if !hasView {
		view = ""
	}
	current := m.page
	if !m.navigateToPage(target, view, false) {
		return
	}
	m.pageHistoryForward = m.pageHistoryForward[:last]
	m.pageHistoryBack = pushPage(m.pageHistoryBack, current)
}

func (m *Model) resetPageHistory() {
	m.pageHistoryBack = nil
	m.pageHistoryForward = nil
}
