// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import (
	"fmt"
	"os"

	"github.com/egoist/mygo"
	"orangecount/internal/desktop"
	"orangecount/internal/query"
)

// OpenQueryAnalysisPath reads a Finder-opened or dropped capture without
// touching the workspace or running the captured query.
func (m *Model) OpenQueryAnalysisPath(path string) {
	window := m.window
	if window == nil || m.closed.Load() || path == "" {
		return
	}
	m.cancelPendingExport()
	m.navigateToPage(pageQuery, "", true)
	m.offlineAnalysisSequence++
	sequence := m.offlineAnalysisSequence
	m.offlineAnalysisLoading = true
	m.offlineAnalysis = nil
	m.offlineAnalysisPath = ""
	m.offlineAnalysisError = ""
	m.querySelection = -1
	m.querySelectionContext = ""
	m.status = "正在读取离线分析文件…"
	go func() {
		document, err := readQueryAnalysisFile(path)
		window.Update(func() {
			if m.closed.Load() || sequence != m.offlineAnalysisSequence {
				return
			}
			m.offlineAnalysisLoading = false
			if err != nil {
				m.offlineAnalysisError = "无法读取离线分析文件：" + err.Error()
				m.status = m.offlineAnalysisError
				return
			}
			m.offlineAnalysis = &document
			m.offlineAnalysisPath = path
			m.offlineAnalysisError = ""
			m.status = "已载入离线捕获；文件中的查询不会自动执行。"
		})
	}()
}

func (m *Model) chooseQueryAnalysis() {
	if m.picking || m.closed.Load() || m.window == nil {
		return
	}
	m.picking = true
	window := m.window
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent: window, Title: m.tr("打开离线查询分析文件"),
			Message: m.tr("只读取已捕获的结果，不会打开账本或重新执行查询"),
			Filters: []mygo.FileFilter{{Name: m.tr("OrangeCount 分析文件"), Extensions: []string{"ocanalysis"}}},
		})
		if err != nil || len(paths) == 0 || m.closed.Load() {
			window.Update(func() {
				if m.closed.Load() {
					return
				}
				m.picking = false
				if err != nil {
					m.status = "无法打开文件面板：" + err.Error()
				}
			})
			return
		}
		path := paths[0]
		document, err := readQueryAnalysisFile(path)
		window.Update(func() {
			if m.closed.Load() {
				return
			}
			m.picking = false
			if err != nil {
				m.offlineAnalysisError = "无法读取离线分析文件：" + err.Error()
				m.status = "无法读取离线分析文件：" + err.Error()
				return
			}
			m.offlineAnalysis = &document
			m.offlineAnalysisPath = path
			m.offlineAnalysisError = ""
			m.querySelection = -1
			m.querySelectionContext = ""
			m.status = "已载入离线捕获；文件中的查询不会自动执行。"
		})
	}()
}

func readQueryAnalysisFile(path string) (desktop.QueryAnalysisDocument, error) {
	file, err := os.Open(path)
	if err != nil {
		return desktop.QueryAnalysisDocument{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return desktop.QueryAnalysisDocument{}, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > query.QueryByteLimit {
		return desktop.QueryAnalysisDocument{}, fmt.Errorf("文件不是受支持大小的普通分析文件")
	}
	document, err := desktop.ReadQueryAnalysis(file, info.Size())
	if err != nil {
		return desktop.QueryAnalysisDocument{}, err
	}
	return document, nil
}
