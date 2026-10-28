// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package query

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"orangecount/internal/ledger"
)

func bigRatInt(value int) *big.Rat { return big.NewRat(int64(value), 1) }

// todayDate returns the wall-clock date the way today() documents it.
func todayDate() string { return time.Now().UTC().Format(isoDateLayout) }

// scalarFunctions lists the BeanQuery scalar functions this compiler binds.
// Aggregate and cast functions keep their dedicated branches.
var scalarFunctionNames = map[string]bool{
	"account_sortkey": true, "close_date": true,
	"commodity": true, "commodity_meta": true, "convert": true,
	"currency_meta": true, "date_add": true, "date_bin": true,
	"date_diff": true, "date_part": true, "date_trunc": true, "empty": true,
	"filter_currency": true, "findfirst": true, "getitem": true,
	"getprice": true, "grep": true, "grepn": true, "has_account": true,
	"interval": true, "joinstr": true, "leaf": true, "length": true,
	"lower": true, "maxwidth": true, "neg": true, "only": true,
	"open_date": true, "open_meta": true, "parent": true, "parse_date": true,
	"possign": true, "quarter": true, "repr": true, "root": true,
	"round": true, "safediv": true, "splitcomp": true, "subst": true,
	"substr": true, "today": true, "upper": true, "value": true,
	"weekday": true, "yearmonth": true,
}

func (compiler typedCompiler) compileScalar(node functionExpr) (typedExpr, bool, error) {
	if !scalarFunctionNames[node.name] {
		return typedExpr{}, false, nil
	}
	compiled, err := compiler.compileScalarCall(node)
	return compiled, true, err
}

func (compiler typedCompiler) compileScalarCall(node functionExpr) (typedExpr, error) {
	fail := func(format string, args ...any) (typedExpr, error) {
		return typedExpr{}, typedFailure("type-mismatch", format, args...)
	}
	args := make([]typedExpr, 0, len(node.args))
	for _, expr := range node.args {
		arg, err := compiler.compile(expr)
		if err != nil {
			return typedExpr{}, err
		}
		args = append(args, arg)
	}
	name := node.name
	expect := func(kinds ...string) error {
		if len(args) != len(kinds) {
			return fmt.Errorf("no typed signature for %s with %d arguments", name, len(args))
		}
		for i, kind := range kinds {
			if kind != "" && args[i].kind != kind {
				return fmt.Errorf("no typed signature for %s(%s)", name, args[i].kind)
			}
		}
		return nil
	}
	first := func(row Row, rows []Row) (any, error) {
		if len(args) == 0 {
			return nil, nil
		}
		return args[0].eval(row, rows)
	}
	stringArg := func(row Row, rows []Row, index int) (string, error) {
		value, err := args[index].eval(row, rows)
		if err != nil || value == nil {
			return "", err
		}
		text, ok := value.(string)
		if !ok {
			return "", fmt.Errorf("argument %d must be a string", index+1)
		}
		return text, nil
	}
	intArg := func(row Row, rows []Row, index int) (int, error) {
		value, err := args[index].eval(row, rows)
		if err != nil || value == nil {
			return 0, err
		}
		number, ok := decimalValue(value)
		if !ok || !number.Rat().IsInt() {
			return 0, fmt.Errorf("argument %d must be an integer", index+1)
		}
		return int(number.Rat().Num().Int64()), nil
	}
	dateArg := func(row Row, rows []Row, index int) (*ledger.Date, error) {
		value, err := args[index].eval(row, rows)
		if err != nil || value == nil {
			return nil, err
		}
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("argument %d must be a date", index+1)
		}
		parsed, err := time.Parse(isoDateLayout, text)
		if err != nil {
			return nil, fmt.Errorf("argument %d must be a date", index+1)
		}
		return &ledger.Date{Raw: parsed.Format(isoDateLayout), Year: parsed.Year(), Month: int(parsed.Month()), Day: parsed.Day()}, nil
	}

	result := typedExpr{name: node.String(), nullable: true, value: first}
	switch name {
	case "today":
		if err := expect(); err != nil {
			return fail("%v", err)
		}
		result.nullable = false
		result.kind = "date"
		result.value = func(Row, []Row) (any, error) { return todayDate(), nil }
		return result, nil
	case "upper", "lower":
		if err := expect("string"); err != nil {
			return fail("%v", err)
		}
		result.kind, result.nullable = "string", args[0].nullable
		result.value = func(row Row, rows []Row) (any, error) {
			text, err := stringArg(row, rows, 0)
			if err != nil || text == "" {
				if text == "" {
					if raw, rawErr := args[0].eval(row, rows); rawErr != nil || raw == nil {
						return nil, rawErr
					}
					return "", nil
				}
				return nil, err
			}
			if name == "upper" {
				return strings.ToUpper(text), nil
			}
			return strings.ToLower(text), nil
		}
		return result, nil
	case "yearmonth":
		if err := expect("date"); err != nil {
			return fail("%v", err)
		}
		result.kind, result.nullable = "date", args[0].nullable
		result.value = func(row Row, rows []Row) (any, error) {
			text, err := stringArg(row, rows, 0)
			if err != nil || text == "" {
				return nil, err
			}
			return yearMonth(text)
		}
		return result, nil
	case "quarter", "weekday":
		if err := expect("date"); err != nil {
			return fail("%v", err)
		}
		result.kind, result.nullable = "string", args[0].nullable
		result.value = func(row Row, rows []Row) (any, error) {
			text, err := stringArg(row, rows, 0)
			if err != nil || text == "" {
				return nil, err
			}
			if name == "quarter" {
				return quarterOf(text)
			}
			return weekdayOf(text)
		}
		return result, nil
	case "date_add":
		if err := expect("date", "integer"); err != nil {
			return fail("%v", err)
		}
		result.kind = "date"
		result.value = func(row Row, rows []Row) (any, error) {
			text, err := stringArg(row, rows, 0)
			if err != nil || text == "" {
				return nil, err
			}
			days, err := intArg(row, rows, 1)
			if err != nil || args[1].nullable && days == 0 {
				if err != nil {
					return nil, err
				}
			}
			return dateAdd(text, days)
		}
		return result, nil
	case "date_diff":
		if err := expect("date", "date"); err != nil {
			return fail("%v", err)
		}
		result.kind = "integer"
		result.value = func(row Row, rows []Row) (any, error) {
			left, err := stringArg(row, rows, 0)
			if err != nil || left == "" {
				return nil, err
			}
			right, err := stringArg(row, rows, 1)
			if err != nil || right == "" {
				return nil, err
			}
			return dateDifference(left, right)
		}
		return result, nil
	case "date_part":
		if err := expect("string", "date"); err != nil {
			return fail("%v", err)
		}
		result.kind = "integer"
		result.value = func(row Row, rows []Row) (any, error) {
			part, err := stringArg(row, rows, 0)
			if err != nil || part == "" {
				return nil, err
			}
			text, err := stringArg(row, rows, 1)
			if err != nil || text == "" {
				return nil, err
			}
			return datePart(part, text)
		}
		return result, nil
	case "date_trunc":
		if err := expect("string", "date"); err != nil {
			return fail("%v", err)
		}
		result.kind = "date"
		result.value = func(row Row, rows []Row) (any, error) {
			part, err := stringArg(row, rows, 0)
			if err != nil || part == "" {
				return nil, err
			}
			text, err := stringArg(row, rows, 1)
			if err != nil || text == "" {
				return nil, err
			}
			return dateTrunc(part, text)
		}
		return result, nil
	case "date_bin":
		if err := expect("interval", "date", "date"); err != nil {
			return fail("%v", err)
		}
		result.kind = "date"
		result.value = func(row Row, rows []Row) (any, error) {
			value, err := args[0].eval(row, rows)
			if err != nil || value == nil {
				return nil, err
			}
			interval := value.(Interval)
			text, err := stringArg(row, rows, 1)
			if err != nil || text == "" {
				return nil, err
			}
			origin, err := stringArg(row, rows, 2)
			if err != nil || origin == "" {
				return nil, err
			}
			return dateBin(interval, text, origin)
		}
		return result, nil
	case "root":
		if len(args) < 1 || len(args) > 2 || args[0].kind != "string" {
			return fail("root requires an account column")
		}
		result.kind = "string"
		result.value = func(row Row, rows []Row) (any, error) {
			text, err := stringArg(row, rows, 0)
			if err != nil {
				return nil, err
			}
			depth := 1
			if len(args) == 2 {
				value, err := intArg(row, rows, 1)
				if err != nil {
					return nil, err
				}
				depth = value
			}
			return accountRoot(text, depth), nil
		}
		return result, nil
	case "parent", "leaf":
		if err := expect("string"); err != nil {
			return fail("%v", err)
		}
		result.kind = "string"
		result.value = func(row Row, rows []Row) (any, error) {
			text, err := stringArg(row, rows, 0)
			if err != nil {
				return nil, err
			}
			if name == "parent" {
				return accountParent(text), nil
			}
			return accountLeaf(text), nil
		}
		return result, nil
	case "account_sortkey":
		if err := expect("string"); err != nil {
			return fail("%v", err)
		}
		result.kind = "string"
		result.value = func(row Row, rows []Row) (any, error) {
			text, err := stringArg(row, rows, 0)
			if err != nil {
				return nil, err
			}
			return accountSortKey(text), nil
		}
		return result, nil
	case "length":
		if len(args) != 1 {
			return fail("length requires one argument")
		}
		result.kind, result.nullable = "integer", args[0].nullable
		result.value = func(row Row, rows []Row) (any, error) {
			value, err := args[0].eval(row, rows)
			if err != nil || value == nil {
				return nil, err
			}
			switch typed := value.(type) {
			case string:
				return ledger.NewDecimal(bigRatInt(len([]rune(typed)))), nil
			case stringSet:
				return ledger.NewDecimal(bigRatInt(len(typed))), nil
			case []string:
				return ledger.NewDecimal(bigRatInt(len(typed))), nil
			case listValue:
				return ledger.NewDecimal(bigRatInt(len(typed))), nil
			default:
				return nil, fmt.Errorf("no typed signature for length(%T)", value)
			}
		}
		if args[0].kind == "string" || args[0].kind == "set<string>" || args[0].kind == "list" {
			return result, nil
		}
		return fail("no typed signature for length(%s)", args[0].kind)
	case "substr":
		if err := expect("string", "integer", "integer"); err != nil {
			return fail("%v", err)
		}
		result.kind = "string"
		result.value = func(row Row, rows []Row) (any, error) {
			text, err := stringArg(row, rows, 0)
			if err != nil || text == "" {
				return nil, err
			}
			start, err := intArg(row, rows, 1)
			if err != nil {
				return nil, err
			}
			end, err := intArg(row, rows, 2)
			if err != nil {
				return nil, err
			}
			return substring(text, start, end), nil
		}
		return result, nil
	case "splitcomp":
		if err := expect("string", "string", "integer"); err != nil {
			return fail("%v", err)
		}
		result.kind = "string"
		result.value = func(row Row, rows []Row) (any, error) {
			text, err := stringArg(row, rows, 0)
			if err != nil || text == "" {
				return nil, err
			}
			delimiter, err := stringArg(row, rows, 1)
			if err != nil {
				return nil, err
			}
			index, err := intArg(row, rows, 2)
			if err != nil {
				return nil, err
			}
			return splitComponent(text, delimiter, index)
		}
		return result, nil
	case "maxwidth":
		if err := expect("string", "integer"); err != nil {
			return fail("%v", err)
		}
		result.kind = "string"
		result.value = func(row Row, rows []Row) (any, error) {
			text, err := stringArg(row, rows, 0)
			if err != nil || text == "" {
				return nil, err
			}
			width, err := intArg(row, rows, 1)
			if err != nil {
				return nil, err
			}
			return shorten(text, width)
		}
		return result, nil
	case "grep", "grepn":
		want := 2
		if name == "grepn" {
			want = 3
		}
		if len(args) != want || args[0].kind != "string" || args[1].kind != "string" {
			return fail("no typed signature for %s", name)
		}
		result.kind = "string"
		result.value = func(row Row, rows []Row) (any, error) {
			pattern, err := stringArg(row, rows, 0)
			if err != nil || pattern == "" {
				return nil, err
			}
			text, err := stringArg(row, rows, 1)
			if err != nil || text == "" {
				return nil, err
			}
			compiled, err := boundedPattern(pattern)
			if err != nil {
				return nil, err
			}
			match := compiled.FindStringSubmatch(text)
			if match == nil {
				return nil, nil
			}
			if name == "grep" {
				return match[0], nil
			}
			group, err := intArg(row, rows, 2)
			if err != nil {
				return nil, err
			}
			if group < 0 || group >= len(match) {
				return nil, fmt.Errorf("no such group")
			}
			return match[group], nil
		}
		return result, nil
	case "subst":
		if err := expect("string", "string", "string"); err != nil {
			return fail("%v", err)
		}
		result.kind = "string"
		result.value = func(row Row, rows []Row) (any, error) {
			pattern, err := stringArg(row, rows, 0)
			if err != nil || pattern == "" {
				return nil, err
			}
			replacement, err := stringArg(row, rows, 1)
			if err != nil {
				return nil, err
			}
			text, err := stringArg(row, rows, 2)
			if err != nil {
				return nil, err
			}
			compiled, err := boundedPattern(pattern)
			if err != nil {
				return nil, err
			}
			// Python's re.sub expands backreferences; Go uses the same $name
			// syntax, so rewriting \1 to $1 keeps the documented behaviour.
			return compiled.ReplaceAllString(text, pythonBackrefs(replacement)), nil
		}
		return result, nil
	case "repr":
		if len(args) != 1 {
			return fail("repr requires one argument")
		}
		result.kind = "string"
		result.value = func(row Row, rows []Row) (any, error) {
			value, err := args[0].eval(row, rows)
			if err != nil || value == nil {
				return nil, err
			}
			if number, ok := value.(ledger.Decimal); ok {
				// repr() of a Decimal keeps the source quantum and sign.
				text := number.String()
				if args[0].text != nil {
					if prompt, promptErr := args[0].text(row, rows); promptErr == nil && prompt != nil {
						text = prompt.(string)
					}
				}
				return "Decimal('" + text + "')", nil
			}
			return fmt.Sprint(value), nil
		}
		return result, nil
	case "joinstr":
		if len(args) != 1 || args[0].kind != "set<string>" {
			return fail("joinstr requires a set column")
		}
		result.kind = "string"
		result.value = func(row Row, rows []Row) (any, error) {
			value, err := args[0].eval(row, rows)
			if err != nil || value == nil {
				return nil, err
			}
			return joinStrings(setValues(value)), nil
		}
		return result, nil
	case "findfirst":
		if len(args) != 2 || args[0].kind != "string" || args[1].kind != "set<string>" {
			return fail("findfirst requires a pattern and a set column")
		}
		result.kind = "string"
		result.value = func(row Row, rows []Row) (any, error) {
			pattern, err := stringArg(row, rows, 0)
			if err != nil || pattern == "" {
				return nil, err
			}
			value, err := args[1].eval(row, rows)
			if err != nil || value == nil {
				return nil, err
			}
			return findFirst(pattern, setValues(value))
		}
		return result, nil
	case "round":
		if len(args) < 1 || len(args) > 2 || !numericKind(args[0].kind) {
			return fail("no typed signature for round")
		}
		result.kind = args[0].kind
		result.value = func(row Row, rows []Row) (any, error) {
			value, err := args[0].eval(row, rows)
			if err != nil || value == nil {
				return nil, err
			}
			number, ok := decimalValue(value)
			if !ok {
				return nil, fmt.Errorf("round requires a number")
			}
			digits := 0
			if len(args) == 2 {
				digits, err = intArg(row, rows, 1)
				if err != nil {
					return nil, err
				}
			}
			return roundDecimal(number, digits), nil
		}
		result.text = func(row Row, rows []Row) (any, error) {
			value, err := result.value(row, rows)
			if err != nil || value == nil {
				return nil, err
			}
			return value.(ledger.Decimal).String(), nil
		}
		return result, nil
	case "safediv":
		if len(args) != 2 || args[0].kind != "decimal" || !numericKind(args[1].kind) {
			return fail("no typed signature for safediv")
		}
		result.kind = "decimal"
		result.value = func(row Row, rows []Row) (any, error) {
			left, err := args[0].eval(row, rows)
			if err != nil || left == nil {
				return nil, err
			}
			right, err := args[1].eval(row, rows)
			if err != nil || right == nil {
				return nil, err
			}
			leftNumber, leftOK := decimalValue(left)
			rightNumber, rightOK := decimalValue(right)
			if !leftOK || !rightOK {
				return nil, fmt.Errorf("safediv requires numbers")
			}
			if rightNumber.IsZero() {
				return ledger.Zero(), nil
			}
			return leftNumber.Quo(rightNumber), nil
		}
		return result, nil
	case "empty":
		if len(args) != 1 || (args[0].kind != "inventory" && args[0].kind != "set<string>" && args[0].kind != "list") {
			return fail("empty requires an inventory or collection")
		}
		result.kind = "bool"
		result.value = func(row Row, rows []Row) (any, error) {
			value, err := args[0].eval(row, rows)
			if err != nil || value == nil {
				return nil, err
			}
			switch typed := value.(type) {
			case inventoryValue:
				return len(typed) == 0, nil
			case stringSet:
				return len(typed) == 0, nil
			case listValue:
				return len(typed) == 0, nil
			}
			return false, nil
		}
		return result, nil
	case "getitem":
		if len(args) < 2 || len(args) > 3 || args[0].kind != "dictionary" || args[1].kind != "string" {
			return fail("no typed signature for getitem")
		}
		result.kind = "object"
		result.value = func(row Row, rows []Row) (any, error) {
			container, err := args[0].eval(row, rows)
			if err != nil || container == nil {
				return nil, err
			}
			key, err := stringArg(row, rows, 1)
			if err != nil {
				return nil, err
			}
			values, ok := container.(dictionaryValue)
			if !ok {
				return nil, fmt.Errorf("getitem requires a dictionary")
			}
			if value, exists := values[key]; exists {
				return value, nil
			}
			if len(args) == 3 {
				return args[2].eval(row, rows)
			}
			return nil, nil
		}
		return result, nil
	case "open_date", "close_date":
		if err := expect("string"); err != nil {
			return fail("%v", err)
		}
		result.kind, result.nullable = "date", true
		result.value = func(row Row, rows []Row) (any, error) {
			account, err := stringArg(row, rows, 0)
			if err != nil || account == "" {
				return nil, err
			}
			state, ok := compiler.context.accountState(account)
			if !ok {
				return nil, nil
			}
			if name == "open_date" {
				if state.Opened.Raw == "" {
					return nil, nil
				}
				return state.Opened.Raw, nil
			}
			if state.Closed == nil {
				return nil, nil
			}
			return state.Closed.Raw, nil
		}
		return result, nil
	case "open_meta":
		if len(args) < 1 || len(args) > 2 || args[0].kind != "string" {
			return fail("no typed signature for open_meta")
		}
		result.kind = "object"
		if len(args) == 1 {
			result.kind = "dictionary"
		}
		result.value = func(row Row, rows []Row) (any, error) {
			account, err := stringArg(row, rows, 0)
			if err != nil || account == "" {
				return nil, err
			}
			entry, ok := compiler.context.openDirective(account)
			if !ok {
				return nil, nil
			}
			scope := directiveMetadata(entry.Directive, entry)
			if len(args) == 1 {
				return metadataDictionary(scope)
			}
			key, err := stringArg(row, rows, 1)
			if err != nil {
				return nil, err
			}
			value, _, err := lookupMetadata(scope, key)
			return value, err
		}
		return result, nil
	case "commodity_meta", "currency_meta":
		if len(args) < 1 || len(args) > 2 || args[0].kind != "string" {
			return fail("no typed signature for %s", name)
		}
		result.kind = "object"
		if len(args) == 1 {
			result.kind = "dictionary"
		}
		result.value = func(row Row, rows []Row) (any, error) {
			currency, err := stringArg(row, rows, 0)
			if err != nil || currency == "" {
				return nil, err
			}
			entry, ok := compiler.context.commodityDirective(currency)
			if !ok {
				return nil, nil
			}
			scope := directiveMetadata(entry.Directive, entry)
			if len(args) == 1 {
				return metadataDictionary(scope)
			}
			key, err := stringArg(row, rows, 1)
			if err != nil {
				return nil, err
			}
			value, _, err := lookupMetadata(scope, key)
			return value, err
		}
		return result, nil
	case "getprice":
		if len(args) < 2 || len(args) > 3 || args[0].kind != "string" || args[1].kind != "string" {
			return fail("no typed signature for getprice")
		}
		result.kind = "decimal"
		result.value = func(row Row, rows []Row) (any, error) {
			base, err := stringArg(row, rows, 0)
			if err != nil || base == "" {
				return nil, err
			}
			quote, err := stringArg(row, rows, 1)
			if err != nil || quote == "" {
				return nil, err
			}
			var date *ledger.Date
			if len(args) == 3 {
				date, err = dateArg(row, rows, 2)
				if err != nil {
					return nil, err
				}
			}
			value, ok := compiler.context.price(strings.ToUpper(base), strings.ToUpper(quote), date)
			if !ok {
				return nil, nil
			}
			return value, nil
		}
		return result, nil
	case "parse_date":
		if len(args) < 1 || len(args) > 2 || args[0].kind != "string" {
			return fail("no typed signature for parse_date")
		}
		result.kind = "date"
		result.value = func(row Row, rows []Row) (any, error) {
			text, err := stringArg(row, rows, 0)
			if err != nil || text == "" {
				return nil, err
			}
			layout := isoDateLayout
			if len(args) == 2 {
				prompt, err := stringArg(row, rows, 1)
				if err != nil || prompt == "" {
					return nil, err
				}
				layout = strftimeLayout(prompt)
			}
			parsed, err := time.Parse(layout, text)
			if err != nil {
				return nil, nil
			}
			return parsed.Format(isoDateLayout), nil
		}
		return result, nil
	case "has_account":
		if len(args) != 1 || args[0].kind != "string" {
			return fail("has_account requires a pattern")
		}
		result.kind, result.nullable = "bool", false
		result.value = func(row Row, rows []Row) (any, error) {
			pattern, err := stringArg(row, rows, 0)
			if err != nil || pattern == "" {
				return nil, err
			}
			compiled, err := boundedPattern(pattern)
			if err != nil {
				return nil, err
			}
			transaction, ok := row["_transaction"].(ledger.Transaction)
			if !ok {
				return false, nil
			}
			for _, posting := range transaction.Postings {
				if compiled.MatchString(posting.Account) {
					return true, nil
				}
			}
			return false, nil
		}
		return result, nil
	case "convert":
		if len(args) < 2 || len(args) > 3 {
			return fail("no typed signature for convert")
		}
		result.kind = "amount"
		if args[0].kind == "inventory" {
			result.kind = "inventory"
		}
		result.value = func(row Row, rows []Row) (any, error) {
			value, err := args[0].eval(row, rows)
			if err != nil || value == nil {
				return nil, err
			}
			target, err := stringArg(row, rows, 1)
			if err != nil || target == "" {
				return nil, err
			}
			var date *ledger.Date
			if len(args) == 3 {
				date, err = dateArg(row, rows, 2)
				if err != nil {
					return nil, err
				}
			}
			convertOne := func(amount amountValue) (amountValue, error) {
				if strings.EqualFold(amount.Currency, target) {
					return amount, nil
				}
				rate, ok := compiler.context.price(amount.Currency, target, date)
				if !ok {
					return amountValue{}, fmt.Errorf("no price for %s in %s", amount.Currency, target)
				}
				return amountValue{Number: amount.Number.Mul(rate), Currency: target, Text: ""}, nil
			}
			switch typed := value.(type) {
			case amountValue:
				return convertOne(typed)
			case positionValue:
				converted, err := convertOne(positionCost(typed))
				return converted, err
			case inventoryValue:
				converted := inventoryAccumulator{}
				for _, position := range typed {
					amount, err := convertOne(positionCost(position))
					if err != nil {
						return nil, err
					}
					converted.add(positionValue{Units: amount})
				}
				return converted.value(), nil
			}
			return nil, fmt.Errorf("convert requires an amount, position, or inventory")
		}
		return result, nil
	case "value":
		if len(args) < 1 || len(args) > 2 {
			return fail("no typed signature for value")
		}
		result.kind = "amount"
		if args[0].kind == "inventory" {
			result.kind = "inventory"
		}
		result.value = func(row Row, rows []Row) (any, error) {
			value, err := args[0].eval(row, rows)
			if err != nil || value == nil {
				return nil, err
			}
			var date *ledger.Date
			if len(args) == 2 {
				date, err = dateArg(row, rows, 1)
				if err != nil {
					return nil, err
				}
			}
			market := func(position positionValue) (amountValue, error) {
				units := position.Units
				if rate, ok := compiler.context.price(units.Currency, "USD", date); ok {
					return amountValue{Number: units.Number.Mul(rate), Currency: "USD", Text: ""}, nil
				}
				return units, nil
			}
			switch typed := value.(type) {
			case positionValue:
				return market(typed)
			case inventoryValue:
				result := inventoryAccumulator{}
				for _, position := range typed {
					amount, err := market(position)
					if err != nil {
						return nil, err
					}
					result.add(positionValue{Units: amount})
				}
				return result.value(), nil
			}
			return nil, fmt.Errorf("value requires a position or inventory")
		}
		return result, nil
	case "only":
		if err := expect("string", "inventory"); err != nil {
			return fail("%v", err)
		}
		result.kind = "amount"
		result.value = func(row Row, rows []Row) (any, error) {
			currency, err := stringArg(row, rows, 0)
			if err != nil || currency == "" {
				return nil, err
			}
			value, err := args[1].eval(row, rows)
			if err != nil || value == nil {
				return nil, err
			}
			for _, position := range value.(inventoryValue) {
				if strings.EqualFold(position.Units.Currency, currency) {
					return position.Units, nil
				}
			}
			return nil, fmt.Errorf("inventory has no position in %s", currency)
		}
		return result, nil
	case "filter_currency":
		if len(args) != 2 || args[1].kind != "string" {
			return fail("no typed signature for filter_currency")
		}
		if args[0].kind != "inventory" && args[0].kind != "position" {
			return fail("no typed signature for filter_currency(%s)", args[0].kind)
		}
		result.kind = args[0].kind
		result.value = func(row Row, rows []Row) (any, error) {
			currency, err := stringArg(row, rows, 1)
			if err != nil || currency == "" {
				return nil, err
			}
			value, err := args[0].eval(row, rows)
			if err != nil || value == nil {
				return nil, err
			}
			if position, ok := value.(positionValue); ok {
				if strings.EqualFold(position.Units.Currency, currency) {
					return position, nil
				}
				return positionValue{}, nil
			}
			filtered := inventoryAccumulator{}
			for _, position := range value.(inventoryValue) {
				if strings.EqualFold(position.Units.Currency, currency) {
					filtered.add(position)
				}
			}
			return filtered.value(), nil
		}
		return result, nil
	case "neg":
		if len(args) != 1 {
			return fail("neg requires one argument")
		}
		switch args[0].kind {
		case "decimal", "integer":
			result.kind = args[0].kind
			result.value = func(row Row, rows []Row) (any, error) {
				value, err := args[0].eval(row, rows)
				if err != nil || value == nil {
					return nil, err
				}
				number := value.(ledger.Decimal)
				number = number.Neg()
				return number, nil
			}
			if args[0].kind == "decimal" {
				result.text = func(row Row, rows []Row) (any, error) {
					value, err := result.value(row, rows)
					if err != nil || value == nil {
						return nil, err
					}
					return value.(ledger.Decimal).String(), nil
				}
			}
		case "amount", "position", "inventory":
			result.kind = args[0].kind
			result.value = func(row Row, rows []Row) (any, error) {
				value, err := args[0].eval(row, rows)
				if err != nil || value == nil {
					return nil, err
				}
				flip := func(amount amountValue) amountValue {
					amount.Number = amount.Number.Neg()
					amount.Text = ""
					return amount
				}
				switch typed := value.(type) {
				case amountValue:
					return flip(typed), nil
				case positionValue:
					typed.Units = flip(typed.Units)
					return typed, nil
				case inventoryValue:
					result := inventoryAccumulator{}
					for _, position := range typed {
						position.Units = flip(position.Units)
						result.add(position)
					}
					return result.value(), nil
				}
				return nil, fmt.Errorf("%s requires a numeric value", name)
			}
		default:
			return fail("no typed signature for %s(%s)", name, args[0].kind)
		}
		return result, nil
	case "possign":
		if len(args) != 2 || args[1].kind != "string" {
			return fail("possign requires a value and an account name")
		}
		sign := func(row Row, rows []Row) (ledger.Decimal, error) {
			account, err := stringArg(row, rows, 1)
			if err != nil || account == "" {
				return ledger.Zero(), err
			}
			return ledger.NewDecimal(bigRatInt(accountSign(account))), nil
		}
		switch args[0].kind {
		case "decimal", "integer":
			result.kind = args[0].kind
			result.value = func(row Row, rows []Row) (any, error) {
				value, err := args[0].eval(row, rows)
				if err != nil || value == nil {
					return nil, err
				}
				factor, err := sign(row, rows)
				if err != nil {
					return nil, err
				}
				return value.(ledger.Decimal).Mul(factor), nil
			}
		case "amount", "position", "inventory":
			result.kind = args[0].kind
			result.value = func(row Row, rows []Row) (any, error) {
				value, err := args[0].eval(row, rows)
				if err != nil || value == nil {
					return nil, err
				}
				factor, err := sign(row, rows)
				if err != nil {
					return nil, err
				}
				scale := func(amount amountValue) amountValue {
					return amountValue{Number: amount.Number.Mul(factor), Currency: amount.Currency, Text: ""}
				}
				switch typed := value.(type) {
				case amountValue:
					return scale(typed), nil
				case positionValue:
					typed.Units = scale(typed.Units)
					return typed, nil
				case inventoryValue:
					result := inventoryAccumulator{}
					for _, position := range typed {
						position.Units = scale(position.Units)
						result.add(position)
					}
					return result.value(), nil
				}
				return nil, fmt.Errorf("possign requires a value")
			}
		default:
			return fail("no typed signature for possign(%s)", args[0].kind)
		}
		return result, nil
	case "commodity":
		if err := expect("amount"); err != nil {
			return fail("%v", err)
		}
		result.kind = "string"
		result.value = func(row Row, rows []Row) (any, error) {
			value, err := args[0].eval(row, rows)
			if err != nil || value == nil {
				return nil, err
			}
			return value.(amountValue).Currency, nil
		}
		return result, nil
	}
	return typedExpr{}, typedFailure("unknown-function", "unsupported typed function %q", name)
}

func setValues(value any) []string {
	switch typed := value.(type) {
	case stringSet:
		return []string(typed)
	case []string:
		return typed
	case listValue:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				result = append(result, text)
			}
		}
		return result
	default:
		return nil
	}
}

func pythonBackrefs(replacement string) string {
	var builder strings.Builder
	for i := 0; i < len(replacement); i++ {
		if replacement[i] == '\\' && i+1 < len(replacement) && replacement[i+1] >= '0' && replacement[i+1] <= '9' {
			builder.WriteByte('$')
			continue
		}
		builder.WriteByte(replacement[i])
	}
	return builder.String()
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// strftimeLayout converts the small set of strftime directives parse_date
// accepts into a Go layout. Unknown directives are passed through literally so
// a malformed style fails instead of silently matching.
func strftimeLayout(format string) string {
	replacer := strings.NewReplacer("%Y", "2006", "%y", "06", "%m", "01", "%d", "02", "%H", "15", "%M", "04", "%S", "05")
	return replacer.Replace(format)
}
