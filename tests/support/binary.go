package support

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

var (
	buildOnce sync.Once
	buildPath string
	buildErr  error
)

const versionPackage = "github.com/catu-ai/easyharness/internal/version"

func RepoRoot(t *testing.T) string {
	t.Helper()
	return repoRoot()
}

func BuildBinary(t *testing.T) string {
	t.Helper()

	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "easyharness-harness-*")
		if err != nil {
			buildErr = fmt.Errorf("create temporary binary directory: %w", err)
			return
		}

		name := "harness"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		buildPath = filepath.Join(dir, name)

		commit, err := repoHeadCommit(repoRoot())
		if err != nil {
			buildErr = fmt.Errorf("resolve harness build commit: %w", err)
			return
		}

		version, err := repoReleaseVersion(repoRoot())
		if err != nil {
			buildErr = fmt.Errorf("resolve harness build version: %w", err)
			return
		}

		ldflags := fmt.Sprintf("-X %s.BuildCommit=%s -X %s.BuildVersion=%s", versionPackage, commit, versionPackage, version)
		output, err := runOutputWithTimeout(repoRoot(), "go", "build", "-ldflags", ldflags, "-o", buildPath, "./cmd/harness")
		if err != nil {
			buildErr = fmt.Errorf("build harness binary: %w\n%s", err, output)
		}
	})

	if buildErr != nil {
		t.Fatalf("build harness binary: %v", buildErr)
	}
	return buildPath
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve tests/support source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func repoHeadCommit(root string) (string, error) {
	output, err := runOutputWithTimeout(root, "git", "-C", root, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %w\n%s", err, output)
	}

	commit := strings.TrimSpace(string(output))
	if commit == "" {
		return "", fmt.Errorf("git rev-parse HEAD returned an empty commit")
	}
	return commit, nil
}

func repoReleaseVersion(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		return "", fmt.Errorf("read VERSION: %w", err)
	}
	version := strings.TrimSpace(string(data))
	if version == "" {
		return "", fmt.Errorf("VERSION file is empty")
	}
	return "v" + version, nil
}

func runOutputWithTimeout(workdir string, argv ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = workdir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return output, fmt.Errorf("run command %v timed out after %s", argv, DefaultCommandTimeout)
	}
	if err != nil {
		return output, fmt.Errorf("run command %v: %w", argv, err)
	}
	return output, nil
}
