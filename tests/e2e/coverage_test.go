package e2e_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type transitionFamily struct {
	ID             string
	From           string
	To             string
	Driver         string
	RequiredInputs string
}

type scenarioCoverage struct {
	ID            string
	TestName      string
	TransitionIDs []string
}

var canonicalTransitionFamilies = []transitionFamily{
	{ID: "idle_to_plan", From: "idle", To: "plan", Driver: "Active plan presence", RequiredInputs: "Exactly one tracked active plan exists."},
	{ID: "plan_to_step_implement", From: "plan", To: "execution/step-<n>/implement", Driver: "harness execute start", RequiredInputs: "The plan has explicit human approval and an unfinished step."},
	{ID: "plan_to_coordinate", From: "plan", To: "execution/coordinate", Driver: "harness execute start", RequiredInputs: "The coordinated root has explicit human approval."},
	{ID: "step_implement_to_next_step_implement", From: "execution/step-<n>/implement", To: "execution/step-<m>/implement", Driver: "Plan edit", RequiredInputs: "Step `<n>` becomes complete and another unfinished step exists."},
	{ID: "step_implement_to_finalize_review", From: "execution/step-<n>/implement", To: "execution/finalize/review", Driver: "Plan edit", RequiredInputs: "Every step is complete. Formal review has not yet established clean coverage for the candidate."},
	{ID: "coordinate_to_coordinate", From: "execution/coordinate", To: "execution/coordinate", Driver: "Subplan edit", RequiredInputs: "At least one subplan is incomplete, waiting on a sibling, or structurally blocked."},
	{ID: "coordinate_to_finalize_review", From: "execution/coordinate", To: "execution/finalize/review", Driver: "Subplan edit", RequiredInputs: "At least one subplan exists, every subplan step and Result is complete, and the package graph is valid."},
	{ID: "finalize_review_to_finalize_fix", From: "execution/finalize/review", To: "execution/finalize/fix", Driver: "harness review submit", RequiredInputs: "The integrated reviewer reports blocking findings or a conservative failure."},
	{ID: "finalize_review_to_finalize_archive", From: "execution/finalize/review", To: "execution/finalize/archive", Driver: "`harness review submit` or derived status", RequiredInputs: "A clean full root, optionally extended by clean linked deltas, covers current candidate HEAD."},
	{ID: "finalize_review_abort_recovery", From: "execution/finalize/review", To: "execution/finalize/review", Driver: "harness review abort", RequiredInputs: "The exact active unfinished round is preserved as aborted history and its active pointer is cleared; completed coverage is unchanged."},
	{ID: "finalize_fix_to_finalize_review", From: "execution/finalize/fix", To: "execution/finalize/review", Driver: "harness review start", RequiredInputs: "A committed repair starts an inferred linked delta, or `--full` explicitly resets materially invalidated coverage."},
	{ID: "finalize_fix_to_new_step_implement", From: "execution/finalize/fix", To: "execution/step-<m>/implement", Driver: "Plan edit after `reopen --mode new-step`", RequiredInputs: "The first new unfinished step is added."},
	{ID: "finalize_archive_to_publish", From: "execution/finalize/archive", To: "execution/finalize/publish", Driver: "harness archive", RequiredInputs: "Acceptance, ordinary root steps or the complete coordinated package, Closeout, and finalize coverage are complete."},
	{ID: "publish_to_await_merge", From: "execution/finalize/publish", To: "execution/finalize/await_merge", Driver: "Evidence", RequiredInputs: "Current publish, CI, and sync evidence supports merge readiness, and the archived branch differs from the reviewed candidate only by the allowed archive move and Closeout update."},
	{ID: "archived_to_finalize_fix", From: "`execution/finalize/publish` or `execution/finalize/await_merge`", To: "execution/finalize/fix", Driver: "`harness reopen` with `finalize-fix`, or `new-step` for standard/lightweight", RequiredInputs: "Feedback or remote change invalidates the archived candidate."},
	{ID: "archived_to_coordinate", From: "`execution/finalize/publish` or `execution/finalize/await_merge`", To: "execution/coordinate", Driver: "harness reopen --mode new-step", RequiredInputs: "A coordinated candidate needs at least one new or reopened subplan before returning to finalize."},
	{ID: "await_merge_to_land", From: "execution/finalize/await_merge", To: "land", Driver: "harness land", RequiredInputs: "Human merge approval exists and the PR has merged."},
	{ID: "land_to_idle", From: "land", To: "idle", Driver: "harness land complete", RequiredInputs: "Required post-merge bookkeeping and release verification are complete."},
}

var currentScenarioCoverage = []scenarioCoverage{
	{
		ID:       "review_workflow",
		TestName: "TestReviewWorkflowWithBuiltBinary",
		TransitionIDs: []string{
			"idle_to_plan",
			"plan_to_step_implement",
			"step_implement_to_next_step_implement",
			"step_implement_to_finalize_review",
			"finalize_review_to_finalize_fix",
			"finalize_fix_to_finalize_review",
			"finalize_review_to_finalize_archive",
		},
	},
	{
		ID:            "review_abort_recovery",
		TestName:      "TestReviewAbortRecoveryWithBuiltBinary",
		TransitionIDs: []string{"finalize_review_abort_recovery"},
	},
	{
		ID:       "archive_reopen_finalize_fix",
		TestName: "TestArchiveReopenFinalizeFixWithBuiltBinary",
		TransitionIDs: []string{
			"plan_to_step_implement",
			"step_implement_to_next_step_implement",
			"step_implement_to_finalize_review",
			"finalize_review_to_finalize_archive",
			"finalize_archive_to_publish",
			"archived_to_finalize_fix",
		},
	},
	{
		ID:       "reopen_new_step",
		TestName: "TestReopenNewStepWithBuiltBinary",
		TransitionIDs: []string{
			"plan_to_step_implement",
			"step_implement_to_next_step_implement",
			"step_implement_to_finalize_review",
			"finalize_review_to_finalize_archive",
			"finalize_archive_to_publish",
			"archived_to_finalize_fix",
			"finalize_fix_to_new_step_implement",
		},
	},
	{
		ID:       "publish_handoff",
		TestName: "TestPublishHandoffWithBuiltBinary",
		TransitionIDs: []string{
			"plan_to_step_implement",
			"step_implement_to_next_step_implement",
			"step_implement_to_finalize_review",
			"finalize_review_to_finalize_archive",
			"finalize_archive_to_publish",
			"publish_to_await_merge",
		},
	},
	{
		ID:       "lightweight_workflow",
		TestName: "TestLightweightWorkflowWithBuiltBinary",
		TransitionIDs: []string{
			"idle_to_plan",
			"plan_to_step_implement",
			"step_implement_to_finalize_review",
			"finalize_review_to_finalize_archive",
			"finalize_archive_to_publish",
			"publish_to_await_merge",
		},
	},
	{
		ID:       "coordinated_workflow",
		TestName: "TestCoordinatedWorkflowWithBuiltBinary",
		TransitionIDs: []string{
			"idle_to_plan",
			"plan_to_coordinate",
			"coordinate_to_coordinate",
			"coordinate_to_finalize_review",
			"finalize_review_to_finalize_archive",
			"finalize_archive_to_publish",
			"archived_to_coordinate",
		},
	},
	{
		ID:       "land_workflow",
		TestName: "TestLandWorkflowWithBuiltBinary",
		TransitionIDs: []string{
			"plan_to_step_implement",
			"step_implement_to_next_step_implement",
			"step_implement_to_finalize_review",
			"finalize_review_to_finalize_archive",
			"finalize_archive_to_publish",
			"publish_to_await_merge",
			"await_merge_to_land",
			"land_to_idle",
		},
	},
	{
		ID:       "await_merge_reopen_finalize_fix",
		TestName: "TestAwaitMergeReopenFinalizeFixWithBuiltBinary",
		TransitionIDs: []string{
			"idle_to_plan",
			"plan_to_step_implement",
			"step_implement_to_next_step_implement",
			"step_implement_to_finalize_review",
			"finalize_review_to_finalize_archive",
			"finalize_archive_to_publish",
			"publish_to_await_merge",
			"archived_to_finalize_fix",
		},
	},
	{
		ID:       "await_merge_reopen_new_step",
		TestName: "TestAwaitMergeReopenNewStepWithBuiltBinary",
		TransitionIDs: []string{
			"plan_to_step_implement",
			"step_implement_to_next_step_implement",
			"step_implement_to_finalize_review",
			"finalize_review_to_finalize_archive",
			"finalize_archive_to_publish",
			"publish_to_await_merge",
			"archived_to_finalize_fix",
			"finalize_fix_to_new_step_implement",
		},
	},
}

func TestCanonicalTransitionCoverageCatalogIsWellFormed(t *testing.T) {
	transitionIDs := map[string]bool{}
	transitionKeys := map[string]bool{}
	for _, family := range canonicalTransitionFamilies {
		if family.ID == "" {
			t.Fatal("transition family id must be non-empty")
		}
		if family.From == "" || family.To == "" || family.Driver == "" || family.RequiredInputs == "" {
			t.Fatalf("canonical transition family must include from, to, driver, and required inputs: %#v", family)
		}
		if transitionIDs[family.ID] {
			t.Fatalf("duplicate transition family id %q", family.ID)
		}
		transitionIDs[family.ID] = true
		key := transitionFamilyKey(family.From, family.To, family.Driver, family.RequiredInputs)
		if transitionKeys[key] {
			t.Fatalf("duplicate canonical transition family triple %q", key)
		}
		transitionKeys[key] = true
	}

	scenarioIDs := map[string]bool{}
	for _, scenario := range currentScenarioCoverage {
		if scenario.ID == "" || scenario.TestName == "" {
			t.Fatalf("scenario coverage entry must have id and test name: %#v", scenario)
		}
		if scenarioIDs[scenario.ID] {
			t.Fatalf("duplicate scenario coverage id %q", scenario.ID)
		}
		scenarioIDs[scenario.ID] = true

		seenInScenario := map[string]bool{}
		for _, transitionID := range scenario.TransitionIDs {
			if !transitionIDs[transitionID] {
				t.Fatalf("scenario %q references unknown transition family %q", scenario.ID, transitionID)
			}
			if seenInScenario[transitionID] {
				t.Fatalf("scenario %q references transition family %q more than once", scenario.ID, transitionID)
			}
			seenInScenario[transitionID] = true
		}
	}
}

func TestScenarioCoverageSpansEveryCanonicalTransitionFamily(t *testing.T) {
	covered := map[string]bool{}
	for _, scenario := range currentScenarioCoverage {
		for _, transitionID := range scenario.TransitionIDs {
			covered[transitionID] = true
		}
	}

	var missing []string
	for _, family := range canonicalTransitionFamilies {
		if !covered[family.ID] {
			missing = append(missing, family.ID)
		}
	}

	if len(missing) != 0 {
		t.Fatalf("scenario coverage is missing canonical transition families: %v", missing)
	}
}

func TestCanonicalTransitionCatalogMatchesTrackedSpecMatrix(t *testing.T) {
	specTransitions := loadTrackedSpecTransitions(t)

	specKeys := map[string]bool{}
	for _, transition := range specTransitions {
		key := transitionFamilyKey(transition.From, transition.To, transition.Driver, transition.RequiredInputs)
		if specKeys[key] {
			t.Fatalf("tracked spec transition matrix contains duplicate transition triple %q", key)
		}
		specKeys[key] = true
	}

	canonicalKeys := map[string]bool{}
	for _, family := range canonicalTransitionFamilies {
		key := transitionFamilyKey(family.From, family.To, family.Driver, family.RequiredInputs)
		canonicalKeys[key] = true
	}

	var missingFromCatalog []string
	for key := range specKeys {
		if !canonicalKeys[key] {
			missingFromCatalog = append(missingFromCatalog, key)
		}
	}

	var missingFromSpec []string
	for key := range canonicalKeys {
		if !specKeys[key] {
			missingFromSpec = append(missingFromSpec, key)
		}
	}

	if len(missingFromCatalog) != 0 || len(missingFromSpec) != 0 {
		t.Fatalf("canonical transition catalog drifted from docs/specs/state-transitions.md; missing from catalog: %v; missing from spec: %v", missingFromCatalog, missingFromSpec)
	}
}

func transitionFamilyKey(from, to, driver, requiredInputs string) string {
	return from + " => " + to + " [" + driver + "] {" + requiredInputs + "}"
}

func loadTrackedSpecTransitions(t *testing.T) []transitionFamily {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to locate coverage test file path")
	}

	specPath := filepath.Join(filepath.Dir(filename), "..", "..", "docs", "specs", "state-transitions.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read transition spec: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	inStatePreserving := false
	var transitions []transitionFamily
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)

		if strings.HasPrefix(line, "## ") {
			inStatePreserving = line == "## State-Preserving Updates"
			continue
		}

		if strings.HasPrefix(line, "|") {
			columns := splitMarkdownRow(line)
			if len(columns) < 4 {
				continue
			}
			if columns[0] == "From" || strings.HasPrefix(columns[0], "---") {
				continue
			}

			transitions = append(transitions, transitionFamily{
				From:           trimInlineCode(columns[0]),
				To:             trimInlineCode(columns[1]),
				Driver:         trimInlineCode(columns[2]),
				RequiredInputs: trimInlineCode(columns[3]),
			})
			continue
		}

		if !inStatePreserving || !strings.HasPrefix(line, "- `") {
			continue
		}

		transitionText, ok := extractInlineCode(line)
		if !ok {
			t.Fatalf("failed to parse state-preserving transition from line %q", line)
		}
		from, to, ok := strings.Cut(transitionText, " -> ")
		if !ok {
			t.Fatalf("state-preserving transition line missing arrow: %q", line)
		}
		transitions = append(transitions, transitionFamily{
			From:           from,
			To:             to,
			Driver:         "state-preserving",
			RequiredInputs: "state-preserving update",
		})
	}

	return transitions
}

func splitMarkdownRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")

	parts := strings.Split(trimmed, "|")
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
	}
	return parts
}

func trimInlineCode(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "`") && strings.HasSuffix(trimmed, "`") && strings.Count(trimmed, "`") == 2 {
		return strings.TrimSuffix(strings.TrimPrefix(trimmed, "`"), "`")
	}
	return trimmed
}

func extractInlineCode(line string) (string, bool) {
	start := strings.Index(line, "`")
	if start == -1 {
		return "", false
	}
	end := strings.Index(line[start+1:], "`")
	if end == -1 {
		return "", false
	}
	return line[start+1 : start+1+end], true
}
