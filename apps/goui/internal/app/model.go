// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"orangecount/apps/goui/internal/charts"
	"orangecount/apps/goui/internal/platform"
	"orangecount/apps/goui/internal/preferences"
	"orangecount/apps/goui/internal/querypresets"
	"orangecount/apps/goui/internal/widgets"
	"orangecount/apps/goui/internal/workspace"
	"orangecount/internal/desktop"
	"orangecount/internal/documents"
	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/report"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
)

const sampleRows = 50_000

type page int

const (
	pageWelcome page = iota
	pageDashboard
	pageBalance
	pageIncome
	pageJournal
	pageAccount
	pagePrices
	pageHoldings
	pageDirectives
	pageSemantics
	pageQuery
	pageAttachments
	pageAttachmentRoots
	pageDiagnostics
	pageTable
	pageCharts
)

// Model is the state owned by one native window. Its fields are read and
// changed on MyGo's main thread; background file work publishes through
// Window.Update.
type Model struct {
	qa                         bool
	windowID                   string
	workspaceEngine            workspace.Engine
	newWindow                  func()
	revealPath                 func(string)
	openSourcePath             func(string) error
	pickAttachmentDirectory    func(mygo.OpenDialogOptions) ([]string, error)
	onLocaleChanged            func()
	window                     *mygo.Window
	closed                     atomic.Bool
	preferenceWork             sync.WaitGroup
	picking                    bool
	openFilePending            bool
	status                     string
	page                       page
	pageHistoryBack            []page
	pageHistoryForward         []page
	searchFocusPending         bool
	searchFocusPage            page
	filterDateFrom             string
	filterDateTo               string
	filterWindow               string
	filterCurrency             string
	filterValuation            string
	locale                     string
	textScale                  float32
	filterError                string
	preferenceError            string
	preferenceStore            *preferences.Store
	queryPresetStore           *querypresets.Store
	queryPresetLedger          string
	queryPresetLoading         bool
	queryPresetBusy            bool
	queryPresetError           string
	queryPresetName            string
	queryPresets               []querypresets.Preset
	queryPresetGeneration      uint64
	queryPresetList            ui.ListState
	recentPaths                []string
	recentPathRecorded         string
	pendingOpen                bool
	pendingPath                string
	pendingPreferences         preferences.LedgerPreferences
	pendingSemanticsSwitch     bool
	pendingSemanticsProfile    ledger.SemanticsProfile
	pendingAttachmentRoots     []string
	pendingInvalidRoots        []string
	pendingRootRestore         bool
	attachmentBookmarks        []platform.DirectoryBookmark
	attachmentRootGeneration   uint64
	invalidAttachmentRoots     []string
	attachmentRootNotice       string
	attachmentActionNotice     string
	attachmentOffset           int
	pendingAttachmentAction    *attachmentActionIntent
	unsubscribeAttachmentRoots func()
	pendingSharedRootChange    *preferences.AttachmentRootsChange
	rootPolicySyncBusy         bool
	table                      ui.ListState
	chartTable                 ui.ListState
	chartTables                map[string]*ui.ListState
	reportTree                 ui.OutlineState[string]
	reportTreeContext          string
	statementSearch            string
	statementSearchDraft       string
	statementSearchError       string
	statementSearchOpenSaved   bool
	statementSearchOpenContext string
	statementSearchOpenKeys    []string
	statementSearchOpenQuery   string
	statementSelection         int
	statementSelectedAccount   string
	journalTable               ui.ListState
	postingTable               ui.ListState
	journalSearch              string
	journalSearchDraft         string
	journalAccount             string
	journalOffset              int
	journalDescending          bool
	journalError               string
	queryTable                 ui.ListState
	querySort                  ui.SortOrder
	queryAppliedSort           ui.SortOrder
	querySelection             int
	querySelectionContext      string
	querySchemaKey             string
	queryDraft                 string
	exportSaveSequence         uint64
	pendingQuerySave           *querySaveRequest
	pendingReportSave          *reportSaveRequest
	activeExportID             uint64
	activeExportCancel         context.CancelFunc
	querySaveDialog            func(context.Context, mygo.SaveDialogOptions) (string, error)
	onExportSaveResult         func(string, error)
	preferenceBackupSequence   uint64
	preferenceBackupCancel     context.CancelFunc
	preferenceBackupSaveDialog func(context.Context, mygo.SaveDialogOptions) (string, error)
	onPreferenceBackupResult   func(string, error)
	offlineAnalysis            *desktop.QueryAnalysisDocument
	offlineAnalysisPath        string
	offlineAnalysisError       string
	offlineAnalysisSequence    uint64
	offlineAnalysisLoading     bool
	queryCatalogueOffset       int
	queryCatalogueDescending   bool
	queryCatalogueSearch       string
	queryCatalogueSearchDraft  string
	queryCatalogueSearchError  string
	queryPresetPeriodFrom      string
	queryPresetPeriodTo        string
	queryPresetPeriodSnapshot  string
	queryPresetPeriodGenerated [2]string
	queryPresetPeriodPresets   []query.Preset
	queryPresetPeriodError     string
	columnLayoutMu             sync.Mutex
	columnLayoutPending        *columnLayoutWrite
	columnLayoutWorker         bool
	columnLayoutRequested      string
	columnLayoutPath           string
	columnLayoutIdentity       string
	columnLayoutPreferences    map[string]ui.TableLayout
	queryLayoutPreferenceKey   string
	queryCatalogueSelectedID   string
	queryCatalogueList         ui.ListState
	journalSelection           int
	journalSelectedID          string
	accountName                string
	accountScope               string
	accountReturnPage          page
	accountReturnResponse      *desktop.ViewResponse
	accountReturnSnapshotID    string
	accountJournalSearch       string
	accountJournalSearchDraft  string
	accountJournalOffset       int
	accountJournalSelection    int
	accountJournalSelectedID   string
	accountJournalTable        ui.ListState
	accountAmountTable         ui.ListState
	priceSection               string
	priceBase                  string
	priceQuote                 string
	priceOffset                int
	commodityOffset            int
	priceDescending            bool
	priceError                 string
	priceSelection             int
	priceSelectedID            string
	priceTable                 ui.ListState
	priceExactTable            ui.ListState
	commoditySearch            string
	commoditySearchDraft       string
	commoditySearchError       string
	commodityTable             ui.ListState
	commoditySort              ui.SortOrder
	commoditySelection         int
	commoditySelectedID        string
	holdingSection             string
	holdingAccount             string
	holdingAccountDraft        string
	holdingScope               string
	holdingGroupBy             string
	holdingGroupID             string
	holdingEventID             string
	holdingLotKind             string
	holdingOffset              int
	holdingDescending          bool
	holdingSelection           int
	holdingSelectedID          string
	holdingGroupTable          ui.ListState
	holdingGroupSort           ui.SortOrder
	holdingEventTable          ui.ListState
	holdingHistoryTable        ui.ListState
	holdingLotTable            ui.ListState
	holdingDetailTable         ui.ListState
	directiveMode              string
	directiveKind              string
	directiveSearch            string
	directiveSearchDraft       string
	directiveError             string
	directiveOffset            int
	directiveDescending        bool
	workspace                  *workspace.Controller
	workspaceState             workspace.State
	dashboardRequestOperation  uint64
	openPath                   func(string) error
	openSourceWithEditor       func(string, string) error
	sourceEditorPicker         func(*mygo.Window) (string, error)
	sourceEditorInspector      func(string) (platform.SourceEditor, error)
	writeClipboard             func(string)
}

type attachmentActionIntent struct {
	id, snapshotID, ledgerPath string
	policyRevision             uint64
	epoch, operation           uint64
	reveal                     bool
}

var nextWindowID atomic.Uint64

func NewModel(qa bool) *Model {
	return newModelWithEngine(qa, nil)
}

func newModelWithEngine(qa bool, engine workspace.Engine) *Model {
	return &Model{
		qa: qa, windowID: fmt.Sprintf("window-%d", nextWindowID.Add(1)), workspaceEngine: engine, journalSelection: -1, accountJournalSelection: -1, commoditySelection: -1, holdingSelection: -1, querySelection: -1, statementSelection: -1, textScale: 1,
		filterValuation: "at-cost", filterWindow: "all", locale: "zh-CN", priceSection: "commodities", priceDescending: true, priceSelection: -1, revealPath: mygo.Shell.ShowItemInFolder, openPath: mygo.Shell.OpenPath, openSourcePath: platform.OpenSource, pickAttachmentDirectory: mygo.Dialog.Open, openSourceWithEditor: platform.OpenSourceWithEditor, sourceEditorInspector: platform.SourceEditorFromApplication, writeClipboard: mygo.Clipboard.WriteText,
		directiveMode: "all", directiveDescending: true, journalDescending: true, queryCatalogueDescending: true,
		holdingSection: "groups", holdingScope: "subtree", holdingGroupBy: "account", holdingLotKind: "consumed", holdingDescending: true,
	}
}

// WindowOptions creates a true MyGo Content window. Page is intentionally
// unset, so the native interface has no WebView or frontend runtime.
func (m *Model) WindowOptions() mygo.WindowOptions {
	return mygo.WindowOptions{
		Title:           "OrangeCount",
		Width:           1180,
		Height:          780,
		MinWidth:        900,
		MinHeight:       600,
		StateKey:        "goui-main-" + m.windowID,
		Content:         ui.View(m.View),
		BackgroundColor: "light-dark(#f5f6f8, #17191d)",
	}
}

func (m *Model) SetWindow(window *mygo.Window) {
	m.window = window
	m.workspace = workspace.NewController(m.windowID, m.workspaceEngine, m.receiveWorkspaceState)
}

func (m *Model) SetNewWindow(open func()) { m.newWindow = open }

func (m *Model) SetLocaleChanged(fn func()) { m.onLocaleChanged = fn }

func (m *Model) SetPreferences(store *preferences.Store) {
	if m.unsubscribeAttachmentRoots != nil {
		m.unsubscribeAttachmentRoots()
		m.unsubscribeAttachmentRoots = nil
	}
	m.preferenceStore = store
	if store == nil {
		m.openSourcePath = platform.OpenSource
		return
	}
	m.openSourcePath = func(path string) error {
		bundleID, err := store.SourceEditorBundleID()
		if err != nil {
			return err
		}
		return m.openSourceWithEditor(path, bundleID)
	}
	m.unsubscribeAttachmentRoots = store.SubscribeAttachmentRoots(m.windowID, m.receiveSharedAttachmentRoots)
	paths, err := store.RecentLedgerPaths()
	if err != nil {
		m.preferenceError = "无法读取最近账本：" + err.Error()
		return
	}
	m.recentPaths = paths
}

// SetPreferenceStartupError keeps preference-directory failures visible even
// when startup could not create a preferences store at all.
func (m *Model) SetPreferenceStartupError(err error) {
	if err != nil {
		m.preferenceError = "应用偏好存储不可用：" + err.Error()
	}
}

func (m *Model) SetQueryPresets(store *querypresets.Store) {
	m.queryPresetStore = store
}

// OpenLedgerPath is the entry used by Finder's Open With and document events.
// It schedules all model changes through the owning native window.
func (m *Model) OpenLedgerPath(path string) {
	window := m.window
	if window == nil || m.closed.Load() || path == "" {
		return
	}
	m.cancelPendingExport()
	m.openFilePending = true
	window.Update(func() {
		m.openFilePending = false
		if m.closed.Load() {
			return
		}
		if m.workspace == nil {
			m.status = "工作区尚未就绪"
			return
		}
		m.page = pageWelcome
		m.searchFocusPending = false
		m.openLedger(path)
	})
}

// CanAcceptDocument reports whether Finder should reuse this welcome window
// for a newly opened ledger. A populated or busy window stays undisturbed.
func (m *Model) CanAcceptDocument() bool {
	if m == nil || m.closed.Load() || m.workspace == nil || m.openFilePending {
		return false
	}
	state := m.workspace.State()
	return state.ActivePath == "" && !state.Busy
}

// CanAcceptQueryAnalysis reports whether this window can display another
// offline capture without replacing a capture that is already being viewed.
func (m *Model) CanAcceptQueryAnalysis() bool {
	return m != nil && !m.closed.Load() && m.workspace != nil &&
		!m.offlineAnalysisLoading && m.offlineAnalysis == nil
}

func (m *Model) Close() {
	if m.closed.Swap(true) {
		return
	}
	m.cancelPendingExport()
	if m.preferenceBackupCancel != nil {
		m.preferenceBackupCancel()
	}
	if m.unsubscribeAttachmentRoots != nil {
		m.unsubscribeAttachmentRoots()
		m.unsubscribeAttachmentRoots = nil
	}
	if m.workspace != nil {
		m.workspace.Close()
	}
	m.columnLayoutMu.Lock()
	m.columnLayoutMu.Unlock()
	m.preferenceWork.Wait()
}

// View builds the native welcome, virtual table, or chart validation page.
func (m *Model) View(c *ui.Context) {
	c.SetTextScale(normalizedTextScale(m.textScale))
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Width(224).FillHeight().Padding(18, 14).Gap(12).Background(t.Surface).Children(func() {
			ui.Text(c, "OrangeCount").FontSize(18).Bold()
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, m.tr("Go UI 开发预览")).FontSize(12).TextColor(t.TextMuted).Grow(1)
				if m.workspaceState.ActivePath != "" {
					localeLabel := "简体中文"
					if m.locale == "en" {
						localeLabel = "English"
					}
					if m.selectLocalized(c, &localeLabel, []string{"简体中文", "English"}).Label(m.tr("界面语言")).Changed() {
						if localeLabel == "English" {
							m.setLocale("en")
						} else {
							m.setLocale("zh-CN")
						}
					}
				}
			})
			ui.Divider(c)
			if ui.PrimaryButton(c, m.tr("打开账本")).Clicked() {
				m.chooseLedger()
			}
			if ui.Button(c, m.tr("新建窗口")).Clicked() && m.newWindow != nil {
				m.newWindow()
			}
			ui.Text(c, m.tr("工作区")).FontSize(11).TextColor(t.TextMuted).Padding(0, 4, 0)
			if ui.Button(c, m.tr("财务概览")).Clicked() {
				m.requestView("dashboard")
			}
			if m.workspaceState.ActivePath != "" {
				if ui.Button(c, m.tr("资产负债表")).Clicked() {
					m.requestView("balance_sheet")
				}
				if ui.Button(c, m.tr("损益表")).Clicked() {
					m.requestView("income_statement")
				}
				if ui.Button(c, m.tr("流水")).Clicked() {
					m.requestView("journal")
				}
				if ui.Button(c, m.tr("商品与价格")).Clicked() {
					m.requestView("prices")
				}
				if ui.Button(c, m.tr("持仓与成本")).Clicked() {
					m.requestView("holdings")
				}
				if ui.Button(c, m.tr("全部指令")).Clicked() {
					m.directiveOffset = 0
					m.requestView("directives")
				}
				if ui.Button(c, m.tr("语义配置")).Clicked() {
					m.requestView("migration")
				}
				if ui.Button(c, m.tr("附件")).Clicked() {
					m.attachmentOffset = 0
					m.requestView("attachments")
				}
				if ui.Button(c, m.tr("查询")).Clicked() {
					m.queryCatalogueOffset = 0
					m.requestView("query")
				}
				if ui.Button(c, m.tr("附件目录")).Clicked() {
					m.navigateToPage(pageAttachmentRoots, "", true)
				}
				if ui.Button(c, m.tr("诊断")).Clicked() {
					m.navigateToPage(pageDiagnostics, "", true)
				}
				if ui.Button(c, m.tr("重新读取")).Clicked() {
					m.reloadActiveLedger()
				}
				if ui.Button(c, m.tr("在 Finder 中显示")).Clicked() {
					m.revealLocation(m.workspaceState.ActivePath)
				}
				m.sourceEditorButton(c, m.workspaceState.ActivePath)
			}
			if m.qa {
				if ui.Button(c, m.tr("虚拟表格 · 50,000 行")).Clicked() {
					m.navigateToPage(pageTable, "", true)
				}
				if ui.Button(c, m.tr("图表绘制试验")).Clicked() {
					m.navigateToPage(pageCharts, "", true)
				}
			}
			ui.Spacer(c)
			if m.qa {
				ui.Text(c, m.tr("QA 模式：不会自动恢复最近账本")).FontSize(11).TextColor(t.TextMuted)
			} else {
				ui.Text(c, m.tr("开发身份与现有客户端数据隔离")).FontSize(11).TextColor(t.TextMuted)
			}
		})
		ui.Column(c).Grow(1).FillHeight().Padding(24).Gap(18).Children(func() {
			ui.Row(c).Gap(10).Children(func() {
				if ui.Button(c, m.tr("返回")).Disabled(len(m.pageHistoryBack) == 0).Clicked() {
					m.navigateBack()
				}
				if ui.Button(c, m.tr("前进")).Disabled(len(m.pageHistoryForward) == 0).Clicked() {
					m.navigateForward()
				}
				title := m.tr("欢迎使用 OrangeCount")
				switch m.page {
				case pageDashboard:
					title = m.tr("财务概览")
				case pageBalance:
					title = m.tr("资产负债表")
				case pageIncome:
					title = m.tr("损益表")
				case pageJournal:
					title = m.tr("流水")
				case pageAccount:
					title = m.accountName
				case pagePrices:
					title = m.tr("商品与价格")
				case pageHoldings:
					title = m.tr("持仓与成本")
				case pageDirectives:
					title = m.tr("全部指令")
				case pageSemantics:
					title = m.tr("语义配置")
				case pageAttachments:
					title = m.tr("附件")
				case pageQuery:
					title = m.tr("查询工作台")
				case pageAttachmentRoots:
					title = m.tr("附件目录")
				case pageDiagnostics:
					title = m.tr("诊断")
				case pageTable:
					title = m.tr("虚拟表格验证")
				case pageCharts:
					title = m.tr("财务绘图验证")
				case pageWelcome:
					if m.workspaceState.ActivePath != "" {
						title = filepath.Base(m.workspaceState.ActivePath)
					}
				}
				ui.Text(c, m.tr(title)).FontSize(25).Bold()
				ui.Spacer(c)
				if m.picking || m.workspaceState.Busy || m.workspaceState.AttachmentRootsBusy || m.rootPolicySyncBusy || m.workspaceState.Ledger.Reloading {
					ui.Text(c, m.tr("正在读取…")).TextColor(t.TextMuted)
				}
			})
			if m.status != "" {
				ui.Text(c, m.tr(m.status)).TextColor(t.TextMuted)
			} else if m.workspaceState.Error != "" {
				ui.Text(c, m.tr(m.workspaceState.Error)).TextColor(t.Danger)
			}
			if m.workspaceState.ActivePath != "" && m.page != pageDirectives && m.page != pageSemantics && m.page != pageAttachments && m.page != pageAttachmentRoots && m.page != pageDiagnostics && m.page != pageQuery {
				m.readingFilters(c)
			}
			m.reportExportControls(c)
			switch m.page {
			case pageDashboard:
				response := m.responseFor("dashboard")
				if response == nil || response.Trends == nil {
					message := "正在读取财务概览…"
					if !m.workspaceState.Busy {
						message = "尚无可显示的财务概览"
					}
					ui.Text(c, m.tr(message)).TextColor(c.Theme().TextMuted)
				} else {
					m.dashboardPage(c, *response.Trends)
				}
			case pageBalance:
				m.statementPage(c, "balance_sheet")
			case pageIncome:
				m.statementPage(c, "income_statement")
			case pageJournal:
				m.journalPage(c)
			case pageAccount:
				m.accountPage(c)
			case pagePrices:
				m.pricesPage(c)
			case pageHoldings:
				m.holdingsPage(c)
			case pageDirectives:
				m.directivesPage(c)
			case pageSemantics:
				m.semanticsPage(c)
			case pageAttachments:
				m.attachmentsPage(c)
			case pageQuery:
				m.queryPage(c)
			case pageAttachmentRoots:
				m.attachmentRootsPage(c)
			case pageDiagnostics:
				if m.workspaceState.Response != nil {
					m.diagnosticsPage(c, m.workspaceState.Response.Summary.Diagnostics)
				} else {
					ui.Text(c, m.tr("当前没有可显示的诊断。 ")).TextColor(c.Theme().TextMuted)
				}
			case pageTable:
				m.tablePage(c)
			case pageCharts:
				m.chartPage(c)
			default:
				m.welcomePage(c)
			}
		})
	})
	m.persistColumnLayouts()
}

func (m *Model) welcomePage(c *ui.Context) {
	t := c.Theme()
	if m.workspaceState.ActivePath != "" {
		var summary *desktop.SummaryPayload
		if m.workspaceState.Response != nil {
			summary = &m.workspaceState.Response.Summary
		}
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Gap(12).Children(func() {
				ui.Column(c).Padding(20).Gap(12).Background(t.Surface).Border(1, t.Border).Radius(12).Children(func() {
					if m.workspaceState.Phase == workspace.PhaseReady {
						ui.Text(c, m.tr("账本已读取")).FontSize(18).Bold()
					} else if m.workspaceState.Phase == workspace.PhaseStale {
						ui.Text(c, m.tr("账本文件有变更，当前显示上次有效快照")).FontSize(18).Bold().TextColor(t.Danger)
					} else if m.workspaceState.Phase == workspace.PhaseInvalid {
						ui.Text(c, m.tr("账本包含诊断")).FontSize(18).Bold().TextColor(t.Danger)
					} else {
						ui.Text(c, m.tr("无法打开新账本，仍保留当前工作区")).FontSize(18).Bold().TextColor(t.Danger)
					}
					if summary != nil {
						ui.Text(c, fmt.Sprintf(m.tr("%d 个账户 · %d 笔交易 · %d 条分录"), len(summary.Accounts), summary.TransactionsCount, summary.EntriesCount))
						if len(summary.Errors) > 0 {
							ui.Text(c, summary.Errors[0]).TextColor(t.Danger)
						}
					}
					ui.Text(c, m.workspaceState.ActivePath).FontSize(12).TextColor(t.TextMuted)
				})
				if m.workspaceState.Response != nil && m.workspaceState.Response.Trends != nil {
					m.dashboardPage(c, *m.workspaceState.Response.Trends)
				}
				if m.workspaceState.Response != nil {
					m.diagnosticsPage(c, m.workspaceState.Response.Summary.Diagnostics)
				}
			})
		})
		return
	}
	ui.Column(c).Grow(1).Center().Gap(14).Children(func() {
		ui.Text(c, m.tr("账本数据保留在本机")).FontSize(24).Bold()
		ui.Text(c, m.tr("打开一个 Beancount 文件，开始查看你的账本。此开发预览不会自动读取最近账本。")).MaxWidth(580).TextColor(t.TextMuted)
		if m.preferenceError != "" {
			ui.Text(c, m.tr(m.preferenceError)).MaxWidth(580).TextColor(t.Danger)
		}
		if ui.PrimaryButton(c, m.tr("选择账本…")).Clicked() {
			m.chooseLedger()
		}
		if len(m.recentPaths) > 0 {
			ui.Column(c).Gap(6).Children(func() {
				ui.Text(c, m.tr("最近账本")).FontSize(14).Bold()
				for _, path := range m.recentPaths {
					path := path
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						if ui.Button(c, filepath.Base(path)).Label(path).Tooltip(path).Clicked() {
							if m.workspace == nil {
								m.status = "工作区尚未就绪"
							} else {
								m.openLedger(path)
							}
						}
						ui.Text(c, filepath.Dir(path)).SingleLine().TextColor(t.TextMuted)
					})
				}
			})
		}
	})
}

func (m *Model) dashboardPage(c *ui.Context, trends desktop.HistoricalTrendsPayload) {
	t := c.Theme()
	ui.Column(c).Gap(12).Padding(18, 0, 0).Children(func() {
		ui.Text(c, m.tr("财务概览")).FontSize(19).Bold()
		if len(trends.CurrencySeries) == 0 {
			if trends.NetWorthAvailability != "" && trends.NetWorthAvailability != "empty" {
				ui.Text(c, m.tr(availabilityText(trends.NetWorthAvailability))).TextColor(t.TextMuted)
			} else {
				ui.Text(c, m.tr("账本尚无可绘制的月度收支记录")).TextColor(t.TextMuted)
			}
			return
		}
		for _, item := range trends.CurrencySeries {
			item := item
			ui.Column(c).Gap(8).Padding(14).Background(t.Surface).Border(1, t.Border).Radius(10).Children(func() {
				ui.Text(c, item.Currency+" · "+item.Valuation).FontSize(15).Bold()
				if item.NetWorthAvailability != "priced" && item.NetWorthAvailability != "empty" {
					ui.Text(c, m.tr(availabilityText(item.NetWorthAvailability))).TextColor(t.TextMuted)
				}
				if len(item.NetWorthPoints) > 0 {
					points := make([]charts.Point, 0, len(item.NetWorthPoints))
					for _, point := range item.NetWorthPoints {
						display := m.tr("不可用")
						available := point.Value != nil && point.Valuation != nil && point.Valuation.CompleteTotal != nil
						detail := m.historyValuationDetail(point.Valuation, trends.SourceFiles)
						if available {
							display = point.Valuation.CompleteTotal.Exact + " " + item.Currency
						}
						value := 0.0
						if point.Value != nil {
							value = *point.Value
						}
						points = append(points, charts.Point{Label: point.Month, Value: value, Exact: display, Detail: detail, Available: available, Segment: point.Segment})
					}
					ui.Text(c, m.tr("净资产变化")).FontSize(14).Bold()
					LineChart(c, charts.Series{Label: m.tr("净资产"), Color: ui.RGB(42, 119, 214), Points: points}, m.tr)
					ExactTable(c, m.chartTableFor("net-worth:"+item.Currency), []charts.Series{{Label: m.tr("净资产"), Color: ui.RGB(42, 119, 214), Points: points}}, m.tr)
				} else {
					ui.Text(c, m.tr("净资产图表暂不可用：")+m.tr(availabilityText(item.NetWorthAvailability))).TextColor(t.TextMuted)
				}

				if len(item.MonthlyCashFlows) > 0 {
					if item.CashFlowAvailability != "priced" && item.CashFlowAvailability != "empty" {
						ui.Text(c, m.tr(availabilityText(item.CashFlowAvailability))).TextColor(t.TextMuted)
					}
					income := charts.Series{Label: m.tr("收入"), Color: ui.RGB(40, 145, 100)}
					expenses := charts.Series{Label: m.tr("支出"), Color: ui.RGB(210, 80, 76)}
					for _, flow := range item.MonthlyCashFlows {
						incomeDisplay, expenseDisplay := m.tr("不可用"), m.tr("不可用")
						incomeAvailable := flow.IncomeValue != nil && flow.Income != nil && flow.Income.CompleteTotal != nil
						expenseAvailable := flow.ExpensesValue != nil && flow.Expenses != nil && flow.Expenses.CompleteTotal != nil
						incomeValue, expenseValue := 0.0, 0.0
						if incomeAvailable {
							incomeValue = *flow.IncomeValue
							incomeDisplay = flow.Income.CompleteTotal.Exact + " " + item.Currency
						}
						if expenseAvailable {
							expenseValue = -*flow.ExpensesValue
							expenseDisplay = flow.Expenses.CompleteTotal.Exact + " " + item.Currency
						}
						income.Points = append(income.Points, charts.Point{Label: flow.Month, Value: incomeValue, Exact: incomeDisplay, Available: incomeAvailable})
						expenses.Points = append(expenses.Points, charts.Point{Label: flow.Month, Value: expenseValue, Exact: expenseDisplay, Available: expenseAvailable})
					}
					series := []charts.Series{income, expenses}
					ui.Text(c, m.tr("月度收支变化")).FontSize(14).Bold()
					GroupedBars(c, series, m.tr)
					ExactTable(c, m.chartTableFor("cash-flow:"+item.Currency), series, m.tr)
					if item.IncomeTotal != nil && item.ExpensesTotal != nil && item.NetTotal != nil {
						ui.Text(c, m.tr("合计：收入 ")+item.IncomeTotal.Display+m.tr(" · 支出 ")+item.ExpensesTotal.Display+m.tr(" · 净流量 ")+item.NetTotal.Display+" "+item.Currency).TextColor(t.TextMuted)
					}
				} else {
					ui.Text(c, m.tr("收支图表暂不可用：")+m.tr(availabilityText(item.CashFlowAvailability))).TextColor(t.TextMuted)
				}
			})
		}
	})
}

func (m *Model) historyValuationDetail(value *report.AccountValuation, sourceFiles map[source.FileID]string) string {
	if value == nil {
		return ""
	}
	const detailLimit = 3
	details := make([]string, 0, detailLimit+1)
	for _, quote := range value.Quotes {
		if len(details) == detailLimit {
			break
		}
		kind := m.tr("直接报价")
		if quote.Inverse {
			kind = m.tr("反向报价")
		}
		text := m.tr("采用报价") + " " + quote.Base + "/" + quote.Quote + " " + quote.Rate.Exact + " · " + quote.Date + " · " + kind
		if path := sourceFiles[quote.Source.File]; path != "" {
			text += " · " + sourceLabel(report.ProjectionSource{Path: path, Line: quote.Source.StartLine, Column: quote.Source.StartColumn})
		}
		details = append(details, text)
	}
	for _, missing := range value.Missing {
		if len(details) == detailLimit {
			break
		}
		details = append(details, missing.Commodity+"："+m.tr(valuationReasonText(missing.Reason)))
	}
	remaining := len(value.Quotes) + len(value.Missing) - min(len(details), detailLimit)
	if remaining > 0 {
		details = append(details, fmt.Sprintf(m.tr("另有 %d 项估值明细"), remaining))
	}
	return strings.Join(details, "；")
}

func (m *Model) chartTableFor(key string) *ui.ListState {
	if m.chartTables == nil {
		m.chartTables = make(map[string]*ui.ListState)
	}
	if state := m.chartTables[key]; state != nil {
		return state
	}
	state := &ui.ListState{}
	m.chartTables[key] = state
	return state
}

func (m *Model) revealLocation(path string) {
	if path != "" && m.revealPath != nil {
		m.revealPath(path)
	}
}

func (m *Model) sourceEditorButton(c *ui.Context, path string) {
	if path == "" {
		return
	}
	disabled, open := m.sourceOpenAction(path)
	if ui.Button(c, m.tr("打开源文件")).Disabled(disabled).Clicked() {
		open()
	}
}

func (m *Model) sourceOpenAction(path string) (bool, func()) {
	epoch, operation, snapshotID := m.workspaceState.Epoch, m.workspaceState.Operation, m.workspaceState.Ledger.SnapshotID
	disabled := m.workspaceState.Response == nil || m.workspaceState.Busy || m.workspaceState.Ledger.Reloading
	return disabled, func() { m.openSourceText(path, epoch, operation, snapshotID) }
}

func (m *Model) openSourceText(path string, epoch, operation uint64, snapshotID string) {
	if path == "" || m.workspace == nil {
		m.status = "来源上下文已失效；请重新读取后再试。"
		return
	}
	published := m.workspaceState
	current := m.workspace.State()
	if published.Epoch != epoch || published.Operation != operation || published.Ledger.SnapshotID != snapshotID ||
		current.Epoch != epoch || current.Operation != operation || current.Ledger.SnapshotID != snapshotID ||
		current.Revision != published.Revision || published.Busy || current.Busy || current.Ledger.Reloading ||
		current.Response == nil || current.Response.State.SnapshotID != snapshotID {
		m.status = "账本或来源视图已变化；没有打开来源。"
		return
	}
	if m.openSourcePath == nil {
		m.status = "外部源文件编辑器不可用。"
		return
	}
	if err := m.openSourcePath(path); err != nil {
		m.status = "无法打开源文件：" + err.Error()
		return
	}
	m.status = "已在外部编辑器中打开源文件。"
}

func (m *Model) diagnosticsPage(c *ui.Context, diagnostics []desktop.DiagnosticDTO) {
	t := c.Theme()
	if m.workspaceState.Phase == workspace.PhaseStale {
		ui.Text(c, m.tr("账本文件已有变化；下列内容来自当前工作区保留的诊断，报表仍使用上次有效快照。 ")).TextColor(t.Danger)
	} else if m.workspaceState.Phase == workspace.PhaseInvalid {
		ui.Text(c, m.tr("当前账本无有效快照，请根据诊断修复来源后重新读取。 ")).TextColor(t.Danger)
	}
	if len(diagnostics) == 0 {
		ui.Text(c, m.tr("没有解析诊断。 ")).TextColor(t.TextMuted)
		return
	}
	const visibleLimit = 100
	if len(diagnostics) > visibleLimit {
		diagnostics = diagnostics[:visibleLimit]
	}
	ui.Column(c).Gap(8).Children(func() {
		ui.Text(c, fmt.Sprintf(m.tr("诊断 · 显示 %d 条"), len(diagnostics))).FontSize(17).Bold()
		for _, item := range diagnostics {
			item := item
			ui.Column(c).Padding(12).Gap(6).Background(t.Surface).Border(1, t.Border).Radius(8).Children(func() {
				ui.Row(c).Gap(8).Children(func() {
					ui.Text(c, string(item.Severity)+" · "+item.Code).Bold()
					ui.Spacer(c)
				})
				ui.Text(c, item.Message)
				if item.Source.Path != "" {
					widgets.SourceRow(c, widgets.SourceLocation{Path: item.Source.Path, Line: item.Source.Line, Column: item.Source.Column}, m.diagnosticSourceActions(item.Source, "复制位置", "在 Finder 中显示来源"))
				}
				for _, related := range item.Related {
					related := related
					if related.Source.Path == "" {
						continue
					}
					ui.Column(c).Gap(4).Children(func() {
						ui.Text(c, related.Message).FontSize(12).TextColor(t.TextMuted).SingleLine()
						widgets.SourceRow(c, widgets.SourceLocation{Path: related.Source.Path, Line: related.Source.Line, Column: related.Source.Column}, m.diagnosticSourceActions(related.Source, "复制关联位置", "显示关联来源"))
					})
				}
			})
		}
		if m.workspaceState.Response != nil && len(m.workspaceState.Response.Summary.Diagnostics) > visibleLimit {
			ui.Text(c, m.tr("其余诊断暂未显示")).TextColor(t.TextMuted)
		}
	})
}

func (m *Model) diagnosticSourceActions(location desktop.SourceLocation, copyLabel, revealLabel string) widgets.SourceActions {
	disabled, open := m.sourceOpenAction(location.Path)
	return widgets.SourceActions{
		CopyLabel: m.tr(copyLabel), RevealLabel: m.tr(revealLabel), OpenLabel: m.tr("打开源文件"), OpenDisabled: disabled,
		Copy:   func() { m.copyDiagnosticLocation(location) },
		Reveal: func() { m.revealLocation(location.Path) },
		Open:   open,
	}
}

func (m *Model) copyDiagnosticLocation(location desktop.SourceLocation) {
	if location.Path == "" || m.writeClipboard == nil {
		return
	}
	m.writeClipboard(widgets.SourceText(widgets.SourceLocation{Path: location.Path, Line: location.Line, Column: location.Column}))
	m.status = "已复制来源位置"
}

func (m *Model) requestView(view string) {
	target, ok := pageForView(view)
	if !ok {
		m.status = "不支持的页面"
		return
	}
	m.navigateToPage(target, view, true)
}

func (m *Model) applyDirectiveSearch() {
	if len(m.directiveSearchDraft) > 65536 {
		m.directiveError = "搜索文字超过 64 KiB 上限。"
		return
	}
	m.directiveSearch = m.directiveSearchDraft
	m.directiveOffset = 0
	m.directiveError = ""
	m.requestView("directives")
}

func requestForView(view string) desktop.Request {
	request := desktop.Request{View: view, Valuation: "at-cost", Locale: "zh-CN"}
	if view == "journal" {
		request.JournalMode = "transactions"
		request.Sort = "descending"
		request.Limit = 100
	}
	return request
}

func (m *Model) requestForView(view string) desktop.Request {
	if view == "directives" {
		mode := m.directiveMode
		if mode != "all" && mode != "config" {
			mode = "all"
		}
		sort := "ascending"
		if mode == "all" && m.directiveDescending {
			sort = "descending"
		}
		return desktop.Request{
			View: "journal", JournalMode: mode, DirectiveKind: m.directiveKind,
			Text: m.directiveSearch, Sort: sort, Offset: m.directiveOffset, Limit: 100, Locale: m.requestLocale(),
		}
	}
	if view == "attachments" {
		return desktop.Request{
			View: "journal", JournalMode: "all", DirectiveKind: "document", Sort: "descending",
			Offset: m.attachmentOffset, Limit: 100, Locale: m.requestLocale(),
		}
	}
	if view == "journal" {
		sortOrder := "ascending"
		if m.journalDescending {
			sortOrder = "descending"
		}
		return desktop.Request{
			View: "journal", JournalMode: "transactions", Text: m.journalSearch,
			Account: m.journalAccount, Sort: sortOrder, Offset: m.journalOffset, Limit: 100,
			SnapshotID: m.workspaceState.Ledger.SnapshotID, DateFrom: m.filterDateFrom,
			DateTo: m.filterDateTo, Window: m.filterWindow, Locale: m.requestLocale(),
		}
	}
	if view == "account" {
		return desktop.Request{
			View: "account", Account: m.accountName, Scope: m.accountScope,
			Text: m.accountJournalSearch, Sort: "descending", Offset: m.accountJournalOffset, Limit: 100,
			SnapshotID: m.workspaceState.Ledger.SnapshotID, DateFrom: m.filterDateFrom, DateTo: m.filterDateTo,
			Window: m.filterWindow, Currency: m.filterCurrency, Valuation: m.filterValuation, Locale: m.requestLocale(),
		}
	}
	if view == "prices" {
		section := m.priceSection
		if section != "history" {
			section = "commodities"
		}
		offset := m.commodityOffset
		priceBase, priceQuote := "", ""
		if section == "history" {
			priceBase, priceQuote = m.priceBase, m.priceQuote
			offset = m.priceOffset
		}
		sortOrder := "ascending"
		if m.priceDescending {
			sortOrder = "descending"
		}
		return desktop.Request{
			View: "prices", PriceSection: section, PriceBase: priceBase, PriceQuote: priceQuote,
			CommoditySearch: func() string {
				if section == "commodities" {
					return m.commoditySearch
				}
				return ""
			}(),
			Sort: sortOrder, Offset: offset, Limit: 100, SnapshotID: m.workspaceState.Ledger.SnapshotID,
			DateFrom: m.filterDateFrom, DateTo: m.filterDateTo, Window: m.filterWindow, Locale: m.requestLocale(),
		}
	}
	if view == "holdings" {
		section := m.holdingSection
		if section == "" {
			section = "groups"
		}
		sortOrder := "ascending"
		if m.holdingDescending {
			sortOrder = "descending"
		}
		request := desktop.Request{
			View: "holdings", HoldingSection: section, Account: m.holdingAccount, Scope: m.holdingScope,
			GroupBy: m.holdingGroupBy, Sort: sortOrder, Offset: m.holdingOffset, Limit: 100,
			SnapshotID: m.workspaceState.Ledger.SnapshotID, DateFrom: m.filterDateFrom, DateTo: m.filterDateTo,
			Window: m.filterWindow, Currency: m.filterCurrency, Valuation: m.filterValuation, Locale: m.requestLocale(),
		}
		if section == "lots" {
			if m.holdingEventID != "" {
				request.EventID, request.LotKind = m.holdingEventID, m.holdingLotKind
			} else {
				request.GroupID = m.holdingGroupID
			}
		} else if section == "history" {
			request.GroupID = m.holdingGroupID
		}
		return request
	}
	if view == "query" {
		sortOrder := "ascending"
		if m.queryCatalogueDescending {
			sortOrder = "descending"
		}
		return desktop.Request{
			View: "query", SnapshotID: m.workspaceState.Ledger.SnapshotID, Sort: sortOrder,
			QuerySearch: m.queryCatalogueSearch, Offset: m.queryCatalogueOffset, Limit: 100, Locale: m.requestLocale(),
		}
	}
	request := requestForView(view)
	request.Locale = m.requestLocale()
	request.DateFrom = m.filterDateFrom
	request.DateTo = m.filterDateTo
	request.Window = m.filterWindow
	request.Currency = m.filterCurrency
	request.Valuation = m.filterValuation
	if view == "migration" {
		request.SnapshotID = m.workspaceState.Ledger.SnapshotID
	}
	return request
}

func (m *Model) readingFilters(c *ui.Context) {
	t := c.Theme()
	currencyOptions := []string{"原币"}
	if m.workspaceState.Response != nil {
		for _, currency := range m.workspaceState.Response.Summary.Currencies {
			currencyOptions = append(currencyOptions, currency)
		}
	}
	selectedCurrency := "原币"
	if m.filterCurrency != "" {
		selectedCurrency = m.filterCurrency
	}
	valuationLabel := "按成本"
	if m.filterValuation == "market-value" {
		valuationLabel = "按市价"
	}
	ui.Column(c).Gap(6).Children(func() {
		periodSelection := 2
		switch m.filterWindow {
		case "1m":
			periodSelection = 0
		case "1y":
			periodSelection = 1
		}
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, m.tr("期间")).FontSize(11).TextColor(t.TextMuted)
			if ui.Segmented(c, &periodSelection, m.tr("近1月"), m.tr("近1年"), m.tr("全部")).Label(m.tr("期间")).Changed() {
				switch periodSelection {
				case 0:
					m.filterWindow = "1m"
				case 1:
					m.filterWindow = "1y"
				default:
					m.filterWindow = "all"
				}
				m.filterDateFrom, m.filterDateTo = "", ""
			}
		})
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).Gap(3).Children(func() {
				ui.Text(c, m.tr("开始日期")).FontSize(11).TextColor(t.TextMuted)
				ui.TextInput(c, &m.filterDateFrom).Placeholder("YYYY-MM-DD").Label(m.tr("开始日期 YYYY-MM-DD"))
			})
			ui.Column(c).Grow(1).Gap(3).Children(func() {
				ui.Text(c, m.tr("结束日期")).FontSize(11).TextColor(t.TextMuted)
				ui.TextInput(c, &m.filterDateTo).Placeholder("YYYY-MM-DD").Label(m.tr("结束日期 YYYY-MM-DD"))
			})
			if ui.Button(c, m.tr("应用筛选")).Clicked() {
				m.applyReadingFilters()
			}
		})
		if m.page == pageJournal {
			ui.Text(c, m.tr("流水按日期、账户和文字筛选；估值与币种不改变原始分录。 ")).FontSize(11).TextColor(t.TextMuted)
		} else if m.page == pagePrices {
			ui.Text(c, m.tr("价格历史只使用日期范围和本地来源报价；没有报价时保留 exact 记录但不连线。 ")).FontSize(11).TextColor(t.TextMuted)
		} else {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, m.tr("币种")).FontSize(11).TextColor(t.TextMuted)
				if m.selectOptions(c, &selectedCurrency, currencyOptions, map[string]string{"原币": m.tr("原币")}).Changed() {
					if selectedCurrency == "原币" {
						m.filterCurrency = ""
					} else {
						m.filterCurrency = selectedCurrency
					}
				}
				ui.Text(c, m.tr("估值")).FontSize(11).TextColor(t.TextMuted)
				if m.selectLocalized(c, &valuationLabel, []string{"按成本", "按市价"}).Changed() {
					if valuationLabel == "按市价" {
						m.filterValuation = "market-value"
					} else {
						m.filterValuation = "at-cost"
					}
				}
			})
		}
		if m.preferenceError != "" {
			ui.Text(c, m.tr(m.preferenceError)).FontSize(12).TextColor(t.Danger)
		}
		if m.filterError != "" {
			ui.Text(c, m.tr(m.filterError)).FontSize(12).TextColor(t.Danger)
		}
	})
}

func (m *Model) applyReadingFilters() {
	from, to, err := validateDateRange(m.filterDateFrom, m.filterDateTo)
	if err != nil {
		m.filterError = err.Error()
		return
	}
	m.filterDateFrom, m.filterDateTo = from, to
	m.filterError = ""
	view := "dashboard"
	switch m.page {
	case pageAccount:
		view = "account"
	case pagePrices:
		view = "prices"
	case pageHoldings:
		view = "holdings"
	case pageBalance:
		view = "balance_sheet"
	case pageIncome:
		view = "income_statement"
	case pageJournal:
		view = "journal"
		m.journalOffset = 0
	case pageDirectives:
		view = "directives"
	}
	if m.page == pagePrices {
		m.priceOffset = 0
		m.priceSelection, m.priceSelectedID = -1, ""
	}
	if m.page == pageHoldings {
		if m.holdingSection == "lots" {
			if m.holdingEventID != "" {
				m.holdingSection = "events"
			} else {
				m.holdingSection = "groups"
			}
		}
		m.holdingOffset = 0
		m.holdingSelection, m.holdingSelectedID = -1, ""
		m.holdingGroupID, m.holdingEventID = "", ""
	}
	m.requestView(view)
	if m.status == "" {
		m.persistReadingPreferences()
	}
}

func (m *Model) persistReadingPreferences() {
	if m.preferenceStore == nil || m.workspaceState.ActivePath == "" {
		return
	}
	store := m.preferenceStore
	path := m.workspaceState.ActivePath
	window := m.filterWindow
	from, to := m.filterDateFrom, m.filterDateTo
	currency, valuation := m.filterCurrency, m.filterValuation
	textScale := normalizedTextScale(m.textScale)
	profile := string(m.workspaceState.Ledger.SemanticsProfile)
	m.preferenceWork.Add(1)
	go func() {
		defer m.preferenceWork.Done()
		err := store.UpdateLedger(path, func(item *preferences.LedgerPreferences) error {
			item.SemanticsProfile = profile
			item.Window = window
			item.DateFrom, item.DateTo = from, to
			item.Currency, item.Valuation = currency, valuation
			item.TextScale = textScale
			return nil
		})
		if err == nil {
			return
		}
		message := "偏好未能保存：" + err.Error()
		nativeWindow := m.window
		if nativeWindow == nil || m.closed.Load() {
			return
		}
		nativeWindow.Update(func() {
			if !m.closed.Load() {
				m.preferenceError = message
			}
		})
	}()
}

func (m *Model) persistSemanticsProfile(path string, profile ledger.SemanticsProfile) {
	if m.preferenceStore == nil || path == "" || (profile != ledger.Legacy && profile != ledger.V3) {
		return
	}
	store := m.preferenceStore
	m.preferenceWork.Add(1)
	go func() {
		defer m.preferenceWork.Done()
		err := store.UpdateLedger(path, func(item *preferences.LedgerPreferences) error {
			item.SemanticsProfile = string(profile)
			return nil
		})
		if err == nil {
			return
		}
		message := "语义配置未能保存：" + err.Error()
		nativeWindow := m.window
		if nativeWindow == nil || m.closed.Load() {
			return
		}
		nativeWindow.Update(func() {
			if !m.closed.Load() {
				m.preferenceError = message
			}
		})
	}()
}

func (m *Model) recordRecentLedger(path string) {
	if m.preferenceStore == nil || path == "" || path == m.recentPathRecorded || m.closed.Load() {
		return
	}
	m.recentPathRecorded = path
	store, nativeWindow := m.preferenceStore, m.window
	m.preferenceWork.Add(1)
	go func() {
		defer m.preferenceWork.Done()
		err := store.RecordRecentLedger(path)
		var paths []string
		if err == nil {
			paths, err = store.RecentLedgerPaths()
		}
		if nativeWindow == nil || m.closed.Load() {
			return
		}
		nativeWindow.Update(func() {
			if m.closed.Load() {
				return
			}
			if err != nil {
				m.preferenceError = "最近账本未能保存：" + err.Error()
				return
			}
			m.preferenceError = ""
			m.recentPaths = paths
		})
	}()
}

func validateDateRange(from, to string) (string, string, error) {
	from, to = strings.TrimSpace(from), strings.TrimSpace(to)
	for _, item := range []struct{ label, value string }{{"开始日期", from}, {"结束日期", to}} {
		label, value := item.label, item.value
		if value == "" {
			continue
		}
		date, err := time.Parse(time.DateOnly, value)
		if err != nil || date.Format(time.DateOnly) != value {
			return "", "", fmt.Errorf("%s格式应为 YYYY-MM-DD", label)
		}
	}
	if from != "" && to != "" && from > to {
		return "", "", fmt.Errorf("开始日期不能晚于结束日期")
	}
	return from, to, nil
}

func (m *Model) responseFor(view string) *desktop.ViewResponse {
	response := m.workspaceState.Response
	if response == nil || response.Request.View != view {
		return nil
	}
	return response
}

func (m *Model) statementPage(c *ui.Context, view string) {
	t := c.Theme()
	response := m.responseFor(view)
	if response == nil || response.Tree == nil {
		message := "正在读取报表…"
		if !m.workspaceState.Busy {
			message = "尚无可显示的报表数据"
		}
		ui.Text(c, m.tr(message)).TextColor(t.TextMuted)
		return
	}
	tree := response.Tree
	accountTree := widgets.NewAccountTree(tree.RootNodes, m.statementSearch)
	currencies := accountTree.Currencies()
	ui.Row(c).Gap(12).Children(func() {
		if tree.DateTo != "" {
			ui.Text(c, m.tr("截至 ")+tree.DateTo).TextColor(t.TextMuted)
		}
		if tree.Availability != "" {
			ui.Text(c, m.tr(availabilityText(tree.Availability))).TextColor(t.TextMuted)
		}
		if tree.Currency != "" {
			ui.Text(c, m.tr("估值币种 ")+tree.Currency).TextColor(t.TextMuted)
		}
	})
	if len(tree.RootNodes) > 0 {
		ui.Column(c).Gap(4).Children(func() {
			ui.Text(c, m.tr("分类汇总 · 不受搜索或折叠影响")).FontSize(12).Bold().TextColor(t.TextMuted)
			for _, root := range tree.RootNodes {
				root := root
				ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
					ui.Text(c, root.Name).FontSize(12).Bold()
					for _, currency := range currencies {
						amount, ok := root.SubtreeAmounts[currency]
						if ok {
							widgets.AmountCell(c, amount, currency).FontSize(12)
						}
					}
				})
				m.statementValuationCell(c, root.SubtreeValuation)
			}
		})
	}
	columns := []ui.TableColumn{
		{Title: m.tr("账户"), ID: "account", Width: 260, Fixed: true},
		{Title: m.tr("直属余额"), ID: "direct-amounts", Width: 190, Align: ui.End},
		{Title: m.tr("汇总余额"), ID: "subtree-amounts", Width: 220, Align: ui.End},
		{Title: m.tr("详情"), ID: "account-detail", Width: 80},
	}
	treeContext := response.Request.View + "\x00" + m.workspaceState.ActivePath
	if m.reportTreeContext != treeContext {
		m.reportTree.Open.Clear()
		accountTree.SetExpanded(&m.reportTree, true)
		m.reportTreeContext = treeContext
		m.clearStatementSearchOpenSnapshot()
		m.statementSelection = -1
		m.statementSelectedAccount = ""
	}
	accountTree.PruneExpanded(&m.reportTree)
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		search := ui.TextInput(c, &m.statementSearchDraft).Placeholder(m.tr("账户名称或完整路径")).Label(m.tr("搜索账户")).Grow(1)
		m.focusSearchInput(m.page, search)
		if ui.PrimaryButton(c, m.tr("筛选账户")).Clicked() {
			if len(m.statementSearchDraft) > 4096 {
				m.statementSearchError = "账户搜索文字超过 4 KiB 上限。"
			} else {
				m.statementSearch = strings.ToLower(strings.TrimSpace(m.statementSearchDraft))
				m.statementSearchError = ""
				m.statementSearchOpenQuery = ""
			}
		}
		if ui.Button(c, m.tr("清除搜索")).Disabled(m.statementSearch == "" && m.statementSearchDraft == "").Clicked() {
			m.statementSearch = ""
			m.statementSearchDraft = ""
			m.statementSearchError = ""
		}
	})
	accountTree = widgets.NewAccountTree(tree.RootNodes, m.statementSearch)
	m.syncStatementSearchExpansion(accountTree, treeContext)
	if m.statementSearchError != "" {
		ui.Text(c, m.tr(m.statementSearchError)).TextColor(t.Danger)
	}
	ui.Text(c, m.tr("搜索只筛选可见账户行；汇总金额由核心报表提供，不会因搜索或折叠而重算。 ")).FontSize(11).TextColor(t.TextMuted)
	visibleRootKeys := accountTree.RootKeys()
	if len(visibleRootKeys) == 0 {
		ui.Text(c, m.tr("没有匹配的账户。 ")).TextColor(t.TextMuted)
	}
	visibleRows := visibleAccountKeys(accountTree, &m.reportTree, visibleRootKeys)
	if m.statementSelectedAccount != "" {
		m.statementSelection = slices.Index(visibleRows, m.statementSelectedAccount)
		if m.statementSelection < 0 {
			if _, exists := widgets.NewAccountTree(tree.RootNodes, "").Node(m.statementSelectedAccount); !exists {
				m.statementSelectedAccount = ""
			}
		}
	}
	m.reportTree.List.Selected = &m.statementSelection
	m.reportTree.List.Label = func(row int) string { return m.reportTree.Item(row) }
	ui.Row(c).Gap(8).Children(func() {
		if ui.Button(c, m.tr("全部展开")).Clicked() {
			accountTree.SetExpanded(&m.reportTree, true)
			m.saveStatementSearchOpenSnapshot(treeContext)
		}
		if ui.Button(c, m.tr("全部折叠")).Clicked() {
			accountTree.SetExpanded(&m.reportTree, false)
			m.saveStatementSearchOpenSnapshot(treeContext)
		}
	})
	rows := ui.OutlineTable(c, &m.reportTree, columns, visibleRootKeys, accountTree.Children, func(name string, col int) {
		account, ok := accountTree.Node(name)
		if !ok {
			ui.Text(c, "—").TextColor(t.TextMuted)
			return
		}
		if col == 0 {
			ui.Text(c, account.Name).SingleLine()
			return
		}
		if col == len(columns)-1 {
			if ui.Button(c, m.tr("查看")).Label(m.tr("查看账户 ") + name).Clicked() {
				m.openAccountDetail(name, response.Request.View)
			}
			return
		}
		if col == 1 {
			m.statementAmountColumn(c, account.DirectAmounts, currencies, account.DirectValuation)
		} else {
			m.statementAmountColumn(c, account.SubtreeAmounts, currencies, account.SubtreeValuation)
		}
	}).Label(m.tr("账户余额")).Grow(1)
	if rows.Changed() {
		if m.statementSelection >= 0 && m.statementSelection < m.reportTree.Rows() {
			m.statementSelectedAccount = m.reportTree.Item(m.statementSelection)
		} else {
			m.statementSelectedAccount = ""
		}
	}
	if m.statementSelectedAccount != "" {
		selectedAccount := m.statementSelectedAccount
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, m.tr("已选账户 · ")+selectedAccount).TextColor(t.TextMuted).Grow(1)
			if ui.PrimaryButton(c, m.tr("打开账户详情")).Label(m.tr("查看账户详情 ") + selectedAccount).Clicked() {
				m.openAccountDetail(selectedAccount, response.Request.View)
			}
		})
	}
}

func (m *Model) syncStatementSearchExpansion(tree *widgets.AccountTree, context string) {
	if m.statementSearch == "" {
		if m.statementSearchOpenSaved && m.statementSearchOpenContext == context {
			m.restoreStatementSearchOpenSnapshot()
		}
		m.clearStatementSearchOpenSnapshot()
		return
	}
	if !m.statementSearchOpenSaved || m.statementSearchOpenContext != context {
		m.statementSearchOpenKeys = m.statementSearchOpenKeys[:0]
		for key := range m.reportTree.Open.All() {
			m.statementSearchOpenKeys = append(m.statementSearchOpenKeys, key)
		}
		m.statementSearchOpenSaved = true
		m.statementSearchOpenContext = context
		m.statementSearchOpenQuery = ""
	}
	if m.statementSearchOpenQuery == m.statementSearch {
		return
	}
	m.restoreStatementSearchOpenSnapshot()
	tree.ExpandVisible(&m.reportTree)
	m.statementSearchOpenQuery = m.statementSearch
}

func (m *Model) restoreStatementSearchOpenSnapshot() {
	m.reportTree.Open.Clear()
	for _, key := range m.statementSearchOpenKeys {
		m.reportTree.Open.Add(key)
	}
}

func (m *Model) saveStatementSearchOpenSnapshot(context string) {
	if !m.statementSearchOpenSaved || m.statementSearchOpenContext != context {
		return
	}
	m.statementSearchOpenKeys = m.statementSearchOpenKeys[:0]
	for key := range m.reportTree.Open.All() {
		m.statementSearchOpenKeys = append(m.statementSearchOpenKeys, key)
	}
}

func (m *Model) clearStatementSearchOpenSnapshot() {
	m.statementSearchOpenSaved = false
	m.statementSearchOpenContext = ""
	m.statementSearchOpenKeys = nil
	m.statementSearchOpenQuery = ""
}

func visibleAccountKeys(tree *widgets.AccountTree, state *ui.OutlineState[string], roots []string) []string {
	keys := make([]string, 0)
	var walk func([]string)
	walk = func(items []string) {
		for _, item := range items {
			keys = append(keys, item)
			if state.Open.Has(item) {
				walk(tree.Children(item))
			}
		}
	}
	walk(roots)
	return keys
}

func availabilityText(availability string) string {
	switch availability {
	case "priced":
		return "估值完整"
	case "native-multi":
		return "各原币种分别显示"
	case "unavailable-currency":
		return "缺少本地报价，估值不完整"
	case "partial":
		return "部分估值缺少本地报价"
	case "unavailable-price":
		return "缺少本地价格，无法绘图"
	case "select-currency":
		return "请先选择估值币种"
	case "empty":
		return "当前期间没有活动"
	default:
		return "估值状态未知"
	}
}

func (m *Model) openAccountDetail(account, returnView string) {
	if account == "" || m.workspace == nil {
		return
	}
	m.accountReturnPage = pageBalance
	if returnView == "income_statement" {
		m.accountReturnPage = pageIncome
	}
	m.accountReturnResponse = m.workspaceState.Response
	m.accountReturnSnapshotID = m.workspaceState.Ledger.SnapshotID
	m.statementSelectedAccount = account
	for row := 0; row < m.reportTree.Rows(); row++ {
		if m.reportTree.Item(row) == account {
			m.statementSelection = row
			break
		}
	}
	m.accountName = account
	m.accountScope = "subtree"
	m.accountJournalSearch = ""
	m.accountJournalSearchDraft = ""
	m.accountJournalOffset = 0
	m.accountJournalSelection = -1
	m.accountJournalSelectedID = ""
	m.requestView("account")
}

func (m *Model) returnFromAccount() {
	if m.workspace != nil && m.accountReturnResponse != nil {
		current := m.workspace.State()
		if current.Ledger.SnapshotID == m.accountReturnSnapshotID {
			m.workspaceState.Response = m.accountReturnResponse
		}
	}
	if len(m.pageHistoryBack) > 0 && m.pageHistoryBack[len(m.pageHistoryBack)-1] == m.accountReturnPage {
		m.navigateBack()
		return
	}
	view := "balance_sheet"
	if m.accountReturnPage == pageIncome {
		view = "income_statement"
	}
	m.requestView(view)
}

func (m *Model) accountPage(c *ui.Context) {
	t := c.Theme()
	response := m.responseFor("account")
	if response == nil || response.Account == nil || !m.accountResponseMatchesFilters(response) {
		message := "正在读取账户详情…"
		if !m.workspaceState.Busy {
			message = "尚无符合当前筛选条件的账户详情"
		}
		ui.Text(c, m.tr(message)).TextColor(t.TextMuted)
		return
	}
	detail := response.Account.AccountDetailProjection
	if detail == nil {
		ui.Text(c, m.tr("核心没有返回账户详情。 ")).TextColor(t.TextMuted)
		return
	}
	ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
		if ui.Button(c, m.tr("返回报表")).Clicked() {
			m.returnFromAccount()
		}
		ui.Text(c, detail.Account).FontSize(18).Bold()
		ui.Spacer(c)
		ui.Text(c, m.tr(accountLifecycleLabel(detail.Lifecycle))).TextColor(t.TextMuted)
	})
	scopeIndex := 1
	if m.accountScope == "direct" {
		scopeIndex = 0
	}
	if ui.Segmented(c, &scopeIndex, m.tr("仅此账户"), m.tr("包含子账户")).Label(m.tr("账户范围")).Changed() {
		if scopeIndex == 0 {
			m.accountScope = "direct"
		} else {
			m.accountScope = "subtree"
		}
		m.accountJournalOffset = 0
		m.accountJournalSelection = -1
		m.accountJournalSelectedID = ""
		m.requestView("account")
	}
	ui.Row(c).Gap(14).Children(func() {
		if detail.Explicit {
			ui.Text(c, m.tr("明确开户 · ")+detail.Opened).TextColor(t.TextMuted)
		} else {
			ui.Text(c, m.tr("目录账户 · 无直接开户资料")).TextColor(t.TextMuted)
		}
		if detail.Closed != nil {
			ui.Text(c, m.tr("关闭日期 ")+*detail.Closed).TextColor(t.TextMuted)
		}
		if detail.Booking != "" {
			ui.Text(c, m.tr("记账方式 ")+detail.Booking).TextColor(t.TextMuted)
		}
		if detail.DateFrom != "" || detail.DateTo != "" {
			ui.Text(c, m.tr("期间 ")+detail.DateFrom+" – "+detail.DateTo).TextColor(t.TextMuted)
		}
	})
	currencies := accountDetailCurrencies(detail)
	if len(currencies) > 0 {
		columns := []ui.TableColumn{
			{Title: m.tr("币种"), ID: "currency", Width: 100, Fixed: true},
			{Title: m.tr("账户余额"), ID: "direct"},
			{Title: m.tr("含子账户"), ID: "subtree"},
			{Title: m.tr("期间变化"), ID: "change"},
		}
		ui.Table(c, &m.accountAmountTable, columns, len(currencies), func(row, col int) {
			currency := currencies[row]
			switch col {
			case 0:
				ui.Text(c, currency)
			case 1:
				ui.Text(c, accountAmountText(detail.DirectAmounts, currency))
			case 2:
				ui.Text(c, accountAmountText(detail.SubtreeAmounts, currency))
			case 3:
				ui.Text(c, accountAmountText(detail.PeriodChange, currency))
			}
		}).Height(120)
	}
	for _, valuation := range []struct {
		label string
		value *report.AccountValuation
	}{{"账户估值", detail.DirectValuation}, {"含子账户估值", detail.SubtreeValuation}, {"期间估值", detail.PeriodValuation}} {
		if valuation.value == nil {
			continue
		}
		value := valuation.value
		ui.Row(c).Gap(10).Children(func() {
			ui.Text(c, valuation.label).Bold()
			if value.CompleteTotal != nil {
				ui.Text(c, value.CompleteTotal.Display)
			} else {
				ui.Text(c, m.tr(valuationCompletenessText(value.Completeness))).TextColor(t.TextMuted)
			}
			for _, missing := range value.Missing {
				ui.Text(c, missing.Commodity+"："+m.tr(valuationReasonText(missing.Reason))).TextColor(t.Danger)
			}
			for _, quote := range value.Quotes {
				path := response.Account.SourceFiles[quote.Source.File]
				source := report.ProjectionSource{Path: path, Line: quote.Source.StartLine, Column: quote.Source.StartColumn}
				ui.Row(c).Gap(8).Children(func() {
					kind := m.tr("直接报价")
					if quote.Inverse {
						kind = m.tr("反向报价")
					}
					ui.Text(c, m.tr("市价来源 · ")+quote.Date+" "+quote.Rate.Exact+" "+quote.Quote+" · "+kind+" · "+sourceLabel(source)).TextColor(t.TextMuted).Grow(1)
					if path != "" && ui.Button(c, m.tr("显示估值报价来源")).Clicked() {
						m.revealLocation(path)
					}
					m.sourceEditorButton(c, path)
				})
			}
		})
	}
	if len(detail.History) > 0 {
		ui.Text(c, m.tr("历史余额")).FontSize(15).Bold()
		for _, history := range detail.History {
			history := history
			points := make([]charts.Point, 0, len(history.Points))
			for _, point := range history.Points {
				display, available := m.tr("缺失估值"), false
				value := 0.0
				if point.Valuation != nil && point.Valuation.CompleteTotal != nil && point.Value != nil {
					display = point.Valuation.CompleteTotal.Exact + " " + history.Currency
					value = *point.Value
					available = true
				} else if point.Valuation != nil && len(point.Valuation.Missing) > 0 {
					display = point.Valuation.Missing[0].Commodity + "：" + m.tr(valuationReasonText(point.Valuation.Missing[0].Reason))
				}
				points = append(points, charts.Point{Label: point.Month, Value: value, Exact: display, Available: available})
			}
			series := charts.Series{Label: history.Currency + " · " + m.tr(history.Mode), Color: ui.RGB(42, 119, 214), Points: points}
			LineChart(c, series, m.tr)
			ExactTable(c, m.chartTableFor("account:"+detail.Account+":"+detail.Scope+":"+history.Currency), []charts.Series{series}, m.tr)
		}
	}
	if len(response.Account.Sources) > 0 {
		ui.Text(c, m.tr("账户生命周期来源")).FontSize(15).Bold()
		ui.Column(c).Gap(4).Children(func() {
			for _, source := range response.Account.Sources {
				source := source
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, m.tr(sourceKindLabel(source.Kind))+" · "+source.Date).FontSize(12)
					ui.Text(c, directiveLocationLabel(source.Source)).FontSize(11).TextColor(t.TextMuted).SingleLine().Grow(1)
					if ui.Button(c, m.tr("在 Finder 中显示")).Clicked() {
						m.revealLocation(source.Source.Path)
					}
					m.sourceEditorButton(c, source.Source.Path)
				})
			}
		})
	}
	m.accountJournalPage(c, response.Journal)
}

func (m *Model) accountJournalPage(c *ui.Context, journal *desktop.JournalPayload) {
	t := c.Theme()
	ui.Text(c, m.tr("账户流水")).FontSize(15).Bold()
	ui.Row(c).Gap(8).Children(func() {
		search := ui.TextInput(c, &m.accountJournalSearchDraft).Placeholder(m.tr("交易文字、标签或金额")).Label(m.tr("账户流水搜索")).Grow(1)
		m.focusSearchInput(pageAccount, search)
		if ui.PrimaryButton(c, m.tr("筛选账户流水")).Disabled(m.workspaceState.Busy).Clicked() {
			if len(m.accountJournalSearchDraft) > 65536 {
				m.status = "账户流水搜索文字超过 64 KiB 上限。"
			} else {
				m.accountJournalSearch = strings.TrimSpace(m.accountJournalSearchDraft)
				m.accountJournalOffset = 0
				m.accountJournalSelectedID = ""
				m.accountJournalSelection = -1
				m.requestView("account")
			}
		}
	})
	if journal == nil {
		ui.Text(c, m.tr("尚无匹配的账户交易。 ")).TextColor(t.TextMuted)
		return
	}
	if len(journal.Transactions) == 0 {
		ui.Text(c, m.tr("当前筛选没有账户交易。 ")).TextColor(t.TextMuted)
		return
	}
	ui.Text(c, fmt.Sprintf(m.tr("%d 笔账户交易 · 显示 %d–%d"), journal.TotalCount, journal.Offset+1, journal.Offset+len(journal.Transactions))).TextColor(t.TextMuted)
	txs := journal.Transactions
	selected := -1
	for index := range txs {
		if txs[index].ID == m.accountJournalSelectedID {
			selected = index
			break
		}
	}
	if selected < 0 && m.accountJournalSelectedID == "" && len(txs) > 0 {
		selected = 0
		m.accountJournalSelectedID = txs[0].ID
	}
	m.accountJournalSelection = selected
	m.accountJournalTable.Selected = &m.accountJournalSelection
	m.accountJournalTable.Key = func(row int) any { return txs[row].ID }
	m.accountJournalTable.Label = func(row int) string {
		return strings.TrimSpace(txs[row].Date + " " + txs[row].Payee + " " + txs[row].Narration)
	}
	columns := []ui.TableColumn{{Title: m.tr("日期"), ID: "date", Width: 110, Fixed: true}, {Title: m.tr("交易"), ID: "description"}, {Title: m.tr("分录"), ID: "postings", Width: 80, Align: ui.End}}
	rows := ui.Table(c, &m.accountJournalTable, columns, len(txs), func(row, col int) {
		tx := txs[row]
		switch col {
		case 0:
			ui.Text(c, tx.Date)
		case 1:
			ui.Text(c, strings.TrimSpace(tx.Payee+" · "+tx.Narration)).SingleLine()
		case 2:
			ui.Text(c, fmt.Sprint(len(tx.Postings)))
		}
	}).Height(190)
	if rows.Changed() && m.accountJournalSelection >= 0 && m.accountJournalSelection < len(txs) {
		m.accountJournalSelectedID = txs[m.accountJournalSelection].ID
	}
	ui.Row(c).Gap(8).Children(func() {
		if ui.Button(c, m.tr("账户上一页")).Disabled(journal.Offset == 0 || m.workspaceState.Busy).Clicked() {
			m.accountJournalOffset = max(0, journal.Offset-journal.Limit)
			m.accountJournalSelectedID, m.accountJournalSelection = "", -1
			m.requestView("account")
		}
		ui.Spacer(c)
		if ui.Button(c, m.tr("账户下一页")).Disabled(!journal.HasMore || m.workspaceState.Busy).Clicked() {
			m.accountJournalOffset = journal.Offset + journal.Limit
			m.accountJournalSelectedID, m.accountJournalSelection = "", -1
			m.requestView("account")
		}
	})
	if selected >= 0 && selected < len(txs) {
		tx := txs[selected]
		ui.Text(c, tx.Date+" · "+tx.Narration).FontSize(13).Bold()
		if len(tx.Tags)+len(tx.Links) > 0 {
			labels := make([]string, 0, len(tx.Tags)+len(tx.Links))
			for _, tag := range tx.Tags {
				labels = append(labels, "#"+tag)
			}
			for _, link := range tx.Links {
				labels = append(labels, "^"+link)
			}
			ui.Text(c, strings.Join(labels, "   ")).TextColor(t.Accent)
		}
		if tx.Source.Path != "" {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, m.tr("交易来源 · ")+directiveLocationLabel(tx.Source)).FontSize(11).TextColor(t.TextMuted).SingleLine().Grow(1)
				if ui.Button(c, m.tr("显示交易来源")).Clicked() {
					m.revealLocation(tx.Source.Path)
				}
				m.sourceEditorButton(c, tx.Source.Path)
			})
		}
		for _, metadata := range tx.Metadata {
			metadata := metadata
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, metadata.Key+"："+formatDirectiveValue(metadata.Value)).FontSize(12).Selectable().Grow(1)
				if metadata.Source.Path != "" {
					ui.Text(c, directiveLocationLabel(metadata.Source)).FontSize(11).TextColor(t.TextMuted).SingleLine()
					if ui.Button(c, m.tr("显示")).Clicked() {
						m.revealLocation(metadata.Source.Path)
					}
				}
			})
		}
		postingColumns := []ui.TableColumn{{Title: m.tr("账户"), ID: "account"}, {Title: m.tr("单位"), ID: "units", Width: 150, Align: ui.End}, {Title: m.tr("成本"), ID: "cost", Width: 170}, {Title: m.tr("价格"), ID: "price", Width: 150}, {Title: m.tr("元数据"), ID: "metadata", Width: 220}, {Title: m.tr("来源"), ID: "source", Width: 140}}
		ui.Table(c, &m.postingTable, postingColumns, len(tx.Postings), func(row, col int) {
			posting := tx.Postings[row]
			switch col {
			case 0:
				ui.Text(c, posting.Account).SingleLine()
			case 1:
				ui.Text(c, posting.UnitsNumber+" "+posting.UnitsCurrency).Selectable()
			case 2:
				if posting.Cost == nil {
					ui.Text(c, "—")
				} else {
					ui.Text(c, posting.Cost.Display).Selectable()
				}
			case 3:
				if posting.Price == nil {
					ui.Text(c, "—")
				} else {
					prefix := "@ "
					if posting.Price.Total {
						prefix = "@@ "
					}
					ui.Text(c, prefix+posting.Price.Number.Exact+" "+posting.Price.Currency).Selectable()
				}
			case 4:
				if len(posting.Metadata) == 0 {
					ui.Text(c, "—").TextColor(t.TextMuted)
				} else {
					ui.Column(c).Gap(2).Children(func() {
						for _, metadata := range posting.Metadata {
							ui.Text(c, metadata.Key+"："+formatDirectiveValue(metadata.Value)).FontSize(11).Selectable()
							if metadata.Source.Path != "" {
								ui.Text(c, directiveLocationLabel(metadata.Source)).FontSize(10).TextColor(t.TextMuted).SingleLine()
							}
						}
					})
				}
			case 5:
				if posting.Source.Path == "" {
					ui.Text(c, "—").TextColor(t.TextMuted)
				} else {
					ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
						ui.Text(c, directiveLocationLabel(posting.Source)).FontSize(10).TextColor(t.TextMuted).SingleLine().Grow(1)
						if ui.Button(c, m.tr("显示")).Clicked() {
							m.revealLocation(posting.Source.Path)
						}
					})
				}
			}
		}).Height(160)
	}
}

func (m *Model) accountResponseMatchesFilters(response *desktop.ViewResponse) bool {
	request := response.Request
	windowMatches := request.Window == m.filterWindow || (m.filterWindow == "all" && request.Window == "")
	return request.View == "account" && request.Account == m.accountName && request.Scope == m.accountScope &&
		request.Text == m.accountJournalSearch && request.Offset == m.accountJournalOffset && request.Limit == 100 &&
		request.DateFrom == m.filterDateFrom && request.DateTo == m.filterDateTo && windowMatches &&
		request.Currency == m.filterCurrency && request.Valuation == m.filterValuation &&
		(response.State.SnapshotID == "" || request.SnapshotID == response.State.SnapshotID) &&
		(m.workspaceState.Ledger.SnapshotID == "" || request.SnapshotID == m.workspaceState.Ledger.SnapshotID)
}

func accountLifecycleLabel(value string) string {
	switch value {
	case "open":
		return "已开户"
	case "closed":
		return "已关闭"
	case "not-opened":
		return "尚未开户"
	case "implicit":
		return "目录账户"
	default:
		return value
	}
}

func sourceKindLabel(value string) string {
	switch value {
	case "open":
		return "开户"
	case "close":
		return "关闭"
	default:
		return value
	}
}

func accountDetailCurrencies(detail *report.AccountDetailProjection) []string {
	set := make(map[string]bool)
	for _, currency := range detail.Currencies {
		set[currency] = true
	}
	for _, values := range []map[string]report.PresentedDecimal{detail.DirectAmounts, detail.SubtreeAmounts, detail.PeriodChange} {
		for currency := range values {
			set[currency] = true
		}
	}
	currencies := make([]string, 0, len(set))
	for currency := range set {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)
	return currencies
}

func accountAmountText(values map[string]report.PresentedDecimal, currency string) string {
	amount, ok := values[currency]
	if !ok {
		return "—"
	}
	return widgets.AmountText(amount)
}

func (m *Model) pricesPage(c *ui.Context) {
	t := c.Theme()
	response := m.responseFor("prices")
	if response == nil || response.Prices == nil || !m.priceResponseMatchesFilters(response) {
		message := "正在读取商品与价格…"
		if !m.workspaceState.Busy {
			message = "尚无符合当前日期范围的价格数据"
		}
		ui.Text(c, m.tr(message)).TextColor(t.TextMuted)
		return
	}
	payload := response.Prices
	sectionIndex := 0
	if m.priceSection == "history" {
		sectionIndex = 1
	}
	if ui.Segmented(c, &sectionIndex, m.tr("商品目录"), m.tr("价格历史")).Label(m.tr("价格页面")).Changed() {
		if sectionIndex == 0 {
			m.priceSection = "commodities"
		} else {
			m.priceSection = "history"
			m.priceOffset = 0
			m.priceSelection, m.priceSelectedID = -1, ""
		}
		m.requestView("prices")
		return
	}
	if m.priceError != "" {
		ui.Text(c, m.tr(m.priceError)).TextColor(t.Danger)
	}
	if m.priceSection == "commodities" {
		m.commodityCatalogPage(c, payload)
		return
	}
	selectedCommodity := payload.Commodity
	if selectedCommodity == nil || selectedCommodity.Currency != m.priceBase {
		ui.Text(c, m.tr("当前快照中没有所选商品。 ")).TextColor(t.Danger)
		return
	}
	quoteOptions := selectedCommodity.QuoteCurrencies
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, m.tr("商品 ")+selectedCommodity.Currency).Bold()
		if ui.Button(c, m.tr("更换商品")).Clicked() {
			m.priceSection = "commodities"
			m.requestView("prices")
			return
		}
		ui.Text(c, m.tr("报价币种")).TextColor(t.TextMuted)
		if len(quoteOptions) == 0 {
			ui.Text(c, m.tr("无本地报价对"))
		} else if m.selectOptions(c, &m.priceQuote, quoteOptions, nil).Label(m.tr("选择报价币种")).Changed() {
			m.priceOffset = 0
			m.priceSelection, m.priceSelectedID = -1, ""
			m.requestView("prices")
		}
		sortIndex := 0
		if m.priceDescending {
			sortIndex = 1
		}
		if ui.Segmented(c, &sortIndex, m.tr("旧到新"), m.tr("新到旧")).Label(m.tr("价格历史排序")).Changed() {
			m.priceDescending = sortIndex == 1
			m.priceOffset = 0
			m.requestView("prices")
		}
	})
	m.priceHistoryPage(c, payload)
}

func (m *Model) commodityCatalogPage(c *ui.Context, payload *report.PricesProjection) {
	t := c.Theme()
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		search := ui.TextInput(c, &m.commoditySearchDraft).Placeholder(m.tr("商品、账户或报价币种")).Label(m.tr("搜索商品目录")).Grow(1)
		m.focusSearchInput(pagePrices, search)
		if ui.PrimaryButton(c, m.tr("筛选商品")).Disabled(m.workspaceState.Busy).Clicked() {
			m.applyCommoditySearch()
		}
		if ui.Button(c, m.tr("清除商品搜索")).Disabled(m.commoditySearch == "" && m.commoditySearchDraft == "").Clicked() {
			m.commoditySearch, m.commoditySearchDraft, m.commoditySearchError = "", "", ""
			m.commodityOffset = 0
			m.requestView("prices")
		}
	})
	if m.commoditySearchError != "" {
		ui.Text(c, m.tr(m.commoditySearchError)).TextColor(t.Danger)
	}
	page := payload.CommoditiesPage
	start, end := 0, 0
	if page.ReturnedCount > 0 {
		start, end = page.Offset+1, page.Offset+page.ReturnedCount
	}
	ui.Text(c, fmt.Sprintf(m.tr("%d 个商品 · 显示 %d–%d"), page.TotalCount, start, end)).TextColor(t.TextMuted)
	commodities := payload.Commodities
	if m.commoditySelectedID == "" {
		for _, item := range commodities {
			if item.Currency == m.priceBase {
				m.commoditySelectedID = item.ID
				break
			}
		}
		if m.commoditySelectedID == "" && len(commodities) > 0 {
			m.commoditySelectedID = commodities[0].ID
		}
	}
	if len(commodities) > 0 {
		for _, item := range commodities {
			if item.ID != m.commoditySelectedID {
				continue
			}
			m.priceBase = item.Currency
			if !slices.Contains(item.QuoteCurrencies, m.priceQuote) {
				m.priceQuote = ""
				if len(item.QuoteCurrencies) > 0 {
					m.priceQuote = item.QuoteCurrencies[0]
				}
			}
			break
		}
	}
	intCompare := func(a, b int) int {
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	}
	m.commodityTable.Sort = &m.commoditySort
	m.commodityTable.Selected = &m.commoditySelection
	table, err := widgets.NewStableTable(&m.commodityTable, commodities,
		func(item report.CommodityInfo) string { return item.ID },
		func(item report.CommodityInfo) string { return item.Currency },
		[]widgets.TableColumn[report.CommodityInfo]{
			{Column: ui.TableColumn{Title: m.tr("商品"), ID: "currency", Width: 140, Fixed: true, Sortable: true}, Compare: func(a, b report.CommodityInfo) int { return strings.Compare(a.Currency, b.Currency) }, Cell: func(c *ui.Context, item report.CommodityInfo) { ui.Text(c, item.Currency) }},
			{Column: ui.TableColumn{Title: m.tr("账户"), ID: "accounts", Sortable: true}, Compare: func(a, b report.CommodityInfo) int {
				return strings.Compare(strings.Join(a.Accounts, "\x00"), strings.Join(b.Accounts, "\x00"))
			}, Cell: func(c *ui.Context, item report.CommodityInfo) {
				ui.Text(c, strings.Join(item.Accounts, ", ")).SingleLine()
			}},
			{Column: ui.TableColumn{Title: m.tr("报价对"), ID: "quotes", Sortable: true}, Compare: func(a, b report.CommodityInfo) int {
				return strings.Compare(strings.Join(a.QuoteCurrencies, "\x00"), strings.Join(b.QuoteCurrencies, "\x00"))
			}, Cell: func(c *ui.Context, item report.CommodityInfo) {
				ui.Text(c, strings.Join(item.QuoteCurrencies, ", ")).SingleLine()
			}},
			{Column: ui.TableColumn{Title: m.tr("价格数"), ID: "prices", Width: 90, Align: ui.End, Sortable: true}, Compare: func(a, b report.CommodityInfo) int { return intCompare(a.PriceCount, b.PriceCount) }, Cell: func(c *ui.Context, item report.CommodityInfo) { ui.Text(c, fmt.Sprint(item.PriceCount)) }},
			{Column: ui.TableColumn{Title: m.tr("声明"), ID: "declarations", Width: 80, Align: ui.End, Sortable: true}, Compare: func(a, b report.CommodityInfo) int { return intCompare(len(a.Declarations), len(b.Declarations)) }, Cell: func(c *ui.Context, item report.CommodityInfo) { ui.Text(c, fmt.Sprint(len(item.Declarations))) }},
		})
	if err != nil {
		ui.Text(c, m.tr("商品表格不可用：")+err.Error()).TextColor(t.Danger)
		return
	}
	selectedIndex := table.IndexOf(m.commoditySelectedID)
	if selectedIndex < 0 && len(commodities) > 0 {
		selectedIndex = 0
		m.commoditySelectedID = table.IDAt(selectedIndex)
	}
	m.commoditySelection = selectedIndex
	rows := table.Render(c).Grow(1)
	if rows.Changed() {
		item, ok := table.At(m.commoditySelection)
		if ok {
			m.commoditySelectedID = item.ID
			m.priceBase = item.Currency
			m.priceQuote = ""
			if len(item.QuoteCurrencies) > 0 {
				m.priceQuote = item.QuoteCurrencies[0]
			}
		}
	}
	selected := table.IndexOf(m.commoditySelectedID)
	if selected >= 0 {
		item, _ := table.At(selected)
		ui.Column(c).Padding(12).Gap(8).Background(t.Surface).Border(1, t.Border).Radius(8).Children(func() {
			ui.Text(c, item.Currency+m.tr(" · 商品详情")).FontSize(16).Bold()
			if len(item.Accounts) > 0 {
				ui.Text(c, m.tr("使用账户 · ")+strings.Join(item.Accounts, ", ")).TextColor(t.TextMuted)
			}
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				if len(item.QuoteCurrencies) == 0 {
					ui.Text(c, m.tr("没有本地报价对")).TextColor(t.TextMuted)
				} else {
					ui.Text(c, m.tr("报价币种")).TextColor(t.TextMuted)
					if m.selectOptions(c, &m.priceQuote, item.QuoteCurrencies, nil).Label(m.tr("选择报价币种")).Changed() {
						m.priceBase = item.Currency
					}
				}
				if ui.PrimaryButton(c, m.tr("查看历史")).Disabled(len(item.QuoteCurrencies) == 0).Clicked() {
					m.priceBase = item.Currency
					m.priceSection = "history"
					m.priceOffset = 0
					m.priceSelection, m.priceSelectedID = -1, ""
					m.requestView("prices")
				}
			})
			if len(item.Declarations) > 0 {
				ui.Text(c, m.tr("商品声明")).Bold()
				for _, declaration := range item.Declarations {
					declaration := declaration
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, declaration.Date+" · "+sourceLabel(declaration.Source)).TextColor(t.TextMuted).Grow(1)
						if declaration.Source.Path != "" && ui.Button(c, m.tr("显示商品声明来源")).Clicked() {
							m.revealLocation(declaration.Source.Path)
						}
						m.sourceEditorButton(c, declaration.Source.Path)
					})
				}
			}
		})
	}
	ui.Row(c).Gap(8).Children(func() {
		if ui.Button(c, m.tr("商品上一页")).Disabled(page.Offset == 0 || m.workspaceState.Busy).Clicked() {
			m.commodityOffset = max(0, page.Offset-page.Limit)
			m.requestView("prices")
		}
		ui.Spacer(c)
		if ui.Button(c, m.tr("商品下一页")).Disabled(!page.HasMore || m.workspaceState.Busy).Clicked() {
			m.commodityOffset = page.Offset + page.Limit
			m.requestView("prices")
		}
	})
	if len(commodities) == 0 {
		ui.Text(c, m.tr("当前账本没有商品或价格声明。 ")).TextColor(t.TextMuted)
	}
}

func (m *Model) priceHistoryPage(c *ui.Context, payload *report.PricesProjection) {
	t := c.Theme()
	if payload.Base == "" || payload.Quote == "" {
		ui.Text(c, m.tr("选择商品和报价币种后查看历史。 ")).TextColor(t.TextMuted)
		return
	}
	if payload.Selected != nil {
		selected := payload.Selected
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			kind := "直接报价"
			if selected.Inverse {
				kind = "反向报价"
			}
			ui.Text(c, m.tr("截至 ")+payload.DateTo+" · "+selected.Rate.Display+" "+payload.Quote+" · "+m.tr(kind)).Bold()
			ui.Text(c, fmt.Sprintf(m.tr("来源 %s/%s · %s · %d 天前"), selected.SourceBase, selected.SourceQuote, selected.Date, selected.AgeDays)).TextColor(t.TextMuted)
			if selected.Source.Path != "" && ui.Button(c, m.tr("显示所选报价来源")).Clicked() {
				m.revealLocation(selected.Source.Path)
			}
			m.sourceEditorButton(c, selected.Source.Path)
		})
	} else {
		ui.Text(c, m.tr(priceReasonLabel(payload.SelectedReason))).TextColor(t.Danger)
	}
	if payload.DateFrom != "" || payload.DateTo != "" {
		ui.Text(c, m.tr("历史区间 ")+payload.DateFrom+" – "+payload.DateTo).TextColor(t.TextMuted)
	}
	if len(payload.Records) > 0 {
		points := priceChartPoints(payload.Records)
		series := charts.Series{Label: payload.Base + " / " + payload.Quote, Color: ui.RGB(42, 119, 214), Points: points}
		LineChart(c, series, m.tr)
		ExactTable(c, &m.priceExactTable, []charts.Series{series}, m.tr)
	}
	page := payload.RecordsPage
	ui.Text(c, fmt.Sprintf(m.tr("%d 条报价 · 显示 %d–%d"), page.TotalCount, page.Offset+1, page.Offset+page.ReturnedCount)).TextColor(t.TextMuted)
	records := payload.Records
	selected := -1
	for index := range records {
		if records[index].ID == m.priceSelectedID {
			selected = index
			break
		}
	}
	m.priceSelection = selected
	m.priceTable.Selected = &m.priceSelection
	m.priceTable.Key = func(row int) any { return records[row].ID }
	m.priceTable.Label = func(row int) string { return records[row].Date + " " + records[row].Rate.Exact + " " + payload.Quote }
	columns := []ui.TableColumn{{Title: m.tr("日期"), ID: "date", Width: 120, Fixed: true}, {Title: m.tr("价格"), ID: "rate", Width: 180, Align: ui.End}, {Title: m.tr("来源序"), ID: "order", Width: 100, Align: ui.End}, {Title: m.tr("来源段"), ID: "segment", Width: 90, Align: ui.End}, {Title: m.tr("来源"), ID: "source"}}
	rows := ui.Table(c, &m.priceTable, columns, len(records), func(row, col int) {
		record := records[row]
		switch col {
		case 0:
			ui.Text(c, record.Date)
		case 1:
			ui.Text(c, record.Rate.Exact+" "+payload.Quote).Selectable()
		case 2:
			ui.Text(c, fmt.Sprint(record.SourceOrder))
		case 3:
			ui.Text(c, fmt.Sprint(record.Segment+1))
		case 4:
			ui.Text(c, sourceLabel(record.Source)).FontSize(11).SingleLine()
		}
	}).Grow(1)
	if rows.Changed() && m.priceSelection >= 0 && m.priceSelection < len(records) {
		m.priceSelectedID = records[m.priceSelection].ID
	}
	if selected >= 0 && selected < len(records) {
		source := records[selected].Source
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, m.tr("选中来源 · ")+sourceLabel(source)).TextColor(t.TextMuted).Grow(1)
			if ui.Button(c, m.tr("显示报价来源")).Clicked() {
				m.revealLocation(source.Path)
			}
			m.sourceEditorButton(c, source.Path)
		})
	}
	ui.Row(c).Gap(8).Children(func() {
		if ui.Button(c, m.tr("报价上一页")).Disabled(page.Offset == 0 || m.workspaceState.Busy).Clicked() {
			m.priceOffset = max(0, page.Offset-page.Limit)
			m.priceSelectedID, m.priceSelection = "", -1
			m.requestView("prices")
		}
		ui.Spacer(c)
		if ui.Button(c, m.tr("报价下一页")).Disabled(!page.HasMore || m.workspaceState.Busy).Clicked() {
			m.priceOffset = page.Offset + page.Limit
			m.priceSelectedID, m.priceSelection = "", -1
			m.requestView("prices")
		}
	})
}

func (m *Model) priceResponseMatchesFilters(response *desktop.ViewResponse) bool {
	request := response.Request
	expectedSearch := ""
	if m.priceSection == "commodities" {
		expectedSearch = m.commoditySearch
	}
	expectedOffset := m.commodityOffset
	expectedSort := "ascending"
	if m.priceDescending {
		expectedSort = "descending"
	}
	if m.priceSection == "history" {
		expectedOffset = m.priceOffset
	}
	windowMatches := request.Window == m.filterWindow || (m.filterWindow == "all" && request.Window == "")
	if request.View != "prices" || request.PriceSection != m.priceSection || request.CommoditySearch != expectedSearch || request.Sort != expectedSort || request.Offset != expectedOffset || request.Limit != 100 || request.DateFrom != m.filterDateFrom || request.DateTo != m.filterDateTo || !windowMatches {
		return false
	}
	if m.priceSection == "history" && (request.PriceBase != m.priceBase || request.PriceQuote != m.priceQuote) {
		return false
	}
	return (request.SnapshotID == "" || request.SnapshotID == response.State.SnapshotID) &&
		(m.workspaceState.Ledger.SnapshotID == "" || request.SnapshotID == m.workspaceState.Ledger.SnapshotID)
}

func priceReasonLabel(reason string) string {
	switch reason {
	case "missing-local-quote":
		return "此日期前没有本地报价"
	case "unspecified-pair":
		return "请选择商品和报价币种"
	default:
		return reason
	}
}

func priceChartPoints(records []report.PriceRecord) []charts.Point {
	points := make([]charts.Point, 0, len(records))
	for _, record := range records {
		value, available := 0.0, record.Value != nil
		if record.Value != nil {
			value = *record.Value
		}
		points = append(points, charts.Point{
			Label: record.Date, Value: value, Exact: record.Rate.Exact + " " + record.Quote,
			Available: available, Segment: record.Segment,
		})
	}
	return points
}

func (m *Model) applyCommoditySearch() {
	if len(m.commoditySearchDraft) > 4096 {
		m.commoditySearchError = "商品搜索文字超过 4 KiB 上限。"
		return
	}
	search := strings.TrimSpace(m.commoditySearchDraft)
	m.commoditySearch = search
	m.commoditySearchError = ""
	m.commodityOffset = 0
	m.commoditySelectedID = ""
	m.commoditySelection = -1
	m.requestView("prices")
}

func sourceLabel(source report.ProjectionSource) string {
	return widgets.SourceText(widgets.SourceLocation{Path: source.Path, Line: source.Line, Column: source.Column})
}

func (m *Model) journalPage(c *ui.Context) {
	t := c.Theme()
	response := m.responseFor("journal")
	if response == nil || response.Journal == nil || !m.journalResponseMatchesFilters(response) {
		message := "正在读取流水…"
		if !m.workspaceState.Busy {
			message = "正在读取符合当前筛选条件的流水…"
		}
		ui.Text(c, m.tr(message)).TextColor(t.TextMuted)
		return
	}
	payload := response.Journal
	txs := payload.Transactions
	accountOptions := []string{"全部账户"}
	if response != nil {
		accountOptions = append(accountOptions, response.Summary.Accounts...)
	}
	selectedAccount := "全部账户"
	if m.journalAccount != "" {
		selectedAccount = m.journalAccount
	}
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		search := ui.TextInput(c, &m.journalSearchDraft).Placeholder(m.tr("交易文字、账户、标签或金额")).Label(m.tr("搜索流水")).Grow(1)
		m.focusSearchInput(pageJournal, search)
		if ui.PrimaryButton(c, m.tr("筛选流水")).Disabled(m.workspaceState.Busy).Clicked() {
			m.applyJournalSearch()
		}
		ui.Text(c, m.tr("账户")).TextColor(t.TextMuted)
		if m.selectOptions(c, &selectedAccount, accountOptions, map[string]string{"全部账户": m.tr("全部账户")}).Changed() {
			if selectedAccount == "全部账户" {
				m.journalAccount = ""
			} else {
				m.journalAccount = selectedAccount
			}
			m.journalOffset = 0
			m.requestView("journal")
		}
		sortIndex := 0
		if !m.journalDescending {
			sortIndex = 1
		}
		if ui.Segmented(c, &sortIndex, m.tr("新到旧"), m.tr("旧到新")).Label(m.tr("流水排序")).Changed() {
			m.journalDescending = sortIndex == 0
			m.journalOffset = 0
			m.requestView("journal")
		}
	})
	if m.journalError != "" {
		ui.Text(c, m.tr(m.journalError)).TextColor(t.Danger)
	}
	start, end := 0, 0
	if len(txs) > 0 {
		start, end = payload.Offset+1, payload.Offset+len(txs)
	}
	ui.Text(c, fmt.Sprintf(m.tr("%d 笔匹配交易 · 显示 %d–%d · 快照 %s"), payload.TotalCount, start, end, shortID(response.State.SnapshotID))).TextColor(t.TextMuted)
	selected := -1
	for index := range txs {
		if txs[index].ID == m.journalSelectedID {
			selected = index
			break
		}
	}
	m.journalSelection = selected
	m.journalTable.Selected = &m.journalSelection
	m.journalTable.Key = func(row int) any { return txs[row].ID }
	m.journalTable.Label = func(row int) string {
		return strings.TrimSpace(txs[row].Date + " " + txs[row].Payee + " " + txs[row].Narration)
	}
	columns := []ui.TableColumn{
		{Title: m.tr("日期"), ID: "date", Width: 110, Fixed: true},
		{Title: m.tr("交易"), ID: "description"},
		{Title: m.tr("分录"), ID: "postings", Width: 90, Align: ui.End},
	}
	rows := ui.Table(c, &m.journalTable, columns, len(txs), func(row, col int) {
		tx := txs[row]
		switch col {
		case 0:
			ui.Text(c, tx.Date)
		case 1:
			description := tx.Narration
			if tx.Payee != "" {
				description = tx.Payee + " · " + description
			}
			ui.Text(c, description).SingleLine()
		case 2:
			ui.Text(c, fmt.Sprintf("%d", len(tx.Postings)))
		}
	}).Height(300)
	if rows.Changed() && m.journalSelection >= 0 && m.journalSelection < len(txs) {
		m.journalSelectedID = txs[m.journalSelection].ID
	}
	ui.Row(c).Gap(8).Children(func() {
		if ui.Button(c, m.tr("上一页")).Disabled(payload.Offset == 0 || m.workspaceState.Busy).Clicked() {
			m.journalOffset = max(0, payload.Offset-payload.Limit)
			m.requestView("journal")
		}
		ui.Spacer(c)
		if ui.Button(c, m.tr("下一页")).Disabled(!payload.HasMore || m.workspaceState.Busy).Clicked() {
			m.journalOffset = payload.Offset + payload.Limit
			m.requestView("journal")
		}
	})
	if m.journalSelection < 0 || m.journalSelection >= len(txs) {
		ui.Text(c, m.tr("选择一笔交易查看分录详情")).TextColor(t.TextMuted)
		return
	}
	tx := txs[m.journalSelection]
	ui.Text(c, strings.TrimSpace(tx.Date+" · "+tx.Payee+" · "+tx.Narration)).FontSize(15).Bold()
	if len(tx.Tags)+len(tx.Links) > 0 {
		labels := make([]string, 0, len(tx.Tags)+len(tx.Links))
		for _, tag := range tx.Tags {
			labels = append(labels, "#"+tag)
		}
		for _, link := range tx.Links {
			labels = append(labels, "^"+link)
		}
		ui.Text(c, strings.Join(labels, "   ")).TextColor(t.Accent)
	}
	if tx.Source.Path != "" {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, m.tr("交易来源 · ")+directiveLocationLabel(tx.Source)).FontSize(11).TextColor(t.TextMuted).SingleLine()
			if ui.Button(c, m.tr("显示交易来源")).Clicked() {
				m.revealLocation(tx.Source.Path)
			}
			m.sourceEditorButton(c, tx.Source.Path)
		})
	}
	for _, metadata := range tx.Metadata {
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, metadata.Key+"："+formatDirectiveValue(metadata.Value)).FontSize(12).Selectable().Grow(1)
			if metadata.Source.Path != "" {
				ui.Text(c, directiveLocationLabel(metadata.Source)).FontSize(11).TextColor(t.TextMuted).SingleLine()
			}
		})
	}
	postingColumns := []ui.TableColumn{
		{Title: m.tr("账户"), ID: "account", Width: 240},
		{Title: m.tr("单位"), ID: "units", Width: 140, Align: ui.End},
		{Title: m.tr("成本"), ID: "cost", Width: 190},
		{Title: m.tr("价格"), ID: "price", Width: 160},
		{Title: m.tr("元数据"), ID: "metadata", Width: 220},
		{Title: m.tr("来源"), ID: "source", Width: 140},
	}
	m.postingTable.Key = func(row int) any { return fmt.Sprintf("%s:%d", tx.ID, row) }
	ui.ScrollHorizontal(c).Children(func() {
		ui.Table(c, &m.postingTable, postingColumns, len(tx.Postings), func(row, col int) {
			posting := tx.Postings[row]
			switch col {
			case 0:
				ui.Text(c, posting.Account).SingleLine()
			case 1:
				if posting.UnitsNumber == "" {
					ui.Text(c, "—").TextColor(t.TextMuted)
				} else {
					ui.Text(c, posting.UnitsNumber+" "+posting.UnitsCurrency).Selectable()
				}
			case 2:
				if posting.Cost == nil {
					ui.Text(c, "—").TextColor(t.TextMuted)
				} else {
					ui.Text(c, posting.Cost.Display).Selectable()
				}
			case 3:
				if posting.Price == nil {
					ui.Text(c, "—").TextColor(t.TextMuted)
				} else {
					prefix := "@ "
					if posting.Price.Total {
						prefix = "@@ "
					}
					ui.Text(c, prefix+posting.Price.Number.Exact+" "+posting.Price.Currency).Selectable()
				}
			case 4:
				if len(posting.Metadata) == 0 {
					ui.Text(c, "—").TextColor(t.TextMuted)
				} else {
					ui.Column(c).Gap(2).Children(func() {
						for _, metadata := range posting.Metadata {
							ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
								ui.Text(c, metadata.Key+"："+formatDirectiveValue(metadata.Value)).FontSize(11).Selectable().Grow(1)
								if metadata.Source.Path != "" {
									ui.Text(c, directiveLocationLabel(metadata.Source)).FontSize(10).TextColor(t.TextMuted).SingleLine()
								}
							})
						}
					})
				}
			case 5:
				if posting.Source.Path == "" {
					ui.Text(c, "—").TextColor(t.TextMuted)
				} else {
					ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
						ui.Text(c, directiveLocationLabel(posting.Source)).FontSize(10).TextColor(t.TextMuted).SingleLine()
						if ui.Button(c, m.tr("在 Finder 中显示来源")).Label(m.tr("在 Finder 中显示来源") + " · " + directiveLocationLabel(posting.Source)).Clicked() {
							m.revealLocation(posting.Source.Path)
						}
						m.sourceEditorButton(c, posting.Source.Path)
					})
				}
			}
		}).Grow(1)
	})
}

func (m *Model) applyJournalSearch() {
	if len(m.journalSearchDraft) > 65536 {
		m.journalError = "流水搜索文字超过 64 KiB 上限。"
		return
	}
	m.journalSearch = strings.TrimSpace(m.journalSearchDraft)
	m.journalOffset = 0
	m.journalError = ""
	m.requestView("journal")
}

func (m *Model) journalResponseMatchesFilters(response *desktop.ViewResponse) bool {
	if response == nil || response.Journal == nil {
		return false
	}
	request := response.Request
	expectedSort := "ascending"
	if m.journalDescending {
		expectedSort = "descending"
	}
	windowMatches := request.Window == m.filterWindow || (m.filterWindow == "all" && request.Window == "")
	return request.View == "journal" && request.JournalMode == "transactions" && request.Text == m.journalSearch &&
		request.Account == m.journalAccount && request.Sort == expectedSort && request.Offset == m.journalOffset && request.Limit == 100 &&
		request.DateFrom == m.filterDateFrom && request.DateTo == m.filterDateTo && windowMatches &&
		(request.SnapshotID == "" || request.SnapshotID == response.State.SnapshotID) &&
		(m.workspaceState.Ledger.SnapshotID == "" || request.SnapshotID == m.workspaceState.Ledger.SnapshotID)
}

func (m *Model) attachmentsPage(c *ui.Context) {
	t := c.Theme()
	response := m.workspaceState.Response
	if response == nil || response.Directives == nil || response.Directives.Mode != "all" {
		ui.Text(c, m.tr("附件声明尚未载入。打开有效账本后会读取其 document 指令。 ")).TextColor(t.TextMuted)
		return
	}
	payload := response.Directives
	ui.Text(c, m.tr("附件路径只由当前账本声明和目录策略解析；打开前会由工作区重新验证文件身份。 ")).TextColor(t.TextMuted)
	if payload.Page.TotalCount == 0 {
		ui.Text(c, m.tr("此账本没有 document 指令。 ")).TextColor(t.TextMuted)
	} else {
		ui.Text(c, fmt.Sprintf(m.tr("%d 项声明 · 第 %d–%d 项"), payload.Page.TotalCount, payload.Page.Offset+1, payload.Page.Offset+len(payload.Records))).TextColor(t.TextMuted)
	}
	if m.attachmentActionNotice != "" {
		ui.Text(c, m.tr(m.attachmentActionNotice)).TextColor(t.Danger).Selectable()
	}
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Gap(10).Children(func() {
			for _, record := range payload.Records {
				record := record
				if len(record.Attachments) == 0 {
					continue
				}
				ui.Column(c).Gap(8).Padding(12).Background(t.Surface).Border(1, t.Border).Radius(8).Children(func() {
					date := "无日期"
					if record.Date != nil {
						date = *record.Date
					}
					ui.Text(c, date+" · "+record.Source.Path+":"+fmt.Sprint(record.Source.Line)).FontSize(12).TextColor(t.TextMuted).SingleLine()
					for _, attachment := range record.Attachments {
						attachment := attachment
						ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
							ui.Text(c, filepath.Base(attachment.Filename)).Grow(1).SingleLine()
							status := attachmentStatusLabel(attachment.Status)
							color := t.TextMuted
							if attachment.Status == documents.Available {
								color = t.Success
							}
							ui.Text(c, m.tr(status)).TextColor(color)
							canAct := attachment.Status == documents.Available && !m.workspaceState.Busy && !m.workspaceState.AttachmentRootsBusy && !m.rootPolicySyncBusy && m.pendingAttachmentAction == nil
							if ui.Button(c, m.tr("打开")).Disabled(!canAct).Clicked() {
								m.requestAttachmentAction(attachment, false)
							}
							if ui.Button(c, m.tr("在 Finder 中显示")).Disabled(!canAct).Clicked() {
								m.requestAttachmentAction(attachment, true)
							}
						})
					}
				})
			}
		})
	})
	ui.Row(c).Gap(10).Children(func() {
		if ui.Button(c, m.tr("上一页")).Disabled(payload.Page.Offset == 0 || m.workspaceState.Busy).Clicked() {
			m.attachmentOffset = max(0, payload.Page.Offset-payload.Page.Limit)
			m.requestView("attachments")
		}
		ui.Spacer(c)
		if ui.Button(c, m.tr("下一页")).Disabled(!payload.Page.HasMore || m.workspaceState.Busy).Clicked() {
			m.attachmentOffset = payload.Page.Offset + payload.Page.Limit
			m.requestView("attachments")
		}
	})
}

func attachmentStatusLabel(status documents.Status) string {
	switch status {
	case documents.Available:
		return "可用"
	case documents.Missing:
		return "文件缺失"
	case documents.OutsideRoot:
		return "不在允许的附件目录中"
	case documents.UnsupportedKind:
		return "不是普通文件"
	case documents.Unreadable:
		return "无法访问或路径无效"
	default:
		return "不可用"
	}
}

func (m *Model) requestAttachmentAction(attachment desktop.AttachmentDTO, reveal bool) {
	response := m.workspaceState.Response
	if response == nil || response.Directives == nil || m.workspace == nil || m.workspaceState.Busy || m.workspaceState.AttachmentRootsBusy || m.rootPolicySyncBusy || m.pendingAttachmentAction != nil {
		return
	}
	if attachment.Status != documents.Available || attachment.ID == "" || attachment.PolicyRevision == 0 || response.Directives.SnapshotID == "" || response.Directives.SnapshotID != m.workspaceState.Ledger.SnapshotID {
		m.attachmentActionNotice = "此附件引用已失效或不可用；请重新读取列表。"
		return
	}
	current := m.workspace.State()
	if current.Epoch != m.workspaceState.Epoch || current.Ledger.SnapshotID != response.Directives.SnapshotID || !current.AttachmentRootsLoaded || current.AttachmentRoots.PolicyRevision != attachment.PolicyRevision {
		m.attachmentActionNotice = "账本或附件目录策略已变化；请重新读取列表。"
		return
	}
	request := desktop.Request{
		View: "journal", JournalMode: "all", DirectiveKind: "document", Locale: m.requestLocale(),
		SnapshotID: response.Directives.SnapshotID, DocumentID: attachment.ID,
		DocumentPolicyRevision: attachment.PolicyRevision,
	}
	m.pendingAttachmentAction = &attachmentActionIntent{
		id: attachment.ID, snapshotID: response.Directives.SnapshotID, ledgerPath: m.workspaceState.ActivePath,
		policyRevision: attachment.PolicyRevision, epoch: m.workspaceState.Epoch, reveal: reveal,
	}
	if err := m.workspace.Dispatch(workspace.Intent{Kind: workspace.IntentRead, Request: request}); err != nil {
		m.pendingAttachmentAction = nil
		m.attachmentActionNotice = "無法驗證附件：" + err.Error()
		return
	}
	m.pendingAttachmentAction.operation = m.workspace.State().Operation
	m.attachmentActionNotice = "正在重新驗證附件…"
}

func (m *Model) finishAttachmentAction(state workspace.State) {
	pending := m.pendingAttachmentAction
	if pending == nil {
		return
	}
	if state.Epoch != pending.epoch || state.ActivePath != pending.ledgerPath || state.Operation > pending.operation {
		m.pendingAttachmentAction = nil
		m.attachmentActionNotice = "账本或请求已变化；没有打开附件。"
		return
	}
	if state.Operation < pending.operation || state.Busy {
		return
	}
	m.pendingAttachmentAction = nil
	if state.Error != "" || state.Response == nil || state.Response.DocumentAction == nil {
		message := "附件当前不可用；请重新读取列表。"
		if state.Error != "" {
			message = "附件重新验证失败：" + state.Error
		}
		m.attachmentActionNotice = message
		return
	}
	current := m.workspace.State()
	response := state.Response
	action := response.DocumentAction
	if current.Revision != state.Revision || current.Epoch != pending.epoch || current.Operation != pending.operation || current.ActivePath != pending.ledgerPath || current.Ledger.SnapshotID != pending.snapshotID || current.AttachmentRoots.PolicyRevision != pending.policyRevision || response.State.SnapshotID != pending.snapshotID || response.Request.DocumentID != pending.id || response.Request.DocumentPolicyRevision != pending.policyRevision || action.ID != pending.id || action.PolicyRevision != pending.policyRevision || !filepath.IsAbs(action.Path) || filepath.Clean(action.Path) != action.Path {
		m.attachmentActionNotice = "附件授权或账本状态已变化；没有打开附件。"
		return
	}
	if pending.reveal {
		if m.revealPath != nil {
			m.revealPath(action.Path)
			m.attachmentActionNotice = "已在 Finder 中显示附件。"
		}
		return
	}
	if m.openPath == nil {
		m.attachmentActionNotice = "系统打开功能不可用。"
		return
	}
	if err := m.openPath(action.Path); err != nil {
		m.attachmentActionNotice = "无法打开附件：" + err.Error()
		return
	}
	m.attachmentActionNotice = "已交给系统默认应用打开。"
}

func (m *Model) attachmentRootsPage(c *ui.Context) {
	t := c.Theme()
	ui.Text(c, m.tr("默认目录是账本入口所在目录。额外目录仅通过显式选择加入；移除后旧附件引用会失效。 ")).TextColor(t.TextMuted)
	if !m.workspaceState.AttachmentRootsLoaded {
		ui.Text(c, m.tr("目录策略尚未载入。打开有效账本后可管理额外目录。"))
		return
	}
	ui.Text(c, m.tr("默认目录（不可移除）")).FontSize(14).Bold()
	ui.Text(c, m.workspaceState.AttachmentRoots.DefaultRoot).SingleLine()
	ui.Divider(c)
	ui.Text(c, m.tr("额外目录")).FontSize(14).Bold()
	if len(m.attachmentBookmarks) == 0 {
		ui.Text(c, m.tr("没有额外目录")).TextColor(t.TextMuted)
	}
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Gap(8).Children(func() {
			for _, bookmark := range m.attachmentBookmarks {
				bookmark := bookmark
				ui.Row(c).Gap(10).Padding(8).Background(t.Surface).Border(1, t.Border).Radius(6).Children(func() {
					ui.Column(c).Grow(1).Children(func() {
						ui.Text(c, bookmark.Path).SingleLine()
						if m.isInvalidAttachmentRoot(bookmark.Path) {
							ui.Text(c, m.tr("路径或文件身份已变化；该目录当前未加入策略。请重新选择或移除。 ")).FontSize(11).TextColor(t.Danger)
						}
					})
					if ui.Button(c, m.tr("移除")).Clicked() && !m.workspaceState.AttachmentRootsBusy && !m.rootPolicySyncBusy && !m.picking {
						remaining := make([]platform.DirectoryBookmark, 0, len(m.attachmentBookmarks)-1)
						for _, item := range m.attachmentBookmarks {
							if item.Path != bookmark.Path {
								remaining = append(remaining, item)
							}
						}
						m.configureAttachmentRoots(remaining, m.workspaceState.AttachmentRoots.PolicyRevision, true)
					}
				})
			}
		})
	})
	if m.attachmentRootNotice != "" {
		ui.Text(c, m.tr(m.attachmentRootNotice)).TextColor(t.Danger).Selectable()
	}
	if m.workspaceState.AttachmentRootsError != "" {
		ui.Text(c, m.workspaceState.AttachmentRootsError).TextColor(t.Danger).Selectable()
	}
	ui.Row(c).Gap(10).Children(func() {
		if ui.PrimaryButton(c, m.tr("添加目录…")).Clicked() {
			m.chooseAttachmentRoot()
		}
		if m.workspaceState.AttachmentRootsBusy || m.rootPolicySyncBusy || m.picking {
			ui.Text(c, m.tr("正在更新目录策略…")).TextColor(t.TextMuted)
		}
	})
}

func (m *Model) isInvalidAttachmentRoot(path string) bool {
	for _, invalid := range m.invalidAttachmentRoots {
		if invalid == path {
			return true
		}
	}
	return false
}

func (m *Model) tablePage(c *ui.Context) {
	t := c.Theme()
	ui.Text(c, m.tr("合成数据只用于检验虚拟化；不会读写私人账本。")).TextColor(t.TextMuted)
	columns := []ui.TableColumn{
		{Title: m.tr("编号"), ID: "id", Width: 112, Fixed: true},
		{Title: m.tr("日期"), ID: "date", Width: 150},
		{Title: m.tr("账户"), ID: "account"},
		{Title: m.tr("金额"), ID: "amount", Width: 150, Align: ui.End},
	}
	ui.Table(c, &m.table, columns, sampleRows, func(row, col int) {
		switch col {
		case 0:
			ui.Text(c, fmt.Sprintf("Entry %06d", row+1))
		case 1:
			ui.Text(c, fmt.Sprintf("2026-%02d-%02d", row%12+1, row%28+1))
		case 2:
			ui.Text(c, fmt.Sprintf("Assets:Cash:%02d", row%40+1))
		case 3:
			ui.Text(c, fmt.Sprintf("%d.%02d USD", row%2500, row%100)).TextColor(t.TextMuted)
		}
	}).Grow(1)
}

func (m *Model) chartPage(c *ui.Context) {
	series := cashflowBarSample(m.tr)
	ui.Text(c, m.tr("合成数值绘图试验 · 零值与缺值分别编码")).TextColor(c.Theme().TextMuted)
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Gap(14).Padding(0, 0, 16).Children(func() {
			ui.Text(c, m.tr("收入与支出")).FontSize(16).Bold()
			LineChart(c, cashflowLineSample(m.tr), m.tr)
			ui.Text(c, m.tr("月度收支分组柱")).FontSize(16).Bold()
			GroupedBars(c, series, m.tr)
			ExactTable(c, &m.chartTable, series, m.tr)
		})
	})
}

func (m *Model) chooseLedger() {
	if m.picking || m.workspaceState.Busy || m.workspaceState.AttachmentRootsBusy || m.rootPolicySyncBusy || m.closed.Load() || m.window == nil {
		return
	}
	m.picking = true
	go m.pickAndOpen()
}

func (m *Model) chooseAttachmentRoot() {
	if m.picking || m.workspaceState.AttachmentRootsBusy || m.rootPolicySyncBusy || !m.workspaceState.AttachmentRootsLoaded || m.closed.Load() || m.window == nil {
		return
	}
	m.picking = true
	window := m.window
	go func() {
		pick := m.pickAttachmentDirectory
		if pick == nil {
			pick = mygo.Dialog.Open
		}
		paths, err := pick(mygo.OpenDialogOptions{
			Parent: window, Title: m.tr("添加附件目录"), Message: m.tr("选择允许账本引用附件的目录"), Directory: true,
		})
		if err != nil || len(paths) == 0 || m.closed.Load() {
			window.Update(func() {
				if m.closed.Load() {
					return
				}
				m.picking = false
				if err != nil {
					m.attachmentRootNotice = "无法选择附件目录：" + err.Error()
				}
			})
			return
		}
		selected := paths[0]
		if isBroadAttachmentRoot(selected) {
			result, confirmErr := mygo.Dialog.Message(mygo.MessageOptions{
				Parent: window, Type: mygo.MessageWarning, Title: m.tr("确认较大的附件目录范围"),
				Message: m.tr("允许整个目录及其子目录中的附件？\n") + selected,
				Buttons: []string{m.tr("允许"), m.tr("取消")}, DefaultButton: 0, CancelButton: 1,
			})
			if confirmErr != nil || result.Button != 0 || m.closed.Load() {
				window.Update(func() { m.picking = false })
				return
			}
		}
		bookmark, err := platform.CreateDirectoryBookmark(selected)
		window.Update(func() {
			if m.closed.Load() {
				return
			}
			m.picking = false
			if err != nil {
				m.attachmentRootNotice = "无法保存所选目录：" + err.Error()
				return
			}
			updated, result := mergeAttachmentRootBookmark(m.attachmentBookmarks, m.invalidAttachmentRoots, m.workspaceState.AttachmentRoots.DefaultRoot, bookmark)
			switch result {
			case attachmentRootBookmarkDuplicate:
				m.attachmentRootNotice = "该附件目录已经在列表中。"
				return
			case attachmentRootBookmarkDefault:
				m.attachmentRootNotice = "账本所在目录已自动作为默认目录，无需重复添加。"
				return
			}
			m.configureAttachmentRoots(updated, m.workspaceState.AttachmentRoots.PolicyRevision, true)
		})
	}()
}

type attachmentRootBookmarkMergeResult uint8

const (
	attachmentRootBookmarkAdded attachmentRootBookmarkMergeResult = iota
	attachmentRootBookmarkReplaced
	attachmentRootBookmarkDuplicate
	attachmentRootBookmarkDefault
)

func mergeAttachmentRootBookmark(existing []platform.DirectoryBookmark, invalidPaths []string, defaultRoot string, selected platform.DirectoryBookmark) ([]platform.DirectoryBookmark, attachmentRootBookmarkMergeResult) {
	if selected.Path == defaultRoot {
		return cloneDirectoryBookmarks(existing), attachmentRootBookmarkDefault
	}
	updated := cloneDirectoryBookmarks(existing)
	for i, current := range updated {
		if current.Path != selected.Path {
			continue
		}
		if slices.Contains(invalidPaths, current.Path) {
			updated[i] = selected
			return updated, attachmentRootBookmarkReplaced
		}
		return updated, attachmentRootBookmarkDuplicate
	}
	return append(updated, selected), attachmentRootBookmarkAdded
}

func (m *Model) configureAttachmentRoots(bookmarks []platform.DirectoryBookmark, revision uint64, persist bool) {
	if m.workspace == nil || m.workspaceState.ActivePath == "" || m.closed.Load() || m.rootPolicySyncBusy {
		return
	}
	m.rootPolicySyncBusy = true
	controller, store, window := m.workspace, m.preferenceStore, m.window
	ledgerPath := m.workspaceState.ActivePath
	currentGeneration := m.attachmentRootGeneration
	records := cloneDirectoryBookmarks(bookmarks)
	m.preferenceWork.Add(1)
	go func() {
		defer m.preferenceWork.Done()
		validPaths := make([]string, 0, len(records))
		invalid := make([]string, 0)
		for _, bookmark := range records {
			path, err := platform.ResolveDirectoryBookmark(bookmark)
			if err != nil {
				invalid = append(invalid, bookmark.Path)
				continue
			}
			validPaths = append(validPaths, path)
		}
		updated, err := controller.ConfigureAttachmentRoots(desktop.AttachmentRootsRequest{PolicyRevision: revision, AdditionalRoots: validPaths})
		generation := currentGeneration
		var saveErr error
		if err == nil && persist {
			if store == nil {
				saveErr = fmt.Errorf("偏好存储不可用")
			} else {
				generation, saveErr = store.UpdateAttachmentRoots(ledgerPath, records, m.windowID)
			}
		}
		if window == nil || m.closed.Load() {
			return
		}
		window.Update(func() {
			if m.closed.Load() || m.workspace != controller || m.workspaceState.ActivePath != ledgerPath {
				return
			}
			m.rootPolicySyncBusy = false
			if err != nil {
				m.attachmentRootNotice = "目录策略未更新：" + err.Error()
				m.applyPendingSharedAttachmentRoots()
				return
			}
			m.attachmentBookmarks = records
			m.invalidAttachmentRoots = invalid
			m.attachmentRootGeneration = generation
			if saveErr != nil {
				m.attachmentRootNotice = "目录本次会话有效，但未保存：" + saveErr.Error()
			} else if len(invalid) > 0 {
				m.attachmentRootNotice = "部分目录已失效，当前未加入策略；请重新选择或移除。"
			} else {
				m.attachmentRootNotice = ""
			}
			if persist && m.workspaceState.Response != nil {
				m.refreshCurrentView()
			}
			_ = updated
			m.applyPendingSharedAttachmentRoots()
		})
	}()
}

func (m *Model) receiveSharedAttachmentRoots(change preferences.AttachmentRootsChange) {
	window := m.window
	if window == nil || m.closed.Load() {
		return
	}
	change.Roots = cloneDirectoryBookmarks(change.Roots)
	window.Update(func() {
		if m.closed.Load() || m.preferenceStore == nil || m.workspace == nil || m.workspaceState.ActivePath == "" {
			return
		}
		identity, err := preferences.Identity(m.workspaceState.ActivePath)
		if err != nil || identity != change.Identity || change.Generation <= m.attachmentRootGeneration {
			return
		}
		if m.pendingSharedRootChange != nil && change.Generation <= m.pendingSharedRootChange.Generation {
			return
		}
		m.pendingSharedRootChange = &change
		m.applyPendingSharedAttachmentRoots()
	})
}

func (m *Model) applyPendingSharedAttachmentRoots() {
	if m.rootPolicySyncBusy || m.pendingSharedRootChange == nil || m.workspace == nil || m.workspaceState.ActivePath == "" || !m.workspaceState.AttachmentRootsLoaded {
		return
	}
	change := *m.pendingSharedRootChange
	m.pendingSharedRootChange = nil
	m.rootPolicySyncBusy = true
	controller, window := m.workspace, m.window
	ledgerPath := m.workspaceState.ActivePath
	revision := m.workspaceState.AttachmentRoots.PolicyRevision
	bookmarks := cloneDirectoryBookmarks(change.Roots)
	m.preferenceWork.Add(1)
	go func() {
		defer m.preferenceWork.Done()
		validPaths := make([]string, 0, len(bookmarks))
		invalid := make([]string, 0)
		for _, bookmark := range bookmarks {
			path, err := platform.ResolveDirectoryBookmark(bookmark)
			if err != nil {
				invalid = append(invalid, bookmark.Path)
				continue
			}
			validPaths = append(validPaths, path)
		}
		updated, err := controller.ConfigureAttachmentRoots(desktop.AttachmentRootsRequest{PolicyRevision: revision, AdditionalRoots: validPaths})
		if window == nil || m.closed.Load() {
			return
		}
		window.Update(func() {
			if m.closed.Load() || m.workspace != controller || m.workspaceState.ActivePath != ledgerPath {
				return
			}
			m.rootPolicySyncBusy = false
			if err != nil {
				m.attachmentRootNotice = "其他窗口的目录撤销尚未同步：" + err.Error()
			} else {
				m.workspaceState.AttachmentRoots = updated
				m.attachmentBookmarks = bookmarks
				m.attachmentRootGeneration = change.Generation
				m.invalidAttachmentRoots = invalid
				if len(invalid) > 0 {
					m.attachmentRootNotice = "部分目录已失效，当前未加入策略；请重新选择或移除。"
				} else {
					m.attachmentRootNotice = ""
				}
				if m.workspaceState.Response != nil {
					m.refreshCurrentView()
				}
			}
			m.applyPendingSharedAttachmentRoots()
		})
	}()
}

func (m *Model) refreshCurrentView() {
	if m.workspace == nil || m.workspaceState.ActivePath == "" {
		return
	}
	view := m.currentViewName()
	if err := m.workspace.Dispatch(workspace.Intent{Kind: workspace.IntentRead, Request: m.requestForView(view)}); err != nil {
		m.status = "无法刷新当前页面：" + err.Error()
		return
	}
	m.dashboardRequestOperation = m.workspace.State().Operation
	m.status = ""
}

func (m *Model) currentViewName() string {
	switch m.page {
	case pageBalance:
		return "balance_sheet"
	case pageIncome:
		return "income_statement"
	case pageJournal:
		return "journal"
	case pageAccount:
		return "account"
	case pagePrices:
		return "prices"
	case pageHoldings:
		return "holdings"
	case pageDirectives:
		return "directives"
	case pageSemantics:
		return "migration"
	case pageQuery:
		return "query"
	case pageAttachments:
		return "attachments"
	default:
		return "dashboard"
	}
}

func (m *Model) requestLocale() string {
	if m.locale == "en" {
		return "en"
	}
	return "zh-CN"
}

func (m *Model) setLocale(locale string) {
	if locale != "zh-CN" && locale != "en" || locale == m.requestLocale() {
		return
	}
	m.locale = locale
	m.preferenceError = ""
	m.persistLocale()
	m.refreshCurrentView()
	if m.onLocaleChanged != nil {
		m.onLocaleChanged()
	}
}

func (m *Model) persistLocale() {
	if m.preferenceStore == nil || m.workspaceState.ActivePath == "" {
		return
	}
	store := m.preferenceStore
	path := m.workspaceState.ActivePath
	locale := m.requestLocale()
	m.preferenceWork.Add(1)
	go func() {
		defer m.preferenceWork.Done()
		err := store.UpdateLedger(path, func(item *preferences.LedgerPreferences) error {
			item.Locale = locale
			return nil
		})
		if err == nil {
			return
		}
		message := "界面语言偏好未能保存：" + err.Error()
		window := m.window
		if window == nil || m.closed.Load() {
			return
		}
		window.Update(func() {
			if !m.closed.Load() {
				m.preferenceError = message
			}
		})
	}()
}

func cloneDirectoryBookmarks(bookmarks []platform.DirectoryBookmark) []platform.DirectoryBookmark {
	if bookmarks == nil {
		return nil
	}
	clone := make([]platform.DirectoryBookmark, len(bookmarks))
	for i, bookmark := range bookmarks {
		bookmark.Data = append([]byte(nil), bookmark.Data...)
		clone[i] = bookmark
	}
	return clone
}

func isBroadAttachmentRoot(path string) bool {
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	if canonical == "/" || canonical == "/Users" || canonical == "/Volumes" {
		return true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	home, err = filepath.EvalSymlinks(home)
	return err == nil && canonical == home
}

func (m *Model) pickAndOpen() {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent:  m.window,
		Title:   m.tr("打开 Beancount 账本"),
		Filters: []mygo.FileFilter{{Name: "Beancount", Extensions: []string{"bean"}}},
	})
	window := m.window
	if window == nil || m.closed.Load() {
		return
	}
	window.Update(func() {
		if m.closed.Load() {
			return
		}
		m.picking = false
		if err != nil {
			m.status = "无法打开文件面板：" + err.Error()
			return
		}
		if len(paths) == 0 {
			return
		}
		m.status = ""
		m.page = pageWelcome
		if m.workspace == nil {
			m.status = "工作区尚未就绪"
			return
		}
		m.openLedger(paths[0])
	})
}

func (m *Model) openLedger(path string) {
	m.openLedgerWithProfile(path, "", "")
}

func (m *Model) openLedgerWithProfile(path string, selectedProfile ledger.SemanticsProfile, inputIdentity string) {
	if m.rootPolicySyncBusy || m.workspaceState.AttachmentRootsBusy {
		m.status = "请等待附件目录策略更新后再打开其他账本。"
		return
	}
	selected := preferences.LedgerPreferences{SemanticsProfile: string(ledger.Legacy), Locale: "zh-CN", TextScale: 1, Window: "all", Valuation: "at-cost"}
	if m.preferenceStore != nil {
		stored, err := m.preferenceStore.ForLedger(path)
		if err != nil {
			m.preferenceError = "无法读取此账本的偏好：" + err.Error()
		} else {
			selected = stored
			m.preferenceError = ""
		}
	}
	profile := ledger.SemanticsProfile(selected.SemanticsProfile)
	if profile != ledger.Legacy && profile != ledger.V3 {
		profile = ledger.Legacy
	}
	if selectedProfile != "" {
		if selectedProfile != ledger.Legacy && selectedProfile != ledger.V3 {
			m.status = "不支持的语义配置。"
			return
		}
		profile = selectedProfile
	}
	restoredRoots := make([]string, 0, len(selected.AttachmentRoots))
	invalidRoots := make([]string, 0)
	for _, bookmark := range selected.AttachmentRoots {
		root, err := platform.ResolveDirectoryBookmark(bookmark)
		if err != nil {
			invalidRoots = append(invalidRoots, bookmark.Path)
			continue
		}
		restoredRoots = append(restoredRoots, root)
	}
	m.pendingOpen = true
	m.cancelPendingExport()
	m.pendingPath = path
	m.pendingPreferences = selected
	m.pendingSemanticsSwitch = inputIdentity != ""
	m.pendingSemanticsProfile = profile
	m.pendingAttachmentRoots = restoredRoots
	m.pendingInvalidRoots = invalidRoots
	m.pendingRootRestore = false
	if err := m.workspace.Dispatch(workspace.Intent{
		Kind: workspace.IntentOpen,
		Path: path,
		Options: workspace.OpenOptions{
			SemanticsProfile: profile,
			InputIdentity:    inputIdentity,
			Watch:            snapshot.WatchOptions{PollInterval: 500 * time.Millisecond},
		},
	}); err != nil {
		m.pendingOpen = false
		m.pendingPath = ""
		m.pendingSemanticsSwitch = false
		m.pendingSemanticsProfile = ""
		m.status = "无法读取账本：" + err.Error()
		return
	}
	m.resetPageHistory()
}

func (m *Model) applyPendingOpenState(state workspace.State) {
	if !m.pendingOpen {
		return
	}
	switch {
	case state.ActivePath == m.pendingPath && (state.Phase == workspace.PhaseReady || state.Phase == workspace.PhaseInvalid || state.Phase == workspace.PhaseStale):
		previousLocale := m.requestLocale()
		m.filterDateFrom = m.pendingPreferences.DateFrom
		m.filterDateTo = m.pendingPreferences.DateTo
		m.locale = m.pendingPreferences.Locale
		if m.locale != "en" {
			m.locale = "zh-CN"
		}
		if previousLocale != m.locale && m.onLocaleChanged != nil {
			m.onLocaleChanged()
		}
		m.filterWindow = m.pendingPreferences.Window
		m.filterCurrency = m.pendingPreferences.Currency
		m.filterValuation = m.pendingPreferences.Valuation
		m.textScale = normalizedTextScale(m.pendingPreferences.TextScale)
		m.restoreColumnPreferences(state.ActivePath, m.pendingPreferences)
		m.attachmentBookmarks = cloneDirectoryBookmarks(m.pendingPreferences.AttachmentRoots)
		m.attachmentRootGeneration = m.pendingPreferences.AttachmentRootsGeneration
		m.invalidAttachmentRoots = append([]string(nil), m.pendingInvalidRoots...)
		m.pendingRootRestore = len(m.pendingAttachmentRoots) > 0
		if len(m.invalidAttachmentRoots) > 0 {
			m.attachmentRootNotice = "部分附件目录需要重新选择或移除。"
		} else {
			m.attachmentRootNotice = ""
		}
		if m.pendingSemanticsSwitch {
			if state.Ledger.SemanticsProfile == m.pendingSemanticsProfile {
				m.queryDraft = ""
				m.querySelection = -1
				m.querySelectionContext = ""
				m.persistSemanticsProfile(state.ActivePath, state.Ledger.SemanticsProfile)
			} else {
				m.status = "语义配置应用失败：新会话返回的配置与所选配置不符。"
			}
		}
		m.pendingOpen = false
		m.pendingPath = ""
		m.pendingSemanticsSwitch = false
		m.pendingSemanticsProfile = ""
	case state.Phase == workspace.PhaseFailed:
		if m.pendingSemanticsSwitch {
			m.status = "语义配置应用失败，当前会话已保留：" + state.Error
		}
		m.pendingOpen = false
		m.pendingPath = ""
		m.pendingSemanticsSwitch = false
		m.pendingSemanticsProfile = ""
		m.pendingAttachmentRoots = nil
		m.pendingInvalidRoots = nil
		m.pendingRootRestore = false
	}
}

func (m *Model) receiveWorkspaceState(state workspace.State) {
	window := m.window
	if window == nil || m.closed.Load() {
		return
	}
	window.Update(func() {
		if m.closed.Load() || state.Revision <= m.workspaceState.Revision {
			return
		}
		previousPath := m.workspaceState.ActivePath
		previousSnapshotID := m.workspaceState.Ledger.SnapshotID
		previousEpoch := m.workspaceState.Epoch
		if state.ActivePath != "" && state.ActivePath != previousPath {
			m.resetPageHistory()
		}
		cancelledExport := (m.activeExportID != 0 || m.pendingQuerySave != nil || m.pendingReportSave != nil) &&
			(state.Epoch != previousEpoch || state.Busy || state.QueryBusy)
		if cancelledExport {
			m.cancelPendingExport()
		}
		m.workspaceState = state
		if previousPath != "" && state.ActivePath != "" && previousPath != state.ActivePath {
			m.queryDraft = ""
			m.querySelection = -1
			m.querySelectionContext = ""
			m.journalSearch = ""
			m.journalSearchDraft = ""
			m.journalAccount = ""
			m.journalDescending = true
			m.journalError = ""
			m.journalOffset = 0
			m.journalSelectedID = ""
			m.journalSelection = -1
			m.accountReturnResponse = nil
			m.accountName = ""
			m.accountJournalSearch = ""
			m.accountJournalSearchDraft = ""
			m.accountJournalOffset = 0
			m.accountJournalSelectedID = ""
			m.accountJournalSelection = -1
			m.priceOffset = 0
			m.priceSelection = -1
			m.priceSelectedID = ""
			m.commodityOffset = 0
			m.commoditySelection = -1
			m.commoditySelectedID = ""
			m.priceBase, m.priceQuote = "", ""
			m.holdingAccount, m.holdingGroupID, m.holdingEventID = "", "", ""
			m.holdingSection = "groups"
			m.holdingOffset, m.holdingSelection, m.holdingSelectedID = 0, -1, ""
			m.queryCatalogueOffset, m.queryCatalogueSelectedID = 0, ""
			m.queryCatalogueSearch, m.queryCatalogueSearchDraft, m.queryCatalogueSearchError = "", "", ""
			m.queryPresetPeriodPresets = nil
			m.queryPresetPeriodSnapshot = ""
			m.queryPresetPeriodError = ""
			m.querySelection, m.querySelectionContext = -1, ""
		} else if previousSnapshotID != "" && state.Ledger.SnapshotID != previousSnapshotID {
			// The core gives unique journal transactions content-based IDs, so
			// keep the current page and selection across source-only edits such
			// as inserted comments. Ambiguous duplicates receive snapshot-bound
			// IDs and therefore will not inherit the old selection.
			m.journalSelection = -1
			m.accountJournalOffset = 0
			m.accountJournalSelectedID = ""
			m.accountJournalSelection = -1
			m.accountReturnResponse = nil
			m.priceOffset = 0
			m.priceSelection = -1
			m.priceSelectedID = ""
			m.commodityOffset = 0
			m.commoditySelection = -1
			m.commoditySelectedID = ""
			m.priceBase, m.priceQuote = "", ""
			if m.holdingSection == "lots" {
				if m.holdingEventID != "" {
					m.holdingSection = "events"
				} else {
					m.holdingSection = "groups"
				}
			}
			m.holdingGroupID, m.holdingEventID = "", ""
			m.holdingOffset, m.holdingSelection, m.holdingSelectedID = 0, -1, ""
			m.queryCatalogueOffset, m.queryCatalogueSelectedID = 0, ""
			m.querySelection, m.querySelectionContext = -1, ""
		}
		if m.queryPresetLedger != state.ActivePath {
			m.queryPresetGeneration++
			m.queryPresetLedger = state.ActivePath
			m.queryPresetLoading = false
			m.queryPresetBusy = false
			m.queryPresetError = ""
			m.queryPresetName = ""
			m.queryPresets = nil
			if state.ActivePath != "" {
				m.loadQueryPresets(state.ActivePath, m.queryPresetGeneration)
			}
		}
		m.status = ""
		if cancelledExport {
			m.status = "工作区正在切换或读取，导出已取消。"
		}
		m.finishPendingReportSave(state)
		m.finishPendingQuerySave(state)
		m.applyPendingOpenState(state)
		m.finishAttachmentAction(state)
		if m.pendingRootRestore && state.AttachmentRootsLoaded && !state.AttachmentRootsBusy {
			m.pendingRootRestore = false
			m.pendingAttachmentRoots = nil
			m.configureAttachmentRoots(m.attachmentBookmarks, state.AttachmentRoots.PolicyRevision, false)
			return
		}
		if state.ActivePath != "" && (state.Phase == workspace.PhaseReady || state.Phase == workspace.PhaseInvalid || state.Phase == workspace.PhaseStale) {
			m.recordRecentLedger(state.ActivePath)
		}
		canReadView := state.Phase == workspace.PhaseReady || state.Phase == workspace.PhaseInvalid || state.Phase == workspace.PhaseStale
		if canReadView && !state.Busy && !state.AttachmentRootsBusy && state.Response == nil && state.Operation != m.dashboardRequestOperation {
			view := "dashboard"
			switch m.page {
			case pageBalance:
				view = "balance_sheet"
			case pageIncome:
				view = "income_statement"
			case pageJournal:
				view = "journal"
			case pageAccount:
				view = "account"
			case pagePrices:
				view = "prices"
			case pageHoldings:
				view = "holdings"
			case pageDirectives:
				view = "directives"
			case pageSemantics:
				view = "migration"
			case pageQuery:
				view = "query"
			case pageAttachments:
				view = "attachments"
			}
			if err := m.workspace.Dispatch(workspace.Intent{Kind: workspace.IntentRead, Request: m.requestForView(view)}); err != nil {
				m.status = "无法生成页面：" + err.Error()
			} else {
				m.dashboardRequestOperation = m.workspace.State().Operation
			}
		}
	})
}
