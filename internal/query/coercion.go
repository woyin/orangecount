// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package query

import (
	"encoding/json"
	"math/big"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"orangecount/internal/ledger"
)

// Source precision affects an object-to-string cast, but not numeric equality
// or grouping. Keep that spelling beside the exact value, never in its key.
type metadataDecimal struct {
	Value ledger.Decimal
	Text  string
}

type metadataAmount struct {
	Value amountValue
	Text  string
}

func (v metadataAmount) MarshalJSON() ([]byte, error) { return json.Marshal(v.Value) }

func (v metadataDecimal) MarshalJSON() ([]byte, error) { return json.Marshal(v.Value) }

func metadataDecimalValue(number ledger.Number) metadataDecimal {
	value := ledger.DecimalFromNumber(number)
	text := strings.ReplaceAll(number.Raw, ",", "")
	if number.DecimalText != "" {
		if represented, err := ledger.ParseDecimal(number.DecimalText); err == nil && represented.Equal(value) {
			text = strings.ReplaceAll(number.DecimalText, ",", "")
		}
	}
	return metadataDecimal{value, finiteDecimalText(value, text, false)}
}

// Render a finite Decimal's coefficient and quantum without floating point.
// Parser-negated source zero loses its sign; an explicit string conversion
// retains the sign of zero. Scientific notation follows the Decimal value,
// not a currency-specific display preference.
func finiteDecimalText(value ledger.Decimal, raw string, signedZero bool) string {
	raw = normalizeDecimalInput(raw)
	if !metadataDecimalSyntax.MatchString(raw) {
		return value.String()
	}
	sign := ""
	if value.Sign() < 0 || signedZero && value.IsZero() && strings.HasPrefix(raw, "-") {
		sign = "-"
	}
	raw = strings.TrimLeft(raw, "+-")
	exponent := new(big.Int)
	if at := strings.IndexAny(raw, "eE"); at >= 0 {
		exponent.SetString(raw[at+1:], 10)
		raw = raw[:at]
	}
	if at := strings.IndexByte(raw, '.'); at >= 0 {
		exponent.Sub(exponent, big.NewInt(int64(len(raw)-at-1)))
		raw = raw[:at] + raw[at+1:]
	}
	coefficient := strings.TrimLeft(raw, "0")
	if coefficient == "" {
		coefficient = "0"
	}
	adjusted := new(big.Int).Add(exponent, big.NewInt(int64(len(coefficient)-1)))
	if exponent.Sign() > 0 || adjusted.Cmp(big.NewInt(-6)) < 0 {
		mantissa := coefficient[:1]
		if len(coefficient) > 1 {
			mantissa += "." + coefficient[1:]
		}
		exponentSign := ""
		if adjusted.Sign() >= 0 {
			exponentSign = "+"
		}
		return sign + mantissa + "E" + exponentSign + adjusted.String()
	}
	// Here -exponent <= len(coefficient)+6, so it fits the input's size.
	scale := int(new(big.Int).Neg(exponent).Int64())
	if scale == 0 {
		return sign + coefficient
	}
	if scale >= len(coefficient) {
		return sign + "0." + strings.Repeat("0", scale-len(coefficient)) + coefficient
	}
	at := len(coefficient) - scale
	return sign + coefficient[:at] + "." + coefficient[at:]
}

// The declared opposite operand chooses the cast. Two object operands cannot
// choose a signature from their eventual row values.
func bindMetadataOperand(expr typedExpr, target string) (typedExpr, error) {
	if expr.kind != "object" || target == "object" {
		return expr, nil
	}
	if target == "integer" {
		target = "decimal"
	}
	if target != "string" && target != "decimal" && target != "date" {
		return typedExpr{}, typedFailure("type-mismatch", "no metadata cast to %s", target)
	}
	read := expr.value
	sourceText := expr.text
	expr.kind, expr.nullable = target, true
	expr.value = func(row Row, rows []Row) (any, error) {
		value, err := read(row, rows)
		if err != nil || value == nil {
			return nil, err
		}
		if target == "date" {
			switch v := value.(type) {
			case ledger.Date:
				return v.Raw, nil
			case string:
				parsed, err := time.Parse("2006-1-2", v)
				if err == nil && parsed.Year() > 0 {
					return parsed.Format("2006-01-02"), nil
				}
			}
			return nil, nil
		}
		if target == "decimal" {
			switch v := unwrapMetadataNumber(value).(type) {
			case ledger.Decimal:
				return v, nil
			case bool:
				if v {
					one, _ := ledger.ParseDecimal("1")
					return one, nil
				}
				return ledger.Zero(), nil
			case string:
				text := normalizeDecimalInput(v)
				if metadataNonFiniteSyntax.MatchString(text) {
					return nil, typedFailure("unsupported-capability", "non-finite Decimal has no exact rational representation")
				}
				if !metadataDecimalSyntax.MatchString(text) {
					return nil, nil
				}
				if err := rowBudget(row, rows).decimalText(text); err != nil {
					return nil, err
				}
				number, err := ledger.ParseDecimal(text)
				if err != nil {
					return nil, nil
				}
				return number, nil
			default:
				return nil, nil
			}
		}
		switch v := value.(type) {
		case string:
			return v, nil
		case metadataDecimal:
			return v.Text, nil
		case metadataInteger:
			return v.Value.String(), nil
		case ledger.Date:
			return v.Raw, nil
		case bool:
			if v {
				return "TRUE", nil
			}
			return "FALSE", nil
		case amountValue:
			return amountText(v), nil
		case positionValue:
			return compoundPositionText(v), nil
		case inventoryValue:
			ordered := append(inventoryValue(nil), v...)
			sort.SliceStable(ordered, func(i, j int) bool { return comparePositionOrder(ordered[i], ordered[j]) < 0 })
			parts := make([]string, len(ordered))
			for i, position := range ordered {
				parts[i] = compoundPositionText(position)
			}
			return "(" + strings.Join(parts, ", ") + ")", nil
		case metadataAmount:
			return v.Text, nil
		default:
			return nil, typedFailure("unsupported-capability", "no metadata string representation for %T", value)
		}
	}
	if target == "decimal" {
		converted := expr.value
		expr.text = func(row Row, rows []Row) (any, error) {
			value, err := converted(row, rows)
			if err != nil || value == nil {
				return nil, err
			}
			source, err := read(row, rows)
			if err != nil {
				return nil, err
			}
			switch original := source.(type) {
			case string:
				return finiteDecimalText(value.(ledger.Decimal), original, true), nil
			case metadataDecimal:
				return original.Text, nil
			}
			if sourceText != nil {
				return sourceText(row, rows)
			}
			return value.(ledger.Decimal).String(), nil
		}
	}
	return expr, nil
}

var metadataDecimalSyntax = regexp.MustCompile(`^[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?$`)
var metadataNonFiniteSyntax = regexp.MustCompile(`(?i)^[+-]?(?:inf(?:inity)?|s?nan[0-9]*)$`)

func compoundPositionText(value positionValue) string {
	text := amountText(value.Units)
	if cost := value.Cost; cost != nil {
		parts := []string{amountText(amountValue{Number: cost.Number, Currency: cost.Currency, Text: cost.Text})}
		if cost.Date != nil {
			parts = append(parts, *cost.Date)
		}
		if cost.Label != nil && *cost.Label != "" {
			parts = append(parts, `"`+*cost.Label+`"`)
		}
		text += " {" + strings.Join(parts, ", ") + "}"
	}
	return text
}

func amountText(value amountValue) string {
	return amountNumberText(value) + " " + value.Currency
}

func amountNumberText(value amountValue) string {
	text := value.Text
	if text == "" {
		text = value.Number.String()
	}
	return text
}

func normalizeDecimalInput(text string) string {
	text = strings.ReplaceAll(strings.TrimSpace(text), "_", "")
	return normalizeNumericDigits(text)
}

func normalizeNumericDigits(text string) string {
	return strings.Map(func(r rune) rune {
		if r <= unicode.MaxASCII {
			return r
		}
		for _, block := range unicode.Digit.R16 {
			if r >= rune(block.Lo) && r <= rune(block.Hi) && (r-rune(block.Lo))%rune(block.Stride) == 0 {
				return '0' + (r-rune(block.Lo))/rune(block.Stride)%10
			}
		}
		for _, block := range unicode.Digit.R32 {
			if r >= rune(block.Lo) && r <= rune(block.Hi) && (r-rune(block.Lo))%rune(block.Stride) == 0 {
				return '0' + (r-rune(block.Lo))/rune(block.Stride)%10
			}
		}
		return r
	}, text)
}

func unwrapMetadataNumber(value any) any {
	switch v := value.(type) {
	case metadataDecimal:
		return v.Value
	case metadataInteger:
		return v.Value
	case metadataAmount:
		return v.Value
	default:
		return value
	}
}

func typedTruth(value any) (bool, error) {
	switch v := unwrapMetadataNumber(value).(type) {
	case amountValue:
		return !v.Number.IsZero(), nil
	case inventoryValue:
		return false, typedFailure("unsupported-capability", "inventory has no implicit truth value")
	default:
		return truthy(v), nil
	}
}
