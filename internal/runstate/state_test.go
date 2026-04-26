package runstate

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveCurrentPlanWritesExactJSON(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".local", "harness", "current-plan.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir current plan dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"plan_path":"old","last_landed_plan_path":"stale-trailing-bytes-should-disappear"}`), 0o644); err != nil {
		t.Fatalf("seed current-plan.json: %v", err)
	}

	savedPath, err := SaveCurrentPlan(root, "docs/plans/active/example.md")
	if err != nil {
		t.Fatalf("SaveCurrentPlan: %v", err)
	}
	if savedPath != path {
		t.Fatalf("saved path = %q, want %q", savedPath, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read current-plan.json: %v", err)
	}
	want, err := json.MarshalIndent(CurrentPlan{PlanPath: "docs/plans/active/example.md"}, "", "  ")
	if err != nil {
		t.Fatalf("marshal expected current plan: %v", err)
	}
	if string(data) != string(want) {
		t.Fatalf("current-plan.json mismatch:\n got: %s\nwant: %s", data, want)
	}
}

func TestSaveStateWritesExactJSON(t *testing.T) {
	root := t.TempDir()
	planStem := "2026-03-26-atomic-save"
	path := filepath.Join(root, ".local", "harness", "plans", planStem, "state.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"revision":999,"execution_started_at":"stale","land":{"landed_at":"stale"}}`), 0o644); err != nil {
		t.Fatalf("seed state.json: %v", err)
	}

	state := &State{
		ExecutionStartedAt: "2026-03-26T10:00:00Z",
		Revision:           1,
		ActiveReviewRound: &ReviewRound{
			RoundID:    "review-001-delta",
			Kind:       "delta",
			Revision:   1,
			Aggregated: false,
		},
	}
	savedPath, err := SaveState(root, planStem, state)
	if err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	if savedPath != path {
		t.Fatalf("saved path = %q, want %q", savedPath, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read state.json: %v", err)
	}
	want, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("marshal expected state: %v", err)
	}
	if string(data) != string(want) {
		t.Fatalf("state.json mismatch:\n got: %s\nwant: %s", data, want)
	}

	loaded, _, err := LoadState(root, planStem)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if loaded == nil {
		t.Fatalf("LoadState returned nil state")
	}
	if loaded.ExecutionStartedAt != state.ExecutionStartedAt || loaded.Revision != state.Revision {
		t.Fatalf("loaded state = %#v, want %#v", loaded, state)
	}
	if loaded.ActiveReviewRound == nil || state.ActiveReviewRound == nil {
		t.Fatalf("expected active review round to survive save/load, got %#v", loaded)
	}
	if loaded.ActiveReviewRound.RoundID != state.ActiveReviewRound.RoundID || loaded.ActiveReviewRound.Kind != state.ActiveReviewRound.Kind || loaded.ActiveReviewRound.Revision != state.ActiveReviewRound.Revision {
		t.Fatalf("loaded state = %#v, want %#v", loaded, state)
	}
}

func TestWriteJSONAtomicPreservesOriginalFileWhenRenameFails(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".local", "harness", "current-plan.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir current plan dir: %v", err)
	}
	original := []byte(`{"plan_path":"docs/plans/active/original.md"}`)
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatalf("seed current-plan.json: %v", err)
	}

	restoreRename := renameFile
	renameFile = func(_, _ string) error {
		return errors.New("rename failed")
	}
	defer func() {
		renameFile = restoreRename
	}()

	if err := writeJSONAtomic(path, []byte(`{"plan_path":"docs/plans/active/new.md"}`), 0o644); err == nil {
		t.Fatal("expected writeJSONAtomic to fail when rename fails")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read current-plan.json: %v", err)
	}
	if string(data) != string(original) {
		t.Fatalf("expected original file to remain intact, got %s", data)
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("read current plan dir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "current-plan.json" {
		t.Fatalf("expected failed atomic write to clean up temp files, got %#v", entries)
	}
}

func TestStateMutationLockHeldDoesNotCreateMissingLockFile(t *testing.T) {
	root := t.TempDir()
	planStem := "2026-04-25-passive-lock-probe"
	lockPath := filepath.Join(root, ".local", "harness", "plans", planStem, ".state-mutation.lock")

	held, err := StateMutationLockHeld(root, planStem)
	if err != nil {
		t.Fatalf("StateMutationLockHeld: %v", err)
	}
	if held {
		t.Fatal("expected missing lock to report not held")
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("expected passive probe not to create lock file, err=%v", err)
	}
}

func TestStateMutationLockHeldDetectsHeldAndReleasedLock(t *testing.T) {
	root := t.TempDir()
	planStem := "2026-04-25-held-lock-probe"
	release, err := AcquireStateMutationLock(root, planStem)
	if err != nil {
		t.Fatalf("AcquireStateMutationLock: %v", err)
	}

	held, err := StateMutationLockHeld(root, planStem)
	if err != nil {
		t.Fatalf("StateMutationLockHeld while held: %v", err)
	}
	if !held {
		t.Fatal("expected held lock to report held")
	}

	release()
	held, err = StateMutationLockHeld(root, planStem)
	if err != nil {
		t.Fatalf("StateMutationLockHeld after release: %v", err)
	}
	if held {
		t.Fatal("expected released lock to report not held")
	}
}

func TestWaitForStateMutationLockReleaseWaitsUntilReleased(t *testing.T) {
	root := t.TempDir()
	planStem := "2026-04-25-wait-lock-release"
	release, err := AcquireStateMutationLock(root, planStem)
	if err != nil {
		t.Fatalf("AcquireStateMutationLock: %v", err)
	}

	released := make(chan struct{})
	go func() {
		time.Sleep(20 * time.Millisecond)
		release()
		close(released)
	}()

	held, err := WaitForStateMutationLockRelease(root, planStem, 500*time.Millisecond, time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForStateMutationLockRelease: %v", err)
	}
	if held {
		t.Fatal("expected wait to observe released lock")
	}
	<-released
}

func TestWaitForStateMutationLockReleaseReportsHeldAfterTimeout(t *testing.T) {
	root := t.TempDir()
	planStem := "2026-04-25-wait-lock-timeout"
	release, err := AcquireStateMutationLock(root, planStem)
	if err != nil {
		t.Fatalf("AcquireStateMutationLock: %v", err)
	}
	defer release()

	held, err := WaitForStateMutationLockRelease(root, planStem, time.Millisecond, time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForStateMutationLockRelease: %v", err)
	}
	if !held {
		t.Fatal("expected timeout to report lock still held")
	}
}
