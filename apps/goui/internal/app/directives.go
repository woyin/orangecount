// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
	"orangecount/apps/goui/internal/widgets"
	"orangecount/internal/desktop"
	"orangecount/internal/documents"
	"orangecount/internal/query"
)

type directiveKindOption struct {
	id, label string
}

var directiveKindOptions = []directiveKindOption{
	{label: "全部类型"},
	{id: "transaction", label: "交易"},
	{id: "open", label: "开户"},
	{id: "close", label: "销户"},
	{id: "commodity", label: "商品"},
	{id: "balance", label: "余额断言"},
	{id: "pad", label: "补齐"},
	{id: "event", label: "事件"},
	{id: "query", label: "查询"},
	{id: "price", label: "价格"},
	{id: "document", label: "附件"},
	{id: "note", label: "备注"},
	{id: "custom", label: "自定义指令"},
	{id: "option", label: "选项"},
	{id: "plugin", label: "插件"},
	{id: "include", label: "Include"},
	{id: "pushtag", label: "压入标签"},
	{id: "poptag", label: "弹出标签"},
	{id: "pushmeta", label: "压入元数据"},
	{id: "popmeta", label: "弹出元数据"},
}

func (m *Model) directivesPage(c *ui.Context) {
	t := c.Theme()
	response := m.workspaceState.Response
	if response == nil {
		ui.Text(c, m.tr("正在读取指令目录，配置记录按源码顺序展示且不受日期筛选。 ")).TextColor(t.TextMuted)
		return
	}
	if response.Directives == nil {
		ui.Text(c, m.tr("账本没有可用快照，指令目录暂不可用。修复账本后可重新读取。 ")).TextColor(t.TextMuted)
		return
	}
	if response.Directives.Mode != m.directiveMode || !m.directiveResponseMatchesFilters(response) {
		ui.Text(c, m.tr("正在读取指令目录，配置记录按源码顺序展示且不受日期筛选。 ")).TextColor(t.TextMuted)
		return
	}
	payload := response.Directives
	ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
		modeIndex := 0
		if m.directiveMode == "config" {
			modeIndex = 1
		}
		if ui.Segmented(c, &modeIndex, m.tr("所有指令"), m.tr("配置")).Label(m.tr("指令范围")).Changed() {
			if modeIndex == 1 {
				m.directiveMode = "config"
			} else {
				m.directiveMode = "all"
			}
			m.directiveOffset = 0
			m.directiveError = ""
			m.requestView("directives")
		}
		ui.Text(c, m.tr("类型")).TextColor(t.TextMuted)
		selectedKind := directiveKindLabel(m.directiveKind)
		if m.selectLocalized(c, &selectedKind, directiveKindLabels()).Changed() {
			m.directiveKind = directiveKindID(selectedKind)
			m.directiveOffset = 0
			m.requestView("directives")
		}
	})
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		search := ui.TextInput(c, &m.directiveSearchDraft).Placeholder(m.tr("搜索文字、账户或元数据")).Label(m.tr("搜索指令")).Grow(1)
		m.focusSearchInput(pageDirectives, search)
		if ui.PrimaryButton(c, m.tr("筛选")).Disabled(m.workspaceState.Busy).Clicked() {
			m.applyDirectiveSearch()
		}
		if m.directiveMode == "all" {
			orderIndex := 0
			if m.directiveDescending {
				orderIndex = 1
			}
			if ui.Segmented(c, &orderIndex, m.tr("正序"), m.tr("倒序")).Label(m.tr("排列顺序")).Changed() {
				m.directiveDescending = orderIndex == 1
				m.directiveOffset = 0
				m.requestView("directives")
			}
		}
	})
	if m.directiveError != "" {
		ui.Text(c, m.tr(m.directiveError)).TextColor(t.Danger)
	}
	if m.directiveMode == "config" {
		ui.Text(c, m.tr("配置记录按源码顺序读取，不应用日期、币种或估值筛选。 ")).TextColor(t.TextMuted)
	}
	if payload.Page.TotalCount == 0 {
		ui.Text(c, m.tr("没有匹配的指令。 ")).TextColor(t.TextMuted)
	} else {
		start := payload.Page.Offset + 1
		if len(payload.Records) == 0 {
			start = 0
		}
		ui.Text(c, fmt.Sprintf(m.tr("%d 条记录 · 第 %d–%d 条 · 快照 %s"), payload.Page.TotalCount, start, payload.Page.Offset+len(payload.Records), shortID(payload.SnapshotID))).FontSize(12).TextColor(t.TextMuted)
	}
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Gap(10).Children(func() {
			for _, record := range payload.Records {
				record := record
				m.directiveRecord(c, record)
			}
		})
	})
	ui.Row(c).Gap(10).Children(func() {
		if ui.Button(c, m.tr("上一页")).Disabled(payload.Page.Offset == 0 || m.workspaceState.Busy).Clicked() {
			m.directiveOffset = max(0, payload.Page.Offset-payload.Page.Limit)
			m.requestView("directives")
		}
		ui.Spacer(c)
		if ui.Button(c, m.tr("下一页")).Disabled(!payload.Page.HasMore || m.workspaceState.Busy).Clicked() {
			m.directiveOffset = payload.Page.Offset + payload.Page.Limit
			m.requestView("directives")
		}
	})
}

func (m *Model) directiveResponseMatchesFilters(response *desktop.ViewResponse) bool {
	if response == nil || response.Directives == nil {
		return false
	}
	sort := "ascending"
	if m.directiveMode == "all" && m.directiveDescending {
		sort = "descending"
	}
	request := response.Request
	return request.View == "journal" && request.JournalMode == m.directiveMode && request.DirectiveKind == m.directiveKind && request.Text == m.directiveSearch && request.Sort == sort && request.Offset == m.directiveOffset && request.Limit == 100
}

func (m *Model) directiveRecord(c *ui.Context, record desktop.DirectiveDTO) {
	t := c.Theme()
	ui.Column(c).Gap(7).Padding(12).Background(t.Surface).Border(1, t.Border).Radius(8).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			date := m.tr("配置")
			if record.Date != nil {
				date = *record.Date
			}
			ui.Text(c, date+" · "+m.tr(directiveTypeLabel(record.Kind))).FontSize(14).Bold()
			if record.Generated {
				ui.Text(c, m.tr("由 ")+m.tr(directiveTypeLabel(record.SourceKind))+m.tr(" 生成")).FontSize(11).TextColor(t.Warning)
			}
			ui.Spacer(c)
			if record.Source.Path != "" {
				location := directiveLocationLabel(record.Source)
				ui.Text(c, location).FontSize(11).TextColor(t.TextMuted).SingleLine()
				if ui.Button(c, m.tr("显示来源")).Clicked() {
					m.revealLocation(record.Source.Path)
				}
				m.sourceEditorButton(c, record.Source.Path)
			}
		})
		if record.OriginID != nil {
			ui.Text(c, m.tr("关联源指令 · ")+shortID(*record.OriginID)).FontSize(11).TextColor(t.TextMuted)
		}
		for _, field := range record.Fields {
			ui.Row(c).Gap(8).Children(func() {
				ui.Text(c, m.tr(directiveFieldLabel(field.Name))+m.tr("：")).TextColor(t.TextMuted)
				ui.Text(c, formatDirectiveValue(field.Value)).Selectable().Grow(1)
			})
		}
		for _, metadata := range record.Metadata {
			ui.Row(c).Gap(8).Children(func() {
				ui.Text(c, metadata.Key+"：").TextColor(t.TextMuted)
				ui.Text(c, formatDirectiveValue(metadata.Value)).Selectable().Grow(1)
				if metadata.Source.Path != "" {
					ui.Text(c, directiveLocationLabel(metadata.Source)).FontSize(11).TextColor(t.TextMuted).SingleLine()
				}
			})
		}
		if len(record.Tags) > 0 || len(record.Links) > 0 {
			labels := make([]string, 0, len(record.Tags)+len(record.Links))
			for _, tag := range record.Tags {
				labels = append(labels, "#"+tag)
			}
			for _, link := range record.Links {
				labels = append(labels, "^"+link)
			}
			ui.Text(c, strings.Join(labels, "   ")).TextColor(t.Accent)
		}
		for _, posting := range record.Postings {
			posting := posting
			ui.Column(c).Gap(4).Padding(8, 0, 0, 10).Border(1, t.Border).Radius(4).Children(func() {
				ui.Row(c).Gap(8).Children(func() {
					ui.Text(c, posting.Posting.Account).Grow(1)
					if posting.Posting.UnitsNumber != "" {
						ui.Text(c, posting.Posting.UnitsNumber+" "+posting.Posting.UnitsCurrency).TextColor(t.TextMuted)
					}
				})
				if posting.Posting.Cost != nil || posting.Posting.Price != nil {
					parts := []string{}
					if posting.Posting.Cost != nil {
						parts = append(parts, m.tr("成本 ")+posting.Posting.Cost.Display)
					}
					if posting.Posting.Price != nil {
						prefix := "@"
						if posting.Posting.Price.Total {
							prefix = "@@"
						}
						parts = append(parts, m.tr("价格 ")+prefix+posting.Posting.Price.Number.Exact+" "+posting.Posting.Price.Currency)
					}
					ui.Text(c, strings.Join(parts, " · ")).FontSize(12).TextColor(t.TextMuted)
				}
				if posting.Posting.Source.Path != "" {
					ui.Text(c, m.tr("过账来源 · ")+directiveLocationLabel(posting.Posting.Source)).FontSize(11).TextColor(t.TextMuted).SingleLine()
				}
				for _, metadata := range posting.Metadata {
					ui.Text(c, metadata.Key+"："+formatDirectiveValue(metadata.Value)+" · "+directiveLocationLabel(metadata.Source)).FontSize(11).TextColor(t.TextMuted)
				}
			})
		}
		for _, attachment := range record.Attachments {
			attachment := attachment
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, m.tr("附件 · ")+filepath.Base(attachment.Filename)).Grow(1)
				ui.Text(c, m.tr(attachmentStatusLabel(attachment.Status))).TextColor(t.TextMuted)
				canAct := attachment.Status == documents.Available && !m.workspaceState.Busy && !m.workspaceState.AttachmentRootsBusy && !m.rootPolicySyncBusy && m.pendingAttachmentAction == nil
				if ui.Button(c, m.tr("打开附件")).Disabled(!canAct).Clicked() {
					m.requestAttachmentAction(attachment, false)
				}
				if ui.Button(c, m.tr("显示附件")).Disabled(!canAct).Clicked() {
					m.requestAttachmentAction(attachment, true)
				}
			})
		}
	})
}

func directiveKindLabels() []string {
	labels := make([]string, len(directiveKindOptions))
	for i, option := range directiveKindOptions {
		labels[i] = option.label
	}
	return labels
}

func directiveKindLabel(id string) string {
	for _, option := range directiveKindOptions {
		if option.id == id {
			return option.label
		}
	}
	return directiveKindOptions[0].label
}

func directiveKindID(label string) string {
	for _, option := range directiveKindOptions {
		if option.label == label {
			return option.id
		}
	}
	return ""
}

func directiveTypeLabel(kind string) string {
	if kind == "dialect" {
		return "方言行"
	}
	for _, option := range directiveKindOptions[1:] {
		if option.id == kind {
			return option.label
		}
	}
	return kind
}

func directiveFieldLabel(name string) string {
	labels := map[string]string{
		"account": "账户", "amount": "金额", "booking": "记账法", "comment": "备注",
		"config": "配置", "currency": "币种", "custom_type": "指令类型", "event_type": "事件类型",
		"filename": "文件名", "flag": "标记", "key": "键", "module": "模块", "narration": "说明",
		"payee": "对象", "path": "路径", "query": "查询", "source_account": "来源账户",
		"tag": "标签", "tolerance": "容差", "value": "值",
	}
	if label := labels[name]; label != "" {
		return label
	}
	return name
}

func formatDirectiveValue(value desktop.DirectiveValue) string {
	return formatDirectiveValueDepth(value, 0)
}

func formatDirectiveValueDepth(value desktop.DirectiveValue, depth int) string {
	if depth > 64 {
		return "…"
	}
	switch value.Type {
	case "null":
		return "NULL"
	case "bool":
		if value.Value == true {
			return "TRUE"
		}
		return "FALSE"
	case "amount":
		if amount, ok := value.Value.(query.PresentedAmount); ok {
			return amount.Number.Exact + " " + amount.Currency
		}
	case "decimal":
		if decimal, ok := value.Value.(query.PresentedDecimal); ok {
			return queryDecimalText(decimal)
		}
		if text, ok := value.Value.(string); ok {
			return text
		}
	case "list":
		items, ok := value.Value.([]desktop.DirectiveValue)
		if !ok {
			break
		}
		parts := make([]string, len(items))
		for i, item := range items {
			parts[i] = formatDirectiveValueDepth(item, depth+1)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case "map":
		fields, ok := value.Value.([]desktop.DirectiveMetadata)
		if !ok {
			break
		}
		parts := make([]string, len(fields))
		for i, field := range fields {
			parts[i] = field.Key + ": " + formatDirectiveValueDepth(field.Value, depth+1)
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case "tag":
		if text, ok := value.Value.(string); ok {
			return "#" + text
		}
	case "link":
		if text, ok := value.Value.(string); ok {
			return "^" + text
		}
	case "string":
		if text, ok := value.Value.(string); ok {
			return fmt.Sprintf("%q", text)
		}
	}
	if text, ok := value.Value.(string); ok {
		return text
	}
	return fmt.Sprint(value.Value)
}

func directiveLocationLabel(location desktop.SourceLocation) string {
	return widgets.SourceText(widgets.SourceLocation{Path: location.Path, Line: location.Line, Column: location.Column})
}

func shortID(value string) string {
	if len(value) > 12 {
		return value[:12]
	}
	return value
}
