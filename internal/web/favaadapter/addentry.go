// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package favaadapter

import (
	"fmt"
	"regexp"
	"strings"
)

// NewEntry is the structured shape the Add Entry modal submits. Only the
// fields the chosen type uses need to be set; the serializer validates every
// value it renders, so the output is always parseable Beancount.
type NewEntry struct {
	Type      string       `json:"type"`
	Date      string       `json:"date"`
	Flag      string       `json:"flag,omitempty"`
	Payee     string       `json:"payee,omitempty"`
	Narration string       `json:"narration,omitempty"`
	Comment   string       `json:"comment,omitempty"`
	Account   string       `json:"account,omitempty"`
	Amount    string       `json:"amount,omitempty"`
	Currency  string       `json:"currency,omitempty"`
	Tags      []string     `json:"tags,omitempty"`
	Links     []string     `json:"links,omitempty"`
	Postings  []NewPosting `json:"postings,omitempty"`
}

// NewPosting is one posting of a new transaction. An empty amount leaves the
// number out so Beancount interpolates it.
type NewPosting struct {
	Account  string `json:"account"`
	Amount   string `json:"amount,omitempty"`
	Currency string `json:"currency,omitempty"`
}

var (
	addEntryDate     = regexp.MustCompile(`\A\d{4}-\d{2}-\d{2}\z`)
	addEntryAccount  = regexp.MustCompile(`\A[A-Z][A-Za-z0-9\-]*(?::[A-Z][A-Za-z0-9\-]*)+\z`)
	addEntryCurrency = regexp.MustCompile(`\A[A-Z][A-Z0-9'._\-]{0,23}\z`)
	addEntryAmount   = regexp.MustCompile(`\A-?\d+(\.\d+)?\z`)
	addEntryTagLink  = regexp.MustCompile(`\A[A-Za-z0-9\-_/.]+\z`)
)

// SerializeNewEntries renders submitted entries as Beancount source, one
// blank-line-separated block per entry, ready to append to a ledger file.
func SerializeNewEntries(entries []NewEntry) (string, error) {
	if len(entries) == 0 {
		return "", fmt.Errorf("no entries to add")
	}
	blocks := make([]string, 0, len(entries))
	for index, entry := range entries {
		block, err := serializeNewEntry(entry)
		if err != nil {
			return "", fmt.Errorf("entry %d: %w", index+1, err)
		}
		blocks = append(blocks, block)
	}
	return strings.Join(blocks, "\n\n"), nil
}

func serializeNewEntry(entry NewEntry) (string, error) {
	if !addEntryDate.MatchString(entry.Date) {
		return "", fmt.Errorf("invalid date %q", entry.Date)
	}
	switch entry.Type {
	case "transaction":
		return serializeNewTransaction(entry)
	case "balance":
		return serializeNewBalance(entry)
	case "note":
		return serializeNewNote(entry)
	default:
		return "", fmt.Errorf("unsupported entry type %q", entry.Type)
	}
}

// serializeNewTransaction renders a transaction as Beancount source: the
// flag/payee/narration/tags/links head, then one indented posting per line.
// Every field is validated against the same shapes the parser accepts so a
// serialized entry round-trips.
func serializeNewTransaction(entry NewEntry) (string, error) {
	flag, err := transactionFlag(entry.Flag)
	if err != nil {
		return "", err
	}
	head, err := transactionHead(entry, flag)
	if err != nil {
		return "", err
	}
	lines := []string{head}
	for index, posting := range entry.Postings {
		line, err := serializeNewPosting(posting, index)
		if err != nil {
			return "", err
		}
		lines = append(lines, line)
	}
	if len(lines) < 2 {
		return "", fmt.Errorf("transaction needs at least one posting")
	}
	return strings.Join(lines, "\n"), nil
}

// transactionFlag normalizes the flag: empty and "txn" mean the default "*".
func transactionFlag(flag string) (string, error) {
	switch flag {
	case "", "txn":
		return "*", nil
	case "*", "!":
		return flag, nil
	default:
		return "", fmt.Errorf("invalid flag %q", flag)
	}
}

// transactionHead renders "DATE flag ["payee"] "narration" [#tag] [^link]".
func transactionHead(entry NewEntry, flag string) (string, error) {
	head := strings.Builder{}
	head.WriteString(entry.Date)
	head.WriteString(" ")
	head.WriteString(flag)
	if strings.TrimSpace(entry.Payee) != "" {
		payee, err := quoteBeancountString(entry.Payee)
		if err != nil {
			return "", fmt.Errorf("payee: %w", err)
		}
		head.WriteString(" " + payee)
	}
	narration, err := quoteBeancountString(entry.Narration)
	if err != nil {
		return "", fmt.Errorf("narration: %w", err)
	}
	head.WriteString(" " + narration)
	for _, tag := range entry.Tags {
		if !addEntryTagLink.MatchString(tag) {
			return "", fmt.Errorf("invalid tag %q", tag)
		}
		head.WriteString(" #" + tag)
	}
	for _, link := range entry.Links {
		if !addEntryTagLink.MatchString(link) {
			return "", fmt.Errorf("invalid link %q", link)
		}
		head.WriteString(" ^" + link)
	}
	return head.String(), nil
}

// serializeNewPosting renders one indented posting line; an empty amount
// leaves the posting for interpolation, otherwise amount and currency must
// both validate.
func serializeNewPosting(posting NewPosting, index int) (string, error) {
	if !addEntryAccount.MatchString(posting.Account) {
		return "", fmt.Errorf("posting %d: invalid account %q", index+1, posting.Account)
	}
	line := "  " + posting.Account
	if strings.TrimSpace(posting.Amount) == "" {
		return line, nil
	}
	if !addEntryAmount.MatchString(posting.Amount) {
		return "", fmt.Errorf("posting %d: invalid amount %q", index+1, posting.Amount)
	}
	if !addEntryCurrency.MatchString(posting.Currency) {
		return "", fmt.Errorf("posting %d: invalid currency %q", index+1, posting.Currency)
	}
	return line + " " + posting.Amount + " " + posting.Currency, nil
}

func serializeNewBalance(entry NewEntry) (string, error) {
	if !addEntryAccount.MatchString(entry.Account) {
		return "", fmt.Errorf("invalid account %q", entry.Account)
	}
	if !addEntryAmount.MatchString(entry.Amount) {
		return "", fmt.Errorf("invalid amount %q", entry.Amount)
	}
	if !addEntryCurrency.MatchString(entry.Currency) {
		return "", fmt.Errorf("invalid currency %q", entry.Currency)
	}
	return fmt.Sprintf("%s balance %s %s %s", entry.Date, entry.Account, entry.Amount, entry.Currency), nil
}

func serializeNewNote(entry NewEntry) (string, error) {
	if !addEntryAccount.MatchString(entry.Account) {
		return "", fmt.Errorf("invalid account %q", entry.Account)
	}
	comment, err := quoteBeancountString(entry.Comment)
	if err != nil {
		return "", fmt.Errorf("comment: %w", err)
	}
	return fmt.Sprintf("%s note %s %s", entry.Date, entry.Account, comment), nil
}

// quoteBeancountString escapes the two characters a Beancount string cannot
// contain raw, then wraps it in double quotes. Newlines are rejected outright
// so a submitted string can never split the entry across lines.
func quoteBeancountString(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if strings.ContainsAny(trimmed, "\n\r") {
		return "", fmt.Errorf("string must not contain newlines")
	}
	trimmed = strings.ReplaceAll(trimmed, "\\", "\\\\")
	trimmed = strings.ReplaceAll(trimmed, "\"", "\\\"")
	return "\"" + trimmed + "\"", nil
}
