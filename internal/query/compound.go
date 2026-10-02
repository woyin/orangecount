// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package query

import (
	"encoding/json"
	"orangecount/internal/ledger"
	"orangecount/internal/presentation"
	"sort"
	"strings"
	"unicode/utf8"
)

// PresentedAmount and PresentedPosition carry financial values, not source
// locations. Cost nil, zero cost, absent label and empty label stay distinct.
type PresentedAmount struct {
	Number   PresentedDecimal `json:"number"`
	Currency string           `json:"currency"`
}

type PresentedCost struct {
	Number   PresentedDecimal `json:"number"`
	Currency string           `json:"currency"`
	Date     *string          `json:"date"`
	Label    *string          `json:"label"`
}

type PresentedPosition struct {
	Units PresentedAmount `json:"units"`
	Cost  *PresentedCost  `json:"cost"`
}

// These exact values are also the group-key representation. No provenance
// enters equality; the JSON tags retain type and nullable identity components.
type amountValue struct {
	Number   ledger.Decimal `json:"number"`
	Currency string         `json:"currency"`
	Text     string         `json:"-"`
}

type costValue struct {
	Number   ledger.Decimal `json:"number"`
	Currency string         `json:"currency"`
	Date     *string        `json:"date"`
	Label    *string        `json:"label"`
	Text     string         `json:"-"`
}

type positionValue struct {
	Units amountValue `json:"units"`
	Cost  *costValue  `json:"cost"`
}

type inventoryValue []positionValue

// Group identity ignores inventory insertion order. ORDER BY still observes
// the reference's stable Position ties, while wire values are canonical.
func (value inventoryValue) MarshalJSON() ([]byte, error) {
	ordered := canonicalInventory(value)
	return json.Marshal([]positionValue(ordered))
}

func canonicalInventory(value inventoryValue) inventoryValue {
	ordered := make(inventoryValue, len(value))
	copy(ordered, value)
	sort.Slice(ordered, func(i, j int) bool { return comparePositionIdentity(ordered[i], ordered[j]) < 0 })
	return ordered
}

type queryLotKey struct {
	Currency, Number, CostCurrency, Date, Label string
	CostPresent, DatePresent, LabelPresent      bool
}

func lotKey(value positionValue) queryLotKey {
	key := queryLotKey{Currency: value.Units.Currency}
	if cost := value.Cost; cost != nil {
		key.CostPresent = true
		key.Number, key.CostCurrency = cost.Number.String(), cost.Currency
		if cost.Date != nil {
			key.DatePresent, key.Date = true, *cost.Date
		}
		if cost.Label != nil {
			key.LabelPresent, key.Label = true, *cost.Label
		}
	}
	return key
}

type inventoryAccumulator struct {
	positions inventoryValue
	indices   map[queryLotKey]int
}

func (inv *inventoryAccumulator) add(value positionValue) {
	if inv.indices == nil {
		inv.indices = make(map[queryLotKey]int)
	}
	key := lotKey(value)
	if index, exists := inv.indices[key]; exists {
		left, right := inv.positions[index].Units, value.Units
		value.Units.Number = left.Number.Add(right.Number)
		leftQuantum, leftErr := decimalQuantum(amountNumberText(left))
		rightQuantum, rightErr := decimalQuantum(amountNumberText(right))
		if leftErr == nil && rightErr == nil {
			number := contextualDecimal(value.Units.Number, min(leftQuantum, rightQuantum),
				strings.HasPrefix(amountNumberText(left), "-") && strings.HasPrefix(amountNumberText(right), "-"))
			value.Units.Number, value.Units.Text = number.Value, number.Text
		} else {
			value.Units.Text = value.Units.Number.String()
		}
		inv.positions[index] = value
		if value.Units.Number.IsZero() {
			delete(inv.indices, key)
		}
	} else if !value.Units.Number.IsZero() {
		inv.indices[key] = len(inv.positions)
		inv.positions = append(inv.positions, value)
	}
}

func (inv inventoryAccumulator) value() inventoryValue {
	result := make(inventoryValue, 0, len(inv.indices))
	for _, value := range inv.positions {
		if !value.Units.Number.IsZero() {
			result = append(result, value)
		}
	}
	return result
}

func comparePositionIdentity(a, b positionValue) int {
	if comparison := strings.Compare(a.Units.Currency, b.Units.Currency); comparison != 0 {
		return comparison
	}
	if a.Cost == nil && b.Cost == nil {
		return 0
	}
	if a.Cost == nil {
		return -1
	}
	if b.Cost == nil {
		return 1
	}
	if comparison := a.Cost.Number.Cmp(b.Cost.Number); comparison != 0 {
		return comparison
	}
	if comparison := strings.Compare(a.Cost.Currency, b.Cost.Currency); comparison != 0 {
		return comparison
	}
	if comparison := compareOptionalText(a.Cost.Date, b.Cost.Date); comparison != 0 {
		return comparison
	}
	return compareOptionalText(a.Cost.Label, b.Cost.Label)
}

func compareOptionalText(a, b *string) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	return strings.Compare(*a, *b)
}

func compareAmountOrder(a, b amountValue) int {
	if comparison := strings.Compare(a.Currency, b.Currency); comparison != 0 {
		return comparison
	}
	return a.Number.Cmp(b.Number)
}

// Query ORDER BY follows the pinned Position comparison, which differs from
// inventory identity: source, date and label do not break its sorting ties.
func comparePositionOrder(a, b positionValue) int {
	left, right := queryCommodityOrder(a.Units.Currency), queryCommodityOrder(b.Units.Currency)
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	leftCost, rightCost := ledger.Zero(), ledger.Zero()
	leftCurrency, rightCurrency := "", ""
	if a.Cost != nil {
		leftCost, leftCurrency = a.Cost.Number, a.Cost.Currency
	}
	if b.Cost != nil {
		rightCost, rightCurrency = b.Cost.Number, b.Cost.Currency
	}
	if comparison := leftCost.Cmp(rightCost); comparison != 0 {
		return comparison
	}
	if comparison := strings.Compare(leftCurrency, rightCurrency); comparison != 0 {
		return comparison
	}
	return a.Units.Number.Cmp(b.Units.Number)
}

func queryCommodityOrder(currency string) int {
	for i, name := range []string{"USD", "EUR", "JPY", "CAD", "GBP", "AUD", "NZD", "CHF"} {
		if currency == name {
			return i
		}
	}
	return 8 + utf8.RuneCountInString(currency)
}

func compareInventoryOrder(a, b inventoryValue) int {
	left, right := append(inventoryValue(nil), a...), append(inventoryValue(nil), b...)
	sort.SliceStable(left, func(i, j int) bool { return comparePositionOrder(left[i], left[j]) < 0 })
	sort.SliceStable(right, func(i, j int) bool { return comparePositionOrder(right[i], right[j]) < 0 })
	for i := 0; i < len(left) && i < len(right); i++ {
		// The reference list comparison first checks full Position equality.
		// Unequal dates/labels can nevertheless be an ORDER BY tie.
		if lotKey(left[i]) == lotKey(right[i]) && left[i].Units.Number.Equal(right[i].Units.Number) {
			continue
		}
		return comparePositionOrder(left[i], right[i])
	}
	if len(left) < len(right) {
		return -1
	}
	if len(left) > len(right) {
		return 1
	}
	return 0
}

func sumCompoundValues(arg typedExpr, rows []Row) (any, error) {
	inventory := inventoryAccumulator{}
	for _, row := range rows {
		value, err := arg.eval(row, []Row{row})
		if err != nil {
			return nil, err
		}
		if value != nil {
			if arg.kind == "inventory" {
				for _, position := range value.(inventoryValue) {
					inventory.add(position)
				}
			} else if arg.kind == "amount" {
				inventory.add(positionValue{Units: value.(amountValue)})
			} else {
				inventory.add(value.(positionValue))
			}
		}
	}
	return inventory.value(), nil
}

func queryPosition(posting ledger.Posting) positionValue {
	units := metadataDecimalValue(posting.Units.Number)
	result := positionValue{Units: amountValue{Number: units.Value, Currency: posting.Units.Currency, Text: units.Text}}
	if posting.Cost == nil {
		return result
	}
	cost := &costValue{}
	for _, item := range posting.Cost.Components {
		switch item.Kind {
		case ledger.ValueAmount:
			cost.Number, cost.Currency = ledger.DecimalFromNumber(item.Amount.Number), item.Amount.Currency
			cost.Text = metadataDecimalValue(item.Amount.Number).Text
		case ledger.ValueDate:
			date := item.Date.Raw
			cost.Date = &date
		case ledger.ValueString:
			label := item.String
			cost.Label = &label
		}
	}
	result.Cost = cost
	return result
}

func presentAmount(value amountValue) PresentedAmount {
	return PresentedAmount{Number: presentation.FormatDecimal(value.Number), Currency: value.Currency}
}

func positionCost(value positionValue) amountValue {
	if cost := value.Cost; cost != nil {
		exact := value.Units.Number.Mul(cost.Number)
		left, leftErr := decimalQuantum(value.Units.Text)
		right, rightErr := decimalQuantum(cost.Text)
		if leftErr != nil || rightErr != nil {
			// AVERAGE can produce exact nonterminating rational costs. It is
			// an independent extension, never an integer-quantum Decimal.
			return amountValue{Number: exact, Currency: cost.Currency, Text: exact.String()}
		}
		number := contextualDecimal(exact, left+right,
			strings.HasPrefix(value.Units.Text, "-") != strings.HasPrefix(cost.Text, "-"))
		return amountValue{Number: number.Value, Currency: cost.Currency, Text: number.Text}
	}
	return value.Units
}

// A total quote is divided by the absolute quantity before query valuation.
// Both that division and the subsequent multiplication use the query context;
// the source total and ledger's exact financial weights remain untouched.
func queryPriceWeight(posting ledger.Posting) amountValue {
	units := metadataDecimalValue(posting.Units.Number)
	quote := metadataDecimalValue(posting.Price.Amount.Number)
	unitQuantum, unitErr := decimalQuantum(units.Text)
	quoteQuantum, quoteErr := decimalQuantum(quote.Text)
	if unitErr != nil || quoteErr != nil {
		value := ledger.BookedPriceWeight(posting)
		return amountValue{Number: value, Currency: posting.Price.Amount.Currency, Text: value.String()}
	}
	quote = queryUnitQuote(posting, units, quote, unitQuantum, quoteQuantum)
	quoteQuantum, _ = decimalQuantum(quote.Text)
	value := contextualDecimal(units.Value.Mul(quote.Value), unitQuantum+quoteQuantum,
		strings.HasPrefix(units.Text, "-") != strings.HasPrefix(quote.Text, "-"))
	return amountValue{Number: value.Value, Currency: posting.Price.Amount.Currency, Text: value.Text}
}

func presentPosition(value positionValue) PresentedPosition {
	result := PresentedPosition{Units: presentAmount(value.Units)}
	if cost := value.Cost; cost != nil {
		result.Cost = &PresentedCost{Number: presentation.FormatDecimal(cost.Number), Currency: cost.Currency, Date: cost.Date, Label: cost.Label}
	}
	return result
}

func queryUnitQuote(posting ledger.Posting, units, quote metadataDecimal, unitQuantum, quoteQuantum int64) metadataDecimal {
	if posting.Price.Total && units.Value.IsZero() {
		// The booked unit quote is a plain zero even when the source total
		// is nonzero or carries fractional digits.
		quote, quoteQuantum = metadataDecimal{ledger.Zero(), "0"}, 0
	} else if posting.Price.Total {
		quantity := units.Value
		if quantity.Sign() < 0 {
			quantity = quantity.Neg()
		}
		quotient := quote.Value.Quo(quantity)
		quantum := quoteQuantum - unitQuantum
		if !quotient.IsZero() {
			if exactQuantum, finite := decimalExactQuantum(quotient); finite {
				quantum = min(quantum, exactQuantum)
			} else {
				quantum = decimalAdjusted(quotient) - 27
			}
		}
		quote = contextualDecimal(quotient, quantum, strings.HasPrefix(quote.Text, "-"))
	}
	return quote
}
func queryUnitPrice(posting ledger.Posting) amountValue {
	quote := metadataDecimalValue(posting.Price.Amount.Number)
	if posting.Units != nil {
		units := metadataDecimalValue(posting.Units.Number)
		unitQuantum, uerr := decimalQuantum(units.Text)
		quoteQuantum, qerr := decimalQuantum(quote.Text)
		if uerr == nil && qerr == nil {
			quote = queryUnitQuote(posting, units, quote, unitQuantum, quoteQuantum)
		}
	}
	return amountValue{Number: quote.Value, Currency: posting.Price.Amount.Currency, Text: quote.Text}
}
