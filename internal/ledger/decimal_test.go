// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ledger

import "testing"

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
