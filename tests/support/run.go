package support

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

type RunOptions struct {
	Workdir string
	Args    []string
	Stdin   string
	Env     []string
	Timeout time.Duration
}

type Result struct {
	Args     []string
	Stdout   string
	Stderr   string
	ExitCode int
}

func (r Result) CombinedOutput() string {
	return r.Stdout + r.Stderr
}

func Run(t *testing.T, workdir string, args ...string) Result {
	t.Helper()
	return RunWithOptions(t, RunOptions{
		Workdir: workdir,
		Args:    args,
	})
}

func RunWithOptions(t *testing.T, opts RunOptions) Result {
	t.Helper()

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = DefaultCommandTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, BuildBinary(t), opts.Args...)
	cmd.Dir = opts.Workdir
	cmd.Env = append(os.Environ(), opts.Env...)
	cmd.Stdin = strings.NewReader(opts.Stdin)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := Result{
		Args:   append([]string(nil), opts.Args...),
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	if err == nil {
		return result
	}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("run harness %v timed out after %s\nstdout:\n%s\nstderr:\n%s", opts.Args, timeout, stdout.String(), stderr.String())
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("run harness %v: %v\nstdout:\n%s\nstderr:\n%s", opts.Args, err, stdout.String(), stderr.String())
	}
	result.ExitCode = exitErr.ExitCode()
	return result
}
