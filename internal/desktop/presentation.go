// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package desktop

import (
	"strings"

	"orangecount/internal/diagnostic"
	"orangecount/internal/ledger"
	"orangecount/internal/repairguidance"
	"orangecount/internal/report"
	"orangecount/internal/source"
)

type SourceLocation struct {
	Path      string `json:"path"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	EndLine   int    `json:"end_line"`
	EndColumn int    `json:"end_column"`
}

func location(path string, span source.Span) SourceLocation {
	return SourceLocation{Path: path, Line: span.StartLine, Column: span.StartColumn, EndLine: span.EndLine, EndColumn: span.EndColumn}
}

type RelatedLocation struct {
	Source  SourceLocation `json:"source"`
	Message string         `json:"message"`
}

type DiagnosticDTO struct {
	Code     string                `json:"code"`
	Severity diagnostic.Severity   `json:"severity"`
	Message  string                `json:"message"`
	Source   SourceLocation        `json:"source"`
	Related  []RelatedLocation     `json:"related"`
	Guide    *repairguidance.Guide `json:"guide,omitempty"`
}

func ProjectDiagnostics(values []diagnostic.Diagnostic, graph *source.Graph, locale string) []DiagnosticDTO {
	result := make([]DiagnosticDTO, 0, len(values))
	for _, value := range values {
		path := value.Path
		if graph != nil && graph.Path(value.Span.File) != "" {
			path = graph.Path(value.Span.File)
		}
		item := DiagnosticDTO{Code: value.Code, Severity: value.Severity, Message: diagnostic.Localize(value, locale).Message, Source: location(path, value.Span), Related: []RelatedLocation{}}
		for _, related := range value.Related {
			relatedPath := ""
			if graph != nil {
				relatedPath = graph.Path(related.Span.File)
			}
			item.Related = append(item.Related, RelatedLocation{Source: location(relatedPath, related.Span), Message: related.Message})
		}
		if guide, ok := repairguidance.Lookup(value.Code, locale); ok {
			item.Guide = &guide
		}
		result = append(result, item)
	}
	return result
}

type CostComponent struct {
	Kind     ledger.ValueKind         `json:"kind"`
	Text     string                   `json:"text"`
	Number   *report.PresentedDecimal `json:"number,omitempty"`
	Currency string                   `json:"currency,omitempty"`
}

type PostingCost struct {
	Total      bool            `json:"total"`
	Display    string          `json:"display"`
	Components []CostComponent `json:"components"`
}

type PostingPrice struct {
	Total    bool                    `json:"total"`
	Number   report.PresentedDecimal `json:"number"`
	Currency string                  `json:"currency"`
}

func postingCost(spec *ledger.CostSpec) *PostingCost {
	if spec == nil {
		return nil
	}
	cost := &PostingCost{Total: spec.Total, Components: []CostComponent{}}
	texts := []string{}
	for _, value := range spec.Components {
		component := CostComponent{Kind: value.Kind, Text: value.Raw}
		switch value.Kind {
		case ledger.ValueAmount:
			number := report.FormatDecimal(ledger.DecimalFromNumber(value.Amount.Number))
			component.Number, component.Currency = &number, value.Amount.Currency
			component.Text = number.Display + " " + value.Amount.Currency
		case ledger.ValueNumber:
			number := report.FormatDecimal(ledger.DecimalFromNumber(value.Number))
			component.Number, component.Text = &number, number.Display
		}
		cost.Components = append(cost.Components, component)
		texts = append(texts, component.Text)
	}
	cost.Display = "{" + strings.Join(texts, ", ") + "}"
	if spec.Total {
		cost.Display = "{" + cost.Display + "}"
	}
	return cost
}

func postingPrice(spec *ledger.PriceSpec) *PostingPrice {
	if spec == nil {
		return nil
	}
	return &PostingPrice{Total: spec.Total, Number: report.FormatDecimal(ledger.DecimalFromNumber(spec.Amount.Number)), Currency: spec.Amount.Currency}
}
