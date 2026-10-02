// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package query

import (
	"errors"
	"strings"
)

// Token spans originate in the lexer, never from searching a displayed name.
func expressionRange(expr Expr) *ByteRange {
	switch node := expr.(type) {
	case typedExpr:
		return node.span
	case literalExpr:
		return node.span
	case fieldExpr:
		return node.span
	case functionExpr:
		return node.span
	case binaryExpr:
		return node.span
	case unaryExpr:
		return node.span
	case parenExpr:
		return expressionRange(node.inner)
	}
	return nil
}

func locatedQueryError(err error, span *ByteRange, phase string) error {
	var typed TypedError
	if errors.As(err, &typed) {
		if typed.Range == nil && span != nil && !strings.HasSuffix(typed.Code, "-budget-exceeded") {
			location := *span
			typed.Range = &location
		}
		if typed.Phase == "" {
			typed.Phase = phase
		}
		return typed
	}
	var parse ParseError
	if errors.As(err, &parse) {
		return TypedError{Code: "syntax", Message: parse.Message, Phase: "parse", Range: parse.Range}
	}
	return err
}
