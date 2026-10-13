// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package app

import (
	"errors"
	"fmt"

	"github.com/egoist/mygo"
	"orangecount/apps/goui/internal/platform"
)

func (m *Model) chooseSourceEditor() <-chan error {
	result := make(chan error, 1)
	if m.closed.Load() || m.picking {
		return completedSourceEditorTask(nil)
	}
	store := m.preferenceStore
	if store == nil {
		m.status = m.tr("来源编辑器偏好存储不可用。")
		return completedSourceEditorTask(errors.New("source editor preferences are unavailable"))
	}
	m.picking = true
	window := m.window
	canceledStatus := m.tr("已取消来源编辑器选择。")
	errorPrefix := m.tr("无法选择来源编辑器：")
	successTemplate := m.tr("来源编辑器已设为 %s。")
	panelTitle := m.tr("选择来源编辑器")
	panelMessage := m.tr("选择用于打开账本来源文件的应用")
	panelButton := m.tr("选择")
	m.preferenceWork.Add(1)
	go func() {
		defer m.preferenceWork.Done()
		path, err := m.pickSourceEditorPath(window, panelTitle, panelMessage, panelButton)
		var editor platform.SourceEditor
		if err == nil && path != "" {
			inspect := m.sourceEditorInspector
			if inspect == nil {
				inspect = platform.SourceEditorFromApplication
			}
			editor, err = inspect(path)
			if err == nil {
				if editor.BundleID == "" {
					err = platform.ErrSourceEditorInvalid
				} else {
					err = store.SetSourceEditorBundleID(editor.BundleID)
				}
			}
		}
		status := canceledStatus
		if err != nil {
			status = errorPrefix + err.Error()
		} else if path != "" {
			status = fmt.Sprintf(successTemplate, editor.Name)
		}
		apply := func() {
			if m.closed.Load() {
				return
			}
			m.picking = false
			m.status = status
		}
		if window != nil {
			window.Update(apply)
		} else {
			apply()
		}
		result <- err
		close(result)
	}()
	return result
}

func (m *Model) restoreSystemSourceEditor() <-chan error {
	return m.persistSourceEditorPreference("", m.tr("已恢复来源文件的系统默认应用。"))
}

func (m *Model) persistSourceEditorPreference(bundleID, success string) <-chan error {
	result := make(chan error, 1)
	if m.closed.Load() || m.picking {
		return completedSourceEditorTask(nil)
	}
	store := m.preferenceStore
	if store == nil {
		m.status = m.tr("来源编辑器偏好存储不可用。")
		return completedSourceEditorTask(errors.New("source editor preferences are unavailable"))
	}
	m.picking = true
	window := m.window
	errorPrefix := m.tr("无法保存来源编辑器偏好：")
	m.preferenceWork.Add(1)
	go func() {
		defer m.preferenceWork.Done()
		err := store.SetSourceEditorBundleID(bundleID)
		status := success
		if err != nil {
			status = errorPrefix + err.Error()
		}
		apply := func() {
			if m.closed.Load() {
				return
			}
			m.picking = false
			m.status = status
		}
		if window != nil {
			window.Update(apply)
		} else {
			apply()
		}
		result <- err
		close(result)
	}()
	return result
}

func completedSourceEditorTask(err error) <-chan error {
	result := make(chan error, 1)
	result <- err
	close(result)
	return result
}

func (m *Model) pickSourceEditorPath(parent *mygo.Window, title, message, button string) (string, error) {
	if m.sourceEditorPicker != nil {
		return m.sourceEditorPicker(parent)
	}
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent: parent, Title: title, Message: message, ButtonLabel: button,
		Directory: true, TreatPackagesAsDirectories: true,
	})
	if err != nil || len(paths) == 0 {
		return "", err
	}
	return paths[0], nil
}
