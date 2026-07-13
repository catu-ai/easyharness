package reviewcoverage

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/runstate"
)

func TestResolveAcceptsFullRootAndDirectRepairDelta(t *testing.T) {
	root, base := coverageGitRepo(t)
	finding := coverageFinding("root-finding", "important")
	writeCoverageRound(t, root, "plan", contracts.ReviewManifest{
		RoundID: "review-001-full", Kind: "full", ReviewedHeadSHA: base, Revision: 1,
	}, contracts.ReviewAggregate{
		RoundID: "review-001-full", Kind: "full", ReviewedHeadSHA: base, Revision: 1,
		Decision: "changes_requested", BlockingFindings: []contracts.ReviewAggregateFinding{finding},
		NonBlockingFindings: []contracts.ReviewAggregateFinding{}, ResolvedFindingIDs: []string{},
		UnresolvedFindingIDs: []string{finding.FindingID}, UnresolvedBlockingFindings: []contracts.ReviewAggregateFinding{finding},
	})
	repairHead := coverageCommit(t, root, "repair")
	repair := &contracts.ReviewRepairReference{RoundID: "review-001-full", FindingIDs: []string{finding.FindingID}}
	writeCoverageRound(t, root, "plan", contracts.ReviewManifest{
		RoundID: "review-002-delta", Kind: "delta", AnchorSHA: base, ReviewedHeadSHA: repairHead, Revision: 1, Repair: repair,
	}, contracts.ReviewAggregate{
		RoundID: "review-002-delta", Kind: "delta", ReviewedHeadSHA: repairHead, Revision: 1, Repair: repair,
		Decision: "pass", BlockingFindings: []contracts.ReviewAggregateFinding{}, NonBlockingFindings: []contracts.ReviewAggregateFinding{},
		ResolvedFindingIDs: []string{finding.FindingID}, UnresolvedFindingIDs: []string{}, UnresolvedBlockingFindings: []contracts.ReviewAggregateFinding{},
	})

	chain, err := Resolve(root, "plan", "review-002-delta", 1)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if chain.RootRoundID != "review-001-full" || chain.TipRoundID != "review-002-delta" || chain.CoveredHeadSHA != repairHead || chain.Decision != "pass" || chain.UnresolvedBlockingCount != 0 {
		t.Fatalf("unexpected chain: %#v", chain)
	}
}

func TestResolveAcceptsEmptyRepairAcrossReopenAfterCleanTip(t *testing.T) {
	root, base := coverageGitRepo(t)
	writeCoverageRound(t, root, "plan", contracts.ReviewManifest{
		RoundID: "review-001-full", Kind: "full", ReviewedHeadSHA: base, Revision: 1,
	}, cleanCoverageAggregate("review-001-full", "full", base, 1, nil))
	reopenedHead := coverageCommit(t, root, "reopened revision")
	repair := &contracts.ReviewRepairReference{RoundID: "review-001-full", FindingIDs: []string{}}
	writeCoverageRound(t, root, "plan", contracts.ReviewManifest{
		RoundID: "review-002-delta", Kind: "delta", AnchorSHA: base, ReviewedHeadSHA: reopenedHead, Revision: 2, Repair: repair,
	}, cleanCoverageAggregate("review-002-delta", "delta", reopenedHead, 2, repair))

	if _, err := Resolve(root, "plan", "review-002-delta", 2); err != nil {
		t.Fatalf("Resolve reopened chain: %v", err)
	}
}

func TestResolveRejectsRepairTargetAbsentFromParent(t *testing.T) {
	root, base := coverageGitRepo(t)
	writeCoverageRound(t, root, "plan", contracts.ReviewManifest{
		RoundID: "review-001-full", Kind: "full", ReviewedHeadSHA: base, Revision: 1,
	}, cleanCoverageAggregate("review-001-full", "full", base, 1, nil))
	repairHead := coverageCommit(t, root, "unrelated repair")
	repair := &contracts.ReviewRepairReference{RoundID: "review-001-full", FindingIDs: []string{"absent-finding"}}
	writeCoverageRound(t, root, "plan", contracts.ReviewManifest{
		RoundID: "review-002-delta", Kind: "delta", AnchorSHA: base, ReviewedHeadSHA: repairHead, Revision: 1, Repair: repair,
	}, cleanCoverageAggregate("review-002-delta", "delta", repairHead, 1, repair))

	_, err := Resolve(root, "plan", "review-002-delta", 1)
	if err == nil || !strings.Contains(err.Error(), "absent from parent") {
		t.Fatalf("expected absent repair target rejection, got %v", err)
	}
}

func TestResolveRejectsTamperedCumulativeFinding(t *testing.T) {
	root, base := coverageGitRepo(t)
	finding := coverageFinding("root-finding", "important")
	writeCoverageRound(t, root, "plan", contracts.ReviewManifest{
		RoundID: "review-001-full", Kind: "full", ReviewedHeadSHA: base, Revision: 1,
	}, contracts.ReviewAggregate{
		RoundID: "review-001-full", Kind: "full", ReviewedHeadSHA: base, Revision: 1,
		Decision: "changes_requested", BlockingFindings: []contracts.ReviewAggregateFinding{finding},
		NonBlockingFindings: []contracts.ReviewAggregateFinding{}, ResolvedFindingIDs: []string{},
		UnresolvedFindingIDs: []string{finding.FindingID}, UnresolvedBlockingFindings: []contracts.ReviewAggregateFinding{finding},
	})
	repairHead := coverageCommit(t, root, "partial repair")
	repair := &contracts.ReviewRepairReference{RoundID: "review-001-full", FindingIDs: []string{finding.FindingID}}
	tampered := finding
	tampered.Details = "rewritten debt"
	writeCoverageRound(t, root, "plan", contracts.ReviewManifest{
		RoundID: "review-002-delta", Kind: "delta", AnchorSHA: base, ReviewedHeadSHA: repairHead, Revision: 1, Repair: repair,
	}, contracts.ReviewAggregate{
		RoundID: "review-002-delta", Kind: "delta", ReviewedHeadSHA: repairHead, Revision: 1, Repair: repair,
		Decision: "changes_requested", BlockingFindings: []contracts.ReviewAggregateFinding{}, NonBlockingFindings: []contracts.ReviewAggregateFinding{},
		ResolvedFindingIDs: []string{}, UnresolvedFindingIDs: []string{finding.FindingID}, UnresolvedBlockingFindings: []contracts.ReviewAggregateFinding{tampered},
	})

	_, err := Resolve(root, "plan", "review-002-delta", 1)
	if err == nil || !strings.Contains(err.Error(), "does not preserve") {
		t.Fatalf("expected tampered cumulative finding rejection, got %v", err)
	}
}

func TestResolveRejectsDuplicateRepairTargetsAndInheritedIDCollisions(t *testing.T) {
	for _, tc := range []struct {
		name       string
		repairIDs  []string
		childBlock bool
		want       string
	}{
		{name: "duplicate repair target", repairIDs: []string{"root-finding", "root-finding"}, want: "duplicate finding id"},
		{name: "new blocker collides with inherited debt", repairIDs: []string{"root-finding"}, childBlock: true, want: "collides with inherited"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, base := coverageGitRepo(t)
			finding := coverageFinding("root-finding", "important")
			writeCoverageRound(t, root, "plan", contracts.ReviewManifest{
				RoundID: "review-001-full", Kind: "full", ReviewedHeadSHA: base, Revision: 1,
			}, contracts.ReviewAggregate{
				RoundID: "review-001-full", Kind: "full", ReviewedHeadSHA: base, Revision: 1,
				Decision: "changes_requested", BlockingFindings: []contracts.ReviewAggregateFinding{finding},
				NonBlockingFindings: []contracts.ReviewAggregateFinding{}, ResolvedFindingIDs: []string{},
				UnresolvedFindingIDs: []string{finding.FindingID}, UnresolvedBlockingFindings: []contracts.ReviewAggregateFinding{finding},
			})
			repairHead := coverageCommit(t, root, "repair attempt")
			repair := &contracts.ReviewRepairReference{RoundID: "review-001-full", FindingIDs: tc.repairIDs}
			blocking := []contracts.ReviewAggregateFinding{}
			cumulative := finding
			if tc.childBlock {
				cumulative = coverageFinding(finding.FindingID, "important")
				cumulative.Details = "colliding rewrite"
				blocking = []contracts.ReviewAggregateFinding{cumulative}
			}
			writeCoverageRound(t, root, "plan", contracts.ReviewManifest{
				RoundID: "review-002-delta", Kind: "delta", AnchorSHA: base, ReviewedHeadSHA: repairHead, Revision: 1, Repair: repair,
			}, contracts.ReviewAggregate{
				RoundID: "review-002-delta", Kind: "delta", ReviewedHeadSHA: repairHead, Revision: 1, Repair: repair,
				Decision: "changes_requested", BlockingFindings: blocking, NonBlockingFindings: []contracts.ReviewAggregateFinding{},
				ResolvedFindingIDs: []string{}, UnresolvedFindingIDs: []string{finding.FindingID}, UnresolvedBlockingFindings: []contracts.ReviewAggregateFinding{cumulative},
			})

			_, err := Resolve(root, "plan", "review-002-delta", 1)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q rejection, got %v", tc.want, err)
			}
		})
	}
}

func cleanCoverageAggregate(roundID, kind, head string, revision int, repair *contracts.ReviewRepairReference) contracts.ReviewAggregate {
	return contracts.ReviewAggregate{
		RoundID: roundID, Kind: kind, ReviewedHeadSHA: head, Revision: revision, Repair: repair, Decision: "pass",
		BlockingFindings: []contracts.ReviewAggregateFinding{}, NonBlockingFindings: []contracts.ReviewAggregateFinding{},
		ResolvedFindingIDs: []string{}, UnresolvedFindingIDs: []string{}, UnresolvedBlockingFindings: []contracts.ReviewAggregateFinding{},
	}
}

func coverageFinding(id, severity string) contracts.ReviewAggregateFinding {
	return contracts.ReviewAggregateFinding{
		FindingID: id, Slot: "integrated", Role: "integrated", Area: "correctness", Severity: severity,
		Title: "Finding", Details: "Details",
	}
}

func writeCoverageRound(t *testing.T, root, planStem string, manifest contracts.ReviewManifest, aggregate contracts.ReviewAggregate) {
	t.Helper()
	dir := runstate.ReviewRoundDir(root, planStem, manifest.RoundID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir review round: %v", err)
	}
	writeCoverageJSON(t, filepath.Join(dir, "manifest.json"), manifest)
	writeCoverageJSON(t, filepath.Join(dir, "aggregate.json"), aggregate)
}

func writeCoverageJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func coverageGitRepo(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	coverageGit(t, root, "init", "-q")
	coverageGit(t, root, "config", "user.email", "tests@example.com")
	coverageGit(t, root, "config", "user.name", "Tests")
	if err := os.WriteFile(filepath.Join(root, "candidate.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write candidate: %v", err)
	}
	coverageGit(t, root, "add", "candidate.txt")
	coverageGit(t, root, "commit", "-q", "-m", "base")
	return root, strings.TrimSpace(coverageGit(t, root, "rev-parse", "HEAD"))
}

func coverageCommit(t *testing.T, root, message string) string {
	t.Helper()
	path := filepath.Join(root, "candidate.txt")
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open candidate: %v", err)
	}
	if _, err := file.WriteString(message + "\n"); err != nil {
		_ = file.Close()
		t.Fatalf("append candidate: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close candidate: %v", err)
	}
	coverageGit(t, root, "add", "candidate.txt")
	coverageGit(t, root, "commit", "-q", "-m", message)
	return strings.TrimSpace(coverageGit(t, root, "rev-parse", "HEAD"))
}

func coverageGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, data)
	}
	return string(data)
}
