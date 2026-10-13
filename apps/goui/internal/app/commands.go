// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"orangecount/apps/goui/internal/i18n"
	"orangecount/apps/goui/internal/workspace"
)

// NewApplicationMenu keeps native menu commands routed to the focused window.
// The caller owns the window registry and may create a fresh window when the
// app has no focused document window.
func NewApplicationMenu(modelForWindow func(*mygo.Window) *Model, createWindow func() *Model, locales ...string) *mygo.Menu {
	locale := i18n.SimplifiedChinese
	if len(locales) > 0 {
		locale = i18n.Normalize(locales[0])
	}
	tr := func(source string) string { return i18n.Text(locale, source) }
	resolveModel := func(window *mygo.Window) *Model {
		if modelForWindow != nil {
			if model := modelForWindow(window); model != nil {
				return model
			}
		}
		return nil
	}
	modelForOpen := func(window *mygo.Window) *Model {
		if model := resolveModel(window); model != nil {
			return model
		}
		if createWindow != nil {
			return createWindow()
		}
		return nil
	}
	return mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu, Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleAbout, Label: tr("关于 OrangeCount")},
			mygo.Separator(),
			{Role: mygo.RoleServices},
			mygo.Separator(),
			{Role: mygo.RoleHide, Label: tr("隐藏 OrangeCount")},
			{Role: mygo.RoleHideOthers, Label: tr("隐藏其他")},
			{Role: mygo.RoleUnhide, Label: tr("全部显示")},
			mygo.Separator(),
			{Role: mygo.RoleQuit, Label: tr("退出 OrangeCount")},
		}},
		{Role: mygo.RoleFileMenu, Label: tr("文件"), Submenu: []*mygo.MenuItem{
			{ID: "open-ledger", Label: tr("打开账本…"), Accelerator: "CmdOrCtrl+O", Click: func(_ *mygo.MenuItem, window *mygo.Window) {
				if model := modelForOpen(window); model != nil {
					model.chooseLedger()
				}
			}},
			{ID: "new-window", Label: tr("新建窗口"), Accelerator: "CmdOrCtrl+N", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
				if createWindow != nil {
					createWindow()
				}
			}},
			{ID: "reload-ledger", Label: tr("重新读取账本"), Accelerator: "CmdOrCtrl+R", Click: func(_ *mygo.MenuItem, window *mygo.Window) {
				if model := resolveModel(window); model != nil {
					model.reloadActiveLedger()
				}
			}},
			{ID: "export-preferences-backup", Label: tr("导出偏好备份…"), Click: func(_ *mygo.MenuItem, window *mygo.Window) {
				if model := modelForOpen(window); model != nil {
					model.exportPreferencesBackup()
				}
			}},
			{ID: "source-editor", Label: tr("选择来源编辑器…"), Click: func(_ *mygo.MenuItem, window *mygo.Window) {
				if model := modelForOpen(window); model != nil {
					model.chooseSourceEditor()
				}
			}},
			{ID: "source-editor-default", Label: tr("使用系统默认应用打开来源"), Click: func(_ *mygo.MenuItem, window *mygo.Window) {
				if model := modelForOpen(window); model != nil {
					model.restoreSystemSourceEditor()
				}
			}},
			mygo.Separator(),
			{Role: mygo.RoleClose, Label: tr("关闭窗口")},
		}},
		{Role: mygo.RoleEditMenu, Label: tr("编辑"), Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleUndo, Label: tr("撤销")},
			{Role: mygo.RoleRedo, Label: tr("重做")},
			mygo.Separator(),
			{Role: mygo.RoleCut, Label: tr("剪切")},
			{Role: mygo.RoleCopy, Label: tr("拷贝")},
			{Role: mygo.RolePaste, Label: tr("粘贴")},
			{Role: mygo.RolePasteAndMatchStyle, Label: tr("粘贴并匹配样式")},
			{Role: mygo.RoleDelete, Label: tr("删除")},
			{Role: mygo.RoleSelectAll, Label: tr("全选")},
			mygo.Separator(),
			{ID: "find", Label: tr("查找"), Accelerator: "CmdOrCtrl+F", Click: func(_ *mygo.MenuItem, window *mygo.Window) {
				if model := resolveModel(window); model != nil {
					model.focusCurrentSearch()
				}
			}},
		}},
		{Role: mygo.RoleViewMenu, Label: tr("显示"), Submenu: []*mygo.MenuItem{
			{ID: "increase-text-size", Label: tr("放大文字"), Accelerator: "CmdOrCtrl+Plus", Click: func(_ *mygo.MenuItem, window *mygo.Window) {
				if model := resolveModel(window); model != nil {
					model.increaseTextScale()
				}
			}},
			{ID: "decrease-text-size", Label: tr("缩小文字"), Accelerator: "CmdOrCtrl+-", Click: func(_ *mygo.MenuItem, window *mygo.Window) {
				if model := resolveModel(window); model != nil {
					model.decreaseTextScale()
				}
			}},
			{ID: "reset-text-size", Label: tr("恢复默认文字大小"), Accelerator: "CmdOrCtrl+0", Click: func(_ *mygo.MenuItem, window *mygo.Window) {
				if model := resolveModel(window); model != nil {
					model.resetTextScale()
				}
			}},
			mygo.Separator(),
			{Role: mygo.RoleToggleFullScreen, Label: tr("切换全屏")},
		}},
		{Role: mygo.RoleWindowMenu, Label: tr("窗口"), Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleMinimize, Label: tr("最小化")},
			{Role: mygo.RoleZoom, Label: tr("缩放")},
			mygo.Separator(),
			{Role: mygo.RoleFront, Label: tr("将所有窗口置于最前")},
		}},
	})
}

func (m *Model) reloadActiveLedger() {
	if m.workspace == nil || m.workspaceState.ActivePath == "" {
		m.status = "请先打开账本。"
		return
	}
	m.cancelPendingExport()
	if err := m.workspace.Dispatch(workspace.Intent{Kind: workspace.IntentReload}); err != nil {
		m.status = "无法重读账本：" + err.Error()
		return
	}
	m.status = "正在重新读取账本…"
}

func (m *Model) focusCurrentSearch() {
	if m.closed.Load() {
		return
	}
	switch m.page {
	case pageBalance, pageIncome, pageJournal, pageAccount, pageHoldings, pageDirectives, pageQuery:
	default:
		m.status = "当前页面没有可搜索的内容。"
		return
	}
	focus := func() {
		if m.closed.Load() {
			return
		}
		m.searchFocusPending = true
		m.searchFocusPage = m.page
	}
	if m.window == nil {
		focus()
		return
	}
	m.window.Update(focus)
}

func (m *Model) focusSearchInput(forPage page, input *ui.Element) {
	if m.searchFocusPending && m.searchFocusPage == forPage && m.page == forPage {
		input.Focus()
		m.searchFocusPending = false
	}
}
