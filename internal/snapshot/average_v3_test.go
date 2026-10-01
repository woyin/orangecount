// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package snapshot

import (
	"orangecount/internal/ledger"
	"orangecount/internal/source"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestV3AverageIncludesZeroCostLots(t *testing.T) {
	for _, price := range []string{"20", "0"} {
		path := filepath.Join(t.TempDir(), "main.bean")
		text := `2000-01-01 open Assets:Stock STK "AVERAGE"
2000-01-01 open Assets:Cash USD
2000-01-01 open Income:Gains USD
2000-01-02 * "zero cost acquisition"
  Assets:Stock 10 STK {0 USD}
  Assets:Cash 0 USD
2000-01-03 * "second acquisition"
  Assets:Stock 10 STK {PRICE USD}
  Assets:Cash CASH USD
2000-01-04 * "sale"
  Assets:Stock -5 STK {} @ 30 USD
  Assets:Cash 150 USD
  Income:Gains
`
		text = strings.ReplaceAll(text, "PRICE", price)
		cash := "-200"
		if price == "0" {
			cash = "0"
		}
		text = strings.ReplaceAll(text, "CASH", cash)
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		result := BuildWithOptions(path, BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
		if result.Snapshot == nil {
			t.Fatalf("zero lot rejected: %+v", result.Diagnostics)
		}
		e := result.Snapshot.Evaluation()
		lots := e.Accounts["Assets:Stock"].Positions
		average, gain := "10", "-100"
		if price == "0" {
			average, gain = "0", "-150"
		}
		if len(lots) != 1 || lots[0].Units.String() != "15" || lots[0].Cost.Number.String() != average || e.Accounts["Income:Gains"].Balances["USD"].String() != gain {
			t.Fatal("zero lot was omitted from average or valued at sale price")
		}
	}
}

func TestV3AverageRejectedReductionIsAtomic(t *testing.T) {
	data, err := os.ReadFile("../../testdata/fixtures/semantics/average-explicit.bean")
	if err != nil {
		t.Fatal(err)
	}
	for _, reduction := range []string{"-40 STK {}", "-10 STK {10 USD}"} {
		text := strings.Replace(string(data), "-10 STK {}", reduction, 1)
		file, bag := ledger.ParseText("synthetic.bean", []byte(text))
		if bag.HasErrors() {
			t.Fatal(bag.All())
		}
		e := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{SemanticsProfile: ledger.V3})
		if e.Valid || e.Accounts["Assets:Stock"].Balances["STK"].String() != "30" || e.Accounts["Assets:Cash"].Balances["USD"].String() != "-500" {
			t.Fatal("rejected average transaction altered preceding state")
		}
	}
}

func TestV3AverageContinuousTradingConservesExactBasis(t *testing.T) {
	data, err := os.ReadFile("../../testdata/fixtures/semantics/average-explicit.bean")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data) + `2000-01-05 * "new purchase after reduction"
  Assets:Stock 10 STK {30 USD}
  Assets:Cash -300 USD
2000-01-06 * "second sale"
  Assets:Stock -5 STK {} @ 40 USD
  Assets:Cash 200 USD
  Income:Gains
`
	path := filepath.Join(t.TempDir(), "main.bean")
	options := BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}}
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	result := BuildWithOptions(path, options)
	if result.Snapshot == nil {
		t.Fatalf("diagnostics: %+v", result.Diagnostics)
	}
	e := result.Snapshot.Evaluation()
	lots := e.Accounts["Assets:Stock"].Positions
	if len(lots) != 1 || lots[0].Units.String() != "25" || lots[0].Cost.Number.String() != "190/9" || lots[0].Cost.Date.Raw != "2000-01-02" {
		t.Fatalf("continuous average identity: %+v", lots)
	}
	if basis := lots[0].Units.Mul(lots[0].Cost.Number).String(); basis != "4750/9" {
		t.Fatalf("residual basis=%s", basis)
	}
	if gain := e.Accounts["Income:Gains"].Balances["USD"].String(); gain != "-2050/9" {
		t.Fatalf("cumulative realized gain=%s", gain)
	}
	text += `2000-01-07 * "full liquidation"
  Assets:Stock -25 STK {} @ 40 USD
  Assets:Cash 1000 USD
  Income:Gains
`
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	result = BuildWithOptions(path, options)
	if result.Snapshot == nil {
		t.Fatalf("full liquidation diagnostics: %+v", result.Diagnostics)
	}
	e = result.Snapshot.Evaluation()
	if len(e.Accounts["Assets:Stock"].Positions) != 0 || !e.Accounts["Assets:Stock"].Balances["STK"].IsZero() {
		t.Fatal("full liquidation left inventory")
	}
	if e.Accounts["Income:Gains"].Balances["USD"].String() != "-700" || e.Accounts["Assets:Cash"].Balances["USD"].String() != "700" {
		t.Fatal("purchase basis + realized proceeds do not conserve cash")
	}
}
