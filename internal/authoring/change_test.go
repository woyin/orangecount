// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package authoring

import (
	"bytes"
	"errors"
	"testing"
)

func TestAppendNormalizesOneBlankSeparator(t *testing.T) {
	change, err := Append("main.bean", "2000-01-02 note Assets:Cash \"x\"")
	if err != nil {
		t.Fatal(err)
	}
	got := change.apply([]byte("2000-01-01 open Assets:Cash USD\n"))
	want := []byte("2000-01-01 open Assets:Cash USD\n\n2000-01-02 note Assets:Cash \"x\"\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("got=%q want=%q", got, want)
	}
}

func TestReplacePreservesExactSubmittedBytes(t *testing.T) {
	input := []byte("option \"title\" \"new\"\n")
	change, err := Replace("main.bean", input)
	if err != nil {
		t.Fatal(err)
	}
	input[0] = 'X'
	if got := change.apply([]byte("old")); !bytes.Equal(got, []byte("option \"title\" \"new\"\n")) {
		t.Fatalf("got=%q", got)
	}
}

func TestChangeRejectsBlankInputs(t *testing.T) {
	if _, err := Append("", "block"); !errors.Is(err, ErrInvalidProposal) {
		t.Fatal("blank target accepted")
	}
	if _, err := Append("main.bean", " \n"); !errors.Is(err, ErrInvalidProposal) {
		t.Fatal("blank block accepted")
	}
	if _, err := Replace(" ", nil); !errors.Is(err, ErrInvalidProposal) {
		t.Fatal("blank target accepted")
	}
}
