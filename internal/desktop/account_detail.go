// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package desktop

import (
	"orangecount/internal/ledger"
	"orangecount/internal/report"
)

type AccountLifecycleSource struct {
	Kind   string         `json:"kind"`
	Date   string         `json:"date"`
	Source SourceLocation `json:"source"`
}

type AccountDetailPayload struct {
	*report.AccountDetailProjection
	Sources []AccountLifecycleSource `json:"sources"`
}

func projectAccount(e ledger.Evaluation, r Request, from, to string) (*AccountDetailPayload, error) {
	projection, err := report.AccountDetailView(e, r.Account, r.Scope, from, to, r.Currency, r.Valuation)
	if err != nil {
		return nil, err
	}
	result := &AccountDetailPayload{AccountDetailProjection: projection, Sources: []AccountLifecycleSource{}}
	for _, entry := range e.Entries {
		account := ""
		switch d := entry.Directive.(type) {
		case ledger.Open:
			account = d.Account
		case *ledger.Open:
			account = d.Account
		case ledger.Close:
			account = d.Account
		case *ledger.Close:
			account = d.Account
		}
		if account == r.Account {
			result.Sources = append(result.Sources, AccountLifecycleSource{Kind: string(entry.Directive.Kind()), Date: entry.Date.Raw, Source: location(entry.File, entry.Span)})
		}
	}
	return result, nil
}
