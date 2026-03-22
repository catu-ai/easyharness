package support

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func RequireExitCode(t *testing.T, result Result, want int) {
	t.Helper()
	if result.ExitCode != want {
		t.Fatalf("expected exit code %d, got %d\nstdout:\n%s\nstderr:\n%s", want, result.ExitCode, result.Stdout, result.Stderr)
	}
}

func RequireSuccess(t *testing.T, result Result) {
	t.Helper()
	RequireExitCode(t, result, 0)
}

func RequireContains(t *testing.T, actual, fragment string) {
	t.Helper()
	if !strings.Contains(actual, fragment) {
		t.Fatalf("expected %q to contain %q", actual, fragment)
	}
}

func RequireNoStderr(t *testing.T, result Result) {
	t.Helper()
	if strings.TrimSpace(result.Stderr) != "" {
		t.Fatalf("expected empty stderr, got:\n%s", result.Stderr)
	}
}

func RequireFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
}

func RequireFileMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("expected %s to be absent", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}

func RequireJSONResult[T any](t *testing.T, result Result) T {
	t.Helper()
	var value T
	if err := json.Unmarshal([]byte(result.Stdout), &value); err != nil {
		t.Fatalf("decode json stdout: %v\nstdout:\n%s\nstderr:\n%s", err, result.Stdout, result.Stderr)
	}
	return value
}

func ReadJSONFile[T any](t *testing.T, path string) T {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return value
}
