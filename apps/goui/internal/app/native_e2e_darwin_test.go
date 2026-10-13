//go:build darwin && orangecount_e2e

package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode"
	"unicode/utf16"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"orangecount/apps/goui/internal/platform"
	"orangecount/apps/goui/internal/preferences"
	"orangecount/apps/goui/internal/querypresets"
	"orangecount/apps/goui/internal/workspace"
	"orangecount/internal/desktop"
	"orangecount/internal/documents"
	"orangecount/internal/ledger"
)

func TestMain(m *testing.M) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		os.Exit(m.Run())
	}
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_QUIT_HELPER") == "1" {
		if code := nativeQuitHelper(); code != 0 {
			os.Exit(code)
		}
		return
	}
	mygo.App.SetName("OrangeCount Go UI E2E")
	mygo.App.OnWindowAllClosed(func() {})
	done := make(chan int, 1)
	mygo.App.WhenReady(func() {
		// A separate development app may already be frontmost in the desktop
		// session. Native menu, key-window, and panel checks need this E2E app
		// to be active so they target the window under test.
		mygo.App.Focus()
		go func() {
			done <- m.Run()
			mygo.App.Quit()
		}()
	})
	if err := mygo.App.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "MyGo native E2E app failed:", err)
		os.Exit(1)
	}
	os.Exit(<-done)
}

func nativeQuitHelper() int {
	mygo.App.SetName("OrangeCount Go UI E2E")
	mygo.App.SetMenu(NewApplicationMenu(nil, nil, "zh-CN"))
	keyResult := make(chan bool, 1)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Hidden: true, Title: "Cmd+Q quit lifecycle"})
		go func() {
			time.Sleep(250 * time.Millisecond)
			handled := mygo.E2EPerformKeyEquivalent("q", false)
			keyResult <- handled
			if !handled {
				mygo.App.Quit()
			}
		}()
	})
	if err := mygo.App.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "MyGo Cmd+Q helper failed:", err)
		return 1
	}
	if !<-keyResult {
		fmt.Fprintln(os.Stderr, "native application menu did not handle Cmd+Q")
		return 1
	}
	fmt.Println("native Cmd+Q returned from App.Run")
	return 0
}

func TestNativeCmdQExitsApplicationProcess(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to verify native Cmd+Q process exit")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "ORANGECOUNT_GOUI_NATIVE_QUIT_HELPER=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("native Cmd+Q helper did not exit cleanly: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "native Cmd+Q returned from App.Run") {
		t.Fatalf("native Cmd+Q helper exited without returning from App.Run:\n%s", output)
	}
}

func TestNativeTwentyWindowOpenCloseCyclesReleaseWorkspaces(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open real macOS Content windows")
	}
	const cycles = 20
	path := nativeFixturePath(t, "native/cashflow.bean")
	rssSamples := make([]int, 0, cycles)
	heapSamples := make([]uint64, 0, cycles)
	var baselineWindowCount int
	mygo.RunOnMain(func() { baselineWindowCount = len(mygo.Windows()) })
	for cycle := 0; cycle < cycles; cycle++ {
		var rssKiB int
		var heapAlloc uint64
		func() {
			model := newModelWithEngine(true, nil)
			var window *mygo.Window
			mygo.RunOnMain(func() {
				options := model.WindowOptions()
				options.Title = fmt.Sprintf("OrangeCount Go UI lifecycle E2E %02d", cycle+1)
				window = mygo.NewWindow(options)
				model.SetWindow(window)
				window.OnClosed(model.Close)
			})
			closed := false
			cleanup := func() {
				if closed {
					return
				}
				window.Destroy()
				model.Close()
				if model.workspace != nil {
					select {
					case <-model.workspace.Done():
					case <-time.After(10 * time.Second):
						t.Errorf("native lifecycle cycle %d workspace did not close", cycle+1)
					}
				}
				closed = true
			}
			defer cleanup()

			model.OpenLedgerPath(path)
			eventuallyNative(t, fmt.Sprintf("cycle %d dashboard snapshot", cycle+1), func() bool {
				state := model.workspace.State()
				return state.Phase == workspace.PhaseReady && !state.Busy && state.Response != nil && state.Response.Request.View == "dashboard"
			})
			eventuallyNative(t, fmt.Sprintf("cycle %d native dashboard accessibility content", cycle+1), func() bool {
				return nativeContentHas(mygo.E2EAccessibility(window), "财务概览")
			})
			cleanup()
			state := model.workspace.State()
			if state.Phase != workspace.PhaseClosed || state.Response != nil || state.QueryResponse != nil || state.QueryCapture != nil || state.ReportCapture != nil {
				t.Fatalf("cycle %d retained workspace state after close: phase=%s response=%t query=%t queryCapture=%t reportCapture=%t", cycle+1, state.Phase, state.Response != nil, state.QueryResponse != nil, state.QueryCapture != nil, state.ReportCapture != nil)
			}
			var openWindowCount int
			mygo.RunOnMain(func() { openWindowCount = len(mygo.Windows()) })
			if openWindowCount != baselineWindowCount {
				t.Fatalf("cycle %d retained native windows: got=%d want=%d", cycle+1, openWindowCount, baselineWindowCount)
			}
			model = nil
			window = nil
			cleanup = nil
			runtime.GC()
			time.Sleep(25 * time.Millisecond)
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			heapAlloc = mem.HeapAlloc
			rssKiB = nativeProcessRSSKiB(t)
		}()
		heapSamples = append(heapSamples, heapAlloc)
		rssSamples = append(rssSamples, rssKiB)
	}
	minRSS, maxRSS := rssSamples[0], rssSamples[0]
	for _, sample := range rssSamples[1:] {
		if sample < minRSS {
			minRSS = sample
		}
		if sample > maxRSS {
			maxRSS = sample
		}
	}
	t.Logf("native window lifecycle soak: cycles=%d rss_kib_first=%d rss_kib_last=%d rss_kib_min=%d rss_kib_max=%d rss_kib_samples=%v heap_alloc_bytes_first=%d heap_alloc_bytes_last=%d heap_alloc_bytes_samples=%v", cycles, rssSamples[0], rssSamples[len(rssSamples)-1], minRSS, maxRSS, rssSamples, heapSamples[0], heapSamples[len(heapSamples)-1], heapSamples)
	lateGrowthKiB := rssSamples[cycles-1] - rssSamples[cycles/2-1]
	if lateGrowthKiB > 4*1024 {
		t.Errorf("RSS kept growing after the first half of native window cycles: cycles 11–20 added %d KiB (threshold 4096 KiB)", lateGrowthKiB)
	}
	lateHeapGrowth := int64(heapSamples[cycles-1]) - int64(heapSamples[cycles/2-1])
	if lateHeapGrowth > 2*1024*1024 {
		t.Errorf("live Go heap kept growing after the first half of native window cycles: cycles 11–20 added %d bytes (threshold 2097152 bytes)", lateHeapGrowth)
	}
}

func TestNativeLedgerWindowsKeepReportFiltersIsolatedAcrossCloseCycles(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to exercise ledger-window isolation")
	}
	root := t.TempDir()
	firstPath := filepath.Join(root, "account-detail.bean")
	secondPath := filepath.Join(root, "cashflow.bean")
	for fixture, destination := range map[string]string{
		"native/account-detail.bean": firstPath,
		"native/cashflow.bean":       secondPath,
	} {
		contents, err := os.ReadFile(nativeFixturePath(t, fixture))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	firstModel, firstWindow := openNativeLedgerWindow(t, firstPath)
	secondModel, secondWindow := openNativeLedgerWindow(t, secondPath)
	openBalanceSheet := func(model *Model, window *mygo.Window, name string) {
		t.Helper()
		performNativeAccessibilityEventually(t, window, "AXButton", "资产负债表", "press", "")
		eventuallyNative(t, name+" balance-sheet tree is ready", func() bool {
			state := model.workspace.State()
			var context string
			mygo.RunOnMain(func() { context = model.reportTreeContext })
			return !state.Busy && state.Response != nil && state.Response.Request.View == "balance_sheet" &&
				context == "balance_sheet\x00"+state.ActivePath
		})
	}
	openBalanceSheet(firstModel, firstWindow, "first")
	openBalanceSheet(secondModel, secondWindow, "second")
	applyAccountFilter := func(model *Model, window *mygo.Window, query, row, name string) {
		t.Helper()
		setNativeFieldValueEventually(t, window, "搜索账户", query)
		clickNativeAccessibilityEventually(t, window, "AXButton", "筛选账户")
		eventuallyNative(t, name+" account filter applies to its own tree", func() bool {
			var search string
			mygo.RunOnMain(func() { search = model.statementSearch })
			return search == strings.ToLower(query) && nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXRow", row)
		})
	}
	applyAccountFilter(firstModel, firstWindow, "Assets:Test:Child", "Assets:Test:Child", "first")
	applyAccountFilter(secondModel, secondWindow, "Assets:Bank", "Assets:Bank", "second")
	firstSnapshot := firstModel.workspace.State().Ledger.SnapshotID
	secondSnapshot := secondModel.workspace.State().Ledger.SnapshotID
	if firstSnapshot == "" || secondSnapshot == "" || firstSnapshot == secondSnapshot {
		t.Fatalf("two ledger windows did not establish distinct snapshots: %q / %q", firstSnapshot, secondSnapshot)
	}
	if nativeRoleLabelExists(mygo.E2EAccessibility(firstWindow), "AXRow", "Assets:Bank") ||
		nativeRoleLabelExists(mygo.E2EAccessibility(secondWindow), "AXRow", "Assets:Test:Child") {
		t.Fatal("account rows crossed between the independently filtered ledgers")
	}
	firstWindow.Destroy()
	eventuallyNative(t, "closing the first window closes only its workspace", func() bool {
		return firstModel.workspace.State().Phase == workspace.PhaseClosed
	})
	baselineWindows := 0
	mygo.RunOnMain(func() { baselineWindows = len(mygo.Windows()) })
	if baselineWindows != 1 {
		t.Fatalf("one stable ledger window remains after closing the first: got %d windows, want 1", baselineWindows)
	}

	for cycle := 1; cycle <= 20; cycle++ {
		model := newModelWithEngine(true, nil)
		var window *mygo.Window
		mygo.RunOnMain(func() {
			options := model.WindowOptions()
			options.Title = fmt.Sprintf("OrangeCount Go UI ledger isolation cycle %02d", cycle)
			window = mygo.NewWindow(options)
			model.SetWindow(window)
			window.OnClosed(model.Close)
		})
		model.OpenLedgerPath(firstPath)
		eventuallyNative(t, fmt.Sprintf("cycle %d opens the first ledger in an isolated window", cycle), func() bool {
			state := model.workspace.State()
			return state.Phase == workspace.PhaseReady && !state.Busy && state.ActivePath == firstPath &&
				state.Response != nil && state.Response.Request.View == "dashboard" &&
				nativeContentHas(mygo.E2EAccessibility(window), "财务概览")
		})
		window.Destroy()
		model.Close()
		select {
		case <-model.workspace.Done():
		case <-time.After(10 * time.Second):
			t.Fatalf("cycle %d workspace did not close", cycle)
		}
		closed := model.workspace.State()
		if closed.Phase != workspace.PhaseClosed || closed.Response != nil || closed.QueryResponse != nil || closed.QueryCapture != nil {
			t.Fatalf("cycle %d retained closed ledger state: phase=%s response=%t query=%t capture=%t", cycle, closed.Phase, closed.Response != nil, closed.QueryResponse != nil, closed.QueryCapture != nil)
		}
		stable := secondModel.workspace.State()
		var stableSearch string
		mygo.RunOnMain(func() { stableSearch = secondModel.statementSearch })
		if stable.Phase != workspace.PhaseReady || stable.Busy || stable.ActivePath != secondPath ||
			stable.Ledger.SnapshotID != secondSnapshot || stable.Response == nil || stable.Response.Request.View != "balance_sheet" ||
			stableSearch != "assets:bank" || !nativeRoleLabelExists(mygo.E2EAccessibility(secondWindow), "AXRow", "Assets:Bank") {
			t.Fatalf("cycle %d changed the surviving ledger window: state=%+v search=%q AX=%s", cycle, stable, stableSearch, nativeNodeSummary(mygo.E2EAccessibility(secondWindow)))
		}
		var windowCount int
		mygo.RunOnMain(func() { windowCount = len(mygo.Windows()) })
		if windowCount != baselineWindows {
			t.Fatalf("cycle %d retained a native window: got %d, want %d", cycle, windowCount, baselineWindows)
		}
	}
}

func TestNativeWindowIdleAndCloseReleaseFrameTimers(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to inspect real macOS Content windows")
	}
	path := nativeFixturePath(t, "native/cashflow.bean")
	var baselineHosts ui.E2EHostLifecycleStats
	mygo.RunOnMain(func() { baselineHosts = ui.E2EHostLifecycle() })
	baselineSurfaces, baselineLinks, baselineRunningLinks, baselineTimers := mygo.E2ESurfaceLifecycle()

	model := newModelWithEngine(true, nil)
	var window *mygo.Window
	mygo.RunOnMain(func() {
		options := model.WindowOptions()
		options.Title = "OrangeCount Go UI idle lifecycle E2E"
		window = mygo.NewWindow(options)
		model.SetWindow(window)
		window.OnClosed(model.Close)
	})
	closed := false
	cleanup := func() {
		if closed {
			return
		}
		window.Destroy()
		model.Close()
		if model.workspace != nil {
			select {
			case <-model.workspace.Done():
			case <-time.After(10 * time.Second):
				t.Error("idle lifecycle workspace did not close")
			}
		}
		closed = true
	}
	defer cleanup()

	model.OpenLedgerPath(path)
	eventuallyNative(t, "idle lifecycle dashboard snapshot", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseReady && !state.Busy && state.Response != nil && state.Response.Request.View == "dashboard"
	})
	eventuallyNative(t, "idle lifecycle native dashboard accessibility content", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), "财务概览")
	})
	eventuallyNative(t, "native frame timers to settle", func() bool {
		var hosts ui.E2EHostLifecycleStats
		mygo.RunOnMain(func() { hosts = ui.E2EHostLifecycle() })
		surfaces, _, runningLinks, timers := mygo.E2ESurfaceLifecycle()
		return hosts.Hosts == baselineHosts.Hosts+1 && hosts.ActiveIdleTimers == 0 &&
			surfaces == baselineSurfaces+1 && runningLinks == 0 && timers == 0
	})

	var liveHosts ui.E2EHostLifecycleStats
	mygo.RunOnMain(func() { liveHosts = ui.E2EHostLifecycle() })
	liveSurfaces, liveLinks, liveRunningLinks, liveTimers := mygo.E2ESurfaceLifecycle()
	if liveHosts.Hosts != baselineHosts.Hosts+1 || liveHosts.DetachedHosts != 0 || liveHosts.ActiveIdleTimers != 0 {
		t.Fatalf("settled window host lifecycle = %+v, baseline=%+v", liveHosts, baselineHosts)
	}
	if liveSurfaces != baselineSurfaces+1 || liveRunningLinks != 0 || liveTimers != 0 {
		t.Fatalf("settled native surfaces=%d displayLinks=%d runningLinks=%d frameTimers=%d; baseline surfaces=%d links=%d running=%d timers=%d", liveSurfaces, liveLinks, liveRunningLinks, liveTimers, baselineSurfaces, baselineLinks, baselineRunningLinks, baselineTimers)
	}

	cleanup()
	eventuallyNative(t, "closed host and native frame resources to return to baseline", func() bool {
		var hosts ui.E2EHostLifecycleStats
		mygo.RunOnMain(func() { hosts = ui.E2EHostLifecycle() })
		surfaces, links, runningLinks, timers := mygo.E2ESurfaceLifecycle()
		return hosts == baselineHosts && surfaces == baselineSurfaces && links == baselineLinks &&
			runningLinks == baselineRunningLinks && timers == baselineTimers
	})
}

func TestNativeAttachmentRootRevocationPropagatesAcrossWindows(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open real macOS Content windows")
	}

	root := t.TempDir()
	ledgerDirectory := filepath.Join(root, "ledger")
	attachmentDirectory := filepath.Join(root, "shared-attachments")
	if err := os.MkdirAll(ledgerDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(attachmentDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	attachmentPath := filepath.Join(attachmentDirectory, "receipt.pdf")
	if err := os.WriteFile(attachmentPath, []byte("synthetic receipt"), 0o600); err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(ledgerDirectory, "ledger.bean")
	contents := "option \"operating_currency\" \"USD\"\n2024-01-01 open Assets:Cash USD\n2024-01-02 document Assets:Cash " + strconv.Quote(attachmentPath) + "\n"
	if err := os.WriteFile(ledgerPath, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	store := preferences.NewStore(filepath.Join(t.TempDir(), "preferences-v1.json"))
	bookmark, err := platform.CreateDirectoryBookmark(attachmentDirectory)
	if err != nil {
		t.Fatalf("create attachment directory bookmark: %v", err)
	}
	canonicalAttachmentDirectory, err := filepath.EvalSymlinks(attachmentDirectory)
	if err != nil {
		t.Fatalf("resolve attachment directory identity: %v", err)
	}
	newWindow := func(title string) (*Model, *mygo.Window) {
		t.Helper()
		model := newModelWithEngine(true, nil)
		model.SetPreferences(store)
		var window *mygo.Window
		mygo.RunOnMain(func() {
			options := model.WindowOptions()
			options.Title = title
			window = mygo.NewWindow(options)
			model.SetWindow(window)
			window.OnClosed(model.Close)
		})
		t.Cleanup(func() {
			window.Destroy()
			model.Close()
			if model.workspace != nil {
				select {
				case <-model.workspace.Done():
				case <-time.After(10 * time.Second):
					t.Errorf("workspace did not close for %s", title)
				}
			}
		})
		model.OpenLedgerPath(ledgerPath)
		eventuallyNative(t, title+" dashboard", func() bool {
			state := model.workspace.State()
			return state.Phase == workspace.PhaseReady && !state.Busy && state.AttachmentRootsLoaded && state.Response != nil && state.Response.Request.View == "dashboard"
		})
		clickNativeAccessibilityEventually(t, window, "AXButton", "附件")
		eventuallyNative(t, title+" attachment list", func() bool {
			state := model.workspace.State()
			return !state.Busy && state.Response != nil && state.Response.Directives != nil && state.Response.Directives.Mode == "all" && state.Response.Request.DirectiveKind == "document"
		})
		return model, window
	}

	first, firstWindow := newWindow("OrangeCount Attachment Roots E2E A")
	second, secondWindow := newWindow("OrangeCount Attachment Roots E2E B")

	firstState := first.workspace.State()
	if attachment, ok := nativeAttachmentFromState(firstState); !ok || attachment.Status != documents.OutsideRoot {
		t.Fatalf("attachment should start outside the ledger's default root: found=%t status=%q", ok, attachment.Status)
	}
	mygo.RunOnMain(func() {
		first.configureAttachmentRoots([]platform.DirectoryBookmark{bookmark}, firstState.AttachmentRoots.PolicyRevision, true)
	})
	rootsApplied := false
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		firstRoots := first.workspace.State().AttachmentRoots
		secondRoots := second.workspace.State().AttachmentRoots
		if len(firstRoots.AdditionalRoots) == 1 && firstRoots.AdditionalRoots[0] == canonicalAttachmentDirectory &&
			len(secondRoots.AdditionalRoots) == 1 && secondRoots.AdditionalRoots[0] == canonicalAttachmentDirectory {
			rootsApplied = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !rootsApplied {
		firstState, secondState := first.workspace.State(), second.workspace.State()
		stored, storeErr := store.ForLedger(ledgerPath)
		var firstModelState, secondModelState string
		mygo.RunOnMain(func() {
			firstModelState = fmt.Sprintf("sync_busy=%t generation=%d notice=%q", first.rootPolicySyncBusy, first.attachmentRootGeneration, first.attachmentRootNotice)
			secondModelState = fmt.Sprintf("sync_busy=%t generation=%d notice=%q pending=%t", second.rootPolicySyncBusy, second.attachmentRootGeneration, second.attachmentRootNotice, second.pendingSharedRootChange != nil)
		})
		storedPaths := make([]string, len(stored.AttachmentRoots))
		for i, storedRoot := range stored.AttachmentRoots {
			storedPaths[i] = storedRoot.Path
		}
		t.Fatalf("bookmarked root did not reach both sessions: first=%+v (%s); second=%+v (%s); stored_paths=%v generation=%d store_error=%v", firstState.AttachmentRoots, firstModelState, secondState.AttachmentRoots, secondModelState, storedPaths, stored.AttachmentRootsGeneration, storeErr)
	}
	eventuallyNative(t, "both attachment lists authorize the document", func() bool {
		firstAttachment, firstOK := nativeAttachmentFromState(first.workspace.State())
		secondAttachment, secondOK := nativeAttachmentFromState(second.workspace.State())
		return firstOK && secondOK && firstAttachment.Status == documents.Available && secondAttachment.Status == documents.Available
	})
	eventuallyNative(t, "both windows show the available attachment", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(firstWindow), "receipt.pdf") && nativeContentHas(mygo.E2EAccessibility(firstWindow), "可用") &&
			nativeContentHas(mygo.E2EAccessibility(secondWindow), "receipt.pdf") && nativeContentHas(mygo.E2EAccessibility(secondWindow), "可用")
	})
	staleAttachment, ok := nativeAttachmentFromState(first.workspace.State())
	if !ok || staleAttachment.Status != documents.Available {
		t.Fatalf("could not capture an authorized attachment before revocation: found=%t status=%q", ok, staleAttachment.Status)
	}

	secondWindow.Update(func() { second.page = pageAttachmentRoots })
	eventuallyNative(t, "second window attachment roots page", func() bool {
		nodes := mygo.E2EAccessibility(secondWindow)
		return nativeContentHas(nodes, "shared-attachments") && nativeRoleLabelExists(nodes, "AXButton", "移除")
	})
	clickNativeAccessibilityEventually(t, secondWindow, "AXButton", "移除")
	eventuallyNative(t, "shared preference removal and both session policy revocations", func() bool {
		firstState := first.workspace.State()
		secondState := second.workspace.State()
		preferencesForLedger, err := store.ForLedger(ledgerPath)
		return err == nil && len(preferencesForLedger.AttachmentRoots) == 0 &&
			len(firstState.AttachmentRoots.AdditionalRoots) == 0 && len(secondState.AttachmentRoots.AdditionalRoots) == 0
	})
	secondWindow.Update(func() {
		second.page = pageAttachments
		second.attachmentOffset = 0
		second.requestView("attachments")
	})
	eventuallyNative(t, "both windows reclassify the document outside policy", func() bool {
		firstAttachment, firstOK := nativeAttachmentFromState(first.workspace.State())
		secondAttachment, secondOK := nativeAttachmentFromState(second.workspace.State())
		return firstOK && secondOK && firstAttachment.Status == documents.OutsideRoot && secondAttachment.Status == documents.OutsideRoot
	})
	eventuallyNative(t, "both attachment pages show the revoked status", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(firstWindow), "不在允许的附件目录中") &&
			nativeContentHas(mygo.E2EAccessibility(secondWindow), "不在允许的附件目录中")
	})

	openedPath := ""
	mygo.RunOnMain(func() {
		first.openPath = func(path string) error { openedPath = path; return nil }
		first.requestAttachmentAction(staleAttachment, false)
	})
	if openedPath != "" {
		t.Fatalf("stale attachment action opened %q after shared revocation", openedPath)
	}
}

func nativeAttachmentFromState(state workspace.State) (desktop.AttachmentDTO, bool) {
	if state.Response == nil || state.Response.Directives == nil {
		return desktop.AttachmentDTO{}, false
	}
	for _, record := range state.Response.Directives.Records {
		if len(record.Attachments) > 0 {
			return record.Attachments[0], true
		}
	}
	return desktop.AttachmentDTO{}, false
}

func TestNativeAttachmentRootOpenPanelIsOwnedAndCancellable(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS directory picker")
	}

	root := t.TempDir()
	ledgerPath := filepath.Join(root, "ledger.bean")
	if err := os.WriteFile(ledgerPath, []byte("option \"operating_currency\" \"USD\"\n2024-01-01 open Assets:Cash USD\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := preferences.NewStore(filepath.Join(t.TempDir(), "preferences-v1.json"))
	model := newModelWithEngine(true, nil)
	model.SetPreferences(store)
	var window *mygo.Window
	mygo.RunOnMain(func() {
		options := model.WindowOptions()
		options.Title = "OrangeCount Attachment Picker E2E"
		window = mygo.NewWindow(options)
		model.SetWindow(window)
		window.OnClosed(model.Close)
	})
	t.Cleanup(func() {
		window.Destroy()
		model.Close()
		if model.workspace != nil {
			select {
			case <-model.workspace.Done():
			case <-time.After(10 * time.Second):
				t.Error("attachment picker workspace did not close")
			}
		}
	})

	model.OpenLedgerPath(ledgerPath)
	eventuallyNative(t, "attachment picker ledger and root policy", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseReady && !state.Busy && state.AttachmentRootsLoaded && state.Response != nil && state.Response.Request.View == "dashboard"
	})
	window.Update(func() { model.navigateToPage(pageAttachmentRoots, "", true) })
	eventuallyNative(t, "attachment root management page", func() bool {
		return nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXButton", "添加目录…")
	})
	clickNativeAccessibilityEventually(t, window, "AXButton", "添加目录…")
	eventuallyNative(t, "owned native directory picker", func() bool {
		return mygo.E2EOpenPanelVisible() && mygo.E2EHasSheet(window)
	})
	if !mygo.E2ECancelSheet(window) {
		t.Fatal("native attachment directory picker could not be cancelled")
	}
	eventuallyNative(t, "cancelled directory picker to close and clear picking state", func() bool {
		var picking bool
		mygo.RunOnMain(func() { picking = model.picking })
		return !mygo.E2EOpenPanelVisible() && !mygo.E2EHasSheet(window) && !picking
	})

	state := model.workspace.State()
	if len(state.AttachmentRoots.AdditionalRoots) != 0 {
		t.Fatalf("cancelling the native directory picker changed workspace roots: %+v", state.AttachmentRoots)
	}
	stored, err := store.ForLedger(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.AttachmentRoots) != 0 {
		t.Fatalf("cancelling the native directory picker persisted roots: %+v", stored.AttachmentRoots)
	}
}

func TestNativeBroadAttachmentRootConfirmationCanBeCancelled(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to show a real macOS confirmation sheet")
	}

	model, window, store, ledgerPath, broadRoot := newNativeBroadAttachmentRootConfirmation(t)
	found, clicked := mygo.E2EClickSheetButton(window, "取消")
	if !found || !clicked {
		t.Fatalf("broad attachment root cancel button: found=%t clicked=%t", found, clicked)
	}
	eventuallyNative(t, "cancelled broad-root confirmation to close without changing policy", func() bool {
		var picking bool
		mygo.RunOnMain(func() { picking = model.picking })
		state := model.workspace.State()
		stored, err := store.ForLedger(ledgerPath)
		return err == nil && !picking && !mygo.E2EHasSheet(window) &&
			len(state.AttachmentRoots.AdditionalRoots) == 0 && len(stored.AttachmentRoots) == 0
	})
	if state := model.workspace.State(); state.AttachmentRoots.DefaultRoot == broadRoot {
		t.Fatalf("test setup unexpectedly made broad root the ledger default: %+v", state.AttachmentRoots)
	}
}

func TestNativeBroadAttachmentRootConfirmationCanBeAccepted(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to show a real macOS confirmation sheet")
	}

	model, window, store, ledgerPath, broadRoot := newNativeBroadAttachmentRootConfirmation(t)
	found, clicked := mygo.E2EClickSheetButton(window, "允许")
	if !found || !clicked {
		t.Fatalf("broad attachment root allow button: found=%t clicked=%t", found, clicked)
	}
	deadline := time.Now().Add(15 * time.Second)
	var state workspace.State
	var stored preferences.LedgerPreferences
	var storeErr error
	var picking, policySyncBusy, sheet bool
	var notice string
	for time.Now().Before(deadline) {
		state = model.workspace.State()
		stored, storeErr = store.ForLedger(ledgerPath)
		mygo.RunOnMain(func() {
			picking = model.picking
			policySyncBusy = model.rootPolicySyncBusy
			notice = model.attachmentRootNotice
		})
		sheet = mygo.E2EHasSheet(window)
		if storeErr == nil && !picking && !state.AttachmentRootsBusy && !sheet &&
			len(state.AttachmentRoots.AdditionalRoots) == 1 && state.AttachmentRoots.AdditionalRoots[0] == broadRoot &&
			len(stored.AttachmentRoots) == 1 && stored.AttachmentRoots[0].Path == broadRoot {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if storeErr != nil || picking || state.AttachmentRootsBusy || sheet || len(state.AttachmentRoots.AdditionalRoots) != 1 ||
		state.AttachmentRoots.AdditionalRoots[0] != broadRoot || len(stored.AttachmentRoots) != 1 || stored.AttachmentRoots[0].Path != broadRoot {
		t.Fatalf("accepted temporary broad root was not applied: picking=%t policy_sync_busy=%t sheet=%t attachment_roots_busy=%t state_roots=%v stored_paths=%v store_error=%v notice=%q", picking, policySyncBusy, sheet, state.AttachmentRootsBusy, state.AttachmentRoots.AdditionalRoots, storedAttachmentRootPaths(stored.AttachmentRoots), storeErr, notice)
	}
	resolved, err := platform.ResolveDirectoryBookmark(stored.AttachmentRoots[0])
	if err != nil || resolved != broadRoot {
		t.Fatalf("accepted test bookmark resolved to %q, %v; want temporary test root %q", resolved, err, broadRoot)
	}
}

func storedAttachmentRootPaths(roots []platform.DirectoryBookmark) []string {
	paths := make([]string, len(roots))
	for i, root := range roots {
		paths[i] = root.Path
	}
	return paths
}

func newNativeBroadAttachmentRootConfirmation(t *testing.T) (*Model, *mygo.Window, *preferences.Store, string, string) {
	t.Helper()

	testRoot := t.TempDir()
	broadRoot := filepath.Join(testRoot, "temporary-home")
	ledgerDirectory := filepath.Join(testRoot, "ledger")
	if err := os.MkdirAll(broadRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(ledgerDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	canonicalBroadRoot, err := filepath.EvalSymlinks(broadRoot)
	if err != nil {
		t.Fatal(err)
	}
	// Treat only this disposable test directory as HOME. The production root
	// classifier then takes the broad-root warning path without granting the
	// app access to the real user's home directory.
	t.Setenv("HOME", broadRoot)
	ledgerPath := filepath.Join(ledgerDirectory, "ledger.bean")
	if err := os.WriteFile(ledgerPath, []byte("option \"operating_currency\" \"USD\"\n2024-01-01 open Assets:Cash USD\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := preferences.NewStore(filepath.Join(t.TempDir(), "preferences-v1.json"))
	model := newModelWithEngine(true, nil)
	model.SetPreferences(store)
	model.pickAttachmentDirectory = func(mygo.OpenDialogOptions) ([]string, error) {
		return []string{broadRoot}, nil
	}
	var window *mygo.Window
	mygo.RunOnMain(func() {
		options := model.WindowOptions()
		options.Title = "OrangeCount Broad Attachment Root E2E"
		window = mygo.NewWindow(options)
		model.SetWindow(window)
		window.OnClosed(model.Close)
	})
	t.Cleanup(func() {
		window.Destroy()
		model.Close()
		if model.workspace != nil {
			select {
			case <-model.workspace.Done():
			case <-time.After(10 * time.Second):
				t.Error("broad attachment root workspace did not close")
			}
		}
	})

	model.OpenLedgerPath(ledgerPath)
	eventuallyNative(t, "broad-root test ledger and default policy", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseReady && !state.Busy && state.AttachmentRootsLoaded && state.Response != nil
	})
	window.Update(func() { model.navigateToPage(pageAttachmentRoots, "", true) })
	eventuallyNative(t, "broad-root management page", func() bool {
		return nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXButton", "添加目录…")
	})
	clickNativeAccessibilityEventually(t, window, "AXButton", "添加目录…")
	eventuallyNative(t, "owned broad-root confirmation sheet", func() bool {
		return mygo.E2EHasSheet(window) && !mygo.E2EOpenPanelVisible()
	})
	return model, window, store, ledgerPath, canonicalBroadRoot
}

func TestNativeAttachmentRootSelectionPersistsAndAuthorizesDocument(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}

	root := t.TempDir()
	ledgerDirectory := filepath.Join(root, "ledger")
	attachmentDirectory := filepath.Join(root, "shared-attachments")
	if err := os.MkdirAll(ledgerDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(attachmentDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	attachmentPath := filepath.Join(attachmentDirectory, "receipt.pdf")
	if err := os.WriteFile(attachmentPath, []byte("synthetic receipt"), 0o600); err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(ledgerDirectory, "ledger.bean")
	contents := "option \"operating_currency\" \"USD\"\n2024-01-01 open Assets:Cash USD\n2024-01-02 document Assets:Cash " + strconv.Quote(attachmentPath) + "\n"
	if err := os.WriteFile(ledgerPath, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	canonicalAttachmentDirectory, err := filepath.EvalSymlinks(attachmentDirectory)
	if err != nil {
		t.Fatal(err)
	}
	store := preferences.NewStore(filepath.Join(t.TempDir(), "preferences-v1.json"))
	model := newModelWithEngine(true, nil)
	model.SetPreferences(store)
	selectedOptions := make(chan mygo.OpenDialogOptions, 1)
	model.pickAttachmentDirectory = func(options mygo.OpenDialogOptions) ([]string, error) {
		selectedOptions <- options
		return []string{attachmentDirectory}, nil
	}
	var window *mygo.Window
	mygo.RunOnMain(func() {
		options := model.WindowOptions()
		options.Title = "OrangeCount Attachment Selection E2E"
		window = mygo.NewWindow(options)
		model.SetWindow(window)
		window.OnClosed(model.Close)
	})
	t.Cleanup(func() {
		window.Destroy()
		model.Close()
		if model.workspace != nil {
			select {
			case <-model.workspace.Done():
			case <-time.After(10 * time.Second):
				t.Error("attachment selection workspace did not close")
			}
		}
	})

	model.OpenLedgerPath(ledgerPath)
	eventuallyNative(t, "attachment selection ledger and default root", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseReady && !state.Busy && state.AttachmentRootsLoaded && state.Response != nil && state.Response.Request.View == "dashboard"
	})
	window.Update(func() {
		model.page = pageAttachments
		model.attachmentOffset = 0
		model.requestView("attachments")
	})
	eventuallyNative(t, "attachment outside the default root", func() bool {
		state := model.workspace.State()
		attachment, ok := nativeAttachmentFromState(state)
		return !state.Busy && state.Response != nil && state.Response.Request.View == "journal" && state.Response.Request.DirectiveKind == "document" &&
			state.Response.Directives != nil && state.Response.Directives.Mode == "all" && ok && attachment.Status == documents.OutsideRoot
	})
	window.Update(func() { model.navigateToPage(pageAttachmentRoots, "", true) })
	eventuallyNative(t, "attachment root management page", func() bool {
		return nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXButton", "添加目录…")
	})
	clickNativeAccessibilityEventually(t, window, "AXButton", "添加目录…")
	var selected mygo.OpenDialogOptions
	select {
	case selected = <-selectedOptions:
	case <-time.After(3 * time.Second):
		t.Fatal("attachment directory picker adapter was not called")
	}
	if selected.Parent != window || !selected.Directory || selected.Title != "添加附件目录" {
		t.Fatalf("attachment directory picker options = %+v; parent=%t directory=%t title=%q", selected, selected.Parent == window, selected.Directory, selected.Title)
	}
	eventuallyNative(t, "selected directory policy and bookmark persistence", func() bool {
		state := model.workspace.State()
		stored, err := store.ForLedger(ledgerPath)
		return err == nil && len(state.AttachmentRoots.AdditionalRoots) == 1 && state.AttachmentRoots.AdditionalRoots[0] == canonicalAttachmentDirectory &&
			len(stored.AttachmentRoots) == 1 && stored.AttachmentRoots[0].Path == canonicalAttachmentDirectory
	})
	stored, err := store.ForLedger(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := platform.ResolveDirectoryBookmark(stored.AttachmentRoots[0])
	if err != nil || resolved != canonicalAttachmentDirectory {
		t.Fatalf("persisted attachment bookmark resolved to %q, %v; want %q", resolved, err, canonicalAttachmentDirectory)
	}

	window.Update(func() {
		model.page = pageAttachments
		model.attachmentOffset = 0
		model.requestView("attachments")
	})
	eventuallyNative(t, "selected root authorizes the document", func() bool {
		state := model.workspace.State()
		attachment, ok := nativeAttachmentFromState(state)
		return !state.Busy && state.Response != nil && state.Response.Request.View == "journal" && state.Response.Request.DirectiveKind == "document" &&
			state.Response.Directives != nil && state.Response.Directives.Mode == "all" && ok && attachment.Status == documents.Available
	})
	eventuallyNative(t, "native attachment page shows authorized receipt", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), "receipt.pdf") && nativeContentHas(mygo.E2EAccessibility(window), "可用")
	})
}

func TestNativeAttachmentRootRestoresBeforeDocumentRead(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to restore an attachment bookmark in a real macOS Content window")
	}

	root := t.TempDir()
	ledgerDirectory := filepath.Join(root, "ledger")
	attachmentDirectory := filepath.Join(root, "shared-attachments")
	if err := os.MkdirAll(ledgerDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(attachmentDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	attachmentPath := filepath.Join(attachmentDirectory, "receipt.pdf")
	if err := os.WriteFile(attachmentPath, []byte("synthetic receipt"), 0o600); err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(ledgerDirectory, "ledger.bean")
	contents := "option \"operating_currency\" \"USD\"\n2024-01-01 open Assets:Cash USD\n2024-01-02 document Assets:Cash " + strconv.Quote(attachmentPath) + "\n"
	if err := os.WriteFile(ledgerPath, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	canonicalAttachmentDirectory, err := filepath.EvalSymlinks(attachmentDirectory)
	if err != nil {
		t.Fatal(err)
	}
	bookmark, err := platform.CreateDirectoryBookmark(canonicalAttachmentDirectory)
	if err != nil {
		t.Fatalf("create persisted attachment bookmark: %v", err)
	}
	preferencesPath := filepath.Join(t.TempDir(), "preferences-v1.json")
	setupStore := preferences.NewStore(preferencesPath)
	initialGeneration, err := setupStore.UpdateAttachmentRoots(ledgerPath, []platform.DirectoryBookmark{bookmark}, "setup")
	if err != nil {
		t.Fatalf("persist attachment bookmark before app restart: %v", err)
	}
	storeAfterRestart := preferences.NewStore(preferencesPath)
	model := newModelWithEngine(true, nil)
	model.SetPreferences(storeAfterRestart)
	var window *mygo.Window
	mygo.RunOnMain(func() {
		options := model.WindowOptions()
		options.Title = "OrangeCount Attachment Restore E2E"
		window = mygo.NewWindow(options)
		model.SetWindow(window)
		window.OnClosed(model.Close)
	})
	t.Cleanup(func() {
		window.Destroy()
		model.Close()
		if model.workspace != nil {
			select {
			case <-model.workspace.Done():
			case <-time.After(10 * time.Second):
				t.Error("attachment restore workspace did not close")
			}
		}
	})

	model.OpenLedgerPath(ledgerPath)
	eventuallyNative(t, "restored attachment root policy", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseReady && state.AttachmentRootsLoaded && !state.AttachmentRootsBusy &&
			len(state.AttachmentRoots.AdditionalRoots) == 1 && state.AttachmentRoots.AdditionalRoots[0] == canonicalAttachmentDirectory
	})
	window.Update(func() { model.requestView("attachments") })
	eventuallyNative(t, "document readable under restored attachment policy", func() bool {
		state := model.workspace.State()
		attachment, ok := nativeAttachmentFromState(state)
		return !state.Busy && state.Response != nil && state.Response.Request.View == "journal" && state.Response.Request.DirectiveKind == "document" &&
			state.Response.Directives != nil && state.Response.Directives.Mode == "all" && ok && attachment.Status == documents.Available
	})
	eventuallyNative(t, "restored attachment is available in the native page", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), "receipt.pdf") && nativeContentHas(mygo.E2EAccessibility(window), "可用")
	})

	stored, err := storeAfterRestart.ForLedger(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	if stored.AttachmentRootsGeneration != initialGeneration || len(stored.AttachmentRoots) != 1 || stored.AttachmentRoots[0].Path != canonicalAttachmentDirectory {
		t.Fatalf("restoring attachment roots changed persisted preference: generation=%d roots=%+v; want generation=%d", stored.AttachmentRootsGeneration, stored.AttachmentRoots, initialGeneration)
	}
}

func TestNativeInvalidAttachmentBookmarkCanBeReselected(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to reselect an invalid attachment bookmark in a real macOS Content window")
	}

	root := t.TempDir()
	ledgerDirectory := filepath.Join(root, "ledger")
	attachmentDirectory := filepath.Join(root, "shared-attachments")
	if err := os.MkdirAll(ledgerDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(attachmentDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(ledgerDirectory, "ledger.bean")
	attachmentPath := filepath.Join(attachmentDirectory, "receipt.pdf")
	contents := "option \"operating_currency\" \"USD\"\n2024-01-01 open Assets:Cash USD\n2024-01-02 document Assets:Cash " + strconv.Quote(attachmentPath) + "\n"
	if err := os.WriteFile(ledgerPath, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	canonicalAttachmentDirectory, err := filepath.EvalSymlinks(attachmentDirectory)
	if err != nil {
		t.Fatal(err)
	}
	staleBookmark, err := platform.CreateDirectoryBookmark(canonicalAttachmentDirectory)
	if err != nil {
		t.Fatalf("create bookmark that will become stale: %v", err)
	}
	preferencesPath := filepath.Join(t.TempDir(), "preferences-v1.json")
	setupStore := preferences.NewStore(preferencesPath)
	initialGeneration, err := setupStore.UpdateAttachmentRoots(ledgerPath, []platform.DirectoryBookmark{staleBookmark}, "setup")
	if err != nil {
		t.Fatalf("persist attachment bookmark before replacing its directory: %v", err)
	}
	movedDirectory := filepath.Join(root, "old-attachments")
	if err := os.Rename(attachmentDirectory, movedDirectory); err != nil {
		t.Fatalf("move bookmarked attachment directory: %v", err)
	}
	if err := os.MkdirAll(attachmentDirectory, 0o700); err != nil {
		t.Fatalf("replace bookmarked attachment directory: %v", err)
	}
	if err := os.WriteFile(attachmentPath, []byte("replacement receipt"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := preferences.NewStore(preferencesPath)
	model := newModelWithEngine(true, nil)
	model.SetPreferences(store)
	selectedOptions := make(chan mygo.OpenDialogOptions, 1)
	model.pickAttachmentDirectory = func(options mygo.OpenDialogOptions) ([]string, error) {
		selectedOptions <- options
		return []string{attachmentDirectory}, nil
	}
	var window *mygo.Window
	mygo.RunOnMain(func() {
		options := model.WindowOptions()
		options.Title = "OrangeCount Invalid Attachment Bookmark E2E"
		window = mygo.NewWindow(options)
		model.SetWindow(window)
		window.OnClosed(model.Close)
	})
	t.Cleanup(func() {
		window.Destroy()
		model.Close()
		if model.workspace != nil {
			select {
			case <-model.workspace.Done():
			case <-time.After(10 * time.Second):
				t.Error("invalid attachment bookmark workspace did not close")
			}
		}
	})

	model.OpenLedgerPath(ledgerPath)
	eventuallyNative(t, "invalid attachment bookmark excluded from policy", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseReady && state.AttachmentRootsLoaded && !state.AttachmentRootsBusy &&
			len(state.AttachmentRoots.AdditionalRoots) == 0 && state.Response != nil
	})
	window.Update(func() { model.requestView("attachments") })
	eventuallyNative(t, "attachment is outside policy while bookmark is invalid", func() bool {
		state := model.workspace.State()
		attachment, ok := nativeAttachmentFromState(state)
		return !state.Busy && state.Response != nil && ok && attachment.Status == documents.OutsideRoot
	})
	window.Update(func() { model.navigateToPage(pageAttachmentRoots, "", true) })
	eventuallyNative(t, "invalid directory is marked for reselection", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), "路径或文件身份已变化") &&
			nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXButton", "添加目录…")
	})
	clickNativeAccessibilityEventually(t, window, "AXButton", "添加目录…")
	var selected mygo.OpenDialogOptions
	select {
	case selected = <-selectedOptions:
	case <-time.After(3 * time.Second):
		t.Fatal("attachment directory reselection adapter was not called")
	}
	if selected.Parent != window || !selected.Directory || selected.Title != "添加附件目录" {
		t.Fatalf("attachment directory reselection options = %+v; parent=%t directory=%t title=%q", selected, selected.Parent == window, selected.Directory, selected.Title)
	}
	eventuallyNative(t, "replacement bookmark persisted", func() bool {
		stored, err := store.ForLedger(ledgerPath)
		return err == nil && stored.AttachmentRootsGeneration == initialGeneration+1 && len(stored.AttachmentRoots) == 1 &&
			stored.AttachmentRoots[0].Path == canonicalAttachmentDirectory && stored.AttachmentRoots[0].Inode != staleBookmark.Inode
	})
	window.Update(func() { model.requestView("attachments") })
	eventuallyNative(t, "reselected directory authorizes the replacement document", func() bool {
		state := model.workspace.State()
		attachment, ok := nativeAttachmentFromState(state)
		return !state.Busy && state.Response != nil && state.Response.Request.DirectiveKind == "document" && ok && attachment.Status == documents.Available
	})
	eventuallyNative(t, "native attachment page shows the reselected receipt", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), "receipt.pdf") && nativeContentHas(mygo.E2EAccessibility(window), "可用")
	})
}

func TestNativeTenMinuteReloadRSSPlateau(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_SOAK") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_SOAK=1 to run the ten-minute native reload soak")
	}
	const reloadInterval = 10 * time.Second
	const reloadCount = 60
	model, window := openNativeCashflowWindow(t)
	opened := model.workspace.State()
	snapshotID := opened.Ledger.SnapshotID
	if snapshotID == "" || opened.Response == nil || opened.Response.Request.View != "dashboard" {
		t.Fatalf("native reload soak did not begin from a ready dashboard: %+v", opened)
	}
	started := time.Now()
	var samples []struct {
		minute    int
		rssKiB    int
		heapAlloc uint64
	}
	for reload := 1; reload <= reloadCount; reload++ {
		nextStart := started.Add(time.Duration(reload) * reloadInterval)
		if delay := time.Until(nextStart); delay > 0 {
			time.Sleep(delay)
		}
		previousOperation := model.workspace.State().Operation
		window.Update(func() { model.reloadActiveLedger() })
		eventuallyNative(t, fmt.Sprintf("native reload %d completion and dashboard read", reload), func() bool {
			state := model.workspace.State()
			return state.Operation > previousOperation && state.Phase == workspace.PhaseReady && !state.Busy && !state.Ledger.Reloading && state.Response != nil && state.Response.Request.View == "dashboard"
		})
		state := model.workspace.State()
		if state.Ledger.SnapshotID != snapshotID || state.Response.Request.SnapshotID != snapshotID || state.Response.Summary.TransactionsCount != opened.Response.Summary.TransactionsCount {
			t.Fatalf("native reload %d changed the unchanged ledger projection: opened=%q ledger=%q response=%q transactions=%d", reload, snapshotID, state.Ledger.SnapshotID, state.Response.Request.SnapshotID, state.Response.Summary.TransactionsCount)
		}
		if reload%6 == 0 {
			runtime.GC()
			time.Sleep(50 * time.Millisecond)
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			sample := struct {
				minute    int
				rssKiB    int
				heapAlloc uint64
			}{minute: reload / 6, rssKiB: nativeProcessRSSKiB(t), heapAlloc: mem.HeapAlloc}
			samples = append(samples, sample)
			t.Logf("native reload soak sample minute=%d rss_kib=%d heap_alloc_bytes=%d", sample.minute, sample.rssKiB, sample.heapAlloc)
		}
	}
	if len(samples) != 10 {
		t.Fatalf("native reload soak collected %d minute samples, want 10", len(samples))
	}
	lateRSSGrowthKiB := samples[len(samples)-1].rssKiB - samples[1].rssKiB
	if lateRSSGrowthKiB > 8*1024 {
		t.Errorf("RSS did not plateau after initial warm-up: minute 2–10 grew %d KiB (threshold 8192 KiB)", lateRSSGrowthKiB)
	}
	lateHeapGrowth := int64(samples[len(samples)-1].heapAlloc) - int64(samples[1].heapAlloc)
	if lateHeapGrowth > 4*1024*1024 {
		t.Errorf("live Go heap did not plateau after initial warm-up: minute 2–10 grew %d bytes (threshold 4194304 bytes)", lateHeapGrowth)
	}
	t.Logf("ten-minute native reload soak complete: reloads=%d elapsed=%s snapshot=%s rss_kib_minute2=%d rss_kib_minute10=%d late_rss_growth_kib=%d late_heap_growth_bytes=%d", reloadCount, time.Since(started), snapshotID, samples[1].rssKiB, samples[len(samples)-1].rssKiB, lateRSSGrowthKiB, lateHeapGrowth)
}

func nativeProcessRSSKiB(t *testing.T) int {
	t.Helper()
	output, err := exec.Command("ps", "-o", "rss=", "-p", strconv.Itoa(os.Getpid())).Output()
	if err != nil {
		t.Fatalf("read native E2E process RSS: %v", err)
	}
	rssKiB, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		t.Fatalf("parse native E2E process RSS %q: %v", output, err)
	}
	return rssKiB
}

func TestNativeCashflowOpenAndAccessibleNavigation(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	model, window := openNativeCashflowWindow(t)

	nodes := mygo.E2EAccessibility(window)
	if !nativeRoleLabelExists(nodes, "AXButton", "资产负债表") {
		t.Fatalf("native accessibility tree has no balance-sheet navigation button; observed labels: %s", nativeLabels(nodes))
	}
	pressNativeAccessibilityEventually(t, window, "资产负债表")
	eventuallyNative(t, "balance-sheet response", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "balance_sheet" && !state.Busy
	})
	eventuallyNative(t, "balance-sheet native content", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "资产负债表") && nativeContentHas(nodes, "Assets:Bank")
	})

	performNativeAccessibilityEventually(t, window, "AXButton", "流水", "press", "")
	eventuallyNative(t, "journal response", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "journal" && !state.Busy
	})
	eventuallyNative(t, "journal native content", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "流水") && nativeContentHas(nodes, "Salary") && nativeContentHas(nodes, "Food")
	})
}

func TestNativeCashflowBalanceCutoffAndPeriodTotalsMatchFixture(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to verify exact cashflow totals in a real macOS window")
	}
	model, window := openNativeCashflowWindow(t)
	initial := model.workspace.State()
	if initial.Response == nil || initial.Response.Trends == nil ||
		initial.Response.Trends.IncomeAmount.Exact != "1000" || initial.Response.Trends.ExpensesAmount.Exact != "300" ||
		initial.Response.Trends.NetAmount.Exact != "700" {
		t.Fatalf("initial cashflow totals do not match the fixed fixture: %+v", initial.Response)
	}
	initialSnapshot := initial.Ledger.SnapshotID

	pressNativeAccessibilityEventually(t, window, "资产负债表")
	eventuallyNative(t, "native balance sheet before the January 2 cutoff", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Tree != nil && state.Response.Request.View == "balance_sheet"
	})
	setNativeFieldValueEventually(t, window, "结束日期 YYYY-MM-DD", "2026-01-02")
	pressNativeAccessibilityEventually(t, window, "应用筛选")
	eventuallyNative(t, "balance sheet at the January 2 cutoff", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Tree != nil &&
			state.Response.Request.View == "balance_sheet" && state.Response.Request.DateTo == "2026-01-02" &&
			state.Response.Tree.DateTo == "2026-01-02" && len(state.Response.Tree.RootNodes) > 0 &&
			state.Response.Tree.RootNodes[0].Name == "Assets" && state.Response.Tree.RootNodes[0].SubtreeBalances["USD"] == "1000"
	})
	if got := model.workspace.State().Ledger.SnapshotID; got != initialSnapshot {
		t.Fatalf("balance-sheet cutoff changed the ledger snapshot: got=%q want=%q", got, initialSnapshot)
	}
	window.Invalidate()
	eventuallyNative(t, "visible USD 1,000 balance at the January 2 cutoff", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "1000 USD") && !nativeContentHas(nodes, "700 USD")
	})

	pressNativeAccessibilityEventually(t, window, "损益表")
	eventuallyNative(t, "native income statement before the January 3 period", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Tree != nil && state.Response.Request.View == "income_statement"
	})
	setNativeFieldValueEventually(t, window, "开始日期 YYYY-MM-DD", "2026-01-03")
	setNativeFieldValueEventually(t, window, "结束日期 YYYY-MM-DD", "2026-01-03")
	pressNativeAccessibilityEventually(t, window, "应用筛选")
	incomeStatementReady := false
	incomeStatementObserved := ""
	incomeStatementDeadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(incomeStatementDeadline) {
		state := model.workspace.State()
		incomeStatementObserved = fmt.Sprintf("phase=%s busy=%t", state.Phase, state.Busy)
		if state.Response != nil {
			incomeStatementObserved = fmt.Sprintf("phase=%s busy=%t request=%+v", state.Phase, state.Busy, state.Response.Request)
			if state.Response.Tree != nil {
				incomeStatementObserved += fmt.Sprintf(" tree=%+v", state.Response.Tree.RootNodes)
			}
		}
		if !state.Busy && state.Response != nil && state.Response.Tree != nil && state.Response.Request.View == "income_statement" &&
			state.Response.Request.DateFrom == "2026-01-03" && state.Response.Request.DateTo == "2026-01-03" && len(state.Response.Tree.RootNodes) == 2 &&
			state.Response.Tree.RootNodes[0].Name == "Income" && state.Response.Tree.RootNodes[0].SubtreeBalances["USD"] == "" &&
			state.Response.Tree.RootNodes[1].Name == "Expenses" && state.Response.Tree.RootNodes[1].SubtreeBalances["USD"] == "300" {
			incomeStatementReady = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !incomeStatementReady {
		t.Fatalf("income statement for January 3 only was not ready: %s", incomeStatementObserved)
	}
	if got := model.workspace.State().Ledger.SnapshotID; got != initialSnapshot {
		t.Fatalf("income-statement period changed the ledger snapshot: got=%q want=%q", got, initialSnapshot)
	}

	pressNativeAccessibilityEventually(t, window, "财务概览")
	eventuallyNative(t, "dashboard totals for the January 3 period with carried net worth", func() bool {
		state := model.workspace.State()
		if state.Busy || state.Response == nil || state.Response.Trends == nil || state.Response.Request.View != "dashboard" ||
			state.Response.Request.DateFrom != "2026-01-03" || state.Response.Request.DateTo != "2026-01-03" ||
			state.Response.Trends.IncomeAmount.Exact != "0" || state.Response.Trends.ExpensesAmount.Exact != "300" ||
			state.Response.Trends.NetAmount.Exact != "-300" || len(state.Response.Trends.NetWorthPoints) != 1 ||
			state.Response.Trends.NetWorthPoints[0].Amount.Exact != "700" {
			return false
		}
		return state.Response.State.SnapshotID == initialSnapshot
	})
	window.Invalidate()
	eventuallyNative(t, "visible exact dashboard values for the January 3 period", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "700 USD") && nativeContentHas(nodes, "300 USD") && nativeContentHas(nodes, "0 USD")
	})
}

func TestNativePageHistoryBackForwardAndBranchReplacement(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	model, window := openNativeCashflowWindow(t)
	if !nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXButton", "返回") {
		t.Fatal("native content does not expose the Back button")
	}
	performNativeAccessibilityEventually(t, window, "AXButton", "资产负债表", "press", "")
	eventuallyNative(t, "native balance-sheet navigation history", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "balance_sheet" && !state.Busy
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "流水", "press", "")
	eventuallyNative(t, "native journal after balance sheet", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "journal" && !state.Busy
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "返回", "press", "")
	eventuallyNative(t, "Back re-reads the balance sheet", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "balance_sheet" && !state.Busy
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "前进", "press", "")
	eventuallyNative(t, "Forward re-reads the journal", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "journal" && !state.Busy
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "返回", "press", "")
	eventuallyNative(t, "Back returns from journal before a new branch", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "balance_sheet" && !state.Busy
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "损益表", "press", "")
	eventuallyNative(t, "new navigation branch replaces forward history", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "income_statement" && !state.Busy && len(model.pageHistoryForward) == 0
	})
	if found, allowed, performed := mygo.E2EAccessibilityPress(window, "前进"); !found || allowed || performed {
		t.Fatalf("cleared Forward button remains actionable: found=%t allowed=%t performed=%t", found, allowed, performed)
	}
}

func TestNativeAverageSemanticsPreviewAndApplyUsesCurrentInput(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	fixturePath := nativeFixturePath(t, "native/average-history.bean")
	fixture, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(t.TempDir(), "average-history.bean")
	if err := os.WriteFile(ledgerPath, fixture, 0o600); err != nil {
		t.Fatal(err)
	}
	model, window := openNativeLedgerWindowWithProfile(t, ledgerPath, string(ledger.Legacy))
	performNativeAccessibilityEventually(t, window, "AXButton", "语义配置", "press", "")
	eventuallyNative(t, "native AVERAGE migration preview", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Migration != nil && state.Response.Request.View == "migration"
	})
	preview := model.workspace.State()
	comparison := preview.Response.Migration
	if preview.Ledger.SemanticsProfile != ledger.Legacy || comparison.InputIdentity == "" || comparison.V3.SnapshotID == "" || len(comparison.Details) != 0 {
		t.Fatalf("AVERAGE preview is not counts-only and input-bound: profile=%q comparison=%+v", preview.Ledger.SemanticsProfile, comparison)
	}
	hasDifference := false
	for _, count := range comparison.ChangeCounts {
		if count > 0 {
			hasDifference = true
			break
		}
	}
	if !hasDifference {
		t.Fatalf("AVERAGE fixture did not produce a visible migration difference: %+v", comparison.ChangeCounts)
	}
	eventuallyNative(t, "Legacy/V3 summary and explicit action in native accessibility", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "Legacy") && nativeContentHas(nodes, "V3") && nativeContentHas(nodes, "差异") &&
			nativeRoleLabelExists(nodes, "AXButton", "应用 V3")
	})
	window.Update(func() { model.queryDraft = "SELECT * FROM entries" })
	eventuallyNative(t, "seeded old query draft before profile replacement", func() bool {
		var draft string
		mygo.RunOnMain(func() { draft = model.queryDraft })
		return draft == "SELECT * FROM entries"
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "应用 V3", "press", "")
	eventuallyNative(t, "native profile replacement and fresh migration preview", func() bool {
		state := model.workspace.State()
		var draft string
		mygo.RunOnMain(func() { draft = model.queryDraft })
		return state.Epoch > preview.Epoch && state.Phase == workspace.PhaseReady && !state.Busy &&
			state.Ledger.SemanticsProfile == ledger.V3 && state.Ledger.SnapshotID == comparison.V3.SnapshotID &&
			state.Response != nil && state.Response.Migration != nil && state.Response.Migration.InputIdentity == comparison.InputIdentity && draft == ""
	})
	item, err := model.preferenceStore.ForLedger(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	if item.SemanticsProfile != string(ledger.V3) {
		t.Fatalf("native profile apply persisted %q, want %q", item.SemanticsProfile, ledger.V3)
	}
	after, err := os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(fixture) {
		t.Fatal("applying V3 changed the ledger file")
	}
}

func TestNativeInvalidLegacyLedgerCanApplyV3Recovery(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	fixturePath := nativeFixturePath(t, "native/holdings-unknowns.bean")
	fixture, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(t.TempDir(), "holdings-unknowns.bean")
	if err := os.WriteFile(ledgerPath, fixture, 0o600); err != nil {
		t.Fatal(err)
	}
	model := newModelWithEngine(true, nil)
	var window *mygo.Window
	mygo.RunOnMain(func() {
		options := model.WindowOptions()
		options.Title = "OrangeCount Go UI Native E2E"
		window = mygo.NewWindow(options)
		model.SetWindow(window)
		window.OnClosed(model.Close)
	})
	t.Cleanup(func() {
		window.Destroy()
		model.Close()
		if model.workspace != nil {
			select {
			case <-model.workspace.Done():
			case <-time.After(10 * time.Second):
				t.Error("native invalid-ledger semantics workspace did not close")
			}
		}
	})
	model.OpenLedgerPath(ledgerPath)
	eventuallyNative(t, "native Legacy interpretation reports its initial inventory error", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseInvalid && !state.Busy && state.ActivePath == ledgerPath &&
			state.Response != nil && state.Response.Summary.ErrorCount > 0
	})

	performNativeAccessibilityEventually(t, window, "AXButton", "语义配置", "press", "")
	eventuallyNative(t, "native invalid-ledger migration preview is current", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Request.View == "migration" &&
			state.Response.Migration != nil && state.Response.Migration.V3.Valid &&
			nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXButton", "应用 V3")
	})
	preview := model.workspace.State()
	inputIdentity := preview.Response.Migration.InputIdentity
	if inputIdentity == "" {
		t.Fatal("native invalid-ledger V3 preview has no input identity")
	}
	performNativeAccessibilityEventually(t, window, "AXButton", "应用 V3", "press", "")
	eventuallyNative(t, "native V3 recovery opens the same ledger with a valid snapshot", func() bool {
		state := model.workspace.State()
		return state.Epoch > preview.Epoch && state.Phase == workspace.PhaseReady && !state.Busy &&
			state.ActivePath == ledgerPath && state.Ledger.SemanticsProfile == ledger.V3 &&
			state.Ledger.Valid && state.Ledger.SnapshotID != ""
	})
	eventuallyNative(t, "native semantics preview shows the V3 recovery session", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "migration" &&
			state.Response.Migration != nil && state.Response.Migration.InputIdentity == inputIdentity &&
			nativeContentHas(mygo.E2EAccessibility(window), "当前配置")
	})
}

func TestNativeStaleSemanticsPreviewCannotApplyAfterInputChanges(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	fixturePath := nativeFixturePath(t, "native/average-history.bean")
	fixture, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(t.TempDir(), "average-history.bean")
	if err := os.WriteFile(ledgerPath, fixture, 0o600); err != nil {
		t.Fatal(err)
	}
	started := make(chan desktop.Request, 1)
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseRead := func() { releaseOnce.Do(func() { close(release) }) }
	defer releaseRead()
	engine := &nativeStaleMigrationGateEngine{started: started, release: release}
	model, window := openNativeLedgerWindowWithEngine(t, ledgerPath, string(ledger.Legacy), engine)
	performNativeAccessibilityEventually(t, window, "AXButton", "语义配置", "press", "")
	eventuallyNative(t, "initial native migration preview", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Migration != nil && !state.Busy && state.Response.Request.View == "migration"
	})
	initial := model.workspace.State()
	oldIdentity := initial.Response.Migration.InputIdentity
	oldSnapshotID := initial.Ledger.SnapshotID
	if oldIdentity == "" || oldSnapshotID == "" {
		t.Fatalf("initial migration preview lacks input identity or snapshot: %+v", initial.Response.Migration)
	}

	changedFixture := append(append([]byte(nil), fixture...), []byte("\n; changed after migration preview\n")...)
	if err := os.WriteFile(ledgerPath, changedFixture, 0o600); err != nil {
		t.Fatal(err)
	}
	var changedRequest desktop.Request
	select {
	case changedRequest = <-started:
	case <-time.After(15 * time.Second):
		t.Fatal("watcher did not request a migration preview for the changed ledger")
	}
	if changedRequest.View != "migration" || changedRequest.SnapshotID == oldSnapshotID {
		t.Fatalf("changed-input migration request did not use a new snapshot: %+v (old %s)", changedRequest, oldSnapshotID)
	}
	eventuallyNative(t, "old migration action removed while the new preview is pending", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "正在生成当前账本快照的语义配置预览") &&
			!nativeRoleLabelExists(nodes, "AXButton", "应用 V3")
	})

	window.Update(func() { model.applySemanticsProfile(ledger.V3, oldIdentity) })
	var staleApplyStatus string
	mygo.RunOnMain(func() { staleApplyStatus = model.status })
	stateDuringGate := model.workspace.State()
	if !strings.Contains(staleApplyStatus, "工作区忙碌") || !stateDuringGate.Busy || stateDuringGate.Ledger.SemanticsProfile != ledger.Legacy ||
		stateDuringGate.Ledger.SnapshotID != changedRequest.SnapshotID {
		t.Fatalf("profile action was not held while the new preview was pending: status=%q state=%+v", staleApplyStatus, stateDuringGate)
	}

	releaseRead()
	eventuallyNative(t, "fresh preview for the changed ledger input", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseReady && !state.Busy && state.Response != nil && state.Response.Migration != nil &&
			state.Response.Request.View == "migration" && state.Response.Request.SnapshotID == changedRequest.SnapshotID &&
			state.Response.Migration.InputIdentity != "" && state.Response.Migration.InputIdentity != oldIdentity
	})
	fresh := model.workspace.State()
	window.Update(func() { model.applySemanticsProfile(ledger.V3, oldIdentity) })
	mygo.RunOnMain(func() { staleApplyStatus = model.status })
	afterStaleApply := model.workspace.State()
	if !strings.Contains(staleApplyStatus, "账本快照已变化") || afterStaleApply.Ledger.SemanticsProfile != ledger.Legacy ||
		afterStaleApply.Ledger.SnapshotID != fresh.Ledger.SnapshotID || afterStaleApply.Epoch != fresh.Epoch {
		t.Fatalf("old input identity changed the active profile: status=%q before=%+v after=%+v", staleApplyStatus, fresh, afterStaleApply)
	}
	eventuallyNative(t, "fresh preview exposes a current V3 action", func() bool {
		return nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXButton", "应用 V3")
	})
}

func TestNativeDashboardCashflowChartKeyboardAndAccessibilityHelp(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	path := filepath.Join(t.TempDir(), "monthly-cashflow.bean")
	source := `option "operating_currency" "USD"
2026-01-01 open Assets:Bank USD
2026-01-01 open Income:Salary USD
2026-01-01 open Expenses:Food USD
2026-01-02 * "January salary"
  Assets:Bank 1000 USD
  Income:Salary -1000 USD
2026-01-03 * "January food"
  Expenses:Food 200 USD
  Assets:Bank -200 USD
2026-02-02 * "February salary"
  Assets:Bank 1500 USD
  Income:Salary -1500 USD
2026-02-03 * "February food"
  Expenses:Food 300 USD
  Assets:Bank -300 USD
2026-03-02 * "March salary"
  Assets:Bank 900 USD
  Income:Salary -900 USD
2026-03-03 * "March food"
  Expenses:Food 100 USD
  Assets:Bank -100 USD
`
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	model, window := openNativeLedgerWindow(t, path)
	response := model.workspace.State().Response
	if response == nil || response.Trends == nil || len(response.Trends.CurrencySeries) == 0 {
		t.Fatal("native cashflow fixture produced no dashboard trends")
	}
	flows := response.Trends.CurrencySeries[0].MonthlyCashFlows
	if len(flows) != 3 {
		t.Fatalf("native cashflow fixture has %d monthly periods, want 3", len(flows))
	}
	netWorthPoints := response.Trends.CurrencySeries[0].NetWorthPoints
	if len(netWorthPoints) != 3 {
		t.Fatalf("native net-worth fixture has %d monthly periods, want 3", len(netWorthPoints))
	}
	for _, flow := range flows {
		if flow.Income == nil || flow.Income.CompleteTotal == nil || flow.Expenses == nil || flow.Expenses.CompleteTotal == nil {
			t.Fatalf("native cashflow fixture has an incomplete exact value for %s", flow.Month)
		}
	}

	const chartLabel = "分组柱状图"
	eventuallyNative(t, "dashboard cashflow chart in native accessibility tree", func() bool {
		return nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXImage", chartLabel)
	})
	if !mygo.E2EPerformAccessibility(window, chartLabel, "focus", "") {
		t.Fatalf("native accessibility could not focus dashboard chart: %s", nativeNodeSummary(mygo.E2EAccessibility(window)))
	}
	checkPeriod := func(index int) {
		t.Helper()
		flow := flows[index]
		income := flow.Income.CompleteTotal.Exact + " USD"
		expenses := flow.Expenses.CompleteTotal.Exact + " USD"
		deadline := time.Now().Add(3 * time.Second)
		var observedHelp string
		for time.Now().Before(deadline) {
			for _, node := range mygo.E2EAccessibility(window) {
				if node.Role == "AXImage" && node.Label == chartLabel {
					observedHelp = node.Help
					if strings.Contains(node.Help, flow.Month) && strings.Contains(node.Help, income) && strings.Contains(node.Help, expenses) {
						return
					}
				}
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("native chart help for %s does not match income %s and expenses %s: %q", flow.Month, income, expenses, observedHelp)
	}
	if !mygo.E2EKey(window, 115, "") { // macOS Home
		t.Fatal("could not send Home to the focused native chart")
	}
	checkPeriod(0)
	if !mygo.E2EKey(window, 124, "") { // macOS Right Arrow
		t.Fatal("could not send Right Arrow to the focused native chart")
	}
	checkPeriod(1)
	if !mygo.E2EKey(window, 119, "") { // macOS End
		t.Fatal("could not send End to the focused native chart")
	}
	checkPeriod(2)
	if !mygo.E2EClickAccessibility(window, chartLabel) {
		t.Fatalf("native UI test could not click the visible grouped-bars plot: %s", nativeNodeSummary(mygo.E2EAccessibility(window)))
	}
	checkPeriod(1)

	const lineChartLabel = "净资产折线图"
	eventuallyNative(t, "dashboard net-worth line chart in native accessibility tree", func() bool {
		return nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXImage", lineChartLabel)
	})
	if !mygo.E2EPerformAccessibility(window, lineChartLabel, "focus", "") {
		t.Fatalf("native accessibility could not focus dashboard line chart: %s", nativeNodeSummary(mygo.E2EAccessibility(window)))
	}
	checkNetWorth := func(index int) {
		t.Helper()
		point := netWorthPoints[index]
		if point.Valuation == nil || point.Valuation.CompleteTotal == nil {
			t.Fatalf("native net-worth point %s has no exact amount: %+v", point.Month, point)
		}
		exact := point.Valuation.CompleteTotal.Exact + " USD"
		deadline := time.Now().Add(3 * time.Second)
		var observedHelp string
		for time.Now().Before(deadline) {
			for _, node := range mygo.E2EAccessibility(window) {
				if node.Role == "AXImage" && node.Label == lineChartLabel {
					observedHelp = node.Help
					if strings.Contains(node.Help, point.Month) && strings.Contains(node.Help, exact) {
						return
					}
				}
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("native line chart help for %s does not match exact net worth %s: %q", point.Month, exact, observedHelp)
	}
	if !mygo.E2EKey(window, 115, "") { // macOS Home
		t.Fatal("could not send Home to the focused native line chart")
	}
	checkNetWorth(0)
	if !mygo.E2EKey(window, 124, "") { // macOS Right Arrow
		t.Fatal("could not send Right Arrow to the focused native line chart")
	}
	checkNetWorth(1)
	if !mygo.E2EClickAccessibility(window, lineChartLabel) {
		t.Fatalf("native UI test could not click the visible line plot: %s", nativeNodeSummary(mygo.E2EAccessibility(window)))
	}
	checkNetWorth(1)

	window.Update(func() { model.setLocale("en") })
	const englishChartLabel = "Grouped bar chart"
	eventuallyNative(t, "English chart label and exact-table period heading", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeRoleLabelExists(nodes, "AXImage", englishChartLabel) && nativeContentHas(nodes, "Period")
	})
	for _, node := range mygo.E2EAccessibility(window) {
		if node.Role != "AXImage" || node.Label != englishChartLabel {
			continue
		}
		if !strings.Contains(node.Help, "Exact amounts for each period") || !strings.Contains(node.Help, "Left/Right") {
			t.Fatalf("English chart accessibility help is incomplete: %q", node.Help)
		}
		for _, r := range node.Help {
			if unicode.Is(unicode.Han, r) {
				t.Fatalf("English chart accessibility help contains Chinese text: %q", node.Help)
			}
		}
	}
	if nativeContentHas(mygo.E2EAccessibility(window), "期间") {
		t.Fatal("English exact-value table still exposes the Chinese period heading")
	}

}

func TestNativeJournalSelectsTransactionAndShowsExactPostings(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	model, window := openNativeCashflowWindow(t)
	performNativeAccessibilityEventually(t, window, "AXButton", "流水", "press", "")
	eventuallyNative(t, "native journal response", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "journal" && !state.Busy
	})
	eventuallyNative(t, "native Salary transaction row", func() bool {
		for _, node := range mygo.E2EAccessibility(window) {
			if node.Role == "AXRow" && strings.Contains(node.Label, "Salary") {
				return true
			}
		}
		return false
	})
	var salaryRow string
	for _, node := range mygo.E2EAccessibility(window) {
		if node.Role == "AXRow" && strings.Contains(node.Label, "Salary") {
			salaryRow = node.Label
			break
		}
	}
	if salaryRow == "" {
		t.Fatalf("native Salary transaction row was not available: %s", nativeNodeSummary(mygo.E2EAccessibility(window)))
	}
	performNativeAccessibilityEventually(t, window, "AXRow", salaryRow, "press", "")
	eventuallyNative(t, "exact transaction postings in native content", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "Assets:Bank") && nativeContentHas(nodes, "Income:Salary") && nativeContentHas(nodes, "1000 USD") && nativeContentHas(nodes, "-1000 USD")
	})
}

func TestNativeJournalPostingCostsPricesAndSourceActionsKeepLocations(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS journal window")
	}

	checkPosting := func(t *testing.T, fixture, narration, account, wantUnits, wantCost, wantPrice string, wantTotal bool, wantLine int) {
		t.Helper()
		path := nativeFixturePath(t, fixture)
		model, window := openNativeLedgerWindow(t, path)
		performNativeAccessibilityEventually(t, window, "AXButton", "流水", "press", "")
		eventuallyNative(t, "native journal response for "+narration, func() bool {
			state := model.workspace.State()
			return state.Response != nil && state.Response.Request.View == "journal" && !state.Busy && state.Response.Journal != nil
		})

		var target desktop.JournalTransactionDTO
		foundTransaction := false
		state := model.workspace.State()
		for _, transaction := range state.Response.Journal.Transactions {
			if transaction.Narration == narration {
				target, foundTransaction = transaction, true
				break
			}
		}
		if !foundTransaction {
			t.Fatalf("journal response does not contain %q: %+v", narration, state.Response.Journal.Transactions)
		}
		var foundPosting bool
		for _, posting := range target.Postings {
			if posting.Account != account {
				continue
			}
			foundPosting = true
			if posting.UnitsNumber+" "+posting.UnitsCurrency != wantUnits {
				t.Fatalf("%s units = %q %s, want %q", account, posting.UnitsNumber, posting.UnitsCurrency, wantUnits)
			}
			gotCost := ""
			if posting.Cost != nil {
				gotCost = posting.Cost.Display
			}
			if gotCost != wantCost {
				t.Fatalf("%s cost = %q, want %q", account, gotCost, wantCost)
			}
			gotPrice := ""
			if posting.Price != nil {
				prefix := "@ "
				if posting.Price.Total {
					prefix = "@@ "
				}
				gotPrice = prefix + posting.Price.Number.Exact + " " + posting.Price.Currency
				if posting.Price.Total != wantTotal {
					t.Fatalf("%s price total flag = %t, want %t", account, posting.Price.Total, wantTotal)
				}
			}
			if gotPrice != wantPrice {
				t.Fatalf("%s price = %q, want %q", account, gotPrice, wantPrice)
			}
			if posting.Source.Path != path || posting.Source.Line != wantLine || posting.Source.Column != 3 {
				t.Fatalf("%s source = %+v, want %s:%d:3", account, posting.Source, path, wantLine)
			}
		}
		if !foundPosting {
			t.Fatalf("transaction %q does not contain posting %s", narration, account)
		}

		var rowLabel string
		eventuallyNative(t, "native transaction row for "+narration, func() bool {
			for _, node := range mygo.E2EAccessibility(window) {
				if node.Role == "AXRow" && strings.Contains(node.Label, narration) {
					rowLabel = node.Label
					return true
				}
			}
			return false
		})
		performNativeAccessibilityEventually(t, window, "AXRow", rowLabel, "press", "")

		location := fmt.Sprintf("%s:%d:3", path, wantLine)
		actionLabel := model.tr("在 Finder 中显示来源") + " · " + location
		var nodes []mygo.E2EAccessNode
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			nodes = mygo.E2EAccessibility(window)
			if nativeRoleLabelExists(nodes, "AXButton", actionLabel) &&
				nativeContentHas(nodes, wantUnits) && nativeContentHas(nodes, location) &&
				(wantCost == "" || nativeContentHas(nodes, wantCost)) &&
				(wantPrice == "" || nativeContentHas(nodes, wantPrice)) {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		if !nativeRoleLabelExists(nodes, "AXButton", actionLabel) ||
			!nativeContentHas(nodes, wantUnits) || !nativeContentHas(nodes, location) ||
			(wantCost != "" && !nativeContentHas(nodes, wantCost)) ||
			(wantPrice != "" && !nativeContentHas(nodes, wantPrice)) {
			t.Fatalf("native posting detail mismatch: action=%t units=%t location=%t cost=%t price=%t; observed: %s",
				nativeRoleLabelExists(nodes, "AXButton", actionLabel), nativeContentHas(nodes, wantUnits), nativeContentHas(nodes, location),
				wantCost == "" || nativeContentHas(nodes, wantCost), wantPrice == "" || nativeContentHas(nodes, wantPrice), nativeNodeSummary(nodes))
		}

		revealed := make(chan string, 1)
		mygo.RunOnMain(func() {
			model.revealPath = func(value string) {
				select {
				case revealed <- value:
				default:
				}
			}
		})
		pressNativeAccessibilityEventually(t, window, actionLabel)
		select {
		case got := <-revealed:
			if got != path {
				t.Fatalf("source reveal path = %q, want %q", got, path)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("activating the source action did not reveal %q", path)
		}
	}

	t.Run("cost annotation", func(t *testing.T) {
		checkPosting(t, "v3-parity/cost-basis.bean", "initial lot", "Assets:Shares", "10 SH", "{2 USD}", "", false, 5)
	})
	t.Run("unit price annotation", func(t *testing.T) {
		checkPosting(t, "v3-parity/at-price.bean", "postings with price annotations", "Assets:Cash", "100 USD", "", "@ 0.9 EUR", false, 6)
	})
	t.Run("total price annotation", func(t *testing.T) {
		checkPosting(t, "v3-parity/at-price.bean", "total price spelling", "Assets:Bank", "200 USD", "", "@@ 180 EUR", true, 9)
	})
}

func TestNativeJournalSelectionSurvivesUnrelatedEditButNotDuplicates(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS journal window")
	}

	path := filepath.Join(t.TempDir(), "journal-selection.bean")
	transaction := `2026-01-02 * "Cafe" "Unique dinner"
  Assets:Cash -12 USD
  Expenses:Dining 12 USD
`
	ledgerHeader := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Expenses:Dining USD
`
	initial := ledgerHeader + transaction
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	model, window := openNativeLedgerWindow(t, path)
	performNativeAccessibilityEventually(t, window, "AXButton", "流水", "press", "")
	eventuallyNative(t, "unique native journal transaction", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "journal" && !state.Busy && state.Response.Summary.TransactionsCount == 1
	})
	var uniqueRow string
	eventuallyNative(t, "unique dinner row to become accessible", func() bool {
		for _, node := range mygo.E2EAccessibility(window) {
			if node.Role == "AXRow" && strings.Contains(node.Label, "Unique dinner") {
				uniqueRow = node.Label
				return true
			}
		}
		return false
	})
	performNativeAccessibilityEventually(t, window, "AXRow", uniqueRow, "press", "")
	eventuallyNative(t, "selected dinner postings", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), "Expenses:Dining")
	})
	initialSnapshot := model.workspace.State().Ledger.SnapshotID

	if err := os.WriteFile(path, []byte("; unrelated comment before existing transaction\n"+initial), 0o600); err != nil {
		t.Fatal(err)
	}
	eventuallyNative(t, "updated journal snapshot after inserting an unrelated comment", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseReady && !state.Busy && state.Ledger.SnapshotID != initialSnapshot &&
			state.Response != nil && state.Response.Request.View == "journal" && state.Response.State.SnapshotID == state.Ledger.SnapshotID
	})
	eventuallyNative(t, "same unique journal selection after unrelated source insertion", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), "Expenses:Dining")
	})

	duplicate := ledgerHeader + transaction + transaction
	if err := os.WriteFile(path, []byte(duplicate), 0o600); err != nil {
		t.Fatal(err)
	}
	eventuallyNative(t, "ambiguous duplicate transactions to clear inherited selection", func() bool {
		state := model.workspace.State()
		nodes := mygo.E2EAccessibility(window)
		return state.Phase == workspace.PhaseReady && !state.Busy && state.Response != nil && state.Response.Request.View == "journal" &&
			state.Response.State.SnapshotID == state.Ledger.SnapshotID && state.Response.Journal != nil && state.Response.Journal.TotalCount == 2 &&
			nativeContentHas(nodes, "选择一笔交易查看分录详情") && !nativeContentHas(nodes, "Expenses:Dining")
	})
}

func TestNativeJournalLongChineseNarrationWrapsAndKeepsExactAmount(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to inspect real macOS transaction text layout")
	}
	path := filepath.Join(t.TempDir(), "long-narration.bean")
	payee := "长文本验收商户"
	narration := strings.Repeat("跨境交易说明包含完整中文上下文和精确金额，", 5)
	amount := "123456789.123456789"
	source := fmt.Sprintf("option \"operating_currency\" \"USD\"\n2026-01-01 open Assets:Cash USD\n2026-01-01 open Equity:Opening USD\n2026-01-02 * %q %q\n  Assets:Cash %s USD\n  Equity:Opening -%s USD\n", payee, narration, amount, amount)
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	model, window := openNativeLedgerWindow(t, path)
	window.SetSize(900, 600)
	if width, height := window.Size(); width != 900 || height != 600 {
		t.Fatalf("native journal window size = %dx%d, want 900x600", width, height)
	}
	performNativeAccessibilityEventually(t, window, "AXButton", "流水", "press", "")
	eventuallyNative(t, "long-narration journal response", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Journal != nil && len(state.Response.Journal.Transactions) == 1 && !state.Busy
	})
	var rowLabel string
	eventuallyNative(t, "long-narration transaction row", func() bool {
		for _, node := range mygo.E2EAccessibility(window) {
			if node.Role == "AXRow" && strings.Contains(node.Label, payee) {
				rowLabel = node.Label
				return true
			}
		}
		return false
	})
	performNativeAccessibilityEventually(t, window, "AXRow", rowLabel, "press", "")
	expectedHeading := "2026-01-02 · " + payee + " · " + narration
	var narrationFrame [4]float64
	eventuallyNative(t, "complete wrapped narration and exact decimal postings", func() bool {
		nodes := mygo.E2EAccessibility(window)
		if !nativeContentHas(nodes, narration) || !nativeContentHas(nodes, amount+" USD") {
			return false
		}
		for _, node := range nodes {
			if node.Role == "AXStaticText" && strings.Contains(node.Label, expectedHeading) {
				narrationFrame = node.Frame
				return true
			}
		}
		return false
	})
	if narrationFrame[3] <= 22 {
		t.Fatalf("long Chinese narration was not laid out on multiple lines: Accessibility frame=%v", narrationFrame)
	}
	contentWidth, contentHeight := window.ContentSize()
	if narrationFrame[2] <= 0 || narrationFrame[2] > float64(contentWidth) {
		t.Fatalf("long Chinese narration frame width %.0f exceeds native content width %d", narrationFrame[2], contentWidth)
	}
	t.Logf("900x600 native transaction narration AX frame: %.0fx%.0f points in %dx%d content; exact amount=%s USD", narrationFrame[2], narrationFrame[3], contentWidth, contentHeight, amount)
}

func TestNativeJournalPaginationStaysOnSnapshot(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	path := filepath.Join(t.TempDir(), "journal.bean")
	var source strings.Builder
	source.WriteString("option \"operating_currency\" \"USD\"\n2026-01-01 open Assets:Cash USD\n2026-01-01 open Income:Salary USD\n")
	for index := 0; index < 105; index++ {
		fmt.Fprintf(&source, "2026-01-02 * \"Txn %03d\"\n  Assets:Cash 1 USD\n  Income:Salary -1 USD\n", index)
	}
	if err := os.WriteFile(path, []byte(source.String()), 0600); err != nil {
		t.Fatal(err)
	}
	model, window := openNativeLedgerWindow(t, path)
	opened := model.workspace.State()
	if opened.Response.Summary.TransactionsCount != 105 {
		t.Fatalf("native pagination fixture has %d transactions, want 105", opened.Response.Summary.TransactionsCount)
	}
	performNativeAccessibilityEventually(t, window, "AXButton", "流水", "press", "")
	eventuallyNative(t, "first native journal page", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Journal != nil && state.Response.Journal.Offset == 0 && state.Response.Journal.TotalCount == 105 && len(state.Response.Journal.Transactions) == 100 && state.Response.Journal.HasMore && !state.Busy
	})
	eventuallyNative(t, "first native journal accessibility range", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), "显示 1–100")
	})
	first := model.workspace.State().Response
	if first.Request.SnapshotID != opened.Ledger.SnapshotID {
		t.Fatalf("native first journal page lost its snapshot or range: opened=%q response=%+v AX=%s", opened.Ledger.SnapshotID, first.Request, nativeNodeSummary(mygo.E2EAccessibility(window)))
	}
	performNativeAccessibilityEventually(t, window, "AXButton", "下一页", "press", "")
	eventuallyNative(t, "second native journal page", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Journal != nil && state.Response.Journal.Offset == 100 && state.Response.Journal.TotalCount == 105 && len(state.Response.Journal.Transactions) == 5 && !state.Response.Journal.HasMore && !state.Busy
	})
	eventuallyNative(t, "second native journal accessibility range", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), "显示 101–105")
	})
	second := model.workspace.State().Response
	if second.Request.SnapshotID != opened.Ledger.SnapshotID {
		t.Fatalf("native second journal page lost its snapshot or range: opened=%q response=%+v AX=%s", opened.Ledger.SnapshotID, second.Request, nativeNodeSummary(mygo.E2EAccessibility(window)))
	}
	performNativeAccessibilityEventually(t, window, "AXButton", "上一页", "press", "")
	eventuallyNative(t, "returned native first journal page", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Journal != nil && state.Response.Journal.Offset == 0 && len(state.Response.Journal.Transactions) == 100 && !state.Busy
	})
	if response := model.workspace.State().Response; response.Request.SnapshotID != opened.Ledger.SnapshotID {
		t.Fatalf("returning to the first native journal page changed snapshots: got=%q want=%q", response.Request.SnapshotID, opened.Ledger.SnapshotID)
	}
}

func TestNativeJournalDateAccountSearchAndSortKeepSnapshotIdentity(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to apply journal filters in a real macOS Content window")
	}
	path := filepath.Join(t.TempDir(), "journal-filters.bean")
	var source strings.Builder
	source.WriteString("option \"operating_currency\" \"USD\"\n2026-01-01 open Assets:Cash USD\n2026-01-01 open Assets:Bank USD\n2026-01-01 open Income:Salary USD\n")
	dates := [...]string{"2026-01-01", "2026-01-02", "2026-01-03"}
	for index := 0; index < 105; index++ {
		date := dates[index%len(dates)]
		account := "Assets:Cash"
		if index%4 >= 2 {
			account = "Assets:Bank"
		}
		payee := fmt.Sprintf("Other Txn %03d", index)
		if index%12 == 7 {
			payee = fmt.Sprintf("Needle Txn %03d", index)
		}
		fmt.Fprintf(&source, "%s * %q\n  %s 1 USD\n  Income:Salary -1 USD\n", date, payee, account)
	}
	if err := os.WriteFile(path, []byte(source.String()), 0o600); err != nil {
		t.Fatal(err)
	}

	model, window := openNativeLedgerWindow(t, path)
	performNativeAccessibilityEventually(t, window, "AXButton", "流水", "press", "")
	eventuallyNative(t, "unfiltered journal fixture", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Journal != nil &&
			state.Response.Journal.TotalCount == 105 && len(state.Response.Journal.Transactions) == 100
	})
	snapshotID := model.workspace.State().Response.State.SnapshotID

	setNativeFieldValueEventually(t, window, "开始日期 YYYY-MM-DD", "2026-01-02")
	setNativeFieldValueEventually(t, window, "结束日期 YYYY-MM-DD", "2026-01-02")
	clickNativeAccessibilityEventually(t, window, "AXButton", "应用筛选")
	eventuallyNative(t, "native date-filtered journal", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Journal != nil &&
			state.Response.Request.DateFrom == "2026-01-02" && state.Response.Request.DateTo == "2026-01-02" &&
			state.Response.Journal.TotalCount == 35 && state.Response.Journal.Offset == 0
	})
	if got := model.workspace.State().Response.State.SnapshotID; got != snapshotID {
		t.Fatalf("date filter changed journal snapshot: got=%q want=%q", got, snapshotID)
	}

	performNativeAccessibilityEventually(t, window, "AXPopUpButton", "全部账户", "press", "")
	clickNativeAccessibilityEventually(t, window, "AXStaticText", "Assets:Bank")
	eventuallyNative(t, "native account-filtered journal", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Journal != nil &&
			state.Response.Request.DateFrom == "2026-01-02" && state.Response.Request.DateTo == "2026-01-02" &&
			state.Response.Request.Account == "Assets:Bank" && state.Response.Journal.TotalCount == 17
	})
	if got := model.workspace.State().Response.State.SnapshotID; got != snapshotID {
		t.Fatalf("account filter changed journal snapshot: got=%q want=%q", got, snapshotID)
	}

	setNativeFieldValueEventually(t, window, "搜索流水", "Needle")
	clickNativeAccessibilityEventually(t, window, "AXButton", "筛选流水")
	eventuallyNative(t, "native text-filtered journal", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Journal != nil &&
			state.Response.Request.DateFrom == "2026-01-02" && state.Response.Request.DateTo == "2026-01-02" &&
			state.Response.Request.Account == "Assets:Bank" && state.Response.Request.Text == "Needle" &&
			state.Response.Journal.TotalCount == 9 && len(state.Response.Journal.Transactions) == 9
	})
	if got := model.workspace.State().Response.State.SnapshotID; got != snapshotID {
		t.Fatalf("text filter changed journal snapshot: got=%q want=%q", got, snapshotID)
	}

	performNativeAccessibilityEventually(t, window, "AXRadioButton", "旧到新", "press", "")
	eventuallyNative(t, "ascending native filtered journal", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Journal != nil &&
			state.Response.Request.Sort == "ascending" && state.Response.Request.Account == "Assets:Bank" &&
			state.Response.Request.Text == "Needle" && len(state.Response.Journal.Transactions) == 9
	})
	response := model.workspace.State().Response
	if response.State.SnapshotID != snapshotID || response.Journal.Transactions[0].Narration != "Needle Txn 007" ||
		response.Journal.Transactions[len(response.Journal.Transactions)-1].Narration != "Needle Txn 103" {
		t.Fatalf("filtered native sort changed snapshot or transaction identity: snapshot=%q rows=%+v", response.State.SnapshotID, response.Journal.Transactions)
	}
	window.Invalidate()
	eventuallyNative(t, "native filtered transaction count and visible search control", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "9 笔匹配交易") && nativeRoleLabelExists(nodes, "AXTextField", "搜索流水")
	})
}

func TestNativeJournalPagingProjectsOnlyOneHundredOf52500Transactions(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	const transactionCount = 52500
	path := filepath.Join(t.TempDir(), "journal-52500.bean")
	var source strings.Builder
	source.Grow(transactionCount * 80)
	source.WriteString("option \"operating_currency\" \"USD\"\n2026-01-01 open Assets:Cash USD\n2026-01-01 open Assets:Bank USD\n2026-01-01 open Income:Salary USD\n")
	for index := 0; index < transactionCount; index++ {
		date := 2 + index/(transactionCount/4)
		narration := fmt.Sprintf("Txn %05d", index)
		if index%10 == 0 {
			narration = "Needle " + narration
		}
		account := "Assets:Cash"
		if index%4 == 0 {
			account = "Assets:Bank"
		}
		fmt.Fprintf(&source, "2026-01-%02d * %q\n  %s 1 USD\n  Income:Salary -1 USD\n", date, narration, account)
	}
	if err := os.WriteFile(path, []byte(source.String()), 0600); err != nil {
		t.Fatal(err)
	}

	openStarted := time.Now()
	model, window := openNativeLedgerWindow(t, path)
	dashboardReady := time.Since(openStarted)
	opened := model.workspace.State()
	if opened.Response == nil || opened.Response.Summary.TransactionsCount != transactionCount {
		var observed int
		if opened.Response != nil {
			observed = opened.Response.Summary.TransactionsCount
		}
		t.Fatalf("native large pagination fixture has %d transactions, want %d", observed, transactionCount)
	}
	snapshotID := opened.Ledger.SnapshotID
	journalStarted := time.Now()
	performNativeAccessibilityEventually(t, window, "AXButton", "流水", "press", "")
	eventuallyNative(t, "first native 52,500-transaction journal page", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Journal != nil && state.Response.Journal.TotalCount == transactionCount && state.Response.Journal.Offset == 0 && len(state.Response.Journal.Transactions) == 100 && state.Response.Journal.HasMore && !state.Busy
	})
	eventuallyNative(t, "first native 52,500-transaction accessibility range", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), "显示 1–100")
	})
	journalPageReady := time.Since(journalStarted)
	first := model.workspace.State().Response
	if first.Request.SnapshotID != snapshotID || first.Request.Window != "all" || first.Request.DateFrom != "" || first.Request.DateTo != "" || first.Journal.Transactions[0].Narration != "Txn 52499" || first.Journal.Transactions[99].Narration != "Needle Txn 52400" {
		t.Fatalf("native large journal first page changed its snapshot: opened=%q response=%+v", snapshotID, first.Request)
	}

	nextPageStarted := time.Now()
	performNativeAccessibilityEventually(t, window, "AXButton", "下一页", "press", "")
	eventuallyNative(t, "second native 52,500-transaction journal page", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Journal != nil && state.Response.Journal.TotalCount == transactionCount && state.Response.Journal.Offset == 100 && len(state.Response.Journal.Transactions) == 100 && state.Response.Journal.HasMore && !state.Busy
	})
	eventuallyNative(t, "second native 52,500-transaction accessibility range", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), "显示 101–200")
	})
	nextPageReady := time.Since(nextPageStarted)
	second := model.workspace.State().Response
	if second.Request.SnapshotID != snapshotID || second.Request.Window != "all" || second.Request.DateFrom != "" || second.Request.DateTo != "" || second.Journal.Transactions[0].Narration != "Txn 52399" || second.Journal.Transactions[99].Narration != "Needle Txn 52300" {
		t.Fatalf("native large journal second page changed its snapshot: opened=%q response=%+v", snapshotID, second.Request)
	}

	setNativeFieldValueEventually(t, window, "开始日期 YYYY-MM-DD", "2026-01-03")
	setNativeFieldValueEventually(t, window, "结束日期 YYYY-MM-DD", "2026-01-03")
	clickNativeAccessibilityEventually(t, window, "AXButton", "应用筛选")
	var dateFilterObserved string
	dateFilterReady := false
	dateFilterDeadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(dateFilterDeadline) {
		state := model.workspace.State()
		dateFilterObserved = fmt.Sprintf("phase=%s busy=%t response=<nil>", state.Phase, state.Busy)
		if response := state.Response; response != nil {
			dateFilterObserved = fmt.Sprintf("phase=%s busy=%t request=%+v", state.Phase, state.Busy, response.Request)
			if response.Journal != nil {
				dateFilterObserved += fmt.Sprintf(" total=%d offset=%d rows=%d", response.Journal.TotalCount, response.Journal.Offset, len(response.Journal.Transactions))
			}
		}
		if !state.Busy && state.Response != nil && state.Response.Journal != nil &&
			state.Response.Request.Window == "all" && state.Response.Request.DateFrom == "2026-01-03" &&
			state.Response.Request.DateTo == "2026-01-03" && state.Response.Journal.TotalCount == 13125 &&
			state.Response.Journal.Offset == 0 && len(state.Response.Journal.Transactions) == 100 {
			dateFilterReady = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !dateFilterReady {
		t.Fatalf("date-filtered large journal was not ready: %s", dateFilterObserved)
	}
	if got := model.workspace.State().Response.State.SnapshotID; got != snapshotID {
		t.Fatalf("date filtering the large journal changed its snapshot: got=%q want=%q", got, snapshotID)
	}
	window.Invalidate()
	eventuallyNative(t, "visible date-filtered large journal count", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), "13125 笔匹配交易")
	})

	performNativeAccessibilityEventually(t, window, "AXPopUpButton", "全部账户", "press", "")
	clickNativeAccessibilityEventually(t, window, "AXStaticText", "Assets:Bank")
	eventuallyNative(t, "account-filtered 52,500-transaction journal", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Journal != nil &&
			state.Response.Request.DateFrom == "2026-01-03" && state.Response.Request.DateTo == "2026-01-03" &&
			state.Response.Request.Account == "Assets:Bank" && state.Response.Journal.TotalCount == 3281 &&
			state.Response.Journal.Offset == 0 && len(state.Response.Journal.Transactions) == 100
	})
	if got := model.workspace.State().Response.State.SnapshotID; got != snapshotID {
		t.Fatalf("account filtering the large journal changed its snapshot: got=%q want=%q", got, snapshotID)
	}
	window.Invalidate()
	eventuallyNative(t, "visible account-filtered large journal count", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), "3281 笔匹配交易")
	})

	setNativeFieldValueEventually(t, window, "搜索流水", "Needle")
	clickNativeAccessibilityEventually(t, window, "AXButton", "筛选流水")
	eventuallyNative(t, "text-filtered 52,500-transaction journal", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Journal != nil &&
			state.Response.Request.DateFrom == "2026-01-03" && state.Response.Request.DateTo == "2026-01-03" &&
			state.Response.Request.Account == "Assets:Bank" && state.Response.Request.Text == "Needle" &&
			state.Response.Journal.TotalCount == 656 && state.Response.Journal.Offset == 0 && len(state.Response.Journal.Transactions) == 100
	})
	if got := model.workspace.State().Response.State.SnapshotID; got != snapshotID {
		t.Fatalf("text filtering the large journal changed its snapshot: got=%q want=%q", got, snapshotID)
	}
	window.Invalidate()
	eventuallyNative(t, "visible text-filtered large journal count", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "656 笔匹配交易") && nativeRoleLabelExists(nodes, "AXTextField", "搜索流水")
	})
	performNativeAccessibilityEventually(t, window, "AXRadioButton", "旧到新", "press", "")
	eventuallyNative(t, "ascending text-filtered large journal", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Journal != nil &&
			state.Response.Request.Sort == "ascending" && state.Response.Request.Account == "Assets:Bank" &&
			state.Response.Request.Text == "Needle" && state.Response.Journal.TotalCount == 656 &&
			state.Response.Journal.Offset == 0 && len(state.Response.Journal.Transactions) == 100
	})
	filteredFirst := model.workspace.State().Response
	checkFilteredPage := func(response *desktop.ViewResponse, offset, firstIndex int, seen map[string]struct{}) {
		t.Helper()
		if response.State.SnapshotID != snapshotID || response.Request.Window != "all" || response.Request.DateFrom != "2026-01-03" ||
			response.Request.DateTo != "2026-01-03" || response.Request.Account != "Assets:Bank" || response.Request.Text != "Needle" ||
			response.Request.Sort != "ascending" || response.Request.Offset != offset || response.Journal.TotalCount != 656 ||
			response.Journal.Limit != 100 || len(response.Journal.Transactions) != 100 {
			t.Fatalf("filtered large journal page %d has wrong context or bounds: request=%+v page=%+v", offset, response.Request, response.Journal)
		}
		for row, transaction := range response.Journal.Transactions {
			index, err := strconv.Atoi(strings.TrimPrefix(transaction.Narration, "Needle Txn "))
			if err != nil || transaction.Date != "2026-01-03" || index != firstIndex+row*20 {
				t.Fatalf("filtered page %d row %d mixed transaction identity: narration=%q date=%q parsed_index=%d err=%v", offset, row, transaction.Narration, transaction.Date, index, err)
			}
			foundAccount := false
			for _, posting := range transaction.Postings {
				if posting.Account == "Assets:Bank" {
					foundAccount = true
				}
			}
			if !foundAccount {
				t.Fatalf("filtered page %d row %d has no Assets:Bank posting: %+v", offset, row, transaction)
			}
			if _, duplicate := seen[transaction.ID]; duplicate {
				t.Fatalf("filtered large journal repeated transaction ID %q across pages", transaction.ID)
			}
			seen[transaction.ID] = struct{}{}
		}
	}
	seenFiltered := make(map[string]struct{}, 200)
	checkFilteredPage(filteredFirst, 0, 13140, seenFiltered)
	if filteredFirst.Journal.Transactions[0].Narration != "Needle Txn 13140" || filteredFirst.Journal.Transactions[99].Narration != "Needle Txn 15120" {
		t.Fatalf("ascending filtered first page has wrong edge identities: first=%q last=%q", filteredFirst.Journal.Transactions[0].Narration, filteredFirst.Journal.Transactions[99].Narration)
	}
	performNativeAccessibilityEventually(t, window, "AXButton", "下一页", "press", "")
	eventuallyNative(t, "second ascending text-filtered large journal page", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Journal != nil &&
			state.Response.Journal.TotalCount == 656 && state.Response.Journal.Offset == 100 && len(state.Response.Journal.Transactions) == 100
	})
	filteredSecond := model.workspace.State().Response
	checkFilteredPage(filteredSecond, 100, 15140, seenFiltered)
	if filteredSecond.Journal.Transactions[0].Narration != "Needle Txn 15140" || filteredSecond.Journal.Transactions[99].Narration != "Needle Txn 17120" {
		t.Fatalf("ascending filtered second page has wrong edge identities: first=%q last=%q", filteredSecond.Journal.Transactions[0].Narration, filteredSecond.Journal.Transactions[99].Narration)
	}
	window.Invalidate()
	eventuallyNative(t, "visible second filtered journal page", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), "显示 101–200")
	})
	t.Logf("warm-process native 52,500-transaction timings: ledger-open-to-accessible-dashboard=%s, dashboard-to-accessible-journal-page-1=%s, page-1-to-accessible-page-2=%s", dashboardReady, journalPageReady, nextPageReady)
}

func TestNativeDirectiveAndConfigModesRenderCoreRecords(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	model, window := openNativeLedgerWindowWithProfile(t, nativeFixturePath(t, "directives/all.bean"), "v3")
	window.Update(func() {
		// Configuration directives are source-ordered and must ignore the app's date filter.
		model.filterDateFrom = "2099-01-01"
		model.filterDateTo = "2099-01-31"
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "全部指令", "press", "")
	eventuallyNative(t, "native all-directives response", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Directives != nil && state.Response.Directives.Mode == "all" && !state.Busy
	})
	eventuallyNative(t, "native typed directives and generated lineage", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "由 方言行 生成") && nativeContentHas(nodes, "关联源指令") && nativeContentHas(nodes, "#explicit") && nativeContentHas(nodes, "1.25000000000000000001") && nativeContentHas(nodes, "NULL") && nativeContentHas(nodes, "TRUE")
	})
	performNativeAccessibilityEventually(t, window, "AXRadioButton", "配置", "press", "")
	eventuallyNative(t, "native configuration response without date scope", func() bool {
		state := model.workspace.State()
		if state.Busy || state.Response == nil || state.Response.Directives == nil || state.Response.Directives.Mode != "config" {
			return false
		}
		request := state.Response.Request
		return request.DateFrom == "" && request.DateTo == "" && state.Response.Directives.Page.TotalCount > 0
	})
	eventuallyNative(t, "native configuration source-order explanation and option value", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "配置记录按源码顺序读取") && nativeContentHas(nodes, "不应用日期") && nativeContentHas(nodes, "All directive types")
	})
}

func TestNativeDirectiveKindSearchAndPaginationKeepSourceIdentity(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	path := filepath.Join(t.TempDir(), "directive-page.bean")
	var source strings.Builder
	source.WriteString("option \"operating_currency\" \"USD\"\n")
	source.WriteString("2026-01-01 open Assets:Cash USD\n")
	source.WriteString("2026-01-02 note Assets:Cash \"noise-000\"\n")
	source.WriteString("2026-01-02 note Assets:Cash \"noise-001\"\n")
	for i := 0; i < 101; i++ {
		fmt.Fprintf(&source, "2026-01-02 note Assets:Cash \"needle-%03d\"\n", i)
	}
	if err := os.WriteFile(path, []byte(source.String()), 0o600); err != nil {
		t.Fatal(err)
	}

	model, window := openNativeLedgerWindow(t, path)
	performNativeAccessibilityEventually(t, window, "AXButton", "全部指令", "press", "")
	eventuallyNative(t, "initial native directives response", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Directives != nil && state.Response.Directives.Mode == "all"
	})

	performNativeAccessibilityEventually(t, window, "AXPopUpButton", "全部类型", "press", "")
	clickNativeAccessibilityEventually(t, window, "AXStaticText", "备注")
	eventuallyNative(t, "native note-kind filter", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Directives != nil &&
			state.Response.Directives.Mode == "all" && state.Response.Request.DirectiveKind == "note" &&
			state.Response.Directives.Page.TotalCount == 103 && len(state.Response.Directives.Records) == 100
	})
	performNativeAccessibilityEventually(t, window, "AXRadioButton", "正序", "press", "")
	eventuallyNative(t, "native ascending directive order", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Directives != nil &&
			state.Response.Request.DirectiveKind == "note" && state.Response.Request.Sort == "ascending" &&
			state.Response.Directives.Page.TotalCount == 103 && len(state.Response.Directives.Records) == 100
	})
	filtered := model.workspace.State().Response
	snapshotID := filtered.State.SnapshotID
	if got := filtered.Directives.Records[0].Source.Line; got != 3 {
		t.Fatalf("first filtered directive source line = %d, want 3", got)
	}
	if got := filtered.Directives.Records[99].Source.Line; got != 102 {
		t.Fatalf("last first-page directive source line = %d, want 102", got)
	}

	setNativeFieldValueEventually(t, window, "搜索指令", "needle-")
	performNativeAccessibilityEventually(t, window, "AXButton", "筛选", "press", "")
	eventuallyNative(t, "native applied directive search", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Directives != nil &&
			state.Response.Request.Text == "needle-" && state.Response.Request.DirectiveKind == "note" &&
			state.Response.Directives.Page.TotalCount == 101 && len(state.Response.Directives.Records) == 100
	})
	searchState := model.workspace.State()
	if searchState.Response.State.SnapshotID != snapshotID || searchState.Response.Directives.Records[0].Source.Line != 5 ||
		searchState.Response.Directives.Records[99].Source.Line != 104 {
		t.Fatalf("applied search changed snapshot or first source identity: %+v", searchState.Response.Directives)
	}

	performNativeAccessibilityEventually(t, window, "AXButton", "下一页", "press", "")
	eventuallyNative(t, "native directive second page", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Directives != nil &&
			state.Response.Request.Text == "needle-" && state.Response.Request.Offset == 100 &&
			state.Response.Directives.Page.TotalCount == 101 && len(state.Response.Directives.Records) == 1 &&
			!state.Response.Directives.Page.HasMore
	})
	secondPage := model.workspace.State().Response
	if secondPage.State.SnapshotID != snapshotID || secondPage.Directives.Records[0].Source.Line != 105 {
		t.Fatalf("native directive second page lost source identity or snapshot: %+v", secondPage.Directives)
	}

	performNativeAccessibilityEventually(t, window, "AXButton", "上一页", "press", "")
	eventuallyNative(t, "native directive page return", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Directives != nil &&
			state.Response.Request.Text == "needle-" && state.Response.Request.Offset == 0 &&
			len(state.Response.Directives.Records) == 100
	})
	returned := model.workspace.State().Response
	if returned.State.SnapshotID != snapshotID || returned.Directives.Records[0].Source.Line != 5 {
		t.Fatalf("native directive previous page changed source identity or snapshot: %+v", returned.Directives)
	}
}

func TestNativeDirectiveNestedValuesWrapWithoutLosingTypes(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to inspect real macOS directive text layout")
	}
	path := filepath.Join(t.TempDir(), "nested-directive.bean")
	narration := strings.Repeat("跨境交易嵌套字段完整显示，", 8)
	value := "[1.25000000000000000001, [None, TRUE, " + strconv.Quote(narration) + "]]"
	source := "option \"operating_currency\" \"USD\"\n2026-01-01 open Assets:Cash USD\n" +
		"2026-01-02 custom \"Nested value showcase\" \"demo\"\n  values: " + value + "\n"
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	model, window := openNativeLedgerWindow(t, path)
	window.SetSize(900, 600)
	if width, height := window.Size(); width != 900 || height != 600 {
		t.Fatalf("native directive window size = %dx%d, want 900x600", width, height)
	}
	performNativeAccessibilityEventually(t, window, "AXButton", "全部指令", "press", "")
	eventuallyNative(t, "nested directive page response", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Directives != nil && state.Response.Directives.Mode == "all" && !state.Busy
	})
	expectedValue := "[1.25000000000000000001, [NULL, TRUE, " + strconv.Quote(narration) + "]]"
	var valueFrame [4]float64
	var nodes []mygo.E2EAccessNode
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		nodes = mygo.E2EAccessibility(window)
		found := false
		for _, node := range nodes {
			if node.Role == "AXStaticText" && strings.Contains(node.Label, expectedValue) {
				valueFrame = node.Frame
				found = true
				break
			}
		}
		if found {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if valueFrame[3] == 0 {
		state := model.workspace.State()
		t.Fatalf("native nested directive value missing %q: workspace error=%q records=%+v accessibility=%s", expectedValue, state.Error, state.Response.Directives, nativeNodeSummary(nodes))
	}
	if valueFrame[3] <= 22 {
		t.Fatalf("long nested Chinese directive value did not wrap: Accessibility frame=%v", valueFrame)
	}
	contentWidth, contentHeight := window.ContentSize()
	if valueFrame[2] <= 0 || valueFrame[2] > float64(contentWidth) {
		t.Fatalf("nested directive value frame width %.0f exceeds native content width %d", valueFrame[2], contentWidth)
	}
	t.Logf("900x600 native nested directive value AX frame: %.0fx%.0f points in %dx%d content; NULL, TRUE and exact decimal retained", valueFrame[2], valueFrame[3], contentWidth, contentHeight)
}

func TestNativeEmptyLedgerStatementShowsNoActivity(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	model, window := openNativeLedgerWindow(t, nativeFixturePath(t, "native/empty.bean"))
	initial := model.workspace.State()
	if initial.Response == nil || !initial.Response.State.Valid || initial.Response.Summary.TransactionsCount != 0 {
		t.Fatalf("native empty ledger state is invalid or has transactions: %+v", initial)
	}
	performNativeAccessibilityEventually(t, window, "AXButton", "损益表", "press", "")
	eventuallyNative(t, "empty income-statement response", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "income_statement" && !state.Busy
	})
	eventuallyNative(t, "native empty-period explanation", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), "当前期间没有活动")
	})
}

func TestNativeMultiCurrencyBalancesKeepCurrenciesSeparate(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	model, window := openNativeLedgerWindowWithProfile(t, nativeFixturePath(t, "v3-parity/options.bean"), "v3")
	performNativeAccessibilityEventually(t, window, "AXButton", "资产负债表", "press", "")
	eventuallyNative(t, "native multi-currency balance sheet", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "balance_sheet" && state.Response.Request.Currency == "" &&
			!state.Busy && state.Response.Tree.Availability == "native-multi"
	})
	var balanceNodes []mygo.E2EAccessNode
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		balanceNodes = mygo.E2EAccessibility(window)
		if nativeContentHas(balanceNodes, "1 USD") && nativeContentHas(balanceNodes, "2 EUR") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !nativeContentHas(balanceNodes, "1 USD") || !nativeContentHas(balanceNodes, "2 EUR") {
		t.Fatalf("native balance sheet did not preserve exact amounts in their original currencies; AX=%s", nativeNodeSummary(balanceNodes))
	}
	if !nativeContentHas(balanceNodes, "直属余额") || !nativeContentHas(balanceNodes, "汇总余额") {
		t.Fatalf("native balance sheet omitted own or subtree balance columns; AX=%s", nativeNodeSummary(balanceNodes))
	}
	window.Focus()
	chooseNativePopupOptionEventually(t, window, "原币", "USD")
	eventuallyNative(t, "USD valuation draft remains unapplied until confirmation", func() bool {
		state := model.workspace.State()
		nodes := mygo.E2EAccessibility(window)
		return nativeRoleLabelExists(nodes, "AXPopUpButton", "USD") && nativeRoleLabelExists(nodes, "AXButton", "应用筛选") &&
			!state.Busy && state.Response != nil && state.Response.Request.Currency == ""
	})
	pressNativeAccessibilityEventually(t, window, "应用筛选")
	eventuallyNative(t, "missing-quote USD balance response", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "balance_sheet" && state.Response.Request.Currency == "USD" && !state.Busy && state.Response.Tree.Availability == "unavailable-currency"
	})
	eventuallyNative(t, "native missing-quote explanation", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "缺少本地报价，估值不完整") && nativeContentHas(nodes, "估值币种 USD") &&
			nativeContentHas(nodes, "已折算部分：1 USD") && nativeContentHas(nodes, "缺报价：EUR")
	})
}

func TestNativeStatementShowsDatedConvertedAccountValues(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to inspect converted values in a real macOS window")
	}
	model, window := openNativeLedgerWindowWithProfile(t, nativeFixturePath(t, "native/valuation.bean"), string(ledger.V3))
	performNativeAccessibilityEventually(t, window, "AXButton", "资产负债表", "press", "")
	eventuallyNative(t, "valuation fixture balance sheet", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "balance_sheet" && !state.Busy
	})
	window.Focus()
	chooseNativePopupOptionEventually(t, window, "原币", "USD")
	eventuallyNative(t, "USD valuation draft is rendered before entering the cutoff", func() bool {
		state := model.workspace.State()
		nodes := mygo.E2EAccessibility(window)
		return nativeRoleLabelExists(nodes, "AXPopUpButton", "USD") && nativeRoleLabelExists(nodes, "AXButton", "应用筛选") &&
			!state.Busy && state.Response != nil && state.Response.Request.Currency == ""
	})
	setNativeFieldValueEventually(t, window, "结束日期 YYYY-MM-DD", "2000-01-03")
	pressNativeAccessibilityEventually(t, window, "应用筛选")
	requestDeadline := time.Now().Add(15 * time.Second)
	var state workspace.State
	var nodes []mygo.E2EAccessNode
	requestReady := false
	for time.Now().Before(requestDeadline) {
		state = model.workspace.State()
		nodes = mygo.E2EAccessibility(window)
		requestReady = state.Response != nil && state.Response.Request.View == "balance_sheet" &&
			state.Response.Request.Currency == "USD" && state.Response.Request.DateTo == "2000-01-03" && !state.Busy
		if requestReady && nativeContentHas(nodes, "折算总计：140 USD") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !requestReady || !nativeContentHas(nodes, "折算总计：140 USD") {
		var request any
		if state.Response != nil {
			request = state.Response.Request
		}
		t.Fatalf("as-of USD statement did not use the Jan 3 EUR/USD quote: busy=%t request=%+v AX=%s", state.Busy, request, nativeNodeSummary(nodes))
	}
	quoteHelpVisible := false
	for _, node := range mygo.E2EAccessibility(window) {
		if strings.Contains(node.Help, "EUR/USD 2 @ 2000-01-03") {
			quoteHelpVisible = true
			break
		}
	}
	if !quoteHelpVisible {
		t.Fatalf("dated exact quote is missing from native accessibility help: %+v", mygo.E2EAccessibility(window))
	}
	setNativeFieldValueEventually(t, window, "结束日期 YYYY-MM-DD", "2000-01-02")
	pressNativeAccessibilityEventually(t, window, "应用筛选")
	eventuallyNative(t, "unquoted EUR remains a partial statement valuation", func() bool {
		state := model.workspace.State()
		nodes := mygo.E2EAccessibility(window)
		return state.Response != nil && state.Response.Request.DateTo == "2000-01-02" && !state.Busy &&
			nativeContentHas(nodes, "已折算部分：100 USD") && nativeContentHas(nodes, "缺报价：EUR")
	})
}

func TestNativeAccountDetailShowsCoreHistoryAndReturnsToStatement(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to inspect account detail in a real macOS window")
	}
	model, window := openNativeLedgerWindowWithProfile(t, nativeFixturePath(t, "native/account-detail.bean"), string(ledger.V3))
	initialSnapshot := model.workspace.State().Ledger.SnapshotID
	performNativeAccessibilityEventually(t, window, "AXButton", "资产负债表", "press", "")
	eventuallyNative(t, "account-detail balance sheet", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Request.View == "balance_sheet" && state.Ledger.SnapshotID == initialSnapshot
	})
	performNativeAccessibilityEventually(t, window, "AXRow", "Assets:Test:Child", "focus", "")
	eventuallyNative(t, "selected account exposes a native detail action", func() bool {
		var selected string
		window.Update(func() { selected = model.statementSelectedAccount })
		return selected == "Assets:Test:Child" && nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXButton", "查看账户详情 Assets:Test:Child")
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "查看账户详情 Assets:Test:Child", "press", "")
	eventuallyNative(t, "selected native account detail", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Request.View == "account" &&
			state.Response.Account != nil && state.Response.Account.AccountDetailProjection != nil
	})
	state := model.workspace.State()
	detail := state.Response.Account.AccountDetailProjection
	if state.Ledger.SnapshotID != initialSnapshot || detail.Account != "Assets:Test:Child" || detail.Closed == nil || *detail.Closed != "2000-01-05" || detail.Booking != "FIFO" {
		t.Fatalf("native account detail lost snapshot or core lifecycle data: snapshot=%s detail=%+v", state.Ledger.SnapshotID, detail)
	}
	eventuallyNative(t, "account lifecycle, history and selected transaction are accessible", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "已关闭") && nativeContentHas(nodes, "关闭日期 2000-01-05") &&
			nativeContentHas(nodes, "记账方式 FIFO") && nativeContentHas(nodes, "历史余额") &&
			nativeContentHas(nodes, "Child reduction") && nativeContentHas(nodes, "manual-review-7")
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "返回报表", "press", "")
	eventuallyNative(t, "account detail returns to the same statement snapshot", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Request.View == "balance_sheet" &&
			state.Ledger.SnapshotID == initialSnapshot && nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXButton", "查看账户详情 Assets:Test:Child")
	})
}

func TestNativeAccountTreeArrowKeysNavigateAndExpandHierarchy(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to exercise native account-tree keyboard navigation")
	}
	source, err := os.ReadFile(nativeFixturePath(t, "native/account-detail.bean"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "account-tree-keyboard.bean")
	if err := os.WriteFile(path, source, 0600); err != nil {
		t.Fatal(err)
	}
	model, window := openNativeLedgerWindow(t, path)
	performNativeAccessibilityEventually(t, window, "AXButton", "资产负债表", "press", "")
	eventuallyNative(t, "native balance-sheet account tree", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Request.View == "balance_sheet"
	})
	state := model.workspace.State()
	if state.Response == nil || state.Response.Tree == nil {
		t.Fatal("balance-sheet response has no account tree")
	}
	rootCount := len(state.Response.Tree.RootNodes)
	clickNativeAccessibilityEventually(t, window, "AXButton", "全部折叠")
	readTree := func() (rows int, openAssets, openTest bool, selected string) {
		mygo.RunOnMain(func() {
			rows = model.reportTree.Rows()
			openAssets = model.reportTree.Open.Has("Assets")
			openTest = model.reportTree.Open.Has("Assets:Test")
			selected = model.statementSelectedAccount
		})
		return
	}
	eventuallyNative(t, "all account branches are collapsed", func() bool {
		rows, openAssets, openTest, _ := readTree()
		return rows == rootCount && !openAssets && !openTest
	})
	rootRow := waitForStableNativeAccessibilityNode(t, window, "AXRow", "Assets", 3)
	if !mygo.E2EClickAccessibilityID(window, rootRow.ID) {
		t.Fatalf("could not click the stable root account row by its native identity: %d", rootRow.ID)
	}
	eventuallyNative(t, "root account row receives keyboard focus and selection", func() bool {
		_, _, _, selected := readTree()
		return selected == "Assets"
	})
	if !mygo.E2EKey(window, 124, "") { // macOS Right Arrow
		t.Fatal("could not send Right Arrow to the focused account row")
	}
	eventuallyNative(t, "Right Arrow expands the selected parent account", func() bool {
		rows, openAssets, _, selected := readTree()
		return rows == rootCount+2 && openAssets && selected == "Assets"
	})
	if !mygo.E2EKey(window, 124, "") {
		t.Fatal("could not send Right Arrow to the expanded account row")
	}
	eventuallyNative(t, "Right Arrow moves selection to the first child", func() bool {
		_, _, _, selected := readTree()
		return selected == "Assets:Test"
	})
	if !mygo.E2EKey(window, 124, "") {
		t.Fatal("could not send Right Arrow to the selected child account")
	}
	eventuallyNative(t, "Right Arrow expands the selected child branch", func() bool {
		rows, _, openTest, selected := readTree()
		return rows == rootCount+3 && openTest && selected == "Assets:Test"
	})
	if !mygo.E2EKey(window, 123, "") { // macOS Left Arrow
		t.Fatal("could not send Left Arrow to the expanded child account")
	}
	eventuallyNative(t, "Left Arrow collapses the selected child branch", func() bool {
		rows, _, openTest, selected := readTree()
		return rows == rootCount+2 && !openTest && selected == "Assets:Test"
	})
	if !mygo.E2EKey(window, 123, "") {
		t.Fatal("could not send Left Arrow to the collapsed child account")
	}
	eventuallyNative(t, "Left Arrow moves selection to the parent", func() bool {
		_, _, _, selected := readTree()
		return selected == "Assets"
	})
	if !mygo.E2EKey(window, 123, "") {
		t.Fatal("could not send Left Arrow to the expanded parent account")
	}
	eventuallyNative(t, "Left Arrow collapses the parent without losing selection", func() bool {
		rows, openAssets, _, selected := readTree()
		return rows == rootCount && !openAssets && selected == "Assets"
	})
}

func TestNativeAccountTreeSearchSelectionSurvivesReloadAndRestoresCollapse(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS account-tree window")
	}
	source, err := os.ReadFile(nativeFixturePath(t, "native/account-detail.bean"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "account-tree-reload.bean")
	if err := os.WriteFile(path, source, 0o600); err != nil {
		t.Fatal(err)
	}
	model, window := openNativeLedgerWindow(t, path)
	performNativeAccessibilityEventually(t, window, "AXButton", "资产负债表", "press", "")
	eventuallyNative(t, "native account-tree balance report", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Request.View == "balance_sheet"
	})
	eventuallyNative(t, "balance-sheet tree context to initialize", func() bool {
		var treeContext string
		mygo.RunOnMain(func() { treeContext = model.reportTreeContext })
		return treeContext == "balance_sheet\x00"+path
	})
	mygo.RunOnMain(func() { model.reportTree.Open.Clear() })
	eventuallyNative(t, "collapsed native account tree", func() bool {
		var open int
		mygo.RunOnMain(func() { open = model.reportTree.Open.Len() })
		return open == 0
	})

	setNativeFieldValueEventually(t, window, "搜索账户", "Assets:Test:Child")
	clickNativeAccessibilityEventually(t, window, "AXButton", "筛选账户")
	var searchRows int
	var assetsOpen, testOpen bool
	var searchApplied string
	var searchNodes []mygo.E2EAccessNode
	deadline := time.Now().Add(15 * time.Second)
	nextSearchClick := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		mygo.RunOnMain(func() {
			searchRows = model.reportTree.Rows()
			assetsOpen = model.reportTree.Open.Has("Assets")
			testOpen = model.reportTree.Open.Has("Assets:Test")
			searchApplied = model.statementSearch
		})
		searchNodes = mygo.E2EAccessibility(window)
		if searchRows == 3 && assetsOpen && testOpen && searchApplied == "assets:test:child" && nativeRoleLabelExists(searchNodes, "AXRow", "Assets:Test:Child") {
			break
		}
		if time.Now().After(nextSearchClick) && nativeRoleLabelExists(searchNodes, "AXButton", "筛选账户") {
			mygo.E2EClickAccessibility(window, "筛选账户")
			nextSearchClick = time.Now().Add(300 * time.Millisecond)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if searchRows != 3 || !assetsOpen || !testOpen || searchApplied != "assets:test:child" || !nativeRoleLabelExists(searchNodes, "AXRow", "Assets:Test:Child") {
		t.Fatalf("deep account search did not expand its path: rows=%d open=%v/%v search=%q, AX=%s", searchRows, assetsOpen, testOpen, searchApplied, nativeNodeSummary(searchNodes))
	}
	performNativeAccessibilityEventually(t, window, "AXRow", "Assets:Test:Child", "focus", "")
	eventuallyNative(t, "deep account selection", func() bool {
		var selected string
		mygo.RunOnMain(func() { selected = model.statementSelectedAccount })
		return selected == "Assets:Test:Child" && nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXButton", "查看账户详情 Assets:Test:Child")
	})
	initialSnapshot := model.workspace.State().Ledger.SnapshotID
	if err := os.WriteFile(path, append([]byte("; unrelated comment before the existing ledger\n"), source...), 0o600); err != nil {
		t.Fatal(err)
	}
	var reloaded workspace.State
	var open, rows int
	var selected, search string
	var reloadNodes []mygo.E2EAccessNode
	deadline = time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		reloaded = model.workspace.State()
		mygo.RunOnMain(func() {
			rows = model.reportTree.Rows()
			selected = model.statementSelectedAccount
			search = model.statementSearch
			assetsOpen = model.reportTree.Open.Has("Assets")
			testOpen = model.reportTree.Open.Has("Assets:Test")
			open = model.reportTree.Open.Len()
		})
		reloadNodes = mygo.E2EAccessibility(window)
		if reloaded.Phase == workspace.PhaseReady && !reloaded.Busy && reloaded.Ledger.SnapshotID != initialSnapshot &&
			reloaded.Response != nil && reloaded.Response.Request.View == "balance_sheet" && reloaded.Response.State.SnapshotID == reloaded.Ledger.SnapshotID &&
			rows == 3 && open > 0 && assetsOpen && testOpen && selected == "Assets:Test:Child" && search == "assets:test:child" &&
			nativeRoleLabelExists(reloadNodes, "AXRow", "Assets:Test:Child") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if reloaded.Phase != workspace.PhaseReady || reloaded.Busy || reloaded.Ledger.SnapshotID == initialSnapshot ||
		reloaded.Response == nil || reloaded.Response.Request.View != "balance_sheet" || reloaded.Response.State.SnapshotID != reloaded.Ledger.SnapshotID ||
		rows != 3 || open == 0 || !assetsOpen || !testOpen || selected != "Assets:Test:Child" || search != "assets:test:child" ||
		!nativeRoleLabelExists(reloadNodes, "AXRow", "Assets:Test:Child") {
		responseSnapshot := ""
		responseView := ""
		if reloaded.Response != nil {
			responseSnapshot = reloaded.Response.State.SnapshotID
			responseView = reloaded.Response.Request.View
		}
		t.Fatalf("account-tree state did not survive source reload: phase=%s busy=%t snapshot=%s/%s initial=%s request=%s rows=%d open=%d branches=%v/%v selected=%q search=%q AX=%s", reloaded.Phase, reloaded.Busy, reloaded.Ledger.SnapshotID, responseSnapshot, initialSnapshot, responseView, rows, open, assetsOpen, testOpen, selected, search, nativeNodeSummary(reloadNodes))
	}
	pressNativeAccessibilityEventually(t, window, "清除搜索")
	var clearOpen, clearRows, clearRootCount int
	var clearSearch string
	var clearNodes []mygo.E2EAccessNode
	deadline = time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		mygo.RunOnMain(func() {
			clearOpen = model.reportTree.Open.Len()
			clearRows = model.reportTree.Rows()
			clearSearch = model.statementSearch
			if response := model.responseFor("balance_sheet"); response != nil && response.Tree != nil {
				clearRootCount = len(response.Tree.RootNodes)
			}
		})
		if clearSearch == "" && clearOpen == 0 && clearRows == clearRootCount && clearRootCount > 0 {
			break
		}
		clearNodes = mygo.E2EAccessibility(window)
		time.Sleep(20 * time.Millisecond)
	}
	if clearSearch != "" || clearOpen != 0 || clearRows != clearRootCount || clearRootCount == 0 {
		t.Fatalf("clearing search did not restore the prior collapsed tree: search=%q open=%d rows=%d roots=%d, AX=%s", clearSearch, clearOpen, clearRows, clearRootCount, nativeNodeSummary(clearNodes))
	}
}

func TestNativeAccountTreeClearSearchRestoresCollapsedTree(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to clear account-tree search in a real macOS window")
	}
	model, window := openNativeLedgerWindow(t, nativeFixturePath(t, "native/account-detail.bean"))
	performNativeAccessibilityEventually(t, window, "AXButton", "资产负债表", "press", "")
	eventuallyNative(t, "native balance-sheet account tree", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Request.View == "balance_sheet" &&
			nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXOutline", "账户余额")
	})
	clickNativeAccessibilityEventually(t, window, "AXButton", "全部折叠")
	var rootCount int
	eventuallyNative(t, "collapsed account-tree baseline", func() bool {
		var rows, open int
		mygo.RunOnMain(func() {
			rows = model.reportTree.Rows()
			open = model.reportTree.Open.Len()
			if response := model.responseFor("balance_sheet"); response != nil && response.Tree != nil {
				rootCount = len(response.Tree.RootNodes)
			}
		})
		return rootCount > 0 && rows == rootCount && open == 0
	})

	setNativeFieldValueEventually(t, window, "搜索账户", "Assets:Test:Child")
	clickNativeAccessibilityEventually(t, window, "AXButton", "筛选账户")
	eventuallyNative(t, "deep account search expands the matching path", func() bool {
		var rows, open int
		var search string
		mygo.RunOnMain(func() {
			rows = model.reportTree.Rows()
			open = model.reportTree.Open.Len()
			search = model.statementSearch
		})
		return search == "assets:test:child" && rows == rootCount+1 && open == 2 && nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXRow", "Assets:Test:Child")
	})
	pressNativeAccessibilityEventually(t, window, "清除搜索")
	deadline := time.Now().Add(3 * time.Second)
	var clearRows, clearOpen int
	var clearSearch, clearDraft string
	for time.Now().Before(deadline) {
		mygo.RunOnMain(func() {
			clearRows = model.reportTree.Rows()
			clearOpen = model.reportTree.Open.Len()
			clearSearch = model.statementSearch
			clearDraft = model.statementSearchDraft
		})
		if clearSearch == "" && clearDraft == "" && clearOpen == 0 && clearRows == rootCount {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("native clear-search button did not restore collapsed tree: search=%q draft=%q open=%d rows=%d roots=%d AX=%s", clearSearch, clearDraft, clearOpen, clearRows, rootCount, nativeNodeSummary(mygo.E2EAccessibility(window)))
}

func TestNativeHoldingsNavigatesGroupsAverageHistoryAndLots(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to inspect holdings in a real macOS window")
	}
	model, window := openNativeLedgerWindowWithProfile(t, nativeFixturePath(t, "native/average-history.bean"), string(ledger.V3))
	initialSnapshot := model.workspace.State().Ledger.SnapshotID
	performNativeAccessibilityEventually(t, window, "AXButton", "持仓与成本", "press", "")
	eventuallyNative(t, "native holdings account filter", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Request.View == "holdings" && state.Response.Request.HoldingSection == "groups"
	})
	state := model.workspace.State()
	window.Focus()
	performNativeAccessibilityEventually(t, window, "AXTextField", "持仓账户", "focus", "")
	eventuallyNative(t, "native holdings account input client", func() bool {
		selection, document := mygo.E2EInputClient(window)
		return document == "" && selection == [2]int{0, 0}
	})
	if !mygo.E2EComposeOver(window, "Assets:Stock", len([]rune("Assets:Stock")), true, 0, 0) {
		t.Fatal("MyGo could not type the holdings account filter through the native input client")
	}
	deadline := time.Now().Add(3 * time.Second)
	var accountDraft, inputDocument string
	var inputSelection [2]int
	for time.Now().Before(deadline) {
		mygo.RunOnMain(func() { accountDraft = model.holdingAccountDraft })
		inputSelection, inputDocument = mygo.E2EInputClient(window)
		if accountDraft == "Assets:Stock" && inputDocument == accountDraft {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if accountDraft != "Assets:Stock" || inputDocument != accountDraft {
		t.Fatalf("native holdings account input did not reach the model: draft=%q selection=%v document=%q focused=%t AX=%s",
			accountDraft, inputSelection, inputDocument,
			nativeAccessibilityFieldFocused(mygo.E2EAccessibility(window), "持仓账户"), nativeNodeSummary(mygo.E2EAccessibility(window)))
	}
	performNativeAccessibilityEventually(t, window, "AXButton", "应用持仓范围", "press", "")
	deadline = time.Now().Add(15 * time.Second)
	var filteredState workspace.State
	for time.Now().Before(deadline) {
		filteredState = model.workspace.State()
		if !filteredState.Busy && filteredState.Response != nil && filteredState.Response.Request.View == "holdings" &&
			filteredState.Response.Request.HoldingSection == "groups" && filteredState.Response.Request.Account == "Assets:Stock" && filteredState.Response.Holdings != nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if filteredState.Response == nil || filteredState.Busy || filteredState.Response.Request.View != "holdings" ||
		filteredState.Response.Request.HoldingSection != "groups" || filteredState.Response.Request.Account != "Assets:Stock" || filteredState.Response.Holdings == nil {
		request := "<none>"
		if filteredState.Response != nil {
			request = fmt.Sprintf("%+v", filteredState.Response.Request)
		}
		t.Fatalf("native holdings groups request did not complete: phase=%v busy=%t request=%s filters=(section %q account %q draft %q scope %q group_by %q) window_focused=%t AX=%s",
			filteredState.Phase, filteredState.Busy, request, model.holdingSection, model.holdingAccount,
			model.holdingAccountDraft, model.holdingScope, model.holdingGroupBy, window.IsFocused(),
			nativeNodeSummary(mygo.E2EAccessibility(window)))
	}
	state = model.workspace.State()
	if state.Ledger.SnapshotID != initialSnapshot || len(state.Response.Holdings.Groups) != 1 {
		t.Fatalf("native holdings group identity changed: snapshot=%s projection=%+v", state.Ledger.SnapshotID, state.Response.Holdings)
	}
	group := state.Response.Holdings.Groups[0]
	if group.Quantity.Exact != "5" || group.CostBasis == nil || group.CostBasis.Exact != "40" || group.AverageCost == nil || group.AverageCost.Exact != "8" {
		t.Fatalf("native holdings page did not receive exact current inventory: %+v", group)
	}
	eventuallyNative(t, "native current inventory exact values", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "5 STK") && nativeContentHas(nodes, "40 USD") && nativeContentHas(nodes, "8 USD")
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "查看平均成本曲线", "press", "")
	eventuallyNative(t, "native AVERAGE history", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Request.HoldingSection == "history" &&
			state.Response.Holdings != nil && len(state.Response.Holdings.AverageHistory) == 1 &&
			state.Response.Holdings.AverageHistory[0].PointsPage.TotalCount == 7
	})
	eventuallyNative(t, "native AVERAGE history exact table", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "50/3 USD") && nativeContentHas(nodes, "空仓") && nativeContentHas(nodes, "8 USD")
	})
	performNativeAccessibilityEventually(t, window, "AXRadioButton", "持仓", "press", "")
	eventuallyNative(t, "return to native holdings groups", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Request.HoldingSection == "groups" && state.Response.Holdings != nil
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "查看当前批次", "press", "")
	eventuallyNative(t, "native current lot details", func() bool {
		state := model.workspace.State()
		if state.Busy || state.Response == nil || state.Response.Request.HoldingSection != "lots" || state.Response.Holdings == nil || len(state.Response.Holdings.Groups) != 1 {
			return false
		}
		page := state.Response.Holdings.Groups[0].LotsPage
		return page.TotalCount == 1 && len(state.Response.Holdings.Groups[0].Lots) == 1 && state.Ledger.SnapshotID == initialSnapshot
	})
	eventuallyNative(t, "native lot detail section is accessible", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), "当前批次")
	})
}

func TestNativeHoldingsKeepCostCurrencyGroupsExact(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to verify cost-currency groups in a real macOS window")
	}
	model, window := openNativeLedgerWindowWithProfile(t, nativeFixturePath(t, "native/holdings-unknowns.bean"), string(ledger.V3))
	initialSnapshot := model.workspace.State().Ledger.SnapshotID
	if initialSnapshot == "" {
		t.Fatal("native holdings fixture opened without a stable ledger snapshot")
	}

	pressNativeAccessibilityEventually(t, window, "持仓与成本")
	eventuallyNative(t, "native holdings groups before applying a cutoff", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Holdings != nil &&
			state.Response.Request.View == "holdings" && state.Response.Request.HoldingSection == "groups"
	})
	setNativeFieldValueEventually(t, window, "结束日期 YYYY-MM-DD", "2000-01-02")
	pressNativeAccessibilityEventually(t, window, "应用筛选")
	eventuallyNative(t, "native holdings cutoff retains the open inventory", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Holdings != nil &&
			state.Response.Request.View == "holdings" && state.Response.Request.DateTo == "2000-01-02" &&
			state.Response.State.SnapshotID == initialSnapshot
	})
	cutoff := model.workspace.State().Response
	if cutoff.Holdings.GroupsPage.TotalCount == 0 {
		t.Fatalf("native holdings cutoff unexpectedly has no groups: request=%+v snapshot=%+v holdings=%+v", cutoff.Request, cutoff.State, cutoff.Holdings)
	}

	window.Focus()
	chooseNativePopupOptionEventually(t, window, "持仓分组", "按成本币种")
	eventuallyNative(t, "pending cost-currency filter retains its controls", func() bool {
		nodes := mygo.E2EAccessibility(window)
		var groupBy string
		mygo.RunOnMain(func() { groupBy = model.holdingGroupBy })
		return groupBy == "cost_currency" && nativeRoleLabelExists(nodes, "AXButton", "应用持仓范围") &&
			nativeContentHas(nodes, "请应用持仓筛选以读取新结果")
	})
	pressNativeAccessibilityEventually(t, window, "应用持仓范围")
	deadline := time.Now().Add(15 * time.Second)
	var groupedReady bool
	for time.Now().Before(deadline) {
		state := model.workspace.State()
		groupedReady = !state.Busy && state.Response != nil && state.Response.Holdings != nil &&
			state.Response.Request.View == "holdings" && state.Response.Request.HoldingSection == "groups" &&
			state.Response.Request.DateTo == "2000-01-02" && state.Response.Request.Account == "" &&
			state.Response.Request.Scope == "subtree" && state.Response.Request.GroupBy == "cost_currency" &&
			state.Response.State.SnapshotID == initialSnapshot
		if groupedReady {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !groupedReady {
		state := model.workspace.State()
		var account, groupBy, scope string
		window.Update(func() { account, groupBy, scope = model.holdingAccount, model.holdingGroupBy, model.holdingScope })
		t.Fatalf("native holdings cost-currency request did not complete: model=(account %q group_by %q scope %q) busy=%t response=%+v AX=%s", account, groupBy, scope, state.Busy, state.Response, nativeNodeSummary(mygo.E2EAccessibility(window)))
	}

	basisByCurrency := make(map[string]string)
	unitsByCurrency := make(map[string]string)
	missingQuoteExplained := false
	for _, group := range model.workspace.State().Response.Holdings.Groups {
		if group.Commodity != "STK" {
			continue
		}
		if group.CostCurrency == nil || group.CostBasis == nil {
			t.Fatalf("native cost-currency projection lost its exact inventory basis: %+v", group)
		}
		basisByCurrency[*group.CostCurrency] = group.CostBasis.Exact
		unitsByCurrency[*group.CostCurrency] = group.Quantity.Exact
		if group.CostCurrency != nil && *group.CostCurrency == "USD" && group.MarketValuation != nil &&
			group.MarketValuation.Completeness == "unavailable" && len(group.MarketValuation.Missing) == 1 &&
			group.MarketValuation.Missing[0].Reason == "missing-local-quote" && len(group.MarketValuation.Quotes) == 0 {
			missingQuoteExplained = true
		}
	}
	if len(basisByCurrency) != 2 || basisByCurrency["USD"] != "100" || basisByCurrency["EUR"] != "60" ||
		unitsByCurrency["USD"] != "10" || unitsByCurrency["EUR"] != "3" || !missingQuoteExplained {
		var marketValuation any
		for _, group := range model.workspace.State().Response.Holdings.Groups {
			if group.Commodity == "STK" && group.CostCurrency != nil && *group.CostCurrency == "USD" {
				marketValuation = group.MarketValuation
				break
			}
		}
		t.Fatalf("native holdings combined cost currencies or invented a market value: basis=%v units=%v missing_quote_explained=%t market_valuation=%+v", basisByCurrency, unitsByCurrency, missingQuoteExplained, marketValuation)
	}
	eventuallyNative(t, "native holdings exact cost currencies are accessible", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "10 STK") && nativeContentHas(nodes, "100 USD") &&
			nativeContentHas(nodes, "3 STK") && nativeContentHas(nodes, "60 EUR") && !nativeContentHas(nodes, "160") &&
			nativeContentHas(nodes, "市价估值 · 不可用") && nativeContentHas(nodes, "缺少本地报价") && !nativeContentHas(nodes, "150 USD")
	})
}

func TestNativePricesNavigatesCommodityCatalogAndQuoteHistory(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to inspect prices in a real macOS window")
	}
	model, window := openNativeLedgerWindowWithProfile(t, nativeFixturePath(t, "native/prices.bean"), string(ledger.V3))
	initialSnapshot := model.workspace.State().Ledger.SnapshotID
	performNativeAccessibilityEventually(t, window, "AXButton", "商品与价格", "press", "")
	eventuallyNative(t, "native commodity catalog", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Request.View == "prices" &&
			state.Response.Request.PriceSection == "commodities" && state.Response.Prices != nil
	})
	state := model.workspace.State()
	if state.Ledger.SnapshotID != initialSnapshot || state.Response.Prices.CommoditiesPage.TotalCount < 3 {
		t.Fatalf("native commodity catalog lost the source snapshot or declarations: %+v", state.Response.Prices)
	}
	declared := map[string]bool{}
	for _, item := range state.Response.Prices.Commodities {
		declared[item.Currency] = len(item.Declarations) > 0
	}
	if !declared["UNQUOTED"] || !declared["DECLAREDONLY"] {
		t.Fatalf("native catalog projection lost declared and unquoted commodities: %+v", state.Response.Prices.Commodities)
	}
	var nodes []mygo.E2EAccessNode
	setNativeFieldValueEventually(t, window, "搜索商品目录", "UNQUOTED")
	performNativeAccessibilityEventually(t, window, "AXButton", "筛选商品", "press", "")
	eventuallyNative(t, "native unquoted commodity detail and declarations", func() bool {
		state := model.workspace.State()
		if state.Busy || state.Response == nil || state.Response.Prices == nil || state.Response.Request.CommoditySearch != "UNQUOTED" || state.Response.Prices.CommoditiesPage.TotalCount != 1 {
			return false
		}
		nodes = mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "UNQUOTED · 商品详情") && nativeContentHas(nodes, "商品声明") && nativeContentHas(nodes, "prices.bean")
	})
	setNativeFieldValueEventually(t, window, "搜索商品目录", "Assets:Stock")
	performNativeAccessibilityEventually(t, window, "AXButton", "筛选商品", "press", "")
	deadline := time.Now().Add(15 * time.Second)
	var catalogReady bool
	for time.Now().Before(deadline) {
		state := model.workspace.State()
		if !state.Busy && state.Response != nil && state.Response.Prices != nil && state.Response.Request.CommoditySearch == "Assets:Stock" && state.Response.Prices.CommoditiesPage.TotalCount == 2 {
			for _, item := range state.Response.Prices.Commodities {
				catalogReady = catalogReady || item.Currency == "STK"
			}
		}
		if catalogReady {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !catalogReady {
		state := model.workspace.State()
		var search, draft string
		window.Update(func() { search, draft = model.commoditySearch, model.commoditySearchDraft })
		t.Fatalf("native catalog did not isolate the STK account: model_search=%q draft=%q state=%+v AX=%s", search, draft, state.Response, nativeNodeSummary(mygo.E2EAccessibility(window)))
	}
	performNativeAccessibilityEventually(t, window, "AXRow", "STK", "focus", "")
	eventuallyNative(t, "native STK commodity is selected", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), "STK · 商品详情")
	})
	performNativeAccessibilityEventually(t, window, "AXPopUpButton", "选择报价币种", "press", "")
	optionDeadline := time.Now().Add(3 * time.Second)
	var optionNodes []mygo.E2EAccessNode
	for time.Now().Before(optionDeadline) {
		optionNodes = mygo.E2EAccessibility(window)
		if nativeRoleLabelExists(optionNodes, "AXStaticText", "JPY") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !nativeRoleLabelExists(optionNodes, "AXStaticText", "JPY") {
		t.Fatalf("native quote selector did not expose its options: %s", nativeNodeSummary(optionNodes))
	}
	var usdOptionID uint64
	var usdMatches int
	for _, node := range optionNodes {
		if node.Role == "AXStaticText" && node.Label == "USD" {
			usdOptionID = node.ID
			usdMatches++
		}
	}
	if usdMatches < 2 || !mygo.E2EClickAccessibilityID(window, usdOptionID) {
		t.Fatalf("native quote selector had no disambiguated USD option (matches=%d): %s", usdMatches, nativeNodeSummary(optionNodes))
	}
	eventuallyNative(t, "native STK catalog quote selection", func() bool {
		for _, node := range mygo.E2EAccessibility(window) {
			if node.Role == "AXPopUpButton" && node.Label == "选择报价币种" && node.Value == "USD" {
				return true
			}
		}
		return false
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "查看历史", "press", "")
	eventuallyNative(t, "native STK quote history", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Prices != nil &&
			state.Response.Request.PriceSection == "history" && state.Response.Request.PriceBase == "STK" &&
			state.Response.Request.PriceQuote == "USD" && state.Response.Prices.Base == "STK" && len(state.Response.Prices.Records) > 0
	})
	eventuallyNative(t, "native STK/USD cutoff quote", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Request.PriceQuote == "USD" &&
			state.Response.Prices != nil && state.Response.Prices.Selected != nil &&
			state.Response.Prices.Selected.Date == "2000-01-20" && state.Response.Prices.Selected.Rate.Exact == "99" &&
			state.Response.Prices.Selected.Source.Path != "" && state.Ledger.SnapshotID == initialSnapshot
	})
	eventuallyNative(t, "native price history exact source and cutoff are accessible", func() bool {
		nodes = mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "99 USD") && nativeContentHas(nodes, "2000-01-20") && nativeContentHas(nodes, "prices.bean")
	})
}

func TestNativeRapidDateCurrencyAndPageChangesStayOnSnapshot(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to exercise real macOS filter and navigation controls")
	}
	path := filepath.Join(t.TempDir(), "rapid-filters.bean")
	ledger := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Income:Sales USD
2026-01-02 * "January"
  Assets:Cash 10 USD
  Income:Sales -10 USD
2026-02-02 * "February"
  Assets:Cash 20 USD
  Income:Sales -20 USD
2026-03-02 * "March"
  Assets:Cash 30 USD
  Income:Sales -30 USD
`
	if err := os.WriteFile(path, []byte(ledger), 0600); err != nil {
		t.Fatal(err)
	}
	model, window := openNativeLedgerWindow(t, path)
	initial := model.workspace.State()
	if initial.Response == nil || initial.Response.Summary.TransactionsCount != 3 {
		t.Fatalf("rapid-filter test ledger did not open with three transactions: %+v", initial)
	}

	setDate := func(label, value string) {
		t.Helper()
		if !mygo.E2EPerformAccessibility(window, label, "value", value) {
			t.Fatalf("native text field %q rejected an accessibility value update", label)
		}
		eventuallyNative(t, "native value for "+label, func() bool {
			for _, node := range mygo.E2EAccessibility(window) {
				if node.Label == label && node.Value == value {
					return true
				}
			}
			return false
		})
	}
	selectUSD := func() {
		t.Helper()
		performNativeAccessibilityEventually(t, window, "AXPopUpButton", "原币", "press", "")
		clickNativeAccessibilityEventually(t, window, "AXStaticText", "USD")
		eventuallyNative(t, "native USD currency selection", func() bool {
			return model.filterCurrency == "USD"
		})
	}
	selectUSD()

	months := []struct {
		from, to, nextPage, nextView string
	}{
		{"2026-01-01", "2026-01-31", "资产负债表", "balance_sheet"},
		{"2026-02-01", "2026-02-28", "损益表", "income_statement"},
		{"2026-03-01", "2026-03-31", "流水", "journal"},
	}
	waitForRequest := func(view, from, to string) {
		t.Helper()
		currency := "USD"
		if view == "journal" {
			currency = "" // Journal rows retain original commodity amounts.
		}
		ready := func() bool {
			state := model.workspace.State()
			return state.Phase == workspace.PhaseReady && !state.Busy && !state.Ledger.Reloading &&
				state.Ledger.SnapshotID == initial.Ledger.SnapshotID && state.Response != nil &&
				state.Response.Request.View == view && state.Response.Request.DateFrom == from &&
				state.Response.Request.DateTo == to && state.Response.Request.Currency == currency
		}
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) && !ready() {
			time.Sleep(20 * time.Millisecond)
		}
		if ready() {
			return
		}
		state := model.workspace.State()
		var response any
		if state.Response != nil {
			response = struct {
				Request      any
				Transactions int
			}{state.Response.Request, state.Response.Summary.TransactionsCount}
		}
		t.Fatalf("native %s request did not settle for %s–%s: page=%d status=%q state=%+v response=%+v AX=%s", view, from, to, model.page, model.status, state, response, nativeNodeSummary(mygo.E2EAccessibility(window)))
	}
	currentView := "dashboard"
	for _, month := range months {
		setDate("开始日期 YYYY-MM-DD", month.from)
		setDate("结束日期 YYYY-MM-DD", month.to)
		performNativeAccessibilityEventually(t, window, "AXButton", "应用筛选", "press", "")
		waitForRequest(currentView, month.from, month.to)
		performNativeAccessibilityEventually(t, window, "AXButton", month.nextPage, "press", "")
		currentView = month.nextView
		waitForRequest(currentView, month.from, month.to)
	}
	performNativeAccessibilityEventually(t, window, "AXButton", "财务概览", "press", "")
	waitForRequest("dashboard", months[2].from, months[2].to)
	eventuallyNative(t, "filtered March dashboard values in native accessibility", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "60 USD") && nativeContentHas(nodes, "30 USD")
	})
	state := model.workspace.State()
	if state.Response.Trends == nil || state.Response.Trends.DateFrom != months[2].from || state.Response.Trends.DateTo != months[2].to ||
		len(state.Response.Trends.CurrencySeries) != 1 || len(state.Response.Trends.CurrencySeries[0].MonthlyCashFlows) != 1 ||
		state.Response.Trends.CurrencySeries[0].MonthlyCashFlows[0].Income == nil ||
		state.Response.Trends.CurrencySeries[0].MonthlyCashFlows[0].Income.CompleteTotal == nil ||
		state.Response.Trends.CurrencySeries[0].MonthlyCashFlows[0].Income.CompleteTotal.Exact != "30" {
		t.Fatalf("final dashboard lost the filtered snapshot: response=%+v trends=%+v AX=%s", state.Response, state.Response.Trends, nativeNodeSummary(mygo.E2EAccessibility(window)))
	}
}

func TestNativeUIRemainsResponsiveDuringPendingRead(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to exercise native UI during a pending read")
	}
	path := filepath.Join(t.TempDir(), "pending-read.bean")
	ledger := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Income:Sales USD
2026-01-02 * "January"
  Assets:Cash 10 USD
  Income:Sales -10 USD
2026-02-02 * "February"
  Assets:Cash 20 USD
  Income:Sales -20 USD
2026-03-02 * "March"
  Assets:Cash 30 USD
  Income:Sales -30 USD
`
	if err := os.WriteFile(path, []byte(ledger), 0600); err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseRead := func() { releaseOnce.Do(func() { close(release) }) }
	readStarted := make(chan desktop.Request, 1)
	engine := &nativeReadGateEngine{release: release, started: readStarted}
	model, window := openNativeLedgerWindowWithEngine(t, path, "", engine)
	defer releaseRead()
	initial := model.workspace.State()

	performNativeAccessibilityEventually(t, window, "AXButton", "资产负债表", "press", "")
	selectNativeRequestEventually(t, readStarted, "balance_sheet")
	eventuallyNative(t, "blocked native balance-sheet read", func() bool {
		state := model.workspace.State()
		return state.Busy && state.Ledger.SnapshotID == initial.Ledger.SnapshotID
	})
	operation := model.workspace.State().Operation
	setDate := func(label, value string) {
		t.Helper()
		if !mygo.E2EPerformAccessibility(window, label, "value", value) {
			t.Fatalf("date field %q was not editable while a read was pending", label)
		}
		eventuallyNative(t, "pending-read date field "+label, func() bool {
			for _, node := range mygo.E2EAccessibility(window) {
				if node.Label == label && node.Value == value {
					return true
				}
			}
			return false
		})
	}
	setDate("开始日期 YYYY-MM-DD", "2026-03-01")
	setDate("结束日期 YYYY-MM-DD", "2026-03-31")
	performNativeAccessibilityEventually(t, window, "AXPopUpButton", "原币", "press", "")
	clickNativeAccessibilityEventually(t, window, "AXStaticText", "USD")
	eventuallyNative(t, "pending-read USD currency selector", func() bool {
		return nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXPopUpButton", "USD")
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "应用筛选", "press", "")
	eventuallyNative(t, "queued filtered request while first read is blocked", func() bool {
		state := model.workspace.State()
		return state.Busy && state.Operation > operation && state.Ledger.SnapshotID == initial.Ledger.SnapshotID
	})
	operation = model.workspace.State().Operation
	performNativeAccessibilityEventually(t, window, "AXButton", "财务概览", "press", "")
	eventuallyNative(t, "native navigation remains responsive during pending read", func() bool {
		state := model.workspace.State()
		return state.Busy && state.Operation > operation && state.Response != nil &&
			state.Response.Request.View == "dashboard" && state.Ledger.SnapshotID == initial.Ledger.SnapshotID &&
			nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXStaticText", "财务概览")
	})

	releaseRead()
	latestDashboardReady := func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Phase == workspace.PhaseReady && state.Ledger.SnapshotID == initial.Ledger.SnapshotID &&
			state.Response != nil && state.Response.Request.View == "dashboard" &&
			state.Response.Request.DateFrom == "2026-03-01" && state.Response.Request.DateTo == "2026-03-31" &&
			state.Response.Request.Currency == "USD"
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && !latestDashboardReady() {
		time.Sleep(20 * time.Millisecond)
	}
	if !latestDashboardReady() {
		state := model.workspace.State()
		var request any
		if state.Response != nil {
			request = state.Response.Request
		}
		t.Fatalf("latest dashboard request was not published after release: page=%d status=%q state=%+v request=%+v", model.page, model.status, state, request)
	}
	eventuallyNative(t, "latest dashboard accessibility after blocked read", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "60 USD") && nativeContentHas(nodes, "30 USD")
	})
}

func TestNativeQueryEditorPreservesMarkedComposition(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	model, window := openNativeCashflowWindow(t)
	performNativeAccessibilityEventually(t, window, "AXButton", "查询", "press", "")
	eventuallyNative(t, "query workspace response", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "query" && !state.Busy
	})
	eventuallyNative(t, "native BQL editor", func() bool {
		return nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXTextArea", "BQL 查询")
	})
	const initial = "SELECT cafe LIMIT 1"
	window.Update(func() { model.queryDraft = initial })
	eventuallyNative(t, "native BQL editor", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeRoleLabelExists(nodes, "AXTextArea", "BQL 查询") && nativeContentHas(nodes, initial)
	})
	window.Focus()
	performNativeAccessibilityEventually(t, window, "AXTextArea", "BQL 查询", "focus", "")
	eventuallyNative(t, "initial query text in focused native input client", func() bool {
		_, document := mygo.E2EInputClient(window)
		return document == initial
	})
	if !mygo.E2EComposeOver(window, "e", 1, false, 10, 1) {
		t.Fatal("MyGo could not deliver marked text to the native BQL editor")
	}
	eventuallyNative(t, "marked input composition", func() bool {
		var draft string
		mygo.RunOnMain(func() { draft = model.queryDraft })
		selection, document := mygo.E2EInputClient(window)
		return draft == "SELECT caf LIMIT 1" && document == initial && selection == [2]int{11, 0}
	})
	if !mygo.E2EComposeOver(window, "é", 1, true, -1, 0) {
		t.Fatal("MyGo could not commit the native input composition")
	}
	eventuallyNative(t, "committed accented text", func() bool {
		var draft string
		mygo.RunOnMain(func() { draft = model.queryDraft })
		selection, document := mygo.E2EInputClient(window)
		return draft == "SELECT café LIMIT 1" && document == draft && selection[1] == 0
	})
}

func TestNativeQueryEditorUndoRedoUsesEditMenuRoles(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to exercise native query-editor undo and redo")
	}
	model, window := openNativeCashflowWindow(t)
	performNativeAccessibilityEventually(t, window, "AXButton", "查询", "press", "")
	eventuallyNative(t, "query workspace response", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "query" && !state.Busy
	})
	const initial = "SELECT 1 LIMIT 1"
	const edited = "SELECT 2 LIMIT 1"
	window.Update(func() { model.queryDraft = initial })
	eventuallyNative(t, "initial query text in native accessibility tree", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), initial)
	})
	mygo.App.SetMenu(NewApplicationMenu(func(candidate *mygo.Window) *Model {
		if candidate == window {
			return model
		}
		return nil
	}, nil))
	t.Cleanup(func() { mygo.App.SetMenu(nil) })
	window.Focus()
	performNativeAccessibilityEventually(t, window, "AXTextArea", "BQL 查询", "focus", "")
	eventuallyNative(t, "focused query editor text", func() bool {
		_, document := mygo.E2EInputClient(window)
		return document == initial
	})
	if !mygo.E2EComposeOver(window, "2", 1, true, 7, 1) {
		t.Fatal("MyGo could not replace a query character through the native input client")
	}
	eventuallyNative(t, "committed query edit", func() bool {
		var draft string
		mygo.RunOnMain(func() { draft = model.queryDraft })
		_, document := mygo.E2EInputClient(window)
		return draft == edited && document == edited
	})
	if err := mygo.E2EActivateMenu("编辑", "撤销"); err != nil {
		t.Fatalf("activate native Edit > Undo: %v", err)
	}
	eventuallyNative(t, "native Edit > Undo restores the query", func() bool {
		var draft string
		mygo.RunOnMain(func() { draft = model.queryDraft })
		_, document := mygo.E2EInputClient(window)
		return draft == initial && document == initial
	})
	if err := mygo.E2EActivateMenu("编辑", "重做"); err != nil {
		t.Fatalf("activate native Edit > Redo: %v", err)
	}
	eventuallyNative(t, "native Edit > Redo restores the edit", func() bool {
		var draft string
		mygo.RunOnMain(func() { draft = model.queryDraft })
		_, document := mygo.E2EInputClient(window)
		return draft == edited && document == edited
	})
	if state := model.workspace.State(); state.QueryBusy || state.QueryResponse != nil {
		t.Fatalf("editing through Undo/Redo unexpectedly ran the query: busy=%t response=%v", state.QueryBusy, state.QueryResponse != nil)
	}
}

func TestNativeQueryEditorCopyAndPasteUsesEditMenuRoles(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to exercise native query-editor clipboard commands")
	}
	model, window := openNativeCashflowWindow(t)
	performNativeAccessibilityEventually(t, window, "AXButton", "查询", "press", "")
	eventuallyNative(t, "query workspace response", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "query" && !state.Busy
	})
	const initial = "SELECT '组合💰' LIMIT 1"
	const pasted = "SELECT 'paste ✓' LIMIT 1"
	preserveNativePasteboard(t)
	window.Update(func() { model.queryDraft = initial })
	eventuallyNative(t, "initial Unicode query text in native accessibility tree", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), initial)
	})
	mygo.App.SetMenu(NewApplicationMenu(func(candidate *mygo.Window) *Model {
		if candidate == window {
			return model
		}
		return nil
	}, nil))
	t.Cleanup(func() { mygo.App.SetMenu(nil) })
	window.Focus()
	performNativeAccessibilityEventually(t, window, "AXTextArea", "BQL 查询", "focus", "")
	eventuallyNative(t, "initial Unicode query text in focused native input client", func() bool {
		_, document := mygo.E2EInputClient(window)
		return document == initial
	})
	if err := mygo.E2EActivateMenu("编辑", "全选"); err != nil {
		t.Fatalf("activate native Edit > Select All: %v", err)
	}
	eventuallyNative(t, "native Select All updates the input client selection", func() bool {
		selected, document := mygo.E2EInputClient(window)
		return document == initial && selected == [2]int{0, len(utf16.Encode([]rune(initial)))}
	})
	if err := mygo.E2EActivateMenu("编辑", "拷贝"); err != nil {
		t.Fatalf("activate native Edit > Copy: %v", err)
	}
	eventuallyNative(t, "native Edit > Copy updates the system clipboard", func() bool { return mygo.Clipboard.ReadText() == initial })
	mygo.Clipboard.WriteText(pasted)
	if err := mygo.E2EActivateMenu("编辑", "粘贴"); err != nil {
		t.Fatalf("activate native Edit > Paste: %v", err)
	}
	eventuallyNative(t, "native Edit > Paste replaces the selected query", func() bool {
		var draft string
		mygo.RunOnMain(func() { draft = model.queryDraft })
		_, document := mygo.E2EInputClient(window)
		return draft == pasted && document == pasted
	})
	if state := model.workspace.State(); state.QueryBusy || state.QueryResponse != nil {
		t.Fatalf("editing through Copy/Paste unexpectedly ran the query: busy=%t response=%v", state.QueryBusy, state.QueryResponse != nil)
	}
}

func TestNativeQueryErrorSelectsUnicodeRangeInInputClient(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	model, window := openNativeCashflowWindow(t)
	performNativeAccessibilityEventually(t, window, "AXButton", "查询", "press", "")
	eventuallyNative(t, "query workspace response", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "query" && !state.Busy
	})
	const query = "SELECT '💰', missing LIMIT 0"
	window.Update(func() { model.queryDraft = query })
	eventuallyNative(t, "Unicode query draft in native accessibility tree", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), query)
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "执行查询", "press", "")
	eventuallyNative(t, "native query parser range", func() bool {
		state := model.workspace.State()
		return state.QueryResponse != nil && !state.QueryBusy && state.QueryResponse.Error != nil && state.QueryResponse.Error.Range != nil
	})
	state := model.workspace.State()
	byteStart := strings.Index(query, "missing")
	byteEnd := byteStart + len("missing")
	if got := state.QueryResponse.Error.Range; got.Start != byteStart || got.End != byteEnd {
		t.Fatalf("native query UTF-8 range = [%d,%d), want [%d,%d)", got.Start, got.End, byteStart, byteEnd)
	}
	performNativeAccessibilityEventually(t, window, "AXButton", "定位到查询错误", "press", "")
	runeStart, runeEnd := len([]rune(query[:byteStart])), len([]rune(query[:byteEnd]))
	want := [2]int{len(utf16.Encode([]rune(query[:byteStart]))), len(utf16.Encode([]rune(query[byteStart:byteEnd])))}
	eventuallyNative(t, "native input-client selection of the Unicode parser token", func() bool {
		selection, document := mygo.E2EInputClient(window)
		return document == query && selection == want
	})
	if !mygo.E2ECompose(window, "account", runeEnd-runeStart, true) {
		t.Fatal("native input client could not replace the selected parser token")
	}
	const corrected = "SELECT '💰', account LIMIT 0"
	eventuallyNative(t, "committed edit after native error selection", func() bool {
		var draft string
		mygo.RunOnMain(func() { draft = model.queryDraft })
		selection, document := mygo.E2EInputClient(window)
		return draft == corrected && document == corrected && selection[1] == 0
	})
}

func TestNativeQueryExecutesOnlyOnUserActionAndRendersTypedRows(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	model, window := openNativeCashflowWindow(t)
	performNativeAccessibilityEventually(t, window, "AXButton", "查询", "press", "")
	eventuallyNative(t, "query workspace response", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "query" && !state.Busy
	})
	const query = "SELECT 1 AS v, 2 AS v, NULL AS n, '' AS blank LIMIT 1"
	window.Update(func() { model.queryDraft = query })
	eventuallyNative(t, "query draft in native accessibility tree", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), query)
	})
	eventuallyNative(t, "native execute-query button", func() bool {
		return nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXButton", "执行查询")
	})
	if state := model.workspace.State(); state.QueryBusy || state.QueryResponse != nil {
		t.Fatalf("editing a native query draft executed it implicitly: busy=%v has_response=%v", state.QueryBusy, state.QueryResponse != nil)
	}
	performNativeAccessibilityEventually(t, window, "AXButton", "执行查询", "press", "")
	eventuallyNative(t, "typed query response", func() bool { return model.workspace.State().QueryResponse != nil })
	state := model.workspace.State()
	eventuallyNative(t, "typed query completion", func() bool { return !model.workspace.State().QueryBusy })
	state = model.workspace.State()
	if response := state.QueryResponse; response.Error != nil || response.Execution == nil || response.Context.Text != query {
		t.Fatalf("native query response lost its success or captured text: %+v", response)
	}
	if execution := state.QueryResponse.Execution; execution.ReturnedRows != 1 || len(execution.Result.Columns) != 4 || len(execution.Result.Rows) != 1 {
		t.Fatalf("native typed query shape = rows:%d columns:%d", state.QueryResponse.Execution.ReturnedRows, len(state.QueryResponse.Execution.Result.Columns))
	}
	eventuallyNative(t, "typed result in native accessibility tree", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "SELECT 1 AS v, 2 AS v, NULL AS n, '' AS blank LIMIT 1") && nativeContentHas(nodes, "NULL") && nativeContentHas(nodes, "1 行")
	})
}

func TestNativeQueryResultSortAndKeyboardSelectionKeepRowIdentity(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to exercise the native query result table")
	}
	preserveNativePasteboard(t)
	model, window := openNativeCashflowWindow(t)
	window.SetSize(900, 600)
	if width, height := window.Size(); width != 900 || height != 600 {
		t.Fatalf("native query-result window size = %dx%d, want 900x600", width, height)
	}
	mygo.App.SetMenu(NewApplicationMenu(func(candidate *mygo.Window) *Model {
		if candidate == window {
			return model
		}
		return nil
	}, nil))
	t.Cleanup(func() { mygo.App.SetMenu(nil) })
	window.Focus()
	for _, scale := range []float32{1.25, 1.5} {
		if err := mygo.E2EActivateMenu("显示", "放大文字"); err != nil {
			t.Fatalf("activate text-size increase to %.0f%%: %v", scale*100, err)
		}
		eventuallyNative(t, fmt.Sprintf("query-result text scale reaches %.0f%%", scale*100), func() bool {
			var got float32
			mygo.RunOnMain(func() { got = model.textScale })
			return got == scale
		})
	}
	performNativeAccessibilityEventually(t, window, "AXButton", "查询", "press", "")
	eventuallyNative(t, "query workspace response", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "query" && !state.Busy
	})
	const query = "SELECT narration, number WHERE account = 'Assets:Bank'"
	window.Update(func() { model.queryDraft = query })
	eventuallyNative(t, "native query draft", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), query)
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "执行查询", "press", "")
	eventuallyNative(t, "native typed query completion", func() bool {
		state := model.workspace.State()
		return state.QueryResponse != nil && !state.QueryBusy
	})
	response := model.workspace.State().QueryResponse
	if response.Error != nil || response.Execution == nil || len(response.Execution.Result.Rows) != 2 || len(response.Execution.Result.Columns) != 2 {
		t.Fatalf("native sort query returned an unexpected shape: %+v", response)
	}
	rows := response.Execution.Result.Rows
	if rows[0][0].Value != "Salary" || rows[1][0].Value != "Food" {
		t.Fatalf("native query source order = %v/%v, want Salary/Food", rows[0][0].Value, rows[1][0].Value)
	}
	eventuallyNative(t, "native source rows are accessible", func() bool {
		return nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXRow", "查询结果第 1 行") &&
			nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXStaticText", "Salary")
	})
	performNativeAccessibilityEventually(t, window, "AXRow", "查询结果第 1 行", "focus", "")
	eventuallyNative(t, "Salary row is selected", func() bool {
		nodes := mygo.E2EAccessibility(window)
		var selection int
		mygo.RunOnMain(func() { selection = model.querySelection })
		return selection == 0 && nativeContentHas(nodes, "所选结果行 · 1") && nativeContentHas(nodes, "Salary")
	})
	if !mygo.E2EKey(window, 119, "") { // macOS End
		t.Fatal("could not send End to the focused native result table")
	}
	eventuallyNative(t, "End selects the last native query row", func() bool {
		var selection int
		mygo.RunOnMain(func() { selection = model.querySelection })
		nodes := mygo.E2EAccessibility(window)
		return selection == 1 && nativeContentHas(nodes, "所选结果行 · 2") && nativeContentHas(nodes, "Food")
	})
	if !mygo.E2EKey(window, 115, "") { // macOS Home
		t.Fatal("could not send Home to the focused native result table")
	}
	eventuallyNative(t, "Home selects the first native query row", func() bool {
		var selection int
		mygo.RunOnMain(func() { selection = model.querySelection })
		nodes := mygo.E2EAccessibility(window)
		return selection == 0 && nativeContentHas(nodes, "所选结果行 · 1") && nativeContentHas(nodes, "Salary")
	})
	sortColumn := response.Execution.Result.Columns[1].ID
	performNativeAccessibilityEventually(t, window, "AXCell", "number", "press", "")
	eventuallyNative(t, "typed numeric result rows are sorted and selection follows Salary", func() bool {
		var selection int
		var order ui.SortOrder
		mygo.RunOnMain(func() {
			selection = model.querySelection
			order = model.querySort
		})
		nodes := mygo.E2EAccessibility(window)
		return order.Column == sortColumn && !order.Descending && selection == 1 &&
			nativeContentHas(nodes, "所选结果行 · 2") && nativeContentHas(nodes, "Salary")
	})
	performNativeAccessibilityEventually(t, window, "AXRow", "查询结果第 2 行", "focus", "")
	if !mygo.E2EKey(window, 126, "") { // macOS Up Arrow.
		t.Fatal("could not send Up to the focused native result table")
	}
	eventuallyNative(t, "Up moves selection after sorting without losing the result row", func() bool {
		var selection int
		mygo.RunOnMain(func() { selection = model.querySelection })
		nodes := mygo.E2EAccessibility(window)
		return selection == 0 && nativeContentHas(nodes, "所选结果行 · 1") && nativeContentHas(nodes, "Food")
	})
	if !mygo.E2EKey(window, 125, "") { // macOS Down Arrow.
		t.Fatal("could not send Down to the focused native result table")
	}
	eventuallyNative(t, "Down restores the sorted Salary row", func() bool {
		var selection int
		mygo.RunOnMain(func() { selection = model.querySelection })
		nodes := mygo.E2EAccessibility(window)
		return selection == 1 && nativeContentHas(nodes, "所选结果行 · 2") && nativeContentHas(nodes, "Salary")
	})
	expected, err := json.Marshal(struct {
		Context  desktop.QueryContext `json:"context"`
		Contract string               `json:"contract"`
		Columns  any                  `json:"columns"`
		Cells    any                  `json:"cells"`
	}{response.Context, response.Execution.Result.Contract, response.Execution.Result.Columns, response.Execution.Result.Rows[0]})
	if err != nil {
		t.Fatal(err)
	}
	const pendingClipboard = "orangecount-native-e2e-query-copy-pending"
	mygo.Clipboard.WriteText(pendingClipboard)
	performNativeAccessibilityEventually(t, window, "AXButton", "复制类型化结果行", "press", "")
	var clipboard string
	eventuallyNative(t, "the selected typed query row replaces the pending pasteboard sentinel", func() bool {
		clipboard = mygo.Clipboard.ReadText()
		return clipboard == string(expected)
	})
	var copied struct {
		Context  desktop.QueryContext `json:"context"`
		Contract string               `json:"contract"`
		Columns  []json.RawMessage    `json:"columns"`
		Cells    []json.RawMessage    `json:"cells"`
	}
	if err := json.Unmarshal([]byte(clipboard), &copied); err != nil {
		t.Fatalf("native copy button produced invalid typed JSON: %v (%q)", err, clipboard)
	}
	if clipboard != string(expected) || len(copied.Columns) != len(response.Execution.Result.Columns) || len(copied.Cells) != len(response.Execution.Result.Rows[0]) {
		t.Fatalf("native copy button lost the sorted row's original identity or typed context: copied=%s expected=%s", clipboard, expected)
	}
}

func TestNativeQueryCancelButtonCancelsCurrentExecution(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to exercise native query cancellation")
	}
	ledgerPath := filepath.Join(t.TempDir(), "query-cancel.bean")
	ledger := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Income:Salary USD
2026-01-02 * "Salary"
  Assets:Cash 10 USD
  Income:Salary -10 USD
`
	if err := os.WriteFile(ledgerPath, []byte(ledger), 0o600); err != nil {
		t.Fatal(err)
	}
	started := make(chan desktop.QueryRequest, 1)
	cancelled := make(chan uint64, 1)
	engine := &nativeQueryCancelGateEngine{started: started, cancelled: cancelled}
	model, window := openNativeLedgerWindowWithEngine(t, ledgerPath, "", engine)
	performNativeAccessibilityEventually(t, window, "AXButton", "查询", "press", "")
	eventuallyNative(t, "query workspace response", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "query" && !state.Busy
	})
	const queryText = "SELECT 1 AS value LIMIT 1"
	window.Update(func() { model.queryDraft = queryText })
	eventuallyNative(t, "query cancellation draft", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), queryText)
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "执行查询", "press", "")
	var request desktop.QueryRequest
	select {
	case request = <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("native query did not reach its gated session")
	}
	state := model.workspace.State()
	if request.SnapshotID != state.Ledger.SnapshotID || request.RequestID != state.QueryRequestID || request.Text != queryText || !state.QueryBusy {
		t.Fatalf("native cancel target is not the current query: request=%+v state=%+v", request, state)
	}
	eventuallyNative(t, "native Cancel Query button while query is running", func() bool {
		return nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXButton", "取消查询")
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "取消查询", "press", "")
	select {
	case requestID := <-cancelled:
		if requestID != request.RequestID {
			t.Fatalf("native cancel requested generation %d, want current generation %d", requestID, request.RequestID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("native Cancel Query action did not reach the current desktop session")
	}
	eventuallyNative(t, "cancelled native query response", func() bool {
		state := model.workspace.State()
		return !state.QueryBusy && !state.QueryCancelling && state.QueryResponse != nil && state.QueryResponse.Error != nil && state.QueryResponse.Error.Code == "cancelled"
	})
	absentSamples := 0
	eventuallyNative(t, "native Cancel Query action to disappear after execution ends", func() bool {
		if nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXButton", "取消查询") {
			absentSamples = 0
			return false
		}
		absentSamples++
		return absentSamples >= 3
	})
}

func TestNativeReloadDuringQueryRejectsLateResult(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to exercise native reload during query execution")
	}

	ledgerPath := filepath.Join(t.TempDir(), "reload-during-query.bean")
	writeLedger := func(includeBonus bool) {
		t.Helper()
		contents := `option "operating_currency" "USD"
2000-01-01 open Assets:Cash USD
2000-01-01 open Income:Salary USD
2000-01-02 * "Salary"
  Assets:Cash 10 USD
  Income:Salary -10 USD
`
		if includeBonus {
			contents += `2000-01-03 * "Bonus"
  Assets:Cash 5 USD
  Income:Salary -5 USD
`
		}
		if err := os.WriteFile(ledgerPath, []byte(contents), 0o600); err != nil {
			t.Fatalf("write reload-race ledger: %v", err)
		}
	}
	writeLedger(false)

	releaseQuery := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseQuery) }) }
	startedQuery := make(chan desktop.QueryRequest, 1)
	finishedQuery := make(chan struct{}, 1)
	engine := &nativeLateQueryGateEngine{release: releaseQuery, started: startedQuery, finished: finishedQuery}
	model, window := openNativeLedgerWindowWithEngine(t, ledgerPath, "", engine)
	t.Cleanup(release)
	t.Cleanup(func() { mygo.App.SetMenu(nil) })
	mygo.App.SetMenu(NewApplicationMenu(func(candidate *mygo.Window) *Model {
		if candidate == window {
			return model
		}
		return nil
	}, nil))

	eventuallyNative(t, "initial native dashboard", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseReady && !state.Busy && state.Response != nil && state.Response.Request.View == "dashboard"
	})
	initialSnapshot := model.workspace.State().Ledger.SnapshotID
	performNativeAccessibilityEventually(t, window, "AXButton", "查询", "press", "")
	eventuallyNative(t, "native query workspace", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "query" && !state.Busy
	})
	const queryText = "SELECT 1 AS value LIMIT 1"
	window.Update(func() { model.queryDraft = queryText })
	eventuallyNative(t, "native query draft", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), queryText)
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "执行查询", "press", "")
	var request desktop.QueryRequest
	select {
	case request = <-startedQuery:
	case <-time.After(5 * time.Second):
		t.Fatal("query execution did not enter the native reload-race gate")
	}
	state := model.workspace.State()
	if request.SnapshotID != initialSnapshot || request.SnapshotID != state.Ledger.SnapshotID || request.Text != queryText || request.RequestID != state.QueryRequestID || !state.QueryBusy {
		t.Fatalf("query started with unexpected identity before reload: request=%+v state=%+v", request, state)
	}

	writeLedger(true)
	window.Focus()
	if err := mygo.E2EActivateMenu("文件", "重新读取账本"); err != nil {
		t.Fatalf("activate native reload while query is running: %v", err)
	}
	reloadDeadline := time.Now().Add(15 * time.Second)
	var reloadedState workspace.State
	queryInvalidated := false
	for time.Now().Before(reloadDeadline) {
		reloadedState = model.workspace.State()
		queryInvalidated = !reloadedState.QueryBusy && !reloadedState.QueryCancelling && reloadedState.QueryResponse == nil
		if queryInvalidated {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !queryInvalidated {
		t.Fatalf("native reload did not invalidate the active query: busy=%t phase=%v epoch=%d snapshot=%q query_busy=%t query_cancelling=%t query_response=%+v request_id=%d initial_snapshot=%q",
			reloadedState.Busy, reloadedState.Phase, reloadedState.Epoch, reloadedState.Ledger.SnapshotID,
			reloadedState.QueryBusy, reloadedState.QueryCancelling, reloadedState.QueryResponse,
			reloadedState.QueryRequestID, initialSnapshot)
	}
	reloadDeadline = time.Now().Add(15 * time.Second)
	for time.Now().Before(reloadDeadline) {
		reloadedState = model.workspace.State()
		if !reloadedState.Busy && reloadedState.Phase == workspace.PhaseReady &&
			reloadedState.Ledger.SnapshotID != "" && reloadedState.Ledger.SnapshotID != initialSnapshot {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if reloadedState.Busy || reloadedState.Phase != workspace.PhaseReady ||
		reloadedState.Ledger.SnapshotID == "" || reloadedState.Ledger.SnapshotID == initialSnapshot {
		t.Fatalf("native reload did not publish the changed ledger snapshot: busy=%t phase=%v epoch=%d snapshot=%q initial_snapshot=%q response=%+v",
			reloadedState.Busy, reloadedState.Phase, reloadedState.Epoch, reloadedState.Ledger.SnapshotID,
			initialSnapshot, reloadedState.Response)
	}
	reloadedSnapshot := model.workspace.State().Ledger.SnapshotID
	if model.workspace.State().QueryResponse != nil {
		t.Fatal("reload retained a query response from the previous snapshot")
	}

	// Release the old operation only after the new snapshot is published. Its
	// completion must not repopulate the query page with an obsolete result.
	release()
	select {
	case <-finishedQuery:
	case <-time.After(5 * time.Second):
		t.Fatal("late native query did not finish after its gate was released")
	}
	eventuallyNative(t, "late query completion is ignored", func() bool {
		state := model.workspace.State()
		return state.Ledger.SnapshotID == reloadedSnapshot && !state.QueryBusy && !state.QueryCancelling && state.QueryResponse == nil
	})
}

func TestNativeProfileReplacementDuringQueryRejectsLateResult(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to exercise native profile replacement during query execution")
	}
	fixture, err := os.ReadFile(nativeFixturePath(t, "native/average-history.bean"))
	if err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(t.TempDir(), "profile-during-query.bean")
	if err := os.WriteFile(ledgerPath, fixture, 0o600); err != nil {
		t.Fatal(err)
	}
	releaseQuery := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseQuery) }) }
	startedQuery := make(chan desktop.QueryRequest, 1)
	finishedQuery := make(chan struct{}, 1)
	engine := &nativeLateQueryGateEngine{release: releaseQuery, started: startedQuery, finished: finishedQuery}
	model, window := openNativeLedgerWindowWithEngine(t, ledgerPath, string(ledger.Legacy), engine)
	t.Cleanup(release)

	eventuallyNative(t, "initial Legacy ledger", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseReady && !state.Busy && state.Ledger.SemanticsProfile == ledger.Legacy
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "查询", "press", "")
	eventuallyNative(t, "native query workspace", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "query" && !state.Busy
	})
	const queryText = "SELECT 1 AS value LIMIT 1"
	window.Update(func() { model.queryDraft = queryText })
	eventuallyNative(t, "native query draft", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), queryText)
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "执行查询", "press", "")
	var request desktop.QueryRequest
	select {
	case request = <-startedQuery:
	case <-time.After(5 * time.Second):
		t.Fatal("profile replacement query did not enter the late-result gate")
	}
	initial := model.workspace.State()
	if request.SnapshotID != initial.Ledger.SnapshotID || request.RequestID != initial.QueryRequestID || request.Text != queryText || !initial.QueryBusy {
		t.Fatalf("query started with unexpected identity before profile replacement: request=%+v state=%+v", request, initial)
	}

	performNativeAccessibilityEventually(t, window, "AXButton", "语义配置", "press", "")
	eventuallyNative(t, "Legacy/V3 profile preview while old query is running", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Migration != nil && state.Response.Request.View == "migration"
	})
	comparison := model.workspace.State().Response.Migration
	if comparison.InputIdentity == "" || comparison.V3.SnapshotID == "" {
		t.Fatalf("native profile preview is missing current input identity: %+v", comparison)
	}
	performNativeAccessibilityEventually(t, window, "AXButton", "应用 V3", "press", "")
	eventuallyNative(t, "V3 profile replacement invalidates running query", func() bool {
		state := model.workspace.State()
		return state.Epoch > initial.Epoch && state.Phase == workspace.PhaseReady && !state.Busy &&
			state.Ledger.SemanticsProfile == ledger.V3 && state.Ledger.SnapshotID == comparison.V3.SnapshotID &&
			state.QueryResponse == nil && !state.QueryBusy && state.Response != nil && state.Response.Migration != nil
	})
	replacement := model.workspace.State()
	if replacement.QueryResponse != nil {
		t.Fatal("profile replacement retained the previous profile's query response")
	}

	// The old query completes after the replacement session is ready. Its
	// cancelled result must not mutate the new profile's workspace state.
	release()
	select {
	case <-finishedQuery:
	case <-time.After(5 * time.Second):
		t.Fatal("late query did not finish after profile replacement released its gate")
	}
	eventuallyNative(t, "late query completion leaves V3 workspace unchanged", func() bool {
		state := model.workspace.State()
		return state.Epoch == replacement.Epoch && state.Ledger.SemanticsProfile == ledger.V3 &&
			state.Ledger.SnapshotID == replacement.Ledger.SnapshotID && !state.QueryBusy && state.QueryResponse == nil
	})
}

func TestNativeQueryCompletionWhileAwayFromQueryPageIsRestored(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to exercise query completion while another page is visible")
	}
	ledgerPath := nativeFixturePath(t, "native/cashflow.bean")
	releaseQuery := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseQuery) }) }
	startedQuery := make(chan desktop.QueryRequest, 1)
	finishedQuery := make(chan struct{}, 1)
	engine := &nativeLateQueryGateEngine{release: releaseQuery, started: startedQuery, finished: finishedQuery}
	model, window := openNativeLedgerWindowWithEngine(t, ledgerPath, "", engine)
	t.Cleanup(release)
	eventuallyNative(t, "native dashboard before query", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "dashboard" && !state.Busy
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "查询", "press", "")
	eventuallyNative(t, "native query workspace", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "query" && !state.Busy
	})
	const queryText = "SELECT 2026 AS report_year LIMIT 1"
	window.Update(func() { model.queryDraft = queryText })
	eventuallyNative(t, "query draft before leaving page", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), queryText)
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "执行查询", "press", "")
	var request desktop.QueryRequest
	select {
	case request = <-startedQuery:
	case <-time.After(5 * time.Second):
		t.Fatal("query did not enter the background-completion gate")
	}
	if state := model.workspace.State(); request.SnapshotID != state.Ledger.SnapshotID || request.RequestID != state.QueryRequestID || !state.QueryBusy {
		t.Fatalf("background query started with unexpected identity: request=%+v state=%+v", request, state)
	}
	performNativeAccessibilityEventually(t, window, "AXButton", "财务概览", "press", "")
	eventuallyNative(t, "dashboard is visible while query remains active", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "dashboard" && !state.Busy && state.QueryBusy
	})

	// A page read must not cancel the independent query runner. Complete it
	// while the dashboard is visible, then return and inspect the saved result.
	release()
	select {
	case <-finishedQuery:
	case <-time.After(5 * time.Second):
		t.Fatal("query did not finish while the dashboard was visible")
	}
	eventuallyNative(t, "query response is retained off-page", func() bool {
		state := model.workspace.State()
		return !state.QueryBusy && state.QueryResponse != nil && state.QueryResponse.Context.RequestID == request.RequestID && state.Response != nil && state.Response.Request.View == "dashboard"
	})
	state := model.workspace.State()
	if state.QueryResponse.Error != nil || state.QueryResponse.Execution == nil || state.QueryResponse.Context.Text != queryText || state.QueryResponse.Context.SnapshotID != request.SnapshotID {
		t.Fatalf("query completed off-page without preserving its successful execution context: %+v", state.QueryResponse)
	}
	performNativeAccessibilityEventually(t, window, "AXButton", "查询", "press", "")
	eventuallyNative(t, "completed query is restored when the query page is reopened", func() bool {
		state := model.workspace.State()
		nodes := mygo.E2EAccessibility(window)
		return state.QueryResponse != nil && state.QueryResponse.Context.RequestID == request.RequestID &&
			nativeContentHas(nodes, queryText) && nativeContentHas(nodes, "1 行")
	})
}

func TestNativeQueryCatalogueIsLedgerScopedAndReloadSafe(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to exercise query catalogues across real ledger windows")
	}

	root := t.TempDir()
	firstLedger := filepath.Join(root, "first.bean")
	secondLedger := filepath.Join(root, "second.bean")
	writeCatalogueLedger := func(path, queryName string) {
		t.Helper()
		contents := fmt.Sprintf("option \"title\" \"Query catalogue lifecycle\"\n2000-01-01 open Assets:Cash USD\n2000-01-02 query %q %q\n", queryName, "SELECT account LIMIT 0")
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatalf("write query catalogue ledger %s: %v", filepath.Base(path), err)
		}
	}
	writeCatalogueLedger(firstLedger, "First ledger query")
	writeCatalogueLedger(secondLedger, "Second ledger query")

	presetStore := querypresets.NewStore(filepath.Join(root, "query-presets-v1.json"))
	if _, err := presetStore.Save(firstLedger, "First only preset", "SELECT 101 AS value LIMIT 1"); err != nil {
		t.Fatal(err)
	}
	if _, err := presetStore.Save(secondLedger, "Second only preset", "SELECT 202 AS value LIMIT 1"); err != nil {
		t.Fatal(err)
	}

	releaseReload := make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(releaseReload) })
	reloadStarted := make(chan struct{}, 1)
	model := newModelWithEngine(true, &nativeQueryReloadGateEngine{started: reloadStarted, release: releaseReload})
	model.SetQueryPresets(presetStore)
	var window *mygo.Window
	mygo.RunOnMain(func() {
		options := model.WindowOptions()
		options.Title = "OrangeCount Go UI query catalogue lifecycle E2E"
		window = mygo.NewWindow(options)
		model.SetWindow(window)
		window.OnClosed(model.Close)
	})
	t.Cleanup(func() {
		mygo.App.SetMenu(nil)
		window.Destroy()
		model.Close()
		select {
		case <-model.workspace.Done():
		case <-time.After(10 * time.Second):
			t.Error("query catalogue workspace did not close")
		}
	})
	mygo.App.SetMenu(NewApplicationMenu(func(candidate *mygo.Window) *Model {
		if candidate == window {
			return model
		}
		return nil
	}, nil))

	model.OpenLedgerPath(firstLedger)
	eventuallyNative(t, "first ledger dashboard", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseReady && !state.Busy && state.Response != nil && state.Response.Request.View == "dashboard"
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "查询", "press", "")
	eventuallyNative(t, "first ledger query catalogue and preset", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "First ledger query") && nativeRoleLabelExists(nodes, "AXButton", "填入草稿 · First only preset")
	})
	firstSnapshot := model.workspace.State().Ledger.SnapshotID
	performNativeAccessibilityEventually(t, window, "AXButton", "资产负债表", "press", "")
	eventuallyNative(t, "first ledger balance sheet before history reset", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "balance_sheet" && !state.Busy
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "流水", "press", "")
	eventuallyNative(t, "first ledger journal before history reset", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "journal" && !state.Busy
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "返回", "press", "")
	eventuallyNative(t, "first ledger history has both directions", func() bool {
		return len(model.pageHistoryBack) > 0 && len(model.pageHistoryForward) > 0
	})

	model.OpenLedgerPath(secondLedger)
	eventuallyNative(t, "second ledger dashboard", func() bool {
		state := model.workspace.State()
		return state.ActivePath == secondLedger && state.Phase == workspace.PhaseReady && !state.Busy && state.Response != nil && state.Response.Request.View == "dashboard" && len(model.pageHistoryBack) == 0 && len(model.pageHistoryForward) == 0
	})
	eventuallyNative(t, "second ledger native welcome content", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "账本已读取") && nativeContentHas(nodes, filepath.Base(secondLedger))
	})
	eventuallyNative(t, "navigation controls become disabled after ledger switch", func() bool {
		for _, label := range []string{"返回", "前进"} {
			found, allowed := mygo.E2EAccessibilityPressAllowed(window, label)
			if !found || allowed {
				return false
			}
		}
		return true
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "查询", "press", "")
	eventuallyNative(t, "second ledger catalogue and isolated preset", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "Second ledger query") && nativeRoleLabelExists(nodes, "AXButton", "填入草稿 · Second only preset") && !nativeRoleLabelExists(nodes, "AXButton", "填入草稿 · First only preset")
	})

	model.OpenLedgerPath(firstLedger)
	eventuallyNative(t, "reopened first ledger dashboard", func() bool {
		state := model.workspace.State()
		return state.ActivePath == firstLedger && state.Phase == workspace.PhaseReady && !state.Busy && state.Response != nil && state.Response.Request.View == "dashboard"
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "查询", "press", "")
	eventuallyNative(t, "reopened first ledger catalogue and preset", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "First ledger query") && nativeRoleLabelExists(nodes, "AXButton", "填入草稿 · First only preset") && !nativeRoleLabelExists(nodes, "AXButton", "填入草稿 · Second only preset")
	})
	if firstSnapshot == "" {
		t.Fatal("first ledger did not publish a snapshot")
	}

	writeCatalogueLedger(firstLedger, "Updated first ledger query")
	window.Focus()
	if err := mygo.E2EActivateMenu("文件", "重新读取账本"); err != nil {
		t.Fatalf("activate native ledger reload command: %v", err)
	}
	select {
	case <-reloadStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("native reload did not reach the gated ledger session")
	}
	deadline := time.Now().Add(3 * time.Second)
	hiddenDuringReload := false
	var lastReloadState workspace.State
	var lastReloadNodes []mygo.E2EAccessNode
	for time.Now().Before(deadline) {
		lastReloadState = model.workspace.State()
		lastReloadNodes = mygo.E2EAccessibility(window)
		if lastReloadState.Busy && lastReloadState.Response != nil && lastReloadState.Response.QueryCatalogue != nil &&
			nativeContentHas(lastReloadNodes, "查询目录正在读取或已过期") &&
			!nativeContentHas(lastReloadNodes, "First ledger query") {
			hiddenDuringReload = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !hiddenDuringReload {
		responseView, responseSnapshot, hasCatalogue := "", "", false
		if lastReloadState.Response != nil {
			responseView = lastReloadState.Response.Request.View
			responseSnapshot = lastReloadState.Response.Request.SnapshotID
			hasCatalogue = lastReloadState.Response.QueryCatalogue != nil
		}
		t.Fatalf("query catalogue remained available during reload: busy=%v phase=%s response_view=%q catalogue=%t ledger_snapshot=%s response_snapshot=%s nodes=%s", lastReloadState.Busy, lastReloadState.Phase, responseView, hasCatalogue, lastReloadState.Ledger.SnapshotID, responseSnapshot, nativeNodeSummary(lastReloadNodes))
	}
	releaseOnce.Do(func() { close(releaseReload) })
	eventuallyNative(t, "updated first ledger query catalogue after reload", func() bool {
		state := model.workspace.State()
		nodes := mygo.E2EAccessibility(window)
		return !state.Busy && state.Response != nil && state.Response.Request.View == "query" &&
			state.Response.QueryCatalogue != nil && state.Response.QueryCatalogue.SnapshotID != firstSnapshot &&
			nativeContentHas(nodes, "Updated first ledger query") &&
			!nativeContentHas(nodes, "First ledger query") &&
			nativeRoleLabelExists(nodes, "AXButton", "填入草稿 · First only preset")
	})
}

func TestNativeQuerySavePanelIsOwnedAndCancellable(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	ledgerPath := filepath.Join(t.TempDir(), "cashflow.bean")
	ledger := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Income:Salary USD
2026-01-02 * "Salary"
  Assets:Cash 10 USD
  Income:Salary -10 USD
`
	if err := os.WriteFile(ledgerPath, []byte(ledger), 0600); err != nil {
		t.Fatal(err)
	}
	model, window := openNativeLedgerWindow(t, ledgerPath)
	performNativeAccessibilityEventually(t, window, "AXButton", "查询", "press", "")
	eventuallyNative(t, "native query workspace", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "query" && !state.Busy
	})
	const queryText = "SELECT 1 AS v LIMIT 1"
	window.Update(func() { model.queryDraft = queryText })
	eventuallyNative(t, "native query draft", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), queryText)
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "执行查询", "press", "")
	eventuallyNative(t, "successful native query result", func() bool {
		state := model.workspace.State()
		return state.QueryResponse != nil && state.QueryResponse.Context.Text == queryText && state.QueryResponse.Execution != nil && state.QueryResponse.Error == nil && !state.QueryBusy
	})
	eventuallyNative(t, "native query save button", func() bool {
		return nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXButton", "保存 CSV…")
	})

	otherModel := NewModel(true)
	var otherWindow *mygo.Window
	mygo.RunOnMain(func() {
		otherWindow = mygo.NewWindow(otherModel.WindowOptions())
		otherModel.SetWindow(otherWindow)
		otherWindow.OnClosed(otherModel.Close)
	})
	t.Cleanup(func() {
		otherWindow.Destroy()
		otherModel.Close()
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "保存 CSV…", "press", "")
	eventuallyNative(t, "visible query panel attached to owner", func() bool {
		return mygo.E2ESavePanelVisible() && mygo.E2EHasSheet(window)
	})
	if mygo.E2EHasSheet(otherWindow) {
		t.Fatal("native query save panel was attached to the other window")
	}
	if !mygo.E2ECancelSheet(window) {
		t.Fatal("native query save panel could not be cancelled")
	}
	eventuallyNative(t, "cancelled native query save", func() bool {
		var status string
		var active uint64
		mygo.RunOnMain(func() {
			status = model.status
			active = model.activeExportID
		})
		return status == "已取消查询结果保存。" && active == 0
	})
	if mygo.E2EHasSheet(window) || mygo.E2EHasSheet(otherWindow) || mygo.E2ESavePanelVisible() {
		t.Fatal("native query save panel remained open after cancellation")
	}
	entries, err := os.ReadDir(filepath.Dir(ledgerPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(ledgerPath) {
		t.Fatalf("cancelled native query save created unexpected files: %v", entries)
	}
}

func TestNativeQuerySavePanelClosesWithOwnerWindow(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to close a query Save Panel with its native owner")
	}
	ledgerPath := filepath.Join(t.TempDir(), "cashflow.bean")
	ledger := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Income:Salary USD
2026-01-02 * "Salary"
  Assets:Cash 10 USD
  Income:Salary -10 USD
`
	if err := os.WriteFile(ledgerPath, []byte(ledger), 0o600); err != nil {
		t.Fatal(err)
	}
	model, window := openNativeLedgerWindow(t, ledgerPath)
	performNativeAccessibilityEventually(t, window, "AXButton", "查询", "press", "")
	eventuallyNative(t, "native query workspace", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "query" && !state.Busy
	})
	const queryText = "SELECT 1 AS v LIMIT 1"
	window.Update(func() { model.queryDraft = queryText })
	eventuallyNative(t, "native query draft", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), queryText)
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "执行查询", "press", "")
	eventuallyNative(t, "captured native query result", func() bool {
		state := model.workspace.State()
		return state.QueryResponse != nil && state.QueryResponse.Context.Text == queryText && !state.QueryBusy
	})
	eventuallyNative(t, "native query save button", func() bool {
		return nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXButton", "保存 CSV…")
	})

	otherModel := NewModel(true)
	var otherWindow *mygo.Window
	mygo.RunOnMain(func() {
		otherWindow = mygo.NewWindow(otherModel.WindowOptions())
		otherModel.SetWindow(otherWindow)
		otherWindow.OnClosed(otherModel.Close)
	})
	t.Cleanup(func() {
		otherWindow.Destroy()
		otherModel.Close()
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "保存 CSV…", "press", "")
	eventuallyNative(t, "query Save Panel owns the frozen capture", func() bool {
		state := model.workspace.State()
		return mygo.E2ESavePanelVisible() && mygo.E2EHasSheet(window) && !state.QueryCaptureBusy && state.QueryCapture != nil
	})
	if mygo.E2EHasSheet(otherWindow) {
		t.Fatal("native query Save Panel was attached to the other window")
	}
	window.Close()
	eventuallyNative(t, "owner window close and query Save Panel dismissal", func() bool {
		return window.IsDestroyed() && !mygo.E2ESavePanelVisible()
	})
	if mygo.E2EHasSheet(otherWindow) {
		t.Fatal("closing the owner left a query Save Panel attached to the other window")
	}
	eventuallyNative(t, "closed query workspace releases response and capture", func() bool {
		select {
		case <-model.workspace.Done():
			state := model.workspace.State()
			return state.Phase == workspace.PhaseClosed && state.QueryResponse == nil && state.QueryCapture == nil && state.Response == nil
		default:
			return false
		}
	})
	entries, err := os.ReadDir(filepath.Dir(ledgerPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(ledgerPath) {
		t.Fatalf("closing the query Save Panel owner created unexpected files: %v", entries)
	}
}

func TestNativeQueryCaptureOwnerCloseDiscardsLateCapture(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to close a native window during query capture")
	}
	ledgerPath := filepath.Join(t.TempDir(), "cashflow.bean")
	ledger := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Income:Salary USD
2026-01-02 * "Salary"
  Assets:Cash 10 USD
  Income:Salary -10 USD
`
	if err := os.WriteFile(ledgerPath, []byte(ledger), 0o600); err != nil {
		t.Fatal(err)
	}
	releaseCapture := make(chan struct{})
	startedCapture := make(chan desktop.QueryExportRequest, 1)
	engine := &nativeQueryCaptureGateEngine{release: releaseCapture, started: startedCapture}
	model, window := openNativeLedgerWindowWithEngine(t, ledgerPath, "", engine)
	performNativeAccessibilityEventually(t, window, "AXButton", "查询", "press", "")
	eventuallyNative(t, "native query workspace", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "query" && !state.Busy
	})
	const queryText = "SELECT 1 AS v LIMIT 1"
	window.Update(func() { model.queryDraft = queryText })
	eventuallyNative(t, "native query draft", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), queryText)
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "执行查询", "press", "")
	eventuallyNative(t, "successful query before gated capture", func() bool {
		state := model.workspace.State()
		return state.QueryResponse != nil && state.QueryResponse.Context.Text == queryText && !state.QueryBusy
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "保存 CSV…", "press", "")
	var request desktop.QueryExportRequest
	select {
	case request = <-startedCapture:
	case <-time.After(5 * time.Second):
		t.Fatal("query capture did not enter the gated core session")
	}
	state := model.workspace.State()
	if request.SnapshotID != state.Ledger.SnapshotID || request.RequestID != state.QueryResponse.Context.RequestID || request.Fingerprint != state.QueryResponse.Context.Fingerprint || request.Format != "csv" {
		t.Fatalf("native save requested the wrong frozen query identity: request=%+v state=%+v", request, state)
	}
	eventuallyNative(t, "query capture remains pending before Save Panel", func() bool {
		state := model.workspace.State()
		return state.QueryCaptureBusy && state.QueryCapture == nil && !mygo.E2ESavePanelVisible()
	})

	otherModel := NewModel(true)
	var otherWindow *mygo.Window
	mygo.RunOnMain(func() {
		otherWindow = mygo.NewWindow(otherModel.WindowOptions())
		otherModel.SetWindow(otherWindow)
		otherWindow.OnClosed(otherModel.Close)
	})
	t.Cleanup(func() {
		otherWindow.Destroy()
		otherModel.Close()
	})
	window.Close()
	eventuallyNative(t, "owner window closes without opening a Save Panel", func() bool {
		return window.IsDestroyed() && !mygo.E2ESavePanelVisible()
	})
	if mygo.E2EHasSheet(otherWindow) {
		t.Fatal("closing the query capture owner left a sheet on another window")
	}
	select {
	case <-model.workspace.Done():
		t.Fatal("workspace completed before the in-flight query capture released its session call")
	default:
	}
	close(releaseCapture)
	eventuallyNative(t, "closed workspace discards late query capture", func() bool {
		select {
		case <-model.workspace.Done():
			state := model.workspace.State()
			return state.Phase == workspace.PhaseClosed && state.QueryResponse == nil && state.QueryCapture == nil && state.Response == nil
		default:
			return false
		}
	})
	if mygo.E2ESavePanelVisible() || mygo.E2EHasSheet(otherWindow) {
		t.Fatal("late capture completion opened or left a native Save Panel")
	}
	entries, err := os.ReadDir(filepath.Dir(ledgerPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(ledgerPath) {
		t.Fatalf("closing during query capture created unexpected files: %v", entries)
	}
}

func TestNativeQueryExecutionOwnerCloseWaitsForLateSessionCall(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to close a native window during query execution")
	}
	ledgerPath := filepath.Join(t.TempDir(), "cashflow.bean")
	ledger := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Income:Salary USD
2026-01-02 * "Salary"
  Assets:Cash 10 USD
  Income:Salary -10 USD
`
	if err := os.WriteFile(ledgerPath, []byte(ledger), 0o600); err != nil {
		t.Fatal(err)
	}
	releaseQuery := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseQuery) }) }
	startedQuery := make(chan desktop.QueryRequest, 1)
	engine := &nativeQueryExecutionGateEngine{release: releaseQuery, started: startedQuery}
	model, window := openNativeLedgerWindowWithEngine(t, ledgerPath, "", engine)
	t.Cleanup(release)
	performNativeAccessibilityEventually(t, window, "AXButton", "查询", "press", "")
	eventuallyNative(t, "native query workspace", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "query" && !state.Busy
	})
	const queryText = "SELECT 1 AS v LIMIT 1"
	window.Update(func() { model.queryDraft = queryText })
	eventuallyNative(t, "native query draft", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), queryText)
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "执行查询", "press", "")
	var request desktop.QueryRequest
	select {
	case request = <-startedQuery:
	case <-time.After(5 * time.Second):
		t.Fatal("query execution did not enter the gated desktop session")
	}
	state := model.workspace.State()
	if request.SnapshotID != state.Ledger.SnapshotID || request.Text != queryText || request.RequestID != state.QueryRequestID || !state.QueryBusy {
		t.Fatalf("native query execution started with the wrong request identity: request=%+v state=%+v", request, state)
	}
	window.Close()
	eventuallyNative(t, "owner window closes while query execution remains gated", func() bool {
		return window.IsDestroyed()
	})
	select {
	case <-model.workspace.Done():
		t.Fatal("workspace completed before the in-flight query session call released")
	default:
	}
	if mygo.E2ESavePanelVisible() {
		t.Fatal("closing during query execution opened a Save Panel")
	}
	release()
	eventuallyNative(t, "closed workspace drops the late query response", func() bool {
		select {
		case <-model.workspace.Done():
			state := model.workspace.State()
			return state.Phase == workspace.PhaseClosed && !state.QueryBusy && state.QueryResponse == nil &&
				state.QueryCapture == nil && state.Response == nil
		default:
			return false
		}
	})
	if mygo.E2ESavePanelVisible() {
		t.Fatal("late query completion opened a Save Panel after owner close")
	}
	entries, err := os.ReadDir(filepath.Dir(ledgerPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(ledgerPath) {
		t.Fatalf("closing during query execution created unexpected files: %v", entries)
	}
}

func TestNativeQueryRunningAndSavePanelOwnersCloseConcurrently(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to close concurrent query and Save Panel owners")
	}
	writeLedger := func(path string) {
		t.Helper()
		ledger := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Income:Salary USD
2026-01-02 * "Salary"
  Assets:Cash 10 USD
  Income:Salary -10 USD
`
		if err := os.WriteFile(path, []byte(ledger), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	prepareQuery := func(model *Model, window *mygo.Window, queryText string) {
		t.Helper()
		performNativeAccessibilityEventually(t, window, "AXButton", "查询", "press", "")
		eventuallyNative(t, "native query workspace", func() bool {
			state := model.workspace.State()
			return state.Response != nil && state.Response.Request.View == "query" && !state.Busy
		})
		window.Update(func() { model.queryDraft = queryText })
		eventuallyNative(t, "native query draft", func() bool {
			return nativeContentHas(mygo.E2EAccessibility(window), queryText)
		})
		performNativeAccessibilityEventually(t, window, "AXButton", "执行查询", "press", "")
	}

	queryLedgerPath := filepath.Join(t.TempDir(), "query-running.bean")
	writeLedger(queryLedgerPath)
	releaseQuery := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseQuery) }) }
	startedQuery := make(chan desktop.QueryRequest, 1)
	queryEngine := &nativeQueryExecutionGateEngine{release: releaseQuery, started: startedQuery}
	queryModel, queryWindow := openNativeLedgerWindowWithEngine(t, queryLedgerPath, "", queryEngine)
	t.Cleanup(release)
	const queryText = "SELECT 1 AS v LIMIT 1"
	prepareQuery(queryModel, queryWindow, queryText)
	select {
	case request := <-startedQuery:
		state := queryModel.workspace.State()
		if request.SnapshotID != state.Ledger.SnapshotID || request.Text != queryText || request.RequestID != state.QueryRequestID || !state.QueryBusy {
			t.Fatalf("gated query has the wrong request identity: request=%+v state=%+v", request, state)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("query execution did not enter the gated session")
	}

	saveLedgerPath := filepath.Join(t.TempDir(), "query-save.bean")
	writeLedger(saveLedgerPath)
	saveModel, saveWindow := openNativeLedgerWindow(t, saveLedgerPath)
	prepareQuery(saveModel, saveWindow, queryText)
	eventuallyNative(t, "successful query response for the Save Panel window", func() bool {
		state := saveModel.workspace.State()
		return state.QueryResponse != nil && state.QueryResponse.Context.Text == queryText && !state.QueryBusy
	})
	performNativeAccessibilityEventually(t, saveWindow, "AXButton", "保存 CSV…", "press", "")
	eventuallyNative(t, "Save Panel owns a frozen query capture while another window executes a query", func() bool {
		state := saveModel.workspace.State()
		return mygo.E2ESavePanelVisible() && mygo.E2EHasSheet(saveWindow) && !state.QueryCaptureBusy && state.QueryCapture != nil
	})
	if mygo.E2EHasSheet(queryWindow) {
		t.Fatal("query execution window unexpectedly owns the other window's Save Panel")
	}

	queryWindow.Close()
	saveWindow.Close()
	eventuallyNative(t, "both concurrent operation owners close and the native Save Panel disappears", func() bool {
		return queryWindow.IsDestroyed() && saveWindow.IsDestroyed() && !mygo.E2ESavePanelVisible()
	})
	if mygo.E2EHasSheet(queryWindow) || mygo.E2EHasSheet(saveWindow) {
		t.Fatal("concurrent owner close left a native sheet attached")
	}
	select {
	case <-queryModel.workspace.Done():
		t.Fatal("query workspace completed before the gated session call was released")
	default:
	}
	release()
	eventuallyNative(t, "both concurrent workspaces release responses and captures", func() bool {
		queryDone, saveDone := false, false
		select {
		case <-queryModel.workspace.Done():
			queryDone = true
		default:
		}
		select {
		case <-saveModel.workspace.Done():
			saveDone = true
		default:
		}
		if !queryDone || !saveDone {
			return false
		}
		queryState, saveState := queryModel.workspace.State(), saveModel.workspace.State()
		return queryState.Phase == workspace.PhaseClosed && queryState.QueryResponse == nil && queryState.QueryCapture == nil && queryState.Response == nil &&
			saveState.Phase == workspace.PhaseClosed && saveState.QueryResponse == nil && saveState.QueryCapture == nil && saveState.Response == nil
	})
	if mygo.E2ESavePanelVisible() {
		t.Fatal("late query execution reopened a Save Panel after both owners closed")
	}
	for _, path := range []string{queryLedgerPath, saveLedgerPath} {
		entries, err := os.ReadDir(filepath.Dir(path))
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].Name() != filepath.Base(path) {
			t.Fatalf("concurrent query/panel close created unexpected files next to %s: %v", filepath.Base(path), entries)
		}
	}
}

func TestNativeTextScaleMenuReflowsAndPersistsPerLedger(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to exercise native text-size commands")
	}
	contents, err := os.ReadFile(nativeFixturePath(t, "native/account-detail.bean"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	ledgerPath := filepath.Join(root, "large-text.bean")
	if err := os.WriteFile(ledgerPath, contents, 0600); err != nil {
		t.Fatal(err)
	}
	store := preferences.NewStore(filepath.Join(root, "preferences-v1.json"))
	model, window := newNativeWelcomeWindow(t)
	model.SetPreferences(store)
	model.OpenLedgerPath(ledgerPath)
	window.SetSize(900, 600)
	if width, height := window.Size(); width != 900 || height != 600 {
		t.Fatalf("native large-text window size = %dx%d, want 900x600", width, height)
	}
	eventuallyNative(t, "text-scale ledger becomes ready", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseReady && !state.Busy && state.ActivePath == ledgerPath
	})
	mygo.App.SetMenu(NewApplicationMenu(func(candidate *mygo.Window) *Model {
		if candidate == window {
			return model
		}
		return nil
	}, nil))
	t.Cleanup(func() { mygo.App.SetMenu(nil) })
	performNativeAccessibilityEventually(t, window, "AXButton", "资产负债表", "press", "")
	eventuallyNative(t, "text-scale balance sheet", func() bool {
		state := model.workspace.State()
		return !state.Busy && state.Response != nil && state.Response.Request.View == "balance_sheet"
	})
	previousAppearance := mygo.Theme.Source()
	t.Cleanup(func() { mygo.Theme.SetSource(previousAppearance) })
	mygo.Theme.SetSource(mygo.ThemeLight)
	eventuallyNative(t, "light appearance keeps native balance-sheet content accessible", func() bool {
		return !mygo.Theme.IsDark() && nativeContentHas(mygo.E2EAccessibility(window), "资产负债表")
	})
	mygo.Theme.SetSource(mygo.ThemeDark)
	eventuallyNative(t, "dark appearance keeps native balance-sheet content accessible", func() bool {
		return mygo.Theme.IsDark() && nativeContentHas(mygo.E2EAccessibility(window), "资产负债表")
	})
	mygo.Theme.SetSource(previousAppearance)
	eventuallyNative(t, "native appearance returns to its original source", func() bool {
		return mygo.Theme.Source() == previousAppearance && nativeContentHas(mygo.E2EAccessibility(window), "资产负债表")
	})
	const measuredLabel = "分类汇总 · 不受搜索或折叠影响"
	initial := waitForStableNativeAccessibilityNode(t, window, "AXStaticText", measuredLabel, 3).Frame
	window.Focus()
	if err := mygo.E2EActivateMenu("显示", "放大文字"); err != nil {
		t.Fatalf("activate Increase Text Size: %v", err)
	}
	large := initial
	eventuallyNative(t, "125 percent text scale is rendered", func() bool {
		var scale float32
		mygo.RunOnMain(func() { scale = model.textScale })
		if scale != 1.25 {
			return false
		}
		for _, node := range mygo.E2EAccessibility(window) {
			if node.Role == "AXStaticText" && node.Label == measuredLabel {
				large = node.Frame
				return node.Frame[3] > initial[3]
			}
		}
		return false
	})
	if !mygo.E2EPerformKeyEquivalent("+", false) {
		t.Fatal("native Cmd+Plus text-size shortcut was not handled")
	}
	maximum := large
	eventuallyNative(t, "150 percent text scale is rendered", func() bool {
		var scale float32
		mygo.RunOnMain(func() { scale = model.textScale })
		if scale != 1.5 {
			return false
		}
		for _, node := range mygo.E2EAccessibility(window) {
			if node.Role == "AXStaticText" && node.Label == measuredLabel {
				maximum = node.Frame
				return node.Frame[3] > large[3]
			}
		}
		return false
	})
	eventuallyNative(t, "per-ledger text scale is saved", func() bool {
		item, err := store.ForLedger(ledgerPath)
		return err == nil && item.TextScale == 1.5
	})
	if err := mygo.E2EActivateMenu("编辑", "查找"); err != nil {
		t.Fatalf("focus account search at 150 percent text: %v", err)
	}
	eventuallyNative(t, "account search remains reachable at minimum window size and large text", func() bool {
		for _, node := range mygo.E2EAccessibility(window) {
			if node.Role == "AXTextField" && node.Label == "搜索账户" && node.Focused {
				return true
			}
		}
		return false
	})
	if !mygo.E2EPerformKeyEquivalent("-", false) {
		t.Fatal("native Cmd+- text-size shortcut was not handled")
	}
	eventuallyNative(t, "decrease text size shortcut returns to 125 percent", func() bool {
		var scale float32
		mygo.RunOnMain(func() { scale = model.textScale })
		if scale != 1.25 {
			return false
		}
		for _, node := range mygo.E2EAccessibility(window) {
			if node.Role == "AXStaticText" && node.Label == measuredLabel {
				return node.Frame[3] < maximum[3] && node.Frame[3] > initial[3]
			}
		}
		return false
	})
	if err := mygo.E2EActivateMenu("显示", "恢复默认文字大小"); err != nil {
		t.Fatalf("activate Reset Text Size: %v", err)
	}
	eventuallyNative(t, "default text size is restored", func() bool {
		var scale float32
		mygo.RunOnMain(func() { scale = model.textScale })
		if scale != 1 {
			return false
		}
		for _, node := range mygo.E2EAccessibility(window) {
			if node.Role == "AXStaticText" && node.Label == measuredLabel {
				return node.Frame[3] == initial[3] && node.Frame[3] < maximum[3]
			}
		}
		return false
	})
	if err := mygo.E2EActivateMenu("显示", "放大文字"); err != nil {
		t.Fatalf("activate Increase Text Size before reset shortcut: %v", err)
	}
	eventuallyNative(t, "menu increase makes Cmd+0 reset observable", func() bool {
		var scale float32
		mygo.RunOnMain(func() { scale = model.textScale })
		return scale == 1.25
	})
	if !mygo.E2EPerformKeyEquivalent("0", false) {
		t.Fatal("native Cmd+0 text-size shortcut was not handled")
	}
	eventuallyNative(t, "reset text size shortcut restores the default scale", func() bool {
		var scale float32
		mygo.RunOnMain(func() { scale = model.textScale })
		if scale != 1 {
			return false
		}
		for _, node := range mygo.E2EAccessibility(window) {
			if node.Role == "AXStaticText" && node.Label == measuredLabel {
				return node.Frame[3] == initial[3]
			}
		}
		return false
	})
	eventuallyNative(t, "default per-ledger text size is saved", func() bool {
		item, err := store.ForLedger(ledgerPath)
		return err == nil && item.TextScale == 1
	})
	reopened, reopenedWindow := newNativeWelcomeWindow(t)
	reopened.SetPreferences(store)
	reopened.OpenLedgerPath(ledgerPath)
	eventuallyNative(t, "reopened ledger restores its persisted text scale", func() bool {
		var scale float32
		mygo.RunOnMain(func() { scale = reopened.textScale })
		state := reopened.workspace.State()
		return scale == 1 && state.Phase == workspace.PhaseReady && !state.Busy && state.ActivePath == ledgerPath &&
			nativeContentHas(mygo.E2EAccessibility(reopenedWindow), "财务概览")
	})
}

func TestNativeAppearanceRendersDistinctLightAndDarkPixels(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to compare native appearance pixels")
	}
	model, window := newNativeWelcomeWindow(t)
	window.SetSize(900, 600)
	model.OpenLedgerPath(nativeFixturePath(t, "native/cashflow.bean"))
	eventuallyNative(t, "appearance fixture becomes ready", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseReady && !state.Busy && state.Response != nil && state.Response.Request.View == "dashboard"
	})

	previousAppearance := mygo.Theme.Source()
	t.Cleanup(func() { mygo.Theme.SetSource(previousAppearance) })
	mygo.Theme.SetSource(mygo.ThemeLight)
	var lightLuminance float64
	var lightPixels []color.RGBA
	eventuallyNative(t, "light appearance is painted into Content pixels", func() bool {
		if mygo.Theme.IsDark() {
			return false
		}
		pixels, luminance, err := nativeContentPixelSummary(window)
		if err != nil || len(pixels) == 0 || luminance < 150 {
			return false
		}
		lightPixels, lightLuminance = pixels, luminance
		return true
	})

	mygo.Theme.SetSource(mygo.ThemeDark)
	var darkPixels []color.RGBA
	var darkLuminance float64
	eventuallyNative(t, "dark appearance is painted into Content pixels", func() bool {
		if !mygo.Theme.IsDark() {
			return false
		}
		pixels, luminance, err := nativeContentPixelSummary(window)
		if err != nil || len(pixels) == 0 || luminance >= lightLuminance-40 {
			return false
		}
		darkPixels, darkLuminance = pixels, luminance
		return true
	})

	if len(lightPixels) != len(darkPixels) {
		t.Fatalf("appearance captures sampled different pixel counts: light=%d dark=%d", len(lightPixels), len(darkPixels))
	}
	changed := 0
	for i, light := range lightPixels {
		dark := darkPixels[i]
		distance := nativeAbsoluteInt(int(light.R)-int(dark.R)) + nativeAbsoluteInt(int(light.G)-int(dark.G)) + nativeAbsoluteInt(int(light.B)-int(dark.B))
		if distance >= 60 {
			changed++
		}
	}
	if lightLuminance < darkLuminance+40 || changed < len(lightPixels)/20 {
		t.Fatalf("light/dark appearance did not produce a broad pixel change: mean luminance %.1f -> %.1f; changed samples %d/%d", lightLuminance, darkLuminance, changed, len(lightPixels))
	}
	t.Logf("900x600 Content appearance samples: mean luminance %.1f -> %.1f; changed pixels %d/%d", lightLuminance, darkLuminance, changed, len(lightPixels))
}

func TestNativePreferencesBackupMenuPanelCancelAndSave(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to exercise the native preferences-backup panel")
	}
	model, window := openNativeCashflowWindow(t)
	storePath := filepath.Join(t.TempDir(), "preferences-v1.json")
	store := preferences.NewStore(storePath)
	model.SetPreferences(store)
	otherModel := NewModel(true)
	var otherWindow *mygo.Window
	mygo.RunOnMain(func() {
		otherWindow = mygo.NewWindow(otherModel.WindowOptions())
		otherModel.SetWindow(otherWindow)
		otherWindow.OnClosed(otherModel.Close)
	})
	t.Cleanup(func() {
		otherWindow.Destroy()
		otherModel.Close()
	})
	mygo.App.SetMenu(NewApplicationMenu(func(candidate *mygo.Window) *Model {
		if candidate == window {
			return model
		}
		return nil
	}, nil))
	window.Focus()
	if err := mygo.E2EActivateMenu("文件", "导出偏好备份…"); err != nil {
		t.Fatalf("activate native preferences-backup menu item: %v", err)
	}
	eventuallyNative(t, "native preferences Save Panel attached to its owner", func() bool {
		return mygo.E2ESavePanelVisible() && mygo.E2EHasSheet(window) && !mygo.E2EHasSheet(otherWindow)
	})
	if mygo.E2ECancelSheet(otherWindow) {
		t.Fatal("preferences-backup Save Panel was attached to the other window")
	}
	if !mygo.E2ECancelSheet(window) {
		t.Fatal("preferences-backup Save Panel was not attached to its owning window")
	}
	eventuallyNative(t, "cancelled preferences-backup Save Panel", func() bool {
		var status string
		mygo.RunOnMain(func() { status = model.status })
		return status == "已取消偏好备份导出。" && !mygo.E2EHasSheet(window) && !mygo.E2ESavePanelVisible()
	})
	if _, err := os.Stat(storePath); !os.IsNotExist(err) {
		t.Fatalf("cancelling the preferences backup changed the live preference store: %v", err)
	}
	target := filepath.Join(t.TempDir(), "preferences-backup.json")
	mygo.RunOnMain(func() {
		model.preferenceBackupSaveDialog = func(context.Context, mygo.SaveDialogOptions) (string, error) {
			return target, nil
		}
	})
	if err := mygo.E2EActivateMenu("文件", "导出偏好备份…"); err != nil {
		t.Fatalf("activate preferences backup a second time: %v", err)
	}
	eventuallyNative(t, "successfully saved native preferences backup", func() bool {
		var status string
		mygo.RunOnMain(func() { status = model.status })
		return status == "偏好备份已导出：preferences-backup.json"
	})
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read saved preferences backup: %v", err)
	}
	var exported preferences.Document
	if err := json.Unmarshal(data, &exported); err != nil || exported.SchemaVersion != preferences.SchemaVersion {
		t.Fatalf("saved preferences backup is invalid: schema=%d err=%v", exported.SchemaVersion, err)
	}
}

func TestNativeInitialInvalidLedgerShowsDiagnosticsWithoutPreviousSnapshot(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open real macOS Content windows")
	}
	previous, previousWindow := openNativeCashflowWindow(t)
	previousState := previous.workspace.State()
	if previousState.Response == nil || !previousState.Response.State.Valid || previousState.Response.Summary.TransactionsCount != 2 {
		t.Fatalf("control ledger did not load before opening the invalid ledger: %+v", previousState)
	}
	fixture, err := os.ReadFile(nativeFixturePath(t, "v3-parity/err-syntax.bean"))
	if err != nil {
		t.Fatalf("read public invalid-ledger fixture: %v", err)
	}
	invalidPath := filepath.Join(t.TempDir(), "invalid.bean")
	if err := os.WriteFile(invalidPath, fixture, 0600); err != nil {
		t.Fatal(err)
	}
	invalid, invalidWindow := newNativeWelcomeWindow(t)
	invalid.OpenLedgerPath(invalidPath)
	eventuallyNative(t, "first load of an invalid ledger without a published snapshot", func() bool {
		state := invalid.workspace.State()
		return state.Phase == workspace.PhaseInvalid && !state.Busy && state.ActivePath == invalidPath &&
			!state.Ledger.HasSnapshot && state.Ledger.SnapshotID == "" && state.Response != nil &&
			!state.Response.State.Valid && state.Response.Summary.ErrorCount > 0
	})
	state := invalid.workspace.State()
	diagnostic := state.Response.Summary.Diagnostics[0]
	if diagnostic.Code != "E-PARSE-STRING" || diagnostic.Source.Line != 1 || diagnostic.Source.Column != 40 {
		t.Fatalf("initial syntax diagnostic lost its exact source location: %+v", diagnostic)
	}
	performNativeAccessibilityEventually(t, invalidWindow, "AXButton", "诊断", "press", "")
	eventuallyNative(t, "initial invalid-ledger diagnostic details in the native window", func() bool {
		nodes := mygo.E2EAccessibility(invalidWindow)
		return nativeContentHas(nodes, "当前账本无有效快照") && nativeContentHas(nodes, "E-PARSE-STRING") &&
			nativeContentHas(nodes, "字符串未结束") && nativeContentHas(nodes, "invalid.bean:1:40")
	})
	if nodes := mygo.E2EAccessibility(invalidWindow); nativeContentHas(nodes, "700 USD") {
		t.Fatalf("initial invalid-ledger window rendered values from another ledger: %s", nativeNodeSummary(nodes))
	}
	unchanged := previous.workspace.State()
	if unchanged.ActivePath != previousState.ActivePath || unchanged.Epoch != previousState.Epoch ||
		unchanged.Ledger.SnapshotID != previousState.Ledger.SnapshotID || unchanged.Response.Summary.TransactionsCount != 2 ||
		!nativeContentHas(mygo.E2EAccessibility(previousWindow), "700 USD") {
		t.Fatalf("opening an invalid ledger changed the populated control window: before=%+v after=%+v", previousState, unchanged)
	}
}

func TestNativeInvalidDiagnosticsRecoverInTheSameWindow(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	path := filepath.Join(t.TempDir(), "main.bean")
	valid := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Equity:Opening USD
2026-01-02 * "Opening"
  Assets:Cash 10 USD
  Equity:Opening -10 USD
`
	if err := os.WriteFile(path, []byte(valid), 0600); err != nil {
		t.Fatal(err)
	}
	model, window := openNativeLedgerWindow(t, path)
	initial := model.workspace.State()
	if initial.Response == nil || !initial.Response.State.Valid || initial.Response.Summary.TransactionsCount != 1 {
		t.Fatalf("initial native ledger state is invalid: %+v", initial)
	}
	if err := os.WriteFile(path, []byte("this is not valid Beancount\n"), 0600); err != nil {
		t.Fatal(err)
	}
	eventuallyNative(t, "stale workspace with current diagnostics", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseStale && !state.Busy && state.Response != nil && !state.Response.State.Valid && state.Response.Summary.ErrorCount > 0
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "诊断", "press", "")
	eventuallyNative(t, "stale diagnostic source location in native content", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "账本文件已有变化") && nativeContentHas(nodes, "main.bean:1:1")
	})
	diagnostic := model.workspace.State().Response.Summary.Diagnostics[0]
	copyButton := fmt.Sprintf("复制位置 · %s:%d:%d", diagnostic.Source.Path, diagnostic.Source.Line, diagnostic.Source.Column)
	wantClipboard := fmt.Sprintf("%s:%d:%d", diagnostic.Source.Path, diagnostic.Source.Line, diagnostic.Source.Column)
	preserveNativePasteboard(t)
	mygo.Clipboard.WriteText("previous clipboard value")
	if got := mygo.Clipboard.ReadText(); got != "previous clipboard value" {
		t.Fatalf("could not establish the native clipboard sentinel: %q", got)
	}
	window.Focus()
	eventuallyNative(t, "diagnostic window becomes key before copying", func() bool { return window.IsFocused() })
	copyNode := waitForStableNativeAccessibilityNode(t, window, "AXButton", copyButton, 8)
	pressFound, pressAllowed, pressResult := mygo.E2EAccessibilityPress(window, copyButton)
	if !pressFound || !pressAllowed || !pressResult {
		t.Fatalf("native accessibility press failed for the stable source action ID %d: found=%t allowed=%t result=%t", copyNode.ID, pressFound, pressAllowed, pressResult)
	}
	copyDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(copyDeadline) && mygo.Clipboard.ReadText() != wantClipboard {
		time.Sleep(20 * time.Millisecond)
	}
	if got := mygo.Clipboard.ReadText(); got != wantClipboard {
		var status string
		mygo.RunOnMain(func() { status = model.status })
		nodes := mygo.E2EAccessibility(window)
		t.Fatalf("native source copy did not update the system clipboard: got=%q want=%q status=%q focused=%t node=%t node_id=%d", got, wantClipboard, status,
			window.IsFocused(), nativeRoleLabelExists(nodes, "AXButton", copyButton), copyNode.ID)
	}
	repaired := valid + `2026-01-03 * "Deposit"
  Assets:Cash 20 USD
  Equity:Opening -20 USD
`
	if err := os.WriteFile(path, []byte(repaired), 0600); err != nil {
		t.Fatal(err)
	}
	performNativeAccessibilityEventually(t, window, "AXButton", "重新读取", "press", "")
	eventuallyNative(t, "same-window repaired snapshot", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseReady && !state.Busy && !state.Ledger.Reloading && state.Response != nil && state.Response.State.Valid && state.Response.Summary.TransactionsCount == 2
	})
	recovered := model.workspace.State()
	if recovered.ActivePath != initial.ActivePath || recovered.Epoch != initial.Epoch {
		t.Fatalf("repair replaced the current window workspace: before=%q/%d after=%q/%d", initial.ActivePath, initial.Epoch, recovered.ActivePath, recovered.Epoch)
	}
	performNativeAccessibilityEventually(t, window, "AXButton", "财务概览", "press", "")
	eventuallyNative(t, "repaired dashboard in the native window", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "dashboard" && !state.Busy && nativeContentHas(mygo.E2EAccessibility(window), "30 USD")
	})
}

func TestNativeExternalFileWatchRecoversWithoutManualReload(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	path := filepath.Join(t.TempDir(), "main.bean")
	valid := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Equity:Opening USD
2026-01-02 * "Opening"
  Assets:Cash 10 USD
  Equity:Opening -10 USD
`
	if err := os.WriteFile(path, []byte(valid), 0600); err != nil {
		t.Fatal(err)
	}
	model, window := openNativeLedgerWindow(t, path)
	initial := model.workspace.State()
	if initial.Response == nil || !initial.Response.State.Valid || initial.Response.Summary.TransactionsCount != 1 {
		t.Fatalf("initial native ledger state is invalid: %+v", initial)
	}

	if err := os.WriteFile(path, []byte("this is not valid Beancount\n"), 0600); err != nil {
		t.Fatal(err)
	}
	eventuallyNative(t, "external invalid edit while retaining the published snapshot", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseStale && !state.Busy && state.Response != nil &&
			!state.Response.State.Valid && state.Response.Summary.ErrorCount > 0
	})
	stale := model.workspace.State()
	if stale.Ledger.SnapshotID != initial.Ledger.SnapshotID || stale.Response.Summary.TransactionsCount != 1 {
		t.Fatalf("invalid external edit discarded the last valid snapshot: before=%+v after=%+v", initial.Ledger, stale)
	}

	repaired := valid + `2026-01-03 * "Deposit"
  Assets:Cash 20 USD
  Equity:Opening -20 USD
`
	if err := os.WriteFile(path, []byte(repaired), 0600); err != nil {
		t.Fatal(err)
	}
	eventuallyNative(t, "automatic watcher recovery and refreshed dashboard", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseReady && !state.Busy && !state.Ledger.Reloading &&
			state.Ledger.SnapshotID != initial.Ledger.SnapshotID && state.Response != nil &&
			state.Response.State.Valid && state.Response.Request.View == "dashboard" &&
			state.Response.Summary.TransactionsCount == 2 && nativeContentHas(mygo.E2EAccessibility(window), "30 USD")
	})
	recovered := model.workspace.State()
	if recovered.ActivePath != initial.ActivePath || recovered.Epoch != initial.Epoch {
		t.Fatalf("automatic file recovery replaced the current window workspace: before=%q/%d after=%q/%d", initial.ActivePath, initial.Epoch, recovered.ActivePath, recovered.Epoch)
	}
}

func TestNativeRepeatedInvalidUpdatesReplaceDiagnosticsAndRecover(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	path := filepath.Join(t.TempDir(), "main.bean")
	valid := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Equity:Opening USD
2026-01-02 * "Opening"
  Assets:Cash 10 USD
  Equity:Opening -10 USD
`
	if err := os.WriteFile(path, []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	model, window := openNativeLedgerWindow(t, path)
	initial := model.workspace.State()
	if initial.Response == nil || !initial.Response.State.Valid || initial.Response.Summary.TransactionsCount != 1 {
		t.Fatalf("initial native ledger state is invalid: %+v", initial)
	}
	diagnosticCodes := func(state workspace.State) string {
		if state.Response == nil {
			return ""
		}
		codes := make([]string, 0, len(state.Response.Summary.Diagnostics))
		for _, item := range state.Response.Summary.Diagnostics {
			codes = append(codes, item.Code)
		}
		return strings.Join(codes, ",")
	}
	if err := os.WriteFile(path, []byte("2026-01-01 * \"unterminated\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	eventuallyNative(t, "first invalid update while retaining the valid projection", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseStale && !state.Busy && state.Response != nil && !state.Response.State.Valid &&
			state.Ledger.SnapshotID == initial.Ledger.SnapshotID && state.Response.Summary.TransactionsCount == 1 &&
			strings.Contains(diagnosticCodes(state), "E-PARSE-STRING")
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "诊断", "press", "")
	eventuallyNative(t, "first syntax diagnostic in the native page", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "E-PARSE-STRING") && nativeContentHas(nodes, "main.bean:1:1")
	})

	unbalanced, err := os.ReadFile(nativeFixturePath(t, "v3-parity/err-unbalanced.bean"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, unbalanced, 0o600); err != nil {
		t.Fatal(err)
	}
	eventuallyNative(t, "second invalid update to replace the first diagnostic", func() bool {
		state := model.workspace.State()
		codes := diagnosticCodes(state)
		return state.Phase == workspace.PhaseStale && !state.Busy && state.Response != nil && !state.Response.State.Valid &&
			state.Ledger.SnapshotID == initial.Ledger.SnapshotID && state.Response.Summary.TransactionsCount == 1 &&
			strings.Contains(codes, "E-EVAL-UNBALANCED") && !strings.Contains(codes, "E-PARSE-STRING") &&
			nativeContentHas(mygo.E2EAccessibility(window), "E-EVAL-UNBALANCED")
	})
	if state := model.workspace.State(); state.Ledger.SnapshotID != initial.Ledger.SnapshotID || state.Response == nil || state.Response.Summary.TransactionsCount != 1 {
		t.Fatalf("two invalid edits discarded the last valid projection: initial=%s transactions=1 current=%s transactions=%d", initial.Ledger.SnapshotID, state.Ledger.SnapshotID, state.Response.Summary.TransactionsCount)
	}

	repaired := valid + `2026-01-03 * "Deposit"
  Assets:Cash 20 USD
  Equity:Opening -20 USD
`
	if err := os.WriteFile(path, []byte(repaired), 0o600); err != nil {
		t.Fatal(err)
	}
	eventuallyNative(t, "automatic recovery after the second invalid update", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseReady && !state.Busy && !state.Ledger.Reloading && state.Response != nil &&
			state.Response.State.Valid && state.Ledger.SnapshotID != initial.Ledger.SnapshotID &&
			state.Response.State.SnapshotID == state.Ledger.SnapshotID && state.Response.Summary.TransactionsCount == 2 &&
			nativeContentHas(mygo.E2EAccessibility(window), "没有解析诊断")
	})
	recovered := model.workspace.State()
	if recovered.ActivePath != initial.ActivePath || recovered.Epoch != initial.Epoch {
		t.Fatalf("automatic repair replaced the current window workspace: before=%q/%d after=%q/%d", initial.ActivePath, initial.Epoch, recovered.ActivePath, recovered.Epoch)
	}
	performNativeAccessibilityEventually(t, window, "AXButton", "财务概览", "press", "")
	eventuallyNative(t, "repaired dashboard in the native window", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "dashboard" && !state.Busy && nativeContentHas(mygo.E2EAccessibility(window), "30 USD")
	})
}

func TestNativeIncludedFileWatchUpdatesWithoutManualReload(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "main.bean")
	includePath := filepath.Join(dir, "activity.bean")
	entry := `option "operating_currency" "USD"
include "activity.bean"
`
	activity := `2026-01-01 open Assets:Cash USD
2026-01-01 open Equity:Opening USD
2026-01-02 * "Opening"
  Assets:Cash 10 USD
  Equity:Opening -10 USD
`
	if err := os.WriteFile(path, []byte(entry), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(includePath, []byte(activity), 0600); err != nil {
		t.Fatal(err)
	}
	model, window := openNativeLedgerWindow(t, path)
	initial := model.workspace.State()
	if initial.Response == nil || !initial.Response.State.Valid || initial.Response.Summary.TransactionsCount != 1 {
		t.Fatalf("initial included ledger state is invalid: %+v", initial)
	}

	replacement := activity + `2026-01-03 * "Deposit"
  Assets:Cash 20 USD
  Equity:Opening -20 USD
`
	tmp := filepath.Join(dir, "activity.bean.tmp")
	if err := os.WriteFile(tmp, []byte(replacement), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, includePath); err != nil {
		t.Fatal(err)
	}
	eventuallyNative(t, "automatic included-file reload and refreshed dashboard", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseReady && !state.Busy && !state.Ledger.Reloading &&
			state.Ledger.SnapshotID != initial.Ledger.SnapshotID && state.Response != nil &&
			state.Response.State.Valid && state.Response.Request.View == "dashboard" &&
			state.Response.Summary.TransactionsCount == 2 && nativeContentHas(mygo.E2EAccessibility(window), "30 USD")
	})
	recovered := model.workspace.State()
	if recovered.ActivePath != initial.ActivePath || recovered.Epoch != initial.Epoch {
		t.Fatalf("included-file change replaced the current window workspace: before=%q/%d after=%q/%d", initial.ActivePath, initial.Epoch, recovered.ActivePath, recovered.Epoch)
	}
}

func TestNativeIncludedFileRemovalRepairAndReplacement(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to exercise include changes in a real macOS window")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "main.bean")
	includePath := filepath.Join(dir, "activity.bean")
	replacementPath := filepath.Join(dir, "replacement.bean")
	entry := `option "operating_currency" "USD"
include "activity.bean"
`
	activity := `2026-01-01 open Assets:Cash USD
2026-01-01 open Equity:Opening USD
2026-01-02 * "Opening"
  Assets:Cash 10 USD
  Equity:Opening -10 USD
`
	if err := os.WriteFile(path, []byte(entry), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(includePath, []byte(activity), 0o600); err != nil {
		t.Fatal(err)
	}
	model, window := openNativeLedgerWindow(t, path)
	initial := model.workspace.State()
	if initial.Response == nil || !initial.Response.State.Valid || initial.Response.Summary.TransactionsCount != 1 {
		t.Fatalf("initial included ledger state is invalid: %+v", initial)
	}
	if err := os.Remove(includePath); err != nil {
		t.Fatal(err)
	}
	eventuallyNative(t, "removed include to publish a current read diagnostic while retaining the last good snapshot", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseStale && !state.Busy && state.Response != nil && !state.Response.State.Valid &&
			state.Ledger.SnapshotID == initial.Ledger.SnapshotID && state.Response.Summary.TransactionsCount == 1 &&
			state.Response.Summary.ErrorCount > 0 && nativeContentHas(mygo.E2EAccessibility(window), "无法读取 include 文件")
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "诊断", "press", "")
	var includeDiagnostic string
	var includeNodes []mygo.E2EAccessNode
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		state := model.workspace.State()
		if state.Response != nil && len(state.Response.Summary.Diagnostics) > 0 {
			diagnostic := state.Response.Summary.Diagnostics[0]
			includeDiagnostic = fmt.Sprintf("%s %s:%d:%d %s", diagnostic.Code, diagnostic.Source.Path, diagnostic.Source.Line, diagnostic.Source.Column, diagnostic.Message)
		}
		includeNodes = mygo.E2EAccessibility(window)
		if nativeContentHas(includeNodes, "E-INCLUDE-READ") && nativeContentHas(includeNodes, "main.bean:2:9") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !nativeContentHas(includeNodes, "E-INCLUDE-READ") || !nativeContentHas(includeNodes, "main.bean:2:9") {
		t.Fatalf("missing-include diagnostic does not identify its parent directive: %s, AX=%s", includeDiagnostic, nativeNodeSummary(includeNodes))
	}

	repairedInclude := activity + `2026-01-03 * "Deposit"
  Assets:Cash 20 USD
  Equity:Opening -20 USD
`
	if err := os.WriteFile(includePath, []byte(repairedInclude), 0o600); err != nil {
		t.Fatal(err)
	}
	eventuallyNative(t, "recreated include to restore the valid ledger automatically", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseReady && !state.Busy && !state.Ledger.Reloading && state.Response != nil &&
			state.Response.State.Valid && state.Ledger.SnapshotID != initial.Ledger.SnapshotID &&
			state.Response.State.SnapshotID == state.Ledger.SnapshotID && state.Response.Summary.TransactionsCount == 2 &&
			nativeContentHas(mygo.E2EAccessibility(window), "没有解析诊断")
	})
	recovered := model.workspace.State()
	if recovered.ActivePath != initial.ActivePath || recovered.Epoch != initial.Epoch {
		t.Fatalf("recreating the include replaced the current workspace: before=%q/%d after=%q/%d", initial.ActivePath, initial.Epoch, recovered.ActivePath, recovered.Epoch)
	}
	performNativeAccessibilityEventually(t, window, "AXButton", "财务概览", "press", "")
	eventuallyNative(t, "repaired included projection", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "dashboard" && !state.Busy && nativeContentHas(mygo.E2EAccessibility(window), "30 USD")
	})

	replacement := activity + `2026-01-03 * "Replacement"
  Assets:Cash 40 USD
  Equity:Opening -40 USD
`
	if err := os.WriteFile(replacementPath, []byte(replacement), 0o600); err != nil {
		t.Fatal(err)
	}
	switchedEntry := `option "operating_currency" "USD"
include "replacement.bean"
`
	if err := os.WriteFile(path, []byte(switchedEntry), 0o600); err != nil {
		t.Fatal(err)
	}
	previousSnapshot := recovered.Ledger.SnapshotID
	eventuallyNative(t, "switch to the replacement include and publish only its latest projection", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseReady && !state.Busy && !state.Ledger.Reloading && state.Response != nil &&
			state.Response.State.Valid && state.Ledger.SnapshotID != previousSnapshot &&
			state.Response.State.SnapshotID == state.Ledger.SnapshotID && state.Response.Summary.TransactionsCount == 2 &&
			state.Response.Request.View == "dashboard" && nativeContentHas(mygo.E2EAccessibility(window), "50 USD")
	})
	final := model.workspace.State()
	if final.ActivePath != initial.ActivePath || final.Epoch != initial.Epoch {
		t.Fatalf("changing the include target replaced the current workspace: before=%q/%d after=%q/%d", initial.ActivePath, initial.Epoch, final.ActivePath, final.Epoch)
	}
}

func TestNativeFinderOpenUsesWelcomeWindowAndPreservesPopulatedWindow(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open real macOS Content windows")
	}
	firstPath := filepath.Join(t.TempDir(), "first.bean")
	secondPath := filepath.Join(t.TempDir(), "second.bean")
	writeLedger := func(path, narration string, amount int) {
		t.Helper()
		ledger := fmt.Sprintf(`option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Equity:Opening USD
2026-01-02 * %q
  Assets:Cash %d USD
  Equity:Opening -%d USD
`, narration, amount, amount)
		if err := os.WriteFile(path, []byte(ledger), 0600); err != nil {
			t.Fatal(err)
		}
	}
	writeLedger(firstPath, "First", 10)
	writeLedger(secondPath, "Second", 20)

	welcome, welcomeWindow := newNativeWelcomeWindow(t)
	active := welcome
	var routed []*Model
	createWindow := func() *Model {
		model, _ := newNativeWelcomeWindow(t)
		routed = append(routed, model)
		return model
	}
	off := mygo.App.OnOpenFile(func(path string) {
		active = RouteOpenDocument(path, active, createWindow)
	})
	defer off()

	mygo.E2EOpenFiles(firstPath)
	eventuallyNative(t, "Finder-opened ledger in the existing welcome window", func() bool {
		state := welcome.workspace.State()
		return active == welcome && state.ActivePath == firstPath && state.Phase == workspace.PhaseReady &&
			!state.Busy && state.Response != nil && state.Response.Summary.TransactionsCount == 1
	})
	firstState := welcome.workspace.State()
	if len(mygo.E2EAccessibility(welcomeWindow)) == 0 {
		t.Fatal("welcome window did not publish a native accessibility tree")
	}

	mygo.E2EOpenFiles(secondPath)
	eventuallyNative(t, "Finder-opened second ledger in a new window", func() bool {
		if active == nil || active == welcome || len(routed) != 1 {
			return false
		}
		state := active.workspace.State()
		return state.ActivePath == secondPath && state.Phase == workspace.PhaseReady && !state.Busy &&
			state.Response != nil && state.Response.Summary.TransactionsCount == 1
	})
	unchanged := welcome.workspace.State()
	if unchanged.ActivePath != firstState.ActivePath || unchanged.Epoch != firstState.Epoch ||
		unchanged.Ledger.SnapshotID != firstState.Ledger.SnapshotID || unchanged.Response.Summary.TransactionsCount != 1 {
		t.Fatalf("opening a second Finder document replaced the populated window: before=%+v after=%+v", firstState, unchanged)
	}
}

func TestNativeWelcomeRecentLedgerOpensOnlyAfterSelection(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS welcome window")
	}
	ledgerPath := nativeFixturePath(t, "native/cashflow.bean")
	store := preferences.NewStore(filepath.Join(t.TempDir(), "preferences-v1.json"))
	if err := store.RecordRecentLedger(ledgerPath); err != nil {
		t.Fatal(err)
	}

	model, window := newNativeWelcomeWindowWithPreferences(t, store, false)
	eventuallyNative(t, "welcome page and explicitly reopenable recent ledger", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), "欢迎使用 OrangeCount") &&
			nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXButton", ledgerPath)
	})
	if state := model.workspace.State(); state.ActivePath != "" || model.workspaceState.ActivePath != "" {
		t.Fatalf("a saved recent ledger opened without explicit selection: controller=%+v model=%+v", state, model.workspaceState)
	}
	if !nativeContentHas(mygo.E2EAccessibility(window), "此开发预览不会自动读取最近账本") {
		t.Fatalf("welcome page omitted its no-auto-restore explanation: %s", nativeNodeSummary(mygo.E2EAccessibility(window)))
	}

	performNativeAccessibilityEventually(t, window, "AXButton", ledgerPath, "press", "")
	eventuallyNative(t, "explicitly selected recent ledger to finish opening", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseReady && !state.Busy && state.ActivePath == ledgerPath &&
			state.Response != nil && state.Response.Summary.TransactionsCount == 2
	})
	eventuallyNative(t, "explicitly opened ledger to render the expected cashflow values", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "1000 USD") && nativeContentHas(nodes, "300 USD")
	})
}

func TestNativeWelcomeShowsPreferenceStartupError(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS welcome window")
	}
	model, window := newNativeWelcomeWindow(t)
	window.Update(func() {
		model.SetPreferenceStartupError(fmt.Errorf("permission denied"))
	})
	eventuallyNative(t, "preference startup error in native welcome page", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(window), "应用偏好存储不可用：permission denied")
	})
}

func TestNativeFinderOpenAndWindowDropReadOfflineAnalysis(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open real macOS Content windows")
	}
	firstPath, firstDocument := writeQueryAnalysisFixture(t,
		filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "query", "compound-values.bean"),
		"SELECT account, position WHERE account = 'Assets:Stock' LIMIT 1",
	)
	secondPath, secondDocument := writeQueryAnalysisFixture(t,
		filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "query", "compound-values.bean"),
		"SELECT account, position WHERE account = 'Assets:Stock' LIMIT 2",
	)
	welcome, welcomeWindow := newNativeWelcomeWindow(t)
	active := welcome
	var opened []*Model
	createWindow := func() *Model {
		model, _ := newNativeWelcomeWindow(t)
		opened = append(opened, model)
		return model
	}
	off := mygo.App.OnOpenFile(func(path string) {
		active = RouteOpenDocument(path, active, createWindow)
	})
	defer off()
	welcomeWindow.OnFileDrop(func(event *mygo.FileDropEvent) {
		active = RouteDroppedFiles(event.Paths, welcome, createWindow)
	})

	mygo.E2EOpenFiles(firstPath)
	eventuallyNative(t, "Finder-opened offline analysis in the existing welcome window", func() bool {
		return active == welcome && welcome.offlineAnalysis != nil &&
			welcome.offlineAnalysis.Context == firstDocument.Context && welcome.page == pageQuery
	})
	eventuallyNative(t, "native welcome window renders the Finder-opened capture", func() bool {
		return nativeContentHas(mygo.E2EAccessibility(welcomeWindow), "离线分析捕获")
	})
	if welcome.workspace.State().ActivePath != "" || !nativeContentHas(mygo.E2EAccessibility(welcomeWindow), "离线分析捕获") {
		t.Fatalf("opening an offline capture changed the ledger workspace or hid its offline identity: workspace=%+v accessibility=%s",
			welcome.workspace.State(), nativeNodeSummary(mygo.E2EAccessibility(welcomeWindow)))
	}
	performNativeAccessibilityEventually(t, welcomeWindow, "AXButton", "返回", "press", "")
	eventuallyNative(t, "Back restores the page before the offline capture", func() bool {
		return welcome.page == pageWelcome && nativeContentHas(mygo.E2EAccessibility(welcomeWindow), "欢迎使用 OrangeCount")
	})
	performNativeAccessibilityEventually(t, welcomeWindow, "AXButton", "前进", "press", "")
	eventuallyNative(t, "Forward restores the offline capture without reading a ledger", func() bool {
		return welcome.page == pageQuery && nativeContentHas(mygo.E2EAccessibility(welcomeWindow), "离线分析捕获") &&
			welcome.workspace.State().ActivePath == "" && welcome.status == ""
	})

	over, dropped := mygo.E2EDropFiles(welcomeWindow, 600, 360, secondPath)
	if !over || !dropped {
		t.Fatalf("Finder-style drop was not accepted by the native window: over=%t dropped=%t", over, dropped)
	}
	eventuallyNative(t, "dropped offline analysis in a new window", func() bool {
		return active != nil && active != welcome && len(opened) == 1 && active.offlineAnalysis != nil &&
			active.offlineAnalysis.Context == secondDocument.Context && active.page == pageQuery
	})
	eventuallyNative(t, "native second window renders the dropped capture", func() bool {
		return len(opened) == 1 && nativeContentHas(mygo.E2EAccessibility(opened[0].window), "离线分析捕获")
	})
	if welcome.offlineAnalysisPath != firstPath || welcome.workspace.State().ActivePath != "" ||
		active.workspace.State().ActivePath != "" || !nativeContentHas(mygo.E2EAccessibility(opened[0].window), "离线分析捕获") {
		t.Fatalf("opening a second offline capture replaced or mutated the first window: first=%q active=%q firstWorkspace=%+v secondWorkspace=%+v",
			welcome.offlineAnalysisPath, active.offlineAnalysisPath, welcome.workspace.State(), active.workspace.State())
	}
	corruptPath := filepath.Join(t.TempDir(), "corrupt.ocanalysis")
	if err := os.WriteFile(corruptPath, []byte("not a query archive"), 0o600); err != nil {
		t.Fatal(err)
	}
	mygo.E2EOpenFiles(corruptPath)
	eventuallyNative(t, "invalid Finder-opened analysis shows a read error in an isolated window", func() bool {
		return active != nil && active != opened[0] && len(opened) == 2 &&
			strings.HasPrefix(active.status, "无法读取离线分析文件：") && active.page == pageQuery
	})
	eventuallyNative(t, "native error window renders the invalid capture message", func() bool {
		return len(opened) == 2 && nativeContentHas(mygo.E2EAccessibility(opened[1].window), "无法读取离线分析文件")
	})
	if active.offlineAnalysis != nil || active.workspace.State().ActivePath != "" ||
		!nativeContentHas(mygo.E2EAccessibility(opened[1].window), "无法读取离线分析文件") {
		t.Fatalf("invalid offline archive was not surfaced without opening a ledger: status=%q workspace=%+v accessibility=%s",
			active.status, active.workspace.State(), nativeNodeSummary(mygo.E2EAccessibility(opened[1].window)))
	}
	performNativeAccessibilityEventually(t, opened[1].window, "AXButton", "返回", "press", "")
	eventuallyNative(t, "Back leaves the invalid offline capture page", func() bool {
		return active.page == pageWelcome && nativeContentHas(mygo.E2EAccessibility(opened[1].window), "欢迎使用 OrangeCount")
	})
	performNativeAccessibilityEventually(t, opened[1].window, "AXButton", "前进", "press", "")
	eventuallyNative(t, "Forward restores the offline capture error", func() bool {
		return active.page == pageQuery && active.status == "" &&
			nativeContentHas(mygo.E2EAccessibility(opened[1].window), "无法读取离线分析文件")
	})
}

func TestNativeMissingRecentLedgerKeepsDiagnosticsWhenOpeningAnotherLedger(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open real macOS Content windows")
	}
	missingPath := filepath.Join(t.TempDir(), "missing.bean")
	validPath := filepath.Join(t.TempDir(), "recovered.bean")
	ledger := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Equity:Opening USD
2026-01-02 * "Opening"
  Assets:Cash 10 USD
  Equity:Opening -10 USD
`
	if err := os.WriteFile(validPath, []byte(ledger), 0600); err != nil {
		t.Fatal(err)
	}
	model, window := newNativeWelcomeWindow(t)
	var active *Model = model
	activeWindow := window
	created := 0
	createWindow := func() *Model {
		created++
		newModel, newWindow := newNativeWelcomeWindow(t)
		activeWindow = newWindow
		return newModel
	}
	mygo.RunOnMain(func() { model.recentPaths = []string{missingPath} })
	off := mygo.App.OnOpenFile(func(path string) { active = RouteOpenFile(path, active, createWindow) })
	defer off()

	performNativeAccessibilityEventually(t, window, "AXButton", missingPath, "press", "")
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		state := model.workspace.State()
		if state.Phase == workspace.PhaseInvalid && !state.Busy && state.Response != nil &&
			!state.Response.State.Valid && state.Response.Summary.ErrorCount > 0 && model.workspaceState.Phase == workspace.PhaseInvalid {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	state := model.workspace.State()
	if state.Phase != workspace.PhaseInvalid || state.Busy || state.ActivePath != missingPath || state.Ledger.HasSnapshot ||
		state.Response == nil || state.Response.State.Valid || state.Response.Summary.ErrorCount == 0 || model.workspaceState.Phase != workspace.PhaseInvalid {
		t.Fatalf("missing recent ledger did not publish its diagnostics: controller=%+v model=%+v accessibility=%s", state, model.workspaceState, nativeNodeSummary(mygo.E2EAccessibility(window)))
	}
	if model.recentPaths[0] != missingPath || !nativeContentHas(mygo.E2EAccessibility(window), missingPath) {
		t.Fatalf("opening an unavailable recent ledger removed its recoverable entry: recents=%q", model.recentPaths)
	}
	firstEpoch, firstRevision := state.Epoch, state.Revision

	mygo.E2EOpenFiles(validPath)
	eventuallyNative(t, "a later open event uses a new window and keeps the missing-ledger diagnostics", func() bool {
		if active == nil || active == model || created != 1 {
			return false
		}
		state := active.workspace.State()
		return state.ActivePath == validPath && state.Phase == workspace.PhaseReady && !state.Busy &&
			state.Response != nil && state.Response.Summary.TransactionsCount == 1 &&
			nativeContentHas(mygo.E2EAccessibility(activeWindow), "1 笔交易")
	})
	unchanged := model.workspace.State()
	if unchanged.ActivePath != missingPath || unchanged.Epoch != firstEpoch || unchanged.Ledger.Revision != state.Ledger.Revision || unchanged.Revision != firstRevision ||
		!nativeContentHas(mygo.E2EAccessibility(window), "无法读取 include 文件") {
		t.Fatalf("opening another ledger replaced the unavailable recent workspace: before=%+v after=%+v", state, unchanged)
	}
}

func TestNativeReportSavePanelIsOwnedAndCancellable(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	ledgerPath := filepath.Join(t.TempDir(), "cashflow.bean")
	ledger := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Income:Salary USD
2026-01-02 * "Salary"
  Assets:Cash 10 USD
  Income:Salary -10 USD
`
	if err := os.WriteFile(ledgerPath, []byte(ledger), 0600); err != nil {
		t.Fatal(err)
	}
	model, window := openNativeLedgerWindow(t, ledgerPath)
	performNativeAccessibilityEventually(t, window, "AXButton", "流水", "press", "")
	eventuallyNative(t, "native report export identity", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "journal" && state.Response.ReportExport != nil && !state.Busy
	})
	eventuallyNative(t, "native report save button", func() bool {
		return nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXButton", "保存报表 CSV…")
	})

	otherModel := NewModel(true)
	var otherWindow *mygo.Window
	mygo.RunOnMain(func() {
		otherWindow = mygo.NewWindow(otherModel.WindowOptions())
		otherModel.SetWindow(otherWindow)
		otherWindow.OnClosed(otherModel.Close)
	})
	t.Cleanup(func() {
		otherWindow.Destroy()
		otherModel.Close()
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "保存报表 CSV…", "press", "")
	eventuallyNative(t, "report save panel attached to its owning window", func() bool {
		if mygo.E2ECancelSheet(otherWindow) {
			t.Fatal("native report save panel was attached to the other window")
		}
		return mygo.E2ECancelSheet(window)
	})
	eventuallyNative(t, "cancelled native report save", func() bool {
		var status string
		var active uint64
		mygo.RunOnMain(func() {
			status = model.status
			active = model.activeExportID
		})
		return status == "已取消报表保存。" && active == 0
	})
	if mygo.E2ECancelSheet(window) || mygo.E2ECancelSheet(otherWindow) {
		t.Fatal("a native save panel remained attached after cancellation")
	}
	entries, err := os.ReadDir(filepath.Dir(ledgerPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(ledgerPath) {
		t.Fatalf("cancelled native report save created unexpected files: %v", entries)
	}
}

func TestNativeReportSavePanelClosesWithOwnerWindow(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to open a real macOS Content window")
	}
	ledgerPath := filepath.Join(t.TempDir(), "cashflow.bean")
	ledger := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Income:Salary USD
2026-01-02 * "Salary"
  Assets:Cash 10 USD
  Income:Salary -10 USD
`
	if err := os.WriteFile(ledgerPath, []byte(ledger), 0600); err != nil {
		t.Fatal(err)
	}
	model, window := openNativeLedgerWindow(t, ledgerPath)
	performNativeAccessibilityEventually(t, window, "AXButton", "流水", "press", "")
	eventuallyNative(t, "native report export identity", func() bool {
		state := model.workspace.State()
		return state.Response != nil && state.Response.Request.View == "journal" && state.Response.ReportExport != nil && !state.Busy
	})
	eventuallyNative(t, "native report save button", func() bool {
		return nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXButton", "保存报表 CSV…")
	})

	otherModel := NewModel(true)
	var otherWindow *mygo.Window
	mygo.RunOnMain(func() {
		otherWindow = mygo.NewWindow(otherModel.WindowOptions())
		otherModel.SetWindow(otherWindow)
		otherWindow.OnClosed(otherModel.Close)
	})
	t.Cleanup(func() {
		otherWindow.Destroy()
		otherModel.Close()
	})
	performNativeAccessibilityEventually(t, window, "AXButton", "保存报表 CSV…", "press", "")
	eventuallyNative(t, "report panel attached to owner", func() bool {
		return mygo.E2EHasSheet(window)
	})
	if !mygo.E2ESavePanelVisible() {
		t.Fatal("the owning Content window has a sheet, but the native NSSavePanel does not report visible")
	}
	if mygo.E2EHasSheet(otherWindow) {
		t.Fatal("native report save panel was attached to the other window")
	}
	window.Close()
	eventuallyNative(t, "owner window close and report panel dismissal", func() bool {
		return window.IsDestroyed() && !mygo.E2ESavePanelVisible()
	})
	if mygo.E2EHasSheet(otherWindow) {
		t.Fatal("closing the owner left a report save sheet attached to the other window")
	}
	eventuallyNative(t, "closed report workspace", func() bool {
		select {
		case <-model.workspace.Done():
			return true
		default:
			return false
		}
	})
	entries, err := os.ReadDir(filepath.Dir(ledgerPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(ledgerPath) {
		t.Fatalf("closing the window during report save created unexpected files: %v", entries)
	}
}

func TestNativeSourceEditorMenuOpensPickerCancelsAndRestoresDefault(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to exercise the native source-editor menu and panel")
	}
	model, window := openNativeCashflowWindow(t)
	store := preferences.NewStore(filepath.Join(t.TempDir(), "preferences-v1.json"))
	if err := store.SetSourceEditorBundleID("com.example.source-editor"); err != nil {
		t.Fatal(err)
	}
	model.SetPreferences(store)
	otherModel := NewModel(true)
	var otherWindow *mygo.Window
	mygo.RunOnMain(func() {
		otherWindow = mygo.NewWindow(otherModel.WindowOptions())
		otherModel.SetWindow(otherWindow)
		otherWindow.OnClosed(otherModel.Close)
	})
	t.Cleanup(func() {
		otherWindow.Destroy()
		otherModel.Close()
	})
	mygo.App.SetMenu(NewApplicationMenu(func(candidate *mygo.Window) *Model {
		if candidate == window {
			return model
		}
		return nil
	}, nil))
	window.Focus()
	if err := mygo.E2EActivateMenu("文件", "选择来源编辑器…"); err != nil {
		t.Fatalf("activate native source-editor chooser: %v", err)
	}
	if mygo.E2ECancelSheet(otherWindow) {
		t.Fatal("source-editor panel was attached to the other window")
	}
	if !mygo.E2ECancelSheet(window) {
		t.Fatal("source-editor panel was not attached to its owning window")
	}
	eventuallyNative(t, "cancelled native source-editor chooser", func() bool {
		var picking bool
		var status string
		mygo.RunOnMain(func() {
			picking = model.picking
			status = model.status
		})
		return !picking && status == "已取消来源编辑器选择。"
	})
	if mygo.E2ECancelSheet(window) || mygo.E2ECancelSheet(otherWindow) {
		t.Fatal("a native file panel remained attached after cancellation")
	}
	if bundleID, err := store.SourceEditorBundleID(); err != nil || bundleID != "com.example.source-editor" {
		t.Fatalf("cancelled native chooser changed editor preference to %q, %v", bundleID, err)
	}
	if err := mygo.E2EActivateMenu("文件", "使用系统默认应用打开来源"); err != nil {
		t.Fatalf("activate native system-default source-editor command: %v", err)
	}
	eventuallyNative(t, "native system-default source-editor preference", func() bool {
		bundleID, err := store.SourceEditorBundleID()
		if err != nil || bundleID != "" {
			return false
		}
		var status string
		mygo.RunOnMain(func() { status = model.status })
		return status == "已恢复来源文件的系统默认应用。"
	})
}

func TestNativeConfiguredSourceEditorRoutesFileToTextEdit(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" || os.Getenv("ORANGECOUNT_GOUI_SOURCE_EDITOR_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 and ORANGECOUNT_GOUI_SOURCE_EDITOR_E2E=1 to launch the configured external editor")
	}
	const bundleID = "com.apple.TextEdit"
	if _, err := os.Stat("/System/Applications/TextEdit.app"); err != nil {
		t.Skipf("system TextEdit bundle is unavailable: %v", err)
	}

	wasRunning := mygo.E2EApplicationRunning(bundleID)
	t.Logf("TextEdit running before the source-open request: %t", wasRunning)
	if !wasRunning {
		t.Cleanup(func() {
			if !mygo.E2EApplicationRunning(bundleID) {
				return
			}
			if !mygo.E2ETerminateApplication(bundleID) {
				t.Errorf("could not terminate TextEdit launched by the native source-editor test")
				return
			}
			eventuallyNative(t, "TextEdit test cleanup", func() bool {
				return !mygo.E2EApplicationRunning(bundleID)
			})
		})
	}

	path := filepath.Join(t.TempDir(), "source.bean")
	contents := "2000-01-01 open Assets:Cash USD\n"
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	if err := platform.OpenSourceWithEditor(path, bundleID); err != nil {
		t.Fatalf("route source to configured TextEdit bundle: %v", err)
	}
	if !wasRunning {
		eventuallyNative(t, "configured TextEdit launch", func() bool {
			return mygo.E2EApplicationRunning(bundleID)
		})
		t.Log("TextEdit started by the configured source-open request")
	}
}

func TestNativeConfiguredSourceEditorLaunchesScriptEditor(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" || os.Getenv("ORANGECOUNT_GOUI_SOURCE_EDITOR_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 and ORANGECOUNT_GOUI_SOURCE_EDITOR_E2E=1 to launch a source editor that is not already running")
	}
	const appPath = "/System/Applications/Utilities/Script Editor.app"
	if _, err := os.Stat(appPath); err != nil {
		t.Skipf("system Script Editor bundle is unavailable: %v", err)
	}
	editor, err := platform.SourceEditorFromApplication(appPath)
	if err != nil {
		t.Fatalf("inspect system Script Editor bundle: %v", err)
	}
	if mygo.E2EApplicationRunning(editor.BundleID) {
		t.Skipf("%s is already running; leaving the existing app and its documents untouched", editor.Name)
	}
	t.Cleanup(func() {
		if !mygo.E2EApplicationRunning(editor.BundleID) {
			return
		}
		if !mygo.E2ETerminateApplication(editor.BundleID) {
			t.Errorf("could not terminate %s launched by the native source-editor test", editor.Name)
			return
		}
		eventuallyNative(t, editor.Name+" test cleanup", func() bool {
			return !mygo.E2EApplicationRunning(editor.BundleID)
		})
	})

	path := filepath.Join(t.TempDir(), "source.bean")
	contents := "2000-01-01 open Assets:Cash USD\n"
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	if err := platform.OpenSourceWithEditor(path, editor.BundleID); err != nil {
		t.Fatalf("route source to configured %s bundle: %v", editor.Name, err)
	}
	eventuallyNative(t, "configured "+editor.Name+" first launch", func() bool {
		return mygo.E2EApplicationRunning(editor.BundleID)
	})
	t.Logf("configured source-open request launched %s for a temporary source file", editor.Name)
}

func TestNativeFindShortcutTargetsTheFocusedLedgerWindow(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to route the native Find shortcut across real windows")
	}
	root := t.TempDir()
	ledgerText := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Equity:Opening USD
2026-01-02 * "Opening"
  Assets:Cash 10 USD
  Equity:Opening -10 USD
`
	firstPath, secondPath := filepath.Join(root, "first.bean"), filepath.Join(root, "second.bean")
	for _, path := range []string{firstPath, secondPath} {
		if err := os.WriteFile(path, []byte(ledgerText), 0600); err != nil {
			t.Fatal(err)
		}
	}
	firstModel, firstWindow := openNativeLedgerWindow(t, firstPath)
	secondModel, secondWindow := openNativeLedgerWindow(t, secondPath)
	mygo.App.SetMenu(NewApplicationMenu(func(candidate *mygo.Window) *Model {
		switch candidate {
		case firstWindow:
			return firstModel
		case secondWindow:
			return secondModel
		default:
			return nil
		}
	}, nil))
	t.Cleanup(func() { mygo.App.SetMenu(nil) })
	openJournal := func(model *Model, window *mygo.Window, label string) {
		t.Helper()
		window.Update(func() { model.requestView("journal") })
		eventuallyNative(t, label+" journal response", func() bool {
			state := model.workspace.State()
			return !state.Busy && state.Response != nil && state.Response.Request.View == "journal"
		})
		eventuallyNative(t, label+" native search field", func() bool {
			return nativeRoleLabelExists(mygo.E2EAccessibility(window), "AXTextField", "搜索流水")
		})
	}
	openJournal(firstModel, firstWindow, "first")
	openJournal(secondModel, secondWindow, "second")
	firstWindow.Focus()
	eventuallyNative(t, "first ledger window becomes key", func() bool { return firstWindow.IsFocused() })
	if !mygo.E2EPerformKeyEquivalent("f", false) {
		t.Fatal("native Command-F was not handled by the application menu")
	}
	eventuallyNative(t, "Find focuses the first ledger search field", func() bool {
		return nativeAccessibilityFieldFocused(mygo.E2EAccessibility(firstWindow), "搜索流水")
	})
	var secondPending bool
	mygo.RunOnMain(func() { secondPending = secondModel.searchFocusPending })
	if secondPending {
		t.Fatal("Find shortcut was routed to the non-key ledger window")
	}

	secondWindow.Focus()
	eventuallyNative(t, "second ledger window becomes key", func() bool { return secondWindow.IsFocused() })
	if !mygo.E2EPerformKeyEquivalent("f", false) {
		t.Fatal("native Command-F was not handled by the application menu for the second window")
	}
	eventuallyNative(t, "Find focuses the second ledger search field", func() bool {
		return nativeAccessibilityFieldFocused(mygo.E2EAccessibility(secondWindow), "搜索流水")
	})
	var firstPending bool
	mygo.RunOnMain(func() { firstPending = firstModel.searchFocusPending })
	if firstPending {
		t.Fatal("Find shortcut remained routed to the prior ledger window")
	}
}

func TestNativeOpenLedgerShortcutTargetsTheFocusedWindow(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to route the native Open shortcut across real windows")
	}
	root := t.TempDir()
	ledgerText := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Equity:Opening USD
2026-01-02 * "Opening"
  Assets:Cash 10 USD
  Equity:Opening -10 USD
`
	firstPath, secondPath := filepath.Join(root, "first.bean"), filepath.Join(root, "second.bean")
	for _, path := range []string{firstPath, secondPath} {
		if err := os.WriteFile(path, []byte(ledgerText), 0600); err != nil {
			t.Fatal(err)
		}
	}
	firstModel, firstWindow := openNativeLedgerWindow(t, firstPath)
	secondModel, secondWindow := openNativeLedgerWindow(t, secondPath)
	mygo.App.SetMenu(NewApplicationMenu(func(candidate *mygo.Window) *Model {
		switch candidate {
		case firstWindow:
			return firstModel
		case secondWindow:
			return secondModel
		default:
			return nil
		}
	}, nil))
	t.Cleanup(func() { mygo.App.SetMenu(nil) })

	for _, target := range []struct {
		name        string
		model       *Model
		window      *mygo.Window
		otherWindow *mygo.Window
	}{
		{name: "first", model: firstModel, window: firstWindow, otherWindow: secondWindow},
		{name: "second", model: secondModel, window: secondWindow, otherWindow: firstWindow},
	} {
		target.window.Focus()
		eventuallyNative(t, target.name+" ledger window becomes key before Open", func() bool { return target.window.IsFocused() })
		if !mygo.E2EPerformKeyEquivalent("o", false) {
			t.Fatalf("native Command-O was not handled by the application menu for the %s window", target.name)
		}
		eventuallyNative(t, target.name+" window owns the Open Panel", func() bool {
			return mygo.E2EHasSheet(target.window) && !mygo.E2EHasSheet(target.otherWindow)
		})
		if !mygo.E2ECancelSheet(target.window) {
			t.Fatalf("native Open Panel for the %s window could not be cancelled", target.name)
		}
		eventuallyNative(t, target.name+" Open Panel cancellation completes", func() bool {
			var picking bool
			mygo.RunOnMain(func() { picking = target.model.picking })
			return !picking && !mygo.E2EHasSheet(target.window) && !mygo.E2EHasSheet(target.otherWindow)
		})
	}
}

func TestNativeReloadShortcutTargetsTheFocusedWindow(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to route the native Reload shortcut across real windows")
	}
	root := t.TempDir()
	ledgerText := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Equity:Opening USD
2026-01-02 * "Opening"
  Assets:Cash 10 USD
  Equity:Opening -10 USD
`
	firstPath, secondPath := filepath.Join(root, "first.bean"), filepath.Join(root, "second.bean")
	for _, path := range []string{firstPath, secondPath} {
		if err := os.WriteFile(path, []byte(ledgerText), 0600); err != nil {
			t.Fatal(err)
		}
	}
	firstModel, firstWindow := openNativeLedgerWindow(t, firstPath)
	secondModel, secondWindow := openNativeLedgerWindow(t, secondPath)
	mygo.App.SetMenu(NewApplicationMenu(func(candidate *mygo.Window) *Model {
		switch candidate {
		case firstWindow:
			return firstModel
		case secondWindow:
			return secondModel
		default:
			return nil
		}
	}, nil))
	t.Cleanup(func() { mygo.App.SetMenu(nil) })

	for _, target := range []struct {
		name   string
		model  *Model
		window *mygo.Window
		other  *Model
	}{
		{name: "first", model: firstModel, window: firstWindow, other: secondModel},
		{name: "second", model: secondModel, window: secondWindow, other: firstModel},
	} {
		before := target.model.workspace.State().Operation
		otherBefore := target.other.workspace.State().Operation
		target.window.Focus()
		eventuallyNative(t, target.name+" ledger window becomes key before Reload", func() bool { return target.window.IsFocused() })
		if !mygo.E2EPerformKeyEquivalent("r", false) {
			t.Fatalf("native Command-R was not handled by the application menu for the %s window", target.name)
		}
		eventuallyNative(t, target.name+" window completes its native Reload", func() bool {
			state := target.model.workspace.State()
			return state.Operation > before && !state.Busy && state.Response != nil && state.Response.Request.View == "dashboard"
		})
		if got := target.other.workspace.State().Operation; got != otherBefore {
			t.Fatalf("Reload from the %s window changed the other window operation from %d to %d", target.name, otherBefore, got)
		}
	}
}

func TestNativeNewWindowShortcutCreatesNativeWindow(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_NATIVE_E2E=1 to create a native window from the application shortcut")
	}
	ledgerPath := filepath.Join(t.TempDir(), "menu-owner.bean")
	ledgerText := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Equity:Opening USD
2026-01-02 * "Opening"
  Assets:Cash 10 USD
  Equity:Opening -10 USD
`
	if err := os.WriteFile(ledgerPath, []byte(ledgerText), 0600); err != nil {
		t.Fatal(err)
	}
	model, window := openNativeLedgerWindow(t, ledgerPath)
	type createdNativeWindow struct {
		model  *Model
		window *mygo.Window
	}
	var created []createdNativeWindow
	t.Cleanup(func() {
		for _, item := range created {
			item.window.Destroy()
			item.model.Close()
		}
	})
	createWindow := func() *Model {
		newModel := NewModel(true)
		newWindow := mygo.NewWindow(newModel.WindowOptions())
		newModel.SetWindow(newWindow)
		newWindow.OnClosed(newModel.Close)
		created = append(created, createdNativeWindow{model: newModel, window: newWindow})
		return newModel
	}
	mygo.App.SetMenu(NewApplicationMenu(func(candidate *mygo.Window) *Model {
		if candidate == window {
			return model
		}
		for _, item := range created {
			if candidate == item.window {
				return item.model
			}
		}
		return nil
	}, createWindow))
	t.Cleanup(func() { mygo.App.SetMenu(nil) })

	for range 2 {
		before := len(created)
		window.Focus()
		eventuallyNative(t, "ledger window becomes key before New Window", func() bool { return window.IsFocused() })
		if !mygo.E2EPerformKeyEquivalent("n", false) {
			t.Fatal("native Command-N was not handled by the application menu")
		}
		eventuallyNative(t, "Command-N creates and displays one new Content window", func() bool {
			return len(created) == before+1 && !created[before].window.IsDestroyed() &&
				nativeRoleLabelExists(mygo.E2EAccessibility(created[before].window), "AXButton", "打开账本")
		})
	}
}

func nativeAccessibilityFieldFocused(nodes []mygo.E2EAccessNode, label string) bool {
	for _, node := range nodes {
		if node.Role == "AXTextField" && node.Label == label && node.Focused {
			return true
		}
	}
	return false
}

func openNativeCashflowWindow(t *testing.T) (*Model, *mygo.Window) {
	t.Helper()
	model, window := openNativeLedgerWindow(t, nativeFixturePath(t, "native/cashflow.bean"))
	eventuallyNative(t, "cashflow amounts in native dashboard", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "1000 USD") && nativeContentHas(nodes, "300 USD")
	})
	return model, window
}

func nativeFixturePath(t *testing.T, relative string) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "../../../.."))
	return filepath.Join(root, "testdata", "fixtures", filepath.FromSlash(relative))
}

func openNativeLedgerWindow(t *testing.T, ledgerPath string) (*Model, *mygo.Window) {
	return openNativeLedgerWindowWithProfile(t, ledgerPath, "")
}

func openNativeLedgerWindowWithProfile(t *testing.T, ledgerPath, profile string) (*Model, *mygo.Window) {
	return openNativeLedgerWindowWithEngine(t, ledgerPath, profile, nil)
}

func openNativeLedgerWindowWithEngine(t *testing.T, ledgerPath, profile string, engine workspace.Engine) (*Model, *mygo.Window) {
	t.Helper()
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Fatal("ORANGECOUNT_GOUI_NATIVE_E2E=1 is required")
	}
	if _, err := os.Stat(ledgerPath); err != nil {
		t.Fatalf("native E2E ledger unavailable: %v", err)
	}
	model := newModelWithEngine(true, engine)
	if profile != "" {
		store := preferences.NewStore(filepath.Join(t.TempDir(), "preferences-v1.json"))
		if err := store.UpdateLedger(ledgerPath, func(item *preferences.LedgerPreferences) error {
			item.SemanticsProfile = profile
			return nil
		}); err != nil {
			t.Fatalf("set native E2E semantics profile: %v", err)
		}
		model.SetPreferences(store)
	}
	var window *mygo.Window
	mygo.RunOnMain(func() {
		options := model.WindowOptions()
		options.Title = "OrangeCount Go UI Native E2E"
		window = mygo.NewWindow(options)
		model.SetWindow(window)
		window.OnClosed(model.Close)
	})
	t.Cleanup(func() {
		window.Destroy()
		model.Close()
		if model.workspace != nil {
			select {
			case <-model.workspace.Done():
			case <-time.After(10 * time.Second):
				t.Error("native E2E workspace did not close")
			}
		}
	})
	model.OpenLedgerPath(ledgerPath)
	eventuallyNative(t, "dashboard snapshot", func() bool {
		state := model.workspace.State()
		return state.Phase == workspace.PhaseReady && !state.Busy && state.Response != nil && state.Response.Request.View == "dashboard"
	})
	eventuallyNative(t, "native dashboard accessibility content", func() bool {
		nodes := mygo.E2EAccessibility(window)
		return nativeContentHas(nodes, "财务概览")
	})
	return model, window
}

type nativeReadGateEngine struct {
	release <-chan struct{}
	started chan<- desktop.Request
}

func (e *nativeReadGateEngine) Open(ctx context.Context, path string, options workspace.OpenOptions) (workspace.Session, error) {
	session, err := (workspace.DesktopEngine{}).Open(ctx, path, options)
	if err != nil {
		return nil, err
	}
	return &nativeReadGateSession{Session: session, release: e.release, started: e.started}, nil
}

type nativeReadGateSession struct {
	workspace.Session
	release <-chan struct{}
	started chan<- desktop.Request
	once    sync.Once
}

type nativeStaleMigrationGateEngine struct {
	started chan<- desktop.Request
	release <-chan struct{}
}

func (e *nativeStaleMigrationGateEngine) Open(ctx context.Context, path string, options workspace.OpenOptions) (workspace.Session, error) {
	session, err := (workspace.DesktopEngine{}).Open(ctx, path, options)
	if err != nil {
		return nil, err
	}
	state, err := session.Poll()
	if err != nil {
		session.Close()
		return nil, err
	}
	return &nativeStaleMigrationGateSession{Session: session, snapshotID: state.SnapshotID, started: e.started, release: e.release}, nil
}

type nativeStaleMigrationGateSession struct {
	workspace.Session
	snapshotID string
	started    chan<- desktop.Request
	release    <-chan struct{}
	once       sync.Once
}

func (s *nativeStaleMigrationGateSession) Read(request desktop.Request) (desktop.ViewResponse, error) {
	if request.View == "migration" && request.SnapshotID != s.snapshotID {
		s.once.Do(func() {
			s.started <- request
			<-s.release
		})
	}
	return s.Session.Read(request)
}

func (s *nativeReadGateSession) Read(request desktop.Request) (desktop.ViewResponse, error) {
	if request.View == "balance_sheet" {
		s.once.Do(func() {
			s.started <- request
			<-s.release
		})
	}
	return s.Session.Read(request)
}

type nativeQueryReloadGateEngine struct {
	started chan<- struct{}
	release <-chan struct{}
}

func (e *nativeQueryReloadGateEngine) Open(ctx context.Context, path string, options workspace.OpenOptions) (workspace.Session, error) {
	options.Watch.PollInterval = time.Hour
	session, err := (workspace.DesktopEngine{}).Open(ctx, path, options)
	if err != nil {
		return nil, err
	}
	return &nativeQueryReloadGateSession{Session: session, started: e.started, release: e.release}, nil
}

type nativeQueryReloadGateSession struct {
	workspace.Session
	started chan<- struct{}
	release <-chan struct{}
	once    sync.Once
}

func (s *nativeQueryReloadGateSession) Reload() error {
	s.once.Do(func() {
		s.started <- struct{}{}
		<-s.release
	})
	return s.Session.Reload()
}

type nativeQueryCaptureGateEngine struct {
	release <-chan struct{}
	started chan<- desktop.QueryExportRequest
}

type nativeQueryCaptureSessionAPI interface {
	workspace.Session
	ExecuteQuery(context.Context, desktop.QueryRequest) (desktop.QueryResponse, error)
	CancelQuery(uint64) error
	CaptureQueryExport(desktop.QueryExportRequest) (desktop.QueryExportCapture, error)
}

func (e *nativeQueryCaptureGateEngine) Open(ctx context.Context, path string, options workspace.OpenOptions) (workspace.Session, error) {
	session, err := (workspace.DesktopEngine{}).Open(ctx, path, options)
	if err != nil {
		return nil, err
	}
	querySession, ok := session.(nativeQueryCaptureSessionAPI)
	if !ok {
		session.Close()
		return nil, fmt.Errorf("desktop session does not support native query capture E2E")
	}
	return &nativeQueryCaptureGateSession{nativeQueryCaptureSessionAPI: querySession, release: e.release, started: e.started}, nil
}

type nativeQueryCaptureGateSession struct {
	nativeQueryCaptureSessionAPI
	release <-chan struct{}
	started chan<- desktop.QueryExportRequest
}

func (s *nativeQueryCaptureGateSession) CaptureQueryExport(request desktop.QueryExportRequest) (desktop.QueryExportCapture, error) {
	s.started <- request
	<-s.release
	return s.nativeQueryCaptureSessionAPI.CaptureQueryExport(request)
}

type nativeQueryExecutionGateEngine struct {
	release <-chan struct{}
	started chan<- desktop.QueryRequest
}

func (e *nativeQueryExecutionGateEngine) Open(ctx context.Context, path string, options workspace.OpenOptions) (workspace.Session, error) {
	session, err := (workspace.DesktopEngine{}).Open(ctx, path, options)
	if err != nil {
		return nil, err
	}
	querySession, ok := session.(nativeQueryCaptureSessionAPI)
	if !ok {
		session.Close()
		return nil, fmt.Errorf("desktop session does not support native query execution E2E")
	}
	return &nativeQueryExecutionGateSession{nativeQueryCaptureSessionAPI: querySession, release: e.release, started: e.started}, nil
}

type nativeQueryExecutionGateSession struct {
	nativeQueryCaptureSessionAPI
	release <-chan struct{}
	started chan<- desktop.QueryRequest
}

func (s *nativeQueryExecutionGateSession) ExecuteQuery(ctx context.Context, request desktop.QueryRequest) (desktop.QueryResponse, error) {
	s.started <- request
	<-s.release
	return s.nativeQueryCaptureSessionAPI.ExecuteQuery(ctx, request)
}

type nativeLateQueryGateEngine struct {
	release  <-chan struct{}
	started  chan<- desktop.QueryRequest
	finished chan<- struct{}
}

func (e *nativeLateQueryGateEngine) Open(ctx context.Context, path string, options workspace.OpenOptions) (workspace.Session, error) {
	options.Watch.PollInterval = time.Hour
	session, err := (workspace.DesktopEngine{}).Open(ctx, path, options)
	if err != nil {
		return nil, err
	}
	querySession, ok := session.(nativeQueryCaptureSessionAPI)
	if !ok {
		session.Close()
		return nil, fmt.Errorf("desktop session does not support native reload-race E2E")
	}
	return &nativeLateQueryGateSession{nativeQueryCaptureSessionAPI: querySession, release: e.release, started: e.started, finished: e.finished}, nil
}

type nativeLateQueryGateSession struct {
	nativeQueryCaptureSessionAPI
	release  <-chan struct{}
	started  chan<- desktop.QueryRequest
	finished chan<- struct{}
}

func (s *nativeLateQueryGateSession) ExecuteQuery(ctx context.Context, request desktop.QueryRequest) (desktop.QueryResponse, error) {
	s.started <- request
	<-s.release
	defer func() { s.finished <- struct{}{} }()
	return s.nativeQueryCaptureSessionAPI.ExecuteQuery(ctx, request)
}

type nativeQueryCancelGateEngine struct {
	started   chan<- desktop.QueryRequest
	cancelled chan<- uint64
}

func (e *nativeQueryCancelGateEngine) Open(ctx context.Context, path string, options workspace.OpenOptions) (workspace.Session, error) {
	session, err := (workspace.DesktopEngine{}).Open(ctx, path, options)
	if err != nil {
		return nil, err
	}
	querySession, ok := session.(nativeQueryCaptureSessionAPI)
	if !ok {
		session.Close()
		return nil, fmt.Errorf("desktop session does not support native query-cancel E2E")
	}
	return &nativeQueryCancelGateSession{nativeQueryCaptureSessionAPI: querySession, started: e.started, cancelled: e.cancelled}, nil
}

type nativeQueryCancelGateSession struct {
	nativeQueryCaptureSessionAPI
	started   chan<- desktop.QueryRequest
	cancelled chan<- uint64
}

func (s *nativeQueryCancelGateSession) ExecuteQuery(ctx context.Context, request desktop.QueryRequest) (desktop.QueryResponse, error) {
	s.started <- request
	<-ctx.Done()
	return s.nativeQueryCaptureSessionAPI.ExecuteQuery(ctx, request)
}

func (s *nativeQueryCancelGateSession) CancelQuery(requestID uint64) error {
	s.cancelled <- requestID
	return s.nativeQueryCaptureSessionAPI.CancelQuery(requestID)
}

func selectNativeRequestEventually(t *testing.T, started <-chan desktop.Request, view string) {
	t.Helper()
	select {
	case request := <-started:
		if request.View != view {
			t.Fatalf("blocked native read started for %q, want %q", request.View, view)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("native %s read did not reach the blocking test session", view)
	}
}

func newNativeWelcomeWindow(t *testing.T) (*Model, *mygo.Window) {
	return newNativeWelcomeWindowWithPreferences(t, nil, true)
}

func newNativeWelcomeWindowWithPreferences(t *testing.T, store *preferences.Store, qa bool) (*Model, *mygo.Window) {
	t.Helper()
	if os.Getenv("ORANGECOUNT_GOUI_NATIVE_E2E") != "1" {
		t.Fatal("ORANGECOUNT_GOUI_NATIVE_E2E=1 is required")
	}
	model := NewModel(qa)
	if store != nil {
		model.SetPreferences(store)
	}
	var window *mygo.Window
	mygo.RunOnMain(func() {
		options := model.WindowOptions()
		options.Title = "OrangeCount Go UI Native E2E"
		window = mygo.NewWindow(options)
		model.SetWindow(window)
		window.OnClosed(model.Close)
	})
	t.Cleanup(func() {
		window.Destroy()
		model.Close()
		if model.workspace != nil {
			select {
			case <-model.workspace.Done():
			case <-time.After(10 * time.Second):
				t.Error("native E2E welcome workspace did not close")
			}
		}
	})
	return model, window
}

func preserveNativePasteboard(t *testing.T) {
	t.Helper()
	snapshot, ok := mygo.E2EPasteboardSnapshot()
	if !ok {
		t.Skip("the general pasteboard contains an item whose promised data cannot be safely restored")
	}
	t.Cleanup(func() {
		if !mygo.E2ERestorePasteboard(snapshot) {
			t.Errorf("could not restore the original general pasteboard after the native integration test")
			return
		}
		restored, readable := mygo.E2EPasteboardSnapshot()
		if !readable || !reflect.DeepEqual(restored, snapshot) {
			t.Errorf("restored general pasteboard does not match its original item/type payload")
		}
	})
}

func eventuallyNative(t *testing.T, what string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func waitForStableNativeAccessibilityNode(t *testing.T, window *mygo.Window, role, label string, samples int) mygo.E2EAccessNode {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	var previousID uint64
	var stableSamples int
	var lastNodes []mygo.E2EAccessNode
	var stableNode mygo.E2EAccessNode
	for time.Now().Before(deadline) {
		lastNodes = mygo.E2EAccessibility(window)
		var matches []mygo.E2EAccessNode
		for _, node := range lastNodes {
			if node.Role == role && node.Label == label {
				matches = append(matches, node)
			}
		}
		if len(matches) == 1 && matches[0].ID != 0 {
			stableNode = matches[0]
			if stableNode.ID == previousID {
				stableSamples++
			} else {
				previousID, stableSamples = stableNode.ID, 1
			}
			if stableSamples >= samples {
				return stableNode
			}
		} else {
			previousID, stableSamples = 0, 0
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("native accessibility node %q did not remain unique and stable for %d samples; observed: %s", label, samples, nativeNodeSummary(lastNodes))
	return mygo.E2EAccessNode{}
}

func performNativeAccessibilityEventually(t *testing.T, window *mygo.Window, role, label, action, value string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	var nodes []mygo.E2EAccessNode
	for time.Now().Before(deadline) {
		nodes = mygo.E2EAccessibility(window)
		for _, node := range nodes {
			if node.Role != role || node.Label != label {
				continue
			}
			if action == "press" {
				found, allowed, performed := mygo.E2EAccessibilityPressID(window, node.ID)
				if found && allowed && performed {
					return
				}
				break
			}
			if !mygo.E2EPerformAccessibility(window, label, action, value) {
				break
			}
			if action == "focus" && (role == "AXTextField" || role == "AXTextArea") {
				for _, current := range mygo.E2EAccessibility(window) {
					if current.Role == role && current.Label == label && current.Focused {
						return
					}
				}
				break
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("native accessibility %s %q was not ready; observed: %s", action, label, nativeNodeSummary(nodes))
}

func pressNativeAccessibilityEventually(t *testing.T, window *mygo.Window, label string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	var nodes []mygo.E2EAccessNode
	var found, allowed, performed bool
	for time.Now().Before(deadline) {
		nodes = mygo.E2EAccessibility(window)
		if nativeRoleLabelExists(nodes, "AXButton", label) {
			found, allowed, performed = mygo.E2EAccessibilityPress(window, label)
			if found && allowed && performed {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("native accessibility press %q did not execute: found=%t allowed=%t performed=%t; observed: %s", label, found, allowed, performed, nativeNodeSummary(nodes))
}

func setNativeFieldValueEventually(t *testing.T, window *mygo.Window, label, value string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	var nodes []mygo.E2EAccessNode
	for time.Now().Before(deadline) {
		nodes = mygo.E2EAccessibility(window)
		if nativeRoleLabelExists(nodes, "AXTextField", label) && mygo.E2EPerformAccessibility(window, label, "value", value) {
			for _, node := range mygo.E2EAccessibility(window) {
				if node.Label == label && node.Value == value {
					return
				}
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("native text field %q did not accept %q; observed: %s", label, value, nativeNodeSummary(nodes))
}

func clickNativeAccessibilityEventually(t *testing.T, window *mygo.Window, role, label string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	var nodes []mygo.E2EAccessNode
	for time.Now().Before(deadline) {
		nodes = mygo.E2EAccessibility(window)
		if nativeRoleLabelExists(nodes, role, label) && mygo.E2EClickAccessibility(window, label) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("native accessibility could not click %q at its visible bounds; observed: %s", label, nativeNodeSummary(nodes))
}

func chooseNativePopupOptionEventually(t *testing.T, window *mygo.Window, triggerLabel, optionLabel string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	pressAttempts := 0
	nextPress := time.Time{}
	var nodes []mygo.E2EAccessNode
	for time.Now().Before(deadline) {
		nodes = mygo.E2EAccessibility(window)
		var optionID uint64
		var optionMatches int
		for _, node := range nodes {
			if node.Role == "AXStaticText" && node.Label == optionLabel {
				optionID = node.ID
				optionMatches++
			}
		}
		if optionMatches == 1 && mygo.E2EClickAccessibilityID(window, optionID) {
			return
		}
		if pressAttempts < 4 && !time.Now().Before(nextPress) {
			for _, node := range nodes {
				if node.Role == "AXPopUpButton" && node.Label == triggerLabel {
					found, allowed, performed := mygo.E2EAccessibilityPressID(window, node.ID)
					if found && allowed && performed {
						pressAttempts++
						nextPress = time.Now().Add(300 * time.Millisecond)
					}
					break
				}
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("native selector %q did not expose a unique clickable option %q after %d presses: %s", triggerLabel, optionLabel, pressAttempts, nativeNodeSummary(nodes))
}

func nativeContentHas(nodes []mygo.E2EAccessNode, value string) bool {
	for _, node := range nodes {
		if strings.Contains(node.Label, value) || strings.Contains(node.Value, value) {
			return true
		}
	}
	return false
}

func nativeContentPixelSummary(window *mygo.Window) ([]color.RGBA, float64, error) {
	pngBytes, err := window.CapturePage()
	if err != nil {
		return nil, 0, err
	}
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return nil, 0, err
	}
	bounds := img.Bounds()
	if bounds.Empty() {
		return nil, 0, fmt.Errorf("captured Content has empty bounds %v", bounds)
	}
	const columns, rows = 48, 32
	pixels := make([]color.RGBA, 0, columns*rows)
	var luminance float64
	for row := 0; row < rows; row++ {
		y := bounds.Min.Y + (2*row+1)*bounds.Dy()/(2*rows)
		for column := 0; column < columns; column++ {
			x := bounds.Min.X + (2*column+1)*bounds.Dx()/(2*columns)
			r, g, b, a := img.At(x, y).RGBA()
			pixel := color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
			pixels = append(pixels, pixel)
			luminance += 0.2126*float64(pixel.R) + 0.7152*float64(pixel.G) + 0.0722*float64(pixel.B)
		}
	}
	return pixels, luminance / float64(len(pixels)), nil
}

func nativeAbsoluteInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func nativeRoleLabelExists(nodes []mygo.E2EAccessNode, role, label string) bool {
	for _, node := range nodes {
		if node.Role == role && node.Label == label {
			return true
		}
	}
	return false
}

func nativeLabels(nodes []mygo.E2EAccessNode) string {
	labels := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if node.Label != "" {
			labels = append(labels, node.Role+":"+node.Label)
		}
	}
	return strings.Join(labels, " | ")
}

func nativeNodeSummary(nodes []mygo.E2EAccessNode) string {
	values := make([]string, 0, len(nodes))
	for _, node := range nodes {
		values = append(values, fmt.Sprintf("%s:%q=%q", node.Role, node.Label, node.Value))
	}
	return strings.Join(values, " | ")
}
