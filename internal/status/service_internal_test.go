package status

import (
	"testing"

	"github.com/catu-ai/easyharness/internal/plan"
)

func TestResolveStepNodeKeepsIntentionalUnresolvedStepReviewBinding(t *testing.T) {
	doc := &plan.Document{Steps: []plan.DocumentStep{
		{Title: "Step 1", Done: true},
		{Title: "Step 2", Done: false},
	}}
	ctx := &reviewContext{
		RoundID:         "review-001-delta",
		Trigger:         "step_closeout",
		TargetStepIndex: 0,
		Aggregated:      true,
		DecisionKnown:   true,
		Decision:        "changes_requested",
	}
	index, node := resolveStepNode(doc, ctx)
	if index != 0 || node != "execution/step-1/implement" {
		t.Fatalf("expected unresolved intentional review to bind step 1, got index=%d node=%q", index, node)
	}
}

func TestResolveStepNodeDoesNotCreateHistoricalReviewDebtAfterPass(t *testing.T) {
	doc := &plan.Document{Steps: []plan.DocumentStep{
		{Title: "Step 1", Done: true},
		{Title: "Step 2", Done: false},
	}}
	ctx := &reviewContext{
		RoundID:         "review-001-delta",
		Trigger:         "step_closeout",
		TargetStepIndex: 0,
		Aggregated:      true,
		DecisionKnown:   true,
		Decision:        "pass",
	}
	index, node := resolveStepNode(doc, ctx)
	if index != 1 || node != "execution/step-2/implement" {
		t.Fatalf("expected plan frontier to advance without review debt, got index=%d node=%q", index, node)
	}
}
