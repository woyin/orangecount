package ledger

import "testing"

// ValidAccountName is the write-path gate: authoring and quick entry serialize
// an account only when this returns true. The cn scheme roots accounts in a
// Chinese name, so the gate must accept a UTF-8 root; otherwise a cn ledger
// could be read but never written through the UI.
func TestValidAccountNameAcceptsChineseRoot(t *testing.T) {
	valid := []string{"资产:钱包", "资产:钱包:微信", "收入:工资", "权益:期初", "支出:餐饮"}
	for _, name := range valid {
		if !ValidAccountName(name) {
			t.Errorf("cn root rejected: %q", name)
		}
	}
}
