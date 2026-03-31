package smoke_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

func TestSyncBootstrapAssetsCheckPassesForCurrentRepo(t *testing.T) {
	repoRoot := support.RepoRoot(t)
	cmd := exec.Command(filepath.Join(repoRoot, "scripts", "sync-bootstrap-assets"), "--check")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("sync-bootstrap-assets --check: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "Bootstrap dogfood outputs are in sync with assets/bootstrap.") {
		t.Fatalf("unexpected check output:\n%s", output)
	}
}
