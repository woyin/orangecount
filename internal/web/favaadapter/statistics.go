package favaadapter

import (
	"sort"
	"strings"

	"orangecount/internal/ledger"
)

// UpdateActivityRow mirrors one row of Fava's "Update activity" statistics
// table: the newest entry that touched an Assets/Liabilities account, plus
// the account's current balances.
type UpdateActivityRow struct {
	Account       string            `json:"account"`
	LastEntryDate string            `json:"last_entry_date"`
	EntryHash     string            `json:"entry_hash"`
	Balances      map[string]string `json:"balances"`
}

// UpdateActivity lists Assets/Liabilities accounts with their most recent
// entry, the counterpart of the account_details data Fava's UpdateActivity
// table consumes. Records are evaluated in journal order, so the last record
// touching an account wins.
func UpdateActivity(e ledger.Evaluation) []UpdateActivityRow {
	type lastEntry struct {
		date string
		hash string
	}
	last := map[string]lastEntry{}
	for _, record := range e.Entries {
		accounts := recordAccounts(record)
		if len(accounts) == 0 {
			continue
		}
		hash := ""
		for _, account := range accounts {
			if !strings.HasPrefix(account, "Assets") && !strings.HasPrefix(account, "Liabilities") {
				continue
			}
			if hash == "" {
				hash = entryHash(record)
			}
			last[account] = lastEntry{date: recordDate(record), hash: hash}
		}
	}
	names := make([]string, 0, len(last))
	for account := range last {
		names = append(names, account)
	}
	sort.Strings(names)
	rows := make([]UpdateActivityRow, 0, len(names))
	for _, account := range names {
		row := UpdateActivityRow{
			Account:       account,
			LastEntryDate: last[account].date,
			EntryHash:     last[account].hash,
			Balances:      map[string]string{},
		}
		if state, ok := e.Account(account); ok {
			currencies := make([]string, 0, len(state.Balances))
			for currency := range state.Balances {
				currencies = append(currencies, currency)
			}
			sort.Strings(currencies)
			for _, currency := range currencies {
				row.Balances[currency] = state.Balances[currency].String()
			}
		}
		rows = append(rows, row)
	}
	return rows
}

// recordAccounts returns every account one ledger entry touches, covering
// directive-level accounts and transaction postings alike.
func recordAccounts(record ledger.EntryRecord) []string {
	switch directive := record.Directive.(type) {
	case *ledger.Transaction:
		return transactionAccounts(directive)
	case ledger.Transaction:
		return transactionAccounts(&directive)
	case ledger.Balance:
		return []string{directive.Account}
	case ledger.Open:
		return []string{directive.Account}
	case ledger.Close:
		return []string{directive.Account}
	case ledger.Note:
		return []string{directive.Account}
	case ledger.Document:
		return []string{directive.Account}
	case ledger.Pad:
		return []string{directive.Account, directive.SourceAccount}
	default:
		return nil
	}
}

func transactionAccounts(transaction *ledger.Transaction) []string {
	accounts := make([]string, 0, len(transaction.Postings))
	for _, posting := range transaction.Postings {
		accounts = append(accounts, posting.Account)
	}
	return accounts
}

func recordDate(record ledger.EntryRecord) string {
	switch directive := record.Directive.(type) {
	case *ledger.Transaction:
		return directive.Date.Raw
	case ledger.Transaction:
		return directive.Date.Raw
	case ledger.Balance:
		return directive.Date.Raw
	case ledger.Open:
		return directive.Date.Raw
	case ledger.Close:
		return directive.Date.Raw
	case ledger.Note:
		return directive.Date.Raw
	case ledger.Document:
		return directive.Date.Raw
	case ledger.Pad:
		return directive.Date.Raw
	case ledger.Event:
		return directive.Date.Raw
	case ledger.Price:
		return directive.Date.Raw
	case ledger.Commodity:
		return directive.Date.Raw
	case ledger.Query:
		return directive.Date.Raw
	case ledger.Custom:
		return directive.Date.Raw
	default:
		return ""
	}
}
