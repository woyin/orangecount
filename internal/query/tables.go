// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package query

import (
	"math/big"
	"orangecount/internal/ledger"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// Table columns are bound before reading any rows. These standard schemas are
// separate from the legacy report adapter's presentation columns.
type sourceColumn struct {
	kind     string
	nullable bool
}

var standardTableSchemas = map[string]map[string]sourceColumn{
	"entries":      {"id": {"string", false}, "type": {"string", false}, "date": {"date", false}, "year": {"integer", false}, "month": {"integer", false}, "day": {"integer", false}, "filename": {"string", false}, "lineno": {"integer", false}, "flag": {"string", true}, "payee": {"string", true}, "narration": {"string", true}, "description": {"string", true}},
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
			tx, _ := row["_transaction"].(ledger.Transaction)
			record, _ := row["_entry_record"].(ledger.EntryRecord)
			// postings.id is the parent transaction identifier, mirroring
			// upstream PostingsTable.id delegating to hash_entry(entry).
			id, idErr := directiveID(record, sourceRoot(evaluation))
			if idErr != nil {
				return nil, idErr
			}
			row["id"] = id
			scope, _ := row["_posting_meta"].(queryMetadata)
			metadata, metaErr := metadataDictionary(scope)
			if metaErr != nil {
				return nil, metaErr
			}
			row["meta"] = metadata
			row["tags"] = stringSet(tx.Tags)
			row["links"] = stringSet(tx.Links)
			row["accounts"] = transactionAccounts(tx)
			others := stringSet{}
			for index, posting := range tx.Postings {
				if index != row["_posting_index"] {
					others = append(others, posting.Account)
				}
			}
			sort.Strings(others)
			unique := listValue{}
			previous := ""
			for i, account := range others {
				if i == 0 || account != previous {
					unique = append(unique, account)
				}
				previous = account
			}
			row["other_accounts"] = unique
			row["entry"], metaErr = transactionStructure(tx, record)
			if metaErr != nil {
				return nil, metaErr
			}
			position, _ := row["position"].(positionValue)
			row["cost_label"] = ""
			if cost := position.Cost; cost != nil {
				row["cost_number"] = cost.Number
				row["cost_currency"] = cost.Currency
				row["cost_label"] = nil
				if cost.Date != nil {
					row["cost_date"] = *cost.Date
				}
				if cost.Label != nil {
					row["cost_label"] = *cost.Label
				}
			}

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
	accountsLifecycle := map[string][2]any{}
	if table == "accounts" {
		var err error
		accountsLifecycle, err = accountStructures(evaluation, budget)
		if err != nil {
			return nil, err
		}
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
		if table != "entries" {
			expected := map[string]ledger.DirectiveKind{"transactions": ledger.KindTxn, "commodities": ledger.KindCommodity, "balances": ledger.KindBalance, "notes": ledger.KindNote, "events": ledger.KindEvent, "documents": ledger.KindDocument, "prices": ledger.KindPrice}
			if table == "accounts" {
				if directive.Kind() != ledger.KindOpen && directive.Kind() != ledger.KindClose {
					continue
				}
			} else if directive.Kind() != expected[table] {
				continue
			}
		}
		row := Row{"date": date.Raw, "_query_execution": budget}
		if table == "entries" {
			// The id column is the directive's stable identity, computed the
			// way beancount.core.compare.hash_entry does.
			id, idErr := directiveID(entry, sourceRoot(evaluation))
			if idErr != nil {
				return nil, idErr
			}
			row["id"] = id
		}
		scope := directiveMetadata(directive, entry)
		metadata, metaErr := metadataDictionary(scope)
		if metaErr != nil {
			return nil, metaErr
		}
		row["meta"] = metadata
		row["_posting_meta"] = scope

		if table == "entries" {
			row["type"] = string(directive.Kind())
			row["year"] = datePartDecimal(date, "year")
			row["month"] = datePartDecimal(date, "month")
			row["day"] = datePartDecimal(date, "day")
			row["filename"] = entry.File
			row["lineno"] = ledger.NewDecimal(new(big.Rat).SetInt64(int64(entry.Span.StartLine)))
			row["accounts"] = directiveAccounts(directive)
			if tx, ok := directive.(ledger.Transaction); ok {
				transactionColumns(row, tx)
			}
			if note, ok := directive.(ledger.Note); ok {
				row["tags"] = stringSet(note.Tags)
				row["links"] = stringSet(note.Links)
			}
			if doc, ok := directive.(ledger.Document); ok {
				row["tags"] = stringSet(doc.Tags)
				row["links"] = stringSet(doc.Links)
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
			row["open"], row["close"] = accountsLifecycle[d.Account][0], accountsLifecycle[d.Account][1]
		case ledger.Close:
			if table != "accounts" || accounts[d.Account] {
				continue
			}
			accounts[d.Account] = true
			row["account"] = d.Account
			row["open"], row["close"] = accountsLifecycle[d.Account][0], accountsLifecycle[d.Account][1]
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
			// BeanQuery reports the assertion difference it observed; a holding
			// assertion keeps the column NULL.
			if d.DiffAmount != nil {
				row["discrepancy"] = sourceAmount(*d.DiffAmount)
			}
		case ledger.Note:
			if table != "notes" {
				continue
			}
			row["account"] = d.Account
			row["comment"] = d.Comment
			row["tags"] = stringSet(d.Tags)
			row["links"] = stringSet(d.Links)
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
			row["tags"] = stringSet(d.Tags)
			row["links"] = stringSet(d.Links)
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
	row["tags"] = stringSet(tx.Tags)
	row["links"] = stringSet(tx.Links)
	row["accounts"] = transactionAccounts(tx)
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

func directiveMetadata(d ledger.Directive, entry ledger.EntryRecord) queryMetadata {
	v := reflect.ValueOf(d)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	var values []ledger.Metadata
	if field := v.FieldByName("Meta"); field.IsValid() {
		values, _ = field.Interface().([]ledger.Metadata)
	}
	scope := queryMetadata{Values: values, Filename: entry.File, Line: entry.Span.StartLine}
	if tx, ok := d.(ledger.Transaction); ok {
		scope.HasTolerances = tx.Tolerances != nil
		scope.Tolerances = tx.Tolerances
	}
	return scope
}
func transactionAccounts(tx ledger.Transaction) stringSet {
	accounts := stringSet{}
	for _, p := range tx.Postings {
		accounts = append(accounts, p.Account)
	}
	return accounts
}
func directiveAccounts(d ledger.Directive) stringSet {
	if tx, ok := d.(ledger.Transaction); ok {
		return transactionAccounts(tx)
	}
	if pad, ok := d.(ledger.Pad); ok {
		return stringSet{pad.Account, pad.SourceAccount}
	}
	v := reflect.ValueOf(d)
	accounts := stringSet{}
	if field := v.FieldByName("Account"); field.IsValid() {
		accounts = append(accounts, field.String())
	}
	if custom, ok := d.(ledger.Custom); ok {
		for _, value := range custom.Values {
			if value.Kind == ledger.ValueAccount {
				accounts = append(accounts, value.String)
			}
		}
	}
	return accounts
}
func transactionStructure(tx ledger.Transaction, entry ledger.EntryRecord) (recordValue, error) {
	metadata, err := metadataDictionary(directiveMetadata(tx, entry))
	if err != nil {
		return recordValue{}, err
	}
	fields := dictionaryValue{"meta": metadata, "date": tx.Date, "flag": tx.Flag, "payee": nil, "narration": tx.Narration, "tags": stringSet(tx.Tags), "links": stringSet(tx.Links), "accounts": transactionAccounts(tx)}
	if tx.HasPayee || tx.Payee != "" {
		fields["payee"] = tx.Payee
	}
	return recordValue{kind: "transaction", fields: fields}, nil
}
func accountStructures(e ledger.Evaluation, budget *executionBudget) (map[string][2]any, error) {
	result := map[string][2]any{}
	for _, entry := range e.Entries {
		if err := budget.step("source"); err != nil {
			return nil, err
		}
		d := entry.Directive
		v := reflect.ValueOf(d)
		if v.Kind() == reflect.Pointer {
			d = v.Elem().Interface().(ledger.Directive)
		}
		switch value := d.(type) {
		case ledger.Open:
			metadata, err := metadataDictionary(directiveMetadata(d, entry))
			if err != nil {
				return nil, err
			}
			fields := dictionaryValue{"meta": metadata, "date": value.Date, "account": value.Account, "currencies": nil, "booking": nil}
			if value.Currencies != nil {
				list := listValue{}
				for _, c := range value.Currencies {
					list = append(list, c)
				}
				fields["currencies"] = list
			}
			if value.Booking != "" {
				fields["booking"] = value.Booking
			}
			pair := result[value.Account]
			pair[0] = recordValue{kind: "open", fields: fields}
			result[value.Account] = pair
		case ledger.Close:
			metadata, err := metadataDictionary(directiveMetadata(d, entry))
			if err != nil {
				return nil, err
			}
			pair := result[value.Account]
			pair[1] = recordValue{kind: "close", fields: dictionaryValue{"meta": metadata, "date": value.Date, "account": value.Account}}
			result[value.Account] = pair
		}
	}
	return result, nil
}
func init() {
	for table, columns := range standardTableSchemas {
		if table != "accounts" {
			columns["meta"] = sourceColumn{"dictionary", false}
		}
		switch table {
		case "transactions", "notes", "documents":
			columns["tags"] = sourceColumn{"set<string>", false}
			columns["links"] = sourceColumn{"set<string>", false}
		case "entries":
			columns["tags"] = sourceColumn{"set<string>", true}
			columns["links"] = sourceColumn{"set<string>", true}
		case "accounts":
			columns["open"] = sourceColumn{"open", true}
			columns["close"] = sourceColumn{"close", true}
		}
		if table == "transactions" || table == "entries" {
			columns["accounts"] = sourceColumn{"set<string>", false}
		}
	}
}

// Wildcards use upstream's declared order, independent of whether rows exist.
// meta is omitted by ordinary directive tables, included by EntriesTable.
func expandStandardWildcard(q *Query, table string) {
	schemas := map[string][]string{
		"postings":     {"id", "date", "flag", "payee", "narration", "position"},
		"entries":      {"id", "type", "filename", "lineno", "date", "year", "month", "day", "flag", "payee", "narration", "description", "tags", "links", "meta", "accounts"},
		"transactions": {"date", "flag", "payee", "narration", "tags", "links", "accounts"},
		"accounts":     {"account", "open", "close"},
		"commodities":  {"meta", "date", "name"},
		"balances":     {"date", "account", "amount", "tolerance", "discrepancy"},
		"notes":        {"date", "account", "comment", "tags", "links"},
		"events":       {"date", "type", "description"},
		"documents":    {"date", "account", "filename", "tags", "links"},
		"prices":       {"date", "currency", "amount"},
	}
	selectItems := []SelectItem{}
	for _, item := range q.Select {
		if _, ok := item.Expr.(starExpr); !ok {
			selectItems = append(selectItems, item)
			continue
		}
		for _, column := range schemas[table] {
			selectItems = append(selectItems, SelectItem{Expr: fieldExpr{name: column}, sourceTitle: column})
		}
	}
	q.Select = selectItems
}

// sourceRoot is the directory directive ids normalize source filenames
// against. Keeping it on the evaluation makes the id independent of the
// checkout location while still derived from the ledger's own layout.
func sourceRoot(evaluation ledger.Evaluation) string {
	if evaluation.EntryPath == "" {
		return ""
	}
	return filepath.Dir(evaluation.EntryPath)
}
