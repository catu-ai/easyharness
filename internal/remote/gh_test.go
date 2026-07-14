package remote

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestObserveRecordedPRUsesGhView(t *testing.T) {
	var calls []commandCall
	svc := Service{
		RunCommand: func(name string, args ...string) CommandResult {
			calls = append(calls, commandCall{Name: name, Args: args})
			return CommandResult{Stdout: `{
				"url": "https://github.com/catu-ai/easyharness/pull/203",
				"number": 203,
				"state": "OPEN",
				"headRefName": "codex/read-only-pr-handoff-identity",
				"headRefOid": "abc123",
				"baseRefName": "main",
				"baseRefOid": "def456"
			}`}
		},
	}

	observation := svc.ObserveRecordedPR(ParseRecordedPRURL("https://github.com/catu-ai/easyharness/pull/203"))

	if observation.Status != PRObservationAvailable {
		t.Fatalf("expected available observation, got %#v", observation)
	}
	if observation.State != "OPEN" || observation.HeadRefName != "codex/read-only-pr-handoff-identity" || observation.BaseRefName != "main" {
		t.Fatalf("unexpected observation: %#v", observation)
	}
	want := []commandCall{{
		Name: "gh",
		Args: []string{
			"pr", "view", "https://github.com/catu-ai/easyharness/pull/203",
			"--json", "url,number,state,isDraft,mergeStateStatus,mergeable,reviewDecision,headRefName,headRefOid,baseRefName,baseRefOid",
		},
	}}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("unexpected gh calls:\nwant %#v\ngot  %#v", want, calls)
	}
}

func TestObserveHandoffMapsPassingChecksAndCleanMergeState(t *testing.T) {
	var calls []commandCall
	svc := Service{
		RunCommand: func(name string, args ...string) CommandResult {
			calls = append(calls, commandCall{Name: name, Args: args})
			switch {
			case len(args) >= 3 && args[0] == "pr" && args[1] == "view":
				return CommandResult{Stdout: `{
					"url": "https://github.com/catu-ai/easyharness/pull/199",
					"number": 199,
					"state": "OPEN",
					"isDraft": false,
					"mergeStateStatus": "CLEAN",
					"mergeable": "MERGEABLE",
					"reviewDecision": "APPROVED",
					"headRefName": "codex/refresh",
					"headRefOid": "abc123",
					"baseRefName": "main",
					"baseRefOid": "def456"
				}`}
			case len(args) >= 3 && args[0] == "pr" && args[1] == "checks":
				return CommandResult{Stdout: `[
					{"name":"Go Test","workflow":"Go Test","bucket":"pass","state":"SUCCESS","link":"https://ci.example/1"},
					{"name":"Lint","workflow":"Go Test","bucket":"skipping","state":"SKIPPED"}
				]`}
			case len(args) >= 2 && args[0] == "api":
				return CommandResult{Stdout: `{"status":"ahead","ahead_by":1,"behind_by":0}`}
			default:
				t.Fatalf("unexpected command %s %v", name, args)
				return CommandResult{}
			}
		},
	}

	observation := svc.ObserveHandoff(ParseRecordedPRURL("https://github.com/catu-ai/easyharness/pull/199"))

	if observation.Status != HandoffObservationAvailable {
		t.Fatalf("expected available handoff observation, got %#v", observation)
	}
	if observation.CI.Status != RemoteCIAvailable || observation.CI.EvidenceStatus != "success" {
		t.Fatalf("expected successful CI observation, got %#v", observation.CI)
	}
	if observation.Sync.Status != RemoteSyncAvailable || observation.Sync.EvidenceStatus != "fresh" {
		t.Fatalf("expected fresh sync observation, got %#v", observation.Sync)
	}
	if len(calls) != 3 {
		t.Fatalf("expected pr view, pr checks, and compare calls, got %#v", calls)
	}
	want := []commandCall{{
		Name: "gh",
		Args: []string{
			"pr", "view", "https://github.com/catu-ai/easyharness/pull/199",
			"--json", "url,number,state,isDraft,mergeStateStatus,mergeable,reviewDecision,headRefName,headRefOid,baseRefName,baseRefOid",
		},
	}, {
		Name: "gh",
		Args: []string{
			"pr", "checks", "https://github.com/catu-ai/easyharness/pull/199",
			"--json", "name,workflow,bucket,state,link",
		},
	}, {
		Name: "gh",
		Args: []string{
			"api", "repos/catu-ai/easyharness/compare/def456...abc123",
		},
	}}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("unexpected gh calls:\nwant %#v\ngot  %#v", want, calls)
	}
}

func TestObserveHandoffDoesNotCallGhWhenNoRecordedPR(t *testing.T) {
	called := false
	svc := Service{
		RunCommand: func(name string, args ...string) CommandResult {
			called = true
			return CommandResult{}
		},
	}

	observation := svc.ObserveHandoff(ParseRecordedPRURL(""))

	if called {
		t.Fatal("did not expect gh to run without a recorded PR URL")
	}
	if observation.Status != HandoffObservationUnavailable {
		t.Fatalf("expected unavailable handoff observation, got %#v", observation)
	}
	if observation.CI.Status != RemoteCIUnavailable || observation.CI.Degraded.Code != DegradedMissingPRURL {
		t.Fatalf("expected CI missing PR degradation, got %#v", observation.CI)
	}
	if observation.Sync.Status != RemoteSyncUnavailable || observation.Sync.Degraded.Code != DegradedMissingPRURL {
		t.Fatalf("expected sync missing PR degradation, got %#v", observation.Sync)
	}
}

func TestObserveHandoffMapsPendingChecks(t *testing.T) {
	svc := Service{RunCommand: fakePRAndChecks(`{"mergeStateStatus":"UNSTABLE"}`, `[
		{"name":"Go Test","bucket":"pass","state":"SUCCESS"},
		{"name":"Smoke","bucket":"pending","state":"IN_PROGRESS"}
	]`)}

	observation := svc.ObserveHandoff(ParseRecordedPRURL("https://github.com/catu-ai/easyharness/pull/199"))

	if observation.CI.Status != RemoteCIAvailable || observation.CI.EvidenceStatus != "pending" {
		t.Fatalf("expected pending CI observation, got %#v", observation.CI)
	}
	if observation.Sync.Status != RemoteSyncAvailable || observation.Sync.EvidenceStatus != "fresh" {
		t.Fatalf("expected UNSTABLE checks not to make sync stale, got %#v", observation.Sync)
	}
}

func TestObserveHandoffMapsFailingAndCancelledChecks(t *testing.T) {
	for _, bucket := range []string{"fail", "cancel"} {
		t.Run(bucket, func(t *testing.T) {
			svc := Service{RunCommand: fakePRAndChecks(`{"mergeStateStatus":"CLEAN"}`, `[
				{"name":"Go Test","bucket":"`+bucket+`","state":"FAILURE"}
			]`)}

			observation := svc.ObserveHandoff(ParseRecordedPRURL("https://github.com/catu-ai/easyharness/pull/199"))

			if observation.CI.Status != RemoteCIAvailable || observation.CI.EvidenceStatus != "failed" {
				t.Fatalf("expected failed CI observation, got %#v", observation.CI)
			}
		})
	}
}

func TestObserveHandoffSeparatesFreshnessConflictAndPolicy(t *testing.T) {
	tests := []struct {
		name         string
		mergeState   string
		mergeable    string
		behindBy     int
		wantSync     string
		wantFresh    string
		wantConflict string
		wantPolicy   string
	}{
		{name: "clean", mergeState: "CLEAN", mergeable: "MERGEABLE", wantSync: "fresh", wantFresh: "fresh", wantConflict: "clear", wantPolicy: "clear"},
		{name: "compare behind despite clean merge state", mergeState: "CLEAN", mergeable: "MERGEABLE", behindBy: 2, wantSync: "stale", wantFresh: "stale", wantConflict: "clear", wantPolicy: "clear"},
		{name: "behind merge state but current comparison", mergeState: "BEHIND", mergeable: "MERGEABLE", wantSync: "fresh", wantFresh: "fresh", wantConflict: "clear", wantPolicy: "clear"},
		{name: "blocked policy", mergeState: "BLOCKED", mergeable: "MERGEABLE", wantSync: "fresh", wantFresh: "fresh", wantConflict: "clear", wantPolicy: "blocked"},
		{name: "hooks policy", mergeState: "HAS_HOOKS", mergeable: "MERGEABLE", wantSync: "fresh", wantFresh: "fresh", wantConflict: "clear", wantPolicy: "blocked"},
		{name: "unstable checks", mergeState: "UNSTABLE", mergeable: "MERGEABLE", wantSync: "fresh", wantFresh: "fresh", wantConflict: "clear", wantPolicy: "clear"},
		{name: "unknown provider state", mergeState: "UNKNOWN", mergeable: "UNKNOWN", wantSync: "fresh", wantFresh: "fresh", wantConflict: "unknown", wantPolicy: "unknown"},
		{name: "conflicted", mergeState: "DIRTY", mergeable: "CONFLICTING", wantSync: "conflicted", wantFresh: "fresh", wantConflict: "conflicted", wantPolicy: "clear"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := Service{RunCommand: fakePRChecksAndCompare(`{"mergeStateStatus":"`+tt.mergeState+`","mergeable":"`+tt.mergeable+`"}`, `[
				{"name":"Go Test","bucket":"pass","state":"SUCCESS"}
			]`, tt.behindBy)}

			observation := svc.ObserveHandoff(ParseRecordedPRURL("https://github.com/catu-ai/easyharness/pull/199"))

			if observation.Sync.Status != RemoteSyncAvailable || observation.Sync.EvidenceStatus != tt.wantSync || observation.Sync.Freshness != tt.wantFresh || observation.Sync.Conflict != tt.wantConflict || observation.Sync.MergePolicy != tt.wantPolicy {
				t.Fatalf("unexpected separated sync observation: %#v", observation.Sync)
			}
		})
	}
}

func TestObserveHandoffDegradesWhenChecksAreUnreadableButMergeIsClear(t *testing.T) {
	svc := Service{RunCommand: func(name string, args ...string) CommandResult {
		if len(args) >= 3 && args[0] == "pr" && args[1] == "view" {
			return CommandResult{Stdout: `{"mergeStateStatus":"CLEAN","mergeable":"MERGEABLE","headRefOid":"abc123","baseRefOid":"def456"}`}
		}
		if len(args) >= 2 && args[0] == "api" {
			return CommandResult{Stdout: `{"behind_by":0}`}
		}
		return CommandResult{Err: exec.ErrNotFound}
	}}

	observation := svc.ObserveHandoff(ParseRecordedPRURL("https://github.com/catu-ai/easyharness/pull/199"))

	if observation.CI.Status != RemoteCIUnavailable || observation.CI.Degraded.Code != DegradedGhMissing {
		t.Fatalf("expected degraded CI observation, got %#v", observation.CI)
	}
	if observation.Sync.Status != RemoteSyncAvailable || observation.Sync.EvidenceStatus != "fresh" {
		t.Fatalf("expected sync to remain refreshable, got %#v", observation.Sync)
	}
}

func TestObserveHandoffDegradesWhenComparisonIsUnreadableButChecksAreClear(t *testing.T) {
	svc := Service{RunCommand: func(name string, args ...string) CommandResult {
		if len(args) >= 3 && args[0] == "pr" && args[1] == "view" {
			return CommandResult{Stdout: `{"mergeStateStatus":"UNSTABLE","mergeable":"MERGEABLE","headRefOid":"abc123","baseRefOid":"def456"}`}
		}
		if len(args) >= 3 && args[0] == "pr" && args[1] == "checks" {
			return CommandResult{Stdout: `[{"name":"Go Test","bucket":"pass","state":"SUCCESS"}]`}
		}
		return CommandResult{Err: errors.New("comparison unavailable")}
	}}

	observation := svc.ObserveHandoff(ParseRecordedPRURL("https://github.com/catu-ai/easyharness/pull/199"))

	if observation.CI.Status != RemoteCIAvailable || observation.CI.EvidenceStatus != "success" {
		t.Fatalf("expected CI to remain refreshable, got %#v", observation.CI)
	}
	if observation.Sync.Status != RemoteSyncUnavailable || observation.Sync.Degraded.Code != DegradedGhCommandFailed {
		t.Fatalf("expected degraded sync observation, got %#v", observation.Sync)
	}
	if observation.Sync.Conflict != "clear" || observation.Sync.MergePolicy != "clear" {
		t.Fatalf("expected independent conflict and policy facts, got %#v", observation.Sync)
	}
}

func TestSnapshotDoesNotObserveRecordedPRImplicitly(t *testing.T) {
	root := seedGitRepo(t)
	called := false
	svc := Service{
		Workdir: root,
		RunCommand: func(name string, args ...string) CommandResult {
			called = true
			return CommandResult{}
		},
	}

	snapshot := svc.Snapshot("https://github.com/catu-ai/easyharness/pull/203")

	if called {
		t.Fatal("snapshot should not call gh; PR observation is explicit")
	}
	if snapshot.PR.Status != PRStatusRecorded {
		t.Fatalf("expected recorded PR identity, got %#v", snapshot.PR)
	}
}

func TestObserveRecordedPRDoesNotCallGhWhenNoRecordedPR(t *testing.T) {
	called := false
	svc := Service{
		RunCommand: func(name string, args ...string) CommandResult {
			called = true
			return CommandResult{}
		},
	}

	observation := svc.ObserveRecordedPR(ParseRecordedPRURL(""))

	if called {
		t.Fatal("did not expect gh to run without a recorded PR URL")
	}
	if observation.Status != PRObservationUnavailable || observation.Degraded.Code != DegradedMissingPRURL {
		t.Fatalf("expected missing PR degradation, got %#v", observation)
	}
}

func TestObserveRecordedPRDegradesWhenGhMissing(t *testing.T) {
	svc := Service{
		RunCommand: func(name string, args ...string) CommandResult {
			return CommandResult{Err: exec.ErrNotFound}
		},
	}

	observation := svc.ObserveRecordedPR(ParseRecordedPRURL("https://github.com/catu-ai/easyharness/pull/203"))

	if observation.Status != PRObservationUnavailable || observation.Degraded.Code != DegradedGhMissing {
		t.Fatalf("expected missing gh degradation, got %#v", observation)
	}
}

func TestDefaultRunnerTimesOutGhReads(t *testing.T) {
	oldTimeout := defaultCommandTimeout
	oldWaitDelay := defaultCommandWaitDelay
	defaultCommandTimeout = 20 * time.Millisecond
	defaultCommandWaitDelay = 20 * time.Millisecond
	defer func() {
		defaultCommandTimeout = oldTimeout
		defaultCommandWaitDelay = oldWaitDelay
	}()

	scriptPath := filepath.Join(t.TempDir(), "slow-command")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nsleep 1\n"), 0o755); err != nil {
		t.Fatalf("write slow command: %v", err)
	}

	start := time.Now()
	result := (Service{}).run(scriptPath)
	elapsed := time.Since(start)

	if !errors.Is(result.Err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %#v", result)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("expected command to be bounded, ran for %s", elapsed)
	}
}

func TestDefaultRunnerBoundsHeldPipesAfterCommandExit(t *testing.T) {
	oldTimeout := defaultCommandTimeout
	oldWaitDelay := defaultCommandWaitDelay
	defaultCommandTimeout = 20 * time.Millisecond
	defaultCommandWaitDelay = 20 * time.Millisecond
	defer func() {
		defaultCommandTimeout = oldTimeout
		defaultCommandWaitDelay = oldWaitDelay
	}()

	scriptPath := filepath.Join(t.TempDir(), "held-pipe-command")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\n(sleep 1) &\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write held-pipe command: %v", err)
	}

	start := time.Now()
	result := (Service{}).run(scriptPath)
	elapsed := time.Since(start)

	if !errors.Is(result.Err, context.DeadlineExceeded) && !errors.Is(result.Err, exec.ErrWaitDelay) {
		t.Fatalf("expected bounded timeout or wait delay error, got %#v", result)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("expected held pipes to be bounded, ran for %s", elapsed)
	}
}

func TestObserveRecordedPRDegradesForGhTimeout(t *testing.T) {
	svc := Service{
		RunCommand: func(name string, args ...string) CommandResult {
			return CommandResult{Err: context.DeadlineExceeded}
		},
	}

	observation := svc.ObserveRecordedPR(ParseRecordedPRURL("https://github.com/catu-ai/easyharness/pull/203"))

	if observation.Status != PRObservationUnavailable || observation.Degraded.Code != DegradedGhTimeout {
		t.Fatalf("expected gh timeout degradation, got %#v", observation)
	}
}

func TestObserveHandoffDegradesTimedOutChecksEvenWithStdout(t *testing.T) {
	svc := Service{RunCommand: func(name string, args ...string) CommandResult {
		if len(args) >= 3 && args[0] == "pr" && args[1] == "view" {
			return CommandResult{Stdout: `{"mergeStateStatus":"CLEAN","mergeable":"MERGEABLE","headRefOid":"abc123","baseRefOid":"def456"}`}
		}
		if len(args) >= 2 && args[0] == "api" {
			return CommandResult{Stdout: `{"behind_by":0}`}
		}
		return CommandResult{
			Stdout: `[{"name":"Go Test","bucket":"pass","state":"SUCCESS"}]`,
			Err:    context.DeadlineExceeded,
		}
	}}

	observation := svc.ObserveHandoff(ParseRecordedPRURL("https://github.com/catu-ai/easyharness/pull/199"))

	if observation.CI.Status != RemoteCIUnavailable || observation.CI.Degraded.Code != DegradedGhTimeout {
		t.Fatalf("expected timeout checks degradation, got %#v", observation.CI)
	}
	if observation.Sync.Status != RemoteSyncAvailable || observation.Sync.EvidenceStatus != "fresh" {
		t.Fatalf("expected sync to remain refreshable, got %#v", observation.Sync)
	}
}

func TestObserveRecordedPRDegradesWhenGhAuthUnavailable(t *testing.T) {
	svc := Service{
		RunCommand: func(name string, args ...string) CommandResult {
			return CommandResult{
				Stderr: "gh: To get started with GitHub CLI, please run: gh auth login",
				Err:    errors.New("exit status 4"),
			}
		},
	}

	observation := svc.ObserveRecordedPR(ParseRecordedPRURL("https://github.com/catu-ai/easyharness/pull/203"))

	if observation.Status != PRObservationUnavailable || observation.Degraded.Code != DegradedGhAuthUnavailable {
		t.Fatalf("expected gh auth degradation, got %#v", observation)
	}
}

func TestObserveRecordedPRDegradesForUnreadablePR(t *testing.T) {
	svc := Service{
		RunCommand: func(name string, args ...string) CommandResult {
			return CommandResult{
				Stderr: "GraphQL: Could not resolve to a PullRequest with the number of 203.",
				Err:    errors.New("exit status 1"),
			}
		},
	}

	observation := svc.ObserveRecordedPR(ParseRecordedPRURL("https://github.com/catu-ai/easyharness/pull/203"))

	if observation.Status != PRObservationUnavailable || observation.Degraded.Code != DegradedPRUnreadable {
		t.Fatalf("expected unreadable PR degradation, got %#v", observation)
	}
}

func TestObserveRecordedPRDegradesForInvalidJSON(t *testing.T) {
	svc := Service{
		RunCommand: func(name string, args ...string) CommandResult {
			return CommandResult{Stdout: `{not-json`}
		},
	}

	observation := svc.ObserveRecordedPR(ParseRecordedPRURL("https://github.com/catu-ai/easyharness/pull/203"))

	if observation.Status != PRObservationUnavailable || observation.Degraded.Code != DegradedGhInvalidJSON {
		t.Fatalf("expected invalid JSON degradation, got %#v", observation)
	}
}

func TestObserveRecordedPRDegradesForGenericGhFailure(t *testing.T) {
	svc := Service{
		RunCommand: func(name string, args ...string) CommandResult {
			return CommandResult{
				Stderr: "temporary failure",
				Err:    errors.New("exit status 1"),
			}
		},
	}

	observation := svc.ObserveRecordedPR(ParseRecordedPRURL("https://github.com/catu-ai/easyharness/pull/203"))

	if observation.Status != PRObservationUnavailable || observation.Degraded.Code != DegradedGhCommandFailed {
		t.Fatalf("expected generic gh failure degradation, got %#v", observation)
	}
}

type commandCall struct {
	Name string
	Args []string
}

func fakePRAndChecks(prFields, checksJSON string) CommandRunner {
	return fakePRChecksAndCompare(prFields, checksJSON, 0)
}

func fakePRChecksAndCompare(prFields, checksJSON string, behindBy int) CommandRunner {
	return func(name string, args ...string) CommandResult {
		if len(args) >= 3 && args[0] == "pr" && args[1] == "view" {
			fields := strings.TrimSpace(prFields)
			fields = strings.TrimPrefix(fields, "{")
			fields = strings.TrimSuffix(fields, "}")
			return CommandResult{Stdout: `{
				"url": "https://github.com/catu-ai/easyharness/pull/199",
				"number": 199,
				"state": "OPEN",
				"isDraft": false,
				"mergeable": "MERGEABLE",
				"reviewDecision": "APPROVED",
				"headRefName": "codex/refresh",
				"headRefOid": "abc123",
				"baseRefName": "main",
				"baseRefOid": "def456",
				` + fields + `
			}`}
		}
		if len(args) >= 3 && args[0] == "pr" && args[1] == "checks" {
			return CommandResult{Stdout: checksJSON}
		}
		if len(args) >= 2 && args[0] == "api" {
			return CommandResult{Stdout: fmt.Sprintf(`{"status":"diverged","behind_by":%d}`, behindBy)}
		}
		return CommandResult{Err: errors.New("unexpected command")}
	}
}
