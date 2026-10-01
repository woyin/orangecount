// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package report

import (
	stdErrors "errors"
	"sort"
	"strconv"
	"time"

	"orangecount/internal/ledger"
	"orangecount/internal/source"
)

var ErrCommodityUnknown = stdErrors.New("commodity is not in the current snapshot")

type CommodityDeclaration struct {
	Date   string           `json:"date"`
	Source ProjectionSource `json:"source"`
}

type CommodityInfo struct {
	ID              string                 `json:"id"`
	Currency        string                 `json:"currency"`
	QuoteCurrencies []string               `json:"quote_currencies"`
	Accounts        []string               `json:"accounts"`
	Declarations    []CommodityDeclaration `json:"declarations"`
	PriceCount      int                    `json:"price_count"`
}

type PriceRecord struct {
	ID          string           `json:"id"`
	Base        string           `json:"base"`
	Quote       string           `json:"quote"`
	Date        string           `json:"date"`
	Rate        PresentedDecimal `json:"rate"`
	Value       *float64         `json:"value"`
	SourceOrder int              `json:"source_order"`
	Segment     int              `json:"segment"`
	Source      ProjectionSource `json:"source"`
}

type SelectedPrice struct {
	Base        string           `json:"base"`
	Quote       string           `json:"quote"`
	Date        string           `json:"date"`
	Rate        PresentedDecimal `json:"rate"`
	Inverse     bool             `json:"inverse"`
	SourceBase  string           `json:"source_base"`
	SourceQuote string           `json:"source_quote"`
	AgeDays     int              `json:"age_days"`
	Source      ProjectionSource `json:"source"`
}

type PricesProjection struct {
	Commodities     []CommodityInfo `json:"commodities"`
	CommoditiesPage ProjectionPage  `json:"commodities_page"`
	Commodity       *CommodityInfo  `json:"commodity"`
	Base            string          `json:"base"`
	Quote           string          `json:"quote"`
	DateFrom        string          `json:"date_from"`
	DateTo          string          `json:"date_to"`
	Section         string          `json:"section"`
	Records         []PriceRecord   `json:"records"`
	RecordsPage     ProjectionPage  `json:"records_page"`
	Selected        *SelectedPrice  `json:"selected"`
	SelectedReason  string          `json:"selected_reason"`
}

// Catalog membership comes from historical use, not today's surviving lots.
// A pair is a local source relationship; its presence does not promise a
// usable nonzero inverse quote at every historical cutoff.
func PriceCatalog(e ledger.Evaluation) []CommodityInfo {
	items := map[string]*CommodityInfo{}
	add := func(currency, account string) *CommodityInfo {
		if currency == "" {
			return nil
		}
		item := items[currency]
		if item == nil {
			item = &CommodityInfo{ID: holdingIdentity("commodity", currency), Currency: currency, QuoteCurrencies: []string{}, Accounts: []string{}, Declarations: []CommodityDeclaration{}}
			items[currency] = item
		}
		if account != "" {
			item.Accounts = appendUniqueString(item.Accounts, account)
		}
		return item
	}
	for _, entry := range e.Entries {
		switch value := entry.Directive.(type) {
		case ledger.Commodity:
			item := add(value.Currency, "")
			item.Declarations = append(item.Declarations, CommodityDeclaration{Date: entry.Date.Raw, Source: projectionLocation(entry.File, entry.Span)})
		case *ledger.Commodity:
			item := add(value.Currency, "")
			item.Declarations = append(item.Declarations, CommodityDeclaration{Date: entry.Date.Raw, Source: projectionLocation(entry.File, entry.Span)})
		case ledger.Open:
			for _, currency := range value.Currencies {
				add(currency, value.Account)
			}
		case *ledger.Open:
			for _, currency := range value.Currencies {
				add(currency, value.Account)
			}
		case ledger.Balance:
			add(value.Amount.Currency, value.Account)
		case *ledger.Balance:
			add(value.Amount.Currency, value.Account)
		}
		if tx := evaluatedTransaction(entry); tx != nil {
			for _, posting := range tx.Postings {
				if posting.Units != nil {
					add(posting.Units.Currency, posting.Account)
				}
				if posting.Cost != nil {
					for _, component := range posting.Cost.Components {
						if component.Kind == ledger.ValueAmount {
							add(component.Amount.Currency, posting.Account)
						}
					}
				}
				if posting.Price != nil {
					add(posting.Price.Amount.Currency, posting.Account)
				}
			}
		}
	}
	for base, prices := range e.Prices {
		for _, price := range prices {
			left, right := add(base, ""), add(price.Currency, "")
			left.QuoteCurrencies = appendUniqueString(left.QuoteCurrencies, price.Currency)
			right.QuoteCurrencies = appendUniqueString(right.QuoteCurrencies, base)
			left.PriceCount++
			if right != left {
				right.PriceCount++
			}
		}
	}
	result := make([]CommodityInfo, 0, len(items))
	for _, key := range sortedKeys(items) {
		item := items[key]
		sort.Strings(item.Accounts)
		sort.Strings(item.QuoteCurrencies)
		result = append(result, *item)
	}
	return result
}

// PriceHistoryView retains actual source directives, independently of the
// selected direct/inverse quote. The visible period never changes PriceAt.
func PriceHistoryView(e ledger.Evaluation, base, quote, from, to string) *PricesProjection {
	result := &PricesProjection{Base: base, Quote: quote, DateFrom: from, DateTo: to, Records: []PriceRecord{}, SelectedReason: "unspecified-pair"}
	paths := map[source.FileID]string{}
	segment := 0
	for order, entry := range e.Entries {
		paths[entry.Span.File] = entry.File
		var price *ledger.Price
		switch value := entry.Directive.(type) {
		case ledger.Price:
			price = &value
		case *ledger.Price:
			price = value
		}
		if price == nil || base == "" || quote == "" || price.Currency != base || price.Amount.Currency != quote || entry.Date.Raw < from || (to != "" && entry.Date.Raw > to) {
			continue
		}
		rate := FormatDecimal(ledger.DecimalFromNumber(price.Amount.Number))
		value := chartCoordinate(&rate)
		result.Records = append(result.Records, PriceRecord{ID: strconv.Itoa(order), Base: base, Quote: quote, Date: entry.Date.Raw, Rate: rate, Value: value, SourceOrder: order, Segment: segment, Source: projectionLocation(entry.File, entry.Span)})
		if value == nil {
			segment++
		}
	}
	if base != "" && quote != "" {
		result.SelectedReason = "missing-local-quote"
		if _, used, ok := PriceAt(e, base, quote, to); ok {
			selected := &SelectedPrice{Base: base, Quote: quote, Date: used.Date, Rate: used.Rate, Inverse: used.Inverse, SourceBase: base, SourceQuote: quote, Source: projectionLocation(paths[used.Source.File], used.Source)}
			if used.Inverse {
				selected.SourceBase, selected.SourceQuote = quote, base
			}
			asOf, endErr := time.Parse("2006-01-02", to)
			date, startErr := time.Parse("2006-01-02", used.Date)
			if endErr == nil && startErr == nil {
				selected.AgeDays = int(asOf.Sub(date) / (24 * time.Hour))
			}
			result.Selected, result.SelectedReason = selected, ""
		}
	}
	return result
}
