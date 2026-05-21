package remote

import (
	"errors"
	"os/exec"
	"reflect"
	"testing"
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
				"baseRefName": "main"
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
			"--json", "url,number,state,headRefName,headRefOid,baseRefName",
		},
	}}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("unexpected gh calls:\nwant %#v\ngot  %#v", want, calls)
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
