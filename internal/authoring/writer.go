// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package authoring

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"orangecount/internal/snapshot"
)

var (
	ErrSnapshotChanged = errors.New("snapshot changed")
	ErrSourceChanged   = errors.New("source file changed")
	ErrTargetMissing   = errors.New("target file unavailable")
	ErrValidation      = errors.New("ledger validation failed")
	ErrInvalidProposal = errors.New("invalid ledger proposal")
	ErrIO              = errors.New("ledger publication I/O failed")
)

type classifiedError struct {
	kind error
	err  error
}

func (e classifiedError) Error() string { return e.err.Error() }
func (e classifiedError) Unwrap() error { return e.kind }

func ioError(format string, args ...any) error {
	return classifiedError{kind: ErrIO, err: fmt.Errorf(format, args...)}
}

func invalidProposalError(format string, args ...any) error {
	return classifiedError{kind: ErrInvalidProposal, err: fmt.Errorf(format, args...)}
}

// Result reports the attempted build, backup display path, and optional receipt.
type Result struct {
	Build   snapshot.BuildResult
	Backup  string
	Receipt *Receipt
}

// Receipt identifies the exact publication that may be conditionally reverted.
type Receipt struct {
	target              string
	before              []byte
	after               []byte
	publishedSnapshotID string
}

// Writer serializes reviewed mutations against one snapshot store.
type Writer struct {
	mu    sync.Mutex
	store *snapshot.Store
}

// NewWriter constructs a reviewed ledger writer for store.
func NewWriter(store *snapshot.Store) (*Writer, error) {
	if store == nil {
		return nil, fmt.Errorf("nil snapshot store")
	}
	return &Writer{store: store}, nil
}

// Publish validates the expected snapshot and source bytes before applying change.
func (w *Writer) Publish(expectedSnapshotID string, change Change) (Result, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	current, file, err := w.resolve(expectedSnapshotID, change.target)
	if err != nil {
		return Result{}, err
	}
	before, err := os.ReadFile(file.Path)
	if err != nil {
		return Result{}, ioError("read target: %w", err)
	}
	if !bytes.Equal(before, file.Data) {
		return Result{}, fmt.Errorf("%w: %s", ErrSourceChanged, change.target)
	}
	after := change.apply(before)
	return w.publish(current, change.target, file.Path, before, after, true)
}

// Revert restores a publication only while its resulting snapshot and bytes remain current.
func (w *Writer) Revert(expectedSnapshotID string, receipt *Receipt) (Result, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if receipt == nil {
		return Result{}, fmt.Errorf("%w: receipt unavailable", ErrTargetMissing)
	}
	if expectedSnapshotID != receipt.publishedSnapshotID {
		return Result{}, fmt.Errorf("%w: publication is no longer current", ErrSnapshotChanged)
	}
	current, file, err := w.resolve(expectedSnapshotID, receipt.target)
	if err != nil {
		return Result{}, err
	}
	disk, err := os.ReadFile(file.Path)
	if err != nil {
		return Result{}, ioError("read target: %w", err)
	}
	if !bytes.Equal(disk, receipt.after) {
		return Result{}, fmt.Errorf("%w: %s", ErrSourceChanged, receipt.target)
	}
	return w.publish(current, receipt.target, file.Path, disk, receipt.before, false)
}

func (w *Writer) resolve(expectedSnapshotID, target string) (*snapshot.Snapshot, *sourceFile, error) {
	current := w.store.Current()
	if current == nil || current.ID != expectedSnapshotID {
		return nil, nil, fmt.Errorf("%w: reload before publishing", ErrSnapshotChanged)
	}
	graph := current.Graph()
	id, ok := graph.FileIDForDisplayPath(target)
	if !ok {
		return nil, nil, fmt.Errorf("%w: %s", ErrTargetMissing, target)
	}
	file := graph.File(id)
	if file == nil {
		return nil, nil, fmt.Errorf("%w: %s", ErrTargetMissing, target)
	}
	return current, &sourceFile{Path: file.Path, Data: file.Data}, nil
}

type sourceFile struct {
	Path string
	Data []byte
}

func (w *Writer) publish(current *snapshot.Snapshot, target, path string, before, after []byte, receipt bool) (Result, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Result{}, ioError("stat target: %w", err)
	}
	backupPath := path + ".orangecount.bak"
	backupDisplay := target + ".orangecount.bak"
	if err := os.WriteFile(backupPath, before, info.Mode().Perm()); err != nil {
		return Result{}, ioError("write backup: %w", err)
	}
	if err := atomicWrite(path, after, info.Mode().Perm()); err != nil {
		return Result{Backup: backupDisplay}, ioError("replace target: %w", err)
	}
	built := w.store.Reload(current.EntryPath, snapshot.BuildOptions{})
	if built.Snapshot == nil {
		if restoreErr := atomicWrite(path, before, info.Mode().Perm()); restoreErr != nil {
			return Result{Build: built, Backup: backupDisplay}, ioError("restore target after validation failure: %w", restoreErr)
		}
		_ = w.store.Reload(current.EntryPath, snapshot.BuildOptions{})
		return Result{Build: built, Backup: backupDisplay}, fmt.Errorf("%w", ErrValidation)
	}
	result := Result{Build: built, Backup: backupDisplay}
	if receipt {
		result.Receipt = &Receipt{
			target: target, before: append([]byte(nil), before...), after: append([]byte(nil), after...),
			publishedSnapshotID: built.Snapshot.ID,
		}
	}
	return result, nil
}

func atomicWrite(path string, content []byte, mode os.FileMode) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".orangecount-write-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(content); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
