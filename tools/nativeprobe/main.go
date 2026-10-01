// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

// Command nativeprobe measures watched reloads on a temporary copy of a
// synthetic single-file ledger. Its trace separates build and publication
// work from file detection, debounce and caller observation.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/trace"
	"time"

	"orangecount/internal/desktop"
	"orangecount/internal/snapshot"
)

func main() {
	ledger := flag.String("ledger", "", "synthetic single-file ledger to copy")
	tracePath := flag.String("trace", "", "execution trace output")
	cycles := flag.Int("cycles", 5, "number of watched edits (1...100)")
	flag.Parse()
	if err := run(*ledger, *tracePath, *cycles); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(path, tracePath string, cycles int) error {
	if path == "" || tracePath == "" || cycles < 1 || cycles > 100 {
		return fmt.Errorf("provide -ledger, -trace and -cycles between 1 and 100")
	}
	base, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	directory, err := os.MkdirTemp("", "orangecount-watch-probe-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	entry := filepath.Join(directory, "main.bean")
	if err := os.WriteFile(entry, base, 0600); err != nil {
		return err
	}
	output, err := os.Create(tracePath)
	if err != nil {
		return err
	}
	defer output.Close()
	if err := trace.Start(output); err != nil {
		return err
	}
	defer trace.Stop()
	s, err := desktop.Open(entry, snapshot.WatchOptions{})
	if err != nil {
		return err
	}
	defer s.Close()
	state, err := s.Poll()
	if err != nil || !state.Valid {
		return fmt.Errorf("probe requires a valid self-contained synthetic ledger: %v", err)
	}
	for cycle := 1; cycle <= cycles; cycle++ {
		data := append(append([]byte(nil), base...), []byte(fmt.Sprintf("\n; native probe edit %d\n", cycle))...)
		if err := os.WriteFile(entry+".new", data, 0600); err != nil {
			return err
		}
		if err := os.Rename(entry+".new", entry); err != nil {
			return err
		}
		trace.Log(context.Background(), "native.edit", "saved")
		start := time.Now()
		previous := state
		for {
			state, err = s.Poll()
			if err != nil {
				return err
			}
			if state.BuildCount > previous.BuildCount && !state.Reloading {
				break
			}
			if time.Since(start) > 10*time.Second {
				return fmt.Errorf("edit %d did not publish within 10 seconds", cycle)
			}
			time.Sleep(time.Millisecond)
		}
		trace.Log(context.Background(), "native.observed", "ready")
		if !state.Valid || state.SnapshotID == previous.SnapshotID || state.BuildCount != previous.BuildCount+1 {
			return fmt.Errorf("edit %d must produce exactly one valid new snapshot", cycle)
		}
		fmt.Printf("edit=%d total_saved_to_observed_ms=%.3f build_count=%d\n", cycle, float64(time.Since(start))/float64(time.Millisecond), state.BuildCount)
	}
	return nil
}
