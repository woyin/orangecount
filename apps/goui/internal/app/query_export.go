// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"orangecount/apps/goui/internal/localfiles"
	"orangecount/apps/goui/internal/workspace"
	"orangecount/internal/desktop"
)

type querySaveRequest struct {
	id           uint64
	epoch        uint64
	ledgerPath   string
	format       string
	context      desktop.QueryContext
	returnedRows int
	rowLimit     int
	truncated    bool
}

type reportSaveRequest struct {
	id         uint64
	epoch      uint64
	ledgerPath string
	format     string
	identity   desktop.ReportExportIdentity
}

func (m *Model) beginQuerySave(format string) {
	if m.closed.Load() || (m.window == nil && m.querySaveDialog == nil) || m.workspace == nil || m.offlineAnalysis != nil {
		return
	}
	response := m.workspaceState.QueryResponse
	if response == nil || response.Error != nil || response.Execution == nil || m.workspaceState.QueryBusy || m.workspaceState.Ledger.SnapshotID == "" {
		m.status = "只有当前快照上的成功查询可以保存。"
		return
	}
	if format != "csv" && format != "analysis" {
		m.status = "不支持的查询导出格式。"
		return
	}
	if m.pendingQuerySave != nil || m.pendingReportSave != nil || m.activeExportID != 0 || m.workspaceState.QueryCaptureBusy || m.workspaceState.ReportCaptureBusy {
		m.status = "已有查询保存流程正在进行。"
		return
	}
	if m.exportSaveSequence == ^uint64(0) {
		m.status = "查询保存操作编号已用尽。"
		return
	}
	m.exportSaveSequence++
	request := querySaveRequest{
		id: m.exportSaveSequence, epoch: m.workspaceState.Epoch,
		ledgerPath: m.workspaceState.ActivePath, format: format, context: response.Context,
		returnedRows: response.Execution.ReturnedRows, rowLimit: response.Execution.RowLimit,
		truncated: response.Execution.Truncated,
	}
	m.pendingQuerySave = &request
	m.status = "正在固定当前查询结果…"
	if err := m.workspace.Dispatch(workspace.Intent{Kind: workspace.IntentCaptureQuery, QueryExport: desktop.QueryExportRequest{
		SnapshotID: request.context.SnapshotID, RequestID: request.context.RequestID,
		Fingerprint: request.context.Fingerprint, Format: format,
	}}); err != nil {
		m.pendingQuerySave = nil
		m.status = "无法准备查询结果保存：" + err.Error()
	}
}

func (m *Model) finishPendingQuerySave(state workspace.State) {
	request := m.pendingQuerySave
	if request == nil {
		return
	}
	if state.Epoch != request.epoch {
		m.pendingQuerySave = nil
		m.status = "账本已切换，查询保存已取消。"
		return
	}
	if state.QueryCaptureBusy {
		return
	}
	if state.QueryCaptureError != "" {
		m.pendingQuerySave = nil
		m.status = "无法固定查询结果：" + state.QueryCaptureError
		return
	}
	if state.QueryCapture == nil {
		return
	}
	captured := state.QueryCapture
	if captured.Context != request.context || captured.Format != request.format {
		m.pendingQuerySave = nil
		m.status = "查询结果身份已变化，请重新执行后再保存。"
		return
	}
	localCapture, err := localfiles.NewCapture(captured.Content, captured.ProtectedPaths)
	if err != nil {
		m.pendingQuerySave = nil
		m.status = "无法固定文件保护信息：" + err.Error()
		return
	}
	m.pendingQuerySave = nil
	ctx, cancel := context.WithCancel(context.Background())
	m.activeExportID = request.id
	m.activeExportCancel = cancel
	m.status = "正在选择查询结果保存位置…"
	window := m.window
	go m.runQuerySave(ctx, cancel, *request, localCapture, window)
}

func (m *Model) beginReportSave(format string) {
	if m.closed.Load() || (m.window == nil && m.querySaveDialog == nil) || m.workspace == nil {
		return
	}
	response := m.currentReportResponse()
	if response == nil || response.ReportExport == nil || m.workspaceState.Busy || m.workspaceState.Ledger.Reloading {
		m.status = "只有当前快照上可导出的报表可以保存。"
		return
	}
	if format != "csv" && format != "analysis" {
		m.status = "不支持的报表导出格式。"
		return
	}
	if m.pendingQuerySave != nil || m.pendingReportSave != nil || m.activeExportID != 0 || m.workspaceState.QueryCaptureBusy || m.workspaceState.ReportCaptureBusy {
		m.status = "已有导出流程正在进行。"
		return
	}
	identity := *response.ReportExport
	if identity.SnapshotID == "" || identity.SnapshotID != m.workspaceState.Ledger.SnapshotID || identity.ProjectionID == "" || identity.CaptureScope == "" {
		m.status = "报表导出身份已过期，请重新读取页面。"
		return
	}
	if m.exportSaveSequence == ^uint64(0) {
		m.status = "报表保存操作编号已用尽。"
		return
	}
	m.exportSaveSequence++
	request := reportSaveRequest{
		id: m.exportSaveSequence, epoch: m.workspaceState.Epoch,
		ledgerPath: m.workspaceState.ActivePath, format: format, identity: identity,
	}
	m.pendingReportSave = &request
	m.status = "正在固定当前报表投影…"
	if err := m.workspace.Dispatch(workspace.Intent{Kind: workspace.IntentCaptureReport, Report: desktop.ReportExportRequest{
		ProjectionID: identity.ProjectionID, SnapshotID: identity.SnapshotID,
		Format: format, CaptureScope: identity.CaptureScope,
	}}); err != nil {
		m.pendingReportSave = nil
		m.status = "无法准备报表保存：" + err.Error()
	}
}

func (m *Model) currentReportResponse() *desktop.ViewResponse {
	var response *desktop.ViewResponse
	switch m.page {
	case pageDashboard:
		response = m.responseFor("dashboard")
		if response != nil && response.Trends == nil {
			return nil
		}
	case pageBalance:
		response = m.responseFor("balance_sheet")
		if response != nil && response.Tree == nil {
			return nil
		}
	case pageIncome:
		response = m.responseFor("income_statement")
		if response != nil && response.Tree == nil {
			return nil
		}
	case pageJournal:
		response = m.workspaceState.Response
		if response == nil || !m.journalResponseMatchesFilters(response) {
			return nil
		}
	case pageAccount:
		response = m.workspaceState.Response
		if response == nil || !m.accountResponseMatchesFilters(response) {
			return nil
		}
	case pagePrices:
		response = m.workspaceState.Response
		if response == nil || !m.priceResponseMatchesFilters(response) {
			return nil
		}
	case pageHoldings:
		response = m.workspaceState.Response
		if response == nil || !m.holdingResponseMatchesFilters(response) {
			return nil
		}
	default:
		return nil
	}
	if response == nil || response.ReportExport == nil || response.State.SnapshotID == "" ||
		response.State.SnapshotID != m.workspaceState.Ledger.SnapshotID ||
		response.ReportExport.SnapshotID != m.workspaceState.Ledger.SnapshotID {
		return nil
	}
	return response
}

func (m *Model) reportExportControls(c *ui.Context) {
	response := m.currentReportResponse()
	if response == nil {
		return
	}
	identity := response.ReportExport
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, fmt.Sprintf(m.tr("可保存：%d 行 · %s"), identity.ReturnedRows, identity.CaptureScope)).TextColor(c.Theme().TextMuted)
		if ui.Button(c, m.tr("保存报表 CSV…")).Disabled(m.workspaceState.Busy || m.workspaceState.Ledger.Reloading || m.workspaceState.ReportCaptureBusy || m.workspaceState.QueryCaptureBusy || m.activeExportID != 0).Clicked() {
			m.beginReportSave("csv")
		}
		if ui.Button(c, m.tr("保存报表分析文件…")).Disabled(m.workspaceState.Busy || m.workspaceState.Ledger.Reloading || m.workspaceState.ReportCaptureBusy || m.workspaceState.QueryCaptureBusy || m.activeExportID != 0).Clicked() {
			m.beginReportSave("analysis")
		}
		if m.pendingReportSave != nil && m.workspaceState.ReportCaptureBusy {
			ui.Text(c, m.tr("正在固定报表…")).TextColor(c.Theme().TextMuted)
			if ui.Button(c, m.tr("取消准备")).Clicked() {
				m.cancelPendingExport()
				m.status = "报表保存已取消。"
			}
		}
		if m.activeExportID != 0 && ui.Button(c, m.tr("取消导出")).Clicked() {
			m.cancelPendingExport()
			m.status = "导出已取消。"
		}
	})
}

func (m *Model) finishPendingReportSave(state workspace.State) {
	request := m.pendingReportSave
	if request == nil {
		return
	}
	if state.Epoch != request.epoch {
		m.pendingReportSave = nil
		m.status = "账本已切换，报表保存已取消。"
		return
	}
	if state.ReportCaptureBusy {
		return
	}
	if state.ReportCaptureError != "" {
		m.pendingReportSave = nil
		m.status = "无法固定报表投影：" + state.ReportCaptureError
		return
	}
	if state.ReportCapture == nil {
		return
	}
	captured := state.ReportCapture
	if captured.Identity != request.identity || captured.Format != request.format {
		m.pendingReportSave = nil
		m.status = "报表投影身份已变化，请重新读取后再保存。"
		return
	}
	localCapture, err := localfiles.NewCapture(captured.Content, captured.ProtectedPaths)
	if err != nil {
		m.pendingReportSave = nil
		m.status = "无法固定报表文件保护信息：" + err.Error()
		return
	}
	m.pendingReportSave = nil
	ctx, cancel := context.WithCancel(context.Background())
	m.activeExportID = request.id
	m.activeExportCancel = cancel
	m.status = "正在选择报表保存位置…"
	window := m.window
	go m.runReportSave(ctx, cancel, *request, localCapture, window)
}

func (m *Model) runQuerySave(ctx context.Context, cancel context.CancelFunc, request querySaveRequest, captured *localfiles.Capture, window *mygo.Window) {
	dialog := m.querySaveDialog
	if dialog == nil {
		dialog = mygo.Dialog.SaveWithContext
	}
	path, err := dialog(ctx, mygo.SaveDialogOptions{
		Parent: window, Title: m.tr("保存查询结果"), ButtonLabel: m.tr("保存"),
		DefaultPath: queryExportDefaultPath(request.ledgerPath, request.format),
		Filters:     []mygo.FileFilter{{Name: m.tr(queryExportFormatName(request.format)), Extensions: []string{queryExportExtension(request.format)}}},
		Message:     queryExportPanelMessage(request, m.tr),
	})
	if err != nil {
		m.finishQuerySaveResult(request.id, cancel, "", err)
		return
	}
	if path == "" {
		m.finishQuerySaveResult(request.id, cancel, "已取消查询结果保存。", nil)
		return
	}
	if ctx.Err() != nil || m.closed.Load() {
		m.finishQuerySaveResult(request.id, cancel, "", context.Canceled)
		return
	}
	path, err = m.saveCapturedExport(ctx, request.epoch, captured, path, queryExportExtension(request.format))
	if err != nil {
		m.finishQuerySaveResult(request.id, cancel, "", err)
		return
	}
	message := "查询结果已保存：" + filepath.Base(path)
	if ctx.Err() != nil {
		message += "（取消发生在文件提交之后）"
	}
	m.finishQuerySaveResult(request.id, cancel, message, nil)
}

func (m *Model) runReportSave(ctx context.Context, cancel context.CancelFunc, request reportSaveRequest, captured *localfiles.Capture, window *mygo.Window) {
	dialog := m.querySaveDialog
	if dialog == nil {
		dialog = mygo.Dialog.SaveWithContext
	}
	extension := reportExportExtension(request.format)
	path, err := dialog(ctx, mygo.SaveDialogOptions{
		Parent: window, Title: m.tr("保存报表"), ButtonLabel: m.tr("保存"),
		DefaultPath: reportExportDefaultPath(request.ledgerPath, request.identity),
		Filters:     []mygo.FileFilter{{Name: m.tr(reportExportFormatName(request.format)), Extensions: []string{extension}}},
		Message:     reportExportPanelMessage(request.identity, request.format, m.tr),
	})
	if err != nil {
		m.finishExportResult(request.id, cancel, "报表保存", "", err)
		return
	}
	if path == "" {
		m.finishExportResult(request.id, cancel, "报表保存", "已取消报表保存。", nil)
		return
	}
	if ctx.Err() != nil || m.closed.Load() {
		m.finishExportResult(request.id, cancel, "报表保存", "", context.Canceled)
		return
	}
	path, err = m.saveCapturedExport(ctx, request.epoch, captured, path, extension)
	if err != nil {
		m.finishExportResult(request.id, cancel, "报表保存", "", err)
		return
	}
	message := "报表已保存：" + filepath.Base(path)
	if ctx.Err() != nil {
		message += "（取消发生在文件提交之后）"
	}
	m.finishExportResult(request.id, cancel, "报表保存", message, nil)
}

func (m *Model) saveCapturedExport(ctx context.Context, epoch uint64, captured *localfiles.Capture, path, extension string) (string, error) {
	controller := m.workspace
	if controller == nil {
		return "", workspace.ErrNoSession
	}
	protection, err := controller.RefreshExportProtection(epoch)
	if err != nil {
		return "", err
	}
	refreshed, err := captured.WithProtectedPaths(protection.ProtectedPaths)
	if err != nil {
		return "", err
	}
	if ctx.Err() != nil {
		return "", context.Canceled
	}
	path = addExpectedExtension(path, extension)
	if err := refreshed.Save(ctx, path); err != nil {
		return "", err
	}
	return path, nil
}

func (m *Model) finishQuerySaveResult(id uint64, cancel context.CancelFunc, success string, err error) {
	m.finishExportResult(id, cancel, "查询结果保存", success, err)
}

func (m *Model) finishExportResult(id uint64, cancel context.CancelFunc, action, success string, err error) {
	if m.onExportSaveResult != nil {
		m.onExportSaveResult(success, err)
	}
	window := m.window
	if window == nil {
		cancel()
		return
	}
	window.Update(func() {
		if m.closed.Load() {
			cancel()
			return
		}
		current := m.activeExportID == id
		if current {
			m.activeExportID = 0
			m.activeExportCancel = nil
		}
		cancel()
		if err != nil {
			if current {
				if errors.Is(err, context.Canceled) {
					m.status = action + "已取消。"
				} else {
					m.status = action + "失败：" + err.Error()
				}
			}
			return
		}
		if success != "" {
			m.status = success
		}
	})
}

func (m *Model) cancelPendingExport() {
	if m.pendingQuerySave != nil {
		m.pendingQuerySave = nil
		if m.workspace != nil && m.workspace.State().QueryCaptureBusy {
			_ = m.workspace.Dispatch(workspace.Intent{Kind: workspace.IntentCancelQueryCapture})
		}
	}
	if m.pendingReportSave != nil {
		m.pendingReportSave = nil
		if m.workspace != nil && m.workspace.State().ReportCaptureBusy {
			_ = m.workspace.Dispatch(workspace.Intent{Kind: workspace.IntentCancelReportCapture})
		}
	}
	if m.activeExportCancel != nil {
		m.activeExportCancel()
		m.activeExportCancel = nil
		m.activeExportID = 0
	}
}

func reportExportDefaultPath(ledgerPath string, identity desktop.ReportExportIdentity) string {
	base := strings.TrimSuffix(filepath.Base(ledgerPath), filepath.Ext(ledgerPath))
	if base == "" || base == "." {
		base = "orangecount"
	}
	section := strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(identity.Section)
	name := base + "-" + section
	if identity.PriceBase != "" && identity.PriceQuote != "" {
		pair := strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(identity.PriceBase + "-" + identity.PriceQuote)
		name += "-" + pair
	}
	if identity.DateFrom != "" || identity.DateTo != "" {
		name += "-" + identity.DateFrom + "-" + identity.DateTo
	}
	return filepath.Join(filepath.Dir(ledgerPath), name+"."+reportExportExtension("csv"))
}

func reportExportPanelMessage(identity desktop.ReportExportIdentity, format string, translate func(string) string) string {
	formatMessage := "CSV 保存当前报表投影"
	if format == "analysis" {
		formatMessage = "分析文件保存报表 schema、上下文与 typed cells"
	}
	completeness := translate(identity.Completeness)
	if completeness == "" {
		completeness = translate("完整性未声明")
	}
	scope := identity.CaptureScope
	if scope == "complete-projection" {
		scope = translate("完整投影")
	} else if scope == "current-page" {
		scope = translate("当前页")
	} else {
		scope = translate(scope)
	}
	dateRange := translate("全部日期")
	if identity.DateFrom != "" || identity.DateTo != "" {
		dateRange = identity.DateFrom + translate(" 至 ") + identity.DateTo
	}
	truncated := ""
	if identity.Truncated {
		truncated = " · " + translate("结果达到数量上限")
	}
	basis := ""
	if identity.Currency != "" || identity.Valuation != "" {
		currency := identity.Currency
		if currency == "" {
			currency = translate("原币")
		}
		basis = " · " + fmt.Sprintf(translate("报表口径 %s / %s"), currency, reportValuationLabel(identity.Valuation, translate))
	}
	pricePair := ""
	if identity.PriceBase != "" && identity.PriceQuote != "" {
		pricePair = " · " + fmt.Sprintf(translate("报价对 %s/%s"), identity.PriceBase, identity.PriceQuote)
	}
	return fmt.Sprintf(translate("%s · %s · %d 行 / %d 条源记录%s · %s · %s · profile %s%s%s"), translate(formatMessage), scope, identity.ReturnedRows, identity.SourceRecordCount, truncated, dateRange, completeness, identity.SemanticsProfile, basis, pricePair)
}

func reportValuationLabel(valuation string, translate func(string) string) string {
	switch valuation {
	case "at-cost":
		return translate("按成本")
	case "market-value":
		return translate("按市价")
	case "units":
		return translate("按数量")
	case "native":
		return translate("原币")
	default:
		return translate("估值方式未指定")
	}
}

func reportExportExtension(format string) string {
	if format == "analysis" {
		return "ocanalysis"
	}
	return "csv"
}

func reportExportFormatName(format string) string {
	if format == "analysis" {
		return "OrangeCount 报表分析文件"
	}
	return "CSV 文件"
}

// CancelPendingExport is called before the native parent window closes so an
// attached Save Panel and any pre-commit file write can be cancelled in time.
func (m *Model) CancelPendingExport() {
	m.cancelPendingExport()
	if m.preferenceBackupCancel != nil {
		m.preferenceBackupCancel()
	}
}

func queryExportDefaultPath(ledgerPath, format string) string {
	base := strings.TrimSuffix(filepath.Base(ledgerPath), filepath.Ext(ledgerPath))
	if base == "" || base == "." {
		base = "orangecount"
	}
	return filepath.Join(filepath.Dir(ledgerPath), base+"-query."+queryExportExtension(format))
}

func queryExportPanelMessage(request querySaveRequest, translate func(string) string) string {
	formatMessage := "CSV 保存查询结果行"
	if request.format == "analysis" {
		formatMessage = "分析文件保存查询上下文、schema 与 typed cells"
	}
	completeness := fmt.Sprintf(translate("%d/%d 行"), request.returnedRows, request.rowLimit)
	if request.truncated {
		completeness += " · " + translate("结果达到行数上限")
	}
	return fmt.Sprintf(translate("%s · %s · snapshot %s · profile %s"), translate(formatMessage), completeness, request.context.SnapshotID, request.context.SemanticsProfile)
}

func queryExportExtension(format string) string {
	if format == "analysis" {
		return "ocanalysis"
	}
	return "csv"
}

func queryExportFormatName(format string) string {
	if format == "analysis" {
		return "OrangeCount 查询分析文件"
	}
	return "CSV 文件"
}

func addExpectedExtension(path, extension string) string {
	if filepath.Ext(path) == "" {
		return path + "." + extension
	}
	return path
}
