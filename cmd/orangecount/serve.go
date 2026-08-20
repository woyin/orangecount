// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"orangecount/internal/logging"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
	"orangecount/internal/web"
)

func runServe(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	locale := fs.String("locale", "en", "display locale (en or zh-CN)")
	addr := fs.String("addr", defaultServeAddr, "loopback listen address")
	sensitiveLogs := fs.Bool("sensitive-logs", false, "temporarily include sensitive diagnostic fields in local logs")
	pollInterval := fs.Duration("poll", 250*time.Millisecond, "include graph polling interval")
	debounce := fs.Duration("debounce", 150*time.Millisecond, "reload debounce duration")
	var documentRoots []string
	fs.Func("document-root", "explicit document attachment root (repeatable)", func(value string) error { documentRoots = append(documentRoots, value); return nil })
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

func watchLedger(ctx context.Context, store *snapshot.Store, entry string, pollInterval, debounce time.Duration, logger *logging.Logger) {
	go func() {
		_ = store.Watch(ctx, entry, snapshot.BuildOptions{}, snapshot.WatchOptions{PollInterval: pollInterval, Debounce: debounce}, func(result snapshot.ReloadResult) {
			_ = logger.Event("reload", map[string]any{"reload": true, "published": result.Published, "snapshot_id": snapshotID(result.Snapshot)})
		})
	}()
}

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
