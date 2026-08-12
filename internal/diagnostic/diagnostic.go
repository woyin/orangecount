// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package diagnostic contains accumulated, stable, localizable diagnostics.
package diagnostic

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"orangecount/internal/source"
)

type Severity string

const (
	Info    Severity = "info"
	Warning Severity = "warning"
	Error   Severity = "error"

	SeverityInfo    = Info
	SeverityWarning = Warning
	SeverityError   = Error
)

// Related is a secondary source location that helps explain a diagnostic.
type Related struct {
	Span    source.Span `json:"span"`
	Message string      `json:"message,omitempty"`
}

// Diagnostic is deliberately serializable without source text. The default
// renderer includes only a normalized source path and location; callers must
// opt in to raw source excerpts separately.
type Diagnostic struct {
	Code       string      `json:"code"`
	Severity   Severity    `json:"severity"`
	MessageKey string      `json:"message_key,omitempty"`
	Message    string      `json:"message"`
	Span       source.Span `json:"span"`
	Related    []Related   `json:"related,omitempty"`
	Path       string      `json:"path,omitempty"`
	Redacted   bool        `json:"redacted,omitempty"`
	Sequence   uint64      `json:"-"`
}

func (d Diagnostic) WithPath(path string) Diagnostic {
	d.Path = path
	return d
}

// Bag accumulates diagnostics in parse/source order and can return a stable
// sorted view. It is intentionally not a live stream: a failed reload can
// retain the previous immutable snapshot while exposing a separate Bag.
type Bag struct {
	items []Diagnostic
	seq   uint64
}

// Diagnostics is a descriptive alias for Bag used by APIs that expose a
// collection rather than its accumulation implementation.
type Diagnostics = Bag

func (b *Bag) Add(d Diagnostic) {
	if b == nil {
		return
	}
	b.seq++
	d.Sequence = b.seq
	b.items = append(b.items, d)
}

func (b *Bag) Extend(ds ...Diagnostic) {
	for _, d := range ds {
		b.Add(d)
	}
}

func (b *Bag) Len() int {
	if b == nil {
		return 0
	}
	return len(b.items)
}

func (b *Bag) Empty() bool { return b == nil || len(b.items) == 0 }

func (b *Bag) HasErrors() bool {
	if b == nil {
		return false
	}
	for _, d := range b.items {
		if d.Severity == Error {
			return true
		}
	}
	return false
}

// All returns diagnostics sorted by source location and then stable code.
func (b *Bag) All() []Diagnostic {
	if b == nil {
		return nil
	}
	out := append([]Diagnostic(nil), b.items...)
	sort.SliceStable(out, func(i, j int) bool {
		a, c := out[i], out[j]
		if a.Path != c.Path {
			return a.Path < c.Path
		}
		if a.Span.Start != c.Span.Start {
			return a.Span.Start < c.Span.Start
		}
		if a.Severity != c.Severity {
			return severityRank(a.Severity) < severityRank(c.Severity)
		}
		if a.Code != c.Code {
			return a.Code < c.Code
		}
		return a.Sequence < c.Sequence
	})
	return out
}

func severityRank(s Severity) int {
	switch s {
	case Error:
		return 0
	case Warning:
		return 1
	default:
		return 2
	}
}

var messages = map[string]map[string]string{
	"E-INCLUDE-CYCLE":    {"en": "include cycle detected", "zh-CN": "检测到 include 循环"},
	"E-INCLUDE-READ":     {"en": "cannot read included file", "zh-CN": "无法读取 include 文件"},
	"E-PARSE-DATE":       {"en": "invalid date", "zh-CN": "日期无效"},
	"E-PARSE-DIRECTIVE":  {"en": "unknown or malformed directive", "zh-CN": "未知或格式错误的指令"},
	"E-PARSE-EXPECTED":   {"en": "unexpected token", "zh-CN": "出现意外的标记"},
	"E-PARSE-TOKEN":      {"en": "invalid token", "zh-CN": "标记无效"},
	"E-PARSE-STRING":     {"en": "unterminated string", "zh-CN": "字符串未结束"},
	"E-SOURCE-UTF8":      {"en": "source is not valid UTF-8", "zh-CN": "源文件不是有效的 UTF-8"},
	"E-EVAL-OPEN":        {"en": "account is opened more than once or has an invalid opening date", "zh-CN": "账户重复开户或开户日期无效"},
	"E-EVAL-REOPEN":      {"en": "account is reopened after it was closed", "zh-CN": "账户在销户后再次开户"},
	"E-EVAL-CLOSE":       {"en": "account close is invalid or occurs after a close", "zh-CN": "账户销户无效或发生在销户之后"},
	"E-EVAL-POSTING":     {"en": "posting references an account outside its open lifecycle", "zh-CN": "记账引用了不在开户生命周期内的账户"},
	"E-EVAL-CURRENCY":    {"en": "posting currency is not allowed for this account", "zh-CN": "记账货币不在该账户允许的货币范围内"},
	"E-EVAL-UNBALANCED":  {"en": "transaction does not balance", "zh-CN": "交易未平衡"},
	"E-EVAL-INFER":       {"en": "an omitted posting amount cannot be inferred", "zh-CN": "无法推断省略的记账金额"},
	"E-EVAL-BALANCE":     {"en": "balance assertion failed", "zh-CN": "余额断言失败"},
	"E-EVAL-INVENTORY":   {"en": "posting consumes more inventory than is available", "zh-CN": "记账消耗的库存超过可用库存"},
	"E-EVAL-PAD":         {"en": "pad source account is unavailable", "zh-CN": "pad 的来源账户不可用"},
	"E-EVAL-TOLERANCE":   {"en": "invalid balance tolerance", "zh-CN": "余额容差无效"},
	"E-EVAL-OPTION":      {"en": "option is unsupported or has an invalid value", "zh-CN": "option 不受支持或值无效"},
	"W-EVAL-PAD-UNUSED":  {"en": "pad directive was not consumed by a later balance assertion", "zh-CN": "pad 指令未被后续余额断言使用"},
	"W-EVAL-UNSUPPORTED": {"en": "directive is preserved but not evaluated", "zh-CN": "指令已保留但尚未求值"},
	"W-PLUGIN-MIGRATION": {"en": "plugin declarations are not executed; migrate this ledger to core directives", "zh-CN": "不会执行 plugin 声明；请将此账本迁移到核心指令"},
	"W-UNSUPPORTED":      {"en": "construct is preserved but not evaluated yet", "zh-CN": "语法已保留，但尚未支持求值"},
}

// New creates a diagnostic from a stable code. args are only used with a
// caller-provided message key and are intentionally not interpolated by the
// default catalog, preventing accidental leakage of ledger values.
func New(code string, severity Severity, span source.Span, message ...string) Diagnostic {
	d := Diagnostic{Code: code, Severity: severity, MessageKey: code, Message: LocalizeCode(code, "en"), Span: span, Redacted: true}
	if len(message) > 0 && message[0] != "" {
		d.Message = RedactMessage(message[0])
		d.MessageKey = ""
	}
	return d
}

func LocalizeCode(code, locale string) string {
	if locale == "" {
		locale = "en"
	}
	if m, ok := messages[code]; ok {
		if msg, ok := m[locale]; ok {
			return msg
		}
		if msg, ok := m["en"]; ok {
			return msg
		}
	}
	return code
}

// ReleasedErrorCodes returns the stable error-severity codes in the built-in
// diagnostic catalogue. Callers receive a sorted copy so coverage checks do
// not depend on map iteration order.
func ReleasedErrorCodes() []string {
	codes := make([]string, 0, len(messages))
	for code := range messages {
		if strings.HasPrefix(code, "E-") {
			codes = append(codes, code)
		}
	}
	sort.Strings(codes)
	return codes
}

func Localize(d Diagnostic, locale string) Diagnostic {
	if d.MessageKey != "" {
		d.Message = LocalizeCode(d.MessageKey, locale)
	}
	return d
}

func (d Diagnostic) MessageKeyOrCode() string {
	if d.MessageKey != "" {
		return d.MessageKey
	}
	return d.Code
}

// RenderHuman writes a stable one-line-per-diagnostic report. Paths are taken
// from the diagnostic or graph, never from source excerpts.
func RenderHuman(w io.Writer, ds []Diagnostic, locale string) error {
	for _, original := range ds {
		d := Localize(original, locale)
		path := d.Path
		if path == "" {
			path = "<source>"
		}
		line, col := d.Span.StartLine, d.Span.StartColumn
		if line == 0 {
			line, col = 1, 1
		}
		if _, err := fmt.Fprintf(w, "%s:%d:%d: %s %s: %s\n", path, line, col, d.Severity, d.Code, d.Message); err != nil {
			return err
		}
	}
	return nil
}

// RenderJSON emits a deterministic JSON array. Since Diagnostic does not
// contain source excerpts it is safe for normal local logs.
func RenderJSON(w io.Writer, ds []Diagnostic, locale string) error {
	out := make([]Diagnostic, len(ds))
	for i, d := range ds {
		out[i] = Localize(d, locale)
		out[i].Related = append([]Related(nil), d.Related...)
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(out)
}

// RedactMessage strips line breaks and a caller-selected set of sensitive
// fragments. It is intentionally conservative: callers should prefer static
// catalog messages and avoid passing raw ledger text in the first place.
func RedactMessage(message string, sensitive ...string) string {
	message = strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(message)
	for _, value := range sensitive {
		if value != "" {
			message = strings.ReplaceAll(message, value, "[redacted]")
		}
	}
	return strings.TrimSpace(message)
}
