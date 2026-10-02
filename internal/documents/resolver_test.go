// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package documents_test

import (
	"errors"
	"orangecount/internal/documents"
	"os"
	"path/filepath"
	"testing"
)

func TestResolverChecksDeclarationBaseRootsAndSymlinkTargets(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "receipt.pdf")
	if err := os.WriteFile(file, []byte("public attachment"), 0600); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(outside, "external.pdf")
	if err := os.WriteFile(external, []byte("public outside file"), 0600); err != nil {
		t.Fatal(err)
	}
	for name, target := range map[string]string{"inside-link": file, "outside-link": external, "outside-directory": outside, "dangling-outside": filepath.Join(outside, "missing.pdf")} {
		if err := os.Symlink(target, filepath.Join(nested, name)); err != nil {
			t.Fatal(err)
		}
	}
	resolver, err := documents.NewResolver([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(nested, "main.bean")
	for _, test := range []struct {
		name, path string
		status     documents.Status
	}{
		{"relative", "../receipt.pdf", documents.Available},
		{"absolute", file, documents.Available},
		{"inside-link", "inside-link", documents.Available},
		{"missing", "missing.pdf", documents.Missing},
		{"directory", ".", documents.UnsupportedKind},
		{"absolute-outside", external, documents.OutsideRoot},
		{"symlink-outside", "outside-link", documents.OutsideRoot},
		{"missing-through-outside-symlink", "outside-directory/not-created.pdf", documents.OutsideRoot},
		{"dangling-final-outside-symlink", "dangling-outside", documents.OutsideRoot},
	} {
		t.Run(test.name, func(t *testing.T) {
			reference := resolver.Resolve(source, test.path)
			if reference.Status != test.status {
				t.Fatalf("status=%s expected=%s", reference.Status, test.status)
			}
			if test.status == documents.Available {
				path, err := resolver.Revalidate(reference)
				if err != nil {
					t.Fatal(err)
				}
				real, _ := filepath.EvalSymlinks(file)
				if path != real {
					t.Fatal("declaration base or canonical target differs")
				}
			}
		})
	}
}

func TestResolverRevalidationRejectsReplacementAndRetargeting(t *testing.T) {
	for _, scenario := range []string{"replace", "retarget", "modify", "remove"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			file := filepath.Join(root, "receipt.pdf")
			other := filepath.Join(root, "other.pdf")
			link := filepath.Join(root, "link.pdf")
			if err := os.WriteFile(file, []byte("public original"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(other, []byte("public replacement"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(file, link); err != nil {
				t.Fatal(err)
			}
			resolver, err := documents.NewResolver([]string{root})
			if err != nil {
				t.Fatal(err)
			}
			reference := resolver.Resolve(filepath.Join(root, "main.bean"), "link.pdf")
			if reference.Status != documents.Available {
				t.Fatal(reference.Status)
			}
			switch scenario {
			case "replace":
				err = os.Rename(other, file)
			case "retarget":
				err = os.Remove(link)
				if err == nil {
					err = os.Symlink(other, link)
				}
			case "modify":
				err = os.WriteFile(file, []byte("public changed content longer"), 0600)
			case "remove":
				err = os.Remove(file)
			}
			if err != nil {
				t.Fatal(err)
			}
			if path, err := resolver.Revalidate(reference); path != "" || (!errors.Is(err, documents.ErrChanged) && !errors.Is(err, documents.ErrUnavailable)) {
				t.Fatal("changed reference accepted", err)
			}
		})
	}
}

func TestResolverRequiresExplicitRootsAndRejectsBorrowedIdentity(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	file := filepath.Join(outside, "receipt.pdf")
	if err := os.WriteFile(file, []byte("public extra root"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, roots := range [][]string{nil, {"relative"}, {file}, {filepath.Join(root, "missing")}} {
		if _, err := documents.NewResolver(roots); !errors.Is(err, documents.ErrInvalidRoots) {
			t.Fatal("invalid roots accepted")
		}
	}
	first, err := documents.NewResolver([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	second, err := documents.NewResolver([]string{root, outside})
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "main.bean")
	if first.Resolve(source, file).Status != documents.OutsideRoot {
		t.Fatal("outside root implicitly allowed")
	}
	ref := second.Resolve(source, file)
	if ref.Status != documents.Available {
		t.Fatal("explicit extra root not allowed")
	}
	if _, err := first.Revalidate(ref); !errors.Is(err, documents.ErrUnavailable) {
		t.Fatal("reference from another root policy accepted")
	}
	ref.Path = filepath.Join(root, "forged.pdf")
	path, err := second.Revalidate(ref)
	real, _ := filepath.EvalSymlinks(file)
	if err != nil || path != real {
		t.Fatal("presentation path changed authorized target")
	}
	sibling := root + "-sibling"
	if first.Resolve(source, filepath.Join(sibling, "file.pdf")).Status != documents.OutsideRoot {
		t.Fatal("root prefix mistaken for containment")
	}
	loop := filepath.Join(root, "loop")
	if err := os.Symlink(loop, loop); err != nil {
		t.Fatal(err)
	}
	if first.Resolve(source, "loop").Status != documents.Unreadable {
		t.Fatal("symlink loop not distinguished from missing")
	}
}

func TestResolverUnreadableFileAndPermissionRevocation(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses file read permissions")
	}
	root := t.TempDir()
	file := filepath.Join(root, "receipt.pdf")
	if err := os.WriteFile(file, []byte("public permissions"), 0600); err != nil {
		t.Fatal(err)
	}
	resolver, err := documents.NewResolver([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "main.bean")
	reference := resolver.Resolve(source, "receipt.pdf")
	if reference.Status != documents.Available {
		t.Fatal(reference.Status)
	}
	if err := os.Chmod(file, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(file, 0600)
	if current := resolver.Resolve(source, "receipt.pdf"); current.Status != documents.Unreadable {
		t.Fatal("permission denial not distinguished", current.Status)
	}
	if _, err := resolver.Revalidate(reference); !errors.Is(err, documents.ErrUnavailable) {
		t.Fatal("permission revocation did not block open")
	}
}
