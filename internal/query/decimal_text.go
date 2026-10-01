// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package query

import (
	"math/big"
	"strconv"
	"strings"

	"orangecount/internal/ledger"
)

// Decimal spelling belongs to the query expression, not financial identity.
// Arithmetic must propagate the operands' quantum instead of formatting the
// result's reduced rational, which would erase trailing zeros and negative zero.
func decimalOperandText(expr typedExpr, row Row, rows []Row) (string, error) {
	if expr.text != nil {
		value, err := expr.text(row, rows)
		if err != nil || value == nil {
			return "", err
		}
		return value.(string), nil
	}
	value, err := expr.eval(row, rows)
	if err != nil || value == nil {
		return "", err
	}
	return value.(ledger.Decimal).String(), nil
}

func decimalQuantum(text string) (int64, error) {
	if !metadataDecimalSyntax.MatchString(text) {
		return 0, typedFailure("unsupported-capability", "Decimal spelling is unavailable for this expression")
	}
	exponent := int64(0)
	if at := strings.IndexAny(text, "eE"); at >= 0 {
		value, err := strconv.ParseInt(text[at+1:], 10, 64)
		if err != nil {
			return 0, typedFailure("unsupported-capability", "Decimal exponent exceeds the supported range")
		}
		exponent, text = value, text[:at]
	}
	if at := strings.IndexByte(text, '.'); at >= 0 {
		exponent -= int64(len(text) - at - 1)
	}
	return exponent, nil
}

func decimalPower(exponent int64) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(exponent), nil)
}

func decimalArithmeticResult(operator string, left, right typedExpr, value ledger.Decimal, row Row, rows []Row) (metadataDecimal, error) {
	leftText, err := decimalOperandText(left, row, rows)
	if err != nil {
		return metadataDecimal{}, err
	}
	rightText, err := decimalOperandText(right, row, rows)
	if err != nil {
		return metadataDecimal{}, err
	}
	leftQuantum, err := decimalQuantum(leftText)
	if err != nil {
		return metadataDecimal{}, err
	}
	rightQuantum, err := decimalQuantum(rightText)
	if err != nil {
		return metadataDecimal{}, err
	}
	quantum := min(leftQuantum, rightQuantum)
	negative := strings.HasPrefix(leftText, "-") && strings.HasPrefix(rightText, "-")
	switch operator {
	case "-":
		negative = strings.HasPrefix(leftText, "-") && !strings.HasPrefix(rightText, "-")
	case "*":
		quantum = leftQuantum + rightQuantum
		negative = strings.HasPrefix(leftText, "-") != strings.HasPrefix(rightText, "-")
	case "/":
		quantum = leftQuantum - rightQuantum
		negative = strings.HasPrefix(leftText, "-") != strings.HasPrefix(rightText, "-")
		if !value.IsZero() {
			if exactQuantum, finite := decimalExactQuantum(value); finite {
				quantum = min(quantum, exactQuantum)
			} else {
				quantum = decimalAdjusted(value) - 27
			}
		}
	}
	return contextualDecimal(value, quantum, negative), nil
}

// Highest exact quantum: 500 can be expressed as 5E+2. Exact division retains
// the preferred input quantum only if it can represent the quotient.
func decimalExactQuantum(value ledger.Decimal) (int64, bool) {
	text := value.String()
	quantum, err := decimalQuantum(text)
	if err != nil {
		return 0, false
	}
	coefficient := strings.ReplaceAll(text, ".", "")
	for strings.HasSuffix(coefficient, "0") {
		coefficient = strings.TrimSuffix(coefficient, "0")
		quantum++
	}
	return quantum, true
}

// The fixed query oracle uses the default 28-significant-digit, half-even
// context for arithmetic. Ledger values and explicit decimal constructors do
// not pass through this operation. The result retains its Decimal quantum.
func contextualDecimal(value ledger.Decimal, quantum int64, negativeZero bool) metadataDecimal {
	if !value.IsZero() {
		quantum = max(quantum, decimalAdjusted(value)-27)
	}
	coefficient := value.Rat()
	negative := coefficient.Sign() < 0
	coefficient.Abs(coefficient)
	if quantum < 0 {
		coefficient.Mul(coefficient, new(big.Rat).SetInt(decimalPower(-quantum)))
	} else {
		coefficient.Quo(coefficient, new(big.Rat).SetInt(decimalPower(quantum)))
	}
	integer, remainder := new(big.Int), new(big.Int)
	integer.QuoRem(coefficient.Num(), coefficient.Denom(), remainder)
	comparison := new(big.Int).Lsh(remainder, 1).Cmp(coefficient.Denom())
	if comparison > 0 || comparison == 0 && integer.Bit(0) != 0 {
		integer.Add(integer, big.NewInt(1))
	}
	// A rounding carry can add a digit: adjust the quantum with the coefficient.
	if len(integer.String()) > 28 {
		integer.Quo(integer, big.NewInt(10))
		quantum++
	}
	if negative {
		integer.Neg(integer)
	}
	text := integer.String()
	if integer.Sign() == 0 && negativeZero {
		text = "-" + text
	}
	text += "e" + strconv.FormatInt(quantum, 10)
	rounded, _ := ledger.ParseDecimal(text)
	return metadataDecimal{rounded, finiteDecimalText(rounded, text, true)}
}

// floor(log10(abs(value))) without a floating-point intermediate.
func decimalAdjusted(value ledger.Decimal) int64 {
	rational := value.Rat()
	rational.Abs(rational)
	numerator, denominator := rational.Num(), rational.Denom()
	exponent := int64(len(numerator.String()) - len(denominator.String()))
	if exponent >= 0 {
		if numerator.Cmp(new(big.Int).Mul(denominator, decimalPower(exponent))) < 0 {
			exponent--
		}
	} else if new(big.Int).Mul(numerator, decimalPower(-exponent)).Cmp(denominator) < 0 {
		exponent--
	}
	return exponent
}

// Both unary minus and abs are contextual Decimal operations; either clears
// the sign of a zero. Integer unary minus keeps its unbounded exact semantics.
func contextualUnary(arg typedExpr, negate bool, row Row, rows []Row) (any, error) {
	value, err := arg.eval(row, rows)
	if err != nil || value == nil {
		return nil, err
	}
	text, err := decimalOperandText(arg, row, rows)
	if err != nil {
		return nil, err
	}
	quantum, err := decimalQuantum(text)
	if err != nil {
		return nil, err
	}
	number := value.(ledger.Decimal)
	if negate || number.Sign() < 0 {
		number = number.Neg()
	}
	return contextualDecimal(number, quantum, false), nil
}

func withDecimalCalculation(expr typedExpr, calculate func(Row, []Row) (any, error)) typedExpr {
	expr.value = func(row Row, rows []Row) (any, error) {
		value, err := calculate(row, rows)
		if err != nil || value == nil {
			return nil, err
		}
		return value.(metadataDecimal).Value, nil
	}
	expr.text = func(row Row, rows []Row) (any, error) {
		value, err := calculate(row, rows)
		if err != nil || value == nil {
			return nil, err
		}
		return value.(metadataDecimal).Text, nil
	}
	return expr
}

// SUM rounds each addition in source order, rather than rounding an exact
// accumulated rational only once. NULL does not alter the accumulator.
func contextualSum(arg typedExpr, rows []Row) (any, error) {
	result := metadataDecimal{ledger.Zero(), "0"}
	quantum := int64(0)
	for _, row := range rows {
		value, err := arg.eval(row, []Row{row})
		if err != nil {
			return nil, err
		}
		if value == nil {
			continue
		}
		text, err := decimalOperandText(arg, row, []Row{row})
		if err != nil {
			return nil, err
		}
		operandQuantum, err := decimalQuantum(text)
		if err != nil {
			return nil, err
		}
		negative := strings.HasPrefix(result.Text, "-") && strings.HasPrefix(text, "-")
		result = contextualDecimal(result.Value.Add(value.(ledger.Decimal)), min(quantum, operandQuantum), negative)
		quantum, err = decimalQuantum(result.Text)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
