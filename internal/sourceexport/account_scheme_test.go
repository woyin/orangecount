// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package sourceexport

import (
	"strings"
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
)

// The cn account-root vocabulary is a dialect extension: upstream rejects the
// option key and admits only ASCII root names, so exporting such a ledger would
// write a file no v3 tool can load. The gate must refuse instead of publishing.
func TestCNSchemeRejectedBeforeTargetWrite(t *testing.T) {
	text := "option \"account_root_scheme\" \"cn\"\n2000-01-01 open 资产:钱包 CNY\n2000-01-01 open 权益:期初 CNY\n"
	p := entry(t, text)
	_, err := Prepare(p, Options{Build: snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}}})
	if err == nil || !strings.Contains(err.Error(), "account_root_scheme") {
		t.Fatalf("wanted cn-scheme export rejection, got %v", err)
	}
}

// The global value is the upstream default. Its option line is dropped from the
// snapshot because upstream rejects the key, and the surviving file must still
// carry the same accounting.
func TestGlobalSchemeOptionDroppedFromSnapshot(t *testing.T) {
	text := "option \"account_root_scheme\" \"global\"\n" + cash
	p := entry(t, text)
	plan, err := Prepare(p, Options{Build: snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}}})
	if err != nil {
		t.Fatalf("global scheme export failed: %v", err)
	}
	for _, file := range plan.files {
		if strings.Contains(strings.ToLower(string(file.data)), "account_root_scheme") {
			t.Fatalf("snapshot kept the dialect-only option:\n%s", file.data)
		}
	}
}

// Only a top-level option line is dropped: an indented line that merely
// mentions the key must survive, or a ledger could lose data.
func TestDropDialectOnlyOptionsKeepsIndentedText(t *testing.T) {
	input := "option \"account_root_scheme\" \"global\"\n" +
		"2000-01-02 * \"x\"\n" +
		"  option \"account_root_scheme\" \"global\"\n" +
		"  Assets:Cash 1 USD\n"
	got := string(dropDialectOnlyOptions([]byte(input)))
	if strings.HasPrefix(got, "option ") {
		t.Fatalf("top-level option survived:\n%s", got)
	}
	for _, want := range []string{"  option \"account_root_scheme\" \"global\"", "Assets:Cash 1 USD"} {
		if !strings.Contains(got, want) {
			t.Errorf("dropped content %q:\n%s", want, got)
		}
	}
}
