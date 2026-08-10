// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"orangecount/tools/fixturegen/gen"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run keeps command-line transport separate from fixture generation so the
// generator's command contract can be tested without writing to real stdout
// or terminating the test process.
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("fixturegen", flag.ContinueOnError)
	flags.SetOutput(stderr)
	output := flags.String("output", "testdata/fixtures/fava-reference", "fixture output directory")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	result, err := gen.Generate(*output, gen.DefaultConfig)
	if err != nil {
		fmt.Fprintf(stderr, "fixturegen: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "fixturegen: wrote %d files, %d accounts, %d transactions to %s\n", len(result.Files), result.Accounts, result.Transactions, *output)
	return 0
}
