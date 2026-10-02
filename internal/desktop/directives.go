// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package desktop

import (
	"encoding/json"
	"fmt"
	"orangecount/internal/ledger"
	"orangecount/internal/presentation"
	"orangecount/internal/query"
	"orangecount/internal/report"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
	"reflect"
)

type DirectiveValue struct {
	Type  string `json:"type"`
	Value any    `json:"value"`
}

func (v DirectiveValue) MarshalJSON() ([]byte, error) {
	if v.Type == "null" {
		return json.Marshal(struct {
			Type string `json:"type"`
		}{v.Type})
	}
	type plain DirectiveValue
	return json.Marshal(plain(v))
}

type DirectiveField struct {
	Name  string         `json:"name"`
	Value DirectiveValue `json:"value"`
}
type DirectiveMetadata struct {
	Key    string         `json:"key"`
	Value  DirectiveValue `json:"value"`
	Source SourceLocation `json:"source"`
}
type DirectivePosting struct {
	Posting  JournalPostingDTO   `json:"posting"`
	Metadata []DirectiveMetadata `json:"metadata"`
}
type DirectiveDTO struct {
	Attachments []AttachmentDTO     `json:"attachments"`
	ID          string              `json:"id"`
	Kind        string              `json:"kind"`
	SourceKind  string              `json:"source_kind"`
	Date        *string             `json:"date"`
	Generated   bool                `json:"generated"`
	OriginID    *string             `json:"origin_id"`
	Source      SourceLocation      `json:"source"`
	Fields      []DirectiveField    `json:"fields"`
	Metadata    []DirectiveMetadata `json:"metadata"`
	Tags        []string            `json:"tags"`
	Links       []string            `json:"links"`
	Postings    []DirectivePosting  `json:"postings"`
}
type DirectivesPayload struct {
	SnapshotID string                `json:"snapshot_id"`
	Mode       string                `json:"mode"`
	DateFrom   string                `json:"date_from"`
	DateTo     string                `json:"date_to"`
	Page       report.ProjectionPage `json:"page"`
	Records    []DirectiveDTO        `json:"records"`
}

func validDirectiveKind(kind string) bool {
	switch ledger.DirectiveKind(kind) {
	case ledger.KindOption, ledger.KindPlugin, ledger.KindInclude, ledger.KindPushTag, ledger.KindPopTag, ledger.KindPushMeta, ledger.KindPopMeta, ledger.KindOpen, ledger.KindClose, ledger.KindCommodity, ledger.KindBalance, ledger.KindPad, ledger.KindEvent, ledger.KindQuery, ledger.KindPrice, ledger.KindDocument, ledger.KindNote, ledger.KindCustom, ledger.KindTxn, ledger.KindDialect:
		return true
	}
	return false
}

func (p *publication) readDirectives(request Request, snapshotID, from, to string) (*DirectivesPayload, error) {
	descending := request.Sort != "ascending"
	if request.JournalMode == "config" {
		descending = request.Sort == "descending"
	}
	page, err := p.snapshot.SelectDirectives(snapshot.DirectiveSelection{Mode: request.JournalMode, Kind: request.DirectiveKind, Text: request.Text, Account: request.Account, DateFrom: from, DateTo: to, Offset: request.Offset, Limit: request.Limit, Descending: descending})
	if err != nil {
		return nil, ErrRequest
	}
	if request.JournalMode == "config" {
		from, to = "", ""
	}
	payload := &DirectivesPayload{DateFrom: from, DateTo: to, SnapshotID: snapshotID, Mode: request.JournalMode, Records: []DirectiveDTO{}, Page: report.ProjectionPage{TotalCount: page.TotalCount, Offset: page.Offset, Limit: page.Limit, ReturnedCount: len(page.Records), HasMore: page.HasMore, Truncated: len(page.Records) < page.TotalCount}}
	p.documents.mu.Lock()
	defer p.documents.mu.Unlock()
	attachmentCount := 0
	for _, record := range page.Records {
		dto, err := projectDirective(record, p.snapshot.Graph())
		if err != nil {
			return nil, err
		}
		if dto.Kind == "document" {
			for _, field := range dto.Fields {
				if field.Name == "filename" {
					attachmentCount++
				}
			}
			if attachmentCount > 500 {
				return nil, ErrResultBudget
			}
		}
		dto.Attachments = []AttachmentDTO{}
		if err := p.projectAttachments(&dto); err != nil {
			return nil, err
		}
		payload.Records = append(payload.Records, dto)
	}
	return payload, nil
}

func projectDirective(record snapshot.DirectiveRecord, graph *source.Graph) (DirectiveDTO, error) {
	dto := DirectiveDTO{ID: record.ID, Kind: string(record.Kind), SourceKind: string(record.SourceKind), Generated: record.Generated, Source: location(record.File, record.Span), Fields: []DirectiveField{}, Metadata: []DirectiveMetadata{}, Tags: []string{}, Links: []string{}, Postings: []DirectivePosting{}}
	if record.Date != "" {
		date := record.Date
		dto.Date = &date
	}
	if record.OriginID != "" {
		id := record.OriginID
		dto.OriginID = &id
	}
	value := reflect.Indirect(reflect.ValueOf(record.Directive))
	if field := value.FieldByName("Meta"); field.IsValid() {
		var err error
		dto.Metadata, err = projectDirectiveMetadata(field.Interface().([]ledger.Metadata), record.File, graph)
		if err != nil {
			return dto, err
		}
	}
	add := func(name string, value DirectiveValue) { dto.Fields = append(dto.Fields, DirectiveField{name, value}) }
	text := func(name, kind, value string) { add(name, DirectiveValue{kind, value}) }
	number := func(name string, n ledger.Number) {
		add(name, DirectiveValue{"decimal", presentation.FormatDecimal(ledger.NewDecimal(n.Rat))})
	}
	amount := func(name string, a ledger.Amount) {
		add(name, DirectiveValue{"amount", query.PresentedAmount{Number: presentation.FormatDecimal(ledger.NewDecimal(a.Number.Rat)), Currency: a.Currency}})
	}
	switch d := value.Interface().(type) {
	case ledger.Option:
		text("key", "string", d.Key)
		text("value", "string", d.Value)
	case ledger.Plugin:
		text("module", "string", d.Module)
		text("config", "string", d.Config)
	case ledger.Include:
		text("path", "string", d.Path)
	case ledger.TagDirective:
		text("tag", "tag", d.Tag)
	case ledger.PushMeta:
		text("key", "string", d.Key)
		v, err := projectDirectiveValue(d.Value, record.File, graph)
		if err != nil {
			return dto, err
		}
		add("value", v)
	case ledger.PopMeta:
		text("key", "string", d.Key)
	case ledger.Open:
		text("account", "account", d.Account)
		for _, currency := range d.Currencies {
			text("currency", "currency", currency)
		}
		text("booking", "string", d.Booking)
	case ledger.Close:
		text("account", "account", d.Account)
	case ledger.Commodity:
		text("currency", "currency", d.Currency)
	case ledger.Balance:
		text("account", "account", d.Account)
		amount("amount", d.Amount)
		if d.Tolerance == nil {
			add("tolerance", DirectiveValue{Type: "null"})
		} else {
			number("tolerance", *d.Tolerance)
		}
	case ledger.Pad:
		text("account", "account", d.Account)
		text("source_account", "account", d.SourceAccount)
	case ledger.Event:
		text("event_type", "string", d.Type)
		text("value", "string", d.Value)
	case ledger.Query:
		text("name", "string", d.Name)
		text("query", "string", d.Query)
	case ledger.Price:
		text("currency", "currency", d.Currency)
		amount("amount", d.Amount)
	case ledger.Document:
		text("account", "account", d.Account)
		for _, filename := range d.Filenames {
			text("filename", "string", filename)
		}
		dto.Tags = append(dto.Tags, d.Tags...)
		dto.Links = append(dto.Links, d.Links...)
	case ledger.Note:
		text("account", "account", d.Account)
		text("comment", "string", d.Comment)
		dto.Tags = append(dto.Tags, d.Tags...)
		dto.Links = append(dto.Links, d.Links...)
	case ledger.Custom:
		text("custom_type", "string", d.Type)
		for _, item := range d.Values {
			v, err := projectDirectiveValue(item, record.File, graph)
			if err != nil {
				return dto, err
			}
			add("value", v)
		}
	case ledger.Transaction:
		text("flag", "string", d.Flag)
		text("payee", "string", d.Payee)
		text("narration", "string", d.Narration)
		dto.Tags = append(dto.Tags, d.Tags...)
		dto.Links = append(dto.Links, d.Links...)
		projection := ProjectJournal(ledger.Evaluation{Entries: []ledger.EntryRecord{{Date: d.Date, Directive: &d, File: record.File, Span: record.Span}}}).Transactions[0]
		for i, posting := range d.Postings {
			meta, err := projectDirectiveMetadata(posting.Meta, record.File, graph)
			if err != nil {
				return dto, err
			}
			projected := projection.Postings[i]
			if posting.SourceMetadata.Present && posting.SourceMetadata.File != "" {
				projected.Source = location(posting.SourceMetadata.File, source.Span{StartLine: posting.SourceMetadata.Line, StartColumn: 1})
			}
			dto.Postings = append(dto.Postings, DirectivePosting{projected, meta})
		}
	default:
		return dto, fmt.Errorf("unsupported directive projection")
	}
	return dto, nil
}
func projectDirectiveMetadata(metadata []ledger.Metadata, file string, graph *source.Graph) ([]DirectiveMetadata, error) {
	return projectDirectiveMetadataDepth(metadata, file, graph, 0)
}
func projectDirectiveMetadataDepth(metadata []ledger.Metadata, file string, graph *source.Graph, depth int) ([]DirectiveMetadata, error) {
	if depth > 64 {
		return nil, ErrResultBudget
	}
	result := []DirectiveMetadata{}
	for _, meta := range metadata {
		path := file
		if graph != nil && graph.Path(meta.Span.File) != "" {
			path = graph.Path(meta.Span.File)
		}
		value, err := projectDirectiveValueDepth(meta.Value, path, graph, depth)
		if err != nil {
			return nil, err
		}
		result = append(result, DirectiveMetadata{meta.Key, value, location(path, meta.Span)})
	}
	return result, nil
}
func projectDirectiveValue(value ledger.Value, file string, graph *source.Graph) (DirectiveValue, error) {
	return projectDirectiveValueDepth(value, file, graph, 0)
}
func projectDirectiveValueDepth(value ledger.Value, file string, graph *source.Graph, depth int) (DirectiveValue, error) {
	if depth > 64 {
		return DirectiveValue{}, ErrResultBudget
	}
	kind := string(value.Kind)
	switch value.Kind {
	case ledger.ValueNull:
		return DirectiveValue{Type: "null"}, nil
	case ledger.ValueString, ledger.ValueAccount, ledger.ValueCurrency, ledger.ValueTag, ledger.ValueLink:
		return DirectiveValue{kind, value.String}, nil
	case ledger.ValueDate:
		return DirectiveValue{"date", value.Date.Raw}, nil
	case ledger.ValueBool:
		return DirectiveValue{"bool", value.Bool}, nil
	case ledger.ValueNumber:
		return DirectiveValue{"decimal", presentation.FormatDecimal(ledger.NewDecimal(value.Number.Rat))}, nil
	case ledger.ValueAmount:
		return DirectiveValue{"amount", query.PresentedAmount{Number: presentation.FormatDecimal(ledger.NewDecimal(value.Amount.Number.Rat)), Currency: value.Amount.Currency}}, nil
	case ledger.ValueList:
		items := []DirectiveValue{}
		for _, item := range value.List {
			v, err := projectDirectiveValueDepth(item, file, graph, depth+1)
			if err != nil {
				return DirectiveValue{}, err
			}
			items = append(items, v)
		}
		return DirectiveValue{"list", items}, nil
	case ledger.ValueMap:
		fields, err := projectDirectiveMetadataDepth(value.Map, file, graph, depth+1)
		return DirectiveValue{"map", fields}, err
	default:
		return DirectiveValue{}, fmt.Errorf("unsupported directive metadata value")
	}
}
