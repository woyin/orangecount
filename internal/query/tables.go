// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package query

import (
	"math/big"
	"orangecount/internal/ledger"
	"path/filepath"
	"reflect"
	"strings"
)

// Table columns are bound before reading any rows. These scalar schemas are
// separate from the legacy report adapter's presentation columns.
type sourceColumn struct {
	kind     string
	nullable bool
}

var scalarTableSchemas = map[string]map[string]sourceColumn{
	"entries":      {"type": {"string", false}, "date": {"date", false}, "year": {"integer", false}, "month": {"integer", false}, "day": {"integer", false}, "filename": {"string", false}, "lineno": {"integer", false}, "flag": {"string", true}, "payee": {"string", true}, "narration": {"string", true}, "description": {"string", true}},
	"transactions": {"date": {"date", false}, "flag": {"string", false}, "payee": {"string", true}, "narration": {"string", false}},
	"accounts":     {"account": {"string", false}},
	"commodities":  {"date": {"date", false}, "name": {"string", false}},
	"balances":     {"date": {"date", false}, "account": {"string", false}, "amount": {"amount", false}, "tolerance": {"decimal", true}, "discrepancy": {"amount", true}},
	"notes":        {"date": {"date", false}, "account": {"string", false}, "comment": {"string", false}},
	"events":       {"date": {"date", false}, "type": {"string", false}, "description": {"string", false}},
	"documents":    {"date": {"date", false}, "account": {"string", false}, "filename": {"string", false}},
	"prices":       {"date": {"date", false}, "currency": {"string", false}, "amount": {"amount", false}},
}

func sourceAmount(value ledger.Amount) amountValue {
	return amountValue{Number: ledger.DecimalFromNumber(value.Number), Currency: value.Currency, Text: metadataDecimalValue(value.Number).Text}
}

func standardRows(table string, evaluation ledger.Evaluation, budget *executionBudget) ([]Row, error) {
	if table == "postings" {
		rows, err := postingRowsWithBudget(evaluation, true, budget)
		for _, row := range rows {
			row["type"] = "transaction"
			row["description"] = transactionDescription(row["payee"], row["narration"])
			if row["payee"] == "" && row["_payee_present"] != true {
				row["payee"] = nil
			}
			if row["posting_flag"] == "" {
				row["posting_flag"] = nil
			}
			if row["filename"] != nil {
				row["location"] = row["filename"].(string) + ":" + row["lineno"].(ledger.Decimal).String() + ":"
			}
		}
		return rows, err
	}
	rows := make([]Row, 0)
	accounts := map[string]bool{}
	for _, entry := range evaluation.Entries {
		if err := budget.step("source"); err != nil {
			return nil, err
		}
		date := entry.Date
		if !date.Valid() {
			continue
		}
		directive := entry.Directive
		// Parsed and generated core directives may be pointers or values.
		v := reflect.ValueOf(directive)
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				continue
			}
			directive = v.Elem().Interface().(ledger.Directive)
		}
		row := Row{"date": date.Raw, "_query_execution": budget}
		if table == "entries" {
			row["type"] = string(directive.Kind())
			row["year"] = datePartDecimal(date, "year")
			row["month"] = datePartDecimal(date, "month")
			row["day"] = datePartDecimal(date, "day")
			row["filename"] = entry.File
			row["lineno"] = ledger.NewDecimal(new(big.Rat).SetInt64(int64(entry.Span.StartLine)))
			if tx, ok := directive.(ledger.Transaction); ok {
				transactionColumns(row, tx)
			}
			rows = append(rows, row)
			continue
		}
		switch d := directive.(type) {
		case ledger.Transaction:
			if table != "transactions" {
				continue
			}
			transactionColumns(row, d)
		case ledger.Open:
			if table != "accounts" || accounts[d.Account] {
				continue
			}
			accounts[d.Account] = true
			row["account"] = d.Account
		case ledger.Close:
			if table != "accounts" || accounts[d.Account] {
				continue
			}
			accounts[d.Account] = true
			row["account"] = d.Account
		case ledger.Commodity:
			if table != "commodities" {
				continue
			}
			row["name"] = d.Currency
		case ledger.Balance:
			if table != "balances" {
				continue
			}
			row["account"] = d.Account
			row["amount"] = sourceAmount(d.Amount)
			if d.Tolerance != nil {
				row["tolerance"] = ledger.DecimalFromNumber(*d.Tolerance)
			}
		case ledger.Note:
			if table != "notes" {
				continue
			}
			row["account"] = d.Account
			row["comment"] = d.Comment
		case ledger.Event:
			if table != "events" {
				continue
			}
			row["type"] = d.Type
			row["description"] = d.Value
		case ledger.Document:
			if table != "documents" {
				continue
			}
			row["account"] = d.Account
			if len(d.Filenames) > 0 {
				name := d.Filenames[0]
				if !filepath.IsAbs(name) {
					name = filepath.Join(filepath.Dir(entry.File), name)
				}
				row["filename"] = filepath.Clean(name)
			}
		case ledger.Price:
			if table != "prices" {
				continue
			}
			row["currency"] = d.Currency
			row["amount"] = sourceAmount(d.Amount)
		default:
			continue
		}
		rows = append(rows, row)
	}
	return rows, nil
}
func transactionColumns(row Row, tx ledger.Transaction) {
	row["flag"] = tx.Flag
	row["narration"] = tx.Narration
	if tx.Payee != "" || tx.HasPayee {
		row["payee"] = tx.Payee
	}
	row["description"] = transactionDescription(tx.Payee, tx.Narration)
}
func transactionDescription(payee, narration any) string {
	parts := []string{}
	for _, v := range []any{payee, narration} {
		if text, ok := v.(string); ok && text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, " | ")
}
