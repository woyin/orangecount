package dialect_test

import (
	"strings"
	"testing"
)

// TestInvestmentBuyRoundTrip locks the buy dialect end to end: a v3 buy
// (cash leg + securities lot with cost batch) dialectizes to one investment
// leg with the cash side derived from quantity × unit cost, and exporting
// restores the same postings with equal balances.
func TestInvestmentBuyRoundTrip(t *testing.T) {
	v3 := `2000-01-01 open Assets:小金库 CNY
2000-01-01 open Assets:基金 "FIFO"
option "operating_currency" "CNY"

2025-09-07 * "我" "定投标普500"
  Assets:小金库 -1501.00 CNY
  Assets:基金 1000 FUND_019305 {1.5010 CNY}
`
	original := writeFile(t, "buy.bean", v3)
	block := dialectizeFile(t, original)
	text := readFile(t, block)
	t.Logf("dialectized:\n%s", text)
	if !strings.Contains(text, "1000 FUND_019305 {1.5010 CNY} @Assets:小金库 -> @Assets:基金") {
		t.Fatalf("buy leg missing:\n%s", text)
	}
	exported := exportFile(t, block)
	exportedText := readFile(t, exported)
	t.Logf("exported:\n%s", exportedText)
	if !strings.Contains(exportedText, "Assets:基金 1000 FUND_019305 {1.5010 CNY}") {
		t.Fatalf("securities posting lost:\n%s", exportedText)
	}
	if !strings.Contains(exportedText, "Assets:小金库 -1501") {
		t.Fatalf("cash posting wrong:\n%s", exportedText)
	}
	assertBalancesEqual(t, original, exported)
}

// TestInvestmentBuyAutoQuantityRoundTrip locks the fund-buy shape: the
// securities leg omits the quantity, the cash leg drives the share count,
// and export restores both postings with equal balances.
func TestInvestmentBuyAutoQuantityRoundTrip(t *testing.T) {
	v3 := `2000-01-01 open Assets:小金库 CNY
2000-01-01 open Assets:基金 "FIFO"
option "operating_currency" "CNY"

2025-09-07 * "我" "定投标普500"
  Assets:小金库 -1000.00 CNY
  Assets:基金 FUND_019305 {1.5010 CNY}
`
	original := writeFile(t, "buy.bean", v3)
	block := dialectizeFile(t, original)
	text := readFile(t, block)
	t.Logf("dialectized:\n%s", text)
	if !strings.Contains(text, "FUND_019305 {1.5010 CNY} @Assets:小金库 -> @Assets:基金") {
		t.Fatalf("auto-quantity buy leg missing:\n%s", text)
	}
	exported := exportFile(t, block)
	assertBalancesEqual(t, original, exported)
}
