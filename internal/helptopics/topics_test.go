package helptopics

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	helpassets "github.com/catu-ai/easyharness/assets/help"
)

func TestRenderParentAddsGeneratedSubtopicsOutsideAssetBody(t *testing.T) {
	body, err := helpassets.Read("repo.md")
	if err != nil {
		t.Fatalf("read asset: %v", err)
	}
	if strings.Contains(body, "Available subtopics:") || strings.Contains(body, "config    Customize") {
		t.Fatalf("repo help asset should not duplicate generated subtopics:\n%s", body)
	}

	var out bytes.Buffer
	if err := Render(&out, []string{"repo"}); err != nil {
		t.Fatalf("render repo help: %v", err)
	}
	if !strings.Contains(out.String(), "Repo Help") {
		t.Fatalf("expected repo body, got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "Available subtopics:") || !strings.Contains(out.String(), "config") {
		t.Fatalf("expected generated subtopics, got:\n%s", out.String())
	}
}

func TestRenderReviewExplainsIntegratedDecisionAndRepairDelta(t *testing.T) {
	var out bytes.Buffer
	if err := Render(&out, []string{"review"}); err != nil {
		t.Fatalf("render review help: %v", err)
	}
	for _, want := range []string{
		"complete judgment",
		"controller does not select dimensions",
		"advisor subagents",
		"separate aggregate command",
		"linked delta",
		"review start --full",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("review help missing %q:\n%s", want, out.String())
		}
	}
}

func TestUnknownTopicReportsNearestSubtopics(t *testing.T) {
	var out bytes.Buffer
	err := Render(&out, []string{"repo", "missing"})
	if err == nil {
		t.Fatalf("expected unknown topic error")
	}

	var unknown UnknownTopicError
	if !errors.As(err, &unknown) {
		t.Fatalf("expected UnknownTopicError, got %T %[1]v", err)
	}
	if got := strings.Join(unknown.Nearest, " "); got != "repo" {
		t.Fatalf("nearest = %q, want repo", got)
	}
	if len(unknown.Subtopics) != 1 || strings.Join(unknown.Subtopics[0].Path, " ") != "repo config" {
		t.Fatalf("unexpected subtopics: %#v", unknown.Subtopics)
	}
}

func TestUnknownLeafChildFallsBackToRootTopics(t *testing.T) {
	var out bytes.Buffer
	err := Render(&out, []string{"repo", "config", "missing"})
	if err == nil {
		t.Fatalf("expected unknown topic error")
	}

	var unknown UnknownTopicError
	if !errors.As(err, &unknown) {
		t.Fatalf("expected UnknownTopicError, got %T %[1]v", err)
	}
	if got := strings.Join(unknown.Nearest, " "); got != "repo config" {
		t.Fatalf("nearest = %q, want repo config", got)
	}
	if !unknown.RootFallback {
		t.Fatalf("expected root fallback for leaf child")
	}
	if len(unknown.Subtopics) != 2 || strings.Join(unknown.Subtopics[0].Path, " ") != "repo" || strings.Join(unknown.Subtopics[1].Path, " ") != "review" {
		t.Fatalf("unexpected fallback topics: %#v", unknown.Subtopics)
	}
}
