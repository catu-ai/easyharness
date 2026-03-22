package support

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

var (
	buildOnce sync.Once
	buildPath string
	buildErr  error
)

func RepoRoot(t *testing.T) string {
	t.Helper()
	return repoRoot()
}

func BuildBinary(t *testing.T) string {
	t.Helper()

	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "superharness-harness-*")
		if err != nil {
			buildErr = fmt.Errorf("create temporary binary directory: %w", err)
			return
		}

		name := "harness"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		buildPath = filepath.Join(dir, name)

		cmd := exec.Command("go", "build", "-o", buildPath, "./cmd/harness")
		cmd.Dir = repoRoot()
		output, err := cmd.CombinedOutput()
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
