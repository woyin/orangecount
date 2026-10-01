// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package desktop_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orangecount/internal/desktop"
	"orangecount/internal/ledger"
	"orangecount/internal/report"
	"orangecount/internal/snapshot"
)

func TestNativeHoldingsReplaysAverageThroughSalesClearingAndRebuy(t *testing.T) {
	s, err := desktop.OpenWithOptions("../../testdata/fixtures/native/average-history.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3, Watch: snapshot.WatchOptions{PollInterval: time.Hour}})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, expected := range []struct {
		date, quantity, basis, average string
		lots                           int
	}{
		{"2000-01-03", "30", "500", "50/3", 2},
		{"2000-01-04", "20", "1000/3", "50/3", 1},
		{"2000-01-05", "30", "2200/3", "220/9", 2},
		{"2000-01-06", "18", "440", "220/9", 1},
		{"2000-01-07", "0", "0", "", 0},
		{"2000-01-08", "5", "40", "8", 1},
	} {
		view, err := s.Read(desktop.Request{View: "holdings", Account: "Assets:Stock", DateFrom: expected.date, DateTo: expected.date})
		if err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(view)
		var wire struct {
			Holdings *struct {
				Groups []struct {
					Quantity report.PresentedDecimal  `json:"quantity"`
					Basis    *report.PresentedDecimal `json:"cost_basis"`
					Average  *report.PresentedDecimal `json:"average_cost"`
					Lots     []struct {
						Date  *string `json:"cost_date"`
						Label *string `json:"cost_label"`
					} `json:"lots"`
				} `json:"groups"`
			} `json:"holdings"`
		}
		if err := json.Unmarshal(data, &wire); err != nil {
			t.Fatal(err)
		}
		if wire.Holdings == nil {
			t.Fatal("missing holdings payload")
		}
		if expected.lots == 0 {
			if len(wire.Holdings.Groups) != 0 {
				t.Fatal("full liquidation must not show current rebought inventory in the past")
			}
			continue
		}
		if len(wire.Holdings.Groups) != 1 {
			t.Fatalf("one STK/USD group expected: %s", data)
		}
		group := wire.Holdings.Groups[0]
		if group.Quantity.Exact != expected.quantity || group.Basis == nil || group.Basis.Exact != expected.basis || group.Average == nil || group.Average.Exact != expected.average || len(group.Lots) != expected.lots {
			t.Fatalf("wrong historical average on %s: %s", expected.date, data)
		}
		if expected.date == "2000-01-04" && (group.Lots[0].Date == nil || *group.Lots[0].Date != "2000-01-02" || group.Lots[0].Label != nil) {
			t.Fatal("merged average lot must retain earliest date and absent label")
		}
	}
}

func TestNativeHoldingsAverageHistoryAndReductionsExcludeMergeLegs(t *testing.T) {
	s, err := desktop.OpenWithOptions("../../testdata/fixtures/native/average-history.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	view, err := s.Read(desktop.Request{View: "holdings", Account: "Assets:Stock", DateTo: "2000-01-08"})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(view.Holdings)
	var wire struct {
		Events []struct {
			Date     string                   `json:"date"`
			Quantity report.PresentedDecimal  `json:"quantity"`
			Gross    *report.PresentedDecimal `json:"gross_gain"`
			Net      *report.PresentedDecimal `json:"net_gain"`
		} `json:"events"`
		History []struct {
			Points []struct {
				Date    string                   `json:"date"`
				Average *report.PresentedDecimal `json:"average_cost"`
				Segment int                      `json:"segment"`
			} `json:"points"`
		} `json:"average_history"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		t.Fatal(err)
	}
	if len(wire.Events) != 3 {
		t.Fatalf("only three economic reductions, no M legs: %s", data)
	}
	for i, gain := range []string{"400/3", "920/3", "10"} {
		if wire.Events[i].Gross == nil || wire.Events[i].Gross.Exact != gain || wire.Events[i].Net == nil || wire.Events[i].Net.Exact != gain {
			t.Fatalf("wrong exact realized gain: %s", data)
		}
	}
	if len(wire.History) != 1 || len(wire.History[0].Points) != 7 {
		t.Fatalf("history must include clearing and rebuy dates: %s", data)
	}
	points := wire.History[0].Points
	for i, average := range []string{"10", "50/3", "50/3", "220/9", "220/9", "", "8"} {
		if average == "" {
			if points[i].Average != nil {
				t.Fatal("empty position has no average cost")
			}
		} else if points[i].Average == nil || points[i].Average.Exact != average {
			t.Fatal("history must agree with independently calculated averages")
		}
	}
	if points[6].Segment == points[4].Segment {
		t.Fatal("rebuy must not draw a line across full liquidation")
	}
}

func TestNativeHoldingsAverageMergeShowsOriginalContributions(t *testing.T) {
	s, err := desktop.OpenWithOptions("../../testdata/fixtures/native/average-history.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	view, err := s.Read(desktop.Request{View: "holdings", Account: "Assets:Stock", DateTo: "2000-01-04"})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(view.Holdings.Events[0])
	var event struct {
		Merge []report.HoldingLot `json:"merge_lots"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatal(err)
	}
	if len(event.Merge) != 2 {
		t.Fatal("a source reduction must retain its two pre-merge contributed lots")
	}
	quantities := map[string]string{}
	for _, lot := range event.Merge {
		quantities[lot.CostNumber.Exact] = lot.Quantity.Exact
		if lot.Source.Line >= view.Holdings.Events[0].Source.Line {
			t.Fatal("pre-merge lots must point to their actual prior establishment")
		}
	}
	if quantities["10"] != "10" || quantities["20"] != "20" {
		t.Fatal("pool contributions describe 10@10 and 20@20 without duplicating the sold cost")
	}
	if view.Holdings.Events[0].CostBasis.Exact != "500/3" {
		t.Fatal("merge pool cost500 is not the consumed cost500/3")
	}
}

func TestNativeHoldingsIntradayRebuyStartsANewAverageSegment(t *testing.T) {
	s, err := desktop.OpenWithOptions("../../testdata/fixtures/native/holdings-intraday.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	view, err := s.Read(desktop.Request{View: "holdings", Account: "Assets:Stock", DateTo: "2000-01-03"})
	if err != nil {
		t.Fatal(err)
	}
	if !view.State.Valid || len(view.Holdings.AverageHistory) != 1 {
		t.Fatal("expected valid stock history")
	}
	points := view.Holdings.AverageHistory[0].Points
	if len(points) != 2 || points[1].AverageCost == nil || points[1].AverageCost.Exact != "8" || points[1].Segment <= points[0].Segment {
		t.Fatal("same-day full liquidation and rebuy must coalesce to 5@8 in a new segment")
	}
}

func TestNativeHoldingsReacquisitionOfSameKeyUsesNewEstablishment(t *testing.T) {
	input := `2000-01-01 open Assets:Stock STK "FIFO"
2000-01-01 open Assets:Cash USD
2000-01-01 open Income:Gains USD
2000-01-02 * "original lot"
  Assets:Stock 1 STK {1 USD, 2000-01-02, "same"}
  Assets:Cash -1 USD
2000-01-03 * "clear"
  Assets:Stock -1 STK {} @ 2 USD
  Assets:Cash 2 USD
  Income:Gains
2000-01-04 * "new establishment with identical cost key"
  Assets:Stock 2 STK {1 USD, 2000-01-02, "same"}
  Assets:Cash -2 USD
`
	path := filepath.Join(t.TempDir(), "main.bean")
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	view, err := s.Read(desktop.Request{View: "holdings", Account: "Assets:Stock"})
	if err != nil || !view.State.Valid || len(view.Holdings.Groups) != 1 {
		t.Fatal("same-key re-establishment needs a valid new inventory")
	}
	group := view.Holdings.Groups[0]
	if group.Quantity.Exact != "2" || group.CostBasis.Exact != "2" || len(group.Lots) != 1 || group.Lots[0].Source.Line != 12 || group.Lots[0].CostDate == nil || *group.Lots[0].CostDate != "2000-01-02" {
		t.Fatal("reacquisition must retain its declared cost date and use its new establishment source")
	}
	if view.Holdings.Events[0].Lots[0].Source.Line != 5 || view.Holdings.Events[0].CostBasis.Exact != "1" {
		t.Fatal("later reacquisition must not mutate the earlier consumption provenance")
	}
}

func TestNativeHoldingsSplitTotalPriceAndConservativeFees(t *testing.T) {
	s, err := desktop.OpenWithOptions("../../testdata/fixtures/native/holdings-trades.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, want := range []struct{ account, basis, gross string }{{"Assets:Fifo", "60", "40"}, {"Assets:Lifo", "70", "30"}} {
		view, err := s.Read(desktop.Request{View: "holdings", Account: want.account, DateTo: "2000-01-04"})
		if err != nil {
			t.Fatal(err)
		}
		if !view.State.Valid {
			t.Fatal("public trade fixture must be valid")
		}
		if len(view.Holdings.Events) != 1 {
			t.Fatal("split booked legs must be one source reduction")
		}
		event := view.Holdings.Events[0]
		if event.Proceeds == nil || event.Proceeds.Exact != "100" || event.CostBasis == nil || event.CostBasis.Exact != want.basis || event.GrossGain == nil || event.GrossGain.Exact != want.gross || len(event.Lots) != 2 {
			t.Fatalf("wrong original @@ total or lot allocation: %+v", event)
		}
		if event.NetGain != nil || event.NetReason != "ambiguous-trade-fees" {
			t.Fatal("another reduction in the same transaction prevents settlement attribution")
		}
		for _, lot := range event.Lots {
			if lot.Source.Line >= event.Source.Line {
				t.Fatal("consumed lot source must retain its establishment, independently of the reducing source posting")
			}
		}
		before, err := s.Read(desktop.Request{View: "holdings", Account: want.account, DateTo: "2000-01-03"})
		if err != nil || len(before.Holdings.Groups) != 1 || before.Holdings.Groups[0].Quantity.Exact != "5" || before.Holdings.Groups[0].CostBasis.Exact != "80" {
			t.Fatal("historical inventory must include both fully consumed and surviving lots")
		}
	}
	for _, account := range []string{"Assets:Fees", "Assets:Unassigned"} {
		view, err := s.Read(desktop.Request{View: "holdings", Account: account})
		if err != nil {
			t.Fatal(err)
		}
		event := view.Holdings.Events[0]
		if event.GrossGain == nil || event.GrossGain.Exact != "100" {
			t.Fatal("fees do not erase provable gross gain")
		}
		if account == "Assets:Fees" {
			if event.Fees == nil || event.Fees.Exact != "5" || event.NetGain == nil || event.NetGain.Exact != "95" {
				t.Fatal("explicit fee plus reconciled 195 cash yields net 95")
			}
		} else if event.NetGain != nil || event.NetReason != "unattributed-expenses" {
			t.Fatal("unassigned expense must not be guessed from its account name")
		}
	}
}

func TestNativeHoldingsPreservesNullableLotKeysAndSignedInventory(t *testing.T) {
	s, err := desktop.OpenWithOptions("../../testdata/fixtures/native/holdings-identities.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	read := func(account string) *report.HoldingsProjection {
		t.Helper()
		view, err := s.Read(desktop.Request{View: "holdings", Account: account})
		if err != nil || !view.State.Valid {
			t.Fatal("identity fixture must be readable and valid")
		}
		return view.Holdings
	}
	tagged := read("Assets:Tagged").Groups[0]
	if len(tagged.Lots) != 2 || tagged.Lots[0].ID == tagged.Lots[1].ID {
		t.Fatal("nil and empty labels must be separate lot identities")
	}
	labels := map[string]int{}
	for _, lot := range tagged.Lots {
		if lot.CostLabel == nil {
			labels["nil"]++
		} else if *lot.CostLabel == "" {
			labels["empty"]++
		}
	}
	if labels["nil"] != 1 || labels["empty"] != 1 {
		t.Fatal("label presence must survive projection")
	}
	zero := read("Assets:Zero").Groups[0]
	plain := read("Assets:Plain").Groups[0]
	if zero.CostBasis == nil || zero.CostBasis.Exact != "0" || zero.AverageCost.Exact != "0" || plain.CostCurrency != nil || plain.CostBasis != nil || plain.AverageCost != nil {
		t.Fatal("zero cost is known; plain inventory has unknown cost")
	}
	short := read("Assets:Short")
	if short.Groups[0].Quantity.Exact != "-3" || short.Groups[0].CostBasis.Exact != "-15" || len(short.Events) != 1 || short.Events[0].ConsumedQuantity.Exact != "0" || short.Events[0].GrossGain != nil || short.Events[0].GrossReason != "short-position-opening" {
		t.Fatal("opening NONE short preserves signed inventory and is not realized long gain")
	}
}

func TestNativeHoldingsValuationUsesCutoffInventoryAndSeparateCostQuotes(t *testing.T) {
	s, err := desktop.OpenWithOptions("../../testdata/fixtures/native/history.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, want := range []struct{ date, market, gain string }{{"2000-01-31", "", ""}, {"2000-02-29", "200", "100"}} {
		view, err := s.Read(desktop.Request{View: "holdings", Account: "Assets:Stock", Currency: "USD", Valuation: "market-value", DateTo: want.date})
		if err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(view.Holdings)
		var wire struct {
			Groups []struct {
				Cost   *report.AccountValuation `json:"cost_valuation"`
				Market *report.AccountValuation `json:"market_valuation"`
				Gain   *report.PresentedDecimal `json:"unrealized_gain"`
			} `json:"groups"`
		}
		if err := json.Unmarshal(data, &wire); err != nil {
			t.Fatal(err)
		}
		if len(wire.Groups) != 1 || wire.Groups[0].Cost == nil || wire.Groups[0].Market == nil {
			t.Fatal("required cost and market valuation states missing")
		}
		group := wire.Groups[0]
		if group.Cost.CompleteTotal == nil || group.Cost.CompleteTotal.Exact != "100" {
			t.Fatal("cost remains known even when market quote is missing")
		}
		if want.market == "" {
			if group.Market.CompleteTotal != nil || group.Gain != nil || group.Market.Completeness != "unavailable" {
				t.Fatal("January cannot borrow future price or manufacture zero gain")
			}
		} else if group.Market.CompleteTotal == nil || group.Market.CompleteTotal.Exact != want.market || group.Gain == nil || group.Gain.Exact != want.gain || len(group.Market.Quotes) != 1 || group.Market.Quotes[0].Date != "2000-02-15" {
			t.Fatal("held inventory and cutoff quote must yield exact market 200 and unrealized 100")
		}
	}
}

func TestNativeHoldingsCommodityGroupingPreservesAccountLots(t *testing.T) {
	s, err := desktop.OpenWithOptions("../../testdata/fixtures/native/holdings-trades.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var request desktop.Request
	if err := json.Unmarshal([]byte(`{"view":"holdings","account":"Assets","group_by":"commodity","date_to":"2000-01-03"}`), &request); err != nil {
		t.Fatal(err)
	}
	view, err := s.Read(request)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Holdings.Groups) != 2 {
		t.Fatal("STK/USD lots from both accounts must share one domain, separate from plain cash")
	}
	for _, group := range view.Holdings.Groups {
		if group.Commodity == "STK" && (group.Quantity.Exact != "10" || group.CostBasis.Exact != "160" || group.AverageCost.Exact != "16" || len(group.Lots) != 4) {
			t.Fatal("grouped cost must sum only the same commodity and cost currency while retaining all accounts")
		}
	}
	if len(view.Holdings.AverageHistory) != 1 || view.Holdings.AverageHistory[0].Points[1].AverageCost.Exact != "16" {
		t.Fatal("grouped average history must reconcile with the same domain's inventory")
	}
}

func TestNativeHoldingsPagesGroupsWithoutChangingFullGroupAmounts(t *testing.T) {
	s, err := desktop.OpenWithOptions("../../testdata/fixtures/native/holdings-trades.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	first, err := s.Read(desktop.Request{View: "holdings", Account: "Assets", DateTo: "2000-01-03", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Holdings.Groups) != 1 {
		t.Fatal("holdings limit must page mathematical groups instead of being ignored")
	}
	next, err := s.Read(desktop.Request{View: "holdings", Account: "Assets", DateTo: "2000-01-03", Limit: 1, Offset: 1, SnapshotID: first.State.SnapshotID})
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Holdings.Groups) != 1 || first.Holdings.Groups[0].ID == next.Holdings.Groups[0].ID || next.Holdings.Groups[0].Quantity.Exact != "5" || next.Holdings.Groups[0].CostBasis.Exact != "80" {
		t.Fatal("next group must preserve full quantity and basis independently of its nested lot page")
	}
}

func TestNativeHoldingsSourceReductionLotsAreRetrievableBeyondPreview(t *testing.T) {
	var input strings.Builder
	input.WriteString("2000-01-01 open Assets:Cash USD\n2000-01-01 open Assets:Stock STK \"FIFO\"\n2000-01-01 open Income:Gains USD\n2000-01-02 * \"many separate lots\"\n")
	for i := 0; i < 101; i++ {
		fmt.Fprintf(&input, "  Assets:Stock 1 STK {1 USD, \"lot-%d\"}\n", i)
	}
	input.WriteString("  Assets:Cash -101 USD\n2000-01-03 * \"full sale\"\n  Assets:Stock -101 STK {} @@ 202 USD\n  Assets:Cash 202 USD\n  Income:Gains\n")
	path := filepath.Join(t.TempDir(), "main.bean")
	if err := os.WriteFile(path, []byte(input.String()), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	view, err := s.Read(desktop.Request{View: "holdings", Account: "Assets:Stock"})
	if err != nil || !view.State.Valid {
		t.Fatal("wide lot source must be valid")
	}
	event := view.Holdings.Events[0]
	if event.LotsPage.TotalCount != 101 || len(event.Lots) != 100 || event.GrossGain.Exact != "101" {
		t.Fatal("preview must disclose truncation without changing full sale gain")
	}
	reversed, err := s.Read(desktop.Request{View: "holdings", Account: "Assets:Stock", HoldingSection: "lots", EventID: event.ID, SnapshotID: view.State.SnapshotID, Sort: "descending", Limit: 1})
	if err != nil || reversed.Holdings.Events[0].Lots[0].CostLabel == nil || *reversed.Holdings.Events[0].Lots[0].CostLabel != "lot-100" {
		t.Fatal("descending applies to the selected source lot collection, not an unrelated group preview")
	}
	var request desktop.Request
	data, _ := json.Marshal(map[string]any{"view": "holdings", "holding_section": "lots", "event_id": event.ID, "snapshot_id": view.State.SnapshotID, "account": "Assets:Stock", "offset": 100, "limit": 1})
	if err := json.Unmarshal(data, &request); err != nil {
		t.Fatal(err)
	}
	detail, err := s.Read(request)
	if err != nil || len(detail.Holdings.Events) != 1 || len(detail.Holdings.Events[0].Lots) != 1 || detail.Holdings.Events[0].LotsPage.Offset != 100 {
		t.Fatal("all consumed lots must be reachable through a snapshot-bound source event selector")
	}
	if err := os.WriteFile(path, []byte(input.String()+"2000-01-04 price STK 4 USD\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := s.Reload(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(request); !errors.Is(err, desktop.ErrStale) {
		t.Fatal("old lot page may not cross a snapshot publication")
	}
	current, err := s.Poll()
	if err != nil {
		t.Fatal(err)
	}
	request.SnapshotID = current.SnapshotID
	if _, err := s.Read(request); !errors.Is(err, desktop.ErrRequest) {
		t.Fatal("old source event ID must not name a record in the new snapshot")
	}
}

func TestNativeHoldingsRejectsOversizedResult(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.bean")
	input := "2000-01-01 open Assets:Stock STK \"FIFO\"\n2000-01-01 open Assets:Cash USD\n2000-01-01 open Income:Gains USD\n2000-01-02 * \"buy\"\n  Assets:Stock 1 STK {1 USD}\n  Assets:Cash -1 USD\n2000-01-03 * \"" + strings.Repeat("x", 17<<20) + "\"\n  Assets:Stock -1 STK {} @ 2 USD\n  Assets:Cash 2 USD\n  Income:Gains\n"
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	state, err := s.Poll()
	if err != nil || !state.Valid {
		t.Fatal("budget test needs valid accounting, not a parse failure")
	}
	if _, err := s.Read(desktop.Request{View: "holdings", Account: "Assets:Stock"}); !errors.Is(err, desktop.ErrResultBudget) {
		t.Fatal("16 MiB result budget must produce a structured error")
	}
}

func TestNativeHoldingsAverageMergePoolIsRetrievableBeyondPreview(t *testing.T) {
	var input strings.Builder
	input.WriteString("2000-01-01 open Assets:Cash USD\n2000-01-01 open Assets:Stock STK \"AVERAGE\"\n2000-01-01 open Income:Gains USD\n2000-01-02 * \"many contributions\"\n")
	for i := 0; i < 101; i++ {
		fmt.Fprintf(&input, "  Assets:Stock 1 STK {1 USD, \"lot-%d\"}\n", i)
	}
	input.WriteString("  Assets:Cash -101 USD\n2000-01-03 * \"full sale\"\n  Assets:Stock -101 STK {} @@ 202 USD\n  Assets:Cash 202 USD\n  Income:Gains\n")
	path := filepath.Join(t.TempDir(), "main.bean")
	if err := os.WriteFile(path, []byte(input.String()), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	view, err := s.Read(desktop.Request{View: "holdings", Account: "Assets:Stock"})
	if err != nil || !view.State.Valid || len(view.Holdings.Events) != 1 {
		t.Fatal("wide AVERAGE pool must produce exactly one economic sale")
	}
	event := view.Holdings.Events[0]
	if event.MergeLotsPage.TotalCount != 101 || len(event.MergeLots) != 100 || event.LotsPage.TotalCount != 1 || event.CostBasis.Exact != "101" || event.GrossGain.Exact != "101" {
		t.Fatal("pool preview must preserve all contributions and the actual full-sale cost independently")
	}
	next, err := s.Read(desktop.Request{View: "holdings", Account: "Assets:Stock", HoldingSection: "lots", EventID: event.ID, LotKind: "merged", SnapshotID: view.State.SnapshotID, Offset: 100, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	detail := next.Holdings.Events[0]
	if len(detail.MergeLots) != 1 || detail.MergeLotsPage.HasMore || detail.MergeLots[0].Source.Line >= event.Source.Line || detail.MergeLots[0].Origin != "posting" || detail.CostBasis.Exact != "101" {
		t.Fatal("last original contribution must be reachable without altering actual consumed cost")
	}
	for _, lot := range event.MergeLots {
		if lot.ID == detail.MergeLots[0].ID {
			t.Fatal("merge page may not duplicate a preview contribution")
		}
	}
}

func TestNativeHoldingsUnknownReasonsAndEventDateConversion(t *testing.T) {
	s, err := desktop.OpenWithOptions("../../testdata/fixtures/native/holdings-unknowns.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	view, err := s.Read(desktop.Request{View: "holdings", Account: "Assets:Stock"})
	if err != nil || !view.State.Valid {
		t.Fatal("unknown-reason fixture must be valid accounting")
	}
	want := []string{"transfer-or-exchange", "missing-sale-price", "missing-sale-price", "missing-proceeds-quote", "", "negative-sale-price"}
	if len(view.Holdings.Events) != len(want) {
		t.Fatal("M-independent reductions and duplicate records must retain their multiplicity")
	}
	for i, reason := range want {
		event := view.Holdings.Events[i]
		if event.GrossReason != reason || (reason != "" && event.GrossGain != nil) {
			t.Fatalf("event %d should have reason %q: %+v", i, reason, event)
		}
	}
	if view.Holdings.Events[1].ID == view.Holdings.Events[2].ID || view.Holdings.Events[1].Source.Line == view.Holdings.Events[2].Source.Line {
		t.Fatal("identical source content on different lines must not collapse")
	}
	if view.Holdings.Events[4].GrossGain.Exact != "0" || view.Holdings.Events[4].NetGain != nil || view.Holdings.Events[4].NetReason != "proceeds-not-reconciled" {
		t.Fatal("declared gross zero and unreconciled cash are separate states")
	}
	for _, expected := range []struct{ date, cost, gain string }{{"2000-01-03", "", ""}, {"2000-01-06", "25", "50"}} {
		priced, err := s.Read(desktop.Request{View: "holdings", Account: "Assets:Stock", Currency: "EUR", Valuation: "market-value", DateTo: expected.date})
		if err != nil {
			t.Fatal(err)
		}
		group := priced.Holdings.Groups[0]
		if group.MarketValuation.CompleteTotal == nil {
			t.Fatal("STK/EUR quote exists independently of USD cost conversion")
		}
		if expected.cost == "" {
			if group.CostValuation.CompleteTotal != nil || group.UnrealizedGain != nil || group.GainReason != "missing-cost-quote" || group.MarketValuation.CompleteTotal.Exact != "120" {
				t.Fatal("available market value must not invent target-currency cost")
			}
		} else if group.CostValuation.CompleteTotal.Exact != expected.cost || group.UnrealizedGain.Exact != expected.gain || group.CostValuation.Quotes[0].Date != "2000-01-06" {
			t.Fatal("cutoff inverse quote must convert 50 USD cost to 25 EUR and gain 50 EUR")
		}
	}
}

func TestNativeHoldingsGroupingNeverCombinesDifferentCostCurrencies(t *testing.T) {
	s, err := desktop.OpenWithOptions("../../testdata/fixtures/native/holdings-unknowns.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, groupBy := range []string{"commodity", "cost_currency"} {
		view, err := s.Read(desktop.Request{View: "holdings", Account: "Assets", GroupBy: groupBy, DateTo: "2000-01-03"})
		if err != nil {
			t.Fatal(err)
		}
		if len(view.Holdings.Groups) != 4 {
			t.Fatal("two cost domains and two plain currency domains must remain distinct")
		}
		for _, group := range view.Holdings.Groups {
			if group.CostCurrency == nil {
				continue
			}
			switch *group.CostCurrency {
			case "USD":
				if group.Quantity.Exact != "10" || group.CostBasis.Exact != "100" || group.AverageCost.Exact != "10" {
					t.Fatal("same commodity/USD aggregate retains transfer conservation")
				}
			case "EUR":
				if group.Quantity.Exact != "3" || group.CostBasis.Exact != "60" || group.AverageCost.Exact != "20" {
					t.Fatal("EUR cost is an independent weighted average")
				}
			default:
				t.Fatal("unexpected cost currency")
			}
		}
	}
	for _, r := range []desktop.Request{{View: "holdings", GroupBy: "all"}, {View: "holdings", Scope: "direct"}, {View: "journal", GroupBy: "commodity"}, {View: "holdings", Text: "silently ignored"}, {View: "holdings", HoldingSection: "lots"}, {View: "holdings", GroupID: "unbound", HoldingSection: "history"}} {
		if _, err := s.Read(r); !errors.Is(err, desktop.ErrRequest) {
			t.Fatal("invalid or unsupported holdings combinations must be rejected")
		}
	}
}
