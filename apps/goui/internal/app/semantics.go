// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import (
	"fmt"
	"sort"

	"github.com/egoist/mygo/ui"
	"orangecount/apps/goui/internal/workspace"
	"orangecount/internal/desktop"
	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
)

func (m *Model) semanticsPage(c *ui.Context) {
	t := c.Theme()
	response := m.workspaceState.Response
	if response == nil || response.Migration == nil || !m.migrationResponseMatchesCurrent(response) {
		ui.Text(c, m.tr("正在生成当前账本快照的语义配置预览。输入发生变化时，旧预览不能用于应用配置。 ")).TextColor(t.TextMuted)
		return
	}

	comparison := response.Migration
	ui.Text(c, m.tr("预览只显示差异数量，不展开账户名称、金额或逐条内容。选择配置会基于此输入快照重开只读会话，不会修改账本文件。 ")).TextColor(t.TextMuted)
	ui.Row(c).Gap(12).Children(func() {
		m.semanticsProfileCard(c, "legacy", comparison.Legacy, comparison.ChangeCounts, comparison.InputIdentity)
		m.semanticsProfileCard(c, "v3", comparison.V3, comparison.ChangeCounts, comparison.InputIdentity)
	})
	ui.Text(c, m.tr("输入身份 · ")+shortID(comparison.InputIdentity)+m.tr("    当前快照 · ")+shortID(m.workspaceState.Ledger.SnapshotID)).FontSize(11).TextColor(t.TextMuted)
}

func (m *Model) semanticsProfileCard(c *ui.Context, profile string, summary snapshot.ProfileSummary, differences map[string]int, inputIdentity string) {
	t := c.Theme()
	selected := ledger.SemanticsProfile(profile)
	label := "Legacy"
	if selected == ledger.V3 {
		label = "V3"
	}
	current := m.workspaceState.Ledger.SemanticsProfile == selected
	ui.Column(c).Grow(1).Gap(8).Padding(16).Background(t.Surface).Border(1, t.Border).Radius(10).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, label).FontSize(17).Bold()
			if current {
				ui.Text(c, m.tr("当前配置")).FontSize(11).TextColor(t.Accent)
			}
		})
		ui.Text(c, fmt.Sprintf(m.tr("%d 条分录 · %d 条错误 · 有效：%t"), summary.EntryCount, summary.ErrorCount, summary.Valid)).FontSize(12)
		if len(summary.Extensions) > 0 {
			ui.Text(c, m.tr("启用扩展：")+joinSummary(summary.Extensions)).FontSize(11).TextColor(t.TextMuted)
		}
		ui.Text(c, m.tr("Legacy 与 V3 差异")).FontSize(12).Bold()
		keys := make([]string, 0, len(differences))
		for key, count := range differences {
			if count > 0 {
				keys = append(keys, key)
			}
		}
		sort.Strings(keys)
		if len(keys) == 0 {
			ui.Text(c, m.tr("没有检测到差异")).FontSize(11).TextColor(t.TextMuted)
		} else {
			for _, key := range keys {
				ui.Text(c, fmt.Sprintf("%s%s%d", m.tr(migrationDimensionLabel(key)), m.tr("："), differences[key])).FontSize(11).TextColor(t.TextMuted)
			}
		}
		if current {
			ui.Button(c, m.tr("已在使用")).Disabled(true)
		} else if ui.PrimaryButton(c, m.tr("应用 ")+label).Disabled(m.workspaceState.Busy || m.workspaceState.AttachmentRootsBusy || m.rootPolicySyncBusy).Clicked() {
			m.applySemanticsProfile(selected, inputIdentity)
		}
	})
}

func (m *Model) applySemanticsProfile(profile ledger.SemanticsProfile, inputIdentity string) {
	state := m.workspaceState
	if m.workspace == nil || state.ActivePath == "" || state.Busy || state.AttachmentRootsBusy || m.rootPolicySyncBusy {
		m.status = "当前工作区忙碌，暂时不能应用语义配置。"
		return
	}
	if state.Phase != workspace.PhaseReady && state.Phase != workspace.PhaseInvalid && state.Phase != workspace.PhaseStale {
		m.status = "当前账本状态不允许应用语义配置。"
		return
	}
	liveState := m.workspace.State()
	if inputIdentity == "" || state.Response == nil || state.Response.Migration == nil || state.Response.Migration.InputIdentity != inputIdentity ||
		liveState.ActivePath != state.ActivePath || liveState.Ledger.SnapshotID != state.Ledger.SnapshotID || !m.migrationResponseMatchesCurrent(state.Response) {
		m.status = "账本快照已变化，请重新读取预览后再应用。"
		return
	}
	if profile == state.Ledger.SemanticsProfile {
		return
	}
	m.openLedgerWithProfile(state.ActivePath, profile, inputIdentity)
}

func (m *Model) migrationResponseMatchesCurrent(response *desktop.ViewResponse) bool {
	if response == nil || response.Migration == nil {
		return false
	}
	identity := response.Migration.InputIdentity
	state := m.workspaceState
	currentSnapshotID := state.Ledger.SnapshotID
	if response.Request.View != "migration" || identity == "" ||
		response.Request.SnapshotID != currentSnapshotID || response.State.SnapshotID != currentSnapshotID {
		return false
	}
	if currentSnapshotID != "" {
		return true
	}
	// An initial evaluation error has no accepted snapshot ID. Migration
	// comparisons still use the latest parsed graph, so bind that preview to
	// the invalid ledger revision and entry path instead of leaving the page
	// in its loading state forever.
	return state.Phase == workspace.PhaseInvalid && response.State.Revision == state.Ledger.Revision &&
		response.State.EntryPath != "" && response.State.EntryPath == state.ActivePath
}

func migrationDimensionLabel(key string) string {
	switch key {
	case "validity":
		return "有效状态"
	case "accounts":
		return "账户定义"
	case "balances":
		return "余额"
	case "lots":
		return "持仓 lot"
	case "entries":
		return "指令/分录"
	case "diagnostics":
		return "诊断"
	default:
		return key
	}
}

func joinSummary(values []string) string {
	if len(values) == 0 {
		return "无"
	}
	return fmt.Sprintf("%v", values)
}
