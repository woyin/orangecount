// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package query implements a deterministic, read-only BeanQuery-shaped
// workbench over evaluated ledger rows.
package query

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"orangecount/internal/ledger"
	"regexp"
	"sync"
)

// Row is one result record keyed by output column name.
type Row map[string]any

// Result is a materialized query result: ordered columns plus rows.
type Result struct {
	Columns []string `json:"columns"`
	Rows    []Row    `json:"rows"`
}

// WriteCSV serializes the result as CSV, header first, missing values as
// empty cells.
func (r Result) WriteCSV(w io.Writer) error {
	writer := csv.NewWriter(w)
	if err := writer.Write(r.Columns); err != nil {
		return err
	}
	for _, row := range r.Rows {
		values := make([]string, len(r.Columns))
		for i, column := range r.Columns {
			values[i] = formatValue(row[column])
		}
		if err := writer.Write(values); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

// Query is a parsed BeanQuery: projection, source table, filters, grouping,
// ordering, and limit.
//
// The standard forms beyond the plain workbench query keep their external
// meaning:
//
//   - Distinct collapses output rows that are equal by canonical typed
//     identity (repeated NULLs collapse to one row, an empty string never
//     equals NULL, and Decimals compare by value, not by spelling).
//   - Having filters whole groups after aggregation; it may reference
//     aggregates and source columns but not projected aliases.
//   - FromSubquery is a derived table: the inner select executes first and the
//     enclosing query reads the inner output columns and their declared types.
//   - Pivot reshapes the projection: Pivot[0] stays the row key, Pivot[1]
//     becomes one output column per distinct value, and every other projected
//     column is a measure repeated for each pivot value. Missing combinations
//     stay NULL.
type Query struct {
	Select   []SelectItem
	From     string
	fromSpan *ByteRange
	Where    Expr
	GroupBy  []Expr
	Having   Expr
	Distinct bool
	Pivot    []Expr
	OrderBy  []OrderItem
	Limit    int
	// LimitSpecified distinguishes LIMIT 0 from an absent limit for typed reads.
	LimitSpecified bool
	// fromSubquery holds a derived table parsed after FROM.
	fromSubquery *Query
	// derivedNames are the output titles of that derived table; the legacy
	// pipeline binds them before execution, the typed compiler builds its own
	// derived readers (see typed.go).
	derivedNames []string
	// havingContextRow is the row upstream leaves in its scan-loop variable when
	// the loop finishes (the last row of the unfiltered source). The typed
	// balance stream consumes WHERE while building its running inventory, so it
	// captures the row here before the filter discards it; evaluateRowsBounded
	// falls back to the last input row when this is nil.
	havingContextRow Row
}

// SelectItem is one projected expression with an optional output alias.
type SelectItem struct {
	sourceTitle string
	Expr        Expr
	Alias       string
}

// OrderItem is one sort key; Desc reverses its direction.
type OrderItem struct {
	Expr Expr
	Desc bool
}

// Expr is a query expression node. eval receives the current row and the
// full row set (window aggregates); aggregate marks nodes that must run
// once per group rather than per row.
type Expr interface {
	String() string
	eval(row Row, rows []Row) (any, error)
	aggregate() bool
}

// ParseError reports a syntax error in the query text.
type ParseError struct {
	Message string
	Range   *ByteRange
}

func (e ParseError) Error() string { return e.Message }

// Parse lexes and parses BeanQuery text into a Query.
func Parse(text string) (Query, error) {
	parser := &queryParser{tokens: lex(text)}
	return parser.parse()
}

// Evaluate parses and runs one query against the evaluated ledger.
func Evaluate(text string, evaluation ledger.Evaluation) (Result, error) {
	query, err := Parse(text)
	if err != nil {
		return Result{}, err
	}
	return EvaluateQuery(query, evaluation)
}

// EvaluateQuery runs a parsed query over an evaluation: resolve the table,
// apply WHERE, group, project the select list, then order and limit. Each
// stage is a helper so the pipeline stays readable in order.
func EvaluateQuery(query Query, evaluation ledger.Evaluation) (Result, error) {
	return evaluateQueryDepth(query, evaluation, 0)
}

// maxDerivedTableDepth bounds recursive derived tables so nesting cannot grow
// silently with the query text.
const maxDerivedTableDepth = 16

func evaluateQueryDepth(query Query, evaluation ledger.Evaluation, depth int) (Result, error) {
	if depth > maxDerivedTableDepth {
		return Result{}, fmt.Errorf("derived tables exceed the maximum depth of %d", maxDerivedTableDepth)
	}
	var rows []Row
	if query.fromSubquery != nil {
		inner, err := evaluateQueryDepth(*query.fromSubquery, evaluation, depth+1)
		if err != nil {
			return Result{}, err
		}
		rows = rowsFromResult(inner)
		query.derivedNames = inner.Columns
		if err := prepareDerivedQuery(&query); err != nil {
			return Result{}, err
		}
	} else {
		if err := prepareQuery(&query); err != nil {
			return Result{}, err
		}
		resolved, err := rowsForTable(query.From, evaluation)
		if err != nil {
			return Result{}, err
		}
		rows = resolved
	}
	result, err := evaluateRows(query, rows)
	if err != nil {
		return Result{}, err
	}
	return pivotLegacyResult(query, result)
}

// rowsFromResult exposes a select result as the input rows of an enclosing
// derived-table query; output titles become the row keys.
func rowsFromResult(result Result) []Row {
	rows := make([]Row, 0, len(result.Rows))
	for _, row := range result.Rows {
		rows = append(rows, row)
	}
	return rows
}

// prepareDerivedQuery validates name references against the derived table's
// output titles instead of a static source schema. A duplicate title resolves
// to its last position, matching the reference's alias map.
func prepareDerivedQuery(query *Query) error {
	resolveGroupAliases(query)
	valid := map[string]bool{}
	for _, name := range query.derivedNames {
		valid[strings.ToLower(name)] = true
	}
	visit := func(expr Expr) error {
		var problem error
		walkFields(expr, func(name string) {
			if !valid[strings.ToLower(name)] && problem == nil {
				problem = fmt.Errorf("unknown column %q in derived table", name)
			}
		})
		return problem
	}
	for _, item := range query.Select {
		if _, ok := item.Expr.(starExpr); ok {
			// A star select names every derived column, so only this item's
			// own field check is skipped. WHERE, GROUP BY, HAVING and PIVOT
			// still run below: skipping them would answer an unanswerable
			// query with plausible zeroes.
			continue
		}
		if err := visit(item.Expr); err != nil {
			return err
		}
	}
	if err := visit(query.Where); err != nil {
		return err
	}
	for _, expr := range query.GroupBy {
		if err := visit(expr); err != nil {
			return err
		}
	}
	if query.Having != nil && !query.Having.aggregate() {
		return fmt.Errorf("the HAVING clause must be an aggregate expression")
	}
	if err := visit(query.Having); err != nil {
		return err
	}
	for _, expr := range query.Pivot {
		if err := visit(expr); err != nil {
			return err
		}
	}
	return validateDerivedOrderByNames(query, valid)
}

// validateDerivedOrderByNames rejects ORDER BY names the derived table cannot
// answer. A sort key may name an output title or any column of the inner
// projection (upstream resolves ORDER BY through its target list, so ordering
// by an inner column that is not projected stays valid). A typo would
// otherwise silently no-op the sort.
func validateDerivedOrderByNames(query *Query, derived map[string]bool) error {
	for i := range query.OrderBy {
		field, ok := query.OrderBy[i].Expr.(fieldExpr)
		if !ok {
			continue
		}
		if derived[strings.ToLower(field.name)] || validOutputName(query, field.name) {
			continue
		}
		return fmt.Errorf("unknown column %q in ORDER BY", field.name)
	}
	return nil
}

// evaluateRows is the common execution pipeline. Typed compilation binds
// expressions before crossing it; legacy callers retain their existing AST.
func evaluateRows(query Query, rows []Row) (Result, error) {
	return evaluateRowsWithCompare(query, rows, legacyComparison)
}

type rowComparison func(any, any) (int, error)

func legacyComparison(left, right any) (int, error) { return compareValues(left, right), nil }

func evaluateRowsWithCompare(query Query, rows []Row, compare rowComparison) (Result, error) {
	return evaluateRowsBounded(query, rows, compare, nil)
}

func evaluateRowsBounded(query Query, rows []Row, compare rowComparison, budget *executionBudget) (Result, error) {
	// Upstream evaluates HAVING against the context row its scan loop leaves
	// behind. The loop variable is assigned before the WHERE test, so the row
	// is the last input row regardless of filtering; a bare source column in
	// HAVING is therefore an aggregate-level test, not a per-group one.
	// Capture that row before grouping reorders the input. A caller that already
	// consumed WHERE while streaming (prepareBalanceStream) passes the row in.
	havingRow := query.havingContextRow
	if havingRow == nil {
		havingRow = havingContext(rows)
	}
	if err := budget.step("where"); err != nil {
		return Result{}, err
	}
	filtered, err := applyWhere(query.Where, rows)
	if err != nil {
		return Result{}, err
	}
	if err := budget.step("group"); err != nil {
		return Result{}, err
	}
	groups, err := groupRows(filtered, query.GroupBy, query.Select)
	if err != nil {
		return Result{}, err
	}
	if query.Having != nil {
		if err := budget.step("having"); err != nil {
			return Result{}, err
		}
		groups, err = applyHaving(query.Having, groups, havingRow)
		if err != nil {
			return Result{}, err
		}
	}
	selectItems := expandStarSelect(filtered, query.Select)
	if err := budget.step("project"); err != nil {
		return Result{}, err
	}
	result, err := projectGroups(selectItems, groups, query.OrderBy, budget)
	if err != nil {
		return Result{}, err
	}
	if query.Distinct {
		if err := budget.step("distinct"); err != nil {
			return Result{}, err
		}
		result, err = distinctResult(result, budget)
		if err != nil {
			return Result{}, err
		}
	}
	if err := budget.step("sort"); err != nil {
		return Result{}, err
	}
	boundedCompare := func(left, right any) (int, error) {
		if err := budget.step(""); err != nil {
			return 0, err
		}
		return compare(left, right)
	}
	if err := sortResultRowsBy(result, query.OrderBy, boundedCompare); err != nil {
		return Result{}, err
	}
	if query.Limit > 0 && len(result.Rows) > query.Limit {
		result.Rows = result.Rows[:query.Limit]
	}
	if err := budget.step(""); err != nil {
		return Result{}, err
	}
	return result, nil
}

// applyHaving keeps only the groups whose HAVING predicate is true. The
// predicate runs once per group against the aggregate-level context row, so it
// sees the same aggregate values the projection sees and resolves a bare
// source column exactly as upstream does (the context row is the last row that
// passed WHERE, which is what makes `HAVING account = 'x'` an aggregate-level
// test rather than a per-group filter). Iteration is bounded and stops on
// cancellation like every other execution stage.
func applyHaving(having Expr, groups [][]Row, context Row) ([][]Row, error) {
	if having == nil {
		return groups, nil
	}
	kept := make([][]Row, 0, len(groups))
	for _, group := range groups {
		if err := rowBudget(context, group).step("having"); err != nil {
			return nil, err
		}
		value, err := having.eval(context, group)
		if err != nil {
			return nil, err
		}
		truth, err := typedTruth(value)
		if err != nil {
			return nil, err
		}
		if truth {
			kept = append(kept, group)
		}
	}
	return kept, nil
}

// havingContext returns the row upstream leaves in its iteration variable
// when its scan loop finishes: the last row of the input, captured before
// WHERE runs because the loop variable is assigned ahead of the filter. A
// HAVING predicate that references a bare source column therefore observes
// that one row for every group, not the group's own first row. An empty input
// leaves the variable unset, which upstream reads as NULL; a zero-value Row
// gives the same NULL result for every column reference.
//
// This is oracle-pinned behavior, not an intentional SQL semantic:
// TestHavingFiltersGroupsAfterAggregation in standard_forms_test.go reproduces
// the reference's cell-for-cell observations, including the WHERE case.
func havingContext(rows []Row) Row {
	if len(rows) == 0 {
		return Row{}
	}
	return rows[len(rows)-1]
}

// distinctResult collapses output rows whose canonical typed values are all
// equal, keeping the first occurrence so ORDER BY and LIMIT still observe the
// reference's stable order. Equal values of different types (the integer 1 and
// the decimal 1.0, or NULL and an empty string) stay distinct rows.
func distinctResult(result Result, budget *executionBudget) (Result, error) {
	seen := make(map[string]bool, len(result.Rows))
	kept := make([]Row, 0, len(result.Rows))
	for _, row := range result.Rows {
		if err := budget.step("distinct"); err != nil {
			return Result{}, err
		}
		values := make([]any, len(result.Columns))
		for i, column := range result.Columns {
			values[i] = row[column]
		}
		key, err := identityKey(values)
		if err != nil {
			return Result{}, err
		}
		if seen[key] {
			continue
		}
		if len(seen) >= QueryDistinctKeyLimit {
			return Result{}, typedFailure("result-budget-exceeded", "SELECT DISTINCT exceeds the unique-key budget (%d keys); narrow the projection or add a LIMIT", QueryDistinctKeyLimit)
		}
		seen[key] = true
		kept = append(kept, row)
	}
	result.Rows = kept
	return result, nil
}

// identityKey encodes values for grouping and DISTINCT. The runtime type is
// part of the identity, so the integer 1, the Decimal 1 and the string "1"
// never share a row, while every NULL encodes alike and collapses.
func identityKey(values []any) (string, error) {
	parts := make([]struct {
		Type  string `json:"type"`
		Value any    `json:"value"`
	}, len(values))
	for i, value := range values {
		parts[i].Type, parts[i].Value = fmt.Sprintf("%T", value), value
	}
	encoded, err := json.Marshal(parts)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// applyWhere filters rows by the WHERE predicate, if present.
func applyWhere(where Expr, rows []Row) ([]Row, error) {
	if where == nil {
		return rows, nil
	}
	filtered := make([]Row, 0, len(rows))
	for _, row := range rows {
		value, err := where.eval(row, []Row{row})
		if err != nil {
			return nil, err
		}
		if truthy(value) {
			filtered = append(filtered, row)
		}
	}
	return filtered, nil
}

// QueryPivotCellLimit bounds the dense PIVOT BY cell matrix (row keys x pivot
// values x measures) before it is allocated. One typed-wire cell encodes to at
// least 15 bytes of JSON ({"type":"null"}), so this ceiling keeps the matrix
// itself near QueryByteLimit; the dominant dense-pivot allocation is actually
// the per-row identity encoding, which pivotCells memoizes. Under the default
// step budget a dense pivot reaches roughly 65k cells (~31 steps per cell),
// so this limit only binds sparse pivots.
const QueryPivotCellLimit = QueryByteLimit / 16

// QueryDistinctKeyLimit bounds the deduplication map of SELECT DISTINCT. The
// map holds one canonical identity key per unique input row (measured 243 B
// per key at one column up to ~3.4 kB at twenty), and DISTINCT runs before
// sort and LIMIT, so without this ceiling a high-cardinality DISTINCT on a
// large ledger grows the map without bound — including on the legacy entry,
// which runs without a step budget. The value keeps the worst-case map at a
// small multiple of QueryByteLimit while staying above QueryRowLimit, so any
// DISTINCT whose result fits the row ceiling still executes and only inputs
// that would truncate anyway fail loudly.
const QueryDistinctKeyLimit = QueryByteLimit / 1024

// pivotLegacyResult reshapes a projected result into the PIVOT BY layout:
// the first pivot column stays the row key, the second turns into one output
// column per distinct value, and each remaining projected column repeats as a
// measure for every pivot value. Row and pivot keys are emitted in ascending
// order and absent combinations stay empty, matching the reference. The
// projection (including ORDER BY and LIMIT) has already been applied, exactly
// like upstream's EvalPivot wrapper.
func pivotLegacyResult(query Query, result Result) (Result, error) {
	if len(query.Pivot) == 0 {
		return result, nil
	}
	rowIndex, columnIndex, err := pivotTargetIndexes(query, query.Pivot, result.Columns)
	if err != nil {
		return Result{}, err
	}
	measures := make([]int, 0, len(result.Columns))
	for i := range result.Columns {
		if i != rowIndex && i != columnIndex {
			measures = append(measures, i)
		}
	}
	cell := func(row Row, index int) any { return row[result.Columns[index]] }
	rows, columns, cells, width, err := pivotCells(result.Rows, rowIndex, columnIndex, measures, cell, legacyComparison, nil, QueryPivotCellLimit)
	if err != nil {
		return Result{}, err
	}
	pivoted := Result{}
	pivoted.Columns = append(pivoted.Columns, result.Columns[rowIndex]+"/"+result.Columns[columnIndex])
	if len(measures) == 0 {
		// A pivot without a measure keeps only the row key column: upstream
		// still spreads the second key, but there is no value to repeat.
		for _, rowKey := range rows {
			pivoted.Rows = append(pivoted.Rows, Row{pivoted.Columns[0]: rowKey})
		}
		return pivoted, nil
	}
	for _, column := range columns {
		for _, measure := range measures {
			name := formatValue(column)
			if len(measures) > 1 {
				name += "/" + result.Columns[measure]
			}
			pivoted.Columns = append(pivoted.Columns, name)
		}
	}
	// The legacy result is a title-keyed map, so a repeated pivot title would
	// silently discard one column. Fail loudly like the duplicate projection
	// titles already do.
	seen := make(map[string]bool, len(pivoted.Columns))
	for _, name := range pivoted.Columns {
		if seen[name] {
			return Result{}, fmt.Errorf("duplicate pivot output title %q cannot be represented by legacy JSON/CSV; use typed-json", name)
		}
		seen[name] = true
	}
	for i, rowKey := range rows {
		row := Row{pivoted.Columns[0]: rowKey}
		for j, value := range cells[i*width : (i+1)*width] {
			row[pivoted.Columns[j+1]] = value
		}
		pivoted.Rows = append(pivoted.Rows, row)
	}
	return pivoted, nil
}

// pivotTargetIndexes resolves the two PIVOT BY columns for the legacy entry:
// an integer literal names a 1-based projection position, a bare name matches
// an output title, and the second column must be a GROUP BY key. The typed
// compiler applies the same rules (see pivotProjectionIndexes) and reports
// them with typed error categories.
func pivotTargetIndexes(query Query, pivot []Expr, names []string) (int, int, error) {
	resolve := func(expr Expr) (int, error) {
		if ordinal, ok := projectionOrdinal(expr); ok {
			if ordinal < 1 || ordinal > len(names) {
				return 0, fmt.Errorf("invalid PIVOT BY column index %d", ordinal)
			}
			return ordinal - 1, nil
		}
		field, ok := expr.(fieldExpr)
		if !ok {
			return 0, fmt.Errorf("PIVOT BY column %s is not in the projection", expr.String())
		}
		for i, name := range names {
			if strings.EqualFold(name, field.name) {
				return i, nil
			}
		}
		return 0, fmt.Errorf("PIVOT BY column %s is not in the projection", expr.String())
	}
	first, err := resolve(pivot[0])
	if err != nil {
		return 0, 0, err
	}
	second, err := resolve(pivot[1])
	if err != nil {
		return 0, 0, err
	}
	if first == second {
		return 0, 0, fmt.Errorf("the two PIVOT BY columns cannot be the same column")
	}
	if second >= len(query.Select) {
		return 0, 0, fmt.Errorf("the second PIVOT BY column must be a GROUP BY column")
	}
	grouped := map[string]bool{}
	for _, expr := range query.GroupBy {
		grouped[expr.String()] = true
	}
	if !grouped[query.Select[second].Expr.String()] {
		return 0, 0, fmt.Errorf("the second PIVOT BY column must be a GROUP BY column")
	}
	return first, second, nil
}

// pivotCells is the shared cell engine for the legacy and typed pipelines.
// value reads one cell of one projected row, compare orders the pivot keys and
// step charges the execution budget (nil for the unbounded legacy path). It
// returns the distinct row keys and pivot values in ascending order together
// with the measure matrix indexed as
// cells[row*len(measures)+column*len(measures)+measure]; an absent
// combination stays nil so the caller can render it as NULL. cellLimit bounds
// the dense matrix before it is allocated; without it a small query text could
// allocate gigabytes before any result-size check runs.
func pivotCells(rows []Row, rowIndex, columnIndex int, measures []int, value func(Row, int) any, compare rowComparison, step func() error, cellLimit int) ([]any, []any, []any, int, error) {
	if step == nil {
		step = func() error { return nil }
	}
	rowKeys := []any{}
	columnValues := []any{}
	rowAt := map[string]int{}
	columnAt := map[string]int{}
	// The fill pass below needs the same identities again; caching them per
	// input row keeps a dense pivot's dominant allocation at one identityKey
	// per row instead of three (the cell matrix itself is only ~16 B/cell).
	rowIdentities := make([]string, 0, len(rows))
	columnIdentities := make([]string, 0, len(rows))
	for _, row := range rows {
		if err := step(); err != nil {
			return nil, nil, nil, 0, err
		}
		rowKey, columnValue := value(row, rowIndex), value(row, columnIndex)
		rowIdentity, err := identityKey([]any{rowKey})
		if err != nil {
			return nil, nil, nil, 0, err
		}
		columnIdentity, err := identityKey([]any{columnValue})
		if err != nil {
			return nil, nil, nil, 0, err
		}
		if _, ok := rowAt[rowIdentity]; !ok {
			rowAt[rowIdentity] = len(rowKeys)
			rowKeys = append(rowKeys, rowKey)
		}
		if _, ok := columnAt[columnIdentity]; !ok {
			columnAt[columnIdentity] = len(columnValues)
			columnValues = append(columnValues, columnValue)
		}
		rowIdentities = append(rowIdentities, rowIdentity)
		columnIdentities = append(columnIdentities, columnIdentity)
	}
	if err := sortPivotKeys(rowKeys, rowAt, compare, step); err != nil {
		return nil, nil, nil, 0, err
	}
	if err := sortPivotKeys(columnValues, columnAt, compare, step); err != nil {
		return nil, nil, nil, 0, err
	}
	width := len(columnValues) * len(measures)
	if cellLimit > 0 && len(rowKeys)*width > cellLimit {
		return nil, nil, nil, 0, typedFailure("result-budget-exceeded", "PIVOT BY result exceeds the cell budget (%d x %d cells); narrow the pivot or add a LIMIT", len(rowKeys), width)
	}
	cells := make([]any, len(rowKeys)*width)
	for i, row := range rows {
		if err := step(); err != nil {
			return nil, nil, nil, 0, err
		}
		rowPosition := rowAt[rowIdentities[i]]
		columnPosition := columnAt[columnIdentities[i]]
		for measureIndex, measure := range measures {
			cells[rowPosition*width+columnPosition*len(measures)+measureIndex] = value(row, measure)
		}
	}
	return rowKeys, columnValues, cells, width, nil
}

func mustIdentity(value any) string {
	identity, _ := identityKey([]any{value})
	return identity
}

// sortPivotKeys orders pivot keys ascending and rebases their positions.
func sortPivotKeys(values []any, positions map[string]int, compare rowComparison, step func() error) error {
	if len(values) == 0 {
		return nil
	}
	identities := make([]string, len(values))
	for i, value := range values {
		identities[i] = mustIdentity(value)
	}
	indices := make([]int, len(values))
	for i := range indices {
		indices[i] = i
	}
	var sortErr error
	sort.SliceStable(indices, func(i, j int) bool {
		if sortErr != nil {
			return false
		}
		if err := step(); err != nil {
			sortErr = err
			return false
		}
		comparison, err := compare(values[indices[i]], values[indices[j]])
		if err != nil {
			sortErr = err
			return false
		}
		return comparison < 0
	})
	if sortErr != nil {
		return sortErr
	}
	ordered := make([]any, len(values))
	for position, index := range indices {
		ordered[position] = values[index]
	}
	copy(values, ordered)
	for position, index := range indices {
		positions[identities[index]] = position
	}
	return nil
}

// expandStarSelect rewrites a lone `SELECT *` into one field item per column
// of the underlying rows, alphabetically so the output is deterministic.
func expandStarSelect(rows []Row, selectItems []SelectItem) []SelectItem {
	if len(selectItems) != 1 {
		return selectItems
	}
	if _, ok := selectItems[0].Expr.(starExpr); !ok {
		return selectItems
	}
	keys := make(map[string]struct{})
	for _, row := range rows {
		for key := range row {
			keys[key] = struct{}{}
		}
	}
	ordered := make([]string, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Strings(ordered)
	expanded := make([]SelectItem, 0, len(ordered))
	for _, key := range ordered {
		expanded = append(expanded, SelectItem{Expr: fieldExpr{name: key}, Alias: key})
	}
	return expanded
}

// projectGroups evaluates the select list once per group, naming each output
// column by its alias (or expression text when unaliased).
// projectGroups projects each group onto the select list. ORDER BY keys
// that are not projected columns (upstream bean-query permits ordering by
// any source expression) ride along under their expression text and are
// stripped by sortResultRows so they never surface as output.
func projectGroups(selectItems []SelectItem, groups [][]Row, orderBy []OrderItem, budget *executionBudget) (Result, error) {
	result := Result{}
	projected := make(map[string]bool, len(selectItems))
	for _, item := range selectItems {
		alias := item.Alias
		if alias == "" {
			alias = item.Expr.String()
		}
		if projected[alias] {
			return Result{}, fmt.Errorf("duplicate output title %q cannot be represented by legacy JSON/CSV; use typed-json", alias)
		}
		result.Columns = append(result.Columns, alias)
		projected[alias] = true
	}
	sortKeys := make([]Expr, 0, len(orderBy))
	for _, item := range orderBy {
		if !projected[item.Expr.String()] {
			sortKeys = append(sortKeys, item.Expr)
		}
	}
	for _, group := range groups {
		if err := budget.step("project"); err != nil {
			return Result{}, err
		}
		row := Row{}
		for _, item := range selectItems {
			alias := item.Alias
			if alias == "" {
				alias = item.Expr.String()
			}
			value, err := item.Expr.eval(firstRow(group), group)
			if err != nil {
				return Result{}, err
			}
			row[alias] = value
		}
		for _, key := range sortKeys {
			value, err := key.eval(firstRow(group), group)
			if err != nil {
				return Result{}, err
			}
			row[key.String()] = value
		}
		result.Rows = append(result.Rows, row)
	}
	return result, nil
}

// sortResultRows applies the ORDER BY list; ties keep their prior (stable)
// order, and rows missing the sort key compare via formatValue.
func sortResultRows(result Result, orderBy []OrderItem) {
	_ = sortResultRowsBy(result, orderBy, legacyComparison)
}

func sortResultRowsBy(result Result, orderBy []OrderItem, compare rowComparison) error {
	if len(orderBy) == 0 {
		return nil
	}
	defer func() {
		// Strip the ORDER BY helper keys that are not declared columns so
		// they never leak into output rows.
		columns := make(map[string]bool, len(result.Columns))
		for _, name := range result.Columns {
			columns[name] = true
		}
		for _, row := range result.Rows {
			for key := range row {
				if !columns[key] {
					delete(row, key)
				}
			}
		}
	}()
	var sortErr error
	sort.SliceStable(result.Rows, func(i, j int) bool {
		if sortErr != nil {
			return false
		}
		left, right := result.Rows[i], result.Rows[j]
		for _, item := range orderBy {
			lv := left[item.Expr.String()]
			rv := right[item.Expr.String()]
			comparison, err := compare(lv, rv)
			if err != nil {
				sortErr = err
				return false
			}
			if comparison == 0 {
				continue
			}
			if item.Desc {
				return comparison > 0
			}
			return comparison < 0
		}
		return false
	})
	return sortErr
}

// groupRows partitions the joined rows for aggregation: no GROUP BY and no
// aggregate keeps every row its own group, no GROUP BY with aggregates makes
// one group, and otherwise rows group by canonical typed key values.
//
// A PIVOT BY column is removed from the grouping key because its values are
// spread across output columns, not down the rows.
func groupRows(rows []Row, groupBy []Expr, selectItems []SelectItem) ([][]Row, error) {
	aggregate := false
	for _, item := range selectItems {
		aggregate = aggregate || item.Expr.aggregate()
	}
	if len(groupBy) == 0 && !aggregate {
		groups := make([][]Row, 0, len(rows))
		for _, row := range rows {
			if err := rowBudget(row, nil).step(""); err != nil {
				return nil, err
			}
			groups = append(groups, []Row{row})
		}
		return groups, nil
	}
	if len(groupBy) == 0 {
		if len(rows) == 0 {
			return nil, nil
		}
		return [][]Row{rows}, nil
	}
	groups := make([][]Row, 0)
	indexes := make(map[string]int)
	for _, row := range rows {
		if err := rowBudget(row, nil).step(""); err != nil {
			return nil, err
		}
		key, err := groupKey(row, groupBy)
		if err != nil {
			return nil, err
		}
		index, ok := indexes[key]
		if !ok {
			index = len(groups)
			indexes[key] = index
			groups = append(groups, nil)
		}
		groups[index] = append(groups[index], row)
	}
	return groups, nil
}

// groupKey encodes the values of a GROUP BY key by their typed identity so
// that only values equal as typed values share one group.
func groupKey(row Row, groupBy []Expr) (string, error) {
	parts := make([]any, len(groupBy))
	for i, expr := range groupBy {
		value, err := expr.eval(row, []Row{row})
		if err != nil {
			return "", err
		}
		parts[i] = value
	}
	return identityKey(parts)
}

func firstRow(rows []Row) Row {
	if len(rows) == 0 {
		return Row{}
	}
	return rows[0]
}

func rowsForTable(table string, evaluation ledger.Evaluation) ([]Row, error) {
	switch strings.ToLower(table) {
	case "postings", "posting":
		return postingRows(evaluation), nil
	case "entries", "entry", "directives":
		return entryRows(evaluation), nil
	case "accounts", "account":
		return accountRows(evaluation), nil
	case "prices", "price":
		return priceRows(evaluation), nil
	default:
		return nil, fmt.Errorf("unknown query table %q", table)
	}
}

// formatInventory renders a per-currency balance map the way upstream
// beanquery prints inventories: components sorted by currency and joined
// with ", " (e.g. "100 CNY, -30 USD"). An empty inventory is the empty
// string, matching NULL rendering.
func formatInventory(balances map[string]ledger.Decimal) string {
	if len(balances) == 0 {
		return ""
	}
	currencies := make([]string, 0, len(balances))
	for currency := range balances {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)
	parts := make([]string, 0, len(currencies))
	for _, currency := range currencies {
		parts = append(parts, balances[currency].String()+" "+currency)
	}
	return strings.Join(parts, ", ")
}

// datePartDecimal extracts the year/month/day of a ledger date as a Decimal
// so the derived columns group, filter, and order like any number.
func datePartDecimal(date ledger.Date, part string) ledger.Decimal {
	switch part {
	case "year":
		return ledger.NewDecimal(big.NewRat(int64(date.Year), 1))
	case "month":
		return ledger.NewDecimal(big.NewRat(int64(date.Month), 1))
	default:
		return ledger.NewDecimal(big.NewRat(int64(date.Day), 1))
	}
}

func postingRows(evaluation ledger.Evaluation) []Row {
	return postingRowsWithSchema(evaluation, false)
}

func postingRowsWithSchema(evaluation ledger.Evaluation, standard bool) []Row {
	rows, _ := postingRowsWithBudget(evaluation, standard, nil)
	return rows
}

func postingRowsWithBudget(evaluation ledger.Evaluation, standard bool, budget *executionBudget) ([]Row, error) {
	rows := make([]Row, 0)
	runningBalances := map[string]map[string]ledger.Decimal{}
	// Upstream beanquery answers from the loader's chronologically sorted
	// entry stream, so the balance column must accumulate in date order (a
	// stable sort keeps source order within one day). Raw source order would
	// interleave years whenever a multi-year file precedes the year files.
	ordered := append([]ledger.EntryRecord(nil), evaluation.Entries...)
	var sortErr error
	sort.SliceStable(ordered, func(i, j int) bool {
		if sortErr != nil {
			return false
		}
		if sortErr = budget.step("source"); sortErr != nil {
			return false
		}
		if ordered[i].Date.Year != ordered[j].Date.Year {
			return ordered[i].Date.Year < ordered[j].Date.Year
		}
		if ordered[i].Date.Month != ordered[j].Date.Month {
			return ordered[i].Date.Month < ordered[j].Date.Month
		}
		return ordered[i].Date.Day < ordered[j].Date.Day
	})
	if sortErr != nil {
		return nil, sortErr
	}
	for _, entry := range ordered {
		if err := budget.step("source"); err != nil {
			return nil, err
		}
		var transaction *ledger.Transaction
		switch value := entry.Directive.(type) {
		case *ledger.Transaction:
			transaction = value
		case ledger.Transaction:
			copy := value
			transaction = &copy
		}
		if transaction == nil {
			continue
		}
		for postingIndex, posting := range transaction.Postings {
			if err := budget.step("source"); err != nil {
				return nil, err
			}
			if budget != nil && posting.Units != nil {
				if err := budget.numericValue(ledger.NewDecimal(posting.Units.Number.Rat)); err != nil {
					return nil, err
				}
				if err := budget.decimalText(posting.Units.Number.Raw); err != nil {
					return nil, err
				}
				if posting.Price != nil {
					if err := budget.decimalText(posting.Price.Amount.Number.Raw); err != nil {
						return nil, err
					}
				}
				if posting.Cost != nil {
					for _, component := range posting.Cost.Components {
						if component.Kind == ledger.ValueAmount {
							if err := budget.decimalText(component.Amount.Number.Raw); err != nil {
								return nil, err
							}
						}
					}
				}
			}
			// Journal directive filters use the transaction flag (the same flag
			// shown in Fava's journal). Preserve a separate posting_flag for
			// callers that need the lower-level posting marker.
			row := Row{"date": transaction.Date.Raw, "account": posting.Account, "flag": transaction.Flag, "posting_flag": posting.Flag, "narration": transaction.Narration, "payee": transaction.Payee, "tags": append([]string(nil), transaction.Tags...), "links": append([]string(nil), transaction.Links...), "file": entry.File, "span": entry.Span.String(), "kind": string(transaction.Kind())}
			if budget != nil {
				row["_query_execution"] = budget
			}
			// BeanQuery's derived date columns let GROUP BY year, month
			// partition postings without spelling year(date).
			row["year"] = datePartDecimal(transaction.Date, "year")
			row["month"] = datePartDecimal(transaction.Date, "month")
			row["day"] = datePartDecimal(transaction.Date, "day")
			if posting.Units != nil {
				row["currency"] = posting.Units.Currency
				row["units"] = ledger.DecimalFromNumber(posting.Units.Number)
				row["number"] = ledger.DecimalFromNumber(posting.Units.Number)
				row["weight"] = ledger.PostingWeight(posting)
				// The running per-account balance accumulates Units in
				// evaluation order; elided legs are already completed in the
				// entry stream (see inferElision), so this reproduces the
				// authoritative account state after each posting.
				running := runningBalances[posting.Account]
				if running == nil {
					running = map[string]ledger.Decimal{}
					runningBalances[posting.Account] = running
				}
				running[posting.Units.Currency] = ledger.NewDecimal(new(big.Rat).Add(running[posting.Units.Currency].Rat(), ledger.DecimalFromNumber(posting.Units.Number).Rat()))
				row["balance"] = formatInventory(running)
			} else {
				row["balance"] = formatInventory(runningBalances[posting.Account])
			}
			if posting.Cost != nil {
				row["cost"] = posting.Cost.Raw
			}
			if standard {
				if posting.Units != nil {
					row["_posting_index"] = postingIndex
					row["_transaction"] = *transaction
					row["_entry_record"] = entry
					if posting.Price != nil {
						row["price"] = queryUnitPrice(posting)
					}
					row["_payee_present"] = transaction.HasPayee || transaction.Payee != ""
					row["_number_source"] = posting.Units.Number
					position := queryPosition(posting)
					row["position"] = position
					if position.Cost != nil || posting.Price == nil {
						row["weight"] = positionCost(position)
					} else {
						row["weight"] = queryPriceWeight(posting)
					}
				}
				row["filename"] = entry.File
				row["lineno"] = ledger.NewDecimal(new(big.Rat).SetInt64(int64(posting.At.StartLine)))
				line := posting.At.StartLine
				if posting.SourceMetadata.Present {
					row["filename"] = posting.SourceMetadata.File
					line = posting.SourceMetadata.Line
					row["lineno"] = ledger.NewDecimal(new(big.Rat).SetInt64(int64(line)))
				}
				if line <= 0 || row["filename"] == "" {
					row["filename"], row["lineno"] = nil, nil
				}
				postingFile, _ := row["filename"].(string)
				row["_posting_meta"] = queryMetadata{Values: posting.Meta, Filename: postingFile, Line: line}
				// Pad-generated entries use balance source metadata on their
				// postings and do not carry the ordinary transaction tolerance map.
				row["_entry_meta"] = queryMetadata{Values: transaction.Meta, Filename: entry.File, Line: entry.Span.StartLine, HasTolerances: !posting.SourceMetadata.Present, Tolerances: transaction.Tolerances}
			}
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func entryRows(evaluation ledger.Evaluation) []Row {
	rows := make([]Row, 0, len(evaluation.Entries))
	for _, entry := range evaluation.Entries {
		rows = append(rows, Row{"date": entry.Date.Raw, "kind": entry.Directive.Kind(), "span": entry.Span.String(), "file": entry.File})
	}
	return rows
}

func accountRows(evaluation ledger.Evaluation) []Row {
	accounts := make([]string, 0, len(evaluation.Accounts))
	for name := range evaluation.Accounts {
		accounts = append(accounts, name)
	}
	sort.Strings(accounts)
	rows := make([]Row, 0)
	for _, name := range accounts {
		account := evaluation.Accounts[name]
		currencies := make([]string, 0, len(account.Balances))
		for currency := range account.Balances {
			currencies = append(currencies, currency)
		}
		sort.Strings(currencies)
		for _, currency := range currencies {
			rows = append(rows, Row{"account": name, "currency": currency, "balance": account.Balances[currency], "opened": account.Opened.Raw})
		}
		if len(currencies) == 0 {
			rows = append(rows, Row{"account": name, "currency": "", "balance": ledger.Zero(), "opened": account.Opened.Raw})
		}
	}
	return rows
}

func priceRows(evaluation ledger.Evaluation) []Row {
	bases := make([]string, 0, len(evaluation.Prices))
	for base := range evaluation.Prices {
		bases = append(bases, base)
	}
	sort.Strings(bases)
	rows := make([]Row, 0)
	for _, base := range bases {
		for _, quote := range evaluation.Prices[base] {
			rows = append(rows, Row{"date": quote.Date.Raw, "currency": quote.Base, "amount": quote.Amount, "quote_currency": quote.Currency})
		}
	}
	return rows
}

type queryTokenKind uint8

const (
	tokenWord queryTokenKind = iota
	tokenString
	tokenNumber
	tokenOperator
	tokenComma
	tokenLParen
	tokenRParen
	tokenStar
	tokenDate
)

type queryToken struct {
	kind       queryTokenKind
	text       string
	start, end int
}

// lex splits a query string into words, numbers, strings, operators, and
// delimiters. Each token family has its own scanner; the loop only routes on
// the leading rune.
func lex(text string) []queryToken {
	return lexQuery(text, false)
}

func lexQuery(text string, dates bool) []queryToken {
	tokens := make([]queryToken, 0)
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		if unicode.IsSpace(r) {
			i += size
			continue
		}
		start := i
		if dates && dateTokenAt(text, i) {
			tokens = append(tokens, queryToken{kind: tokenDate, text: text[i : i+10], start: i, end: i + 10})
			i += 10
			continue
		}
		if r == '\'' || r == '"' {
			var token queryToken
			token, i = lexString(text, i, byte(r), size)
			token.start, token.end = start, i
			tokens = append(tokens, token)
			continue
		}
		if kind, ok := delimiterKind(r); ok {
			tokens = append(tokens, queryToken{kind: kind, text: string(r), start: i, end: i + size})
			i += size
			continue
		}
		if strings.ContainsRune("=<>!+-/~%?", r) {
			var token queryToken
			token, i = lexOperator(text, i, size)
			token.start, token.end = start, i
			tokens = append(tokens, token)
			continue
		}
		var token queryToken
		token, i = lexWord(text, i)
		if dates && metadataDecimalSyntax.MatchString(token.text) && !strings.ContainsAny(token.text, "eE") {
			token.kind = tokenNumber
		}
		token.start, token.end = start, i
		tokens = append(tokens, token)
	}
	return tokens
}

func dateTokenAt(text string, offset int) bool {
	if len(text)-offset < 10 {
		return false
	}
	value := text[offset : offset+10]
	for i := 0; i < 10; i++ {
		if i == 4 || i == 7 {
			if value[i] != '-' {
				return false
			}
		} else if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	if len(text) == offset+10 {
		return true
	}
	r, _ := utf8.DecodeRuneInString(text[offset+10:])
	return unicode.IsSpace(r) || strings.ContainsRune(",()*<>=!+-/", r)
}

// delimiterKind maps one-rune delimiters to their token kinds.
func delimiterKind(r rune) (queryTokenKind, bool) {
	switch r {
	case ',':
		return tokenComma, true
	case '(':
		return tokenLParen, true
	case ')':
		return tokenRParen, true
	case '*':
		return tokenStar, true
	default:
		return tokenWord, false
	}
}

// lexString consumes a quoted string starting at text[i]; backslash escapes
// the next character, and an unterminated string ends at end-of-input.
func lexString(text string, i int, quote byte, size int) (queryToken, int) {
	i += size
	var value strings.Builder
	for i < len(text) && text[i] != quote {
		if text[i] == '\\' && i+1 < len(text) {
			i++
		}
		value.WriteByte(text[i])
		i++
	}
	if i < len(text) {
		i++
	}
	return queryToken{kind: tokenString, text: value.String()}, i
}

// lexOperator consumes an operator starting at text[i]: two-character forms
// (<=, >=, !=, <>) are glued, everything else is the single rune.
func lexOperator(text string, i int, size int) (queryToken, int) {
	start := i
	i += size
	if i < len(text) {
		next := text[i]
		glue := next == '=' || (text[start] == '<' && next == '>') ||
			((text[start] == '!' || text[start] == '?') && next == '~')
		if glue {
			i++
		}
	}
	return queryToken{kind: tokenOperator, text: text[start:i]}, i
}

// lexWord consumes a bare word and classifies it as a number when it parses
// as one, mirroring how the primary parser treats numeric literals.
func lexWord(text string, i int) (queryToken, int) {
	start := i
	for i < len(text) {
		r, size := utf8.DecodeRuneInString(text[i:])
		if unicode.IsSpace(r) || strings.ContainsRune("(),*<>=!+-/\"", r) {
			break
		}
		i += size
	}
	word := text[start:i]
	kind := tokenWord
	if _, err := strconv.ParseFloat(word, 64); err == nil {
		kind = tokenNumber
	}
	return queryToken{kind: kind, text: word}, i
}

type queryParser struct {
	inputText       string
	tokens          []queryToken
	index           int
	defaultPostings bool
	inputBytes      int
}

// parse reads one SELECT statement clause by clause. Each clause parser
// consumes its keywords so this loop only sequences them and verifies the
// statement is fully consumed at the end.
func (p *queryParser) parse() (Query, error) {
	return p.parseStatement(false)
}

// parseStatement parses one complete SELECT. A derived table re-enters this
// routine and leaves the closing parenthesis for its caller, so the "statement
// fully consumed" check is skipped at that level.
func (p *queryParser) parseStatement(nested bool) (Query, error) {
	query := Query{}
	if !p.acceptWord("select") {
		return query, p.errorf("query must start with SELECT")
	}
	query.Distinct = p.acceptWord("distinct")
	if err := p.parseSelectList(&query); err != nil {
		return query, err
	}
	if len(query.Select) == 0 {
		return query, p.errorf("SELECT requires at least one expression")
	}
	if p.acceptWord("from") {
		if err := p.parseFrom(&query); err != nil {
			return query, err
		}
	} else if p.defaultPostings {
		query.From = "postings"
	} else {
		return query, p.errorf("query requires FROM")
	}
	if err := p.parseWhere(&query); err != nil {
		return query, err
	}
	if err := p.parseGroupBy(&query); err != nil {
		return query, err
	}
	if err := p.parseOrderBy(&query); err != nil {
		return query, err
	}
	if err := p.parsePivotBy(&query); err != nil {
		return query, err
	}
	if err := p.parseLimit(&query); err != nil {
		return query, err
	}
	if !nested && (p.peek().kind != 0 || p.index < len(p.tokens)) {
		return query, p.errorf("unexpected token %q", p.peek().text)
	}
	return query, nil
}

// parseFrom consumes either a table name or a parenthesized derived table.
func (p *queryParser) parseFrom(query *Query) error {
	if p.peek().kind == tokenLParen {
		open := p.next()
		inner, err := p.parseStatement(true)
		if err != nil {
			return err
		}
		if !p.accept(tokenRParen) {
			return p.errorf("missing closing parenthesis after derived table")
		}
		query.fromSubquery = &inner
		query.fromSpan = &ByteRange{Start: open.start, End: open.end}
		return nil
	}
	sourceToken := p.peek()
	from, err := p.expectWord("table name")
	if sourceToken.kind == tokenString && p.defaultPostings {
		from, err = sourceToken.text, nil
	}
	if err != nil {
		return err
	}
	query.From = from
	query.fromSpan = &ByteRange{Start: sourceToken.start, End: sourceToken.end}
	return nil
}

// parseSelectList consumes the comma-separated projection expressions. A
// bare word after an expression (not a clause keyword) is its implicit alias.
func (p *queryParser) parseSelectList(query *Query) error {
	for {
		if p.accept(tokenStar) {
			query.Select = append(query.Select, SelectItem{Expr: starExpr{}})
		} else {
			start := p.index
			expr, err := p.expression(0)
			if err != nil {
				return err
			}
			item := SelectItem{Expr: expr}
			if p.defaultPostings && p.inputText != "" {
				item.sourceTitle = p.inputText[p.tokens[start].start:p.tokens[p.index-1].end]
				inner := expr
				for strings.HasPrefix(item.sourceTitle, "+") {
					item.sourceTitle = strings.TrimSpace(item.sourceTitle[1:])
				}
				for {
					paren, ok := inner.(parenExpr)
					if !ok {
						break
					}
					item.sourceTitle = strings.TrimSpace(item.sourceTitle[1 : len(item.sourceTitle)-1])
					inner = paren.inner
				}
				if field, ok := inner.(fieldExpr); ok {
					item.sourceTitle = strings.ToLower(field.name)
				}
			}
			if p.acceptWord("as") {
				alias, aliasErr := p.expectWord("column alias")
				if aliasErr != nil {
					return aliasErr
				}
				item.Alias = alias
			} else if p.peek().kind == tokenWord && !isClauseWord(p.peek().text) {
				item.Alias = p.next().text
			}
			query.Select = append(query.Select, item)
		}
		if !p.accept(tokenComma) {
			return nil
		}
	}
}

// parseWhere consumes the optional WHERE predicate.
func (p *queryParser) parseWhere(query *Query) error {
	if !p.acceptWord("where") {
		return nil
	}
	where, err := p.expression(0)
	if err != nil {
		return err
	}
	query.Where = where
	return nil
}

// parseGroupBy consumes the optional GROUP BY expression list and the HAVING
// predicate that upstream's grammar only admits after it.
func (p *queryParser) parseGroupBy(query *Query) error {
	if !p.acceptWord("group") {
		return nil
	}
	if !p.acceptWord("by") {
		return p.errorf("GROUP requires BY")
	}
	groupBy, err := p.expressionList()
	if err != nil {
		return err
	}
	query.GroupBy = groupBy
	if p.acceptWord("having") {
		having, err := p.expression(0)
		if err != nil {
			return err
		}
		query.Having = having
	}
	return nil
}

// parsePivotBy consumes the optional PIVOT BY pair. Upstream accepts exactly
// two columns and places the clause between ORDER BY and LIMIT.
func (p *queryParser) parsePivotBy(query *Query) error {
	if !p.acceptWord("pivot") {
		return nil
	}
	if !p.acceptWord("by") {
		return p.errorf("PIVOT requires BY")
	}
	pivot, err := p.expressionList()
	if err != nil {
		return err
	}
	if len(pivot) != 2 {
		return p.errorf("PIVOT BY requires two columns")
	}
	query.Pivot = pivot
	return nil
}

// parseOrderBy consumes the optional ORDER BY list with per-item ASC/DESC.
func (p *queryParser) parseOrderBy(query *Query) error {
	if !p.acceptWord("order") {
		return nil
	}
	if !p.acceptWord("by") {
		return p.errorf("ORDER requires BY")
	}
	for {
		expr, err := p.expression(0)
		if err != nil {
			return err
		}
		item := OrderItem{Expr: expr}
		if p.acceptWord("desc") {
			item.Desc = true
		} else {
			p.acceptWord("asc")
		}
		query.OrderBy = append(query.OrderBy, item)
		if !p.accept(tokenComma) {
			return nil
		}
	}
}

// parseLimit consumes the optional non-negative LIMIT n.
func (p *queryParser) parseLimit(query *Query) error {
	if !p.acceptWord("limit") {
		return nil
	}
	token := p.next()
	if token.kind != tokenNumber {
		return p.errorAtf(token, "LIMIT requires a number")
	}
	limit, err := strconv.Atoi(token.text)
	if err != nil || limit < 0 {
		return p.errorAtf(token, "invalid LIMIT")
	}
	query.Limit = limit
	query.LimitSpecified = true
	return nil
}

func (p *queryParser) expression(minPrecedence int) (Expr, error) {
	left, err := p.primary()
	if err != nil {
		return nil, err
	}
	for {
		token := p.peek()
		precedence := operatorPrecedence(token)
		isNull := p.defaultPostings && token.kind == tokenWord && strings.EqualFold(token.text, "is")
		if isNull {
			precedence = 3
		}
		isBetween := p.defaultPostings && token.kind == tokenWord && strings.EqualFold(token.text, "between")
		isNotBetween := p.defaultPostings && token.kind == tokenWord && strings.EqualFold(token.text, "not") && p.peekWord(1, "between")
		isNotIn := p.defaultPostings && token.kind == tokenWord && strings.EqualFold(token.text, "not") && p.peekWord(1, "in")
		if isBetween || isNotBetween || isNotIn {
			precedence = 3
		}
		if precedence < minPrecedence {
			break
		}
		if isBetween || isNotBetween {
			if isNotBetween {
				p.next()
			}
			p.next()
			lower, err := p.expression(4)
			if err != nil {
				return nil, err
			}
			if !p.acceptWord("and") {
				return nil, p.errorf("BETWEEN requires AND")
			}
			upper, err := p.expression(4)
			if err != nil {
				return nil, err
			}
			bounds := binaryExpr{
				operator: "and",
				left:     binaryExpr{operator: ">=", left: left, right: lower, span: &ByteRange{Start: token.start, End: token.end}},
				right:    binaryExpr{operator: "<=", left: left, right: upper, span: &ByteRange{Start: token.start, End: token.end}},
				span:     &ByteRange{Start: token.start, End: token.end},
			}
			if isNotBetween {
				left = unaryExpr{operator: "not", inner: bounds, span: &ByteRange{Start: token.start, End: token.end}}
			} else {
				left = bounds
			}
			continue
		}
		p.next()
		if isNull {
			operator := "is null"
			if p.acceptWord("not") {
				operator = "is not null"
			}
			if !p.acceptWord("null") {
				return nil, p.errorf("IS requires NULL or NOT NULL")
			}
			left = unaryExpr{operator: operator, inner: left, span: &ByteRange{Start: token.start, End: token.end}}
			continue
		}
		if p.defaultPostings && token.kind == tokenWord && strings.EqualFold(token.text, "in") {
			var membershipErr error
			left, membershipErr = p.membership(left, false, token)
			if membershipErr != nil {
				return nil, membershipErr
			}
			continue
		}
		if p.defaultPostings && token.kind == tokenWord && strings.EqualFold(token.text, "not") && p.peekWord(0, "in") {
			p.next()
			var membershipErr error
			left, membershipErr = p.membership(left, true, token)
			if membershipErr != nil {
				return nil, membershipErr
			}
			continue
		}
		right, rightErr := p.expression(precedence + 1)
		if rightErr != nil {
			return nil, rightErr
		}
		left = binaryExpr{operator: strings.ToLower(token.text), left: left, right: right, span: &ByteRange{Start: token.start, End: token.end}}
	}
	return left, nil
}

// primary parses one primary expression: literals, parenthesized groups,
// unary operators, function calls, and field references.
func (p *queryParser) primary() (Expr, error) {
	if p.defaultPostings && p.index >= len(p.tokens) {
		return nil, p.errorf("unexpected end of expression")
	}
	token := p.next()
	switch token.kind {
	case tokenString:
		return literalExpr{span: &ByteRange{Start: token.start, End: token.end}, value: token.text, display: strconv.Quote(token.text)}, nil
	case tokenNumber:
		decimal, err := ledger.ParseDecimal(token.text)
		if err != nil {
			return nil, p.errorAtf(token, "invalid number %q", token.text)
		}
		return literalExpr{span: &ByteRange{Start: token.start, End: token.end}, value: decimal, display: token.text}, nil
	case tokenDate:
		parsed, err := time.Parse("2006-01-02", token.text)
		if err != nil || parsed.Year() < 1 {
			return nil, p.errorAtf(token, "invalid date %q", token.text)
		}
		return literalExpr{span: &ByteRange{Start: token.start, End: token.end}, value: parseDate(token.text), display: token.text}, nil
	case tokenLParen:
		return p.parenthesized()
	case tokenOperator:
		if p.defaultPostings && token.text == "+" {
			return p.primary()
		}
		if token.text == "-" {
			inner, err := p.primary()
			if err != nil {
				return nil, err
			}
			return unaryExpr{operator: "-", inner: inner, span: &ByteRange{Start: token.start, End: token.end}}, nil
		}
		return nil, p.errorAtf(token, "unexpected operator %q", token.text)
	case tokenWord:
		return p.wordPrimary(token)
	case tokenStar:
		return starExpr{}, nil
	default:
		return nil, p.errorAtf(token, "unexpected token %q", token.text)
	}
}

// parenthesized parses "( expression )".
func (p *queryParser) parenthesized() (Expr, error) {
	expr, err := p.expression(0)
	if err != nil {
		return nil, err
	}
	if !p.accept(tokenRParen) {
		return nil, p.errorf("missing closing parenthesis")
	}
	if p.defaultPostings {
		return parenExpr{inner: expr}, nil
	}
	return expr, nil
}

func (p *queryParser) intervalPrimary(token queryToken) (Expr, error) {
	if !p.accept(tokenLParen) {
		return nil, p.errorAtf(token, "interval requires an argument")
	}
	inner, err := p.expression(0)
	if err != nil {
		return nil, err
	}
	if !p.accept(tokenRParen) {
		return nil, p.errorAtf(token, "interval requires one argument")
	}
	return intervalExpr{text: inner, span: &ByteRange{Start: token.start, End: token.end}}, nil
}

func (p *queryParser) wordPrimary(token queryToken) (Expr, error) {
	if token.text == "" {
		return nil, p.errorAtf(token, "unexpected end of expression")
	}
	if p.defaultPostings && strings.EqualFold(token.text, "interval") {
		return p.intervalPrimary(token)
	}
	if p.defaultPostings {
		switch strings.ToLower(token.text) {
		case "null":
			return literalExpr{span: &ByteRange{Start: token.start, End: token.end}, value: nil, display: "NULL"}, nil
		case "true", "false":
			return literalExpr{span: &ByteRange{Start: token.start, End: token.end}, value: strings.EqualFold(token.text, "true"), display: strings.ToUpper(token.text)}, nil
		}
	}
	if strings.EqualFold(token.text, "not") {
		var inner Expr
		var err error
		if p.defaultPostings {
			inner, err = p.expression(3)
		} else {
			inner, err = p.primary()
		}
		if err != nil {
			return nil, err
		}
		return unaryExpr{operator: "not", inner: inner, span: &ByteRange{Start: token.start, End: token.end}}, nil
	}
	if p.accept(tokenLParen) {
		args, err := p.callArguments()
		if err != nil {
			return nil, err
		}
		return functionExpr{name: strings.ToLower(token.text), args: args, span: &ByteRange{Start: token.start, End: token.end}}, nil
	}
	return fieldExpr{name: token.text, span: &ByteRange{Start: token.start, End: token.end}}, nil
}

// callArguments parses a function argument list up to the closing
// parenthesis; a bare * inside the list (as in count(*)) is an argument.
// intervalExpr is a relativedelta value produced by interval(text). Its
// value is a period description, never a database row.
type intervalExpr struct {
	text Expr
	span *ByteRange
}

func (e intervalExpr) String() string  { return "interval(" + e.text.String() + ")" }
func (e intervalExpr) aggregate() bool { return false }
func (e intervalExpr) eval(row Row, rows []Row) (any, error) {
	value, err := e.text.eval(row, rows)
	if err != nil || value == nil {
		return nil, err
	}
	text, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("interval requires a string period")
	}
	interval, err := parseInterval(text)
	if err != nil {
		return nil, err
	}
	return interval, nil
}

func (p *queryParser) callArguments() ([]Expr, error) {
	args := make([]Expr, 0)
	if p.accept(tokenRParen) {
		return args, nil
	}
	for {
		if p.accept(tokenStar) {
			args = append(args, starExpr{})
		} else {
			expr, err := p.expression(0)
			if err != nil {
				return nil, err
			}
			args = append(args, expr)
		}
		if p.accept(tokenRParen) {
			return args, nil
		}
		if !p.accept(tokenComma) {
			return nil, p.errorf("function arguments require commas")
		}
	}
}

// membership parses "IN (a, b, ...)" and "IN container". The list form is
// desugared into equality connectives so membership needs no new AST node.
func (p *queryParser) membership(left Expr, negate bool, token queryToken) (Expr, error) {
	span := &ByteRange{Start: token.start, End: token.end}
	if !p.accept(tokenLParen) {
		container, err := p.expression(4)
		if err != nil {
			return nil, err
		}
		operator := "in"
		if negate {
			operator = "not in"
		}
		return binaryExpr{operator: operator, left: left, right: container, span: span}, nil
	}
	values, err := p.expressionList()
	if err != nil {
		return nil, err
	}
	if !p.accept(tokenRParen) {
		return nil, p.errorf("missing closing parenthesis after IN list")
	}
	if len(values) == 0 {
		return literalExpr{value: negate, display: map[bool]string{true: "TRUE", false: "FALSE"}[negate]}, nil
	}
	operator := "="
	if negate {
		operator = "!="
	}
	var combined Expr
	for _, value := range values {
		comparison := binaryExpr{operator: operator, left: left, right: value, span: span}
		if combined == nil {
			combined = comparison
			continue
		}
		connective := "or"
		if negate {
			connective = "and"
		}
		combined = binaryExpr{operator: connective, left: combined, right: comparison, span: span}
	}
	return combined, nil
}

func (p *queryParser) peekWord(offset int, value string) bool {
	index := p.index + offset
	if index >= len(p.tokens) {
		return false
	}
	token := p.tokens[index]
	return token.kind == tokenWord && strings.EqualFold(token.text, value)
}

func (p *queryParser) expressionList() ([]Expr, error) {
	list := make([]Expr, 0)
	for {
		expr, err := p.expression(0)
		if err != nil {
			return nil, err
		}
		list = append(list, expr)
		if !p.accept(tokenComma) {
			return list, nil
		}
	}
}

func (p *queryParser) accept(kind queryTokenKind) bool {
	if p.peek().kind != kind {
		return false
	}
	p.index++
	return true
}

func (p *queryParser) acceptWord(value string) bool {
	if p.peek().kind != tokenWord || !strings.EqualFold(p.peek().text, value) {
		return false
	}
	p.index++
	return true
}

func (p *queryParser) expectWord(label string) (string, error) {
	token := p.next()
	if token.kind != tokenWord {
		return "", p.errorAtf(token, "expected %s", label)
	}
	return token.text, nil
}

func (p *queryParser) peek() queryToken {
	if p.index >= len(p.tokens) {
		return queryToken{start: p.inputBytes, end: p.inputBytes}
	}
	return p.tokens[p.index]
}

func (p *queryParser) next() queryToken {
	token := p.peek()
	if p.index < len(p.tokens) {
		p.index++
	}
	return token
}

func (p *queryParser) errorf(format string, args ...any) error {
	return p.errorAtf(p.peek(), format, args...)
}

func (p *queryParser) errorAtf(token queryToken, format string, args ...any) error {
	failure := ParseError{Message: fmt.Sprintf(format, args...)}
	if p.defaultPostings {
		failure.Range = &ByteRange{Start: token.start, End: token.end}
	}
	return failure
}

// operatorPrecedence ranks query operators for expression parsing:
// OR < AND < comparisons < + - < * /.
func operatorPrecedence(token queryToken) int {
	if token.kind == tokenStar {
		return 5
	}
	if token.kind == tokenWord {
		switch strings.ToLower(token.text) {
		case "or":
			return 1
		case "and":
			return 2
		}
	}
	if token.kind == tokenWord {
		switch strings.ToLower(token.text) {
		case "in", "between":
			return 3
		}
	}
	if token.kind != tokenOperator {
		return -1
	}
	switch token.text {
	case "=", "!=", "<>", "<", "<=", ">", ">=", "~", "!~", "?~":
		return 3
	case "+", "-":
		return 4
	case "*", "/", "%":
		return 5
	default:
		return -1
	}
}

// targetSchemas lists every column each query table exposes. Column
// references are validated against this static schema before evaluation so a
// typo like sum(amount) fails loudly instead of silently returning zeroes.
var targetSchemas = map[string][]string{
	"postings": {"account", "balance", "cost", "currency", "date", "day", "file", "flag", "kind", "links", "month", "narration", "number", "payee", "posting_flag", "span", "tags", "units", "weight", "year"},
	"entries":  {"date", "file", "kind", "span"},
	"accounts": {"account", "balance", "currency", "opened"},
	"prices":   {"amount", "currency", "date", "quote_currency"},
}

// prepareQuery resolves select-list aliases referenced by GROUP BY / ORDER BY
// (SELECT year(date) AS y ... GROUP BY y) and then rejects unknown column
// names with the table's valid columns, honoring the honest-filtering
// principle: an unanswerable query must say so, never return plausible
// zeroes.
func prepareQuery(query *Query) error {
	resolveGroupAliases(query)
	if err := validateOrderByNames(query); err != nil {
		return err
	}
	if query.Having != nil && !query.Having.aggregate() {
		return fmt.Errorf("the HAVING clause must be an aggregate expression")
	}
	return validateColumnNames(query)
}

// resolveGroupAliases rewrites bare GROUP BY fields that name a select alias
// into the aliased expression, so GROUP BY y works for SELECT year(date) AS y.
func resolveGroupAliases(query *Query) {
	aliases := map[string]Expr{}
	for _, item := range query.Select {
		if item.Alias != "" {
			aliases[strings.ToLower(item.Alias)] = item.Expr
		}
	}
	for i, expr := range query.GroupBy {
		if field, ok := expr.(fieldExpr); ok {
			if aliased, ok := aliases[strings.ToLower(field.name)]; ok {
				query.GroupBy[i] = aliased
			}
		}
	}
}

// validateOrderByNames accepts ORDER BY fields that name an output column or
// a source-table column (upstream bean-query lets ORDER BY reference any
// source expression; projectGroups carries the value), and fails loudly on a
// name that is neither — a typo would otherwise silently no-op the sort.
func validateOrderByNames(query *Query) error {
	if _, isStar := starSelect(query); isStar {
		return nil // a star select projects every schema column; any name sorts
	}
	schema := targetSchemas[canonicalTable(query.From)]
	for i := range query.OrderBy {
		field, ok := query.OrderBy[i].Expr.(fieldExpr)
		if !ok {
			continue
		}
		known := validOutputName(query, field.name)
		for _, column := range schema {
			if strings.EqualFold(column, field.name) {
				known = true
			}
		}
		if !known {
			return fmt.Errorf("unknown column %q in ORDER BY", field.name)
		}
	}
	return nil
}

// validateColumnNames rejects SELECT/WHERE/GROUP BY column references the
// table does not carry, listing the valid columns. ORDER BY is deliberately
// absent: it references projected output columns, not source-table columns.
func validateColumnNames(query *Query) error {
	schema := targetSchemas[canonicalTable(query.From)]
	if schema == nil {
		return nil // unknown tables are diagnosed by rowsForTable
	}
	valid := make(map[string]bool, len(schema))
	for _, column := range schema {
		valid[column] = true
	}
	visit := func(expr Expr) error {
		var problem error
		walkFields(expr, func(name string) {
			if !valid[strings.ToLower(name)] && problem == nil {
				problem = fmt.Errorf("unknown column %q (available: %s)", name, strings.Join(schema, ", "))
			}
		})
		return problem
	}
	for _, item := range query.Select {
		if err := visit(item.Expr); err != nil {
			return err
		}
	}
	if err := visit(query.Where); err != nil {
		return err
	}
	for _, expr := range query.GroupBy {
		if err := visit(expr); err != nil {
			return err
		}
	}
	return visit(query.Having)
}

// outputNames lists the projected column names of a select list. A star
// select has no fixed names, so it reports nil and ORDER BY validation is
// skipped by the caller.
func outputNames(query *Query) []string {
	names := make([]string, 0, len(query.Select))
	for _, item := range query.Select {
		if item.Alias != "" {
			names = append(names, item.Alias)
		} else {
			names = append(names, item.Expr.String())
		}
	}
	return names
}

// starSelect reports whether any select item is the bare star.
func starSelect(query *Query) (SelectItem, bool) {
	for _, item := range query.Select {
		if _, ok := item.Expr.(starExpr); ok {
			return item, true
		}
	}
	return SelectItem{}, false
}

func validOutputName(query *Query, name string) bool {
	for _, candidate := range outputNames(query) {
		if strings.EqualFold(candidate, name) {
			return true
		}
	}
	return false
}

// walkFields visits every column reference in an expression tree.
func walkFields(expr Expr, visit func(string)) {
	switch node := expr.(type) {
	case fieldExpr:
		visit(node.name)
	case unaryExpr:
		walkFields(node.inner, visit)
	case parenExpr:
		walkFields(node.inner, visit)
	case binaryExpr:
		walkFields(node.left, visit)
		walkFields(node.right, visit)
	case functionExpr:
		for _, arg := range node.args {
			walkFields(arg, visit)
		}
	}
}

// canonicalTable normalizes a FROM spelling to its schema key.
func canonicalTable(from string) string {
	from = strings.TrimPrefix(strings.TrimSpace(from), "#")
	switch strings.ToLower(from) {
	case "postings", "posting":
		return "postings"
	case "entries", "entry", "directives":
		return "entries"
	case "accounts", "account":
		return "accounts"
	case "prices", "price":
		return "prices"
	case "transactions", "balances", "notes", "events", "documents", "commodities":
		return strings.ToLower(strings.TrimSpace(from))
	default:
		return ""
	}
}

func isClauseWord(value string) bool {
	switch strings.ToLower(value) {
	case "from", "where", "group", "having", "order", "pivot", "limit", "asc", "desc", "by":
		return true
	default:
		return false
	}
}

type fieldExpr struct {
	name string
	span *ByteRange
}

func (e fieldExpr) String() string  { return e.name }
func (e fieldExpr) aggregate() bool { return false }
func (e fieldExpr) eval(row Row, _ []Row) (any, error) {
	if value, ok := row[e.name]; ok {
		return value, nil
	}
	for key, value := range row {
		if strings.EqualFold(key, e.name) {
			return value, nil
		}
	}
	return nil, nil
}

type literalExpr struct {
	span    *ByteRange
	value   any
	display string
}

func (e literalExpr) String() string               { return e.display }
func (e literalExpr) aggregate() bool              { return false }
func (e literalExpr) eval(Row, []Row) (any, error) { return e.value, nil }

type starExpr struct{}

func (starExpr) String() string                     { return "*" }
func (starExpr) aggregate() bool                    { return false }
func (starExpr) eval(row Row, _ []Row) (any, error) { return row, nil }

type unaryExpr struct {
	operator string
	inner    Expr
	span     *ByteRange
}

type parenExpr struct{ inner Expr }

func (e parenExpr) String() string                        { return "(" + e.inner.String() + ")" }
func (e parenExpr) aggregate() bool                       { return e.inner.aggregate() }
func (e parenExpr) eval(row Row, rows []Row) (any, error) { return e.inner.eval(row, rows) }

func (e unaryExpr) String() string  { return e.operator + " " + e.inner.String() }
func (e unaryExpr) aggregate() bool { return e.inner.aggregate() }
func (e unaryExpr) eval(row Row, rows []Row) (any, error) {
	value, err := e.inner.eval(row, rows)
	if err != nil {
		return nil, err
	}
	if e.operator == "not" {
		return !truthy(value), nil
	}
	decimal, ok := decimalValue(value)
	if !ok {
		return nil, fmt.Errorf("unary minus requires a number")
	}
	return decimal.Neg(), nil
}

type binaryExpr struct {
	operator string
	left     Expr
	right    Expr
	span     *ByteRange
}

func (e binaryExpr) String() string {
	return e.left.String() + " " + e.operator + " " + e.right.String()
}
func (e binaryExpr) aggregate() bool { return e.left.aggregate() || e.right.aggregate() }

// eval evaluates a binary operator: boolean connectives, comparisons, and
// arithmetic. Arithmetic requires numbers on both sides; comparisons fall
// back to formatted string order for non-numeric operands.
func (e binaryExpr) eval(row Row, rows []Row) (any, error) {
	left, err := e.left.eval(row, rows)
	if err != nil {
		return nil, err
	}
	right, err := e.right.eval(row, rows)
	if err != nil {
		return nil, err
	}
	switch e.operator {
	case "and":
		return truthy(left) && truthy(right), nil
	case "or":
		return truthy(left) || truthy(right), nil
	default:
		return e.evalComparison(left, right)
	}
}

// evalComparison evaluates comparisons and arithmetic on already-evaluated
// operands.
func (e binaryExpr) evalComparison(left, right any) (any, error) {
	switch e.operator {
	case "=":
		return compareValues(left, right) == 0, nil
	case "!=", "<>":
		return compareValues(left, right) != 0, nil
	case "<":
		return compareValues(left, right) < 0, nil
	case "<=":
		return compareValues(left, right) <= 0, nil
	case ">":
		return compareValues(left, right) > 0, nil
	case ">=":
		return compareValues(left, right) >= 0, nil
	case "~":
		return evalRegexMatch(left, right)
	case "+", "-", "*", "/", "%":
		return e.evalArithmetic(left, right)
	default:
		return nil, fmt.Errorf("unsupported operator %q", e.operator)
	}
}

// regexCache memoizes compiled patterns so a WHERE filter never recompiles
// its regular expression per row. Patterns come from user queries and the
// set in a session is tiny, so the cache is unbounded by design.
var regexCache sync.Map

// evalRegexMatch implements the BeanQuery "~" operator: the right operand is
// a regular expression (Go RE2 syntax, a deliberate subset of Python re)
// matched against the left operand. NULL never matches.
func evalRegexMatch(left, right any) (any, error) {
	pattern, ok := right.(string)
	if !ok {
		return false, fmt.Errorf("~ requires a string pattern, got %v", right)
	}
	text, ok := left.(string)
	if !ok {
		return false, nil
	}
	compiled, ok := regexCache.Load(pattern)
	if !ok {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return false, fmt.Errorf("invalid regular expression %q: %v", pattern, err)
		}
		regexCache.Store(pattern, re)
		compiled = re
	}
	return compiled.(*regexp.Regexp).MatchString(text), nil
}

// evalArithmetic applies +, -, *, / to numeric operands.
func (e binaryExpr) evalArithmetic(left, right any) (any, error) {
	if e.operator == "+" || e.operator == "-" {
		if text, ok := left.(string); ok && isISODate(text) {
			switch value := right.(type) {
			case Interval:
				if e.operator == "-" {
					value.Years, value.Months, value.Days = -value.Years, -value.Months, -value.Days
					value.Hours, value.Minutes, value.Seconds = -value.Hours, -value.Minutes, -value.Seconds
				}
				return addInterval(text, value), nil
			case ledger.Decimal:
				days, ok := wholeDays(value)
				if !ok {
					return nil, fmt.Errorf("date arithmetic requires whole days")
				}
				if e.operator == "-" {
					days = -days
				}
				return shiftISODate(text, days), nil
			case string:
				if e.operator == "-" && isISODate(value) {
					return isoDateDifference(text, value), nil
				}
			}
			return nil, fmt.Errorf("date arithmetic requires an integer or interval")
		}
		if e.operator == "+" {
			if text, ok := right.(string); ok && isISODate(text) {
				if days, ok := wholeDays(left); ok {
					return shiftISODate(text, days), nil
				}
				if interval, ok := left.(Interval); ok {
					return addInterval(text, interval), nil
				}
			}
			if _, ok := left.(Interval); ok {
				if rightInterval, ok := right.(Interval); ok {
					return Interval{Years: left.(Interval).Years + rightInterval.Years, Months: left.(Interval).Months + rightInterval.Months, Days: left.(Interval).Days + rightInterval.Days, Hours: left.(Interval).Hours + rightInterval.Hours, Minutes: left.(Interval).Minutes + rightInterval.Minutes, Seconds: left.(Interval).Seconds + rightInterval.Seconds}, nil
				}
				return nil, fmt.Errorf("interval arithmetic requires a date or interval")
			}
			leftText, leftOK := left.(string)
			rightText, rightOK := right.(string)
			if leftOK && rightOK {
				return leftText + rightText, nil
			}
		}
	}
	if e.operator == "-" {
		if interval, ok := left.(Interval); ok {
			if rightInterval, ok := right.(Interval); ok {
				return Interval{Years: interval.Years - rightInterval.Years, Months: interval.Months - rightInterval.Months, Days: interval.Days - rightInterval.Days, Hours: interval.Hours - rightInterval.Hours, Minutes: interval.Minutes - rightInterval.Minutes, Seconds: interval.Seconds - rightInterval.Seconds}, nil
			}
		}
		// Interval subtraction is only defined against another interval; the
		// numeric fallback below must not mask that with a generic error.
		if _, ok := left.(Interval); ok {
			return nil, fmt.Errorf("interval subtraction requires an interval operand")
		}
	}
	leftNumber, leftOK := decimalValue(left)
	rightNumber, rightOK := decimalValue(right)
	if !leftOK || !rightOK {
		return nil, fmt.Errorf("arithmetic requires numbers")
	}
	switch e.operator {
	case "+":
		return leftNumber.Add(rightNumber), nil
	case "-":
		return leftNumber.Sub(rightNumber), nil
	case "*":
		return leftNumber.Mul(rightNumber), nil
	case "%":
		if rightNumber.IsZero() {
			return nil, nil
		}
		quotient := new(big.Rat).Quo(leftNumber.Rat(), rightNumber.Rat())
		truncated := new(big.Int).Quo(quotient.Num(), quotient.Denom())
		remainder := new(big.Rat).Sub(leftNumber.Rat(), new(big.Rat).Mul(new(big.Rat).SetInt(truncated), rightNumber.Rat()))
		return ledger.NewDecimal(remainder), nil
	default: // "/"
		if rightNumber.IsZero() {
			return nil, fmt.Errorf("division by zero")
		}
		return ledger.NewDecimal(new(big.Rat).Quo(leftNumber.Rat(), rightNumber.Rat())), nil
	}
}

const isoDateLayout = "2006-01-02"

func isISODate(text string) bool {
	if len(text) != len(isoDateLayout) {
		return false
	}
	_, err := time.Parse(isoDateLayout, text)
	return err == nil
}

func wholeDays(value any) (int, bool) {
	number, ok := decimalValue(value)
	if !ok || !number.Rat().IsInt() {
		return 0, false
	}
	days := number.Rat().Num()
	if !days.IsInt64() {
		return 0, false
	}
	return int(days.Int64()), true
}

func shiftISODate(text string, days int) string {
	parsed, err := time.Parse(isoDateLayout, text)
	if err != nil {
		return text
	}
	return parsed.AddDate(0, 0, days).Format(isoDateLayout)
}

func isoDateDifference(left, right string) ledger.Decimal {
	leftDate, leftErr := time.Parse(isoDateLayout, left)
	rightDate, rightErr := time.Parse(isoDateLayout, right)
	if leftErr != nil || rightErr != nil {
		return ledger.Zero()
	}
	return ledger.NewDecimal(new(big.Rat).SetInt64(int64(leftDate.Sub(rightDate).Hours() / 24)))
}

func typedMembership(operand, container any) (bool, error) {
	switch values := container.(type) {
	case stringSet:
		text, ok := operand.(string)
		if !ok {
			return false, nil
		}
		for _, value := range values {
			if value == text {
				return true, nil
			}
		}
		return false, nil
	case []string:
		text, ok := operand.(string)
		if !ok {
			return false, nil
		}
		for _, value := range values {
			if value == text {
				return true, nil
			}
		}
		return false, nil
	case dictionaryValue:
		text, ok := operand.(string)
		if !ok {
			return false, nil
		}
		_, exists := values[text]
		return exists, nil
	case listValue:
		for _, value := range values {
			if compareValues(operand, value) == 0 {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, fmt.Errorf("membership requires a set, dictionary, or list operand")
	}
}

type functionExpr struct {
	name string
	args []Expr
	span *ByteRange
}

func (e functionExpr) String() string {
	args := make([]string, len(e.args))
	for i, arg := range e.args {
		args[i] = arg.String()
	}
	return e.name + "(" + strings.Join(args, ", ") + ")"
}

func (e functionExpr) aggregate() bool {
	switch e.name {
	case "sum", "count", "min", "max", "first", "last", "avg":
		return true
	default:
		for _, arg := range e.args {
			if arg.aggregate() {
				return true
			}
		}
		return false
	}
}

// eval dispatches a function call to its evaluation family. Counting and the
// numeric aggregates fold over the group's rows; date parts and membership
// predicates evaluate their single argument against the current row.
func (e functionExpr) eval(row Row, rows []Row) (any, error) {
	switch e.name {
	case "count":
		return e.evalCount(rows)
	case "sum", "avg", "min", "max", "first", "last":
		return e.evalAggregate(row, rows)
	case "year", "month", "day":
		return e.evalDatePart(row, rows)
	case "has_tag", "has_link":
		return e.evalMembership(row, rows)
	case "root":
		return e.evalRoot(row, rows)
	default:
		return nil, fmt.Errorf("unsupported function %q", e.name)
	}
}

// evalRoot implements BeanQuery's root(account, n): the first n colon-
// separated components of an account name. n larger than the depth returns
// the whole name; n <= 0 returns an empty string.
func (e functionExpr) evalRoot(row Row, rows []Row) (any, error) {
	if len(e.args) != 2 {
		return nil, fmt.Errorf("root requires two arguments")
	}
	value, err := e.args[0].eval(row, rows)
	if err != nil {
		return nil, err
	}
	name, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("root requires an account column, got %v", value)
	}
	depth, err := e.args[1].eval(row, rows)
	if err != nil {
		return nil, err
	}
	parts, ok := decimalValue(depth)
	if !ok {
		return nil, fmt.Errorf("root requires a numeric depth, got %v", depth)
	}
	if !parts.Rat().IsInt() {
		return nil, fmt.Errorf("root requires an integer depth, got %v", depth)
	}
	n := int(parts.Rat().Num().Int64())
	if n <= 0 {
		return "", nil
	}
	components := strings.Split(name, ":")
	if n >= len(components) {
		return name, nil
	}
	return strings.Join(components[:n], ":"), nil
}

// evalCount implements count() and count(expr): the bare form counts rows,
// the argument form counts rows where the argument is non-NULL.
func (e functionExpr) evalCount(rows []Row) (any, error) {
	if len(e.args) == 0 || (len(e.args) == 1 && e.args[0].String() == "*") {
		return ledger.NewDecimal(big.NewRat(int64(len(rows)), 1)), nil
	}
	count := 0
	for _, candidate := range rows {
		value, err := e.args[0].eval(candidate, []Row{candidate})
		if err != nil {
			return nil, err
		}
		if value != nil {
			count++
		}
	}
	return ledger.NewDecimal(big.NewRat(int64(count), 1)), nil
}

// evalAggregate implements sum/avg/min/max/first/last over the group's rows;
// rows where the argument is missing or non-numeric are skipped. An empty
// input yields 0 (matching the workbench's deterministic presentation).
func (e functionExpr) evalAggregate(row Row, rows []Row) (any, error) {
	if len(e.args) != 1 {
		return nil, fmt.Errorf("%s requires one argument", e.name)
	}
	values := make([]ledger.Decimal, 0, len(rows))
	for _, candidate := range rows {
		value, err := e.args[0].eval(candidate, []Row{candidate})
		if err != nil {
			return nil, err
		}
		if value == nil {
			continue // NULL rows (e.g. an unpriced posting's cost) stay skipped
		}
		decimal, ok := decimalValue(value)
		if !ok {
			return nil, fmt.Errorf("%s cannot aggregate %q values; use a numeric column", e.name, e.args[0].String())
		}
		values = append(values, decimal)
	}
	if len(values) == 0 {
		return ledger.Zero(), nil
	}
	switch e.name {
	case "sum":
		return sumDecimals(values), nil
	case "avg":
		sum, _ := functionExpr{name: "sum", args: e.args}.eval(row, rows)
		return ledger.NewDecimal(new(big.Rat).Quo(sum.(ledger.Decimal).Rat(), big.NewRat(int64(len(values)), 1))), nil
	case "min", "max":
		return extremalDecimal(values, e.name == "min"), nil
	case "first":
		return values[0], nil
	default: // "last"
		return values[len(values)-1], nil
	}
}

// evalDatePart implements year()/month()/day() over a date (or date string).
func (e functionExpr) evalDatePart(row Row, rows []Row) (any, error) {
	if len(e.args) != 1 {
		return nil, fmt.Errorf("%s requires one argument", e.name)
	}
	value, err := e.args[0].eval(row, rows)
	if err != nil {
		return nil, err
	}
	date, ok := value.(ledger.Date)
	if !ok {
		if raw, isString := value.(string); isString {
			date = parseDate(raw)
		}
	}
	var part int
	switch e.name {
	case "year":
		part = date.Year
	case "month":
		part = date.Month
	default: // "day"
		part = date.Day
	}
	return ledger.NewDecimal(big.NewRat(int64(part), 1)), nil
}

// evalMembership implements has_tag/has_link: exact membership of a value in
// a collection such as the tags or links column.
func (e functionExpr) evalMembership(row Row, rows []Row) (any, error) {
	if len(e.args) != 2 {
		return nil, fmt.Errorf("%s requires a collection and value", e.name)
	}
	collection, err := e.args[0].eval(row, rows)
	if err != nil {
		return nil, err
	}
	needle, err := e.args[1].eval(row, rows)
	if err != nil {
		return nil, err
	}
	needleString := formatValue(needle)
	if values, ok := collection.([]string); ok {
		for _, value := range values {
			if value == needleString {
				return true, nil
			}
		}
	}
	return false, nil
}

// sumDecimals adds a list of decimals.
func sumDecimals(values []ledger.Decimal) ledger.Decimal {
	result := ledger.Zero()
	for _, value := range values {
		result = result.Add(value)
	}
	return result
}

// extremalDecimal picks the minimum or maximum of a non-empty list.
func extremalDecimal(values []ledger.Decimal, min bool) ledger.Decimal {
	result := values[0]
	for _, value := range values[1:] {
		if (min && value.Cmp(result) < 0) || (!min && value.Cmp(result) > 0) {
			result = value
		}
	}
	return result
}

func decimalValue(value any) (ledger.Decimal, bool) {
	switch typed := value.(type) {
	case ledger.Decimal:
		return typed, true
	case ledger.Number:
		return ledger.DecimalFromNumber(typed), true
	case int:
		return ledger.NewDecimal(big.NewRat(int64(typed), 1)), true
	case int64:
		return ledger.NewDecimal(big.NewRat(typed, 1)), true
	case float64:
		decimal, err := ledger.ParseDecimal(strconv.FormatFloat(typed, 'f', -1, 64))
		return decimal, err == nil
	case string:
		decimal, err := ledger.ParseDecimal(typed)
		return decimal, err == nil
	default:
		return ledger.Zero(), false
	}
}

func truthy(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case ledger.Decimal:
		return !typed.IsZero()
	case string:
		return typed != ""
	default:
		return true
	}
}

func compareValues(left, right any) int {
	if leftNumber, ok := decimalValue(left); ok {
		if rightNumber, ok := decimalValue(right); ok {
			return leftNumber.Cmp(rightNumber)
		}
	}
	leftString, rightString := formatValue(left), formatValue(right)
	if leftString < rightString {
		return -1
	}
	if leftString > rightString {
		return 1
	}
	return 0
}

func formatValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case ledger.Decimal:
		return typed.String()
	case ledger.Date:
		return typed.Raw
	case ledger.DirectiveKind:
		return string(typed)
	case []string:
		return strings.Join(typed, ",")
	case bool:
		return strconv.FormatBool(typed)
	default:
		return fmt.Sprint(typed)
	}
}

func parseDate(raw string) ledger.Date {
	parts := strings.Split(raw, "-")
	if len(parts) != 3 {
		return ledger.Date{Raw: raw}
	}
	year, _ := strconv.Atoi(parts[0])
	month, _ := strconv.Atoi(parts[1])
	day, _ := strconv.Atoi(parts[2])
	return ledger.Date{Year: year, Month: month, Day: day, Raw: raw}
}
