package main

import (
	"os"

	"github.com/yzhang1918/microharness/internal/cli"
)

func main() {
	app := cli.New(os.Stdout, os.Stderr)
	os.Exit(app.Run(os.Args[1:]))
}
