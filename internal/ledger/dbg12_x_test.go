package ledger_test

import (
	"os"
	"path/filepath"
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
)

func TestDbgAllEvaluated(t *testing.T) {
	text := "2000-01-01 open Assets:Broker:Cash USD\n2000-01-01 open Assets:Broker:AAPL AAPL\noption \"operating_currency\" \"USD\"\n2026-08-16 * \"mai\" \"dai\"\n  1010 USD 10 AAPL {101.00 USD} @Assets:Broker:Cash -> @Assets:Broker:AAPL\n"
	path := filepath.Join(t.TempDir(), "m.bean")
	os.WriteFile(path, []byte(text), 0o600)
	result := snapshot.Build(path)
	for _, f := range result.Snapshot.Parsed() {
		for _, d := range f.Directives {
			switch v := d.(type) {
			case *ledger.Transaction:
				t.Logf("txn postings=%d", len(v.Postings))
				for i, p := range v.Postings {
					units := "<elided>"
					if p.Units != nil {
						units = p.Units.Number.Raw + " " + p.Units.Currency
					}
					t.Logf("p%d acct=%q units=%s", i, p.Account, units)
				}
			case ledger.Dialect:
				t.Logf("dialect amount=%s cur=%s qty=%s sec=%s src=%q dst=%q", v.Amount.Raw, v.Currency, v.Quantity.Raw, v.Security, v.SourceRef, v.DestRef)
			}
		}
	}
}
