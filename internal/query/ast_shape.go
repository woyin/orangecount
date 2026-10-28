// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package query

// ASTNodeKinds reports the typed node decomposition of a parsed statement using
// the upstream beanquery.parser.ast node names. The syntax.Node capability is
// the AST base contract: parsing one statement text must decompose into the
// same set of typed nodes, so a probe can compare the two implementations
// without either exposing its private structs.
//
// The mapping is deliberately explicit: a node name appears here only when the
// Go parser has the corresponding construct, and an unknown construct is
// reported by its Go type name instead of being silently dropped.
func ASTNodeKinds(query Query) []string {
	kinds := []string{"Select"}
	if query.From != "" {
		kinds = append(kinds, "Table")
	}
	if query.Distinct {
		kinds = append(kinds, "Distinct")
	}
	for _, item := range query.Select {
		kinds = append(kinds, "Target")
		kinds = append(kinds, exprNodeKinds(item.Expr)...)
	}
	if query.Where != nil {
		kinds = append(kinds, "Where")
		kinds = append(kinds, exprNodeKinds(query.Where)...)
	}
	for _, expr := range query.GroupBy {
		kinds = append(kinds, "GroupBy")
		kinds = append(kinds, exprNodeKinds(expr)...)
	}
	if query.Having != nil {
		kinds = append(kinds, "Having")
		kinds = append(kinds, exprNodeKinds(query.Having)...)
	}
	for _, expr := range query.Pivot {
		kinds = append(kinds, "PivotBy")
		kinds = append(kinds, exprNodeKinds(expr)...)
	}
	for _, item := range query.OrderBy {
		kinds = append(kinds, "OrderBy")
		kinds = append(kinds, exprNodeKinds(item.Expr)...)
	}
	return kinds
}

// operatorNodeNames maps a query operator to its upstream AST node name.
var operatorNodeNames = map[string]string{
	"+": "Add", "-": "Sub", "*": "Mul", "/": "Div", "%": "Mod",
	"=": "Equal", "!=": "NotEqual", ">": "Greater", ">=": "GreaterEq",
	"<": "Less", "<=": "LessEq", "and": "And", "or": "Or",
	"~": "Match", "!~": "NotMatch", "?~": "Matches",
	"in": "In", "not in": "NotIn",
}

func exprNodeKinds(expr Expr) []string {
	switch node := expr.(type) {
	case nil:
		return nil
	case fieldExpr:
		return []string{"Column"}
	case literalExpr:
		return []string{"Constant"}
	case starExpr:
		return []string{"Asterisk"}
	case functionExpr:
		kinds := []string{"Function"}
		for _, arg := range node.args {
			kinds = append(kinds, exprNodeKinds(arg)...)
		}
		return kinds
	case binaryExpr:
		name, ok := operatorNodeNames[node.operator]
		if !ok {
			name = node.operator
		}
		kinds := []string{name}
		kinds = append(kinds, exprNodeKinds(node.left)...)
		kinds = append(kinds, exprNodeKinds(node.right)...)
		return kinds
	case unaryExpr:
		name := node.operator
		switch node.operator {
		case "-":
			name = "Neg"
		case "not":
			name = "Not"
		}
		kinds := []string{name}
		kinds = append(kinds, exprNodeKinds(node.inner)...)
		return kinds
	case parenExpr:
		return exprNodeKinds(node.inner)
	case intervalExpr:
		return []string{"Interval"}
	case typedExpr:
		return []string{"Cast"}
	}
	return []string{"Unknown"}
}
