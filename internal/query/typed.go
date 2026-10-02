// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package query

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"unicode/utf8"

	"orangecount/internal/ledger"
	"orangecount/internal/presentation"
)

// PresentedDecimal is the same exact/display contract used by report views.
type PresentedDecimal = presentation.Decimal

// TypedError identifies the execution/compilation category independently of
// the human message. Source ranges are added with the native query protocol.
type TypedError struct {
	Code    string     `json:"code"`
	Message string     `json:"message"`
	Phase   string     `json:"phase,omitempty"`
	Range   *ByteRange `json:"range,omitempty"`
}

func (e TypedError) Error() string { return e.Message }

func typedFailure(code, format string, args ...any) error {
	return TypedError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// Column identifies an output by position; titles may repeat.
type Column struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

// Cell carries its actual type. Null has no value, independently of schema.
type Cell struct {
	Type  string `json:"type"`
	Value any    `json:"value"`
}

// MarshalJSON distinguishes null from empty strings, false and empty inventories.
func (c Cell) MarshalJSON() ([]byte, error) {
	if c.Type == "null" {
		return json.Marshal(struct {
			Type string `json:"type"`
		}{Type: "null"})
	}
	type wire Cell
	return json.Marshal(wire(c))
}

// TypedResult never identifies a value by its display title.
type TypedResult struct {
	Contract string   `json:"contract"`
	Columns  []Column `json:"columns"`
	Rows     [][]Cell `json:"rows"`
}

// ParseStandard shares the grammar with Parse, with the standard default
// postings source. Parse remains the compatibility entry for older callers.
func ParseStandard(text string) (Query, error) {
	p := queryParser{tokens: lexQuery(text, true), defaultPostings: true, inputBytes: len(text)}
	return p.parse()
}

// EvaluateTyped compiles a read against declared source types, then uses the
// common execution pipeline. Output IDs prevent title-based data loss.
func EvaluateTyped(text string, evaluation ledger.Evaluation) (TypedResult, error) {
	return evaluateTyped(text, evaluation, 0, nil)
}

func evaluateTyped(text string, evaluation ledger.Evaluation, rowCeiling int, budget *executionBudget) (TypedResult, error) {
	q, err := parseTyped(text, budget)
	if err != nil {
		if budget != nil {
			return TypedResult{}, err
		}
		return TypedResult{}, locatedQueryError(err, nil, "parse")
	}
	result := TypedResult{Contract: "beanquery-0.2.0-typed-v1", Columns: make([]Column, 0), Rows: make([][]Cell, 0)}
	if canonicalTable(q.From) != "postings" {
		code := "unsupported-capability"
		if canonicalTable(q.From) == "" {
			code = "unknown-table"
		}
		return TypedResult{}, locatedQueryError(typedFailure(code, "unsupported typed source %q", q.From), q.fromSpan, "compile")
	}
	if err := resolveTypedAliases(&q); err != nil {
		return TypedResult{}, err
	}
	if q.Where != nil {
		compiled, err := compileTyped(q.Where)
		if err != nil {
			return TypedResult{}, err
		}
		q.Where = typedExpr{kind: "bool", grouped: compiled.grouped, value: func(row Row, rows []Row) (any, error) {
			value, err := compiled.eval(row, rows)
			if err != nil {
				return nil, err
			}
			return typedTruth(value)
		}}
	}
	for i, expr := range q.GroupBy {
		compiled, err := compileTyped(expr)
		if err != nil {
			return TypedResult{}, err
		}
		q.GroupBy[i] = compiled
	}
	for i, item := range q.OrderBy {
		compiled, err := compileTyped(item.Expr)
		if err != nil {
			return TypedResult{}, err
		}
		q.OrderBy[i].Expr = compiled
	}
	for i, item := range q.Select {
		compiled, err := compileTyped(item.Expr)
		if err != nil {
			return TypedResult{}, err
		}
		title := item.Alias
		if title == "" {
			title = standardExprName(item.Expr)
		}
		id := fmt.Sprintf("c%d", i)
		result.Columns = append(result.Columns, Column{ID: id, Name: title, Type: compiled.kind, Nullable: compiled.nullable})
		q.Select[i].Expr = compiled
		q.Select[i].Alias = id
	}
	hasAggregate := false
	for _, item := range q.Select {
		hasAggregate = hasAggregate || item.Expr.aggregate()
	}
	if len(q.GroupBy) == 0 && hasAggregate {
		// The standard compiler groups mixed targets by every nonaggregate
		// projection. Legacy callers retain their explicit grouping behavior.
		for _, item := range q.Select {
			if !item.Expr.aggregate() {
				q.GroupBy = append(q.GroupBy, item.Expr)
			}
		}
	}
	if q.LimitSpecified && q.Limit == 0 {
		return result, nil
	}
	if rowCeiling > 0 && (q.Limit == 0 || q.Limit > rowCeiling) {
		q.Limit = rowCeiling
	}
	rows, err := postingRowsWithBudget(evaluation, true, budget)
	if err != nil {
		return TypedResult{}, err
	}
	raw, err := evaluateRowsBounded(q, rows, compareTypedValues, budget)
	if err != nil {
		return TypedResult{}, err
	}
	byteCount := 0
	if budget != nil {
		header, _ := json.Marshal(Execution{Result: result, ReturnedRows: min(len(raw.Rows), budget.limits.Rows), RowLimit: budget.limits.Rows, Truncated: len(raw.Rows) > budget.limits.Rows})
		byteCount = len(header)
	}
	for index, row := range raw.Rows {
		if err := budget.step("encode"); err != nil {
			return TypedResult{}, err
		}
		cells := make([]Cell, len(result.Columns))
		for i, col := range result.Columns {
			value := row[col.ID]
			cell, err := typedCell(col.Type, value)
			if err != nil {
				return TypedResult{}, err
			}
			cells[i] = cell
		}
		if budget != nil && index < budget.limits.Rows {
			encoded, err := json.Marshal(cells)
			if err != nil {
				return TypedResult{}, err
			}
			byteCount += len(encoded)
			if index > 0 {
				byteCount++
			}
			if byteCount > budget.limits.Bytes {
				return TypedResult{}, typedFailure("result-budget-exceeded", "Query result exceeds the byte budget")
			}
		}
		result.Rows = append(result.Rows, cells)
	}
	return result, nil
}

type typedExpr struct {
	name     string
	kind     string
	nullable bool
	grouped  bool
	value    func(Row, []Row) (any, error)
	text     func(Row, []Row) (any, error)
}

func (e typedExpr) String() string                        { return e.name }
func (e typedExpr) aggregate() bool                       { return e.grouped }
func (e typedExpr) eval(row Row, rows []Row) (any, error) { return e.value(row, rows) }

// compileTyped binds types independently of data rows, including empty input.
func compileTyped(expr Expr) (typedExpr, error) {
	compiled, err := compileTypedNode(expr)
	if err != nil {
		return typedExpr{}, locatedQueryError(err, expressionRange(expr), "compile")
	}
	value := compiled.value
	compiled.value = func(row Row, rows []Row) (any, error) {
		if err := rowBudget(row, rows).step(""); err != nil {
			return nil, err
		}
		result, err := value(row, rows)
		if err != nil {
			return nil, locatedQueryError(err, expressionRange(expr), "evaluate")
		}
		if err := rowBudget(row, rows).numericValue(result); err != nil {
			return nil, err
		}
		return result, nil
	}
	if text := compiled.text; text != nil {
		compiled.text = func(row Row, rows []Row) (any, error) {
			if err := rowBudget(row, rows).step(""); err != nil {
				return nil, err
			}
			value, err := text(row, rows)
			return value, locatedQueryError(err, expressionRange(expr), "evaluate")
		}
	}
	return compiled, nil
}

func compileTypedNode(expr Expr) (typedExpr, error) {
	result := typedExpr{name: expr.String(), value: expr.eval}
	switch node := expr.(type) {
	case fieldExpr:
		switch strings.ToLower(node.name) {
		case "account":
			result.kind = "string"
		case "currency", "narration", "flag":
			result.kind = "string"
		case "date":
			result.kind = "date"
		case "year", "month", "day":
			result.kind = "integer"
		case "number":
			result.kind = "decimal"
			result.text = func(row Row, _ []Row) (any, error) {
				number, ok := row["_number_source"].(ledger.Number)
				if !ok {
					return nil, nil
				}
				return metadataDecimalValue(number).Text, nil
			}
		case "position":
			result.kind = "position"
		case "weight":
			result.kind = "amount"
		case "filename":
			result.kind, result.nullable = "string", true
		case "lineno":
			result.kind, result.nullable = "integer", true
		default:
			return typedExpr{}, typedFailure("unknown-column", "unknown typed column %q", node.name)
		}
	case literalExpr:
		switch node.value.(type) {
		case nil:
			result.kind, result.nullable = "null", true
		case bool:
			result.kind = "bool"
		case string:
			result.kind = "string"
		case ledger.Decimal:
			result.kind = "integer"
			if strings.ContainsAny(node.display, ".eE") {
				result.kind = "decimal"
			}
			result.text = func(Row, []Row) (any, error) {
				return metadataDecimalValue(ledger.Number{Raw: node.display, Rat: node.value.(ledger.Decimal).Rat()}).Text, nil
			}
		case ledger.Date:
			result.kind = "date"
			result.value = func(Row, []Row) (any, error) { return node.value.(ledger.Date).Raw, nil }
		default:
			return typedExpr{}, typedFailure("unsupported-capability", "unsupported typed literal %q", node.display)
		}
	case parenExpr:
		inner, err := compileTyped(node.inner)
		if err != nil {
			return typedExpr{}, err
		}
		result.kind, result.nullable, result.grouped, result.value = inner.kind, inner.nullable, inner.grouped, inner.value
		result.text = inner.text
	case unaryExpr:
		inner, err := compileTyped(node.inner)
		if err != nil {
			return typedExpr{}, err
		}
		result.grouped = inner.grouped
		if node.operator == "not" || node.operator == "is null" || node.operator == "is not null" {
			result.kind = "bool"
			result.value = func(row Row, rows []Row) (any, error) {
				value, err := inner.eval(row, rows)
				if err != nil {
					return nil, err
				}
				if node.operator == "is null" {
					return value == nil, nil
				}
				if node.operator == "is not null" {
					return value != nil, nil
				}
				truth, err := typedTruth(value)
				return !truth, err
			}
		} else {
			if !numericKind(inner.kind) {
				return typedExpr{}, typedFailure("type-mismatch", "unary minus requires a numeric operand")
			}
			result.kind, result.nullable = inner.kind, inner.nullable
			result.value = func(row Row, rows []Row) (any, error) {
				value, err := inner.eval(row, rows)
				if err != nil || value == nil {
					return nil, err
				}
				return value.(ledger.Decimal).Neg(), nil
			}
			if inner.text != nil {
				result.text = func(row Row, rows []Row) (any, error) {
					value, err := result.value(row, rows)
					if err != nil || value == nil {
						return nil, err
					}
					text, err := inner.text(row, rows)
					if err != nil || text == nil {
						return nil, err
					}
					return finiteDecimalText(value.(ledger.Decimal), text.(string), false), nil
				}
			}
		}
		if result.kind == "decimal" {
			result = withDecimalCalculation(result, func(row Row, rows []Row) (any, error) {
				return contextualUnary(inner, true, row, rows)
			})
		}
	case binaryExpr:
		left, err := compileTyped(node.left)
		if err != nil {
			return typedExpr{}, err
		}
		right, err := compileTyped(node.right)
		if err != nil {
			return typedExpr{}, err
		}
		if node.operator != "and" && node.operator != "or" {
			left, err = bindMetadataOperand(left, right.kind)
			if err != nil {
				return typedExpr{}, err
			}
			right, err = bindMetadataOperand(right, left.kind)
			if err != nil {
				return typedExpr{}, err
			}
		}
		result.kind = "bool"
		result.nullable = left.nullable || right.nullable
		result.grouped = left.grouped || right.grouped
		switch node.operator {
		case "+", "-", "*", "/":
			if !numericKind(left.kind) || !numericKind(right.kind) {
				return typedExpr{}, typedFailure("type-mismatch", "arithmetic requires numeric types")
			}
			result.kind = "integer"
			if left.kind == "decimal" || right.kind == "decimal" {
				result.kind = "decimal"
			}
			if node.operator == "/" && result.kind == "integer" {
				return typedExpr{}, typedFailure("unsupported-capability", "integer division yields an unsupported float type")
			}
			if node.operator == "/" {
				result.nullable = true
			}
		case "and", "or":
			// These operators accept truth values of any supported source type.
		case "~":
			if left.kind != "string" || right.kind != "string" {
				return typedExpr{}, typedFailure("type-mismatch", "regular expression requires string types")
			}
		default:
			if left.kind == "bool" || right.kind == "bool" || left.kind == "null" || right.kind == "null" {
				return typedExpr{}, typedFailure("type-mismatch", "no comparison signature for %s and %s", left.kind, right.kind)
			}
			if left.kind == "object" && right.kind == "object" {
				return typedExpr{}, typedFailure("type-mismatch", "comparison requires a statically typed operand")
			}
			if compoundKind(left.kind) || compoundKind(right.kind) {
				return typedExpr{}, typedFailure("type-mismatch", "no comparison signature for %s and %s", left.kind, right.kind)
			}
			if left.kind != right.kind && !(numericKind(left.kind) && numericKind(right.kind)) {
				return typedExpr{}, typedFailure("type-mismatch", "incompatible comparison types %s and %s", left.kind, right.kind)
			}
		}
		result.value = func(row Row, rows []Row) (any, error) {
			lv, err := left.eval(row, rows)
			if err != nil {
				return nil, err
			}
			if node.operator == "and" {
				if lv == nil {
					return nil, nil
				}
				truth, err := typedTruth(lv)
				if err != nil {
					return nil, err
				}
				if !truth {
					return false, nil
				}
			} else if node.operator == "or" {
				truth, err := typedTruth(lv)
				if err != nil {
					return nil, err
				}
				if truth {
					return true, nil
				}
			}
			rv, err := right.eval(row, rows)
			if err != nil {
				return nil, err
			}
			if node.operator == "and" {
				if rv == nil {
					return nil, nil
				}
				return typedTruth(rv)
			}
			if node.operator == "or" {
				truth, err := typedTruth(rv)
				if err != nil {
					return nil, err
				}
				if truth {
					return true, nil
				}
				if lv == nil || rv == nil {
					return nil, nil
				}
				return false, nil
			}
			if lv == nil || rv == nil {
				return nil, nil
			}
			if node.operator == "/" && rv.(ledger.Decimal).IsZero() {
				return nil, nil
			}
			if (left.kind == "string" || left.kind == "date") && node.operator != "~" {
				comparison := strings.Compare(lv.(string), rv.(string))
				switch node.operator {
				case "=":
					return comparison == 0, nil
				case "!=", "<>":
					return comparison != 0, nil
				case "<":
					return comparison < 0, nil
				case "<=":
					return comparison <= 0, nil
				case ">":
					return comparison > 0, nil
				case ">=":
					return comparison >= 0, nil
				}
			}
			if node.operator == "~" {
				return boundedRegex(row, rows, lv, rv)
			}
			return node.evalComparison(lv, rv)
		}
		if result.kind == "decimal" {
			exact := result.value
			calculate := func(row Row, rows []Row) (any, error) {
				value, err := exact(row, rows)
				if err != nil || value == nil {
					return nil, err
				}
				return decimalArithmeticResult(node.operator, left, right, value.(ledger.Decimal), row, rows)
			}
			result = withDecimalCalculation(result, calculate)
		}
	case functionExpr:
		switch node.name {
		case "abs", "length", "count", "sum", "first", "last", "min", "max", "year", "month", "day", "meta", "entry_meta", "any_meta", "units", "cost", "number", "currency", "str", "decimal", "date", "bool", "int":
		default:
			return typedExpr{}, typedFailure("unknown-function", "unsupported typed function %q", node.name)
		}
		if node.name == "count" && len(node.args) == 1 {
			if _, ok := node.args[0].(starExpr); ok {
				result.kind, result.grouped = "integer", true
				result.value = func(_ Row, rows []Row) (any, error) { return node.evalCount(rows) }
				return result, nil
			}
		}
		if len(node.args) != 1 {
			if node.name == "date" && len(node.args) == 3 {
				return compileDateConstructor(node)
			}
			return typedExpr{}, typedFailure("type-mismatch", "no typed signature for %s with %d arguments", node.name, len(node.args))
		}
		arg, err := compileTyped(node.args[0])
		if err != nil {
			return typedExpr{}, err
		}
		result.nullable, result.grouped = arg.nullable, arg.grouped
		switch node.name {
		case "year", "month", "day":
			return compileDatePart(node, arg)
		case "int":
			if !numericKind(arg.kind) && arg.kind != "bool" && arg.kind != "string" && arg.kind != "object" && arg.kind != "null" {
				return typedExpr{}, typedFailure("type-mismatch", "no typed signature for int(%s)", arg.kind)
			}
			result.kind, result.nullable = "integer", true
			result.value = func(row Row, rows []Row) (any, error) {
				value, err := arg.eval(row, rows)
				if err != nil {
					return nil, err
				}
				if text, ok := value.(string); ok {
					if err := rowBudget(row, rows).decimalText(text); err != nil {
						return nil, err
					}
				}
				return castInteger(value)
			}
		case "bool":
			result.kind = "bool"
			result.value = func(row Row, rows []Row) (any, error) {
				value, err := arg.eval(row, rows)
				if err != nil || value == nil {
					return nil, err
				}
				return typedTruth(value)
			}
		case "date":
			if arg.kind != "date" && arg.kind != "string" && arg.kind != "object" && arg.kind != "null" {
				return typedExpr{}, typedFailure("type-mismatch", "no typed signature for date(%s)", arg.kind)
			}
			operand := arg
			operand.kind = "object"
			cast, err := bindMetadataOperand(operand, "date")
			if err != nil {
				return typedExpr{}, err
			}
			cast.name = node.String()
			return cast, nil
		case "decimal":
			if !numericKind(arg.kind) && arg.kind != "bool" && arg.kind != "string" && arg.kind != "object" && arg.kind != "null" {
				return typedExpr{}, typedFailure("type-mismatch", "no typed signature for decimal(%s)", arg.kind)
			}
			operand := arg
			operand.kind = "object"
			cast, err := bindMetadataOperand(operand, "decimal")
			if err != nil {
				return typedExpr{}, err
			}
			cast.name = node.String()

			return cast, nil
		case "str":
			if arg.text != nil {
				result.kind, result.value = "string", arg.text
				return result, nil
			}
			if arg.kind == "integer" {
				result.kind = "string"
				result.value = func(row Row, rows []Row) (any, error) {
					value, err := arg.eval(row, rows)
					if err != nil || value == nil {
						return nil, err
					}
					return value.(ledger.Decimal).String(), nil
				}
				return result, nil
			}
			operand := arg
			operand.kind = "object"
			cast, err := bindMetadataOperand(operand, "string")
			if err != nil {
				return typedExpr{}, err
			}
			cast.name = node.String()
			return cast, nil
		case "number", "currency":
			if arg.kind != "amount" {
				return typedExpr{}, typedFailure("type-mismatch", "no typed signature for %s(%s)", node.name, arg.kind)
			}
			result.kind = "decimal"
			if node.name == "currency" {
				result.kind = "string"
			}
			result.value = func(row Row, rows []Row) (any, error) {
				value, err := arg.eval(row, rows)
				if err != nil || value == nil {
					return nil, err
				}
				amount := value.(amountValue)
				if node.name == "number" {
					return amount.Number, nil
				}
				return amount.Currency, nil
			}
			if node.name == "number" {
				result.text = func(row Row, rows []Row) (any, error) {
					value, err := arg.eval(row, rows)
					if err != nil || value == nil {
						return nil, err
					}
					amount := value.(amountValue)
					if amount.Text != "" {
						return amount.Text, nil
					}
					return amount.Number.String(), nil
				}
			}
		case "units", "cost":
			if arg.kind != "position" && arg.kind != "inventory" {
				return typedExpr{}, typedFailure("type-mismatch", "no typed signature for %s(%s)", node.name, arg.kind)
			}
			result.kind = "amount"
			if arg.kind == "inventory" {
				result.kind = "inventory"
			}
			result.value = func(row Row, rows []Row) (any, error) {
				value, err := arg.eval(row, rows)
				if err != nil || value == nil {
					return nil, err
				}
				if arg.kind == "inventory" {
					reduced := inventoryAccumulator{}
					for _, position := range value.(inventoryValue) {
						amount := position.Units
						if node.name == "cost" {
							amount = positionCost(position)
						}
						reduced.add(positionValue{Units: amount})
					}
					return reduced.value(), nil
				}
				position := value.(positionValue)
				if node.name == "cost" {
					return positionCost(position), nil
				}
				return position.Units, nil
			}
		case "meta", "entry_meta", "any_meta":
			if arg.kind != "string" {
				return typedExpr{}, typedFailure("type-mismatch", "metadata key requires string")
			}
			result.kind, result.nullable = "object", true
			result.value = func(row Row, rows []Row) (any, error) {
				key, err := arg.eval(row, rows)
				if err != nil || key == nil {
					return nil, err
				}
				if node.name != "entry_meta" {
					value, found, err := lookupMetadata(row["_posting_meta"], key.(string))
					if err != nil || found || node.name == "meta" {
						return value, err
					}
				}
				value, _, err := lookupMetadata(row["_entry_meta"], key.(string))
				return value, err
			}
		case "first", "last", "min", "max":
			if arg.grouped {
				return typedExpr{}, typedFailure("type-mismatch", "nested aggregate is not supported")
			}
			result.kind, result.grouped, result.nullable = arg.kind, true, true
			result.value = func(_ Row, rows []Row) (any, error) {
				selected, err := selectAggregateValue(node.name, arg, rows)
				return selected.value, err
			}
			if arg.kind == "decimal" {
				result.text = func(_ Row, rows []Row) (any, error) {
					selected, err := selectAggregateValue(node.name, arg, rows)
					if err != nil || selected.value == nil {
						return nil, err
					}
					return decimalOperandText(arg, selected.row, []Row{selected.row})
				}
			}
		case "sum":
			if (arg.kind == "position" || arg.kind == "amount") && !arg.grouped {
				result.kind, result.grouped, result.nullable = "inventory", true, false
				result.value = func(_ Row, rows []Row) (any, error) { return sumCompoundValues(arg, rows) }
				return result, nil
			}
			if !numericKind(arg.kind) || arg.grouped {
				return typedExpr{}, typedFailure("type-mismatch", "no typed signature for sum(%s)", arg.kind)
			}
			result.kind, result.grouped, result.nullable = arg.kind, true, false
			bound := functionExpr{name: "sum", args: []Expr{arg}}
			result.value = bound.evalAggregate
			if arg.kind == "decimal" {
				result = withDecimalCalculation(result, func(_ Row, rows []Row) (any, error) {
					return contextualSum(arg, rows)
				})
			}
		case "count":
			if arg.grouped {
				return typedExpr{}, typedFailure("type-mismatch", "nested aggregate is not supported")
			}
			result.kind, result.grouped, result.nullable = "integer", true, false
			bound := functionExpr{name: "count", args: []Expr{arg}}
			result.value = func(_ Row, rows []Row) (any, error) { return bound.evalCount(rows) }
		case "abs":
			if arg.kind != "decimal" {
				return typedExpr{}, typedFailure("type-mismatch", "no typed signature for abs(%s)", arg.kind)
			}
			result.kind = arg.kind
			result = withDecimalCalculation(result, func(row Row, rows []Row) (any, error) {
				return contextualUnary(arg, false, row, rows)
			})
		case "length":
			if arg.kind != "string" {
				return typedExpr{}, typedFailure("type-mismatch", "no typed signature for length(%s)", arg.kind)
			}
			result.kind = "integer"
			result.value = func(row Row, rows []Row) (any, error) {
				v, err := arg.eval(row, rows)
				if err != nil || v == nil {
					return nil, err
				}
				return ledger.NewDecimal(new(big.Rat).SetInt64(int64(utf8.RuneCountInString(v.(string))))), nil
			}
		default:
			return typedExpr{}, typedFailure("unknown-function", "unsupported typed function %q", node.name)
		}
	default:
		return typedExpr{}, typedFailure("unsupported-capability", "unsupported typed expression %q", expr.String())
	}
	return result, nil
}

func typedCell(kind string, value any) (Cell, error) {
	if value == nil {
		return Cell{Type: "null"}, nil
	}
	switch kind {
	case "object":
		switch v := value.(type) {
		case ledger.Decimal:
			return typedCell("decimal", v)
		case string:
			return typedCell("string", v)
		case bool:
			return typedCell("bool", v)
		case ledger.Date:
			return typedCell("date", v.Raw)
		case amountValue:
			return typedCell("amount", v)
		case metadataAmount:
			return typedCell("amount", v.Value)
		case metadataInteger:
			return typedCell("integer", v.Value)
		case metadataDecimal:
			return typedCell("decimal", v.Value)
		default:
			return Cell{}, typedFailure("unsupported-capability", "unsupported metadata result type %T", value)
		}
	case "decimal":
		n, ok := value.(ledger.Decimal)
		if !ok {
			return Cell{}, typedFailure("type-mismatch", "decimal result has type %T", value)
		}
		return Cell{Type: kind, Value: presentation.FormatDecimal(n)}, nil
	case "integer":
		n, ok := value.(ledger.Decimal)
		if !ok || !n.Rat().IsInt() {
			return Cell{}, typedFailure("type-mismatch", "integer result has type %T or fractional value", value)
		}
		return Cell{Type: kind, Value: n.Rat().Num().String()}, nil
	case "string":
		if _, ok := value.(string); !ok {
			return Cell{}, typedFailure("type-mismatch", "string result has type %T", value)
		}
		return Cell{Type: kind, Value: value}, nil
	case "date":
		if _, ok := value.(string); !ok {
			return Cell{}, typedFailure("type-mismatch", "date result has type %T", value)
		}
		return Cell{Type: kind, Value: value}, nil
	case "bool":
		if _, ok := value.(bool); !ok {
			return Cell{}, typedFailure("type-mismatch", "bool result has type %T", value)
		}
		return Cell{Type: kind, Value: value}, nil
	case "position":
		position, ok := value.(positionValue)
		if !ok {
			return Cell{}, typedFailure("type-mismatch", "position result has type %T", value)
		}
		return Cell{Type: kind, Value: presentPosition(position)}, nil
	case "amount":
		amount, ok := value.(amountValue)
		if !ok {
			return Cell{}, typedFailure("type-mismatch", "amount result has type %T", value)
		}
		return Cell{Type: kind, Value: presentAmount(amount)}, nil
	case "inventory":
		inventory, ok := value.(inventoryValue)
		if !ok {
			return Cell{}, typedFailure("type-mismatch", "inventory result has type %T", value)
		}
		positions := make([]PresentedPosition, 0, len(inventory))
		for _, position := range canonicalInventory(inventory) {
			positions = append(positions, presentPosition(position))
		}
		return Cell{Type: kind, Value: positions}, nil
	default:
		return Cell{}, typedFailure("unsupported-capability", "unsupported result type %q", kind)
	}
}

type queryMetadata struct {
	Values        []ledger.Metadata
	Filename      string
	Line          int
	HasTolerances bool
}

type metadataInteger struct{ Value ledger.Decimal }

func lookupMetadata(raw any, key string) (any, bool, error) {
	scope, _ := raw.(queryMetadata)
	if key == "__tolerances__" && scope.HasTolerances {
		return nil, true, typedFailure("unsupported-capability", "system tolerance map has no typed result representation")
	}
	meta := scope.Values
	for i := len(meta) - 1; i >= 0; i-- {
		if meta[i].Key != key {
			continue
		}
		v := meta[i].Value
		switch v.Kind {
		case ledger.ValueNull:
			return nil, true, nil
		case ledger.ValueString, ledger.ValueAccount, ledger.ValueCurrency, ledger.ValueTag, ledger.ValueLink:
			return v.String, true, nil
		case ledger.ValueNumber:
			return metadataDecimalValue(v.Number), true, nil
		case ledger.ValueBool:
			return v.Bool, true, nil
		case ledger.ValueDate:
			return v.Date, true, nil
		case ledger.ValueAmount:
			number := metadataDecimalValue(v.Amount.Number)
			return metadataAmount{amountValue{Number: number.Value, Currency: v.Amount.Currency, Text: number.Text}, number.Text + " " + v.Amount.Currency}, true, nil
		default:
			return nil, true, typedFailure("unsupported-capability", "unsupported metadata value kind %s", v.Kind)
		}
	}
	if scope.Filename != "" && scope.Line > 0 {
		switch key {
		case "filename":
			return scope.Filename, true, nil
		case "lineno":
			return metadataInteger{ledger.NewDecimal(new(big.Rat).SetInt64(int64(scope.Line)))}, true, nil
		}
	}
	return nil, false, nil
}

func standardExprName(expr Expr) string {
	switch node := expr.(type) {
	case parenExpr:
		return "(" + standardExprName(node.inner) + ")"
	case literalExpr:
		if value, ok := node.value.(string); ok {
			return "'" + strings.NewReplacer("\\", "\\\\", "'", "\\'", "\n", "\\n", "\r", "\\r", "\t", "\\t").Replace(value) + "'"
		}
	case functionExpr:
		args := make([]string, len(node.args))
		for i, arg := range node.args {
			args[i] = standardExprName(arg)
		}
		return node.name + "(" + strings.Join(args, ", ") + ")"
	case binaryExpr:
		op := node.operator
		if op == "and" || op == "or" {
			op = strings.ToUpper(op)
		}
		return standardExprName(node.left) + " " + op + " " + standardExprName(node.right)
	case unaryExpr:
		if node.operator == "is null" || node.operator == "is not null" {
			return standardExprName(node.inner) + " " + strings.ToUpper(node.operator)
		}
		if node.operator == "not" {
			return "NOT " + standardExprName(node.inner)
		}
		return "-" + standardExprName(node.inner)
	}
	return expr.String()
}

func numericKind(kind string) bool { return kind == "integer" || kind == "decimal" }

func compoundKind(kind string) bool {
	return kind == "amount" || kind == "position" || kind == "inventory"
}

func compareTypedValues(left, right any) (int, error) {
	left, right = unwrapMetadataNumber(left), unwrapMetadataNumber(right)
	if left == nil && right == nil {
		return 0, nil
	}
	if left == nil {
		return -1, nil
	}
	if right == nil {
		return 1, nil
	}
	switch l := left.(type) {
	case ledger.Decimal:
		if r, ok := right.(ledger.Decimal); ok {
			return l.Cmp(r), nil
		}
	case string:
		if r, ok := right.(string); ok {
			return strings.Compare(l, r), nil
		}
	case ledger.Date:
		if r, ok := right.(ledger.Date); ok {
			return strings.Compare(l.Raw, r.Raw), nil
		}
	case bool:
		if r, ok := right.(bool); ok {
			if l == r {
				return 0, nil
			}
			if !l {
				return -1, nil
			}
			return 1, nil
		}
	case amountValue:
		if r, ok := right.(amountValue); ok {
			return compareAmountOrder(l, r), nil
		}
	case positionValue:
		if r, ok := right.(positionValue); ok {
			return comparePositionOrder(l, r), nil
		}
	case inventoryValue:
		if r, ok := right.(inventoryValue); ok {
			return compareInventoryOrder(l, r), nil
		}
	}
	return 0, typedFailure("type-mismatch", "incompatible sort values %T and %T", left, right)
}

func resolveTypedAliases(q *Query) error {
	aliases := map[string]Expr{}
	counts := map[string]int{}
	for _, item := range q.Select {
		if item.Alias != "" {
			key := strings.ToLower(item.Alias)
			aliases[key] = item.Expr
			counts[key]++
		}
	}
	resolve := func(expr Expr) (Expr, error) {
		if field, ok := expr.(fieldExpr); ok {
			key := strings.ToLower(field.name)
			if counts[key] > 1 {
				return nil, locatedQueryError(typedFailure("ambiguous-column", "ambiguous output alias %q", field.name), field.span, "compile")
			}
			if replacement, ok := aliases[key]; ok {
				return replacement, nil
			}
		}
		return expr, nil
	}
	for i, expr := range q.GroupBy {
		resolved, err := resolve(expr)
		if err != nil {
			return err
		}
		q.GroupBy[i] = resolved
	}
	for i, item := range q.OrderBy {
		resolved, err := resolve(item.Expr)
		if err != nil {
			return err
		}
		q.OrderBy[i].Expr = resolved
	}
	return nil
}
