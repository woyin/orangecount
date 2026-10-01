// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package query

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"orangecount/internal/ledger"
)

const (
	QueryTextLimit = 64 * 1024
	QueryRowLimit  = 10000
	QueryByteLimit = 16 * 1024 * 1024
	QueryStepLimit = 2000000
)

// ByteRange uses UTF-8 byte offsets and an exclusive end.
type ByteRange struct {
	Start int `json:"start_byte"`
	End   int `json:"end_byte"`
}

// ExecutionLimits may lower the native execution ceilings, never raise them.
type ExecutionLimits struct {
	Rows  int
	Bytes int
	Steps uint64
}

type Execution struct {
	Result       TypedResult       `json:"result"`
	ReturnedRows int               `json:"returned_rows"`
	RowLimit     int               `json:"row_limit"`
	Truncated    bool              `json:"truncated"`
	PhaseSteps   map[string]uint64 `json:"-"`
}

type executionBudget struct {
	ctx      context.Context
	limits   ExecutionLimits
	used     uint64
	phase    string
	counts   map[string]uint64
	patterns map[string]*regexp.Regexp
}

func (b *executionBudget) step(phase string) error {
	if b == nil {
		return nil
	}
	if err := b.ctx.Err(); err != nil {
		return err
	}
	if phase != "" {
		b.phase = phase
	}
	b.used++
	b.counts[b.phase]++
	if b.used > b.limits.Steps {
		return TypedError{Code: "work-budget-exceeded", Message: "Query execution exceeded its work budget", Phase: b.phase}
	}
	return nil
}

func rowBudget(row Row, rows []Row) *executionBudget {
	if value, ok := row["_query_execution"].(*executionBudget); ok {
		return value
	}
	if len(rows) != 0 {
		value, _ := rows[0]["_query_execution"].(*executionBudget)
		return value
	}
	return nil
}

func (b *executionBudget) decimalText(text string) error {
	if b == nil {
		return nil
	}
	text = normalizeDecimalInput(text)
	if !metadataDecimalSyntax.MatchString(text) {
		return nil
	}
	if len(text) > 4096 {
		return typedFailure("numeric-budget-exceeded", "Decimal coefficient exceeds the interactive budget")
	}
	if i := strings.IndexAny(text, "eE"); i >= 0 {
		exponent, err := strconv.ParseInt(text[i+1:], 10, 64)
		if err != nil || exponent < -10000 || exponent > 10000 {
			return typedFailure("numeric-budget-exceeded", "Decimal exponent exceeds the interactive budget")
		}
	}
	return b.step("")
}

// Bound operands before they can feed another exact rational operation.
// One multiplication may transiently double this size; subsequent growth is
// rejected, so repeated arithmetic cannot allocate exponentially large values.
func (b *executionBudget) numericValue(value any) error {
	if b == nil {
		return nil
	}
	value = unwrapMetadataNumber(value)
	switch n := value.(type) {
	case ledger.Decimal:
		if err := b.step(""); err != nil {
			return err
		}
		r := n.Rat()
		if r.Num().BitLen() > 65536 || r.Denom().BitLen() > 65536 {
			return typedFailure("numeric-budget-exceeded", "Query number exceeds the interactive size budget")
		}
	case amountValue:
		return b.numericValue(n.Number)
	case positionValue:
		if err := b.numericValue(n.Units.Number); err != nil {
			return err
		}
		if n.Cost != nil {
			return b.numericValue(n.Cost.Number)
		}
	case inventoryValue:
		for _, position := range n {
			if err := b.numericValue(position); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseTyped(text string, budget *executionBudget) (Query, error) {
	if budget == nil {
		return ParseStandard(text)
	}
	if !utf8.ValidString(text) {
		return Query{}, typedFailure("syntax", "Query text is not valid UTF-8")
	}
	if err := budget.step("parse"); err != nil {
		return Query{}, err
	}
	tokens := lexQuery(text, true)
	if len(tokens) > 4096 {
		return Query{}, typedFailure("complexity-budget-exceeded", "Query exceeds the token budget")
	}
	depth := 0
	for _, token := range tokens {
		if err := budget.step(""); err != nil {
			return Query{}, err
		}
		if token.kind == tokenLParen {
			depth++
		}
		if depth > 64 {
			return Query{}, typedFailure("complexity-budget-exceeded", "Query exceeds the expression depth budget")
		}
		if token.kind == tokenRParen {
			depth--
		}
		if token.kind == tokenNumber {
			if err := budget.decimalText(token.text); err != nil {
				return Query{}, err
			}
		}
	}
	parser := queryParser{tokens: tokens, defaultPostings: true}
	q, err := parser.parse()
	if err != nil {
		return Query{}, typedFailure("syntax", "%s", err)
	}
	type pending struct {
		expr  Expr
		depth int
	}
	stack := []pending{}
	for _, item := range q.Select {
		stack = append(stack, pending{item.Expr, 1})
	}
	for _, expr := range q.GroupBy {
		stack = append(stack, pending{expr, 1})
	}
	for _, item := range q.OrderBy {
		stack = append(stack, pending{item.Expr, 1})
	}
	if q.Where != nil {
		stack = append(stack, pending{q.Where, 1})
	}
	for len(stack) != 0 {
		last := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if err := budget.step(""); err != nil {
			return Query{}, err
		}
		if last.depth > 64 {
			return Query{}, typedFailure("complexity-budget-exceeded", "Query exceeds the expression depth budget")
		}
		push := func(expr Expr) { stack = append(stack, pending{expr, last.depth + 1}) }
		switch node := last.expr.(type) {
		case binaryExpr:
			push(node.left)
			push(node.right)
		case unaryExpr:
			push(node.inner)
		case parenExpr:
			push(node.inner)
		case functionExpr:
			for _, arg := range node.args {
				push(arg)
			}
		}
	}
	return q, nil
}

func boundedRegex(row Row, rows []Row, left, right any) (any, error) {
	budget := rowBudget(row, rows)
	if budget == nil {
		return evalRegexMatch(left, right)
	}
	pattern, text := right.(string), left.(string)
	if len(pattern) > 4096 {
		return nil, typedFailure("regex-budget-exceeded", "Regular expression exceeds the interactive budget")
	}
	compiled := budget.patterns[pattern]
	if compiled == nil {
		if len(budget.patterns) >= 64 {
			return nil, typedFailure("regex-budget-exceeded", "Query exceeded its regular expression cache budget")
		}
		var err error
		compiled, err = regexp.Compile(pattern)
		if err != nil {
			return nil, typedFailure("syntax", "Invalid regular expression")
		}
		budget.patterns[pattern] = compiled
	}
	if err := budget.step(""); err != nil {
		return nil, err
	}
	return compiled.MatchString(text), nil
}

// ExecuteTyped is the bounded entry for interactive native executions.
// EvaluateTyped retains the existing CLI and public compatibility contract.
func ExecuteTyped(ctx context.Context, text string, evaluation ledger.Evaluation, limits ExecutionLimits) (Execution, error) {
	if err := ctx.Err(); err != nil {
		return Execution{}, err
	}
	if len(text) > QueryTextLimit {
		return Execution{}, typedFailure("text-budget-exceeded", "Query text exceeds 64 KiB of UTF-8")
	}
	if limits.Rows <= 0 || limits.Rows > QueryRowLimit {
		limits.Rows = QueryRowLimit
	}
	if limits.Bytes <= 0 || limits.Bytes > QueryByteLimit {
		limits.Bytes = QueryByteLimit
	}
	if limits.Steps == 0 || limits.Steps > QueryStepLimit {
		limits.Steps = QueryStepLimit
	}
	budget := &executionBudget{ctx: ctx, limits: limits, counts: map[string]uint64{}, patterns: map[string]*regexp.Regexp{}}
	result, err := evaluateTyped(text, evaluation, limits.Rows+1, budget)
	if err != nil {
		return Execution{}, err
	}
	execution := Execution{Result: result, RowLimit: limits.Rows, PhaseSteps: budget.counts}
	if len(execution.Result.Rows) > limits.Rows {
		execution.Truncated = true
		execution.Result.Rows = execution.Result.Rows[:limits.Rows]
	}
	execution.ReturnedRows = len(execution.Result.Rows)
	encoded, err := json.Marshal(execution)
	if err != nil {
		return Execution{}, err
	}
	if len(encoded) > limits.Bytes {
		return Execution{}, typedFailure("result-budget-exceeded", "Query result exceeds the byte budget")
	}
	if err := ctx.Err(); err != nil {
		return Execution{}, err
	}
	return execution, nil
}
