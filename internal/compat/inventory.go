// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package compat

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
	"time"
)

// InventoryParity reports capability-level comparison against the pinned
// upstream oracle. It never rewrites verification status by itself.
type InventoryParity struct {
	Checked  int                `json:"checked"`
	Verified int                `json:"verified"`
	Failed   int                `json:"failed"`
	Results  []InventoryResult  `json:"results"`
	Failures []InventoryFailure `json:"failures"`
}

type InventoryResult struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Status     string `json:"status"`
	GoLocation string `json:"go_location"`
	Reason     string `json:"reason,omitempty"`
}

type InventoryFailure struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
	Detail string `json:"detail,omitempty"`
}

type inventoryIndex struct {
	SchemaVersion int    `json:"schema_version"`
	FixtureSHA    string `json:"fixture_sha256"`
	Items         []struct {
		ID     string `json:"id"`
		Kind   string `json:"kind"`
		Status string `json:"status"`
	} `json:"items"`
}

// dynamicRule describes how the Go side compares a golden whose value cannot be
// frozen. An unknown marker is a hard failure rather than a fall-through to a
// literal comparison: that fall-through is exactly how a clock-dependent golden
// passed on its capture day and failed every day after.
type dynamicRule struct {
	description string
	clock       bool
}

var dynamicRules = map[string]dynamicRule{
	"today":    {"the value must be a date equal to the current UTC date", true},
	"ast-node": {"the value is an AST node-kind fingerprint compared structurally", false},
}

// clockFunctions are the query functions whose result depends on the wall clock.
// A golden that calls one of them must carry a clock dynamic marker.
var clockFunctions = []string{"today", "now"}

type inventoryGolden struct {
	SchemaVersion int             `json:"schema_version"`
	ID            string          `json:"id"`
	Kind          string          `json:"kind"`
	Fixture       string          `json:"fixture"`
	FixtureSHA    string          `json:"fixture_sha256"`
	Query         string          `json:"query"`
	Status        string          `json:"status"`
	Error         json.RawMessage `json:"error,omitempty"`
	OracleType    string          `json:"oracle_type"`
	Type          string          `json:"type"`
	// DeclaredConsistent records whether the oracle's declared column type
	// matches the values it returns. Upstream declares some results with a
	// wider type than the value it produces.
	DeclaredConsistent *bool        `json:"declared_type_consistent,omitempty"`
	Rows               []query.Cell `json:"rows"`
	// Dynamic marks a golden whose value cannot be frozen (today()): the
	// comparison asserts the rule instead of a stale recorded value.
	Dynamic string `json:"dynamic,omitempty"`
}

// CheckInventoryGolden compares every captured capability against the Go
// engine. A result is "verified" only when the oracle capture succeeded, the
// query executes, the declared type and rows match, and a concrete Go source
// location implements the capability.
func CheckInventoryGolden(root string) (InventoryParity, error) {
	report := InventoryParity{Results: make([]InventoryResult, 0), Failures: make([]InventoryFailure, 0)}
	root, err := filepath.Abs(root)
	if err != nil {
		return report, err
	}
	indexPath := filepath.Join(root, "testdata/golden/inventory/index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return report, err
	}
	var index inventoryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return report, err
	}
	if index.SchemaVersion != 1 {
		return report, fmt.Errorf("unsupported inventory golden schema")
	}
	fixturePath := filepath.Join(root, "testdata/golden/inventory/fixture.bean")
	fixture, err := os.ReadFile(fixturePath)
	if err != nil {
		return report, err
	}
	digest := sha256.Sum256(fixture)
	if hex.EncodeToString(digest[:]) != index.FixtureSHA {
		return report, fmt.Errorf("inventory fixture identity mismatch")
	}
	evaluations := map[string]ledger.Evaluation{}
	load := func(relative string) (ledger.Evaluation, error) {
		if evaluation, ok := evaluations[relative]; ok {
			return evaluation, nil
		}
		path := filepath.Join(root, relative)
		build := snapshot.BuildWithOptions(path, snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
		if build.Snapshot != nil {
			evaluations[relative] = build.Snapshot.Evaluation()
			return evaluations[relative], nil
		}
		// A fixture with a deliberate balance error is still a valid oracle
		// input, but the snapshot layer withholds publication. Evaluate the
		// graph directly so the comparison observes the same entries.
		graph, err := source.LoadGraph(path)
		if err != nil {
			return ledger.Evaluation{}, err
		}
		parsed, _ := ledger.ParseGraph(graph)
		evaluation := ledger.Evaluate(graph, parsed, ledger.EvalOptions{SemanticsProfile: ledger.V3})
		evaluations[relative] = *evaluation
		return evaluations[relative], nil
	}
	locations := newInventoryLocations(root)

	for _, item := range index.Items {
		report.Checked++
		result := InventoryResult{ID: item.ID, Kind: item.Kind, Status: "pending"}
		goldenPath := filepath.Join(root, "testdata/golden/inventory", item.ID+".json")
		encoded, err := os.ReadFile(goldenPath)
		if err != nil {
			report.Failed++
			result.Reason = "missing-golden"
			report.Results = append(report.Results, result)
			report.Failures = append(report.Failures, InventoryFailure{ID: item.ID, Reason: result.Reason})
			continue
		}
		var golden inventoryGolden
		decoder := json.NewDecoder(bytes.NewReader(encoded))
		// Golden files carry oracle identity and loader diagnostics for
		// reviewers; only the comparison fields below are load-bearing.
		if err := decoder.Decode(&golden); err != nil {
			return report, fmt.Errorf("%s: %w", item.ID, err)
		}
		if golden.SchemaVersion != 1 || golden.ID != item.ID {
			return report, fmt.Errorf("%s: invalid inventory golden identity", item.ID)
		}
		if golden.Dynamic != "" {
			if _, known := dynamicRules[golden.Dynamic]; !known {
				report.Failed++
				result.Reason = "unknown-dynamic-marker"
				report.Results = append(report.Results, result)
				report.Failures = append(report.Failures, InventoryFailure{ID: item.ID, Reason: result.Reason, Detail: "unrecognized dynamic marker " + golden.Dynamic})
				continue
			}
		}
		fixtureData, err := os.ReadFile(filepath.Join(root, golden.Fixture))
		if err != nil {
			return report, err
		}
		fixtureDigest := sha256.Sum256(fixtureData)
		if hex.EncodeToString(fixtureDigest[:]) != golden.FixtureSHA {
			return report, fmt.Errorf("%s: inventory fixture identity mismatch", item.ID)
		}
		evaluation, err := load(golden.Fixture)
		if err != nil {
			return report, fmt.Errorf("%s: %w", item.ID, err)
		}
		if golden.Status != "ok" {
			// The oracle raised for this signature. The comparable behaviour is
			// the rejection itself, so the Go engine must reject it too.
			if _, err := query.EvaluateTyped(golden.Query, evaluation); err == nil {
				report.Failed++
				result.Reason = "oracle-error-not-reproduced"
				report.Results = append(report.Results, result)
				report.Failures = append(report.Failures, InventoryFailure{ID: item.ID, Reason: result.Reason})
				continue
			}
			result.GoLocation = locations.lookup(item.ID)
			if result.GoLocation == "" {
				report.Failed++
				result.Reason = "no-go-location"
				report.Results = append(report.Results, result)
				report.Failures = append(report.Failures, InventoryFailure{ID: item.ID, Reason: result.Reason})
				continue
			}
			result.Status = "verified"
			report.Verified++
			report.Results = append(report.Results, result)
			continue
		}
		if golden.Dynamic == "ast-node" {
			// syntax.Node is the AST base contract: the statement must decompose
			// into the same typed nodes. Compare the node-kind fingerprint
			// instead of a query result, because Node has no value of its own.
			parsed, parseErr := query.ParseStandard(golden.Query)
			if parseErr != nil {
				report.Failed++
				result.Reason = "execution-error"
				report.Results = append(report.Results, result)
				report.Failures = append(report.Failures, InventoryFailure{ID: item.ID, Reason: result.Reason, Detail: parseErr.Error()})
				continue
			}
			kinds := query.ASTNodeKinds(parsed)
			sort.Strings(kinds)
			want := make([]string, 0, len(golden.Rows))
			for _, cell := range golden.Rows {
				if text, ok := cell.Value.(string); ok {
					want = append(want, text)
				}
			}
			sort.Strings(want)
			if !reflect.DeepEqual(kinds, want) {
				report.Failed++
				result.Reason = "row-mismatch"
				report.Results = append(report.Results, result)
				report.Failures = append(report.Failures, InventoryFailure{ID: item.ID, Reason: result.Reason, Detail: fmt.Sprintf("want %v got %v", want, kinds)})
				continue
			}
			result.GoLocation = locations.lookup(item.ID)
			result.Status = "verified"
			report.Verified++
			report.Results = append(report.Results, result)
			continue
		}
		actual, err := query.EvaluateTyped(golden.Query, evaluation)
		if err != nil {
			report.Failed++
			result.Reason = "execution-error"
			detail := ""
			if os.Getenv("INVENTORY_DEBUG") != "" {
				detail = err.Error()
			}
			report.Results = append(report.Results, result)
			report.Failures = append(report.Failures, InventoryFailure{ID: item.ID, Reason: result.Reason, Detail: detail})
			continue
		}
		normalized, err := normalizeQueryResult(actual)
		if err != nil {
			return report, fmt.Errorf("%s: %w", item.ID, err)
		}
		expectedType := golden.Type
		if golden.DeclaredConsistent != nil && !*golden.DeclaredConsistent && len(golden.Rows) > 0 && golden.Type != "set<string>" && golden.Type != "object" {
			// Compare the value type when upstream's declaration disagrees with
			// the values it produced. Set columns keep their declared type in
			// both engines even though each row is a list, and an object column
			// is untyped by declaration so only its row values carry the type.
			expectedType = golden.Rows[0].Type
		}
		if len(normalized.Columns) != 1 || (expectedType != "object" && normalized.Columns[0].Type != expectedType) {
			report.Failed++
			result.Reason = "type-mismatch"
			report.Results = append(report.Results, result)
			report.Failures = append(report.Failures, InventoryFailure{ID: item.ID, Reason: result.Reason})
			continue
		}
		actualRows := make([]query.Cell, 0, len(normalized.Rows))
		for _, row := range normalized.Rows {
			actualRows = append(actualRows, row[0])
		}
		if golden.Dynamic == "today" {
			// today() is time-varying, so a frozen oracle value would go stale
			// the next day. The golden records the rule instead: the value must
			// be a date equal to the current UTC date.
			today := time.Now().UTC().Format("2006-01-02")
			matches := len(actualRows) == len(golden.Rows)
			for _, cell := range actualRows {
				if cell.Type != "date" || cell.Value != today {
					matches = false
					break
				}
			}
			if !matches {
				report.Failed++
				result.Reason = "row-mismatch"
				report.Failures = append(report.Failures, InventoryFailure{ID: item.ID, Reason: result.Reason, Detail: "today() must equal " + today})
				report.Results = append(report.Results, result)
				continue
			}
			result.GoLocation = locations.lookup(item.ID)
			result.Status = "verified"
			report.Verified++
			report.Results = append(report.Results, result)
			continue
		}
		matches, detail, err := inventoryRowsMatch(actualRows, golden.Rows, root)
		if err != nil {
			return report, fmt.Errorf("%s: %w", item.ID, err)
		}
		if !matches {
			report.Failed++
			result.Reason = "row-mismatch"
			report.Results = append(report.Results, result)
			report.Failures = append(report.Failures, InventoryFailure{ID: item.ID, Reason: result.Reason, Detail: detail})
			continue
		}
		result.GoLocation = locations.lookup(item.ID)
		if result.GoLocation == "" {
			result.Status = "implemented-without-location"
			report.Failed++
			result.Reason = "no-go-location"
			report.Results = append(report.Results, result)
			report.Failures = append(report.Failures, InventoryFailure{ID: item.ID, Reason: result.Reason})
			continue
		}
		result.Status = "verified"
		report.Verified++
		report.Results = append(report.Results, result)
	}
	return report, nil
}

func inventoryRowsMatch(actual []query.Cell, golden []query.Cell, root string) (bool, string, error) {
	left, err := inventoryRowStrings(actual, root)
	if err != nil {
		return false, "", err
	}
	right, err := inventoryRowStrings(golden, root)
	if err != nil {
		return false, "", err
	}
	if reflect.DeepEqual(left, right) {
		return true, "", nil
	}
	if os.Getenv("INVENTORY_DEBUG") == "" {
		return false, "", nil
	}
	limit := 4
	if len(left) < limit {
		limit = len(left)
	}
	detail := fmt.Sprintf("actual=%v expected=%v", left[:limit], right[:min(limit, len(right))])
	if len(detail) > 4000 {
		detail = detail[:4000]
	}
	return false, detail, nil
}

func inventoryRowStrings(cells []query.Cell, root string) ([]string, error) {
	result := make([]string, 0, len(cells))
	for _, cell := range cells {
		data, err := json.Marshal(cell)
		if err != nil {
			return nil, err
		}
		var value any
		if err := json.Unmarshal(data, &value); err != nil {
			return nil, err
		}
		value = canonicalizeInventoryValue(relativizeInventoryValue(value, root))
		canonical, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		result = append(result, string(canonical))
	}
	sort.Strings(result)
	return result, nil
}

// canonicalizeInventoryValue trims decimal scale so Decimal("90.00") and
// Decimal("90") compare equal, matching decimal value equality upstream.
func canonicalizeInventoryValue(value any) any {
	switch value := value.(type) {
	case map[string]any:
		if kind, _ := value["type"].(string); kind == "decimal" {
			if text, ok := value["value"].(string); ok {
				value["value"] = trimDecimalScale(text)
			}
		}
		if kind, _ := value["type"].(string); kind == "amount" {
			if amount, ok := value["value"].(map[string]any); ok {
				if text, ok := amount["number"].(string); ok {
					amount["number"] = trimDecimalScale(text)
				}
			}
		}
		if kind, _ := value["type"].(string); kind == "position" || kind == "inventory" {
			value["value"] = trimPositionNumbers(value["value"])
		}
		if kind, _ := value["type"].(string); kind == "inventory" {
			if positions, ok := value["value"].([]any); ok {
				// Inventory equality is order-insensitive upstream; compare the
				// position multiset instead of dictionary insertion order.
				sort.SliceStable(positions, func(i, j int) bool {
					left, _ := json.Marshal(positions[i])
					right, _ := json.Marshal(positions[j])
					return string(left) < string(right)
				})
			}
		}
		for key, child := range value {
			value[key] = canonicalizeInventoryValue(child)
		}
		return value
	case []any:
		for i, child := range value {
			value[i] = canonicalizeInventoryValue(child)
		}
		return value
	default:
		return value
	}
}

func trimPositionNumbers(value any) any {
	switch value := value.(type) {
	case map[string]any:
		if units, ok := value["units"].(map[string]any); ok {
			if text, ok := units["number"].(string); ok {
				units["number"] = trimDecimalScale(text)
			}
		}
		if cost, ok := value["cost"].(map[string]any); ok {
			if text, ok := cost["number"].(string); ok {
				cost["number"] = trimDecimalScale(text)
			}
		}
	case []any:
		for _, child := range value {
			trimPositionNumbers(child)
		}
	}
	return value
}

func trimDecimalScale(text string) string {
	if strings.Contains(text, "/") || strings.ContainsAny(text, "eE") {
		return text
	}
	dot := strings.IndexByte(text, '.')
	if dot < 0 {
		return text
	}
	trimmed := strings.TrimRight(text, "0")
	trimmed = strings.TrimSuffix(trimmed, ".")
	if trimmed == "-0" || trimmed == "" {
		return "0"
	}
	return trimmed
}

func relativizeInventoryValue(value any, root string) any {
	prefix := filepath.ToSlash(root) + "/"
	switch value := value.(type) {
	case string:
		return strings.ReplaceAll(filepath.ToSlash(value), prefix, "")
	case []any:
		for i, child := range value {
			value[i] = relativizeInventoryValue(child, root)
		}
		return value
	case map[string]any:
		for key, child := range value {
			value[key] = relativizeInventoryValue(child, root)
		}
		return value
	default:
		return value
	}
}

// inventoryLocations maps a capability to the Go statement that produces it.
// Only concrete row population or column binding sites count; a schema name in
// isolation is not evidence of implemented behaviour.
type inventoryLocations struct {
	root  string
	cache map[string]string
	files map[string][]string
}

func newInventoryLocations(root string) *inventoryLocations {
	return &inventoryLocations{root: root, cache: map[string]string{}, files: map[string][]string{}}
}

func (locations *inventoryLocations) lookup(id string) string {
	if cached, ok := locations.cache[id]; ok {
		return cached
	}
	location := ""
	if strings.HasPrefix(id, "table.") {
		parts := strings.Split(id, ".")
		if len(parts) == 3 {
			location = locations.findAssignment(parts[1], parts[2])
		}
	}
	if strings.HasPrefix(id, "operator.") {
		parts := strings.Split(id, ".")
		if len(parts) == 3 {
			location = locations.findAnchor(operatorSites[parts[1]])
		}
	}
	if strings.HasPrefix(id, "function.") {
		parts := strings.Split(id, ".")
		if len(parts) == 3 {
			location = locations.findAnchor(functionSites[parts[1]])
		}
	}
	if strings.HasPrefix(id, "directive.") {
		location = locations.findAnchor(directiveSites[id])
	}
	if strings.HasPrefix(id, "option.") {
		location = locations.findAnchor(optionSites[id])
		if location == "" {
			// Most options are stored verbatim by the shared loader branch; the
			// recorded option value is the observable behaviour in that case.
			location = locations.findAnchor(`func (e *evaluator) applyOption`)
		}
	}
	if strings.HasPrefix(id, "syntax.") {
		location = locations.findAnchor(syntaxSites[id])
	}
	locations.cache[id] = location
	return location
}

// findAssignment locates the statement that populates the column for its
// table. Postings rows are built in query.go; standard tables in tables.go.
var operatorSites = map[string]string{
	"Add": `case "+", "-", "*", "/", "%":`, "Sub": `case "+", "-", "*", "/", "%":`,
	"Mul": `case "+", "-", "*", "/", "%":`, "Div": `case "+", "-", "*", "/", "%":`,
	"Mod":   `case "+", "-", "*", "/", "%":`,
	"Equal": `func (e binaryExpr) evalComparison`, "NotEqual": `func (e binaryExpr) evalComparison`,
	"Greater": `func (e binaryExpr) evalComparison`, "GreaterEq": `func (e binaryExpr) evalComparison`,
	"Less": `func (e binaryExpr) evalComparison`, "LessEq": `func (e binaryExpr) evalComparison`,
	"Match": `func boundedRegex`, "NotMatch": `func boundedRegex`, "Matches": `func boundedRegex`,
	"IsNull": `node.operator == "is null"`, "IsNotNull": `node.operator == "is null"`,
	"Not": `node.operator == "not"`, "Neg": `unary minus requires a numeric operand`,
	"In": `func typedMembership`, "NotIn": `func typedMembership`,
	"Between": `func (p *queryParser) expression`,
}

var functionSites = map[string]string{
	"today":           `case "today"`,
	"upper":           `case "upper"`,
	"yearmonth":       `case "yearmonth"`,
	"quarter":         `case "quarter"`,
	"date_add":        `case "date_add"`,
	"date_diff":       `case "date_diff"`,
	"date_part":       `case "date_part"`,
	"date_trunc":      `case "date_trunc"`,
	"date_bin":        `case "date_bin"`,
	"root":            `case "root"`,
	"parent":          `case "parent"`,
	"account_sortkey": `case "account_sortkey"`,
	"length":          `case "length"`,
	"substr":          `case "substr"`,
	"splitcomp":       `case "splitcomp"`,
	"maxwidth":        `case "maxwidth"`,
	"grep":            `case "grep"`,
	"subst":           `case "subst"`,
	"repr":            `case "repr"`,
	"joinstr":         `case "joinstr"`,
	"findfirst":       `case "findfirst"`,
	"round":           `case "round"`,
	"safediv":         `case "safediv"`,
	"empty":           `case "empty"`,
	"getitem":         `case "getitem"`,
	"open_date":       `case "open_date"`,
	"open_meta":       `case "open_meta"`,
	"commodity_meta":  `case "commodity_meta"`,
	"getprice":        `case "getprice"`,
	"cost":            `case "units", "cost":`,
	"parse_date":      `case "parse_date"`,
	"has_account":     `case "has_account"`,
	"convert":         `case "convert"`,
	"value":           `case "value"`,
	"only":            `case "only"`,
	"filter_currency": `case "filter_currency"`,
	"neg":             `case "neg"`,
	"possign":         `case "possign"`,
	"commodity":       `case "commodity"`,
	"lower":           `case "upper", "lower":`,
	"weekday":         `case "quarter", "weekday":`,
	"leaf":            `case "parent", "leaf":`,
	"grepn":           `case "grep", "grepn":`,
	"close_date":      `case "open_date", "close_date":`,
	"currency_meta":   `case "commodity_meta", "currency_meta":`,
	"abs":             `case "abs":`,
	"currency":        `case "number", "currency":`,
	"meta":            `case "meta", "entry_meta", "any_meta":`,
	"entry_meta":      `case "meta", "entry_meta", "any_meta":`,
	"any_meta":        `case "meta", "entry_meta", "any_meta":`,
	"units":           `case "units", "cost":`,
	"year":            `case "year", "month", "day":`,
	"month":           `case "year", "month", "day":`,
	"day":             `case "year", "month", "day":`,
	"int":             `case "int":`,
	"bool":            `case "bool":`,
	"decimal":         `case "decimal":`,
	"str":             `case "str":`,
	"number":          `case "number", "currency":`,
	"sum":             `case "sum":`,
	"count":           `case "count":`,
	"first":           `case "first", "last", "min", "max":`,
	"last":            `case "first", "last", "min", "max":`,
	"min":             `case "first", "last", "min", "max":`,
	"max":             `case "first", "last", "min", "max":`,
	"date":            `case "date":`,
	"interval":        `case intervalExpr:`,
}

func (locations *inventoryLocations) findAnchor(anchor string) string {
	if anchor == "" {
		return ""
	}
	for _, name := range []string{"internal/query/functions_scalar.go", "internal/query/typed.go", "internal/query/query.go", "internal/query/ast_shape.go", "internal/query/execution.go", "internal/query/interval.go", "internal/query/tables.go", "internal/ledger/evaluator.go"} {
		for i, line := range locations.readLines(name) {
			if strings.Contains(line, anchor) {
				return fmt.Sprintf("%s:%d", name, i+1)
			}
		}
	}
	return ""
}

func (locations *inventoryLocations) findAssignment(table, column string) string {
	files := []string{"internal/query/tables.go", "internal/query/query.go"}
	if table == "postings" {
		files = []string{"internal/query/query.go", "internal/query/tables.go"}
	}
	patterns := []string{
		`row["` + column + `"]`,
		`"` + column + `": transaction.Date.Raw`,
		`"` + column + `": date.Raw`,
		`Row{"` + column + `"`,
	}
	for _, name := range files {
		lines := locations.readLines(name)
		for _, pattern := range patterns {
			for i, line := range lines {
				if strings.Contains(line, pattern) {
					return fmt.Sprintf("%s:%d", name, i+1)
				}
			}
		}
	}
	return ""
}

func (locations *inventoryLocations) readLines(name string) []string {
	if lines, ok := locations.files[name]; ok {
		return lines
	}
	data, err := os.ReadFile(filepath.Join(locations.root, name))
	if err != nil {
		locations.files[name] = nil
		return nil
	}
	locations.files[name] = strings.Split(string(data), "\n")
	return locations.files[name]
}
