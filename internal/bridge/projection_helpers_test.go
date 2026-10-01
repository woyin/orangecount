// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"orangecount/internal/desktop"
	"orangecount/internal/diagnostic"
	"orangecount/internal/ledger"
	"orangecount/internal/report"
	"orangecount/internal/snapshot"
)

type SummaryPayload = desktop.SummaryPayload
type JournalPostingDTO = desktop.JournalPostingDTO
type JournalTransactionDTO = desktop.JournalTransactionDTO
type JournalPayload = desktop.JournalPayload
type TreeReportPayload = desktop.TreeReportPayload
type TrendPointDTO = desktop.TrendPointDTO
type CashFlowBarDTO = desktop.CashFlowBarDTO
type HistoricalTrendsPayload = desktop.HistoricalTrendsPayload

func loadSnapshot(path string) SummaryPayload {
	res := snapshot.Build(path)
	var evaluation *ledger.Evaluation
	if res.Snapshot != nil {
		value := res.Snapshot.Evaluation()
		evaluation = &value
	}
	return desktop.ProjectSummary(path, evaluation, desktop.ProjectDiagnostics(res.Diagnostics, res.Graph, "zh-CN"), res.Snapshot != nil)
}

func loadJournal(path, filterJSON string) JournalPayload {
	res := snapshot.Build(path)
	if res.Snapshot == nil {
		return JournalPayload{Transactions: []JournalTransactionDTO{}, Error: firstErrorMessage(res.Diagnostics)}
	}
	return desktop.ProjectJournal(res.Snapshot.Evaluation())
}

func loadTreeReport(path, reportType string) TreeReportPayload {
	res := snapshot.Build(path)
	if res.Snapshot == nil {
		return TreeReportPayload{ReportType: reportType, OperatingCurrencies: []string{}, RootNodes: []report.AccountTreeNode{}, Error: firstErrorMessage(res.Diagnostics)}
	}
	return desktop.ProjectTree(res.Snapshot.Evaluation(), reportType)
}

func loadHistoricalTrends(path string) HistoricalTrendsPayload {
	res := snapshot.Build(path)
	if res.Snapshot == nil {
		return HistoricalTrendsPayload{NetWorthAvailability: "invalid-ledger", CashFlowAvailability: "invalid-ledger", NetWorthPoints: []TrendPointDTO{}, MonthlyCashFlows: []CashFlowBarDTO{}, Error: firstErrorMessage(res.Diagnostics)}
	}
	return desktop.ProjectTrends(res.Snapshot.Evaluation(), "")
}

func firstErrorMessage(diags []diagnostic.Diagnostic) string {
	for _, d := range diags {
		if d.Severity == diagnostic.Error {
			if d.Span.StartLine > 0 {
				return fmt.Sprintf("line %d: %s", d.Span.StartLine, d.Message)
			}
			return d.Message
		}
	}
	return "ledger snapshot unavailable"
}
