// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package i18n

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

func TestTextUsesChineseDefaultAndEnglishMessages(t *testing.T) {
	if got := Text(SimplifiedChinese, "打开账本"); got != "打开账本" {
		t.Fatalf("Chinese text = %q", got)
	}
	if got := Text(English, "打开账本"); got != "Open Ledger" {
		t.Fatalf("English text = %q", got)
	}
	if got := Text("unknown", "打开账本"); got != "打开账本" {
		t.Fatalf("unsupported locale did not preserve Chinese default: %q", got)
	}
}

func TestTextTranslatesApplicationMessagePrefixesOnly(t *testing.T) {
	got := Text(English, "无法读取账本：permission denied")
	if got != "Could not read ledger: permission denied" {
		t.Fatalf("localized error = %q", got)
	}
	for _, tc := range []struct{ source, want string }{
		{"持仓表格不可用：duplicate column", "Holdings table unavailable: duplicate column"},
		{"商品表格不可用：duplicate column", "Commodity table unavailable: duplicate column"},
	} {
		if got := Text(English, tc.source); got != tc.want {
			t.Errorf("localized table error = %q, want %q", got, tc.want)
		}
	}
	ledgerValue := "月度收支"
	if got := Text(English, ledgerValue+".bean"); got != ledgerValue+".bean" {
		t.Fatalf("unmatched ledger text was changed: %q", got)
	}
}

func TestEnglishCatalogHasNoUntranslatedHan(t *testing.T) {
	for source, translated := range english {
		for _, r := range translated {
			if unicode.Is(unicode.Han, r) {
				t.Errorf("English translation for %q contains Han: %q", source, translated)
				break
			}
		}
		if strings.TrimSpace(source) == "" {
			t.Errorf("empty source key maps to %q", translated)
		}
	}
}

func TestEnglishCatalogHasUniqueSourceKeys(t *testing.T) {
	seen := make(map[string]bool, len(englishPairs))
	for _, pair := range englishPairs {
		if seen[pair[0]] {
			t.Errorf("duplicate English catalog key %q", pair[0])
		}
		seen[pair[0]] = true
	}
}

func TestLiteralApplicationTranslationsHaveEnglishCatalogEntries(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "app", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("could not find application source files for localization audit")
	}
	fset := token.NewFileSet()
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Errorf("parse %s: %v", path, err)
			continue
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			for _, expression := range visibleUILabelExpressions(call) {
				if source, ok := untranslatedHanLiteral(expression); ok {
					t.Errorf("%s: visible UI label %q is not passed through the translation catalog", fset.Position(expression.Pos()), source)
				}
			}
			if len(call.Args) != 1 {
				return true
			}
			usesTranslation := false
			switch function := call.Fun.(type) {
			case *ast.SelectorExpr:
				usesTranslation = function.Sel.Name == "tr"
			case *ast.Ident:
				usesTranslation = function.Name == "tr"
			}
			if !usesTranslation {
				return true
			}
			literal, ok := call.Args[0].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			source, err := strconv.Unquote(literal.Value)
			if err != nil || !hasHan(source) {
				return true
			}
			if got := Text(English, source); got == source {
				t.Errorf("%s: no English catalog entry for application text %q (translated as %q)", fset.Position(literal.Pos()), source, got)
			}
			return true
		})
	}
}

func TestChartTranslationCallSitesHaveEnglishCatalogEntries(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "charts", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("could not find chart source files for localization audit")
	}
	fset := token.NewFileSet()
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Errorf("parse %s: %v", path, err)
			continue
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) != 2 {
				return true
			}
			function, ok := call.Fun.(*ast.Ident)
			if !ok || function.Name != "chartText" {
				return true
			}
			literal, ok := call.Args[1].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			source, err := strconv.Unquote(literal.Value)
			if err != nil || !hasHan(source) {
				return true
			}
			if got := Text(English, source); got == source {
				t.Errorf("%s: chart UI text %q has no English catalog entry", fset.Position(literal.Pos()), source)
			}
			return true
		})
	}
}

func visibleUILabelExpressions(call *ast.CallExpr) []ast.Expr {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	if receiver, ok := selector.X.(*ast.Ident); ok && receiver.Name == "ui" {
		switch selector.Sel.Name {
		case "Text", "Heading", "Button", "PrimaryButton":
			if len(call.Args) > 1 {
				return []ast.Expr{call.Args[1]}
			}
		case "Segmented":
			if len(call.Args) > 2 {
				return call.Args[2:]
			}
		}
	}
	switch selector.Sel.Name {
	case "Label", "Placeholder":
		if len(call.Args) > 0 {
			return []ast.Expr{call.Args[0]}
		}
	}
	return nil
}

func untranslatedHanLiteral(expression ast.Expr) (string, bool) {
	var source string
	ast.Inspect(expression, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
				if selector.Sel.Name == "tr" || selector.Sel.Name == "pageDescription" {
					return false
				}
			}
		}
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(literal.Value)
		if err == nil && hasHan(value) {
			source = value
			return false
		}
		return source == ""
	})
	return source, source != ""
}

func hasHan(value string) bool {
	for _, r := range value {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}
