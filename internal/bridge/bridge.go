// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

/*
#include <stdlib.h>
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"unsafe"

	"orangecount/internal/diagnostic"
	"orangecount/internal/ledger"
	"orangecount/internal/report"
	"orangecount/internal/snapshot"
)

// SummaryPayload is returned to Swift as pure memory JSON.
type SummaryPayload struct {
	Valid        bool     `json:"valid"`
	Title        string   `json:"title"`
	OperatingCur []string `json:"operating_currencies"`
	Accounts     []string `json:"accounts"`
	EntriesCount int      `json:"entries_count"`
	ErrorCount   int      `json:"error_count"`
	Errors       []string `json:"errors"`
}

//export OC_Ping
func OC_Ping() *C.char {
	return C.CString("OrangeCount Core (Go) is connected via C-Bridge!")
}

//export OC_LoadSnapshot
func OC_LoadSnapshot(cPath *C.char) *C.char {
	if cPath == nil {
		return errorJSON("ledger path is nil")
	}
	path := C.GoString(cPath)
	payload := loadSnapshot(path)
	encoded, err := json.Marshal(payload)
	if err != nil {
		return errorJSON(err.Error())
	}
	return C.CString(string(encoded))
}

func loadSnapshot(path string) SummaryPayload {
	res := snapshot.Build(path)

	payload := SummaryPayload{
		Valid:        res.Snapshot != nil && res.Snapshot.Valid(),
		Accounts:     []string{},
		OperatingCur: []string{},
		Errors:       []string{},
	}

	for _, d := range res.Diagnostics {
		if d.Severity == diagnostic.Error {
			payload.ErrorCount++
			payload.Errors = append(payload.Errors, d.Message)
		}
	}

	if res.Snapshot != nil {
		eval := res.Snapshot.Evaluation()
		payload.EntriesCount = len(eval.Entries)
		if title, ok := eval.Options["title"]; ok {
			payload.Title = title
		}
		if opCur, ok := eval.Options["operating_currency"]; ok {
			payload.OperatingCur = append(payload.OperatingCur, opCur)
		}

		for acct := range eval.Accounts {
			payload.Accounts = append(payload.Accounts, acct)
		}
		sort.Strings(payload.Accounts)
	}

	return payload
}

// JournalPostingDTO represents one posting inside a transaction for desktop UI
type JournalPostingDTO struct {
	Account       string `json:"account"`
	UnitsNumber   string `json:"units_number"`
	UnitsCurrency string `json:"units_currency"`
	Cost          string `json:"cost,omitempty"`
	Price         string `json:"price,omitempty"`
}

// JournalTransactionDTO represents one complete transaction entry
type JournalTransactionDTO struct {
	ID        string              `json:"id"`
	Date      string              `json:"date"`
	Flag      string              `json:"flag"`
	Payee     string              `json:"payee"`
	Narration string              `json:"narration"`
	Tags      []string            `json:"tags"`
	Links     []string            `json:"links"`
	Postings  []JournalPostingDTO `json:"postings"`
}

// JournalPayload is returned by OC_GetJournal
type JournalPayload struct {
	TotalCount   int                     `json:"total_count"`
	Transactions []JournalTransactionDTO `json:"transactions"`
	Error        string                  `json:"error,omitempty"`
}

//export OC_GetJournal
func OC_GetJournal(cPath *C.char, cFilterJSON *C.char) *C.char {
	if cPath == nil {
		return errorJSON("ledger path is nil")
	}
	path := C.GoString(cPath)
	filterJSON := ""
	if cFilterJSON != nil {
		filterJSON = C.GoString(cFilterJSON)
	}

	payload := loadJournal(path, filterJSON)
	encoded, err := json.Marshal(payload)
	if err != nil {
		return errorJSON(err.Error())
	}
	return C.CString(string(encoded))
}

func loadJournal(path, filterJSON string) JournalPayload {
	res := snapshot.Build(path)
	if res.Snapshot == nil {
		return JournalPayload{
			TotalCount:   0,
			Transactions: []JournalTransactionDTO{},
			Error:        firstErrorMessage(res.Diagnostics),
		}
	}

	eval := res.Snapshot.Evaluation()
	var txs []JournalTransactionDTO

	for i, record := range eval.Entries {
		var tx *ledger.Transaction
		switch t := record.Directive.(type) {
		case *ledger.Transaction:
			tx = t
		case ledger.Transaction:
			tx = &t
		}
		if tx == nil {
			continue
		}

		postings := make([]JournalPostingDTO, 0, len(tx.Postings))
		for _, p := range tx.Postings {
			num := ""
			cur := ""
			if p.Units != nil {
				num = p.Units.Number.Raw
				cur = p.Units.Currency
			}
			postings = append(postings, JournalPostingDTO{
				Account:       p.Account,
				UnitsNumber:   num,
				UnitsCurrency: cur,
			})
		}

		tags := tx.Tags
		if tags == nil {
			tags = []string{}
		}
		links := tx.Links
		if links == nil {
			links = []string{}
		}

		txs = append(txs, JournalTransactionDTO{
			ID:        fmt.Sprintf("tx_%04d_%s", i+1, record.Date.Raw),
			Date:      record.Date.Raw,
			Flag:      tx.Flag,
			Payee:     tx.Payee,
			Narration: tx.Narration,
			Tags:      tags,
			Links:     links,
			Postings:  postings,
		})
	}

	_ = filterJSON // client-side swift filtering is sub-millisecond; keep raw list in payload

	return JournalPayload{
		TotalCount:   len(txs),
		Transactions: txs,
	}
}

type TreeReportPayload struct {
	ReportType          string                   `json:"report_type"`
	OperatingCurrencies []string                 `json:"operating_currencies"`
	RootNodes           []report.AccountTreeNode `json:"root_nodes"`
	Error               string                   `json:"error,omitempty"`
}

//export OC_GetTreeReport
func OC_GetTreeReport(cPath *C.char, cReportType *C.char) *C.char {
	if cPath == nil {
		return errorJSON("ledger path is nil")
	}
	path := C.GoString(cPath)
	reportType := "balance_sheet"
	if cReportType != nil {
		reportType = C.GoString(cReportType)
	}

	payload := loadTreeReport(path, reportType)
	encoded, err := json.Marshal(payload)
	if err != nil {
		return errorJSON(err.Error())
	}
	return C.CString(string(encoded))
}

func loadTreeReport(path, reportType string) TreeReportPayload {
	res := snapshot.Build(path)
	if res.Snapshot == nil {
		return TreeReportPayload{
			ReportType:          reportType,
			OperatingCurrencies: []string{},
			RootNodes:           []report.AccountTreeNode{},
			Error:               firstErrorMessage(res.Diagnostics),
		}
	}

	eval := res.Snapshot.Evaluation()
	var opCurs []string
	if opCur, ok := eval.Options["operating_currency"]; ok {
		opCurs = append(opCurs, opCur)
	}

	var roots []string
	if reportType == "income_statement" {
		roots = []string{"Income", "Expenses"}
	} else {
		reportType = "balance_sheet"
		roots = []string{"Assets", "Liabilities", "Equity"}
	}

	nodes := report.BuildAccountTree(eval, roots...)
	return TreeReportPayload{
		ReportType:          reportType,
		OperatingCurrencies: opCurs,
		RootNodes:           nodes,
	}
}

// TrendPointDTO is one date/value point for Swift Charts
type TrendPointDTO struct {
	Date     string  `json:"date"`
	Value    float64 `json:"value"`
	Currency string  `json:"currency"`
}

// CashFlowBarDTO is one month's income vs expenses for Swift Charts
type CashFlowBarDTO struct {
	Month    string  `json:"month"`
	Income   float64 `json:"income"`
	Expenses float64 `json:"expenses"`
	Net      float64 `json:"net"`
	Currency string  `json:"currency"`
}

// HistoricalTrendsPayload is returned by OC_GetHistoricalTrends
type HistoricalTrendsPayload struct {
	OperatingCurrency string           `json:"operating_currency"`
	NetWorthPoints    []TrendPointDTO  `json:"net_worth_points"`
	MonthlyCashFlows  []CashFlowBarDTO `json:"monthly_cash_flows"`
	Error             string           `json:"error,omitempty"`
}

//export OC_GetHistoricalTrends
func OC_GetHistoricalTrends(cPath *C.char) *C.char {
	if cPath == nil {
		return errorJSON("ledger path is nil")
	}
	path := C.GoString(cPath)
	payload := loadHistoricalTrends(path)
	encoded, err := json.Marshal(payload)
	if err != nil {
		return errorJSON(err.Error())
	}
	return C.CString(string(encoded))
}

func loadHistoricalTrends(path string) HistoricalTrendsPayload {
	res := snapshot.Build(path)
	if res.Snapshot == nil {
		return HistoricalTrendsPayload{
			OperatingCurrency: "USD",
			NetWorthPoints:    []TrendPointDTO{},
			MonthlyCashFlows:  []CashFlowBarDTO{},
			Error:             firstErrorMessage(res.Diagnostics),
		}
	}
	eval := res.Snapshot.Evaluation()
	cur := "USD"
	if opCur, ok := eval.Options["operating_currency"]; ok && opCur != "" {
		cur = opCur
	}

	bsChart := report.ReportChart(eval, "balance-sheet", "month", cur, "at-cost", "")
	isChart := report.ReportChart(eval, "income-statement", "month", cur, "at-cost", "")

	var netWorthPoints []TrendPointDTO
	for _, series := range bsChart.Series {
		if series.Label == "Net worth" || series.Label == cur || len(bsChart.Series) == 1 {
			for _, pt := range series.Points {
				val, _ := pt.Value.Rat().Float64()
				netWorthPoints = append(netWorthPoints, TrendPointDTO{
					Date:     pt.Date,
					Value:    val,
					Currency: cur,
				})
			}
			break
		}
	}

	monthIncome := map[string]float64{}
	monthExpense := map[string]float64{}
	var months []string

	for _, series := range isChart.Series {
		for _, pt := range series.Points {
			val, _ := pt.Value.Rat().Float64()
			if _, ok := monthIncome[pt.Date]; !ok {
				months = append(months, pt.Date)
			}
			if series.Label == "Income" {
				monthIncome[pt.Date] += val
			} else {
				monthExpense[pt.Date] += val
			}
		}
	}
	sort.Strings(months)

	var cashFlows []CashFlowBarDTO
	for _, m := range months {
		inc := monthIncome[m]
		exp := monthExpense[m]
		cashFlows = append(cashFlows, CashFlowBarDTO{
			Month:    m,
			Income:   inc,
			Expenses: exp,
			Net:      inc - exp,
			Currency: cur,
		})
	}

	return HistoricalTrendsPayload{
		OperatingCurrency: cur,
		NetWorthPoints:    netWorthPoints,
		MonthlyCashFlows:  cashFlows,
	}
}

//export OC_GetLedgerRevision
func OC_GetLedgerRevision(cPath *C.char) *C.char {
	if cPath == nil {
		return C.CString("rev:0")
	}
	path := C.GoString(cPath)
	return C.CString(getLedgerRevision(path))
}

func getLedgerRevision(path string) string {
	res := snapshot.Build(path)
	if res.Snapshot == nil {
		return "rev:invalid"
	}
	graph := res.Snapshot.Graph()
	if graph == nil {
		return "rev:1"
	}
	var totalMtime int64
	for _, f := range graph.Files {
		if fi, err := os.Stat(f.Path); err == nil {
			totalMtime += fi.ModTime().UnixNano() + fi.Size()
		}
	}
	return fmt.Sprintf("rev:%x:%d", totalMtime, len(res.Diagnostics))
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

//export OC_FreeString
func OC_FreeString(ptr *C.char) {
	if ptr != nil {
		C.free(unsafe.Pointer(ptr))
	}
}

func errorJSON(msg string) *C.char {
	data, _ := json.Marshal(map[string]any{"error": msg, "valid": false})
	return C.CString(string(data))
}

func main() {}
