package watchlist

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	envHome = "EASYHARNESS_HOME"
	version = 1
)

var ErrNotGitWorkspace = errors.New("workspace is not git-backed")

type Service struct {
	LookupEnv   func(string) (string, bool)
	UserHomeDir func() (string, error)
	Now         func() time.Time
}

type File struct {
	Version    int         `json:"version"`
	Workspaces []Workspace `json:"workspaces"`
}

type Workspace struct {
	WorkspacePath string `json:"workspace_path"`
	WatchedAt     string `json:"watched_at"`
	LastSeenAt    string `json:"last_seen_at"`
}

func (s Service) Read() (File, error) {
	home, err := s.easyharnessHome()
	if err != nil {
		return File{}, err
	}
	return loadFile(filepath.Join(home, "watchlist.json"))
}

func (s Service) Touch(workdir string) error {
	if strings.TrimSpace(workdir) == "" {
		return fmt.Errorf("resolve workspace: empty path")
	}

	home, err := s.easyharnessHome()
	if err != nil {
		return err
	}
	canonicalPath, err := canonicalWorkspacePath(workdir)
	if err != nil {
		return err
	}

	release, err := acquireLock(home)
	if err != nil {
		return err
	}
	defer release()

	path := filepath.Join(home, "watchlist.json")
	data, err := loadFile(path)
	if err != nil {
		return err
	}

	now := s.now().UTC().Format(time.RFC3339)
	data.Workspaces = upsertWorkspace(data.Workspaces, canonicalPath, now)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal watchlist.json: %w", err)
	}
	return writeJSONAtomic(path, payload, 0o644)
}

func (s Service) easyharnessHome() (string, error) {
	if lookup := s.LookupEnv; lookup != nil {
		if value, ok := lookup(envHome); ok && strings.TrimSpace(value) != "" {
			return s.resolveConfiguredHome(value)
		}
	} else if value := strings.TrimSpace(os.Getenv(envHome)); value != "" {
		return s.resolveConfiguredHome(value)
	}

	home, err := s.userHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".easyharness"), nil
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s Service) resolveConfiguredHome(value string) (string, error) {
	candidate := filepath.Clean(strings.TrimSpace(value))
	if filepath.IsAbs(candidate) {
		return candidate, nil
	}
	home, err := s.userHomeDir()
	if err != nil {
		return "", err
	}
	resolved := filepath.Clean(filepath.Join(home, candidate))
	relative, err := filepath.Rel(home, resolved)
	if err != nil {
		return "", fmt.Errorf("resolve configured home: %w", err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("resolve configured home: relative path escapes user home")
	}
	return resolved, nil
}

func (s Service) userHomeDir() (string, error) {
	var (
		home string
		err  error
	)
	if s.UserHomeDir != nil {
		home, err = s.UserHomeDir()
	} else {
		home, err = os.UserHomeDir()
	}
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	if strings.TrimSpace(home) == "" {
		return "", fmt.Errorf("resolve user home: empty path")
	}
	return filepath.Clean(strings.TrimSpace(home)), nil
}

func loadFile(path string) (File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return File{Version: version, Workspaces: []Workspace{}}, nil
		}
		return File{}, err
	}

	var decoded File
	if err := json.Unmarshal(data, &decoded); err != nil {
		return File{}, fmt.Errorf("parse watchlist.json: %w", err)
	}
	switch decoded.Version {
	case 0:
		return File{}, fmt.Errorf("parse watchlist.json: missing version")
	case version:
	default:
		return File{}, fmt.Errorf("parse watchlist.json: unsupported version %d", decoded.Version)
	}
	if decoded.Workspaces == nil {
		decoded.Workspaces = []Workspace{}
	}
	return decoded, nil
}

func upsertWorkspace(workspaces []Workspace, canonicalPath, seenAt string) []Workspace {
	next := make([]Workspace, 0, len(workspaces)+1)
	indexByPath := make(map[string]int, len(workspaces)+1)
	merged := false
	for _, workspace := range workspaces {
		workspace.WorkspacePath = strings.TrimSpace(workspace.WorkspacePath)
		if workspace.WorkspacePath == canonicalPath {
			if strings.TrimSpace(workspace.WatchedAt) == "" {
				workspace.WatchedAt = seenAt
			}
			workspace.LastSeenAt = laterTimestamp(workspace.LastSeenAt, seenAt)
			merged = true
		}

		if idx, ok := indexByPath[workspace.WorkspacePath]; ok {
			existing := &next[idx]
			existing.WatchedAt = earlierTimestamp(existing.WatchedAt, workspace.WatchedAt)
			existing.LastSeenAt = laterTimestamp(existing.LastSeenAt, workspace.LastSeenAt)
			continue
		}

		next = append(next, workspace)
		indexByPath[workspace.WorkspacePath] = len(next) - 1
	}
	if !merged {
		next = append(next, Workspace{
			WorkspacePath: canonicalPath,
			WatchedAt:     seenAt,
			LastSeenAt:    seenAt,
		})
	}
	return next
}

func canonicalWorkspacePath(workdir string) (string, error) {
	root, err := gitWorkspaceRoot(workdir)
	if err != nil {
		return "", err
	}
	absolute, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return "", fmt.Errorf("resolve workspace absolute path: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve workspace symlinks: %w", err)
	}
	return filepath.Clean(resolved), nil
}

func gitWorkspaceRoot(workdir string) (string, error) {
	workdir = strings.TrimSpace(workdir)
	if workdir == "" {
		return "", fmt.Errorf("resolve workspace: empty path")
	}
	current, err := filepath.Abs(workdir)
	if err != nil {
		return "", fmt.Errorf("resolve workspace absolute path: %w", err)
	}
	output, err := exec.Command("git", "-C", filepath.Clean(current), "rev-parse", "--show-toplevel").CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", ErrNotGitWorkspace
		}
		return "", fmt.Errorf("detect git workspace root: %w", err)
	}
	root := strings.TrimSpace(string(output))
	if root == "" {
		return "", ErrNotGitWorkspace
	}
	return root, nil
}

func earlierTimestamp(a, b string) string {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	switch {
	case a == "":
		return b
	case b == "":
		return a
	}
	at, aerr := time.Parse(time.RFC3339, a)
	bt, berr := time.Parse(time.RFC3339, b)
	if aerr == nil && berr == nil {
		if bt.Before(at) {
			return b
		}
		return a
	}
	if b < a {
		return b
	}
	return a
}

func laterTimestamp(a, b string) string {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	switch {
	case a == "":
		return b
	case b == "":
		return a
	}
	at, aerr := time.Parse(time.RFC3339, a)
	bt, berr := time.Parse(time.RFC3339, b)
	if aerr == nil && berr == nil {
		if bt.After(at) {
			return b
		}
		return a
	}
	if b > a {
		return b
	}
	return a
}

func acquireLock(home string) (func(), error) {
	lockPath := filepath.Join(home, ".watchlist.lock")
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, err
	}
	return func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	}, nil
}

func writeJSONAtomic(path string, data []byte, perm os.FileMode) (err error) {
	dir := filepath.Dir(path)
	tempFile, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer func() {
		if err == nil {
			return
		}
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	if err := tempFile.Chmod(perm); err != nil {
		return err
	}
	if _, err := tempFile.Write(data); err != nil {
		return err
	}
	if err := tempFile.Sync(); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}

	dirFile, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer dirFile.Close()
	if err := dirFile.Sync(); err != nil {
		return err
	}
	return nil
}
