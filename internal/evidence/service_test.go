package evidence_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/internal/evidence"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/remote"
	"github.com/catu-ai/easyharness/internal/runstate"
)

func TestSubmitCIEvidenceWritesArtifactWithoutStateCache(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
		},
	}.Submit("ci", []byte(`{"status":"success","provider":"buildkite","url":"https://ci.example/1"}`))
	if !result.OK {
		t.Fatalf("expected success, got %#v", result)
	}
	if result.Artifacts == nil || result.Artifacts.RecordID != "ci-001" {
		t.Fatalf("unexpected artifacts: %#v", result.Artifacts)
	}

	state, _, err := runstate.LoadState(root, "2026-03-21-evidence-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state != nil {
		t.Fatalf("expected CI submit to avoid state cache writes, got %#v", state)
	}
	assertStateFileAbsent(t, root, "2026-03-21-evidence-plan")

	record, err := evidence.LoadLatestCI(root, "2026-03-21-evidence-plan", 1)
	if err != nil {
		t.Fatalf("load latest CI record: %v", err)
	}
	if record == nil || record.Status != "success" || record.Provider != "buildkite" {
		t.Fatalf("unexpected CI record: %#v", record)
	}
}

func TestRefreshWritesCIAndSyncEvidenceFromRecordedPR(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	svc := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
		},
	}
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/99"}`)); !result.OK {
		t.Fatalf("seed publish evidence: %#v", result)
	}

	refresh := evidence.Service{
		Workdir:    root,
		Now:        svc.Now,
		RunCommand: fakeRefreshCommands(`"CLEAN"`, `[{"name":"Go Test","bucket":"pass","state":"SUCCESS","link":"https://ci.example/run"}]`),
	}.Refresh()

	if !refresh.OK {
		t.Fatalf("expected refresh success, got %#v", refresh)
	}
	if refresh.Artifacts == nil || refresh.Artifacts.CIRecordID != "ci-001" || refresh.Artifacts.SyncRecordID != "sync-001" {
		t.Fatalf("unexpected refresh artifacts: %#v", refresh.Artifacts)
	}
	ci, err := evidence.LoadLatestCI(root, "2026-03-21-evidence-plan", 1)
	if err != nil {
		t.Fatalf("load CI: %v", err)
	}
	if ci == nil || ci.Status != "success" || ci.Provider != "github-actions" || ci.URL != "https://ci.example/run" {
		t.Fatalf("unexpected CI record: %#v", ci)
	}
	sync, err := evidence.LoadLatestSync(root, "2026-03-21-evidence-plan", 1)
	if err != nil {
		t.Fatalf("load sync: %v", err)
	}
	if sync == nil || sync.Status != "fresh" || sync.BaseRef != "main" || sync.HeadRef != "codex/test" {
		t.Fatalf("unexpected sync record: %#v", sync)
	}
}

func TestRefreshRejectsMissingRecordedPRWithoutGuessing(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	called := false

	result := evidence.Service{
		Workdir: root,
		RunCommand: func(name string, args ...string) remote.CommandResult {
			called = true
			return remote.CommandResult{}
		},
	}.Refresh()

	if result.OK {
		t.Fatalf("expected missing PR refresh failure, got %#v", result)
	}
	if called {
		t.Fatal("refresh should not call gh without recorded publish PR URL")
	}
	if len(result.Errors) == 0 || result.Errors[0].Path != "publish.pr_url" {
		t.Fatalf("expected publish.pr_url error, got %#v", result.Errors)
	}
	if !refreshResultHasNextAction(result, "harness evidence submit --kind publish --input <json>") {
		t.Fatalf("expected publish fallback guidance, got %#v", result.NextAction)
	}
	if ci, err := evidence.LoadLatestCI(root, "2026-03-21-evidence-plan", 1); err != nil || ci != nil {
		t.Fatalf("expected no CI evidence, got %#v err=%v", ci, err)
	}
}

func TestRefreshRejectsNotAppliedPublishPRURL(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if result := (evidence.Service{Workdir: root}).Submit("publish", []byte(`{
		"status":"not_applied",
		"pr_url":"https://github.com/catu-ai/easyharness/pull/99",
		"reason":"remote handoff is not available"
	}`)); !result.OK {
		t.Fatalf("seed not_applied publish evidence: %#v", result)
	}
	called := false

	result := evidence.Service{
		Workdir: root,
		RunCommand: func(name string, args ...string) remote.CommandResult {
			called = true
			return remote.CommandResult{}
		},
	}.Refresh()

	if result.OK {
		t.Fatalf("expected not_applied publish refresh failure, got %#v", result)
	}
	if called {
		t.Fatal("refresh should not call gh for publish status not_applied")
	}
}

func TestRefreshWritesOnlyClearDomainEvidence(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	svc := evidence.Service{Workdir: root}
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/99"}`)); !result.OK {
		t.Fatalf("seed publish evidence: %#v", result)
	}

	result := evidence.Service{
		Workdir:    root,
		RunCommand: fakeRefreshCommands(`""`, `[{"name":"Go Test","bucket":"pass","state":"SUCCESS"}]`),
	}.Refresh()

	if !result.OK {
		t.Fatalf("expected partial refresh success, got %#v", result)
	}
	if result.Artifacts == nil || result.Artifacts.CIRecordID != "ci-001" || result.Artifacts.SyncRecordID != "" {
		t.Fatalf("unexpected partial refresh artifacts: %#v", result.Artifacts)
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("expected degraded sync warning, got %#v", result)
	}
	if !refreshResultHasNextAction(result, "harness evidence submit --kind sync --input <json>") {
		t.Fatalf("expected sync fallback guidance, got %#v", result.NextAction)
	}
	if sync, err := evidence.LoadLatestSync(root, "2026-03-21-evidence-plan", 1); err != nil || sync != nil {
		t.Fatalf("expected no sync evidence, got %#v err=%v", sync, err)
	}
}

func TestRefreshDoesNotLeaveCIWhenSyncRecordLocationFails(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if result := (evidence.Service{Workdir: root}).Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/99"}`)); !result.OK {
		t.Fatalf("seed publish evidence: %#v", result)
	}
	syncPath := filepath.Join(root, ".local", "harness", "plans", "2026-03-21-evidence-plan", "evidence", "sync")
	if err := os.MkdirAll(filepath.Dir(syncPath), 0o755); err != nil {
		t.Fatalf("mkdir evidence dir: %v", err)
	}
	if err := os.WriteFile(syncPath, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write sync path blocker: %v", err)
	}

	result := evidence.Service{
		Workdir:    root,
		RunCommand: fakeRefreshCommands(`"CLEAN"`, `[{"name":"Go Test","bucket":"pass","state":"SUCCESS"}]`),
	}.Refresh()

	if result.OK {
		t.Fatalf("expected sync record-location failure, got %#v", result)
	}
	if ci, err := evidence.LoadLatestCI(root, "2026-03-21-evidence-plan", 1); err != nil || ci != nil {
		t.Fatalf("expected no unreported CI evidence, got %#v err=%v", ci, err)
	}
}

func TestSubmitPublishRejectsMissingPRURL(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := evidence.Service{Workdir: root}.Submit("publish", []byte(`{"status":"recorded"}`))
	if result.OK {
		t.Fatalf("expected validation failure, got %#v", result)
	}
}

func TestSubmitPublishRejectsUnknownField(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := evidence.Service{Workdir: root}.Submit("publish", []byte(`{
		"status":"recorded",
		"pr_url":"https://github.com/catu-ai/easyharness/pull/99",
		"unexpected":true
	}`))
	if result.OK {
		t.Fatalf("expected validation failure, got %#v", result)
	}
	assertEvidenceError(t, result, "input.unexpected")
}

func TestSubmitCIRejectsUnknownSchemaProperty(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := evidence.Service{Workdir: root}.Submit("ci", []byte(`{
		"status": "success",
		"provider": "buildkite",
		"unexpected": true
	}`))
	if result.OK {
		t.Fatalf("expected schema validation failure, got %#v", result)
	}
	if len(result.Errors) == 0 || result.Errors[0].Path != "input.unexpected" {
		t.Fatalf("expected unknown-property error, got %#v", result.Errors)
	}
}

func TestSubmitCIRejectsUnknownField(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := evidence.Service{Workdir: root}.Submit("ci", []byte(`{"status":"success","unexpected":true}`))
	if result.OK {
		t.Fatalf("expected validation failure, got %#v", result)
	}
	assertEvidenceError(t, result, "input.unexpected")
}

func TestSubmitCIRejectsWrongStatusType(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := evidence.Service{Workdir: root}.Submit("ci", []byte(`{"status":1}`))
	if result.OK {
		t.Fatalf("expected validation failure, got %#v", result)
	}
	assertEvidenceError(t, result, "input.status")
}

func TestSubmitPublishWritesArtifactWithoutStateCache(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 21, 10, 2, 0, 0, time.UTC)
		},
	}.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/99","branch":"codex/test","base":"main"}`))
	if !result.OK {
		t.Fatalf("expected success, got %#v", result)
	}

	state, _, err := runstate.LoadState(root, "2026-03-21-evidence-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state != nil {
		t.Fatalf("expected publish submit to avoid state cache writes, got %#v", state)
	}
	assertStateFileAbsent(t, root, "2026-03-21-evidence-plan")

	record, err := evidence.LoadLatestPublish(root, "2026-03-21-evidence-plan", 1)
	if err != nil {
		t.Fatalf("load latest publish record: %v", err)
	}
	if record == nil || record.Status != "recorded" || record.PRURL != "https://github.com/catu-ai/easyharness/pull/99" {
		t.Fatalf("unexpected publish record: %#v", record)
	}
}

func TestSubmitSyncSupportsExplicitNotApplied(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 21, 10, 5, 0, 0, time.UTC)
		},
	}.Submit("sync", []byte(`{"status":"not_applied","reason":"repository has no merge target in this environment"}`))
	if !result.OK {
		t.Fatalf("expected success, got %#v", result)
	}

	state, _, err := runstate.LoadState(root, "2026-03-21-evidence-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state != nil {
		t.Fatalf("expected sync submit to avoid state cache writes, got %#v", state)
	}
	assertStateFileAbsent(t, root, "2026-03-21-evidence-plan")
	record, err := evidence.LoadLatestSync(root, "2026-03-21-evidence-plan", 1)
	if err != nil {
		t.Fatalf("load latest sync record: %v", err)
	}
	if record == nil || record.Status != "not_applied" {
		t.Fatalf("unexpected sync record: %#v", record)
	}
}

func TestSubmitSyncRejectsWrongHeadRefType(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := evidence.Service{Workdir: root}.Submit("sync", []byte(`{
		"status":"fresh",
		"head_ref":true
	}`))
	if result.OK {
		t.Fatalf("expected validation failure, got %#v", result)
	}
	assertEvidenceError(t, result, "input.head_ref")
}

func TestSubmitSyncFreshWritesArtifactWithoutStateCache(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 21, 10, 7, 0, 0, time.UTC)
		},
	}.Submit("sync", []byte(`{"status":"fresh","base_ref":"main","head_ref":"codex/test"}`))
	if !result.OK {
		t.Fatalf("expected success, got %#v", result)
	}

	state, _, err := runstate.LoadState(root, "2026-03-21-evidence-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state != nil {
		t.Fatalf("expected sync submit to avoid state cache writes, got %#v", state)
	}
	assertStateFileAbsent(t, root, "2026-03-21-evidence-plan")

	record, err := evidence.LoadLatestSync(root, "2026-03-21-evidence-plan", 1)
	if err != nil {
		t.Fatalf("load latest sync record: %v", err)
	}
	if record == nil || record.Status != "fresh" || record.BaseRef != "main" {
		t.Fatalf("unexpected sync record: %#v", record)
	}
}

func TestSubmitEvidenceRequiresArchivedPlan(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeActivePlan(t, root, "docs/plans/active/2026-03-21-active-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := evidence.Service{Workdir: root}.Submit("ci", []byte(`{"status":"success"}`))
	if result.OK {
		t.Fatalf("expected archived-plan requirement failure, got %#v", result)
	}
}

func TestSubmitEvidenceRejectsLandInProgress(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, err := runstate.SaveState(root, "2026-03-21-evidence-plan", &runstate.State{
		Land: &runstate.LandState{
			PRURL:    "https://github.com/catu-ai/easyharness/pull/99",
			LandedAt: "2026-03-21T11:00:00Z",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	result := evidence.Service{Workdir: root}.Submit("ci", []byte(`{"status":"success"}`))
	if result.OK {
		t.Fatalf("expected land-in-progress evidence rejection, got %#v", result)
	}
}

func TestSubmitEvidenceRejectsWhenStateMutationLockIsHeld(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	release, err := runstate.AcquireStateMutationLock(root, "2026-03-21-evidence-plan")
	if err != nil {
		t.Fatalf("acquire state lock: %v", err)
	}
	defer release()

	result := evidence.Service{Workdir: root}.Submit("ci", []byte(`{"status":"success"}`))
	if result.OK {
		t.Fatalf("expected state-lock contention failure, got %#v", result)
	}
	if result.Summary != "Another local state mutation is already in progress." {
		t.Fatalf("unexpected summary: %#v", result)
	}
	if len(result.Errors) != 1 || result.Errors[0].Path != "state" {
		t.Fatalf("unexpected errors: %#v", result.Errors)
	}
}

func TestLoadLatestCIPrefersNewestRecord(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	first := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
		},
	}.Submit("ci", []byte(`{"status":"pending","provider":"buildkite","url":"https://ci.example/1"}`))
	if !first.OK {
		t.Fatalf("expected first CI submit success, got %#v", first)
	}
	second := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 21, 10, 3, 0, 0, time.UTC)
		},
	}.Submit("ci", []byte(`{"status":"success","provider":"buildkite","url":"https://ci.example/2"}`))
	if !second.OK {
		t.Fatalf("expected second CI submit success, got %#v", second)
	}

	record, err := evidence.LoadLatestCI(root, "2026-03-21-evidence-plan", 1)
	if err != nil {
		t.Fatalf("load latest CI record: %v", err)
	}
	if record == nil || record.RecordID != "ci-002" || record.Status != "success" || record.URL != "https://ci.example/2" {
		t.Fatalf("expected newest CI record to win, got %#v", record)
	}
}

func TestLoadLatestPublishPrefersNewestRecord(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	first := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
		},
	}.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/99","branch":"codex/test","base":"main","commit":"abc123"}`))
	if !first.OK {
		t.Fatalf("expected first publish submit success, got %#v", first)
	}
	second := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 21, 10, 3, 0, 0, time.UTC)
		},
	}.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/100","branch":"codex/test-2","base":"main","commit":"def456"}`))
	if !second.OK {
		t.Fatalf("expected second publish submit success, got %#v", second)
	}

	record, err := evidence.LoadLatestPublish(root, "2026-03-21-evidence-plan", 1)
	if err != nil {
		t.Fatalf("load latest publish record: %v", err)
	}
	if record == nil || record.RecordID != "publish-002" || record.PRURL != "https://github.com/catu-ai/easyharness/pull/100" || record.Commit != "def456" {
		t.Fatalf("expected newest publish record to win, got %#v", record)
	}
}

func TestLoadLatestSyncPrefersNewestRecord(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	first := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
		},
	}.Submit("sync", []byte(`{"status":"stale","base_ref":"main","head_ref":"codex/test"}`))
	if !first.OK {
		t.Fatalf("expected first sync submit success, got %#v", first)
	}
	second := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 21, 10, 3, 0, 0, time.UTC)
		},
	}.Submit("sync", []byte(`{"status":"fresh","base_ref":"main","head_ref":"codex/test-2"}`))
	if !second.OK {
		t.Fatalf("expected second sync submit success, got %#v", second)
	}

	record, err := evidence.LoadLatestSync(root, "2026-03-21-evidence-plan", 1)
	if err != nil {
		t.Fatalf("load latest sync record: %v", err)
	}
	if record == nil || record.RecordID != "sync-002" || record.Status != "fresh" || record.HeadRef != "codex/test-2" {
		t.Fatalf("expected newest sync record to win, got %#v", record)
	}
}

func TestLoadLatestRecordIgnoresOlderRevisionEvidence(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	first := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
		},
	}.Submit("ci", []byte(`{"status":"success","provider":"buildkite","url":"https://ci.example/1"}`))
	if !first.OK {
		t.Fatalf("expected first CI submit success, got %#v", first)
	}
	if _, err := runstate.SaveState(root, "2026-03-21-evidence-plan", &runstate.State{Revision: 2}); err != nil {
		t.Fatalf("save reopened revision state: %v", err)
	}

	record, err := evidence.LoadLatestCI(root, "2026-03-21-evidence-plan", 2)
	if err != nil {
		t.Fatalf("load latest CI record: %v", err)
	}
	if record != nil {
		t.Fatalf("expected older revision evidence to be ignored, got %#v", record)
	}
}

func assertEvidenceError(t *testing.T, result evidence.Result, path string) {
	t.Helper()
	for _, issue := range result.Errors {
		if issue.Path == path {
			return
		}
	}
	t.Fatalf("expected evidence error for %s, got %#v", path, result.Errors)
}

func assertStateFileAbsent(t *testing.T, root, planStem string) {
	t.Helper()
	path := filepath.Join(root, ".local", "harness", "plans", planStem, "state.json")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected state.json to stay absent, got %v", err)
	}
}

func fakeRefreshCommands(mergeStateJSON, checksJSON string) remote.CommandRunner {
	return func(name string, args ...string) remote.CommandResult {
		if len(args) >= 3 && args[0] == "pr" && args[1] == "view" {
			return remote.CommandResult{Stdout: `{
				"url":"https://github.com/catu-ai/easyharness/pull/99",
				"number":99,
				"state":"OPEN",
				"isDraft":false,
				"mergeStateStatus":` + mergeStateJSON + `,
				"mergeable":"MERGEABLE",
				"reviewDecision":"APPROVED",
				"headRefName":"codex/test",
				"headRefOid":"abc123",
				"baseRefName":"main"
			}`}
		}
		if len(args) >= 3 && args[0] == "pr" && args[1] == "checks" {
			return remote.CommandResult{Stdout: checksJSON}
		}
		return remote.CommandResult{}
	}
}

func refreshResultHasNextAction(result evidence.RefreshResult, command string) bool {
	for _, action := range result.NextAction {
		if action.Command != nil && *action.Command == command {
			return true
		}
	}
	return false
}

func writeArchivedPlan(t *testing.T, root, relPath string) string {
	t.Helper()
	return writePlan(t, root, relPath, "Archived Evidence Plan")
}

func writeActivePlan(t *testing.T, root, relPath string) string {
	t.Helper()
	return writePlan(t, root, relPath, "Active Evidence Plan")
}

func writePlan(t *testing.T, root, relPath, title string) string {
	t.Helper()
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      title,
		Timestamp:  time.Date(2026, 3, 21, 9, 0, 0, 0, time.UTC),
		SourceType: "direct_request",
	})
	if err != nil {
		t.Fatalf("render template: %v", err)
	}
	rendered = strings.Replace(rendered, "size: REPLACE_WITH_PLAN_SIZE", "size: M", 1)
	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	return relPath
}
