// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"orangecount/internal/diagnostic"
	"orangecount/internal/dialect"
	"orangecount/internal/ledger"
	"orangecount/internal/logging"
	"orangecount/internal/query"
	"orangecount/internal/repairguidance"
	"orangecount/internal/snapshot"
	"path/filepath"

	"orangecount/internal/source"
	"orangecount/internal/web"
)

// version is overridable at build time via -ldflags "-X main.version=<tag>" so
// release binaries report their release tag; local builds keep the dev marker.
var version = "0.1.3-dev"

const defaultServeAddr = "127.0.0.1:5000"

type portOwner struct {
	PID     int
	Command string
}

var (
	inspectPortOwners = lsofPortOwners
	stopPortOwner     = terminatePortOwner
	waitForPort       = waitForPortRelease
	runLsof           = func(port string) ([]byte, error) {
		return exec.Command("lsof", "-nP", "-iTCP:"+port, "-sTCP:LISTEN", "-Fpc").Output()
	}
)

func main() {
	os.Exit(runWithInput(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	return runWithInput(args, nil, stdout, stderr)
}

// runWithInput dispatches one CLI invocation and returns the exit code;
// bare or -h invocations print the usage summary.
func runWithInput(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || (args[0] == "help" && len(args) == 1) {
		fmt.Fprintln(stdout, "orangecount check [--locale en|zh-CN] [--json] <entry.bean>")
		fmt.Fprintf(stdout, "orangecount serve [--addr %s] [--document-root DIR] <entry.bean>\n", defaultServeAddr)
		fmt.Fprintln(stdout, "orangecount query [--locale en|zh-CN] [--format json|csv] <entry.bean> <query>")
		fmt.Fprintln(stdout, "orangecount export -out FILE|DIR <entry.bean>")
		fmt.Fprintln(stdout, "orangecount dialectize -out FILE|DIR <entry.bean>")
		fmt.Fprintln(stdout, "orangecount help [--locale en|zh-CN] diagnostics/<CODE>")
		return 0
	}
	if args[0] == "version" || args[0] == "--version" {
		fmt.Fprintln(stdout, version)
		return 0
	}
	switch args[0] {
	case "help":
		return runHelp(args[1:], stdout, stderr)
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "serve":
		return runServe(args[1:], stdin, stdout, stderr)
	case "query":
		return runQuery(args[1:], stdout, stderr)
	case "export":
		return runExport(args[1:], stdout, stderr)
	case "dialectize":
		return runDialectize(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "orangecount: unknown command %q\n", args[0])
		return 2
	}
}

// runHelp implements the help subcommand for a diagnostic code's causes and
// remedies.
func runHelp(args []string, stdout, stderr io.Writer) int {
	args = normalizeHelpArgs(args)
	fs := flag.NewFlagSet("help", flag.ContinueOnError)
	fs.SetOutput(stderr)
	locale := fs.String("locale", "en", "help display locale (en or zh-CN)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *locale != repairguidance.LocaleEnglish && *locale != repairguidance.LocaleChinese {
		fmt.Fprintf(stderr, "orangecount help: unsupported locale %q\n", *locale)
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "orangecount help: expected a topic such as diagnostics/E-EVAL-UNBALANCED")
		return 2
	}
	topic := strings.TrimSpace(fs.Arg(0))
	if !strings.HasPrefix(topic, "diagnostics/") {
		fmt.Fprintf(stderr, "orangecount help: unknown topic %q\n", topic)
		return 1
	}
	code := strings.TrimPrefix(topic, "diagnostics/")
	guide, ok := repairguidance.Lookup(code, *locale)
	if !ok || guide.Topic != topic {
		if *locale == repairguidance.LocaleChinese {
			fmt.Fprintf(stderr, "orangecount help：找不到本地帮助主题 %q，请检查诊断代码。\n", topic)
		} else {
			fmt.Fprintf(stderr, "orangecount help: topic %q not found\n", topic)
		}
		return 1
	}
	return renderGuide(stdout, guide, *locale)
}

// normalizeHelpArgs keeps the help command friendly to both conventional Go
// flag ordering and the documented `help diagnostics/<CODE> --locale ...`
// spelling. The other subcommands retain their existing flag behavior.
func normalizeHelpArgs(args []string) []string {
	var flags, topics []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--locale" && i+1 < len(args) {
			flags = append(flags, args[i], args[i+1])
			i++
			continue
		}
		if strings.HasPrefix(args[i], "--locale=") {
			flags = append(flags, args[i])
			continue
		}
		topics = append(topics, args[i])
	}
	return append(flags, topics...)
}

func renderGuide(w io.Writer, guide repairguidance.Guide, locale string) int {
	labels := map[string][2]string{
		"topic":      {"Topic", "主题"},
		"what":       {"What happened", "发生了什么"},
		"why":        {"Why it blocks", "为什么阻塞"},
		"inspect":    {"Where to inspect", "去哪里检查"},
		"steps":      {"Safe checks and changes", "安全检查与修改"},
		"example":    {"Generic example", "通用示例"},
		"before":     {"Before", "修改前"},
		"after":      {"After", "修改后"},
		"note":       {"Note", "说明"},
		"revalidate": {"Next step", "下一步"},
	}
	label := func(key string) string {
		value := labels[key]
		if locale == repairguidance.LocaleChinese {
			return value[1]
		}
		return value[0]
	}
	fmt.Fprintf(w, "%s: %s\n\n", label("topic"), guide.Topic)
	fmt.Fprintf(w, "%s\n%s\n\n", label("what"), guide.What)
	fmt.Fprintf(w, "%s\n%s\n\n", label("why"), guide.Why)
	fmt.Fprintf(w, "%s\n", label("inspect"))
	for _, item := range guide.Inspect {
		fmt.Fprintf(w, "- %s\n", item)
	}
	fmt.Fprintf(w, "\n%s\n", label("steps"))
	for _, item := range guide.SafeSteps {
		fmt.Fprintf(w, "- %s\n", item)
	}
	fmt.Fprintf(w, "\n%s\n%s:\n%s\n%s:\n%s\n%s:\n%s\n\n%s\n%s\n", label("example"), label("before"), guide.Example.Before, label("after"), guide.Example.After, label("note"), guide.Example.Note, label("revalidate"), guide.Revalidate)
	return 0
}

// runQuery implements the query subcommand: load the ledger, evaluate one
// BeanQuery-shaped query, and print the result as JSON or CSV. Flag parsing
// and output rendering are extracted so the command body is the pipeline.
func runQuery(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("query", flag.ContinueOnError)
	fs.SetOutput(stderr)
	locale := fs.String("locale", "en", "diagnostic display locale (en or zh-CN)")
	format := fs.String("format", "json", "result format (json or csv)")
	jsonOutput := fs.Bool("json", false, "alias for --format json")
	csvOutput := fs.Bool("csv", false, "alias for --format csv")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	options, ok := validateQueryOptions(queryOptions{locale: *locale, format: *format}, *jsonOutput, *csvOutput, fs.NArg(), stderr)
	if !ok {
		return 2
	}
	result := snapshot.Build(fs.Arg(0))
	if result.Snapshot == nil || result.Err != nil || hasDiagnosticErrors(result.Diagnostics) {
		if err := diagnostic.RenderHuman(stderr, result.Diagnostics, options.locale); err != nil {
			return 1
		}
		return 1
	}
	value, err := query.Evaluate(fs.Arg(1), result.Snapshot.Evaluation())
	if err != nil {
		fmt.Fprintf(stderr, "orangecount query: %s\n", err)
		return 1
	}
	return writeQueryResult(stdout, stderr, value, options.format)
}

// queryOptions is the validated flag set of the query subcommand.
type queryOptions struct {
	locale string
	format string
}

// validateQueryOptions checks locale, the --json/--csv aliases, the output
// format, and the positional argument count. Misuse is reported to stderr
// the way flag.Parse would report it.
func validateQueryOptions(options queryOptions, jsonOutput, csvOutput bool, argc int, stderr io.Writer) (queryOptions, bool) {
	if options.locale != "en" && options.locale != "zh-CN" {
		fmt.Fprintf(stderr, "orangecount: unsupported locale %q\n", options.locale)
		return options, false
	}
	if jsonOutput && csvOutput {
		fmt.Fprintln(stderr, "orangecount query: --json and --csv cannot be combined")
		return options, false
	}
	switch {
	case jsonOutput:
		options.format = "json"
	case csvOutput:
		options.format = "csv"
	}
	if options.format != "json" && options.format != "csv" {
		fmt.Fprintf(stderr, "orangecount query: unsupported format %q\n", options.format)
		return options, false
	}
	if argc != 2 {
		fmt.Fprintln(stderr, "orangecount query: expected an entry ledger path and query text")
		return options, false
	}
	return options, true
}

// writeQueryResult renders an evaluated query result; the encoder mirrors
// the API layer (no HTML escaping) so both surfaces print identical JSON.
func writeQueryResult(stdout, stderr io.Writer, value query.Result, format string) int {
	if format == "csv" {
		if err := value.WriteCSV(stdout); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

// runCheck implements the check subcommand: parse and evaluate a ledger,
// render diagnostics in text, locale, or JSON form.
func runCheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	locale := fs.String("locale", "en", "diagnostic display locale (en or zh-CN)")
	jsonOutput := fs.Bool("json", false, "render diagnostics as JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *locale != "en" && *locale != "zh-CN" {
		fmt.Fprintf(stderr, "orangecount: unsupported locale %q\n", *locale)
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "orangecount check: expected exactly one entry ledger path")
		return 2
	}
	entry := fs.Arg(0)
	result := snapshot.Build(entry)
	ds := result.Diagnostics
	if *jsonOutput {
		if err := diagnostic.RenderJSON(stdout, ds, *locale); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	} else if err := renderCheckHuman(stdout, ds, *locale); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if result.Snapshot == nil || result.Err != nil || hasDiagnosticErrors(ds) {
		return 1
	}
	return 0
}

func renderCheckHuman(w io.Writer, ds []diagnostic.Diagnostic, locale string) error {
	if err := diagnostic.RenderHuman(w, ds, locale); err != nil {
		return err
	}
	for _, value := range ds {
		if value.Severity != diagnostic.Error {
			continue
		}
		guide, ok := repairguidance.Lookup(value.Code, locale)
		if !ok {
			continue
		}
		if _, err := fmt.Fprintf(w, "  -> %s (help: %s)\n", guide.ShortAction, guide.Topic); err != nil {
			return err
		}
	}
	return nil
}

// runServe implements the serve subcommand: build the initial snapshot, bind
// the loopback web server (prompting on port conflicts), then reload on
// source changes until shutdown.
// runServe implements the serve subcommand: build the initial snapshot, bind
// the loopback web server (prompting on port conflicts), then reload on
// source changes until shutdown. Each setup stage is a helper so the body is
// the serve pipeline in order.
func runServe(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	locale := fs.String("locale", "en", "display locale (en or zh-CN)")
	addr := fs.String("addr", defaultServeAddr, "loopback listen address")
	sensitiveLogs := fs.Bool("sensitive-logs", false, "temporarily include sensitive diagnostic fields in local logs")
	pollInterval := fs.Duration("poll", 250*time.Millisecond, "include graph polling interval")
	debounce := fs.Duration("debounce", 150*time.Millisecond, "reload debounce duration")
	var documentRoots []string
	fs.Func("document-root", "explicit document attachment root (repeatable)", func(value string) error {
		documentRoots = append(documentRoots, value)
		return nil
	})
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *locale != "en" && *locale != "zh-CN" {
		fmt.Fprintf(stderr, "orangecount: unsupported locale %q\n", *locale)
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "orangecount serve: expected exactly one entry ledger path")
		return 2
	}
	entry, initial, code := serveInitialSnapshot(fs.Arg(0), *locale, stdout, stderr)
	if initial == nil {
		return code
	}
	roots, err := source.NewDocumentRoots(documentRoots)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if code := resolveExistingPortConflict(*addr, stdin, stderr); code != 0 {
		return code
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	store := snapshot.NewStore(initial.Snapshot)
	server, serveDone, code := startServeLoop(ctx, store, roots, *addr, stdin, stderr)
	if server == nil {
		return code
	}
	watchLedger(ctx, store, entry, *pollInterval, *debounce, logging.New(stderr, logging.Options{Sensitive: *sensitiveLogs}))
	fmt.Fprintf(stdout, "serving on http://%s\n", server.Addr())
	if err := <-serveDone; err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

// serveInitialSnapshot builds and renders the startup snapshot; a nil result
// carries the command's exit code. The snapshot must be valid before the
// server can serve anything.
func serveInitialSnapshot(entry, locale string, stdout, stderr io.Writer) (string, *snapshot.BuildResult, int) {
	initial := snapshot.Build(entry)
	if err := renderDiagnostics(stdout, locale, false, initial.Diagnostics); err != nil {
		fmt.Fprintln(stderr, err)
		return entry, nil, 1
	}
	if initial.Snapshot == nil || initial.Err != nil || hasDiagnosticErrors(initial.Diagnostics) {
		return entry, nil, 1
	}
	return entry, &initial, 0
}

// resolveExistingPortConflict pre-checks the listen address for other owners
// and, when found, offers the interactive close-and-retry prompt. A non-zero
// return is the command's exit code; zero means continue serving.
func resolveExistingPortConflict(addr string, stdin io.Reader, stderr io.Writer) int {
	owners, inspectErr := portOwnersAt(addr)
	if inspectErr != nil || len(owners) == 0 {
		return 0
	}
	retry, promptErr := resolvePortConflict(addr, stdin, stderr)
	if promptErr != nil {
		fmt.Fprintln(stderr, promptErr)
		return 1
	}
	if !retry {
		return 1
	}
	return 0
}

// watchLedger reloads the include graph on source changes and logs each
// reload until the context is cancelled.
func watchLedger(ctx context.Context, store *snapshot.Store, entry string, pollInterval, debounce time.Duration, logger *logging.Logger) {
	go func() {
		_ = store.Watch(ctx, entry, snapshot.BuildOptions{}, snapshot.WatchOptions{PollInterval: pollInterval, Debounce: debounce}, func(result snapshot.ReloadResult) {
			_ = logger.Event("reload", map[string]any{"reload": true, "published": result.Published, "snapshot_id": snapshotID(result.Snapshot)})
		})
	}()
}

// startServeLoop binds the web server, retrying after an interactive port
// conflict resolution. It returns the running server with its done channel,
// or an exit code when serving could not start (server nil).
func startServeLoop(ctx context.Context, store *snapshot.Store, roots source.DocumentRoots, addr string, stdin io.Reader, stderr io.Writer) (*web.Server, <-chan error, int) {
	for {
		candidate, err := web.NewServer(web.Config{Store: store, DocumentRoots: roots, Addr: addr})
		if err != nil {
			fmt.Fprintln(stderr, err)
			return nil, nil, 2
		}
		done := make(chan error, 1)
		go func() { done <- candidate.Serve(ctx) }()
		readyErr := candidate.WaitReady(ctx)
		if readyErr == nil {
			return candidate, done, 0
		}
		if ctx.Err() != nil {
			return nil, nil, 0
		}
		<-done
		if errors.Is(readyErr, syscall.EADDRINUSE) {
			retry, promptErr := resolvePortConflict(addr, stdin, stderr)
			if promptErr != nil {
				fmt.Fprintln(stderr, promptErr)
				return nil, nil, 1
			}
			if retry {
				continue
			}
		}
		fmt.Fprintln(stderr, readyErr)
		return nil, nil, 1
	}
}

// resolvePortConflict handles a busy serve address: it lists the owning
// processes and, when input is available, asks whether to close them; true
// means the port was freed and serving should retry.
func resolvePortConflict(addr string, stdin io.Reader, stderr io.Writer) (bool, error) {
	owners, err := portOwnersAt(addr)
	_, port, splitErr := net.SplitHostPort(addr)
	if splitErr != nil {
		return false, nil
	}
	if err != nil || len(owners) == 0 {
		if err != nil {
			fmt.Fprintf(stderr, "orangecount: port %s is already in use (unable to identify the owner: %v)\n", port, err)
		} else {
			fmt.Fprintf(stderr, "orangecount: port %s is already in use\n", port)
		}
		return false, nil
	}
	fmt.Fprintf(stderr, "orangecount: port %s is currently used by:\n", port)
	for _, owner := range owners {
		fmt.Fprintf(stderr, "  PID %d: %s\n", owner.PID, owner.Command)
	}
	if stdin == nil {
		fmt.Fprintln(stderr, "orangecount: run interactively to choose whether to close it and start OrangeCount.")
		return false, nil
	}
	fmt.Fprint(stderr, "Close the listed process(es) and start OrangeCount? [y/N] ")
	answer, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer != "y" && answer != "yes" && answer != "是" {
		fmt.Fprintln(stderr, "OrangeCount was not started.")
		return false, nil
	}
	for _, owner := range owners {
		if err := stopPortOwner(owner.PID); err != nil {
			return false, fmt.Errorf("could not stop PID %d (%s): %w", owner.PID, owner.Command, err)
		}
	}
	if err := waitForPort(addr, 3*time.Second); err != nil {
		return false, err
	}
	return true, nil
}

func portOwnersAt(addr string) ([]portOwner, error) {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	return inspectPortOwners(port)
}

func lsofPortOwners(port string) ([]portOwner, error) {
	output, err := runLsof(port)
	if err != nil {
		return nil, err
	}
	return parsePortOwners(string(output)), nil
}

// parsePortOwners decodes lsof's stable process/command record format. It is
// deliberately independent from process execution so the platform-specific
// inspection adapter remains small and the parsing contract is directly
// testable.
func parsePortOwners(output string) []portOwner {
	owners := make([]portOwner, 0)
	var current *portOwner
	seen := make(map[int]bool)
	for _, line := range strings.Split(output, "\n") {
		if len(line) < 2 {
			continue
		}
		switch line[0] {
		case 'p':
			pid, parseErr := strconv.Atoi(line[1:])
			if parseErr != nil || seen[pid] {
				current = nil
				continue
			}
			owners = append(owners, portOwner{PID: pid, Command: "unknown process"})
			current = &owners[len(owners)-1]
			seen[pid] = true
		case 'c':
			if current != nil && strings.TrimSpace(line[1:]) != "" {
				current.Command = strings.TrimSpace(line[1:])
			}
		}
	}
	return owners
}

func terminatePortOwner(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Signal(syscall.SIGTERM)
}

func waitForPortRelease(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			return listener.Close()
		}
		if !errors.Is(err, syscall.EADDRINUSE) {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("port %s is still in use after waiting %s", addr, timeout)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func snapshotID(value *snapshot.Snapshot) string {
	if value == nil {
		return ""
	}
	return value.ID
}

func hasDiagnosticErrors(ds []diagnostic.Diagnostic) bool {
	for _, d := range ds {
		if d.Severity == diagnostic.Error {
			return true
		}
	}
	return false
}

func renderDiagnostics(w io.Writer, locale string, jsonOutput bool, ds []diagnostic.Diagnostic) error {
	if jsonOutput {
		return diagnostic.RenderJSON(w, ds, locale)
	}
	return diagnostic.RenderHuman(w, ds, locale)
}

// runExport implements the export subcommand: compile dialect shorthand in a
// ledger and write the pure Beancount v3 artifact for external tools. The
// output is a disposable snapshot (ADR-0045): it must never be hand-edited.
func runExport(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("out", "", "output file (single-file ledger) or directory (include graph)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "orangecount export: expected exactly one entry ledger path")
		return 2
	}
	if *out == "" {
		fmt.Fprintln(stderr, "orangecount export: -out is required")
		return 2
	}
	return writeConverted(fs.Arg(0), *out, "export", stdout, stderr, false)
}

// runDialectize implements the dialectize subcommand: rewrite a standard v3
// ledger's trivially-representable transactions into dialect shorthand while
// preserving every other byte (ADR-0045 filter conversion).
func runDialectize(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dialectize", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("out", "", "output file (single-file ledger) or directory (include graph)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "orangecount dialectize: expected exactly one entry ledger path")
		return 2
	}
	if *out == "" {
		fmt.Fprintln(stderr, "orangecount dialectize: -out is required")
		return 2
	}
	return writeConverted(fs.Arg(0), *out, "dialectize", stdout, stderr, true)
}

// writeConverted is the shared body of export and dialectize: load the graph,
// render every file's converted text, and mirror the include structure under
// the output path. A single-file ledger writes one file; an include graph
// writes a directory tree that preserves relative include paths.
func writeConverted(entry, out, label string, stdout, stderr io.Writer, reverse bool) int {
	graph, err := source.LoadGraph(entry)
	if err != nil {
		fmt.Fprintf(stderr, "orangecount %s: %v\n", label, err)
		return 1
	}
	parsed, bag := ledger.ParseGraph(graph)
	if bag.HasErrors() {
		renderBag(stderr, bag, label)
		return 1
	}
	outputs := make(map[source.FileID][]byte)
	if reverse {
		for fileID, file := range parsed {
			if file == nil || file.Source == nil {
				continue
			}
			edits, _ := dialect.Dialectize(file)
			outputs[fileID] = dialect.ApplyEdits(file.Source.Data, edits)
		}
	} else {
		rendered, diags := dialect.ExportText(graph, parsed)
		for _, d := range diags {
			if d.Severity == diagnostic.Error {
				fmt.Fprintf(stderr, "orangecount %s: %s %s\n", label, d.Code, d.Message)
				return 1
			}
		}
		outputs = rendered
	}
	return writeConvertedFiles(graph, outputs, out, label, stdout, stderr)
}

// writeConvertedFiles mirrors the include graph under the output path: a
// single-file ledger writes one file, an include graph preserves relative
// paths inside a directory tree.
func writeConvertedFiles(graph *source.Graph, outputs map[source.FileID][]byte, out, label string, stdout, stderr io.Writer) int {
	fail := func(err error) int {
		fmt.Fprintf(stderr, "orangecount %s: %v\n", label, err)
		return 1
	}
	if len(graph.Order) == 1 {
		if dir := filepath.Dir(out); dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fail(err)
			}
		}
		if err := os.WriteFile(out, outputs[graph.Entry], 0o600); err != nil {
			return fail(err)
		}
		fmt.Fprintf(stdout, "orangecount %s: wrote %s\n", label, out)
		return 0
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return fail(err)
	}
	entryDir := filepath.Dir(graph.Path(graph.Entry))
	for fileID, data := range outputs {
		rel, err := filepath.Rel(entryDir, graph.Path(fileID))
		if err != nil {
			return fail(err)
		}
		target := filepath.Join(out, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fail(err)
		}
		if err := os.WriteFile(target, data, 0o600); err != nil {
			return fail(err)
		}
	}
	fmt.Fprintf(stdout, "orangecount %s: wrote %d files under %s\n", label, len(outputs), out)
	return 0
}

func renderBag(w io.Writer, bag *diagnostic.Bag, label string) {
	for _, d := range bag.All() {
		fmt.Fprintf(w, "orangecount %s: %s %s\n", label, d.Code, d.Message)
	}
}
