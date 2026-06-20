package smoke_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

func TestSyncBootstrapAssetsCheckPassesForCurrentRepo(t *testing.T) {
	repoRoot := support.RepoRoot(t)
	result := support.RunCommand(t, repoRoot, nil, filepath.Join(repoRoot, "scripts", "sync-bootstrap-assets"), "--check")
	if result.ExitCode != 0 {
		t.Fatalf("sync-bootstrap-assets --check exited with code %d\n%s", result.ExitCode, result.CombinedOutput())
	}
	if !strings.Contains(result.CombinedOutput(), "Bootstrap dogfood outputs are in sync with assets/bootstrap.") {
		t.Fatalf("unexpected check output:\n%s", result.CombinedOutput())
	}
}
