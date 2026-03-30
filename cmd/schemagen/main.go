package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/catu-ai/easyharness/internal/contracts"
)

func main() {
	outputDir := flag.String("output-dir", "docs/schemas", "Directory to write generated JSON Schemas into.")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "Usage: schemagen [--output-dir <dir>]")
		os.Exit(2)
	}
	if err := contracts.GenerateSchemaFiles(*outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "generate schemas: %v\n", err)
		os.Exit(1)
	}
}
