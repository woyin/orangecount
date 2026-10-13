// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

//go:build darwin

package localfiles

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"orangecount/apps/goui/internal/platform"
)

func TestSaveAtomicallyReplacesRegularFileWithFrozenBytes(t *testing.T) {
	root := t.TempDir()
	source := writeFile(t, filepath.Join(root, "source.bean"), "ledger contents")
	target := writeFile(t, filepath.Join(root, "export.csv"), "old complete contents")
	capture, err := NewCapture([]byte("new complete contents"), []string{source})
	if err != nil {
		t.Fatal(err)
	}
	if err := capture.Save(context.Background(), target); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "new complete contents" {
		t.Fatalf("saved target = %q, %v", got, err)
	}
	info, err := os.Stat(target)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("saved target mode = %v, %v; want 0600", info, err)
	}
	newTarget := filepath.Join(root, "new-export.csv")
	if err := capture.Save(context.Background(), newTarget); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(newTarget)
	if err != nil || string(got) != "new complete contents" {
		t.Fatalf("new saved target = %q, %v", got, err)
	}
	assertNoTemporaryFiles(t, root)
}

func TestSaveRejectsProtectedPathHardlinkAndMovedSourceIdentity(t *testing.T) {
	root := t.TempDir()
	source := writeFile(t, filepath.Join(root, "source.bean"), "source bytes")
	capture, err := NewCapture([]byte("export bytes"), []string{source})
	if err != nil {
		t.Fatal(err)
	}
	if err := capture.Save(context.Background(), source); !errors.Is(err, ErrProtectedTarget) {
		t.Fatalf("saving over source error = %v, want %v", err, ErrProtectedTarget)
	}
	linked := filepath.Join(root, "linked.csv")
	if err := os.Link(source, linked); err != nil {
		t.Fatal(err)
	}
	if err := capture.Save(context.Background(), linked); !errors.Is(err, ErrProtectedTarget) {
		t.Fatalf("saving over source hardlink error = %v, want %v", err, ErrProtectedTarget)
	}
	moved := filepath.Join(root, "moved.bean")
	if err := os.Rename(source, moved); err != nil {
		t.Fatal(err)
	}
	if err := capture.Save(context.Background(), linked); !errors.Is(err, ErrProtectedTarget) {
		t.Fatalf("saving over hardlink after source moved error = %v, want %v", err, ErrProtectedTarget)
	}
	got, err := os.ReadFile(moved)
	if err != nil || string(got) != "source bytes" {
		t.Fatalf("moved source = %q, %v", got, err)
	}
}

func TestProtectionRefreshRetainsMovedSourceIdentity(t *testing.T) {
	root := t.TempDir()
	source := writeFile(t, filepath.Join(root, "source.bean"), "source bytes")
	refreshedSource := writeFile(t, filepath.Join(root, "refreshed.bean"), "refreshed source")
	capture, err := NewCapture([]byte("export"), []string{source})
	if err != nil {
		t.Fatal(err)
	}
	refreshed, err := capture.WithProtectedPaths([]string{refreshedSource})
	if err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(root, "moved.bean")
	if err := os.Rename(source, moved); err != nil {
		t.Fatal(err)
	}
	hardlink := filepath.Join(root, "hardlink.csv")
	if err := os.Link(moved, hardlink); err != nil {
		t.Fatal(err)
	}
	if err := refreshed.Save(context.Background(), hardlink); !errors.Is(err, ErrProtectedTarget) {
		t.Fatalf("refreshed capture forgot moved source identity: %v", err)
	}
}

func TestSaveProtectsMissingSourcePathAndRejectsSymlinkOrDirectory(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "not-yet-created.bean")
	capture, err := NewCapture([]byte("export"), []string{source})
	if err != nil {
		t.Fatal(err)
	}
	if err := capture.Save(context.Background(), source); !errors.Is(err, ErrProtectedTarget) {
		t.Fatalf("saving to missing protected path error = %v, want %v", err, ErrProtectedTarget)
	}

	other := writeFile(t, filepath.Join(root, "other.txt"), "keep")
	symlink := filepath.Join(root, "symlink.csv")
	if err := os.Symlink(other, symlink); err != nil {
		t.Fatal(err)
	}
	if err := capture.Save(context.Background(), symlink); !errors.Is(err, ErrUnsupportedTarget) {
		t.Fatalf("saving over symlink error = %v, want %v", err, ErrUnsupportedTarget)
	}
	directory := filepath.Join(root, "directory.csv")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := capture.Save(context.Background(), directory); !errors.Is(err, ErrUnsupportedTarget) {
		t.Fatalf("saving over directory error = %v, want %v", err, ErrUnsupportedTarget)
	}
	got, err := os.ReadFile(other)
	if err != nil || string(got) != "keep" {
		t.Fatalf("symlink referent changed = %q, %v", got, err)
	}
}

func TestSaveRejectsFinderAliasTarget(t *testing.T) {
	root := t.TempDir()
	source := writeFile(t, filepath.Join(root, "source.bean"), "source")
	target := writeFile(t, filepath.Join(root, "target.txt"), "target")
	alias := filepath.Join(root, "target-alias")
	createFinderAlias(t, target, alias)
	isAlias, err := platform.IsFinderAlias(alias)
	if err != nil || !isAlias {
		t.Fatalf("IsFinderAlias(%q) = %t, %v", alias, isAlias, err)
	}
	capture, err := NewCapture([]byte("export"), []string{source})
	if err != nil {
		t.Fatal(err)
	}
	if err := capture.Save(context.Background(), alias); !errors.Is(err, ErrUnsupportedTarget) {
		t.Fatalf("saving over Finder alias error = %v, want %v", err, ErrUnsupportedTarget)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "target" {
		t.Fatalf("Finder alias target changed = %q, %v", got, err)
	}
}

func TestSaveWriteFailurePreservesDestinationAndCleansTemporaryFile(t *testing.T) {
	root := t.TempDir()
	source := writeFile(t, filepath.Join(root, "source.bean"), "source")
	target := writeFile(t, filepath.Join(root, "export.csv"), "previous complete file")
	capture, err := NewCapture([]byte("new export contents"), []string{source})
	if err != nil {
		t.Fatal(err)
	}
	writeErr := syscall.EIO
	err = saveCapture(context.Background(), capture, target, saveHooks{
		write: func(writer io.Writer, content []byte) error {
			if _, err := writer.Write(content[:4]); err != nil {
				return err
			}
			return writeErr
		},
	})
	if !errors.Is(err, ErrFilesystem) || !errors.Is(err, writeErr) {
		t.Fatalf("partial write error = %v, want filesystem EIO", err)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "previous complete file" {
		t.Fatalf("target after failed write = %q, %v", got, err)
	}
	got, err = os.ReadFile(source)
	if err != nil || string(got) != "source" {
		t.Fatalf("source after failed write = %q, %v", got, err)
	}
	assertNoTemporaryFiles(t, root)
}

func TestSaveCancellationBeforeCommitPreservesExistingTarget(t *testing.T) {
	root := t.TempDir()
	source := writeFile(t, filepath.Join(root, "source.bean"), "source")
	target := writeFile(t, filepath.Join(root, "export.csv"), "previous complete file")
	capture, err := NewCapture([]byte("new export"), []string{source})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	err = saveCapture(ctx, capture, target, saveHooks{beforeCommit: cancel})
	if !errors.Is(err, ErrCancelled) || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel before commit error = %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "previous complete file" {
		t.Fatalf("target after cancel = %q, %v", got, err)
	}
	assertNoTemporaryFiles(t, root)

	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	if err := capture.Save(ctx, filepath.Join(root, "cancelled.csv")); !errors.Is(err, ErrCancelled) {
		t.Fatalf("pre-cancelled Save error = %v", err)
	}
}

func TestSaveRejectsParentDirectoryReplacementBeforeCommit(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "exports")
	if err := os.Mkdir(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	oldTarget := writeFile(t, filepath.Join(parent, "export.csv"), "previous complete file")
	source := writeFile(t, filepath.Join(root, "source.bean"), "source")
	capture, err := NewCapture([]byte("new export"), []string{source})
	if err != nil {
		t.Fatal(err)
	}
	movedParent := filepath.Join(root, "exports-moved")
	err = saveCapture(context.Background(), capture, oldTarget, saveHooks{beforeCommit: func() {
		if renameErr := os.Rename(parent, movedParent); renameErr != nil {
			t.Errorf("rename parent: %v", renameErr)
			return
		}
		if mkdirErr := os.Mkdir(parent, 0o700); mkdirErr != nil {
			t.Errorf("replace parent: %v", mkdirErr)
		}
	}})
	if !errors.Is(err, ErrParentChanged) {
		t.Fatalf("parent replacement error = %v, want %v", err, ErrParentChanged)
	}
	got, readErr := os.ReadFile(filepath.Join(movedParent, filepath.Base(oldTarget)))
	if readErr != nil || string(got) != "previous complete file" {
		t.Fatalf("target in moved parent after rejection = %q, %v", got, readErr)
	}
	entries, readErr := os.ReadDir(movedParent)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 1 || entries[0].Name() != "export.csv" {
		t.Fatalf("moved parent contains temp or unexpected files: %v", entries)
	}
}

func TestSaveRevalidatesDestinationProtectionImmediatelyBeforeCommit(t *testing.T) {
	root := t.TempDir()
	source := writeFile(t, filepath.Join(root, "source.bean"), "source bytes")
	target := writeFile(t, filepath.Join(root, "export.csv"), "old export")
	capture, err := NewCapture([]byte("new export"), []string{source})
	if err != nil {
		t.Fatal(err)
	}
	err = saveCapture(context.Background(), capture, target, saveHooks{beforeCommit: func() {
		if removeErr := os.Remove(target); removeErr != nil {
			t.Errorf("remove destination for race simulation: %v", removeErr)
			return
		}
		if linkErr := os.Link(source, target); linkErr != nil {
			t.Errorf("replace destination with protected hardlink: %v", linkErr)
		}
	}})
	if !errors.Is(err, ErrProtectedTarget) {
		t.Fatalf("replacement destination error = %v, want %v", err, ErrProtectedTarget)
	}
	got, err := os.ReadFile(source)
	if err != nil || string(got) != "source bytes" {
		t.Fatalf("source after rejected destination replacement = %q, %v", got, err)
	}
	got, err = os.ReadFile(target)
	if err != nil || string(got) != "source bytes" {
		t.Fatalf("hardlink after rejected destination replacement = %q, %v", got, err)
	}
	assertNoTemporaryFiles(t, root)
}

func TestWriteAllHandlesShortWritesAndInterruptedWrites(t *testing.T) {
	writer := &shortWriter{interruptOnce: true}
	if err := writeAll(writer, []byte("complete")); err != nil {
		t.Fatal(err)
	}
	if got := string(writer.content); got != "complete" {
		t.Fatalf("writeAll content = %q", got)
	}
	if err := writeAll(zeroWriter{}, []byte("one byte")); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("zero-progress write error = %v, want %v", err, io.ErrShortWrite)
	}
}

func writeFile(t *testing.T, path, content string) string {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertNoTemporaryFiles(t *testing.T, directory string) {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".orangecount-export-") {
			t.Fatalf("temporary export was left behind: %s", entry.Name())
		}
	}
}

type shortWriter struct {
	content       []byte
	interruptOnce bool
}

func (w *shortWriter) Write(p []byte) (int, error) {
	if w.interruptOnce {
		w.interruptOnce = false
		return 0, syscall.EINTR
	}
	count := len(p)
	if count > 2 {
		count = 2
	}
	w.content = append(w.content, p[:count]...)
	return count, nil
}

type zeroWriter struct{}

func (zeroWriter) Write([]byte) (int, error) { return 0, nil }

func createFinderAlias(t *testing.T, targetPath, aliasPath string) {
	t.Helper()
	if _, err := platform.IsFinderAlias(targetPath); err != nil {
		t.Fatal(err)
	}
	problem := ""
	withLocalfilesObjCPool(func() {
		nsURL := objc.ID(objc.GetClass("NSURL"))
		targetURL := nsURL.Send(localfilesSelector("fileURLWithPath:"), localfilesString(targetPath))
		if targetURL == 0 {
			problem = "create target URL"
			return
		}
		data := objc.Send[objc.ID](targetURL, localfilesSelector("bookmarkDataWithOptions:includingResourceValuesForKeys:relativeToURL:error:"), uintptr(1<<10), objc.ID(0), objc.ID(0), objc.ID(0))
		if data == 0 {
			problem = "create alias bookmark data"
			return
		}
		aliasURL := nsURL.Send(localfilesSelector("fileURLWithPath:"), localfilesString(aliasPath))
		var failure objc.ID
		written := objc.Send[bool](nsURL, localfilesSelector("writeBookmarkData:toURL:options:error:"), data, aliasURL, uintptr(0), unsafe.Pointer(&failure))
		if !written {
			problem = "write Finder alias"
			if failure != 0 {
				description := objc.Send[objc.ID](failure, localfilesSelector("localizedDescription"))
				if description != 0 {
					problem = objc.Send[string](description, localfilesSelector("UTF8String"))
				}
			}
		}
	})
	if problem != "" {
		t.Fatal(fmt.Errorf("create Finder alias fixture: %s", problem))
	}
}

func withLocalfilesObjCPool(fn func()) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(localfilesSelector("alloc")).Send(localfilesSelector("init"))
	defer pool.Send(localfilesSelector("drain"))
	fn()
}

func localfilesString(value string) objc.ID {
	return objc.ID(objc.GetClass("NSString")).Send(localfilesSelector("stringWithUTF8String:"), value+"\x00")
}

func localfilesSelector(name string) objc.SEL { return objc.RegisterName(name) }
