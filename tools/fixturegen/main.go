// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"flag"
	"fmt"
	"os"

	"orangecount/tools/fixturegen/gen"
)

func main() {
	output := flag.String("output", "testdata/fixtures/fava-reference", "fixture output directory")
	flag.Parse()
	result, err := gen.Generate(*output, gen.DefaultConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fixturegen: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("fixturegen: wrote %d files, %d accounts, %d transactions to %s\n", len(result.Files), result.Accounts, result.Transactions, *output)
}
