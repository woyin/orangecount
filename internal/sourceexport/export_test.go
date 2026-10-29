// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package sourceexport

import (
	"fmt"
	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const cash = `2000-01-01 open Assets:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-02 * "opening"
  Assets:Cash 5 USD
  Equity:Opening -5 USD
`

func entry(t *testing.T, text string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "main.bean")
	if e := os.WriteFile(p, []byte(text), 0600); e != nil {
		t.Fatal(e)
	}
	return p
}
func TestEffectiveAverageRejectedBeforeTargetWrite(t *testing.T) {
	for _, text := range []string{`option "booking_method" "AVERAGE"` + "\n" + cash, strings.Replace(cash, "Assets:Cash USD", "Assets:Cash USD \"AVERAGE\"", 1)} {
		p := entry(t, text)
		_, err := Prepare(p, Options{Build: snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}}})
		if err == nil || !strings.Contains(err.Error(), "AVERAGE") {
			t.Fatalf("wanted effective AVERAGE rejection, got %v", err)
		}
	}
}

func TestPublishProtectsSourcesAndAttachments(t *testing.T) {
	p := entry(t, cash+"2000-01-03 document Assets:Cash \"receipt.bean\"\n")
	receipt := filepath.Join(filepath.Dir(p), "receipt.bean")
	if err := os.WriteFile(receipt, []byte("receipt"), 0600); err != nil {
		t.Fatal(err)
	}
	plan, err := Prepare(p, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{p, receipt, filepath.Join(filepath.Dir(p), "new.bean")} {
		if err = plan.Write(target); err == nil {
			t.Fatalf("unsafe target accepted: %s", target)
		}
	}
	if b, _ := os.ReadFile(receipt); string(b) != "receipt" {
		t.Fatal("attachment changed")
	}
	target := filepath.Join(t.TempDir(), "export.bean")
	if err = plan.Write(target); err != nil {
		t.Fatal(err)
	}
	if got := snapshot.BuildWithOptions(target, snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}}); got.Snapshot == nil {
		t.Fatalf("published ledger invalid: %+v", got.Diagnostics)
	}
}

func TestTreeFailureLeavesNoPartialOutput(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "main.bean")
	child := filepath.Join(root, "child.bean")
	os.WriteFile(p, []byte("include \"child.bean\"\n2000-01-01 open Assets:Cash USD\n"), 0600)
	os.WriteFile(child, []byte("2000-01-01 open Equity:Opening USD\n2000-01-02 * \"opening\"\n  Assets:Cash 5 USD\n  Equity:Opening -5 USD\n"), 0600)
	plan, err := Prepare(p, Options{})
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "tree")
	count := 0
	err = plan.write(target, func(stage, path string) error {
		if stage == "write" {
			count++
			if count == 2 {
				return fmt.Errorf("injected second write")
			}
		}
		return nil
	})
	if err == nil {
		t.Fatal("fault ignored")
	}
	if _, err = os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("partial target survived")
	}
	children, _ := os.ReadDir(filepath.Dir(target))
	if len(children) != 0 {
		t.Fatal("staging tree leaked")
	}
	if err = plan.Write(target); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(target, "child.bean")); err != nil || !strings.Contains(string(b), "5 USD") {
		t.Fatal("missing published child")
	}
	if err = plan.Write(target); err == nil {
		t.Fatal("existing tree accepted")
	}
}

func TestFailuresPreserveExistingFile(t *testing.T) {
	for _, failure := range []string{"write", "written", "sync", "rename"} {
		t.Run(failure, func(t *testing.T) {
			plan, err := Prepare(entry(t, cash), Options{})
			if err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(t.TempDir(), "old.bean")
			os.WriteFile(target, []byte("old bytes"), 0600)
			err = plan.write(target, func(stage, path string) error {
				if stage == failure {
					if stage == "written" {
						os.Truncate(path, 3)
					}
					return fmt.Errorf("injected %s failure", stage)
				}
				return nil
			})
			if err == nil {
				t.Fatal("fault ignored")
			}
			b, _ := os.ReadFile(target)
			if string(b) != "old bytes" {
				t.Fatal("old target corrupted")
			}
			files, _ := os.ReadDir(filepath.Dir(target))
			if len(files) != 1 {
				t.Fatal("staging file leaked")
			}
		})
	}
}
func TestStaleSourceAndRetargetedParentRejected(t *testing.T) {
	p := entry(t, cash)
	plan, err := Prepare(p, Options{})
	if err != nil {
		t.Fatal(err)
	}
	parent := filepath.Join(t.TempDir(), "output")
	os.Mkdir(parent, 0700)
	target := filepath.Join(parent, "out.bean")
	err = plan.write(target, func(stage, path string) error {
		if stage == "publish" {
			os.Rename(parent, parent+"-old")
			os.Mkdir(parent, 0700)
		}
		return nil
	})
	if err == nil {
		t.Fatal("replaced parent accepted")
	}
	files, _ := os.ReadDir(parent + "-old")
	if len(files) != 0 {
		t.Fatal("owned staging tree leaked after parent move")
	}
	os.WriteFile(p, []byte(cash+"; changed\n"), 0600)
	if err = plan.Write(target); err == nil {
		t.Fatal("changed source accepted")
	}
}

func TestTargetsRejectLinksAndMissingExternalAttachments(t *testing.T) {
	external := t.TempDir()
	missing := filepath.Join(external, "receipt.bean")
	p := entry(t, cash+"2000-01-03 document Assets:Cash \""+missing+"\"\n")
	plan, err := Prepare(p, Options{Reverse: true})
	if err != nil {
		t.Fatal(err)
	}
	if err = plan.Write(missing); err == nil {
		t.Fatal("missing declared attachment overwritten")
	}
	hard := filepath.Join(t.TempDir(), "hard.bean")
	if err = os.Link(p, hard); err != nil {
		t.Fatal(err)
	}
	if err = plan.Write(hard); err == nil {
		t.Fatal("source hardlink replaced")
	}
	symlink := filepath.Join(t.TempDir(), "symlink.bean")
	if err = os.Symlink(p, symlink); err != nil {
		t.Fatal(err)
	}
	if err = plan.Write(symlink); err == nil {
		t.Fatal("symlink accepted")
	}
	if b, _ := os.ReadFile(p); !strings.Contains(string(b), "opening") {
		t.Fatal("source damaged")
	}
}
func TestIncludeEscapeAndDuplicateRejected(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child.bean")
	os.WriteFile(child, []byte(cash), 0600)
	sub := filepath.Join(root, "sub")
	os.Mkdir(sub, 0700)
	for _, text := range []string{"include \"../child.bean\"\n", "include \"../child.bean\"\ninclude \"../child.bean\"\n", "include \"" + child + "\"\n"} {
		p := filepath.Join(sub, "main.bean")
		os.WriteFile(p, []byte(text), 0600)
		if _, err := Prepare(p, Options{}); err == nil || !(strings.Contains(err.Error(), "E-EXPORT-INCLUDE") || strings.Contains(err.Error(), "E-INCLUDE-DUPLICATE")) {
			t.Fatalf("unsafe include accepted: %v", err)
		}
	}
}
func TestGlobIncludeRejectedBeforePublication(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "parts"), 0700)
	p := filepath.Join(root, "main.bean")
	os.WriteFile(p, []byte("include \"parts/*.bean\"\n"), 0600)
	os.WriteFile(filepath.Join(root, "parts", "one.bean"), []byte(cash), 0600)
	if _, err := Prepare(p, Options{}); err == nil {
		t.Fatal("unsupported glob accepted")
	}
}
func TestUnprovenOptionsPluginsAndMetadataRejected(t *testing.T) {
	for _, text := range []string{"plugin \"example.plugin\"\n" + cash, "option \"tolerance\" \"0.01\"\n" + cash, strings.Replace(cash, "  Assets:Cash 5 USD", "  Assets:Cash 5 USD\n    filename: \"override\"", 1)} {
		_, err := Prepare(entry(t, text), Options{Build: snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}}})
		if err == nil {
			t.Fatal("unproven extension accepted")
		}
	}
}

// TestRenderCommasIsAcceptedAsDisplayOnly pins the option policy: render_commas
// has no reader in this codebase and only affects the upstream renderer, so the
// export accepts it. The companion assertion keeps the refusal path alive for an
// option that is still unproven.
func TestRenderCommasIsAcceptedAsDisplayOnly(t *testing.T) {
	withOption := "option \"render_commas\" \"TRUE\"\n" + cash
	p := entry(t, withOption)
	plan, err := Prepare(p, Options{Build: snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}}})
	if err != nil {
		t.Fatalf("render_commas blocked export: %v", err)
	}
	target := filepath.Join(t.TempDir(), "out.bean")
	if err := plan.Write(target); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	written, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	// The option line is preserved and no dialect line leaks into the export.
	if !strings.Contains(string(written), "render_commas") {
		t.Fatal("export dropped a preserved option")
	}
	if _, err := Prepare(entry(t, "option \"totally_unproven_option\" \"1\"\n"+cash), Options{Build: snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}}}); err == nil {
		t.Fatal("an unproven option was accepted")
	}
}

// TestFingerprintAcceptsEquivalentLayouts pins the two canonicalizations that a
// faithful dialect rewrite relies on: a transaction whose source line is
// aggregated may reappear split into per-leg lines, and same-date entries may
// change sequence. Both must still compare equal, while a genuine accounting
// change must not.
func TestFingerprintAcceptsEquivalentLayouts(t *testing.T) {
	base := func(t *testing.T, body string) ledger.Evaluation {
		t.Helper()
		p := entry(t, `2000-01-01 open Assets:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-01 open Expenses:Food USD
option "operating_currency" "USD"

2026-01-01 event "note" "same day as the transaction"

2026-01-02 * "a" "split"
`+body)
		b := snapshot.Build(p)
		if b.Snapshot == nil {
			t.Fatalf("fixture did not evaluate: %v", b.Diagnostics)
		}
		return b.Snapshot.Evaluation()
	}
	aggregated := base(t, "  Assets:Cash -50 USD\n  Equity:Opening -50 USD\n  Expenses:Food 100 USD\n")
	split := base(t, "  Assets:Cash -30 USD\n  Assets:Cash -20 USD\n  Equity:Opening -50 USD\n  Expenses:Food 100 USD\n")
	want, err := financialFingerprint(aggregated)
	if err != nil {
		t.Fatal(err)
	}
	got, err := financialFingerprint(split)
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Fatal("an equivalent aggregated/split layout changed the fingerprint")
	}
	lossy := base(t, "  Assets:Cash -50 USD\n  Equity:Opening -49 USD\n  Expenses:Food 99 USD\n")
	if third, err := financialFingerprint(lossy); err != nil {
		t.Fatal(err)
	} else if third == want {
		t.Fatal("a genuine accounting change was hidden by canonicalization")
	}
}

func TestReverseConversionDoesNotApplyExportCompatibilityGate(t *testing.T) {
	p := entry(t, "option \"booking_method\" \"AVERAGE\"\n"+strings.Replace(cash, "  Assets:Cash 5 USD\n  Equity:Opening -5 USD", "  Equity:Opening -5 USD\n  Assets:Cash 5 USD", 1))
	plan, err := Prepare(p, Options{Reverse: true})
	if err != nil {
		t.Fatal(err)
	}
	if err = plan.Write(filepath.Join(t.TempDir(), "reverse.bean")); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentNewDirectoryIsNeverReplaced(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "main.bean")
	os.WriteFile(p, []byte("include \"child.bean\"\n"), 0600)
	os.WriteFile(filepath.Join(root, "child.bean"), []byte(cash), 0600)
	plan, err := Prepare(p, Options{})
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "tree")
	var created os.FileInfo
	err = plan.write(target, func(stage, path string) error {
		if stage == "rename" {
			if err := os.Mkdir(target, 0700); err != nil {
				return err
			}
			created, _ = os.Stat(target)
		}
		return nil
	})
	if err == nil {
		t.Fatal("concurrent target overwritten")
	}
	current, err := os.Stat(target)
	if err != nil || created == nil || !os.SameFile(created, current) {
		t.Fatal("concurrent directory identity changed")
	}
	children, _ := os.ReadDir(target)
	if len(children) != 0 {
		t.Fatal("concurrent directory populated")
	}
	children, _ = os.ReadDir(filepath.Dir(target))
	if len(children) != 1 {
		t.Fatal("staging tree leaked")
	}
}
func TestSavedAttachmentInodeStaysProtectedAfterDeclarationPathMoves(t *testing.T) {
	root := t.TempDir()
	doc := filepath.Join(root, "receipt.txt")
	os.WriteFile(doc, []byte("original receipt"), 0600)
	p := entry(t, cash+"2000-01-03 document Assets:Cash \""+doc+"\"\n")
	plan, err := Prepare(p, Options{})
	if err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(root, "moved.txt")
	os.Rename(doc, moved)
	os.WriteFile(doc, []byte("replacement receipt"), 0600)
	if err = plan.Write(moved); err == nil {
		t.Fatal("captured attachment inode overwritten")
	}
	if b, _ := os.ReadFile(moved); string(b) != "original receipt" {
		t.Fatal("moved attachment changed")
	}
}

// Replacement must not unlink the previous inode. On macOS a Finder alias is an
// ordinary file whose alias record lives in an extended attribute, so a build
// without cgo cannot recognize one. Moving the old file aside removes the need
// to recognize anything: whatever occupied the name keeps existing under the
// backup path.
func TestReplacementKeepsPreviousFileAsBackup(t *testing.T) {
	plan, err := Prepare(entry(t, cash), Options{})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "book.bean")
	if err := os.WriteFile(target, []byte("old bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = plan.Write(target); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(target + ".orangecount.bak"); string(b) != "old bytes" {
		t.Fatalf("previous file not preserved: %q", b)
	}
	if b, _ := os.ReadFile(target); !strings.Contains(string(b), "Assets:Cash") {
		t.Fatalf("new file not published: %q", b)
	}
}

func TestFailedPublicationRestoresPreviousFile(t *testing.T) {
	plan, err := Prepare(entry(t, cash), Options{})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "book.bean")
	if err := os.WriteFile(target, []byte("old bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	err = plan.write(target, func(stage, path string) error {
		if stage == "replaced" {
			return fmt.Errorf("injected %s failure", stage)
		}
		return nil
	})
	if err == nil {
		t.Fatal("fault ignored")
	}
	if b, _ := os.ReadFile(target); string(b) != "old bytes" {
		t.Fatalf("previous file not restored: %q", b)
	}
	if _, err := os.Lstat(target + ".orangecount.bak"); !os.IsNotExist(err) {
		t.Fatal("backup left behind")
	}
	if files, _ := os.ReadDir(dir); len(files) != 1 {
		t.Fatal("staging output leaked")
	}
}

func TestReplacementRefusesTargetsItCannotPreserve(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real.bean")
	if err := os.WriteFile(real, []byte("keep me"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.bean")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	hard := filepath.Join(dir, "hard.bean")
	if err := os.Link(real, hard); err != nil {
		t.Fatal(err)
	}
	// A backup slot that is not an ordinary file is never overwritten either.
	occupied := filepath.Join(dir, "occupied.bean")
	if err := os.WriteFile(occupied, []byte("old bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, occupied+".orangecount.bak"); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{link, hard, occupied} {
		plan, err := Prepare(entry(t, cash), Options{})
		if err != nil {
			t.Fatal(err)
		}
		if err = plan.Write(target); err == nil {
			t.Fatalf("%s accepted", filepath.Base(target))
		}
	}
	if b, _ := os.ReadFile(real); string(b) != "keep me" {
		t.Fatal("linked destination content changed")
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("symlink target replaced")
	}
	if info, err := os.Lstat(occupied + ".orangecount.bak"); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("backup slot replaced")
	}
	if b, _ := os.ReadFile(occupied); string(b) != "old bytes" {
		t.Fatal("refused target modified")
	}
}

// A target that still matches the publication record is this tool's own output,
// so replacing it is not a loss. The backup slot then stays untouched, which is
// what keeps a preserved Finder alias safe across repeated exports.
func TestRepeatedPublicationReplacesItsOwnOutput(t *testing.T) {
	plan, err := Prepare(entry(t, cash), Options{})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "book.bean")
	if err := os.WriteFile(target+".orangecount.bak", []byte("preserved original"), 0600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err = plan.Write(target); err != nil {
			t.Fatalf("publication %d: %v", i+1, err)
		}
	}
	if b, _ := os.ReadFile(target + ".orangecount.bak"); string(b) != "preserved original" {
		t.Fatalf("repeat publication changed the backup slot: %q", b)
	}
	if b, _ := os.ReadFile(target); !strings.Contains(string(b), "Assets:Cash") {
		t.Fatal("target not published")
	}
}

// A target edited after the last publication no longer matches the record, so it
// is preserved instead of overwritten.
func TestEditedTargetIsPreservedNotOverwritten(t *testing.T) {
	plan, err := Prepare(entry(t, cash), Options{})
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "book.bean")
	if err = plan.Write(target); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("hand edited"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = plan.Write(target); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(target + ".orangecount.bak"); string(b) != "hand edited" {
		t.Fatalf("edited target not preserved: %q", b)
	}
	// The next publication is again this tool's own output, so the preserved edit
	// must survive it.
	if err = plan.Write(target); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(target + ".orangecount.bak"); string(b) != "hand edited" {
		t.Fatalf("later publication discarded the preserved edit: %q", b)
	}
}

func TestOccupiedBackupSlotStopsTheExport(t *testing.T) {
	plan, err := Prepare(entry(t, cash), Options{})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "book.bean")
	if err := os.WriteFile(target, []byte("user content"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target+".orangecount.bak", []byte("earlier version"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = plan.Write(target); err == nil {
		t.Fatal("occupied backup slot ignored")
	}
	for path, want := range map[string]string{
		target:                      "user content",
		target + ".orangecount.bak": "earlier version",
	} {
		if b, _ := os.ReadFile(path); string(b) != want {
			t.Fatalf("%s changed: %q", filepath.Base(path), b)
		}
	}
}
