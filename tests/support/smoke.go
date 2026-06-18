package support

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

const DefaultCommandTimeout = 5 * time.Minute

func (r CommandResult) CombinedOutput() string {
	return r.Stdout + r.Stderr
}

func RunCommand(t *testing.T, workdir string, env []string, argv ...string) CommandResult {
	t.Helper()

	return RunCommandWithTimeout(t, DefaultCommandTimeout, workdir, env, argv...)
}

func RunCommandWithTimeout(t *testing.T, timeout time.Duration, workdir string, env []string, argv ...string) CommandResult {
	t.Helper()

	var (
		cmd    *exec.Cmd
		cancel func()
		ctx    context.Context
	)
	if timeout == 0 {
		timeout = DefaultCommandTimeout
	}
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
		cmd = exec.CommandContext(ctx, argv[0], argv[1:]...)
	} else {
		cmd = exec.Command(argv[0], argv[1:]...)
	}

	cmd.Dir = workdir
	cmd.Env = env

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := CommandResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	if err == nil {
		return result
	}

	if ctx != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("run command %v timed out after %s\nstdout:\n%s\nstderr:\n%s", argv, timeout, stdout.String(), stderr.String())
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		if errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("run command %v timed out after %s\nstdout:\n%s\nstderr:\n%s", argv, timeout, stdout.String(), stderr.String())
		}
		t.Fatalf("run command %v: %v", argv, err)
	}
	result.ExitCode = exitErr.ExitCode()
	return result
}

func CopyInstallerFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	sourceRoot := RepoRoot(t)
	for _, rel := range []string{
		"VERSION",
		"go.mod",
		"go.sum",
		"assets",
		"cmd",
		"internal",
		"scripts/build-embedded-ui",
		"scripts/install-dev-harness",
	} {
		CopyPath(t, filepath.Join(sourceRoot, rel), filepath.Join(root, rel))
	}
	for _, rel := range []string{
		"web/index.html",
		"web/package.json",
		"web/pnpm-lock.yaml",
		"web/tsconfig.json",
		"web/vite.config.ts",
		"web/src",
	} {
		CopyPath(t, filepath.Join(sourceRoot, rel), filepath.Join(root, rel))
	}
	_ = os.RemoveAll(filepath.Join(root, "internal", "ui", "generated", "build"))
	return root
}

func EnvWithOverrides(t *testing.T, overrides map[string]string) []string {
	t.Helper()

	env := os.Environ()
	for key, value := range overrides {
		prefix := key + "="
		replaced := false
		for i, entry := range env {
			if strings.HasPrefix(entry, prefix) {
				env[i] = prefix + value
				replaced = true
				break
			}
		}
		if !replaced {
			env = append(env, prefix+value)
		}
	}
	return env
}

func InstallerEnv(t *testing.T, overrides map[string]string) []string {
	t.Helper()

	if overrides == nil {
		overrides = map[string]string{}
	}
	if _, ok := overrides["GOCACHE"]; !ok {
		overrides["GOCACHE"] = sharedInstallerGoCache(t)
	}
	if _, ok := overrides["GOMODCACHE"]; !ok {
		overrides["GOMODCACHE"] = sharedInstallerGoModCache(t)
	}
	if _, ok := overrides["GOFLAGS"]; !ok {
		overrides["GOFLAGS"] = "-modcacherw"
	}
	return EnvWithOverrides(t, overrides)
}

func InitializeInstallerCaches(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("install-dev-harness smoke test skipped in short mode")
	}

	initializeSharedInstallerCaches()
	if installerCacheErr != nil {
		t.Fatalf("initialize shared installer caches: %v", installerCacheErr)
	}
}

func InstallerPath(t *testing.T, extraDirs ...string) string {
	t.Helper()
	InitializeInstallerCaches(t)

	var dirs []string
	seen := make(map[string]bool)
	addDir := func(dir string) {
		if dir == "" || seen[dir] {
			return
		}
		seen[dir] = true
		dirs = append(dirs, dir)
	}

	for _, dir := range extraDirs {
		addDir(dir)
	}
	addDir(filepath.Join(RepoRoot(t), "node_modules", ".bin"))
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		addDir(dir)
	}
	addDir("/usr/bin")
	addDir("/bin")
	addDir("/usr/sbin")
	addDir("/sbin")

	return strings.Join(dirs, string(os.PathListSeparator))
}

func ReleaseSmokePath(t *testing.T, extraToolPaths ...string) string {
	t.Helper()

	toolPaths := make([]string, 0, len(extraToolPaths)+2)
	if goPath, err := exec.LookPath("go"); err == nil {
		toolPaths = append(toolPaths, goPath)
	} else {
		t.Fatalf("find go on PATH: %v", err)
	}
	if pnpmPath, err := exec.LookPath("pnpm"); err == nil {
		toolPaths = append(toolPaths, pnpmPath)
	} else {
		t.Fatalf("find pnpm on PATH: %v", err)
	}
	toolPaths = append(toolPaths, extraToolPaths...)

	return ReleaseSmokePathFromToolPaths(os.Getenv("PATH"), toolPaths...)
}

func ReleaseSmokePathFromToolPaths(currentPath string, toolPaths ...string) string {
	var dirs []string
	seen := make(map[string]bool)
	addDir := func(dir string) {
		if dir == "" || seen[dir] {
			return
		}
		seen[dir] = true
		dirs = append(dirs, dir)
	}

	for _, toolPath := range toolPaths {
		addDir(filepath.Dir(toolPath))
	}
	addDir(filepath.Join(repoRoot(), "node_modules", ".bin"))
	for _, dir := range filepath.SplitList(currentPath) {
		addDir(dir)
	}
	addDir("/usr/bin")
	addDir("/bin")
	addDir("/usr/sbin")
	addDir("/sbin")

	return strings.Join(dirs, string(os.PathListSeparator))
}

func GitHeadCommit(t *testing.T, repoRoot string) string {
	t.Helper()

	commit, err := repoHeadCommit(repoRoot)
	if err != nil {
		t.Fatalf("%v", err)
	}
	return commit
}

func RequireVersionField(t *testing.T, output, field string) string {
	t.Helper()

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("expected JSON version output: %v\noutput:\n%s", err, output)
	}
	value, ok := payload[field]
	if !ok {
		t.Fatalf("expected version field %q in output:\n%s", field, output)
	}
	switch typed := value.(type) {
	case string:
		if typed == "" {
			t.Fatalf("expected version field %q to be non-empty\noutput:\n%s", field, output)
		}
		return typed
	case bool:
		return fmt.Sprintf("%t", typed)
	default:
		return fmt.Sprint(typed)
	}
}

func RequireSubstringOrder(t *testing.T, haystack, first, second string) {
	t.Helper()
	firstIndex := strings.Index(haystack, first)
	if firstIndex < 0 {
		t.Fatalf("expected workflow to contain %q", first)
	}
	secondIndex := strings.Index(haystack, second)
	if secondIndex < 0 {
		t.Fatalf("expected workflow to contain %q", second)
	}
	if firstIndex > secondIndex {
		t.Fatalf("expected %q to appear before %q", first, second)
	}
}

var (
	installerCacheOnce sync.Once
	installerGoCache   string
	installerModCache  string
	installerCacheErr  error
)

func sharedInstallerGoCache(t *testing.T) string {
	t.Helper()
	InitializeInstallerCaches(t)
	return installerGoCache
}

func sharedInstallerGoModCache(t *testing.T) string {
	t.Helper()
	InitializeInstallerCaches(t)
	return installerModCache
}

func initializeSharedInstallerCaches() {
	installerCacheOnce.Do(func() {
		root, err := os.MkdirTemp("", "easyharness-install-smoke-cache-*")
		if err != nil {
			installerCacheErr = err
			return
		}
		installerGoCache = filepath.Join(root, "go-build")
		installerModCache = filepath.Join(root, "gomod")
		for _, dir := range []string{installerGoCache, installerModCache} {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				installerCacheErr = err
				return
			}
		}
	})
}

func WriteFixtureFile(t *testing.T, path, contents string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func CopyPath(t *testing.T, src, dst string) {
	t.Helper()

	info, err := os.Stat(src)
	if err != nil {
		t.Fatalf("stat %s: %v", src, err)
	}

	if info.IsDir() {
		if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
			t.Fatalf("mkdir %s: %v", dst, err)
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			t.Fatalf("read dir %s: %v", src, err)
		}
		for _, entry := range entries {
			CopyPath(t, filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name()))
		}
		return
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("mkdir parent for %s: %v", dst, err)
	}

	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("open %s: %v", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		t.Fatalf("create %s: %v", dst, err)
	}
	defer out.Close()

	if _, err := out.ReadFrom(in); err != nil {
		t.Fatalf("copy %s to %s: %v", src, dst, err)
	}
}
