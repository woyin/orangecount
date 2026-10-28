// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package query

import (
	"context"
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

const CurrentContract = "beanquery-0.2.0-typed-v3"
const PreviousContract = "beanquery-0.2.0-typed-v2"
const LegacyContract = "beanquery-0.2.0-typed-v1"

// TypedResult never identifies a value by its display title.
type TypedResult struct {
	Contract string   `json:"contract"`
	Columns  []Column `json:"columns"`
	Rows     [][]Cell `json:"rows"`
}

// ParseStandard shares the grammar with Parse, with the standard default
// postings source. Parse remains the compatibility entry for older callers.
func ParseStandard(text string) (Query, error) {
	p := queryParser{tokens: lexQuery(text, true), defaultPostings: true, inputBytes: len(text), inputText: text}
	return p.parse()
}

// EvaluateTyped compiles a read against declared source types, then uses the
// common execution pipeline. Output IDs prevent title-based data loss.
func EvaluateTyped(text string, evaluation ledger.Evaluation) (TypedResult, error) {
	execution, err := ExecuteTyped(context.Background(), text, evaluation, ExecutionLimits{})
	if err != nil {
		return TypedResult{}, err
	}
	if execution.Truncated {
		return TypedResult{}, typedFailure("result-budget-exceeded", "Typed result exceeds the row budget; use bounded execution for explicit truncation")
	}
	return execution.Result, nil
}

func evaluateTyped(text string, evaluation ledger.Evaluation, rowCeiling int, budget *executionBudget) (TypedResult, error) {
	q, err := parseTyped(text, budget)
	if err != nil {
		if budget != nil {
			return TypedResult{}, err
		}
		return TypedResult{}, locatedQueryError(err, nil, "parse")
	}
	plan, err := executeTypedPlan(q, evaluation, rowCeiling, budget, 0)
	if err != nil {
		return TypedResult{}, err
	}
	return encodeTypedPlan(plan, budget)
}

// typedPlan is a compiled typed query before cell encoding: the bound output
// columns plus the raw typed rows keyed by column ID. A derived table consumes
// this shape directly, which is why values stay unencoded inside the plan.
type typedPlan struct {
	columns []Column
	rows    []Row
}

// typedDerivedTable describes the columns an enclosing query sees when it
// reads a derived table. Duplicate titles keep their own positions, so a star
// select stays lossless;
// a bare name reference resolves to the last position carrying that title,
// matching the reference's alias map.
type typedDerivedTable struct {
	names    []string
	kinds    []string
	nullable []bool
	ids      []string
	byName   map[string]int
	byID     map[string]int
}

func derivedFromPlan(plan typedPlan) *typedDerivedTable {
	derived := &typedDerivedTable{byName: map[string]int{}, byID: map[string]int{}}
	for i, column := range plan.columns {
		derived.names = append(derived.names, column.Name)
		derived.kinds = append(derived.kinds, column.Type)
		derived.nullable = append(derived.nullable, column.Nullable)
		derived.ids = append(derived.ids, column.ID)
		derived.byName[strings.ToLower(column.Name)] = i
		derived.byID[column.ID] = i
	}
	return derived
}

// expandDerivedWildcard rewrites `SELECT *` over a derived table into one
// positional read per inner column, in the inner projection order, so the
// schema survives an empty inner result and duplicate inner titles stay
// distinguishable.
func expandDerivedWildcard(q *Query, derived *typedDerivedTable) {
	selectItems := make([]SelectItem, 0, len(q.Select))
	for _, item := range q.Select {
		if _, ok := item.Expr.(starExpr); !ok {
			selectItems = append(selectItems, item)
			continue
		}
		for i, name := range derived.names {
			selectItems = append(selectItems, SelectItem{Expr: derivedColumnExpr{id: derived.ids[i], name: name}, sourceTitle: name})
		}
	}
	q.Select = selectItems
}

// derivedColumnExpr addresses one derived column by position.
type derivedColumnExpr struct {
	id   string
	name string
}

func (e derivedColumnExpr) String() string                     { return e.name }
func (e derivedColumnExpr) aggregate() bool                    { return false }
func (e derivedColumnExpr) eval(row Row, _ []Row) (any, error) { return row[e.id], nil }

// executeTypedPlan compiles one SELECT against its declared source and runs
// the bounded execution pipeline. A derived table executes its inner select
// first and hands its output columns and typed rows to this level.
func executeTypedPlan(q Query, evaluation ledger.Evaluation, rowCeiling int, budget *executionBudget, depth int) (typedPlan, error) {
	if depth > maxDerivedTableDepth {
		return typedPlan{}, typedFailure("complexity-budget-exceeded", "Derived tables exceed the maximum depth of %d", maxDerivedTableDepth)
	}
	plan := typedPlan{columns: make([]Column, 0)}
	var (
		compiler typedCompiler
		rows     []Row
		table    string
	)
	if q.fromSubquery != nil {
		// The inner select is intentionally unbounded: the enclosing row
		// ceiling exists only to detect truncation of the final result, and
		// applying it inside would drop rows before the outer ORDER BY.
		inner, err := executeTypedPlan(*q.fromSubquery, evaluation, 0, budget, depth+1)
		if err != nil {
			return typedPlan{}, err
		}
		derived := derivedFromPlan(inner)
		compiler = typedCompiler{derived: derived, context: queryContext{evaluation: evaluation}}
		rows = inner.rows
		// Derived rows keep the execution budget so that WHERE, GROUP BY,
		// HAVING, projection and sorting in the enclosing query still count
		// steps and check cancellation per row.
		for _, row := range rows {
			row["_query_execution"] = budget
		}
		expandDerivedWildcard(&q, derived)
	} else {
		table = canonicalTable(q.From)
		if table == "" {
			return typedPlan{}, locatedQueryError(typedFailure("unknown-table", "unknown typed source %q", q.From), q.fromSpan, "compile")
		}
		compiler = typedCompiler{table: table, context: queryContext{evaluation: evaluation}}
		expandStandardWildcard(&q, table)
	}
	if err := resolveTypedAliases(&q); err != nil {
		return typedPlan{}, err
	}
	balanceTargets := []typedExpr{}
	if q.Where != nil {
		compiled, err := compiler.compile(q.Where)
		if err != nil {
			return typedPlan{}, err
		}
		if compiled.grouped {
			return typedPlan{}, locatedQueryError(typedFailure("invalid-group", "aggregates are not allowed in WHERE"), expressionRange(q.Where), "compile")
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
		compiled, err := compiler.compile(expr)
		if err != nil {
			return typedPlan{}, err
		}
		if compiled.grouped || compiled.kind == "inventory" {
			return typedPlan{}, locatedQueryError(typedFailure("invalid-group", "GROUP BY requires a nonaggregate hashable expression"), expressionRange(expr), "compile")
		}
		q.GroupBy[i] = compiled
	}
	if q.Having != nil {
		compiled, err := compiler.compile(q.Having)
		if err != nil {
			return typedPlan{}, err
		}
		if !compiled.grouped {
			return typedPlan{}, locatedQueryError(typedFailure("invalid-group", "the HAVING clause must be an aggregate expression"), expressionRange(q.Having), "compile")
		}
		q.Having = compiled
	}
	for i, item := range q.OrderBy {
		compiled, err := compiler.compile(item.Expr)
		if err != nil {
			return typedPlan{}, err
		}
		if q.fromSubquery == nil && referencesBalance(item.Expr) {
			balanceTargets = append(balanceTargets, compiled)
		}
		q.OrderBy[i].Expr = compiled
	}
	projection := make([]string, len(q.Select))
	titles := make([]string, len(q.Select))
	for i, item := range q.Select {
		projection[i] = item.Expr.String()
		titles[i] = item.Alias
		if titles[i] == "" {
			titles[i] = item.sourceTitle
		}
		if titles[i] == "" {
			titles[i] = standardExprName(item.Expr)
		}
	}
	for i, item := range q.Select {
		compiled, err := compiler.compile(item.Expr)
		if err != nil {
			return typedPlan{}, err
		}
		id := fmt.Sprintf("c%d", i)
		plan.columns = append(plan.columns, Column{ID: id, Name: titles[i], Type: compiled.kind, Nullable: compiled.nullable})
		if q.fromSubquery == nil && referencesBalance(item.Expr) {
			balanceTargets = append(balanceTargets, compiled)
		}
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
	if len(q.GroupBy) > 0 {
		covered := map[string]bool{}
		for _, expr := range q.GroupBy {
			covered[expr.String()] = true
		}
		for _, item := range q.Select {
			if !item.Expr.aggregate() && !covered[item.Expr.String()] {
				return typedPlan{}, locatedQueryError(typedFailure("invalid-group", "all nonaggregates must be covered by GROUP BY: %s", item.Expr.String()), expressionRange(item.Expr), "compile")
			}
		}
	}
	pivotIndexes, err := pivotProjectionIndexes(q, projection, titles)
	if err != nil {
		return typedPlan{}, err
	}
	if q.LimitSpecified && q.Limit == 0 {
		return pivotTypedPlan(q, plan, pivotIndexes, budget)
	}
	// The row ceiling is a pre-reshape implementation budget. Applying it to a
	// PIVOT BY query would clip the rows before their pivot values become
	// columns, silently dropping pivot columns and reporting a complete result;
	// for pivots the ceiling is applied to the final (reshaped) rows instead.
	if len(q.Pivot) == 0 && rowCeiling > 0 && (q.Limit == 0 || q.Limit > rowCeiling) {
		q.Limit = rowCeiling
	}
	if q.fromSubquery == nil {
		rows, err = standardRows(table, evaluation, budget)
		if err != nil {
			return typedPlan{}, err
		}
		rows, err = prepareBalanceStream(&q, rows, balanceTargets, budget)
		if err != nil {
			return typedPlan{}, err
		}
	}
	raw, err := evaluateRowsBounded(q, rows, compareTypedValues, budget)
	if err != nil {
		return typedPlan{}, err
	}
	plan.rows = raw.Rows
	return pivotTypedPlan(q, plan, pivotIndexes, budget)
}

// pivotProjectionIndexes resolves PIVOT BY against the projection the same way
// the reference does: an integer literal names a 1-based projection position,
// a bare name matches a projection title (alias first, expression text
// otherwise, case-insensitively), and any other expression is rejected because
// upstream resolves PIVOT BY through the target list only. It also enforces
// that the two pivot columns differ and that the column spread across output
// columns is one of the GROUP BY keys.
func pivotProjectionIndexes(q Query, projection, titles []string) ([2]int, error) {
	indexes := [2]int{-1, -1}
	if len(q.Pivot) == 0 {
		return indexes, nil
	}
	for j, expr := range q.Pivot {
		if ordinal, ok := projectionOrdinal(expr); ok {
			if ordinal < 1 || ordinal > len(projection) {
				return indexes, locatedQueryError(typedFailure("invalid-pivot", "invalid PIVOT BY column index %d", ordinal), expressionRange(expr), "compile")
			}
			indexes[j] = ordinal - 1
			continue
		}
		field, ok := expr.(fieldExpr)
		if !ok {
			return indexes, locatedQueryError(typedFailure("invalid-pivot", "PIVOT BY column %s is not in the projection", expr.String()), expressionRange(expr), "compile")
		}
		found := -1
		for i, title := range titles {
			if strings.EqualFold(title, field.name) {
				found = i
				break
			}
		}
		if found < 0 {
			return indexes, locatedQueryError(typedFailure("invalid-pivot", "PIVOT BY column %s is not in the projection", expr.String()), expressionRange(expr), "compile")
		}
		indexes[j] = found
	}
	if indexes[0] == indexes[1] {
		return indexes, locatedQueryError(typedFailure("invalid-pivot", "the two PIVOT BY columns cannot be the same column"), expressionRange(q.Pivot[0]), "compile")
	}
	grouped := map[string]bool{}
	for _, expr := range q.GroupBy {
		grouped[expr.String()] = true
	}
	if !grouped[q.Select[indexes[1]].Expr.String()] {
		return indexes, locatedQueryError(typedFailure("invalid-pivot", "the second PIVOT BY column must be a GROUP BY column"), expressionRange(q.Pivot[1]), "compile")
	}
	return indexes, nil
}

// projectionOrdinal reports the integer literal naming a projection position.
func projectionOrdinal(expr Expr) (int, bool) {
	literal, ok := expr.(literalExpr)
	if !ok || strings.ContainsAny(literal.display, ".eE") {
		return 0, false
	}
	number, ok := literal.value.(ledger.Decimal)
	if !ok || !number.Rat().IsInt() {
		return 0, false
	}
	return int(number.Rat().Num().Int64()), true
}

// pivotTypedPlan reshapes a compiled plan for PIVOT BY: the first pivot column
// stays the output row key, the second becomes one output column per distinct
// value, and every other projected column repeats as a measure for each of
// them. Row keys and pivot values are emitted in ascending typed order; a
// combination without a row stays NULL, so measure columns are always
// nullable. A query without PIVOT BY returns its plan unchanged. When every
// projected column is a pivot key there is no measure to repeat, and the
// reference keeps only the row-key column.
func pivotTypedPlan(q Query, plan typedPlan, indexes [2]int, budget *executionBudget) (typedPlan, error) {
	if len(q.Pivot) == 0 {
		return plan, nil
	}
	rowIndex, columnIndex := indexes[0], indexes[1]
	measures := make([]int, 0, len(plan.columns))
	for i := range plan.columns {
		if i != rowIndex && i != columnIndex {
			measures = append(measures, i)
		}
	}
	value := func(row Row, index int) any { return row[plan.columns[index].ID] }
	step := func() error { return budget.step("pivot") }
	rowKeys, columnValues, cells, width, err := pivotCells(plan.rows, rowIndex, columnIndex, measures, value, compareTypedValues, step, budget.pivotCellLimit())
	if err != nil {
		return typedPlan{}, err
	}
	pivoted := typedPlan{columns: make([]Column, 0, 1+len(columnValues)*len(measures))}
	pivoted.columns = append(pivoted.columns, Column{
		ID: "c0", Name: plan.columns[rowIndex].Name + "/" + plan.columns[columnIndex].Name,
		Type: plan.columns[rowIndex].Type, Nullable: plan.columns[rowIndex].Nullable,
	})
	if len(measures) == 0 {
		for _, rowKey := range rowKeys {
			pivoted.rows = append(pivoted.rows, Row{pivoted.columns[0].ID: rowKey})
		}
		return pivoted, nil
	}
	for _, columnValue := range columnValues {
		for _, measure := range measures {
			name := formatValue(columnValue)
			if len(measures) > 1 {
				name += "/" + plan.columns[measure].Name
			}
			pivoted.columns = append(pivoted.columns, Column{ID: fmt.Sprintf("c%d", len(pivoted.columns)), Name: name, Type: plan.columns[measure].Type, Nullable: true})
		}
	}
	for i, rowKey := range rowKeys {
		row := Row{pivoted.columns[0].ID: rowKey}
		for j, cell := range cells[i*width : (i+1)*width] {
			row[pivoted.columns[j+1].ID] = cell
		}
		pivoted.rows = append(pivoted.rows, row)
	}
	return pivoted, nil
}

// encodeTypedPlan converts a compiled plan into the wire contract. Encoding
// keeps the previous row, byte and cancellation budgets.
func encodeTypedPlan(plan typedPlan, budget *executionBudget) (TypedResult, error) {
	result := TypedResult{Contract: CurrentContract, Columns: plan.columns, Rows: make([][]Cell, 0)}
	if result.Columns == nil {
		result.Columns = make([]Column, 0)
	}
	byteCount := 0
	if budget != nil {
		header, _ := json.Marshal(Execution{Result: result, ReturnedRows: min(len(plan.rows), budget.limits.Rows), RowLimit: budget.limits.Rows, Truncated: len(plan.rows) > budget.limits.Rows})
		byteCount = len(header)
	}
	for index, row := range plan.rows {
		if err := budget.step("encode"); err != nil {
			return TypedResult{}, err
		}
		cells := make([]Cell, len(result.Columns))
		for i, col := range result.Columns {
			cell, err := encodeTypedCell(col.Type, row[col.ID], 0, budget)
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
	span     *ByteRange
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
type typedCompiler struct {
	table   string
	derived *typedDerivedTable
	context queryContext
}

func (compiler typedCompiler) compile(expr Expr) (typedExpr, error) {
	compiled, err := compiler.compileNode(expr)
	if err != nil {
		return typedExpr{}, locatedQueryError(err, expressionRange(expr), "compile")
	}
	compiled.span = expressionRange(expr)
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

func (compiler typedCompiler) compileNode(expr Expr) (typedExpr, error) {
	result := typedExpr{name: expr.String(), value: expr.eval}
	switch node := expr.(type) {
	case intervalExpr:
		// interval() is a calendar period value, so it has no scalar type.
		if _, err := compiler.compile(node.text); err != nil {
			return typedExpr{}, err
		}
		result.kind, result.nullable = "interval", false
		result.value = node.eval
	case derivedColumnExpr:
		index, exists := compiler.derived.byID[node.id]
		if !exists {
			return typedExpr{}, typedFailure("unknown-column", "unknown derived column %q", node.name)
		}
		result.kind, result.nullable = compiler.derived.kinds[index], compiler.derived.nullable[index]
		return result, nil
	case fieldExpr:
		if compiler.derived != nil {
			index, exists := compiler.derived.byName[strings.ToLower(node.name)]
			if !exists {
				return typedExpr{}, typedFailure("unknown-column", "unknown column %q in derived table", node.name)
			}
			id := compiler.derived.ids[index]
			result.kind, result.nullable = compiler.derived.kinds[index], compiler.derived.nullable[index]
			result.value = func(row Row, _ []Row) (any, error) { return row[id], nil }
			return result, nil
		}
		if compiler.table != "postings" {
			column, exists := standardTableSchemas[compiler.table][strings.ToLower(node.name)]
			if !exists {
				return typedExpr{}, typedFailure("unknown-column", "unknown typed column %q on %s", node.name, compiler.table)
			}
			result.kind, result.nullable = column.kind, column.nullable
			return result, nil
		}
		switch strings.ToLower(node.name) {
		case "id", "account":
			result.kind = "string"
		case "currency", "narration", "flag", "type", "description":
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
		case "balance":
			result.kind = "inventory"
			result.value = func(row Row, _ []Row) (any, error) {
				if observation, ok := row["_balance_observation"].(*balanceObservation); ok {
					return observation.read(), nil
				}
				return inventoryValue{}, nil
			}
		case "position":
			result.kind = "position"
		case "weight", "price":
			result.kind = "amount"
			result.nullable = node.name == "price"
		case "payee", "posting_flag", "location", "filename":
			result.kind, result.nullable = "string", true
		case "lineno":
			result.kind, result.nullable = "integer", true
		case "tags", "links", "accounts", "other_accounts":
			result.kind = "set<string>"
		case "meta":
			result.kind, result.nullable = "dictionary", true
		case "entry":
			result.kind = "transaction"
		case "cost_number":
			result.kind, result.nullable = "decimal", true
		case "cost_currency", "cost_label":
			result.kind, result.nullable = "string", true
		case "cost_date":
			result.kind, result.nullable = "date", true
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
		inner, err := compiler.compile(node.inner)
		if err != nil {
			return typedExpr{}, err
		}
		result.kind, result.nullable, result.grouped, result.value = inner.kind, inner.nullable, inner.grouped, inner.value
		result.text = inner.text
	case unaryExpr:
		inner, err := compiler.compile(node.inner)
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
		left, err := compiler.compile(node.left)
		if err != nil {
			return typedExpr{}, err
		}
		right, err := compiler.compile(node.right)
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
		case "+", "-", "*", "/", "%":
			switch {
			case node.operator == "+" && left.kind == "string" && right.kind == "string":
				result.kind = "string"
			case node.operator == "+" && left.kind == "interval" && right.kind == "interval",
				node.operator == "-" && left.kind == "interval" && right.kind == "interval":
				result.kind = "interval"
			case node.operator == "+" && left.kind == "date" && right.kind == "interval",
				node.operator == "+" && left.kind == "interval" && right.kind == "date",
				node.operator == "-" && left.kind == "date" && right.kind == "interval":
				result.kind = "date"
			case node.operator == "+" && left.kind == "date" && right.kind == "integer",
				node.operator == "+" && left.kind == "integer" && right.kind == "date",
				node.operator == "-" && left.kind == "date" && right.kind == "integer":
				result.kind = "date"
			case node.operator == "-" && left.kind == "date" && right.kind == "date":
				result.kind = "integer"
			case numericKind(left.kind) && numericKind(right.kind):
				result.kind = "integer"
				if left.kind == "decimal" || right.kind == "decimal" {
					result.kind = "decimal"
				}
				if left.kind == "interval" && right.kind == "interval" {
					result.kind = "interval"
				}
				if node.operator == "/" {
					// BeanQuery divides integers exactly and returns Decimal.
					result.kind = "decimal"
				}
				if node.operator == "/" || node.operator == "%" {
					result.nullable = true
				}
			default:
				return typedExpr{}, typedFailure("type-mismatch", "no typed signature for %s on %s and %s", node.operator, left.kind, right.kind)
			}
		case "and", "or":
			// These operators accept truth values of any supported source type.
		case "~", "!~", "?~":
			if left.kind != "string" || right.kind != "string" {
				return typedExpr{}, typedFailure("type-mismatch", "regular expression requires string types")
			}
		case "in", "not in":
			if right.kind != "set<string>" && right.kind != "dictionary" && right.kind != "list" {
				return typedExpr{}, typedFailure("type-mismatch", "membership requires a set, dictionary, or list operand")
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
			if (left.kind == "string" || left.kind == "date") && orderingComparison(node.operator) {
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
			switch node.operator {
			case "~":
				return boundedRegex(row, rows, lv, rv)
			case "!~":
				value, err := boundedRegex(row, rows, lv, rv)
				if err != nil || value == nil {
					return nil, err
				}
				return !value.(bool), nil
			case "?~":
				return boundedRegex(row, rows, rv, lv)
			case "in", "not in":
				value, err := typedMembership(lv, rv)
				if err != nil {
					return nil, err
				}
				if node.operator == "not in" {
					value = !value
				}
				return value, nil
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
		if compiled, handled, err := compiler.compileScalar(node); handled {
			return compiled, err
		}
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
				return compiler.compileDateConstructor(node)
			}
			return typedExpr{}, typedFailure("type-mismatch", "no typed signature for %s with %d arguments", node.name, len(node.args))
		}
		arg, err := compiler.compile(node.args[0])
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
			if (arg.kind == "position" || arg.kind == "amount" || arg.kind == "inventory") && !arg.grouped {
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
			switch arg.kind {
			case "decimal":
				result.kind = arg.kind
				result = withDecimalCalculation(result, func(row Row, rows []Row) (any, error) {
					return contextualUnary(arg, false, row, rows)
				})
			case "position", "inventory":
				result.kind = arg.kind
				result.value = func(row Row, rows []Row) (any, error) {
					value, err := arg.eval(row, rows)
					if err != nil || value == nil {
						return nil, err
					}
					absolute := func(amount amountValue) amountValue {
						if amount.Number.Sign() >= 0 {
							return amount
						}
						return amountValue{Number: amount.Number.Neg(), Currency: amount.Currency, Text: ""}
					}
					if position, ok := value.(positionValue); ok {
						position.Units = absolute(position.Units)
						return position, nil
					}
					reduced := inventoryAccumulator{}
					for _, position := range value.(inventoryValue) {
						position.Units = absolute(position.Units)
						reduced.add(position)
					}
					return reduced.value(), nil
				}
			default:
				return typedExpr{}, typedFailure("type-mismatch", "no typed signature for abs(%s)", arg.kind)
			}
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
	return encodeTypedCell(kind, value, 0, nil)
}

func encodeTypedCell(kind string, value any, depth int, budget *executionBudget) (Cell, error) {
	if err := budget.step("encode"); err != nil {
		return Cell{}, err
	}
	if depth > 32 {
		return Cell{}, typedFailure("result-budget-exceeded", "Typed value nesting exceeds 32")
	}
	if err := wireNumericSize(value); err != nil {
		return Cell{}, err
	}

	if value == nil {
		return Cell{Type: "null"}, nil
	}
	switch kind {
	case "dictionary", "list", "set<string>", "open", "close", "transaction":
		return collectionCell(kind, value, depth, budget)
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
		case dictionaryValue:
			return encodeTypedCell("dictionary", v, depth, budget)
		case listValue:
			return encodeTypedCell("list", v, depth, budget)
		case stringSet:
			return encodeTypedCell("set<string>", v, depth, budget)
		case recordValue:
			return encodeTypedCell(v.kind, v, depth, budget)
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
	case "interval":
		interval, ok := value.(Interval)
		if !ok {
			return Cell{}, typedFailure("type-mismatch", "interval result has type %T", value)
		}
		return Cell{Type: kind, Value: interval}, nil
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
	Tolerances    map[string]ledger.Decimal
}

type metadataInteger struct{ Value ledger.Decimal }

func lookupMetadata(raw any, key string) (any, bool, error) {
	scope, _ := raw.(queryMetadata)
	if key == "__tolerances__" && scope.HasTolerances {
		values := dictionaryValue{}
		for currency, value := range scope.Tolerances {
			values[currency] = value
		}
		return values, true, nil
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
		case ledger.ValueList, ledger.ValueMap:
			value, err := metadataCollection(v)
			return value, true, err
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

func orderingComparison(operator string) bool {
	switch operator {
	case "=", "!=", "<>", "<", "<=", ">", ">=":
		return true
	default:
		return false
	}
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
	for _, item := range q.Select {
		if item.Alias != "" {
			key := strings.ToLower(item.Alias)
			aliases[key] = item.Expr
		}
	}
	resolve := func(expr Expr, code string) (Expr, error) {
		if literal, ok := expr.(literalExpr); ok {
			if n, ok := literal.value.(ledger.Decimal); ok {
				if !n.Rat().IsInt() || strings.ContainsAny(literal.display, ".eE") {
					return nil, locatedQueryError(typedFailure("syntax", "projection ordinal must be an integer token"), expressionRange(expr), "compile")
				}
				r := n.Rat().Num()
				if !r.IsInt64() || r.Int64() < 1 || r.Int64() > int64(len(q.Select)) {
					return nil, locatedQueryError(typedFailure(code, "invalid projection column index %s", literal.display), expressionRange(expr), "compile")
				}
				return q.Select[int(r.Int64())-1].Expr, nil
			}
		}
		if field, ok := expr.(fieldExpr); ok {
			key := strings.ToLower(field.name)

			if replacement, ok := aliases[key]; ok {
				return replacement, nil
			}
		}
		return expr, nil
	}
	for i, expr := range q.GroupBy {
		resolved, err := resolve(expr, "invalid-group")
		if err != nil {
			return err
		}
		q.GroupBy[i] = resolved
	}
	for i, item := range q.OrderBy {
		resolved, err := resolve(item.Expr, "invalid-order")
		if err != nil {
			return err
		}
		q.OrderBy[i].Expr = resolved
	}
	return nil
}

// Keep successful typed values representable by the existing analysis format.
// Intermediate arithmetic has a separate bit budget; final exact spelling has
// the same 4096-byte ceiling as the strict reader, including rational extensions.
func wireNumericSize(value any) error {
	value = unwrapMetadataNumber(value)
	switch v := value.(type) {
	case ledger.Decimal:
		if len(v.String()) > 4096 {
			return typedFailure("numeric-budget-exceeded", "Exact output exceeds 4096 bytes")
		}
	case amountValue:
		return wireNumericSize(v.Number)
	case positionValue:
		if err := wireNumericSize(v.Units); err != nil {
			return err
		}
		if v.Cost != nil {
			return wireNumericSize(v.Cost.Number)
		}
	case inventoryValue:
		for _, position := range v {
			if err := wireNumericSize(position); err != nil {
				return err
			}
		}
	}
	return nil
}
