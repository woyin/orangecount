// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package sourceexport

import (
	"fmt"
	"orangecount/internal/documents"
	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
	"os"
	"path/filepath"
	"reflect"
)

// Write publishes one file or a complete tree in an existing target parent.
// Multi-file publication requires a new directory; existing trees are refused.
func (p *Plan) Write(target string) error { return p.write(target, nil) }

// fault is a filesystem transaction test seam; production always passes nil.
func (p *Plan) write(target string, fault func(stage, path string) error) error {
	if p == nil {
		return fmt.Errorf("E-EXPORT-PLAN: nil plan")
	}
	absolute, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	parent := filepath.Dir(absolute)
	realParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return fmt.Errorf("E-EXPORT-TARGET: parent must already exist: %w", err)
	}
	parentHandle, err := os.Open(realParent)
	if err != nil {
		return err
	}
	defer parentHandle.Close()
	parentInfo, err := parentHandle.Stat()
	if err != nil || !parentInfo.IsDir() {
		return fmt.Errorf("E-EXPORT-TARGET: invalid parent")
	}
	// Work exclusively under the pinned canonical parent; check the user's
	// original spelling again before publication to detect retargeted symlinks.
	destination := filepath.Join(realParent, filepath.Base(absolute))
	if err = p.protect(absolute, destination); err != nil {
		return err
	}
	before, statErr := os.Lstat(destination)
	if statErr != nil && !os.IsNotExist(statErr) {
		return statErr
	}
	if before != nil {
		if len(p.files) > 1 || !before.Mode().IsRegular() || linkCount(before) > 1 {
			return fmt.Errorf("E-EXPORT-TARGET: requires ordinary file or new tree")
		}
	}
	stage, err := os.MkdirTemp(realParent, ".orangecount-export-")
	if err != nil {
		return err
	}
	stageInfo, err := os.Stat(stage)
	if err != nil {
		return err
	}
	defer func() {
		directory, err := pathForDirectory(parentHandle, realParent)
		if err != nil {
			return
		}
		owned := filepath.Join(directory, filepath.Base(stage))
		info, err := os.Stat(owned)
		if err == nil && os.SameFile(stageInfo, info) {
			_ = os.RemoveAll(owned)
		}
	}()
	currentParent, err := os.Stat(realParent)
	if err != nil || !os.SameFile(parentInfo, currentParent) {
		return fmt.Errorf("E-EXPORT-TARGET: parent changed during staging")
	}
	event := func(kind, path string) error {
		if fault != nil {
			return fault(kind, path)
		}
		return nil
	}
	for _, f := range p.files {
		path := filepath.Join(stage, f.path)
		if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
		if err = event("write", path); err != nil {
			return err
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		n, werr := file.Write(f.data)
		if werr == nil && n != len(f.data) {
			werr = fmt.Errorf("short source write")
		}
		if werr == nil {
			werr = event("written", path)
		}
		if werr == nil {
			werr = event("sync", path)
		}
		if werr == nil {
			werr = file.Sync()
		}
		cerr := file.Close()
		if werr != nil {
			return werr
		}
		if cerr != nil {
			return cerr
		}
	}
	reloadOptions := p.options.Build
	if !p.options.Reverse {
		reloadOptions = snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}}
	}
	loaded := snapshot.BuildWithOptions(filepath.Join(stage, p.entry), reloadOptions)
	if loaded.Snapshot == nil {
		return fmt.Errorf("E-EXPORT-RELOAD: %w", diagnosticsError(loaded.Diagnostics))
	}
	fingerprint, err := financialFingerprint(loaded.Snapshot.Evaluation())
	if err != nil {
		return err
	}
	if fingerprint != p.fingerprint {
		return fmt.Errorf("E-EXPORT-RELOAD: accounting or intermediate posting semantics changed")
	}
	// Sync each staging directory after file creation, before the publish point.
	err = filepath.WalkDir(stage, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		err = f.Sync()
		closeErr := f.Close()
		if err != nil {
			return err
		}
		return closeErr
	})
	if err != nil {
		return err
	}
	if err = event("publish", destination); err != nil {
		return err
	}
	nowParent, err := filepath.EvalSymlinks(parent)
	if err != nil || nowParent != realParent {
		return fmt.Errorf("E-EXPORT-TARGET: parent changed")
	}
	nowInfo, err := os.Stat(realParent)
	if err != nil || !os.SameFile(parentInfo, nowInfo) {
		return fmt.Errorf("E-EXPORT-TARGET: parent identity changed")
	}
	if err = p.protect(absolute, destination); err != nil {
		return err
	}
	after, err := os.Lstat(destination)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if (before == nil) != (after == nil) || (before != nil && (!os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()))) {
		return fmt.Errorf("E-EXPORT-TARGET: target changed")
	}
	if before != nil && linkCount(after) > 1 {
		return fmt.Errorf("E-EXPORT-TARGET: target became hardlink")
	}
	if err = event("rename", destination); err != nil {
		return err
	}
	if len(p.files) > 1 {
		return publishNewTree(parentHandle, filepath.Base(stage), filepath.Base(destination))
	}
	staged := filepath.Join(filepath.Base(stage), p.entry)
	if before == nil {
		return publishFile(parentHandle, staged, filepath.Base(destination), true)
	}
	return replaceFile(parentHandle, staged, destination, event)
}

// replaceFile publishes over an existing ordinary file without destroying it.
//
// Replacement by rename unlinks the old inode, and no pre-publication check can
// tell whether that inode is a Finder alias: the alias record lives in an
// extended attribute that only a cgo build can read, and any check is separated
// from the rename by a window in which the target can change. Moving the old
// file aside removes the question: the previous inode keeps a name under the
// backup path, so an alias, or anything else the user placed there, survives.
// This is the same <file>.orangecount.bak convention the ledger writer uses.
func replaceFile(parent *os.File, staged, destination string, event func(kind, path string) error) error {
	backup := destination + ".orangecount.bak"
	if info, err := os.Lstat(backup); err == nil {
		// The backup slot belongs to this tool, but never destroy a name that is
		// not an ordinary single-link file.
		if !info.Mode().IsRegular() || linkCount(info) > 1 {
			return fmt.Errorf("E-EXPORT-TARGET: move %q aside first; it is not an ordinary file", backup)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(destination, backup); err != nil {
		return err
	}
	// The event seam sits between the move and the publish so that the recovery
	// path below is exercised by a test instead of only by production failures.
	if err := event("replaced", destination); err != nil {
		if restoreErr := os.Rename(backup, destination); restoreErr != nil {
			return fmt.Errorf("%w (the previous file is kept at %q)", err, backup)
		}
		return err
	}
	if err := publishFile(parent, staged, filepath.Base(destination), true); err != nil {
		if restoreErr := os.Rename(backup, destination); restoreErr != nil {
			return fmt.Errorf("%w (the previous file is kept at %q)", err, backup)
		}
		return err
	}
	return nil
}
func (p *Plan) protect(literal, target string) error {
	if p.graph.ChangedOnDisk() {
		return fmt.Errorf("E-EXPORT-STALE: source changed after preparation")
	}
	fresh, err := source.LoadGraph(p.graph.Path(p.graph.Entry))
	if err != nil || snapshot.InputIdentity(fresh) != snapshot.InputIdentity(p.graph) {
		return fmt.Errorf("E-EXPORT-STALE: source graph or include glob changed")
	}
	if contained(p.root, target) {
		return fmt.Errorf("E-EXPORT-PROTECTED: target lies inside source directory")
	}
	targetInfo, _ := os.Stat(target)
	for _, item := range p.protected {
		canonical, err := documents.CanonicalPath(item.path)
		if err != nil {
			return fmt.Errorf("E-EXPORT-PROTECTED: cannot resolve protected path: %w", err)
		}
		if filepath.Clean(literal) == filepath.Clean(item.path) || target == canonical || (len(p.files) > 1 && contained(target, canonical)) {
			return fmt.Errorf("E-EXPORT-PROTECTED: target overlaps source or declared attachment")
		}
		current, _ := os.Stat(item.path)
		if targetInfo != nil && ((current != nil && os.SameFile(current, targetInfo)) || (item.info != nil && os.SameFile(item.info, targetInfo))) {
			return fmt.Errorf("E-EXPORT-PROTECTED: target aliases source or declared attachment")
		}
	}
	return nil
}
func contained(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && insideRelative(rel)
}
func linkCount(info os.FileInfo) uint64 {
	v := reflect.ValueOf(info.Sys())
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct {
		f := v.FieldByName("Nlink")
		if f.IsValid() {
			switch f.Kind() {
			case reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
				return f.Uint()
			case reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
				return uint64(f.Int())
			}
		}
	}
	return 1
}
