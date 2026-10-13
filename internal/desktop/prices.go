// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package desktop

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"

	"orangecount/internal/ledger"
	"orangecount/internal/report"
)

func projectPrices(e ledger.Evaluation, r Request, snapshotID, from, to string) (*report.PricesProjection, error) {
	result := report.PriceHistoryView(e, r.PriceBase, r.PriceQuote, from, to)
	result.Commodities = report.PriceCatalog(e)
	known := map[string]bool{}
	for i := range result.Commodities {
		item := &result.Commodities[i]
		known[item.Currency] = true
		sum := sha256.Sum256([]byte(snapshotID + "\x00" + item.ID))
		item.ID = hex.EncodeToString(sum[:])
		if item.Currency == r.PriceBase {
			selected := *item
			result.Commodity = &selected
		}
	}
	if (r.PriceBase != "" && !known[r.PriceBase]) || (r.PriceQuote != "" && !known[r.PriceQuote]) {
		return nil, report.ErrCommodityUnknown
	}
	result.Section = r.PriceSection
	if result.Section == "" {
		result.Section = "commodities"
		if r.PriceBase != "" {
			result.Section = "history"
		}
	}
	if result.Section == "commodities" && strings.TrimSpace(r.CommoditySearch) != "" {
		needle := strings.ToLower(strings.TrimSpace(r.CommoditySearch))
		filtered := result.Commodities[:0]
		for _, item := range result.Commodities {
			searchable := strings.ToLower(item.Currency + " " + strings.Join(item.Accounts, " ") + " " + strings.Join(item.QuoteCurrencies, " "))
			if strings.Contains(searchable, needle) {
				filtered = append(filtered, item)
			}
		}
		result.Commodities = filtered
	}
	for i := range result.Records {
		sum := sha256.Sum256([]byte(snapshotID + "\x00" + result.Records[i].ID))
		result.Records[i].ID = hex.EncodeToString(sum[:])
	}
	if r.Sort == "descending" {
		slices.Reverse(result.Records)
		slices.Reverse(result.Commodities)
	}
	limit := r.Limit
	if limit == 0 {
		limit = 100
	}
	if result.Section == "commodities" {
		result.Commodities, result.CommoditiesPage = holdingPage(result.Commodities, r.Offset, limit)
		result.Records, result.RecordsPage = holdingPage(result.Records, 0, 0)
	} else {
		result.Commodities, result.CommoditiesPage = holdingPage(result.Commodities, 0, 100)
		result.Records, result.RecordsPage = holdingPage(result.Records, r.Offset, limit)
	}
	return result, nil
}
