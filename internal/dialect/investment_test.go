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

// TestInvestmentSellRoundTrip locks both sell shapes: explicit cash with an
// elided gain (shape A) and elided cash with a fee (shape B).
func TestInvestmentSellRoundTrip(t *testing.T) {
	v3 := `2000-01-01 open Assets:华泰现金 CNY
2000-01-01 open Assets:华泰持股 "FIFO"
2000-01-01 open Expenses:Fees:交易费用 CNY
2000-01-01 open Income:Passive:投资收益
option "operating_currency" "CNY"

2022-08-03 * "我" "卖出东方财富2200股" #stock-investment
  Assets:华泰现金  47212.00 CNY
  Assets:华泰持股      -2200 STOCKA_300059 {} @ 21.46 CNY
  Expenses:Fees:交易费用               55.23 CNY
  Income:Passive:投资收益

2025-11-13 * "我" "卖出豫光金铅2000股" #stock-investment
  Assets:华泰现金  
  Assets:华泰持股      -2000 STOCKA_600531 {} @12.60 CNY
  Expenses:Fees:交易费用                6.39 CNY
`
	original := writeFile(t, "sell.bean", v3)
	dialectVersion := dialectizeFile(t, original)
	text := readFile(t, dialectVersion)
	t.Logf("dialectized:\n%s", text)
	if !strings.Contains(text, "-> @Income:Passive:投资收益 手续费 55.23 CNY @Expenses:Fees:交易费用") {
		t.Fatalf("sell leg with gain and fee missing:\n%s", text)
	}
	if !strings.Contains(text, "-2000 STOCKA_600531 {} @ 12.60 CNY") && !strings.Contains(text, "2000 STOCKA_600531 {} @ 12.60 CNY") {
		t.Fatalf("elided-cash sell leg missing:\n%s", text)
	}
	exported := exportFile(t, dialectVersion)
	assertBalancesEqual(t, original, exported)
}

// TestInvestmentFeeBuyRoundTrip locks the buy-with-fee shape: elided cash
// absorbs quantity × unit cost plus the fee.
func TestInvestmentFeeBuyRoundTrip(t *testing.T) {
	v3 := `2000-01-01 open Assets:华泰现金 CNY
2000-01-01 open Assets:华泰持股 "FIFO"
2000-01-01 open Expenses:Fees:交易费用 CNY
option "operating_currency" "CNY"

2021-03-01 * "我" "购买东方财富500股" #stock-investment
  Assets:华泰现金  
  Assets:华泰持股        500 STOCKA_300059 {31.12 CNY}
  Expenses:Fees:交易费用                5.31 CNY
`
	original := writeFile(t, "feebuy.bean", v3)
	dialectVersion := dialectizeFile(t, original)
	text := readFile(t, dialectVersion)
	t.Logf("dialectized:\n%s", text)
	if !strings.Contains(text, "500 STOCKA_300059 {31.12 CNY} @Assets:华泰现金 -> @Assets:华泰持股 手续费 5.31 CNY @Expenses:Fees:交易费用") {
		t.Fatalf("fee buy leg missing:\n%s", text)
	}
	exported := exportFile(t, dialectVersion)
	assertBalancesEqual(t, original, exported)
}

// TestInvestmentParenNarrationRoundTrip locks the relaxed narration gate:
// investment headers quote the narration, so lexer-significant characters
// like parentheses convert.
func TestInvestmentParenNarrationRoundTrip(t *testing.T) {
	v3 := `2000-01-01 open Assets:小金库 CNY
2000-01-01 open Assets:基金 "FIFO"
option "operating_currency" "CNY"

2025-09-15 * "我" "购买 华宝纳斯达克精选股票发起式(QDII)A" #fund-investment
  Assets:小金库 -1000.00 CNY
  Assets:基金 FUND_017436 {2.2364 CNY}
`
	original := writeFile(t, "paren.bean", v3)
	dialectVersion := dialectizeFile(t, original)
	text := readFile(t, dialectVersion)
	t.Logf("dialectized:\n%s", text)
	if !strings.Contains(text, `"购买 华宝纳斯达克精选股票发起式(QDII)A"`) {
		t.Fatalf("paren-narration buy not converted:\n%s", text)
	}
	exported := exportFile(t, dialectVersion)
	assertBalancesEqual(t, original, exported)
}

// TestInvestmentBonusShareRoundTrip locks the bonus-share shape: a
// securities lot sourced from an elided income leg, no cash.
func TestInvestmentBonusShareRoundTrip(t *testing.T) {
	v3 := `2000-01-01 open Assets:华泰持股 "FIFO"
2000-01-01 open Income:Passive:投资收益
option "operating_currency" "CNY"

2021-05-26 * "我" "东方财富红股入账240股" #stock-investment
  Assets:华泰持股        240 STOCKA_300059 {37.62 CNY}
  Income:Passive:投资收益
`
	original := writeFile(t, "bonus.bean", v3)
	dialectVersion := dialectizeFile(t, original)
	text := readFile(t, dialectVersion)
	t.Logf("dialectized:\n%s", text)
	if !strings.Contains(text, "240 STOCKA_300059 {37.62 CNY} @Income:Passive:投资收益 -> @Assets:华泰持股") {
		t.Fatalf("bonus-share leg missing:\n%s", text)
	}
	exported := exportFile(t, dialectVersion)
	assertBalancesEqual(t, original, exported)
}

// TestInvestmentElidedCashBuyRoundTrip locks the elided-cash clean buy: the
// leg derives cash as quantity × unit cost.
func TestInvestmentElidedCashBuyRoundTrip(t *testing.T) {
	v3 := `2000-01-01 open Assets:华泰现金 CNY
2000-01-01 open Assets:华泰持股 "FIFO"
option "operating_currency" "CNY"

2022-12-01 * "我" "购买杉杉股份股票2500股" #stock-investment
  Assets:华泰现金  
  Assets:华泰持股       2500 STOCKA_600884 {15.17 CNY}
`
	original := writeFile(t, "elided.bean", v3)
	dialectVersion := dialectizeFile(t, original)
	text := readFile(t, dialectVersion)
	t.Logf("dialectized:\n%s", text)
	if !strings.Contains(text, "2500 STOCKA_600884 {15.17 CNY} @Assets:华泰现金 -> @Assets:华泰持股") {
		t.Fatalf("elided-cash buy leg missing:\n%s", text)
	}
	exported := exportFile(t, dialectVersion)
	assertBalancesEqual(t, original, exported)
}

// TestInvestmentExplicitCashFeeBuyRoundTrip locks the explicit-cash
// fee-buy: the amount form carries both cash and quantity exactly.
func TestInvestmentExplicitCashFeeBuyRoundTrip(t *testing.T) {
	v3 := `2000-01-01 open Assets:华泰现金 CNY
2000-01-01 open Assets:华泰持股 "FIFO"
2000-01-01 open Expenses:Fees:交易费用 CNY
option "operating_currency" "CNY"

2025-09-18 * "我" "购买兴业银锡股票1100股" #stock-investment
  Assets:华泰现金  -26295.26 CNY
  Assets:华泰持股       1100 STOCKA_000426 {23.90 CNY}
  Expenses:Fees:交易费用                5.26 CNY
`
	original := writeFile(t, "explicit.bean", v3)
	dialectVersion := dialectizeFile(t, original)
	text := readFile(t, dialectVersion)
	t.Logf("dialectized:\n%s", text)
	if !strings.Contains(text, "26295.26 CNY 1100 STOCKA_000426 {23.90 CNY}") {
		t.Fatalf("explicit-cash fee-buy leg missing:\n%s", text)
	}
	exported := exportFile(t, dialectVersion)
	assertBalancesEqual(t, original, exported)
}

// TestMultiAssetBuyRoundTrip locks the multi-asset buy: one cash leg plus
// several same-account cost lots converts to parallel derived legs, and the
// export merges the derived cash sides back into the original posting.
func TestMultiAssetBuyRoundTrip(t *testing.T) {
	v3 := `2000-01-01 open Assets:Wallet:微信 CNY
2000-01-01 open Assets:收藏品 "FIFO"
option "operating_currency" "CNY"

2026-01-16 * "我" "购买2026马年纪念币和纪念钞" #collection-investment
  Assets:Wallet:微信              -1400.00 CNY
  Assets:收藏品              20 COIN_2026_HORSE {10.00 CNY}
  Assets:收藏品              60 NOTE_2026_HORSE {20.00 CNY}
`
	original := writeFile(t, "multi.bean", v3)
	dialectVersion := dialectizeFile(t, original)
	text := readFile(t, dialectVersion)
	t.Logf("dialectized:\n%s", text)
	if !strings.Contains(text, "20 COIN_2026_HORSE {10.00 CNY} @Assets:Wallet:微信 -> @Assets:收藏品") ||
		!strings.Contains(text, "60 NOTE_2026_HORSE {20.00 CNY} @Assets:Wallet:微信 -> @Assets:收藏品") {
		t.Fatalf("multi-asset legs missing:\n%s", text)
	}
	exported := exportFile(t, dialectVersion)
	export := readFile(t, exported)
	if !strings.Contains(export, "Assets:Wallet:微信 -200 CNY") ||
		!strings.Contains(export, "Assets:Wallet:微信 -1200 CNY") {
		t.Fatalf("per-asset cash records not kept in export:\n%s", export)
	}
	assertBalancesEqual(t, original, exported)

	// Second hop: the per-record export (split cash lines) must
	// re-convert to the identical block — the fixed point.
	second := dialectizeFile(t, writeFile(t, "multi2.bean", export))
	if block2 := readFile(t, second); block2 != text {
		t.Fatalf("multi-asset fixed point broken:\nfirst:\n%s\nsecond:\n%s", text, block2)
	}
}

// TestMultiAssetBuyElidedCash locks the elided-cash variant: the residual
// equals the sum of the lots.
func TestMultiAssetBuyElidedCash(t *testing.T) {
	v3 := `2000-01-01 open Assets:Wallet:微信 CNY
2000-01-01 open Assets:收藏品 "FIFO"
option "operating_currency" "CNY"

2026-01-16 * "我" "购买套装"
  Assets:Wallet:微信
  Assets:收藏品              20 COIN_2026_HORSE {10.00 CNY}
  Assets:收藏品              60 NOTE_2026_HORSE {20.00 CNY}
`
	original := writeFile(t, "multie.bean", v3)
	dialectVersion := dialectizeFile(t, original)
	text := readFile(t, dialectVersion)
	if !strings.Contains(text, "-> @Assets:收藏品") {
		t.Fatalf("elided multi-asset should convert:\n%s", text)
	}
	assertBalancesEqual(t, original, exportFile(t, dialectVersion))
}
