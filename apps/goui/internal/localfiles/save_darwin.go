// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

//go:build darwin

package localfiles

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"orangecount/apps/goui/internal/platform"
)

type saveHooks struct {
	write        func(io.Writer, []byte) error
	beforeCommit func()
}

// Save atomically writes the frozen content to an absolute local path. It
// checks cancellation before opening files and again immediately before the
// rename commit point. Cancellation after that final check cannot undo a save.
func (c *Capture) Save(ctx context.Context, destination string) error {
	return saveCapture(ctx, c, destination, saveHooks{})
}

func saveCapture(ctx context.Context, capture *Capture, destination string, hooks saveHooks) error {
	if capture == nil || ctx == nil {
		return ErrInvalidCapture
	}
	if err := checkCancelled(ctx); err != nil {
		return err
	}
	if destination == "" || !filepath.IsAbs(destination) {
		return ErrInvalidTarget
	}
	destination = filepath.Clean(destination)
	name := filepath.Base(destination)
	if name == "" || name == "." || name == ".." || name == string(filepath.Separator) {
		return ErrInvalidTarget
	}
	originalParent := filepath.Dir(destination)
	canonicalParent, err := filepath.EvalSymlinks(originalParent)
	if err != nil {
		return fmt.Errorf("resolve save parent: %w: %w", ErrFilesystem, err)
	}
	canonicalParent = filepath.Clean(canonicalParent)
	targetPath := filepath.Join(canonicalParent, name)
	if err := validateTargetAt(nil, capture, targetPath, name, nil, "", ""); err != nil {
		return err
	}

	root, err := os.OpenRoot(canonicalParent)
	if err != nil {
		return fmt.Errorf("open save parent: %w: %w", ErrFilesystem, err)
	}
	defer root.Close()
	rootInfo, err := root.Stat(".")
	if err != nil {
		return fmt.Errorf("stat opened save parent: %w: %w", ErrFilesystem, err)
	}
	if !rootInfo.IsDir() {
		return ErrInvalidTarget
	}
	if err := verifyParent(root, rootInfo, originalParent, canonicalParent); err != nil {
		return err
	}
	if err := validateTargetAt(root, capture, targetPath, name, rootInfo, originalParent, canonicalParent); err != nil {
		return err
	}
	if err := verifyParent(root, rootInfo, originalParent, canonicalParent); err != nil {
		return err
	}

	temporary, file, err := createTemporary(root)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = root.Remove(temporary)
		}
	}()
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
	}()

	write := hooks.write
	if write == nil {
		write = writeAll
	}
	if err := write(file, capture.content); err != nil {
		return fmt.Errorf("write temporary export: %w: %w", ErrFilesystem, err)
	}
	if err := checkCancelled(ctx); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync temporary export: %w: %w", ErrFilesystem, err)
	}
	if err := file.Close(); err != nil {
		closed = true
		return fmt.Errorf("close temporary export: %w: %w", ErrFilesystem, err)
	}
	closed = true

	if hooks.beforeCommit != nil {
		hooks.beforeCommit()
	}
	if err := verifyParent(root, rootInfo, originalParent, canonicalParent); err != nil {
		return err
	}
	if err := validateTargetAt(root, capture, targetPath, name, rootInfo, originalParent, canonicalParent); err != nil {
		return err
	}
	if err := verifyParent(root, rootInfo, originalParent, canonicalParent); err != nil {
		return err
	}
	if err := checkCancelled(ctx); err != nil {
		return err
	}
	if err := root.Rename(temporary, name); err != nil {
		return fmt.Errorf("commit export: %w: %w", ErrFilesystem, err)
	}
	committed = true
	return nil
}

func verifyParent(root *os.Root, opened os.FileInfo, originalParent, canonicalParent string) error {
	resolved, err := filepath.EvalSymlinks(originalParent)
	if err != nil || filepath.Clean(resolved) != canonicalParent {
		return fmt.Errorf("resolve original save parent: %w", ErrParentChanged)
	}
	current, err := os.Stat(canonicalParent)
	if err != nil || !os.SameFile(opened, current) {
		return fmt.Errorf("save parent identity changed: %w", ErrParentChanged)
	}
	anchored, err := root.Stat(".")
	if err != nil || !os.SameFile(opened, anchored) {
		return fmt.Errorf("opened save parent identity changed: %w", ErrParentChanged)
	}
	return nil
}

func validateTargetAt(root *os.Root, capture *Capture, targetPath, name string, rootInfo os.FileInfo, originalParent, canonicalParent string) error {
	var info os.FileInfo
	var err error
	if root == nil {
		info, err = os.Lstat(targetPath)
	} else {
		info, err = root.Lstat(name)
	}
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
			if root != nil {
				if parentErr := verifyParent(root, rootInfo, originalParent, canonicalParent); parentErr != nil {
					return parentErr
				}
			}
			protected, protectErr := capture.protects(targetPath, nil)
			if protectErr != nil {
				return protectErr
			}
			if protected {
				return ErrProtectedTarget
			}
			return nil
		}
		return fmt.Errorf("inspect save target: %w: %w", ErrFilesystem, err)
	}
	if !info.Mode().IsRegular() {
		return ErrUnsupportedTarget
	}
	alias, err := platform.IsFinderAlias(targetPath)
	if err != nil {
		return fmt.Errorf("inspect Finder alias: %w: %w", ErrUnsupportedTarget, err)
	}
	if alias {
		return ErrUnsupportedTarget
	}
	if root != nil {
		if err := verifyParent(root, rootInfo, originalParent, canonicalParent); err != nil {
			return err
		}
		current, err := root.Lstat(name)
		if err != nil || !current.Mode().IsRegular() || !os.SameFile(info, current) {
			return ErrUnsupportedTarget
		}
		info, err = root.Stat(name)
	} else {
		info, err = os.Stat(targetPath)
	}
	if err != nil {
		return fmt.Errorf("inspect save target identity: %w: %w", ErrFilesystem, err)
	}
	protected, err := capture.protects(targetPath, info)
	if err != nil {
		return err
	}
	if protected {
		return ErrProtectedTarget
	}
	return nil
}

func createTemporary(root *os.Root) (string, *os.File, error) {
	for attempt := 0; attempt < 8; attempt++ {
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			return "", nil, fmt.Errorf("generate temporary export name: %w: %w", ErrFilesystem, err)
		}
		name := ".orangecount-export-" + hex.EncodeToString(random[:])
		file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0o600)
		if err == nil {
			return name, file, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return "", nil, fmt.Errorf("create temporary export: %w: %w", ErrFilesystem, err)
		}
	}
	return "", nil, fmt.Errorf("create unique temporary export: %w", ErrFilesystem)
}

func writeAll(writer io.Writer, content []byte) error {
	for len(content) > 0 {
		written, err := writer.Write(content)
		if written < 0 || written > len(content) {
			return fmt.Errorf("invalid write count %d", written)
		}
		content = content[written:]
		if err != nil {
			if errors.Is(err, syscall.EINTR) {
				continue
			}
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}

func checkCancelled(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: %w", ErrCancelled, err)
	}
	return nil
}
