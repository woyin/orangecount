// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"orangecount/apps/goui/internal/localfiles"
)

func (m *Model) exportPreferencesBackup() {
	if m.closed.Load() {
		return
	}
	if m.preferenceStore == nil {
		m.status = m.tr("偏好存储不可用，无法导出备份。")
		return
	}
	if m.preferenceBackupCancel != nil {
		m.status = m.tr("偏好备份正在进行。")
		return
	}
	if m.window == nil && m.preferenceBackupSaveDialog == nil {
		m.status = m.tr("偏好备份需要一个原生窗口。")
		return
	}
	if m.preferenceBackupSequence == ^uint64(0) {
		m.status = m.tr("偏好备份编号已耗尽，请重新打开窗口。")
		return
	}
	m.preferenceBackupSequence++
	sequence := m.preferenceBackupSequence
	ctx, cancel := context.WithCancel(context.Background())
	m.preferenceBackupCancel = cancel
	m.status = m.tr("正在准备偏好备份…")
	window := m.window
	store := m.preferenceStore
	dialog := m.preferenceBackupSaveDialog
	options := mygo.SaveDialogOptions{
		Parent:      window,
		Title:       m.tr("导出偏好备份"),
		ButtonLabel: m.tr("导出"),
		DefaultPath: preferencesBackupDefaultPath(),
		Filters:     []mygo.FileFilter{{Name: m.tr("JSON 偏好备份"), Extensions: []string{"json"}}},
		Message:     m.tr("备份包含最近账本路径、每账本读取偏好、附件目录书签和应用设置；不包含账本内容、查询草稿或查询结果。"),
	}
	m.preferenceWork.Add(1)
	go func() {
		defer m.preferenceWork.Done()
		defer cancel()

		snapshot, err := store.BackupSnapshot()
		var capture *localfiles.Capture
		if err == nil {
			capture, err = localfiles.NewCapture(snapshot.Content, snapshot.ProtectedPaths)
		}
		path := ""
		if err == nil && ctx.Err() == nil {
			if dialog == nil {
				dialog = mygo.Dialog.SaveWithContext
			}
			path, err = dialog(ctx, options)
		}
		if err == nil && path != "" && ctx.Err() == nil {
			if !strings.EqualFold(filepath.Ext(path), ".json") {
				path += ".json"
			}
			err = capture.Save(ctx, path)
		} else if err == nil && ctx.Err() != nil {
			err = context.Canceled
		}
		cancelledAfterCommit := err == nil && path != "" && ctx.Err() != nil
		if m.onPreferenceBackupResult != nil {
			m.onPreferenceBackupResult(path, err)
		}
		if window == nil {
			return
		}
		window.Update(func() {
			if m.closed.Load() || m.preferenceBackupSequence != sequence {
				return
			}
			m.preferenceBackupCancel = nil
			switch {
			case errors.Is(err, context.Canceled):
				m.status = m.tr("已取消偏好备份导出。")
			case err != nil:
				m.status = m.tr("无法导出偏好备份：") + err.Error()
			case path == "":
				m.status = m.tr("已取消偏好备份导出。")
			default:
				message := m.tr("偏好备份已导出：") + filepath.Base(path)
				if cancelledAfterCommit {
					message += m.tr("（取消发生在文件提交之后）")
				}
				m.status = message
			}
		})
	}()
}

func preferencesBackupDefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "OrangeCount-preferences-backup.json"
	}
	return filepath.Join(home, "OrangeCount-preferences-backup.json")
}
