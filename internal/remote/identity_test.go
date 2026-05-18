package remote

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParseRecordedPRURLSupportsGitHub(t *testing.T) {
	identity := ParseRecordedPRURL("https://github.com/catu-ai/easyharness/pull/203")

	if identity.Status != PRStatusRecorded {
		t.Fatalf("expected recorded status, got %#v", identity)
	}
	if identity.Owner != "catu-ai" || identity.Repo != "easyharness" || identity.Number != 203 {
		t.Fatalf("unexpected parsed identity: %#v", identity)
	}
	if identity.URL != "https://github.com/catu-ai/easyharness/pull/203" {
		t.Fatalf("expected canonical URL to preserve input, got %q", identity.URL)
	}
}

func TestParseRecordedPRURLDegradesWhenMissingOrUnsupported(t *testing.T) {
	missing := ParseRecordedPRURL("")
	if missing.Status != PRStatusMissing || missing.Degraded.Code != DegradedMissingPRURL {
		t.Fatalf("expected missing PR URL degradation, got %#v", missing)
	}

	unsupported := ParseRecordedPRURL("https://gitlab.com/catu-ai/easyharness/-/merge_requests/1")
	if unsupported.Status != PRStatusUnsupported || unsupported.Degraded.Code != DegradedUnsupportedPRURL {
		t.Fatalf("expected unsupported PR URL degradation, got %#v", unsupported)
	}
}

func TestSnapshotReportsBranchAndGitHubRemoteContext(t *testing.T) {
	root := seedGitRepo(t)
	runGit(t, root, "remote", "add", "origin", "git@github.com:catu-ai/easyharness.git")

	snapshot := Service{Workdir: root}.Snapshot("https://github.com/catu-ai/easyharness/pull/203")

	if snapshot.PR.Status != PRStatusRecorded {
		t.Fatalf("expected recorded PR identity, got %#v", snapshot.PR)
	}
	if !snapshot.Local.InGitRepo || snapshot.Local.Branch == "" || snapshot.Local.Detached {
		t.Fatalf("expected branch context, got %#v", snapshot.Local)
	}
	if snapshot.Local.Remote == nil {
		t.Fatalf("expected remote context, got %#v", snapshot.Local)
	}
	if snapshot.Local.Remote.Name != "origin" || snapshot.Local.Remote.Owner != "catu-ai" || snapshot.Local.Remote.Repo != "easyharness" {
		t.Fatalf("unexpected remote context: %#v", snapshot.Local.Remote)
	}
}

func TestLocalContextDegradesForDetachedHead(t *testing.T) {
	root := seedGitRepo(t)
	head := runGit(t, root, "rev-parse", "HEAD")
	runGit(t, root, "checkout", "--detach", head)

	local := InspectLocal(root)

	if !local.Detached || local.Branch != "" {
		t.Fatalf("expected detached local context, got %#v", local)
	}
	if !hasDegradation(local.Degraded, DegradedDetachedHead) {
		t.Fatalf("expected detached degradation, got %#v", local.Degraded)
	}
}

func TestLocalContextDegradesForMissingRemote(t *testing.T) {
	root := seedGitRepo(t)

	local := InspectLocal(root)

	if local.Remote != nil {
		t.Fatalf("expected no remote context, got %#v", local.Remote)
	}
	if !hasDegradation(local.Degraded, DegradedMissingRemote) {
		t.Fatalf("expected missing remote degradation, got %#v", local.Degraded)
	}
}

func TestLocalContextDegradesForUnsupportedRemote(t *testing.T) {
	root := seedGitRepo(t)
	runGit(t, root, "remote", "add", "origin", "ssh://git@example.com/catu-ai/easyharness.git")

	local := InspectLocal(root)

	if local.Remote == nil || local.Remote.Supported {
		t.Fatalf("expected unsupported remote context, got %#v", local.Remote)
	}
	if !hasDegradation(local.Degraded, DegradedUnsupportedRemote) {
		t.Fatalf("expected unsupported remote degradation, got %#v", local.Degraded)
	}
}

func TestLocalContextReportsNotGitRepository(t *testing.T) {
	local := InspectLocal(t.TempDir())

	if local.InGitRepo {
		t.Fatalf("expected non-git context, got %#v", local)
	}
	if !hasDegradation(local.Degraded, DegradedNotGitRepository) {
		t.Fatalf("expected not-git degradation, got %#v", local.Degraded)
	}
}

func seedGitRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	runGit(t, root, "init", "-b", "main")
	runGit(t, root, "config", "user.name", "Codex Test")
	runGit(t, root, "config", "user.email", "codex@example.com")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("test repo\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "fixture")
	return root
}

func runGit(t *testing.T, root string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
	return string(bytesTrimSpace(output))
}

func hasDegradation(degraded []Degradation, code string) bool {
	for _, item := range degraded {
		if item.Code == code {
			return true
		}
	}
	return false
}

func bytesTrimSpace(input []byte) []byte {
	for len(input) > 0 {
		switch input[0] {
		case ' ', '\n', '\r', '\t':
			input = input[1:]
		default:
			goto trimRight
		}
	}
	return input

trimRight:
	for len(input) > 0 {
		switch input[len(input)-1] {
		case ' ', '\n', '\r', '\t':
			input = input[:len(input)-1]
		default:
			return input
		}
	}
	return input
}
