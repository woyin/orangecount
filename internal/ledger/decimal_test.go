// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ledger

import (
	"encoding/json"
	"math/big"
	"testing"
)

func TestDecimalExactArithmetic(t *testing.T) {
	a, err := ParseDecimal("0.10")
	if err != nil {
		t.Fatal(err)
	}
	b, err := ParseDecimal("0.20")
	if err != nil {
		t.Fatal(err)
	}
	if got := a.Add(b).String(); got != "0.3" {
		t.Fatalf("sum=%s", got)
	}
	if got := a.Sub(b).String(); got != "-0.1" {
		t.Fatalf("difference=%s", got)
	}
}

func TestDecimalParsingFormattingAndJSONRemainExact(t *testing.T) {
	for _, raw := range []string{"", "not-a-number"} {
		if _, err := ParseDecimal(raw); err == nil {
			t.Errorf("ParseDecimal(%q) accepted invalid input", raw)
		}
	}
	value, err := ParseDecimal(" 1,234.50 ")
	if err != nil || value.String() != "1234.5" || value.Sign() != 1 || value.IsZero() {
		t.Fatalf("parsed=%s sign=%d zero=%v err=%v", value.String(), value.Sign(), value.IsZero(), err)
	}
	third := NewDecimal(big.NewRat(1, 3))
	if third.String() != "1/3" {
		t.Fatalf("third=%s", third.String())
	}
	if got := value.Quo(ParseMust(t, "2")).String(); got != "617.25" {
		t.Fatalf("quotient=%s", got)
	}
	if got := value.Neg().String(); got != "-1234.5" {
		t.Fatalf("negative=%s", got)
	}
	if !value.Equal(NewDecimal(value.Rat())) || value.Cmp(Zero()) <= 0 {
		t.Fatal("comparison or defensive rat copy failed")
	}
	rat := value.Rat()
	rat.SetInt64(0)
	if value.IsZero() {
		t.Fatal("Rat leaked decimal state")
	}
	encoded, err := json.Marshal(third)
	if err != nil || string(encoded) != `"1/3"` {
		t.Fatalf("json=%s err=%v", encoded, err)
	}
	if !Zero().IsZero() || Zero().Sign() != 0 || NewDecimal(nil).String() != "0" {
		t.Fatal("zero behavior is inconsistent")
	}
}

func ParseMust(t *testing.T, raw string) Decimal {
	t.Helper()
	value, err := ParseDecimal(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
