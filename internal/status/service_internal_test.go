package status

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/catu-ai/easyharness/internal/plan"
)

func TestResolveStepNodeUsesOnlyImplementationFrontier(t *testing.T) {
	doc := &plan.Document{Steps: []plan.DocumentStep{
		{Title: "Step 1", Done: true},
		{Title: "Step 2", Done: false},
	}}
	index, node := resolveStepNode(doc)
	if index != 1 || node != "execution/step-2/implement" {
		t.Fatalf("expected the first unfinished implementation step, got index=%d node=%q", index, node)
	}
}

func TestApplyPlanProgressFactsDerivesStepAndAcceptanceCounts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "2026-07-14-progress.md")
	content := `---
template_version: 0.3.0
created_at: "2026-07-14T00:00:00Z"
source_type: direct_request
source_refs: []
size: S
---

# Progress

## Goal

Track progress.

### Decisions and Constraints

- Keep it derived.

## Scope

### In Scope

- Progress.

### Out of Scope

- Writes.

## Acceptance Criteria

- [x] First criterion.
- [ ] Second criterion.

## Review Focus

- Check progress.

## Deferred Items

- None.

## Work Breakdown

### Step 1: First

- Done: [x]
- Outcome: First done.
- Covers: First criterion.

### Step 2: Second

- Done: [ ]
- Outcome: Second pending.
- Covers: Second criterion.

## Validation Strategy

- Test it.

## Closeout

- Validation: PENDING_UNTIL_ARCHIVE
- Review: PENDING_UNTIL_ARCHIVE
- Delivered: PENDING_UNTIL_ARCHIVE
- Not Delivered: PENDING_UNTIL_ARCHIVE
- Follow-Up Issues: NONE
- PR: PENDING_UNTIL_ARCHIVE
- Ready: PENDING_UNTIL_ARCHIVE
- Merge Handoff: PENDING_UNTIL_ARCHIVE
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	doc, err := plan.LoadFile(path)
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	facts := &Facts{}
	applyPlanProgressFacts(facts, doc)
	if facts.CurrentStep != "Step 2: Second" || facts.CurrentStepNumber != 2 || facts.StepCompleted != 1 || facts.StepTotal != 2 {
		t.Fatalf("unexpected step progress: %#v", facts)
	}
	if facts.AcceptanceCompleted != 1 || facts.AcceptanceTotal != 2 {
		t.Fatalf("unexpected acceptance progress: %#v", facts)
	}
}
