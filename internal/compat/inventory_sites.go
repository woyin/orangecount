// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package compat

// directiveSites maps each directive capability to the source-table branch that
// reads it. Pad, Query, and Custom are recognized through their generic
// directive record rather than a dedicated table.
var directiveSites = map[string]string{
	"directive.Open":        `case ledger.Open:`,
	"directive.Close":       `case ledger.Close:`,
	"directive.Commodity":   `case ledger.Commodity:`,
	"directive.Balance":     `case ledger.Balance:`,
	"directive.Note":        `case ledger.Note:`,
	"directive.Event":       `case ledger.Event:`,
	"directive.Document":    `case ledger.Document:`,
	"directive.Price":       `case ledger.Price:`,
	"directive.Pad":         `func directiveAccounts`,
	"directive.Query":       `func directiveAccounts`,
	"directive.Custom":      `func directiveAccounts`,
	"directive.Transaction": `case ledger.Transaction:`,
}

// optionSites maps each accounting option to the evaluator that consumes it.
var optionSites = map[string]string{
	"option.booking_method":                `case "booking_method":`,
	"option.tolerance_multiplier":          `case "tolerance_multiplier", "inferred_tolerance_multiplier":`,
	"option.inferred_tolerance_multiplier": `case "tolerance_multiplier", "inferred_tolerance_multiplier":`,
}

// syntaxSites maps each query grammar node to the parser or evaluator branch
// that implements it.
var syntaxSites = map[string]string{
	"syntax.Add":       `case "+", "-", "*", "/", "%":`,
	"syntax.Sub":       `case "+", "-", "*", "/", "%":`,
	"syntax.Mul":       `case "+", "-", "*", "/", "%":`,
	"syntax.Div":       `case "+", "-", "*", "/", "%":`,
	"syntax.Mod":       `case "+", "-", "*", "/", "%":`,
	"syntax.Neg":       `unary minus requires a numeric operand`,
	"syntax.Not":       `node.operator == "not"`,
	"syntax.And":       `case "and", "or":`,
	"syntax.Or":        `case "and", "or":`,
	"syntax.Equal":     `func (e binaryExpr) evalComparison`,
	"syntax.NotEqual":  `func (e binaryExpr) evalComparison`,
	"syntax.Greater":   `func (e binaryExpr) evalComparison`,
	"syntax.GreaterEq": `func (e binaryExpr) evalComparison`,
	"syntax.Less":      `func (e binaryExpr) evalComparison`,
	"syntax.LessEq":    `func (e binaryExpr) evalComparison`,
	"syntax.Match":     `func boundedRegex`,
	"syntax.NotMatch":  `func boundedRegex`,
	"syntax.Matches":   `func boundedRegex`,
	"syntax.In":        `func typedMembership`,
	"syntax.NotIn":     `func typedMembership`,
	"syntax.IsNull":    `node.operator == "is null"`,
	"syntax.IsNotNull": `node.operator == "is null"`,
	"syntax.From":      `func (p *queryParser) parseFrom`,
	// Ordering is the ASC/DESC enum consumed by parseOrderBy.
	"syntax.Ordering": `// parseOrderBy consumes the optional ORDER BY list`,
	// Node is the AST base contract, expressed by the query AST itself.
	"syntax.Node": `func ASTNodeKinds`,
}
