// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

// Command queryparity reads reviewed public goldens and fails on any mismatch.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"orangecount/internal/compat"
)

func main() {
	golden := flag.String("golden", "", "single reviewed public query golden (defaults to all registered corpora)")
	flag.Parse()
	var result compat.QueryParity
	var err error
	if *golden == "" {
		result, err = compat.CheckQueryCorpus(".")
	} else {
		result, err = compat.CheckQueryGolden(".", *golden)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "query parity:", err)
		os.Exit(2)
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		os.Exit(2)
	}
	if result.Failed != 0 {
		os.Exit(1)
	}
}
