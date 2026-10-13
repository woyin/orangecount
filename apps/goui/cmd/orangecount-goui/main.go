// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package main

import (
	"log"
	"os"

	"github.com/egoist/mygo"
	"orangecount/apps/goui/internal/app"
	"orangecount/apps/goui/internal/preferences"
	"orangecount/apps/goui/internal/querypresets"
)

// qaBuild is set for the QA bundle, which has a separate bundle identifier
// and user-data directory from the development app.
var qaBuild = "false"

func main() {
	isolatedUserData, err := configureIsolatedUserDataPath(
		mygo.App.SetPath, os.Getenv("ORANGECOUNT_ISOLATED_USER_DATA_DIR"),
	)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	qa := qaBuild == "true" || os.Getenv("ORANGECOUNT_QA") == "1"
	if qa {
		mygo.App.SetName("OrangeCount Go UI QA")
	} else {
		mygo.App.SetName("OrangeCount Go UI Development")
	}

	var activeModel *app.Model
	var pendingFiles []string
	var createWindow func() *app.Model
	var openFile func(string)
	mygo.App.OnOpenFile(func(path string) {
		if openFile == nil {
			pendingFiles = append(pendingFiles, path)
			return
		}
		openFile(path)
	})
	mygo.App.WhenReady(func() {
		var preferenceStore *preferences.Store
		var queryPresetStore *querypresets.Store
		var preferenceStartupError error
		modelsByWindow := make(map[*mygo.Window]*app.Model)
		modelForWindow := func(window *mygo.Window) *app.Model { return modelsByWindow[window] }
		setApplicationMenu := func(locale string) {
			mygo.App.SetMenu(app.NewApplicationMenu(modelForWindow, createWindow, locale))
		}
		if userData, err := mygo.App.Path(mygo.PathUserData); err != nil {
			preferenceStartupError = err
			log.Printf("cannot locate app preferences: %v", err)
		} else {
			preferenceStore, queryPresetStore = openApplicationStores(
				userData, qa || isolatedUserData, preferences.OpenLegacyDefaults, log.Printf,
			)
		}
		createWindow = func() *app.Model {
			model := app.NewModel(qa)
			model.SetNewWindow(func() { createWindow() })
			model.SetPreferences(preferenceStore)
			model.SetPreferenceStartupError(preferenceStartupError)
			model.SetQueryPresets(queryPresetStore)
			window := mygo.NewWindow(model.WindowOptions())
			model.SetWindow(window)
			modelsByWindow[window] = model
			activeModel = model
			model.SetLocaleChanged(func() {
				if activeModel == model {
					setApplicationMenu(model.Locale())
				}
			})
			window.OnFocus(func() {
				activeModel = model
				setApplicationMenu(model.Locale())
			})
			window.OnFileDrop(func(event *mygo.FileDropEvent) {
				activeModel = app.RouteDroppedFiles(event.Paths, model, createWindow)
			})
			window.OnClose(func(*mygo.CloseEvent) {
				model.CancelPendingExport()
			})
			window.OnClosed(func() {
				model.Close()
				delete(modelsByWindow, window)
				if activeModel == model {
					activeModel = nil
					setApplicationMenu("zh-CN")
				}
			})
			return model
		}
		openFile = func(path string) {
			activeModel = app.RouteOpenDocument(path, activeModel, createWindow)
		}
		setApplicationMenu("zh-CN")
		createWindow()
		for _, path := range pendingFiles {
			openFile(path)
		}
		pendingFiles = nil
	})
	if err := mygo.App.Run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
