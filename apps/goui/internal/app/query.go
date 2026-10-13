// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"orangecount/apps/goui/internal/querypresets"
	"orangecount/apps/goui/internal/widgets/queryeditor"
	"orangecount/apps/goui/internal/workspace"
	"orangecount/internal/desktop"
	"orangecount/internal/query"
	"orangecount/internal/report"
)

func (m *Model) queryPage(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Gap(12).Children(func() {
			if m.offlineAnalysisLoading {
				ui.Text(c, m.tr("正在读取离线分析文件…")).TextColor(t.TextMuted)
			}
			if m.offlineAnalysisError != "" {
				ui.Text(c, m.tr(m.offlineAnalysisError)).TextColor(t.Danger)
			}
			if catalogue := m.queryCatalogueResponse(); catalogue != nil {
				m.queryCataloguePage(c, catalogue)
			} else if m.workspaceState.Response != nil && m.workspaceState.Response.Request.View == "query" {
				ui.Text(c, m.tr("查询目录正在读取或已过期；旧查询不会用于当前账本。 ")).TextColor(t.TextMuted)
			}
			ui.Text(c, m.tr("在当前账本快照上显式执行查询。草稿不会自动运行。")).TextColor(t.TextMuted)
			editor := ui.TextArea(c, &m.queryDraft).Label(m.tr("BQL 查询")).MinHeight(220).FillWidth()
			if editor.Shortcut(ui.Cmd, ui.KeyEnter) {
				m.executeQuery(editor.Composing())
			}
			m.queryPresetsPage(c)
			ui.Row(c).Gap(8).Children(func() {
				if ui.PrimaryButton(c, m.tr("执行查询")).Clicked() {
					m.executeQuery(editor.Composing())
				}
				if m.workspaceState.QueryBusy {
					progress := "正在执行…"
					if m.workspaceState.QueryCancelling {
						progress = "正在取消…"
					}
					ui.Text(c, m.tr(progress)).TextColor(t.TextMuted)
					if ui.Button(c, m.tr("取消查询")).Disabled(m.workspaceState.QueryCancelling).Clicked() && m.workspace != nil {
						if err := m.workspace.Dispatch(workspace.Intent{Kind: workspace.IntentCancelQuery}); err != nil {
							m.status = "无法取消查询：" + err.Error()
						}
					}
				}
			})
			ui.Row(c).Gap(8).Children(func() {
				if ui.Button(c, m.tr("打开离线分析文件…")).Disabled(m.picking || m.closed.Load()).Clicked() {
					m.chooseQueryAnalysis()
				}
				if m.offlineAnalysis != nil && ui.Button(c, m.tr("关闭离线捕获")).Clicked() {
					m.offlineAnalysis = nil
					m.offlineAnalysisPath = ""
					m.offlineAnalysisError = ""
					m.querySelection = -1
					m.querySelectionContext = ""
				}
			})
			if response := m.displayedQueryResponse(); response != nil {
				offline := m.offlineAnalysis != nil
				ui.Divider(c)
				if offline {
					capture := m.offlineAnalysis.Context
					ui.Text(c, m.tr("离线分析捕获 · 不读取活动账本，也不会重新执行原查询")).Bold().TextColor(t.TextMuted)
					ui.Text(c, fmt.Sprintf(m.tr("文件 %s · %s · format %d · %s"), filepath.Base(m.offlineAnalysisPath), capture.Producer, capture.FormatVersion, capture.Contract)).SingleLine().FontSize(12).TextColor(t.TextMuted)
					ui.Row(c).Gap(8).Children(func() {
						ui.Text(c, m.tr("捕获查询")).Bold()
						ui.Text(c, response.Context.Text).SingleLine().TextColor(t.TextMuted)
						ui.Spacer(c)
						if response.Context.Text != m.queryDraft && ui.Button(c, m.tr("填入捕获查询草稿")).Clicked() {
							m.queryDraft = response.Context.Text
							m.status = "已填入捕获中的查询文本；尚未执行。"
						}
					})
					completeness := "完整捕获"
					if capture.Truncated {
						completeness = "已截断捕获"
					}
					ui.Text(c, fmt.Sprintf("%s · capture scope %s · row limit %d", completeness, capture.CaptureScope, capture.RowLimit)).FontSize(12).TextColor(t.TextMuted)
				} else {
					ui.Row(c).Gap(8).Children(func() {
						ui.Text(c, m.tr("执行文本")).Bold()
						ui.Text(c, response.Context.Text).SingleLine().TextColor(t.TextMuted)
						ui.Spacer(c)
						if response.Context.Text != m.queryDraft && ui.Button(c, m.tr("恢复执行文本")).Clicked() {
							m.queryDraft = response.Context.Text
						}
					})
				}
				ui.Text(c, fmt.Sprintf(m.tr("快照 %s · profile %s · evaluation %s"), response.Context.SnapshotID, response.Context.SemanticsProfile, response.Context.EvaluationRevision)).SingleLine().FontSize(12).TextColor(t.TextMuted)
				ui.Text(c, m.tr("查询指纹 ")+response.Context.Fingerprint).SingleLine().FontSize(12).TextColor(t.TextMuted)
				if response.Error != nil {
					ui.Text(c, response.Error.Code+" · "+response.Error.Message).TextColor(t.Danger)
					if response.Error.Range != nil {
						ui.Text(c, fmt.Sprintf(m.tr("UTF-8 字节范围 [%d,%d)"), response.Error.Range.Start, response.Error.Range.End)).FontSize(12).TextColor(t.TextMuted)
						if response.Context.Text == m.queryDraft && ui.Button(c, m.tr("定位到查询错误")).Clicked() {
							m.locateQueryError(editor, response)
						}
					}
					return
				}
				if response.Execution == nil {
					ui.Text(c, m.tr("查询没有返回可显示的结果")).TextColor(t.TextMuted)
					return
				}
				execution := response.Execution
				if offline {
					ui.Text(c, fmt.Sprintf(m.tr("%d/%d 行 · %d/%d UTF-8 字节 · 原始工作步数未存入捕获%s"), execution.ReturnedRows, execution.RowLimit, len(response.Context.Text), query.QueryTextLimit, m.tr(truncationLabel(execution.Truncated)))).TextColor(t.TextMuted)
				} else {
					ui.Text(c, fmt.Sprintf(m.tr("%d/%d 行 · %d/%d UTF-8 字节 · %d/%d 工作步数%s"), execution.ReturnedRows, execution.RowLimit, len(response.Context.Text), query.QueryTextLimit, querySteps(execution.PhaseSteps), query.QueryStepLimit, m.tr(truncationLabel(execution.Truncated)))).TextColor(t.TextMuted)
					ui.Row(c).Gap(8).Children(func() {
						if ui.Button(c, m.tr("保存 CSV…")).Disabled(m.workspaceState.Busy || m.workspaceState.QueryBusy || m.workspaceState.QueryCaptureBusy || m.activeExportID != 0).Clicked() {
							m.beginQuerySave("csv")
						}
						if ui.Button(c, m.tr("保存分析文件…")).Disabled(m.workspaceState.Busy || m.workspaceState.QueryBusy || m.workspaceState.QueryCaptureBusy || m.activeExportID != 0).Clicked() {
							m.beginQuerySave("analysis")
						}
						if m.pendingQuerySave != nil && m.workspaceState.QueryCaptureBusy {
							ui.Text(c, m.tr("正在固定结果…")).TextColor(t.TextMuted)
							if ui.Button(c, m.tr("取消准备")).Clicked() {
								m.cancelPendingExport()
								m.status = "查询结果保存已取消。"
							}
						}
						if m.activeExportID != 0 && ui.Button(c, m.tr("取消保存")).Clicked() {
							m.cancelPendingExport()
							m.status = "查询结果保存已取消。"
						}
					})
					if m.workspaceState.QueryCaptureError != "" {
						ui.Text(c, m.tr("查询结果捕获失败 · ")+m.workspaceState.QueryCaptureError).TextColor(t.Danger)
					}
				}
				columns := make([]ui.TableColumn, 0, len(execution.Result.Columns))
				for _, column := range execution.Result.Columns {
					columns = append(columns, ui.TableColumn{Title: column.Name, ID: column.ID, Width: 180, Sortable: queryCellTypeSortable(column.Type)})
				}
				schema, _ := json.Marshal(execution.Result.Columns)
				if key := string(schema); m.querySchemaKey != key {
					m.querySchemaKey = key
					m.restoreQueryColumnLayout(key)
					m.queryTable.Sort = nil
					m.querySort = ui.SortOrder{}
					m.queryAppliedSort = ui.SortOrder{}
				}
				rows := execution.Result.Rows
				selectionContext := queryResultSelectionContext(response.Context, m.offlineAnalysisPath)
				if m.querySelectionContext != selectionContext {
					m.querySelectionContext = selectionContext
					m.querySelection = -1
				}
				m.queryTable.Sort = &m.querySort
				m.queryTable.Selected = &m.querySelection
				displayRows := queryRowsInDisplayOrder(execution.Result, m.querySort)
				m.queryTable.Key = func(row int) any { return fmt.Sprintf("%s:%d", selectionContext, displayRows[row]) }
				m.queryTable.Label = func(row int) string { return fmt.Sprintf(m.tr("查询结果第 %d 行"), row+1) }
				selectedRow := -1
				if m.querySelection >= 0 && m.querySelection < len(displayRows) {
					selectedRow = displayRows[m.querySelection]
				}
				ui.Table(c, &m.queryTable, columns, len(rows), func(row, col int) {
					ui.Text(c, queryCellTextWithLocale(rows[displayRows[row]][col], m.tr)).SingleLine()
				}).Height(320)
				if m.querySort != m.queryAppliedSort {
					displayRows = queryRowsInDisplayOrder(execution.Result, m.querySort)
					if selectedRow >= 0 {
						m.querySelection = slices.Index(displayRows, selectedRow)
					}
					m.queryAppliedSort = m.querySort
				}
				if m.querySelection >= 0 && m.querySelection < len(displayRows) {
					selectedRow = displayRows[m.querySelection]
					cells := rows[selectedRow]
					ui.Column(c).Padding(12).Gap(8).Background(t.Surface).Border(1, t.Border).Radius(8).Children(func() {
						ui.Row(c).Gap(8).Children(func() {
							ui.Text(c, fmt.Sprintf(m.tr("所选结果行 · %d"), m.querySelection+1)).Bold().Grow(1)
							if ui.Button(c, m.tr("复制类型化结果行")).Disabled(m.writeClipboard == nil).Clicked() {
								m.copyQueryResultRow(response, selectedRow)
							}
						})
						for index, cell := range cells {
							column := execution.Result.Columns[index]
							ui.Row(c).Gap(8).Children(func() {
								ui.Text(c, column.Name+" · "+column.Type).FontSize(12).TextColor(t.TextMuted)
								ui.Column(c).Gap(4).Grow(1).Children(func() {
									for _, line := range queryCellDetailsWithLocale(cell, m.tr) {
										ui.Text(c, line).Selectable().Grow(1)
									}
								})
							})
						}
					})
				}
			} else if m.workspaceState.QueryError != "" {
				ui.Text(c, m.tr("查询执行失败 · ")+m.workspaceState.QueryError).TextColor(t.Danger)
			}
		})
	})
}

func (m *Model) locateQueryError(editor *ui.Element, response *desktop.QueryResponse) {
	if editor == nil || response == nil || response.Error == nil || response.Error.Range == nil || response.Context.Text != m.queryDraft {
		return
	}
	if !queryeditor.SelectByteRange(editor, response.Context.Text, response.Error.Range.Start, response.Error.Range.End) {
		m.status = "当前输入法组合尚未结束，错误定位已保留"
	}
}

func (m *Model) queryPresetsPage(c *ui.Context) {
	t := c.Theme()
	path := m.workspaceState.ActivePath
	loaded := path != "" && path == m.queryPresetLedger && !m.queryPresetLoading
	ui.Column(c).Padding(12).Gap(8).Background(t.Surface).Border(1, t.Border).Radius(8).Children(func() {
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, fmt.Sprintf(m.tr("我的查询预设 · %d 条"), len(m.queryPresets))).Bold().Grow(1)
			ui.Text(c, m.tr("仅保存你明确命名的草稿；选择只填入编辑器")).FontSize(12).TextColor(t.TextMuted)
		})
		if m.queryPresetStore == nil {
			ui.Text(c, m.tr("此窗口未配置预设存储。 ")).TextColor(t.TextMuted)
		} else if path == "" {
			ui.Text(c, m.tr("打开账本后可保存和读取该账本的预设。 ")).TextColor(t.TextMuted)
		} else if m.queryPresetLoading || !loaded {
			ui.Text(c, m.tr("正在读取此账本的预设…")).TextColor(t.TextMuted)
		} else if len(m.queryPresets) == 0 {
			ui.Text(c, m.tr("此账本还没有自定义预设。 ")).TextColor(t.TextMuted)
		} else {
			presets := m.queryPresets
			m.queryPresetList.Key = func(row int) any { return presets[row].ID }
			m.queryPresetList.Label = func(row int) string { return presets[row].Name }
			ui.List(c, &m.queryPresetList, len(presets), func(row int) {
				preset := presets[row]
				ui.Row(c).Gap(8).Padding(4, 8).AlignItems(ui.Center).Children(func() {
					if ui.Button(c, m.tr("填入草稿 · ")+preset.Name).Disabled(!loaded).Clicked() {
						m.fillSavedPreset(preset.ID)
					}
					ui.Text(c, preset.Text).SingleLine().FontSize(11).TextColor(t.TextMuted).Grow(1)
					if ui.Button(c, m.tr("删除 · ")+preset.Name).Disabled(m.queryPresetBusy || !loaded).Clicked() {
						m.deleteQueryPreset(preset.ID)
					}
				})
			}).Height(140)
		}
		if m.queryPresetError != "" {
			ui.Text(c, m.tr("预设存储错误 · ")+m.queryPresetError).TextColor(t.Danger)
		}
		ui.Row(c).Gap(8).Children(func() {
			ui.TextInput(c, &m.queryPresetName).Label(m.tr("预设名称")).Placeholder(m.tr("例如：季度现金流")).Grow(1)
			if ui.Button(c, m.tr("保存当前草稿为预设")).Disabled(m.queryPresetStore == nil || !loaded || m.queryPresetBusy || strings.TrimSpace(m.queryDraft) == "").Clicked() {
				m.saveQueryPreset()
			}
			if m.queryPresetBusy {
				ui.Text(c, m.tr("正在保存…")).TextColor(t.TextMuted)
			}
		})
	})
}

func (m *Model) fillSavedPreset(id string) {
	if m.queryPresetLedger == "" || m.queryPresetLedger != m.workspaceState.ActivePath || m.queryPresetLoading {
		m.status = "此账本的预设尚未就绪。"
		return
	}
	for _, preset := range m.queryPresets {
		if preset.ID == id {
			m.fillQueryDraft(preset.Text, m.workspaceState.Ledger.SnapshotID)
			return
		}
	}
	m.status = "预设已变化；请重新读取后再选择。"
}

func (m *Model) loadQueryPresets(path string, generation uint64) {
	if m.queryPresetStore == nil || path == "" || m.window == nil || m.closed.Load() {
		return
	}
	store, window := m.queryPresetStore, m.window
	m.queryPresetLoading = true
	m.preferenceWork.Add(1)
	go func() {
		defer m.preferenceWork.Done()
		presets, err := store.LoadLedger(path)
		window.Update(func() {
			if m.closed.Load() || generation != m.queryPresetGeneration || m.queryPresetLedger != path || m.workspaceState.ActivePath != path {
				return
			}
			m.queryPresetLoading = false
			if err != nil {
				m.queryPresetError = queryPresetErrorMessage(err)
				return
			}
			m.queryPresetError = ""
			m.queryPresets = presets
		})
	}()
}

func (m *Model) saveQueryPreset() {
	if m.queryPresetStore == nil || m.queryPresetBusy || m.queryPresetLoading || m.window == nil || m.closed.Load() {
		return
	}
	path := m.workspaceState.ActivePath
	if path == "" || path != m.queryPresetLedger {
		m.queryPresetError = "当前账本预设尚未就绪。"
		return
	}
	name, text := strings.TrimSpace(m.queryPresetName), m.queryDraft
	if name == "" || strings.TrimSpace(text) == "" {
		m.queryPresetError = "请输入预设名称和查询草稿。"
		return
	}
	store, window, generation := m.queryPresetStore, m.window, m.queryPresetGeneration
	m.queryPresetBusy = true
	m.queryPresetError = ""
	m.preferenceWork.Add(1)
	go func() {
		defer m.preferenceWork.Done()
		presets, err := store.Save(path, name, text)
		window.Update(func() {
			if m.closed.Load() || generation != m.queryPresetGeneration || m.queryPresetLedger != path || m.workspaceState.ActivePath != path {
				return
			}
			m.queryPresetBusy = false
			if err != nil {
				m.queryPresetError = queryPresetErrorMessage(err)
				return
			}
			m.queryPresetName = ""
			m.queryPresets = presets
			m.queryPresetError = ""
			m.status = "已保存自定义预设；尚未执行。"
		})
	}()
}

func (m *Model) deleteQueryPreset(id string) {
	if m.queryPresetStore == nil || m.queryPresetBusy || m.queryPresetLoading || m.window == nil || m.closed.Load() {
		return
	}
	path := m.workspaceState.ActivePath
	if path == "" || path != m.queryPresetLedger {
		m.queryPresetError = "当前账本预设尚未就绪。"
		return
	}
	store, window, generation := m.queryPresetStore, m.window, m.queryPresetGeneration
	m.queryPresetBusy = true
	m.queryPresetError = ""
	name := ""
	for _, preset := range m.queryPresets {
		if preset.ID == id {
			name = preset.Name
			break
		}
	}
	m.preferenceWork.Add(1)
	go func() {
		defer m.preferenceWork.Done()
		confirmation, confirmErr := mygo.Dialog.Message(mygo.MessageOptions{
			Parent: window, Type: mygo.MessageWarning, Title: m.tr("删除查询预设"),
			Message: fmt.Sprintf(m.tr("删除此账本的自定义预设“%s”？"), name),
			Buttons: []string{m.tr("删除"), m.tr("取消")}, DefaultButton: 1, CancelButton: 1,
		})
		if confirmErr != nil || confirmation.Button != 0 || m.closed.Load() {
			window.Update(func() {
				if m.closed.Load() || generation != m.queryPresetGeneration || m.queryPresetLedger != path || m.workspaceState.ActivePath != path {
					return
				}
				m.queryPresetBusy = false
				if confirmErr != nil {
					m.queryPresetError = queryPresetErrorMessage(confirmErr)
				}
			})
			return
		}
		presets, err := store.Delete(path, id)
		window.Update(func() {
			if m.closed.Load() || generation != m.queryPresetGeneration || m.queryPresetLedger != path || m.workspaceState.ActivePath != path {
				return
			}
			m.queryPresetBusy = false
			if err != nil {
				m.queryPresetError = queryPresetErrorMessage(err)
				return
			}
			m.queryPresets = presets
			m.queryPresetError = ""
			m.status = "已删除此账本的自定义预设。"
		})
	}()
}

func queryPresetErrorMessage(err error) string {
	switch {
	case errors.Is(err, querypresets.ErrDuplicateName):
		return "此账本已有同名预设，请换一个名称。"
	case errors.Is(err, querypresets.ErrNotFound):
		return "预设已不存在，请重新读取。"
	case errors.Is(err, querypresets.ErrInvalid):
		return "预设名称或查询文本无效，或超出大小限制。"
	case errors.Is(err, querypresets.ErrLimit):
		return "预设数量或存储容量已达上限。"
	case errors.Is(err, querypresets.ErrSchema):
		return "预设文件版本不受支持，原文件未被覆盖。"
	default:
		return "无法读取或保存预设：" + err.Error()
	}
}

func (m *Model) displayedQueryResponse() *desktop.QueryResponse {
	if document := m.offlineAnalysis; document != nil {
		capture := document.Context
		return &desktop.QueryResponse{
			Context: capture.NormalizedRequest,
			Execution: &query.Execution{
				Result: document.Result, ReturnedRows: capture.ReturnedRows,
				RowLimit: capture.RowLimit, Truncated: capture.Truncated,
			},
		}
	}
	return m.workspaceState.QueryResponse
}

func queryResultSelectionContext(context desktop.QueryContext, offlinePath string) string {
	return fmt.Sprintf("%s:%s:%d:%s:%s", context.SnapshotID, context.SemanticsProfile, context.RequestID, context.Fingerprint, offlinePath)
}

func queryCellTypeSortable(kind string) bool {
	switch kind {
	case "string", "date", "integer", "decimal", "bool", "amount", "position", "inventory":
		return true
	default:
		return false
	}
}

func queryRowsInDisplayOrder(result query.TypedResult, order ui.SortOrder) []int {
	indices := make([]int, len(result.Rows))
	for index := range indices {
		indices[index] = index
	}
	if order.Column == "" {
		return indices
	}
	column := slices.IndexFunc(result.Columns, func(candidate query.Column) bool { return candidate.ID == order.Column })
	if column < 0 || !queryCellTypeSortable(result.Columns[column].Type) {
		return indices
	}
	var compareErr error
	sort.SliceStable(indices, func(leftIndex, rightIndex int) bool {
		leftRow, rightRow := result.Rows[indices[leftIndex]], result.Rows[indices[rightIndex]]
		if column >= len(leftRow) || column >= len(rightRow) {
			return indices[leftIndex] < indices[rightIndex]
		}
		comparison, err := query.CompareCells(leftRow[column], rightRow[column])
		if err != nil {
			compareErr = err
			return indices[leftIndex] < indices[rightIndex]
		}
		if order.Descending {
			return comparison > 0
		}
		return comparison < 0
	})
	if compareErr != nil {
		return naturalQueryRowOrder(len(result.Rows))
	}
	return indices
}

func naturalQueryRowOrder(rowCount int) []int {
	indices := make([]int, rowCount)
	for index := range indices {
		indices[index] = index
	}
	return indices
}

func (m *Model) queryCatalogueResponse() *desktop.QueryCatalogue {
	if m.workspaceState.Phase != workspace.PhaseReady || m.workspaceState.Busy || m.workspaceState.Ledger.Reloading {
		return nil
	}
	response := m.responseFor("query")
	if response == nil || response.QueryCatalogue == nil {
		return nil
	}
	catalogue := response.QueryCatalogue
	expectedSort := "ascending"
	if m.queryCatalogueDescending {
		expectedSort = "descending"
	}
	if catalogue.SnapshotID != m.workspaceState.Ledger.SnapshotID || response.Request.SnapshotID != catalogue.SnapshotID || response.Request.Offset != m.queryCatalogueOffset || response.Request.Limit != 100 || response.Request.Sort != expectedSort || response.Request.QuerySearch != m.queryCatalogueSearch {
		return nil
	}
	return catalogue
}

func (m *Model) queryCataloguePage(c *ui.Context, catalogue *desktop.QueryCatalogue) {
	t := c.Theme()
	ui.Column(c).Padding(12).Gap(8).Background(t.Surface).Border(1, t.Border).Radius(8).Children(func() {
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, m.tr("查询目录")).FontSize(16).Bold()
			ui.Spacer(c)
			ui.Text(c, m.tr("选择只填入草稿，不会运行查询")).TextColor(t.TextMuted)
		})
		if catalogue.SnapshotID != m.workspaceState.Ledger.SnapshotID {
			ui.Text(c, m.tr("此目录来自旧快照；请重新读取后再选择。 ")).TextColor(t.Danger)
			return
		}
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			search := ui.TextInput(c, &m.queryCatalogueSearchDraft).Label(m.tr("搜索账本查询")).Placeholder(m.tr("名称、日期或 BQL 文本")).Grow(1)
			m.focusSearchInput(pageQuery, search)
			if ui.Button(c, m.tr("搜索目录")).Disabled(m.workspaceState.Busy).Clicked() {
				m.applyQueryCatalogueSearch()
			}
			if ui.Button(c, m.tr("切换排序 · 当前日期")+m.tr(querySortLabel(m.queryCatalogueDescending))).Disabled(m.workspaceState.Busy).Clicked() {
				m.queryCatalogueDescending = !m.queryCatalogueDescending
				m.queryCatalogueOffset = 0
				m.requestView("query")
			}
			if m.queryCatalogueSearch != "" && ui.Button(c, m.tr("清除搜索")).Disabled(m.workspaceState.Busy).Clicked() {
				m.queryCatalogueSearch = ""
				m.queryCatalogueSearchDraft = ""
				m.queryCatalogueOffset = 0
				m.queryCatalogueSearchError = ""
				m.requestView("query")
			}
		})
		if m.queryCatalogueSearchError != "" {
			ui.Text(c, m.tr(m.queryCatalogueSearchError)).TextColor(t.Danger)
		}
		ui.Text(c, m.tr("内置模板")).Bold()
		ui.Row(c).Gap(8).Children(func() {
			for _, preset := range catalogue.Presets {
				preset := preset
				if ui.Button(c, m.tr("填入预设 · ")+preset.NameZH).Disabled(catalogue.SnapshotID == "").Clicked() {
					m.fillQueryDraft(preset.Text, catalogue.SnapshotID)
				}
			}
		})
		ui.Text(c, m.tr("按期间生成内置模板")).Bold()
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.TextInput(c, &m.queryPresetPeriodFrom).Label(m.tr("开始日期 YYYY-MM-DD")).Placeholder(m.tr("可留空")).Grow(1)
			ui.TextInput(c, &m.queryPresetPeriodTo).Label(m.tr("结束日期 YYYY-MM-DD")).Placeholder(m.tr("可留空")).Grow(1)
			if ui.Button(c, m.tr("生成期间模板")).Disabled(catalogue.SnapshotID == "").Clicked() {
				m.generateQueryPeriodPresets(catalogue.SnapshotID)
			}
		})
		if m.queryPresetPeriodError != "" {
			ui.Text(c, m.tr(m.queryPresetPeriodError)).TextColor(t.Danger)
		}
		periodIsCurrent := len(m.queryPresetPeriodPresets) > 0 && m.queryPresetPeriodSnapshot == catalogue.SnapshotID && m.queryPresetPeriodGenerated == [2]string{m.queryPresetPeriodFrom, m.queryPresetPeriodTo}
		if len(m.queryPresetPeriodPresets) > 0 {
			if !periodIsCurrent {
				ui.Text(c, m.tr("期间模板参数已变化；请重新生成后再填入。 ")).TextColor(t.TextMuted)
			}
			ui.Row(c).Gap(8).Children(func() {
				for _, preset := range m.queryPresetPeriodPresets {
					preset := preset
					if ui.Button(c, m.tr("填入期间 · ")+preset.NameZH).Disabled(!periodIsCurrent).Clicked() {
						m.fillQueryDraft(preset.Text, catalogue.SnapshotID)
					}
				}
			})
		}
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, fmt.Sprintf(m.tr("账本保存的查询 · %d 条"), catalogue.Page.TotalCount)).Bold().Grow(1)
			ui.Text(c, fmt.Sprintf(m.tr("显示 %d–%d"), queryPageStart(catalogue.Page), catalogue.Page.Offset+catalogue.Page.ReturnedCount)).TextColor(t.TextMuted)
		})
		if len(catalogue.Queries) == 0 {
			if m.queryCatalogueSearch != "" {
				ui.Text(c, m.tr("没有匹配当前搜索条件的 BQL 查询。 ")).TextColor(t.TextMuted)
			} else {
				ui.Text(c, m.tr("当前账本没有保存的 BQL 查询。 ")).TextColor(t.TextMuted)
			}
		} else {
			queries := catalogue.Queries
			m.queryCatalogueList.Key = func(row int) any { return queries[row].ID }
			m.queryCatalogueList.Label = func(row int) string { return queries[row].Name + " " + queries[row].Date }
			ui.List(c, &m.queryCatalogueList, len(queries), func(row int) {
				item := queries[row]
				ui.Row(c).Gap(8).Padding(5, 8).AlignItems(ui.Center).Children(func() {
					if ui.Button(c, fmt.Sprintf(m.tr("填入草稿 · %s · %s · %d"), item.Name, item.Date, item.Source.Line)).Clicked() {
						m.queryCatalogueSelectedID = item.ID
						m.fillQueryDraft(item.Text, catalogue.SnapshotID)
					}
					ui.Text(c, item.Source.Path+":"+fmt.Sprint(item.Source.Line)).FontSize(11).TextColor(t.TextMuted).SingleLine().Grow(1)
				})
			}).Height(180)
		}
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, m.tr("查询目录上一页")).Disabled(catalogue.Page.Offset == 0 || m.workspaceState.Busy).Clicked() {
				m.queryCatalogueOffset = max(0, catalogue.Page.Offset-catalogue.Page.Limit)
				m.requestView("query")
			}
			ui.Spacer(c)
			if ui.Button(c, m.tr("查询目录下一页")).Disabled(!catalogue.Page.HasMore || m.workspaceState.Busy).Clicked() {
				m.queryCatalogueOffset = catalogue.Page.Offset + catalogue.Page.Limit
				m.requestView("query")
			}
		})
	})
}

func (m *Model) generateQueryPeriodPresets(snapshotID string) {
	if snapshotID == "" || snapshotID != m.workspaceState.Ledger.SnapshotID {
		m.queryPresetPeriodPresets = nil
		m.queryPresetPeriodError = "账本快照已变化；没有生成期间模板。"
		return
	}
	presets, err := query.PresetsForPeriod(m.queryPresetPeriodFrom, m.queryPresetPeriodTo)
	if err != nil {
		m.queryPresetPeriodPresets = nil
		m.queryPresetPeriodSnapshot = ""
		m.queryPresetPeriodError = "期间无效：" + err.Error()
		return
	}
	m.queryPresetPeriodPresets = presets
	m.queryPresetPeriodSnapshot = snapshotID
	m.queryPresetPeriodGenerated = [2]string{m.queryPresetPeriodFrom, m.queryPresetPeriodTo}
	m.queryPresetPeriodError = ""
}

func (m *Model) applyQueryCatalogueSearch() {
	if len(m.queryCatalogueSearchDraft) > 65536 || !utf8.ValidString(m.queryCatalogueSearchDraft) {
		m.queryCatalogueSearchError = "目录搜索不能超过 64 KiB 的有效 UTF-8 文本。"
		return
	}
	m.queryCatalogueSearch = strings.TrimSpace(m.queryCatalogueSearchDraft)
	m.queryCatalogueOffset = 0
	m.queryCatalogueSearchError = ""
	m.requestView("query")
}

func querySortLabel(descending bool) string {
	if descending {
		return "降序"
	}
	return "升序"
}

func (m *Model) fillQueryDraft(text, snapshotID string) {
	if snapshotID == "" || snapshotID != m.workspaceState.Ledger.SnapshotID {
		m.status = "账本快照已变化；没有填入旧查询。"
		return
	}
	m.queryDraft = text
	m.status = "已填入草稿；尚未执行。"
}

func (m *Model) copyQueryResultRow(response *desktop.QueryResponse, row int) {
	if m.writeClipboard == nil || response == nil || response.Execution == nil || row < 0 || row >= len(response.Execution.Result.Rows) {
		return
	}
	result := response.Execution.Result
	if len(result.Rows[row]) != len(result.Columns) {
		m.status = "所选结果行的列结构无效，未复制。"
		return
	}
	payload := struct {
		Context  desktop.QueryContext `json:"context"`
		Contract string               `json:"contract"`
		Columns  []query.Column       `json:"columns"`
		Cells    []query.Cell         `json:"cells"`
	}{
		Context: response.Context, Contract: result.Contract, Columns: result.Columns, Cells: result.Rows[row],
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		m.status = "无法编码所选查询结果：" + err.Error()
		return
	}
	m.writeClipboard(string(encoded))
	m.status = "已复制带执行上下文和类型的结果行。"
}

func queryPageStart(page report.ProjectionPage) int {
	if page.ReturnedCount == 0 {
		return 0
	}
	return page.Offset + 1
}

func (m *Model) executeQuery(composing bool) {
	switch {
	case composing:
		m.status = "请先完成输入法组合，再执行查询"
	case len(m.queryDraft) > query.QueryTextLimit:
		m.status = "查询文本超过 64 KiB UTF-8 限制"
	case m.workspace == nil || m.workspaceState.ActivePath == "":
		m.status = "请先打开账本"
	default:
		if err := m.workspace.Dispatch(workspace.Intent{Kind: workspace.IntentExecuteQuery, QueryText: m.queryDraft}); err != nil {
			m.status = "无法执行查询：" + err.Error()
		} else {
			m.offlineAnalysis = nil
			m.offlineAnalysisPath = ""
			m.querySelection = -1
			m.querySelectionContext = ""
			m.status = ""
		}
	}
}

func querySteps(phases map[string]uint64) uint64 {
	var steps uint64
	for _, count := range phases {
		steps += count
	}
	return steps
}

func queryCellText(cell query.Cell) string {
	return queryCellTextWithLocale(cell, func(value string) string { return value })
}

func queryCellTextWithLocale(cell query.Cell, translate func(string) string) string {
	if cell.Type == "null" {
		return "NULL"
	}
	switch value := cell.Value.(type) {
	case query.PresentedDecimal:
		return queryDecimalText(value)
	case query.PresentedAmount:
		return queryAmountText(value)
	case query.PresentedPosition:
		return queryPositionText(value)
	case []query.PresentedPosition:
		return fmt.Sprintf(translate("库存 · %d 项"), len(value))
	case query.PresentedDictionary:
		return fmt.Sprintf(translate("%s · %d 项"), translate(queryCollectionLabel(cell.Type)), len(value))
	case query.PresentedList:
		return fmt.Sprintf(translate("列表 · %d 项"), len(value))
	case []string:
		return fmt.Sprintf(translate("集合 · %d 项"), len(value))
	}
	if value, ok := cell.Value.(string); ok {
		if value == "" {
			return `""`
		}
		return value
	}
	encoded, err := json.Marshal(cell.Value)
	if err != nil {
		return fmt.Sprint(cell.Value)
	}
	return string(encoded)
}

func queryCellDetails(cell query.Cell) []string {
	return queryCellDetailsWithLocale(cell, func(value string) string { return value })
}

func queryCellDetailsWithLocale(cell query.Cell, translate func(string) string) []string {
	lines := make([]string, 0, 8)
	truncated := false
	appendQueryCellDetails(&lines, "", cell, 0, &truncated, translate)
	if truncated {
		lines = append(lines, translate("…其余详情已折叠（最多显示 199 项）"))
	}
	if len(lines) == 0 {
		return []string{queryCellTextWithLocale(cell, translate)}
	}
	return lines
}

func appendQueryCellDetails(lines *[]string, prefix string, cell query.Cell, depth int, truncated *bool, translate func(string) string) {
	if len(*lines) >= 199 {
		*truncated = true
		return
	}
	appendLine := func(value string) {
		if len(*lines) >= 199 {
			*truncated = true
			return
		}
		*lines = append(*lines, prefix+value)
	}
	if depth >= 8 {
		appendLine(queryCellTextWithLocale(cell, translate) + translate("（深层结构已折叠）"))
		return
	}
	if cell.Type == "null" {
		appendLine("NULL")
		return
	}
	switch value := cell.Value.(type) {
	case query.PresentedDecimal:
		appendLine(queryDecimalText(value))
	case query.PresentedAmount:
		appendLine(queryAmountText(value))
	case query.PresentedPosition:
		appendLine(queryPositionText(value))
	case []query.PresentedPosition:
		if len(value) == 0 {
			appendLine(translate("空库存"))
			return
		}
		for index, position := range value {
			appendLine(fmt.Sprintf("[%d] %s", index+1, queryPositionText(position)))
			if len(*lines) >= 199 {
				*truncated = true
				return
			}
		}
	case query.PresentedDictionary:
		if len(value) == 0 {
			appendLine(translate(queryCollectionLabel(cell.Type)) + translate("为空"))
			return
		}
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			appendQueryCellDetails(lines, prefix+strconv.Quote(key)+" · ", value[key], depth+1, truncated, translate)
			if *truncated {
				return
			}
		}
	case query.PresentedList:
		if len(value) == 0 {
			appendLine(translate("空列表"))
			return
		}
		for index, item := range value {
			appendQueryCellDetails(lines, prefix+fmt.Sprintf("[%d] · ", index+1), item, depth+1, truncated, translate)
			if *truncated {
				return
			}
		}
	case []string:
		if len(value) == 0 {
			appendLine(translate("空集合"))
			return
		}
		for index, item := range value {
			appendLine(fmt.Sprintf("[%d] %s", index+1, strconv.Quote(item)))
			if len(*lines) >= 199 {
				*truncated = true
				return
			}
		}
	default:
		appendLine(queryCellTextWithLocale(cell, translate))
	}
}

func queryCollectionLabel(kind string) string {
	switch kind {
	case "open":
		return "账户开立"
	case "close":
		return "账户关闭"
	case "transaction":
		return "交易元数据"
	default:
		return "字典"
	}
}

func queryDecimalText(value query.PresentedDecimal) string {
	if value.Approximate {
		return value.Display + "（近似显示；精确值 " + value.Exact + "）"
	}
	return value.Exact
}

func queryAmountText(value query.PresentedAmount) string {
	return queryDecimalText(value.Number) + " " + value.Currency
}

func queryPositionText(value query.PresentedPosition) string {
	text := "数量 " + queryAmountText(value.Units)
	if value.Cost == nil {
		return text + "；成本缺失"
	}
	text += "；成本 " + queryDecimalText(value.Cost.Number) + " " + value.Cost.Currency
	if value.Cost.Date == nil {
		text += "；成本日期缺失"
	} else {
		text += "；成本日期 " + *value.Cost.Date
	}
	if value.Cost.Label == nil {
		text += "；标签缺失"
	} else {
		text += "；标签 " + strconv.Quote(*value.Cost.Label)
	}
	return text
}

func truncationLabel(truncated bool) string {
	if truncated {
		return " · 已截断"
	}
	return ""
}
