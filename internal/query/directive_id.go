// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package query

import (
	"crypto/md5" // #nosec G501 -- the pinned oracle defines the id digest.
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"

	"orangecount/internal/ledger"
)

// Directive ids reproduce beancount.core.compare.hash_entry: an MD5 over each
// directive field rendered by Python's str(), with list and set members hashed
// individually and sorted. A namedtuple member is hashed recursively, exactly
// as stable_hash_namedtuple does. Source filenames are made relative to the
// ledger entry's directory before hashing, so an id never depends on the
// checkout location; that normalization is applied to both the oracle capture
// and this implementation.
type pyMember struct {
	text   string
	nested []pyField
}

type pyField struct {
	text    string
	members []pyMember
	isSeq   bool
}

func scalarPy(text string) pyField { return pyField{text: text} }

// sequencePy renders a list or set field. A nil slice is Python None, which
// hashes as a scalar, not as an empty container.
func sequencePy(members []pyMember) pyField {
	if members == nil {
		return pyField{text: "None"}
	}
	if members == nil {
		return pyField{isSeq: true}
	}
	return pyField{isSeq: true, members: members}
}

func stringMembers(values []string) []pyMember {
	members := make([]pyMember, 0, len(values))
	for _, value := range values {
		members = append(members, pyMember{text: value})
	}
	return members
}

// setMembers always yields a sequence. Upstream tag and link fields are
// frozensets, so an empty set hashes as zero members, never as None.
func setMembers(values []string) []pyMember { return stringMembers(values) }

// sequenceOrNone keeps Open.currencies semantics: a missing list is Python
// None, while a present but empty list hashes as zero members.
func sequenceOrNone(values []string) pyField {
	if values == nil {
		return pyField{text: "None"}
	}
	return sequencePy(stringMembers(values))
}

// listOrNone keeps a nil slice distinguishable from an empty one.
func listOrNone(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string{}, values...)
}

// directiveID returns the stable id of one directive.
func directiveID(entry ledger.EntryRecord, root string) (string, error) {
	fields, err := directiveFields(entry, root)
	if err != nil {
		return "", err
	}
	digest := md5.New() // #nosec G401 -- pinned oracle digest.
	writeField(digest, fields)
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func writeField(digest interface{ Write([]byte) (int, error) }, fields []pyField) {
	for _, field := range fields {
		if !field.isSeq {
			_, _ = digest.Write([]byte(field.text))
			continue
		}
		hashes := make([]string, 0, len(field.members))
		for _, member := range field.members {
			if member.nested != nil {
				nested := md5.New() // #nosec G401 -- pinned oracle digest.
				writeField(nested, member.nested)
				hashes = append(hashes, hex.EncodeToString(nested.Sum(nil)))
				continue
			}
			sum := md5.Sum([]byte(member.text)) // #nosec G401 -- pinned oracle digest.
			hashes = append(hashes, hex.EncodeToString(sum[:]))
		}
		sort.Strings(hashes)
		for _, hash := range hashes {
			_, _ = digest.Write([]byte(hash))
		}
	}
}

// relativeSource renders a source filename relative to the entry directory.
func relativeSource(name, root string) string {
	if name == "" || root == "" {
		return name
	}
	relative, err := filepath.Rel(root, name)
	if err != nil {
		return name
	}
	return filepath.ToSlash(relative)
}

func directiveFields(entry ledger.EntryRecord, root string) ([]pyField, error) {
	switch d := entry.Directive.(type) {
	case *ledger.Transaction:
		return directiveFields(ledger.EntryRecord{Date: entry.Date, Directive: *d, Span: entry.Span, File: entry.File}, root)
	case *ledger.Open:
		return directiveFields(ledger.EntryRecord{Date: entry.Date, Directive: *d, Span: entry.Span, File: entry.File}, root)
	case *ledger.Close:
		return directiveFields(ledger.EntryRecord{Date: entry.Date, Directive: *d, Span: entry.Span, File: entry.File}, root)
	case *ledger.Commodity:
		return directiveFields(ledger.EntryRecord{Date: entry.Date, Directive: *d, Span: entry.Span, File: entry.File}, root)
	case *ledger.Balance:
		return directiveFields(ledger.EntryRecord{Date: entry.Date, Directive: *d, Span: entry.Span, File: entry.File}, root)
	case *ledger.Note:
		return directiveFields(ledger.EntryRecord{Date: entry.Date, Directive: *d, Span: entry.Span, File: entry.File}, root)
	case *ledger.Event:
		return directiveFields(ledger.EntryRecord{Date: entry.Date, Directive: *d, Span: entry.Span, File: entry.File}, root)
	case *ledger.Price:
		return directiveFields(ledger.EntryRecord{Date: entry.Date, Directive: *d, Span: entry.Span, File: entry.File}, root)
	case *ledger.Document:
		return directiveFields(ledger.EntryRecord{Date: entry.Date, Directive: *d, Span: entry.Span, File: entry.File}, root)
	case *ledger.Query:
		return directiveFields(ledger.EntryRecord{Date: entry.Date, Directive: *d, Span: entry.Span, File: entry.File}, root)
	case *ledger.Custom:
		return directiveFields(ledger.EntryRecord{Date: entry.Date, Directive: *d, Span: entry.Span, File: entry.File}, root)
	case *ledger.Pad:
		return directiveFields(ledger.EntryRecord{Date: entry.Date, Directive: *d, Span: entry.Span, File: entry.File}, root)
	case ledger.Open:
		booking := "None"
		if d.Booking != "" {
			booking = d.Booking
		}
		return []pyField{
			mustMeta(entry, root, nil),
			scalarPy(d.Date.Raw),
			scalarPy(d.Account),
			sequenceOrNone(d.Currencies),
			scalarPy(booking),
		}, nil
	case ledger.Close:
		return []pyField{mustMeta(entry, root, nil), scalarPy(d.Date.Raw), scalarPy(d.Account)}, nil
	case ledger.Commodity:
		return []pyField{mustMeta(entry, root, nil), scalarPy(d.Date.Raw), scalarPy(d.Currency)}, nil
	case ledger.Pad:
		return []pyField{mustMeta(entry, root, nil), scalarPy(d.Date.Raw), scalarPy(d.Account), scalarPy(d.SourceAccount)}, nil
	case ledger.Balance:
		tolerance := "None"
		if d.Tolerance != nil {
			tolerance = d.Tolerance.Raw
		}
		diff := "None"
		if d.DiffAmount != nil {
			diff = pyAmount(*d.DiffAmount)
		}
		return []pyField{
			mustMeta(entry, root, nil),
			scalarPy(d.Date.Raw),
			scalarPy(d.Account),
			scalarPy(pyAmount(d.Amount)),
			scalarPy(tolerance),
			scalarPy(diff),
		}, nil
	case ledger.Transaction:
		postings := make([]pyMember, 0, len(d.Postings))
		for _, posting := range d.Postings {
			nested, err := pyPosting(posting, entry, root)
			if err != nil {
				return nil, err
			}
			postings = append(postings, pyMember{nested: nested})
		}
		payee := "None"
		if d.HasPayee || d.Payee != "" {
			payee = d.Payee
		}
		flag := "None"
		if d.Flag != "" {
			flag = d.Flag
		}
		return []pyField{
			mustMeta(entry, root, d.Tolerances),
			scalarPy(d.Date.Raw),
			scalarPy(flag),
			scalarPy(payee),
			scalarPy(d.Narration),
			sequencePy(setMembers(d.Tags)),
			sequencePy(setMembers(d.Links)),
			sequencePy(postings),
		}, nil
	case ledger.Note:
		return []pyField{
			mustMeta(entry, root, nil),
			scalarPy(d.Date.Raw),
			scalarPy(d.Account),
			scalarPy(d.Comment),
			sequencePy(setMembers(d.Tags)),
			sequencePy(setMembers(d.Links)),
		}, nil
	case ledger.Event:
		return []pyField{
			mustMeta(entry, root, nil),
			scalarPy(d.Date.Raw),
			scalarPy(d.Type),
			scalarPy(d.Value),
		}, nil
	case ledger.Price:
		return []pyField{
			mustMeta(entry, root, nil),
			scalarPy(d.Date.Raw),
			scalarPy(d.Currency),
			scalarPy(pyAmount(d.Amount)),
		}, nil
	case ledger.Document:
		// Upstream records the document filename as an absolute path. The
		// parser keeps the source spelling, so resolve it against the ledger
		// directory before hashing.
		filename := ""
		if len(d.Filenames) > 0 {
			filename = d.Filenames[0]
			if !filepath.IsAbs(filename) && entry.File != "" {
				filename = filepath.Join(filepath.Dir(entry.File), filename)
			}
			filename = relativeSource(filename, root)
		}
		return []pyField{
			mustMeta(entry, root, nil),
			scalarPy(d.Date.Raw),
			scalarPy(d.Account),
			scalarPy(filename),
			sequencePy(setMembers(d.Tags)),
			sequencePy(setMembers(d.Links)),
		}, nil
	case ledger.Query:
		return []pyField{
			mustMeta(entry, root, nil),
			scalarPy(d.Date.Raw),
			scalarPy(d.Name),
			scalarPy(d.Query),
		}, nil
	case ledger.Custom:
		values := make([]pyMember, 0, len(d.Values))
		for _, value := range d.Values {
			values = append(values, pyMember{text: pyCustomValue(value)})
		}
		return []pyField{
			mustMeta(entry, root, nil),
			scalarPy(d.Date.Raw),
			scalarPy(d.Type),
			sequencePy(values),
		}, nil
	default:
		return nil, fmt.Errorf("no directive id signature for %T", entry.Directive)
	}
}
