package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/catu-ai/easyharness/internal/bootstrapsync"
)

func main() {
	var workdir string
	var check bool

	flag.StringVar(&workdir, "workdir", ".", "repository root to sync or check")
	flag.BoolVar(&check, "check", false, "fail when tracked dogfood outputs drift from assets/bootstrap")
	flag.Parse()

	var err error
	if check {
		_, err = bootstrapsync.Check(workdir)
		if err == nil {
			fmt.Fprintln(os.Stdout, "Bootstrap dogfood outputs are in sync with assets/bootstrap.")
			return
		}
	} else {
		result, syncErr := bootstrapsync.Sync(workdir)
		err = syncErr
		if err == nil {
			fmt.Fprintln(os.Stdout, result.Summary)
			return
		}
	}

	var driftErr *bootstrapsync.DriftError
	if errors.As(err, &driftErr) {
		fmt.Fprintln(os.Stderr, driftErr.Error())
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
