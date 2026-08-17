// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package dialect_test

import (
	"strings"
	"testing"
)

// TestBlockMetadataRoundTrip locks transaction metadata pass-through: a
// dialect block carries meta between the header and the legs, and the
// export restores it as standard v3 metadata.
func TestBlockMetadataRoundTrip(t *testing.T) {
	v3 := `2000-01-01 open Assets:Cash CNY
2000-01-01 open Assets:Fund "FIFO"
option "operating_currency" "CNY"

2026-07-31 * "我" "购买华宝纳斯达克" #fund-investment ^nav-placeholder
  todo: "净值为 7.30 占位，待更新为实际净值"
  Assets:Cash  -50.00 CNY
  Assets:Fund  FUND_017436 {2.1569 CNY}
`
	original := writeFile(t, "meta.bean", v3)
	dialectVersion := dialectizeFile(t, original)
	text := readFile(t, dialectVersion)
	t.Logf("dialectized:\n%s", text)
	if !strings.Contains(text, `todo: "净值为 7.30 占位，待更新为实际净值"`) {
		t.Fatalf("metadata dropped from dialect block:\n%s", text)
	}
	exported := exportFile(t, dialectVersion)
	export := readFile(t, exported)
	if !strings.Contains(export, `todo: "净值为 7.30 占位，待更新为实际净值"`) {
		t.Fatalf("metadata dropped from export:\n%s", export)
	}
	assertBalancesEqual(t, original, exported)
}

// TestTodoCommentPromotion locks the TODO comment rule: `; TODO:` and
// `; FIXME:` comments convert to todo metadata (information is promoted to
// data instead of being deleted), free-text comments keep the block
// standard, and an existing todo key keeps the block standard.
func TestTodoCommentPromotion(t *testing.T) {
	base := `2000-01-01 open Assets:Cash CNY
2000-01-01 open Assets:Fund "FIFO"
option "operating_currency" "CNY"

`
	cases := []struct {
		name     string
		body     string
		converts bool
		contains string
	}{
		{"todo comment promotes", `
2026-01-01 * "我" "购买基金"
  ; TODO: 净值占位
  Assets:Cash  -50.00 CNY
  Assets:Fund  FUND_017436 {2.1569 CNY}
`, true, `todo: "净值占位"`},
		{"fixme comment promotes", `
2026-01-01 * "我" "购买基金"
  ; FIXME: 确认费率
  Assets:Cash  -50.00 CNY
  Assets:Fund  FUND_017436 {2.1569 CNY}
`, true, `todo: "确认费率"`},
		{"free comment stays standard", `
2026-01-01 * "我" "购买基金"
  ; 在京东金融APP下单
  Assets:Cash  -50.00 CNY
  Assets:Fund  FUND_017436 {2.1569 CNY}
`, false, "; 在京东金融APP下单"},
		{"existing todo key stays standard", `
2026-01-01 * "我" "购买基金"
  todo: "already tracked"
  ; TODO: 净值占位
  Assets:Cash  -50.00 CNY
  Assets:Fund  FUND_017436 {2.1569 CNY}
`, false, "; TODO: 净值占位"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original := writeFile(t, "todo.bean", base+tc.body)
			dialectVersion := dialectizeFile(t, original)
			text := readFile(t, dialectVersion)
			hasLeg := strings.Contains(text, "-> @Assets:Fund")
			if hasLeg != tc.converts {
				t.Fatalf("converts=%v want %v:\n%s", hasLeg, tc.converts, text)
			}
			if tc.contains != "" && !strings.Contains(text, tc.contains) {
				t.Fatalf("missing %q in:\n%s", tc.contains, text)
			}
		})
	}
}

// TestPostingMetadataStillRejects locks the boundary: metadata on a posting
// (not the transaction) keeps the block standard, since a dialect leg maps
// to two postings and cannot carry per-posting data.
func TestPostingMetadataStillRejects(t *testing.T) {
	v3 := `2000-01-01 open Assets:Cash CNY
2000-01-01 open Assets:Fund "FIFO"
option "operating_currency" "CNY"

2026-01-01 * "我" "购买基金"
  Assets:Cash  -50.00 CNY
  Assets:Fund  FUND_017436 {2.1569 CNY}
    channel: "app"
`
	original := writeFile(t, "pmeta.bean", v3)
	dialectVersion := dialectizeFile(t, original)
	text := readFile(t, dialectVersion)
	if !strings.Contains(text, "Assets:Fund ") {
		t.Fatalf("posting metadata should keep the block standard:\n%s", text)
	}
}
