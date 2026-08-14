package ledger

import "testing"

func TestTmpExprLeadingJunk(t *testing.T) {
 for _, s := range []string{
  `2000-01-01 custom "expr" foo 1 + 2 USD`,
  "2000-01-01 * \"x\"\n  Assets:Cash Extra 1 USD\n  Equity:Opening -1 USD",
  `2000-01-01 custom "expr" "x" foo 1 USD`,
 } {
  f,b:=ParseText("t.bean",[]byte(s))
  t.Logf("INPUT %q errors=%v", s, b.All())
  for _,d:= range f.Directives { t.Logf("%#v",d) }
 }
}
