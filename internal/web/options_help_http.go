// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"fmt"
	"net/http"
	"strings"

	"orangecount/internal/repairguidance"
)

func (s *Server) handleOptions(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && !requireSameOrigin(w, r) {
		return
	}
	if r.Method == http.MethodGet {
		current := s.store.Current()
		evaluationOptions := map[string]string{}
		if current != nil {
			for key, value := range current.Evaluation().Options {
				evaluationOptions[key] = value
			}
		}
		s.optionsMu.RLock()
		for key, value := range s.options {
			evaluationOptions[key] = value
		}
		s.optionsMu.RUnlock()
		writeJSON(w, struct {
			Options map[string]string `json:"options"`
		}{Options: evaluationOptions})
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var values map[string]string
	if err := decodeJSONBody(w, r, &values, 64<<10); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	for key, value := range values {
		if err := validateLocalOption(key, value); err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	s.optionsMu.Lock()
	for key, value := range values {
		s.options[key] = value
	}
	s.optionsMu.Unlock()
	writeJSON(w, struct {
		Saved bool `json:"saved"`
	}{Saved: true})
}

// validateLocalOption checks a settable local option's value; unknown keys
// and out-of-domain values are rejected so the options file stays clean.
func validateLocalOption(key, value string) error {
	switch key {
	case "locale":
		if value != "en" && value != "zh-CN" {
			return fmt.Errorf("locale must be en or zh-CN")
		}
	case "currency":
		if len(value) < 2 || len(value) > 12 {
			return fmt.Errorf("currency must be a short uppercase code")
		}
		for _, character := range value {
			if character < 'A' || character > 'Z' {
				return fmt.Errorf("currency must be a short uppercase code")
			}
		}
	case "time":
		if value != "all" && value != "year" && value != "month" {
			return fmt.Errorf("time must be all, year, or month")
		}
	default:
		return fmt.Errorf("unsupported option %q", key)
	}
	return nil
}

func helpSections(locales ...string) []map[string]string {
	locale := repairguidance.LocaleEnglish
	if len(locales) > 0 && locales[0] == repairguidance.LocaleChinese {
		locale = repairguidance.LocaleChinese
	}
	sections := []struct{ id, enTitle, enBody, zhTitle, zhBody string }{
		{"navigation", "Navigation", "Use the sidebar or the menu button to move between reports.", "导航", "使用侧边栏或菜单按钮在报表之间移动。"},
		{"filters", "Filters", "Global time, account, and text filters are bookmarkable URL state.", "筛选", "全局时间、账户和文本筛选会保存在可收藏的 URL 状态中。"},
		{"options", "Options", "The color scheme, locale, and fava options live on the Options page. Beancount options are declared in the ledger and shown there read-only.", "选项", "配色、语言和 Fava 选项位于选项页。Beancount 选项在账本中声明，并以只读方式显示。"},
		{"editor", "Editor safety", "Validate before saving. Saves are atomic, backed up, and revalidated before publication.", "编辑器安全", "保存前会验证。保存采用原子写入、备份，并在发布前重新验证。"},
		{"import", "Import review", "Preview imported postings and explicitly commit them to a selected ledger file.", "导入审核", "预览导入的记账，并明确提交到选定的账本文件。"},
		{"prices", "Local prices", "Market valuation uses only price directives in the local ledger. Missing quotes are shown as unavailable; no external provider is contacted.", "本地价格", "市值估算只使用本地账本中的 price 指令。缺少报价时显示不可用；不会联系外部服务。"},
		{"plugins", "Plugin migration", "Python plugins are never executed. Plugin directives remain visible as diagnostics so they can be migrated explicitly.", "插件迁移", "不会执行 Python 插件。插件指令会保留为诊断，以便明确迁移。"},
		{"diagnostics", "Diagnostics", "Open a diagnostic to see its repair order, safe checks, generic example, and local source context.", "诊断", "打开诊断可查看修复顺序、安全检查、通用示例和本地源上下文。"},
		{"shortcuts", "Keyboard", "Tab reaches controls; Enter applies filters and runs queries.", "键盘", "使用 Tab 到达控件，使用 Enter 应用筛选并运行查询。"},
		{"quick-entry", "Quick Entry", "Use the Quick tab (a q) to capture two-posting transactions with compact shorthand. Template form: 'lunch 28 @wechat'. Explicit form: '28 CNY @source -> @dest : narration'. Press Ctrl+Enter to preview, then Ctrl+Enter again to commit. Aliases and templates are defined in the ledger as dated custom directives; manage them under /quick-profile.", "速记", "使用 Quick 标签页（快捷键 a q）以紧凑语法快速记录双过账交易。模板形式：'午餐 28 @微信'。显式形式：'28 CNY @微信 -> @餐饮 : 摘要'。按 Ctrl+Enter 预览，再按 Ctrl+Enter 确认提交。别名和模板在账本中定义为带日期的 custom 指令；可在 /quick-profile 管理。"},
	}
	result := make([]map[string]string, 0, len(sections))
	for _, section := range sections {
		title, body := section.enTitle, section.enBody
		if locale == repairguidance.LocaleChinese {
			title, body = section.zhTitle, section.zhBody
		}
		result = append(result, map[string]string{"id": section.id, "title": title, "body": body})
	}
	return result
}

func (s *Server) handleHelp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if topic := strings.TrimSpace(r.URL.Query().Get("topic")); topic != "" {
		code := strings.TrimPrefix(topic, "diagnostics/")
		guide, ok := repairguidance.Lookup(code, requestedLocale(r))
		if !ok || topic != guide.Topic {
			writeAPIError(w, http.StatusNotFound, helpTopicNotFoundMessage(requestedLocale(r)))
			return
		}
		writeJSON(w, guide)
		return
	}
	writeJSON(w, struct {
		Sections []map[string]string `json:"sections"`
	}{Sections: helpSections(requestedLocale(r))})
}
