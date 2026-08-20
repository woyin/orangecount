// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

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
