// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package dialect

import (
	"strconv"
	"strings"

	"orangecount/internal/ledger"
)

// SerializeTransaction renders a compiled dialect transaction as the
// canonical three-line Beancount v3 block used by export. Date and flag are
// always written; payee appears only when present; tags and links follow the
// narration; both posting amounts are explicit.
func SerializeTransaction(txn *ledger.Transaction) string {
	var head strings.Builder
	head.WriteString(txn.Date.Raw)
	head.WriteString(" ")
	head.WriteString(flagOrDefault(txn.Flag))
	if txn.Payee != "" {
		head.WriteString(" ")
		head.WriteString(strconv.Quote(txn.Payee))
	}
	head.WriteString(" ")
	head.WriteString(strconv.Quote(txn.Narration))
	for _, tag := range txn.Tags {
		head.WriteString(" #")
		head.WriteString(tag)
	}
	for _, link := range txn.Links {
		head.WriteString(" ^")
		head.WriteString(link)
	}
	var block strings.Builder
	block.WriteString(head.String())
	for _, posting := range txn.Postings {
		block.WriteString("\n  ")
		block.WriteString(posting.Account)
		block.WriteString(" ")
		if posting.Units != nil {
			block.WriteString(posting.Units.Number.Raw)
			if posting.Units.Currency != "" {
				block.WriteString(" ")
				block.WriteString(posting.Units.Currency)
			}
		}
	}
	return block.String()
}

func flagOrDefault(flag string) string {
	if flag == "" {
		return "*"
	}
	return flag
}

// SerializeDialect renders the dialect shorthand line for dialectize. It is
// the inverse of the parser's dialect grammar; only reparse-safe text may be
// produced (eligible transactions are filtered before this runs).
func SerializeDialect(txn *ledger.Transaction) string {
	var line strings.Builder
	line.WriteString(canonicalDate(txn.Date))
	if txn.Flag == "!" {
		line.WriteString(" !")
	}
	line.WriteString(" ")
	positive := positivePosting(txn)
	if positive == nil || positive.Units == nil {
		return ""
	}
	negative := otherPosting(txn, positive)
	line.WriteString(strings.TrimPrefix(positive.Units.Number.Raw, "+"))
	line.WriteString(" ")
	line.WriteString(positive.Units.Currency)
	line.WriteString(" @")
	line.WriteString(negative.Account)
	line.WriteString(" -> @")
	line.WriteString(positive.Account)
	if txn.Payee != "" {
		line.WriteString(" ")
		line.WriteString(strconv.Quote(txn.Payee))
	}
	if txn.Narration != "" {
		line.WriteString(" : ")
		line.WriteString(txn.Narration)
	}
	for _, tag := range txn.Tags {
		line.WriteString(" #")
		line.WriteString(tag)
	}
	for _, link := range txn.Links {
		line.WriteString(" ^")
		line.WriteString(link)
	}
	return line.String()
}

func canonicalDate(date ledger.Date) string {
	if date.Raw != "" && date.Valid() {
		return date.Raw
	}
	return ""
}

func positivePosting(txn *ledger.Transaction) *ledger.Posting {
	for i := range txn.Postings {
		units := txn.Postings[i].Units
		if units != nil && units.Number.Rat != nil && units.Number.Rat.Sign() > 0 {
			return &txn.Postings[i]
		}
	}
	return nil
}

func otherPosting(txn *ledger.Transaction, posting *ledger.Posting) *ledger.Posting {
	for i := range txn.Postings {
		if &txn.Postings[i] != posting {
			return &txn.Postings[i]
		}
	}
	return posting
}
