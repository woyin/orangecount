// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package ledger

import "strings"

// ValidAccountName reports whether name is a Beancount account name.
//
// The rule mirrors the pinned upstream lexer (parser/lexer.l):
//
//	ACCOUNTTYPE  ([A-Z]|UTF-8-ONLY)([A-Za-z0-9\-]|UTF-8-ONLY)*
//	ACCOUNTNAME  ([A-Z0-9]|UTF-8-ONLY)([A-Za-z0-9\-]|UTF-8-ONLY)*
//	{ACCOUNTTYPE}(:{ACCOUNTNAME})+
//
// The UTF-8 arm matters for real ledgers: an account such as
// "Assets:Wallet:微信" is valid upstream, so the authoring and quick-entry paths
// must accept it instead of rejecting every non-ASCII component.
func ValidAccountName(name string) bool {
	if !strings.Contains(name, ":") {
		return false
	}
	for index, segment := range strings.Split(name, ":") {
		if segment == "" {
			return false
		}
		first := true
		for _, r := range segment {
			utf8Only := r > 0x7F
			if first {
				first = false
				if utf8Only {
					continue
				}
				if index == 0 {
					if r < 'A' || r > 'Z' {
						return false
					}
					continue
				}
				if !((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
					return false
				}
				continue
			}
			if utf8Only {
				continue
			}
			if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
				return false
			}
		}
	}
	return true
}
