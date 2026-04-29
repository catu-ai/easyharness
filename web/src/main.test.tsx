import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/preact";
import { useState } from "preact/hooks";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

import { App } from "./main";
import { PlanWorkspace, ReviewWorkspace, TimelineWorkspace } from "./pages";
import type {
  PlanResult,
  PlanWorkspaceState,
  ReviewResult,
  ReviewWorkspaceState,
  StatusResult,
  TimelineResult,
  TimelineWorkspaceState,
  WorkspaceRouteResult,
} from "./types";

const workspaceResult: WorkspaceRouteResult = {
  ok: true,
  resource: "workspace",
  summary: "Workspace is active.",
  watched: true,
  workspace: {
    workspace_key: "wk_alpha",
    workspace_name: "alpha",
    workspace_path: "/tmp/alpha",
    dashboard_state: "active",
    current_node: "execution/step-1/implement",
    summary: "Alpha summary",
  },
};

const statusResult: StatusResult = {
  ok: true,
  command: "status",
  summary: "Step 1 is active.",
  state: { current_node: "execution/step-1/implement" },
  next_actions: [],
  warnings: [],
  blockers: [],
  errors: [],
};

const planResult: PlanResult = {
  ok: true,
  resource: "plan",
  summary: "Plan loaded.",
  warnings: [],
  document: {
    title: "Warm Plan",
    path: "docs/plans/active/warm.md",
    markdown: "# Warm Plan\n\n## Scope\n\nKeep state.\n\n## Validation\n\nTest it.",
    headings: [
      { id: "scope", label: "Scope", level: 2, anchor: "scope" },
      { id: "validation", label: "Validation", level: 2, anchor: "validation" },
    ],
  },
  supplements: null,
};

const supplementsOnlyPlanResult: PlanResult = {
  ok: true,
  resource: "plan",
  summary: "Plan supplements loaded.",
  warnings: [],
  document: null,
  supplements: {
    id: "supplements",
    kind: "directory",
    label: "supplements",
    path: "docs/plans/active/supplements",
    children: [
      {
        id: "notes",
        kind: "file",
        label: "notes.md",
        path: "docs/plans/active/supplements/notes.md",
        preview: {
          status: "supported",
          content_type: "text",
          content: "supplement notes",
          byte_size: 16,
        },
      },
    ],
  },
};

const timelineResult: TimelineResult = {
  ok: true,
  resource: "timeline",
  summary: "Timeline loaded.",
  events: [
    {
      event_id: "event-new",
      sequence: 2,
      recorded_at: "2026-04-28T10:00:00Z",
      kind: "execute",
      command: "new event",
      summary: "New event",
      plan_stem: "warm",
      input: { newer: true },
    },
    {
      event_id: "event-old",
      sequence: 1,
      recorded_at: "2026-04-28T09:00:00Z",
      kind: "plan",
      command: "old event",
      summary: "Old event",
      plan_stem: "warm",
      input: { older: true },
    },
  ],
};

const reviewResult: ReviewResult = {
  ok: true,
  resource: "review",
  summary: "Review loaded.",
  warnings: [],
  artifacts: {},
  rounds: [
    {
      round_id: "review-002-delta",
      kind: "delta",
      review_title: "Second review",
      status: "passed",
      status_summary: "Passed.",
      reviewers: [{ slot: "tests", name: "Tests", status: "submitted", summary: "Looks good." }],
      artifacts: [{ label: "submission", path: ".local/review/submission.json", content_type: "json", content: { ok: true } }],
    },
    {
      round_id: "review-001-full",
      kind: "full",
      review_title: "First review",
      status: "passed",
      status_summary: "Passed.",
      reviewers: [{ slot: "ui", name: "UI", status: "submitted", summary: "Looks good." }],
      artifacts: [
        { label: "notes", path: ".local/review/notes.md", content_type: "text", content: "notes" },
        { label: "trace", path: ".local/review/trace.md", content_type: "text", content: "trace payload" },
      ],
    },
  ],
};

let currentPlanResult: PlanResult = planResult;

function mockApi() {
  vi.stubGlobal(
    "fetch",
    vi.fn((input: RequestInfo | URL) => {
      const path = String(input);
      const payloadByPath: Record<string, unknown> = {
        "/api/workspace/wk_alpha": workspaceResult,
        "/api/workspace/wk_alpha/status": statusResult,
        "/api/workspace/wk_alpha/plan": currentPlanResult,
        "/api/workspace/wk_alpha/timeline": timelineResult,
        "/api/workspace/wk_alpha/review": reviewResult,
      };
      const payload = payloadByPath[path];
      if (!payload) {
        return Promise.reject(new Error(`unexpected path ${path}`));
      }
      return Promise.resolve({ ok: true, json: async () => payload } as Response);
    }),
  );
}

function activePlanTreeText(): string {
  return document.querySelector(".plan-tree-row.is-active .plan-tree-text")?.textContent ?? "";
}

function planFetchCount(): number {
  return vi.mocked(fetch).mock.calls.filter(([input]) => String(input) === "/api/workspace/wk_alpha/plan").length;
}

function clickPlanTreeLabel(label: string) {
  const target = Array.from(document.querySelectorAll<HTMLButtonElement>(".plan-tree-label")).find(
    (nextElement) => nextElement.querySelector(".plan-tree-text")?.textContent === label,
  );
  if (!target) throw new Error(`Missing plan tree label ${label}`);
  fireEvent.click(target);
}

function activeInspectorTabText(): string {
  return document.querySelector(".inspector-tab.is-active")?.textContent ?? "";
}

function activeArtifactTabText(): string {
  return Array.from(document.querySelectorAll(".raw-json-overlay .inspector-tab.is-active")).at(-1)?.textContent ?? "";
}

function activeArtifactBodyText(): string {
  return document.querySelector(".raw-json-overlay .artifact-panel .inspector-json")?.textContent ?? "";
}

function activeExplorerTitleText(): string {
  return document.querySelector(".explorer-item.is-active .explorer-item-title")?.textContent ?? "";
}

function explorerHasTitle(title: string): boolean {
  return Array.from(document.querySelectorAll(".explorer-item-title")).some((nextElement) => nextElement.textContent === title);
}

function clickExplorerItem(title: string) {
  const target = Array.from(document.querySelectorAll<HTMLButtonElement>(".explorer-item")).find(
    (nextElement) => nextElement.querySelector(".explorer-item-title")?.textContent === title,
  );
  if (!target) throw new Error(`Missing explorer item ${title}`);
  fireEvent.click(target);
}

function PlanStateHarness() {
  const [mounted, setMounted] = useState(true);
  const [state, setState] = useState<PlanWorkspaceState>({ selectedNodeId: null, expandedNodeIds: null });
  return (
    <>
      <button type="button" onClick={() => setMounted((current) => !current)}>
        Toggle Plan
      </button>
      {mounted ? (
        <PlanWorkspace
          loading={false}
          error={null}
          summary={planResult.summary}
          document={planResult.document ?? null}
          supplements={planResult.supplements ?? null}
          warnings={planResult.warnings ?? []}
          state={state}
          onStateChange={setState}
        />
      ) : null}
    </>
  );
}

function TimelineStateHarness() {
  const [mounted, setMounted] = useState(true);
  const [state, setState] = useState<TimelineWorkspaceState>({ selectedEventId: null, selectedTab: "event" });
  return (
    <>
      <button type="button" onClick={() => setMounted((current) => !current)}>
        Toggle Timeline
      </button>
      {mounted ? <TimelineWorkspace loading={false} error={null} events={timelineResult.events ?? []} state={state} onStateChange={setState} /> : null}
    </>
  );
}

function ReviewStateHarness() {
  const [mounted, setMounted] = useState(true);
  const [state, setState] = useState<ReviewWorkspaceState>({
    selectedRoundId: null,
    selectedDetailTab: "summary",
    selectedArtifactKey: null,
    showArtifacts: false,
  });
  return (
    <>
      <button type="button" onClick={() => setMounted((current) => !current)}>
        Toggle Review
      </button>
      {mounted ? (
        <ReviewWorkspace
          loading={false}
          error={null}
          summary={reviewResult.summary}
          rounds={reviewResult.rounds ?? []}
          warnings={reviewResult.warnings ?? []}
          artifacts={[]}
          state={state}
          onStateChange={setState}
        />
      ) : null}
    </>
  );
}

describe("workbench page state continuity", () => {
  beforeEach(() => {
    currentPlanResult = planResult;
    mockApi();
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
    window.history.replaceState({}, "", "/");
  });

  test("keeps Plan selection warm across tab switches while refetching data", async () => {
    window.history.pushState({}, "", "/workspace/wk_alpha/plan");
    render(
      <App
        initialPlanWorkspaceState={{
          selectedNodeId: "heading:scope",
          expandedNodeIds: ["document:docs/plans/active/warm.md"],
        }}
      />,
    );

    await waitFor(() => expect(document.querySelector(".plan-tree-text")?.textContent).toBe("Warm Plan"));
    const initialPlanFetches = planFetchCount();
    await waitFor(() => expect(activePlanTreeText()).toBe("Scope"));

    fireEvent.click(screen.getByLabelText("Timeline"));
    await waitFor(() => expect(explorerHasTitle("new event")).toBe(true));
    fireEvent.click(screen.getByLabelText("Plan"));

    await waitFor(() => expect(document.querySelector(".plan-tree-text")?.textContent).toBe("Warm Plan"));
    await waitFor(() => expect(planFetchCount()).toBeGreaterThan(initialPlanFetches));
    await waitFor(() => expect(activePlanTreeText()).toBe("Scope"));
  });

  test("keeps supplements-only Plan child selection warm across tab switches", async () => {
    currentPlanResult = supplementsOnlyPlanResult;
    window.history.pushState({}, "", "/workspace/wk_alpha/plan");
    render(
      <App
        initialPlanWorkspaceState={{
          selectedNodeId: "file:docs/plans/active/supplements/notes.md",
          expandedNodeIds: ["directory:docs/plans/active/supplements"],
        }}
      />,
    );

    await waitFor(() => expect(document.querySelector(".plan-tree-text")?.textContent).toBe("supplements"));
    await waitFor(() => expect(activePlanTreeText()).toBe("notes.md"));

    fireEvent.click(screen.getByLabelText("Timeline"));
    await waitFor(() => expect(explorerHasTitle("new event")).toBe(true));
    fireEvent.click(screen.getByLabelText("Plan"));

    await waitFor(() => expect(activePlanTreeText()).toBe("notes.md"));
  });

  test("keeps Timeline event and detail tab warm across tab switches", async () => {
    window.history.pushState({}, "", "/workspace/wk_alpha/timeline");
    render(
      <App
        initialTimelineWorkspaceState={{
          selectedEventId: "event-old",
          selectedTab: "input",
        }}
      />,
    );

    await waitFor(() => expect(explorerHasTitle("old event")).toBe(true));
    await waitFor(() => expect(activeExplorerTitleText()).toBe("old event"));
    await waitFor(() => expect(activeInspectorTabText()).toBe("Input"));

    fireEvent.click(screen.getByLabelText("Plan"));
    await waitFor(() => expect(document.querySelector(".plan-tree-text")?.textContent).toBe("Warm Plan"));
    fireEvent.click(screen.getByLabelText("Timeline"));

    await waitFor(() => expect(explorerHasTitle("old event")).toBe(true));
    await waitFor(() => expect(activeExplorerTitleText()).toBe("old event"));
    expect(activeInspectorTabText()).toBe("Input");
  });

  test("keeps Review round, detail tab, and artifacts panel warm across tab switches", async () => {
    window.history.pushState({}, "", "/workspace/wk_alpha/review");
    render(
      <App
        initialReviewWorkspaceState={{
          selectedRoundId: "review-001-full",
          selectedDetailTab: "ui",
          selectedArtifactKey: ".local/review/notes.md",
          showArtifacts: true,
        }}
      />,
    );

    await waitFor(() => expect(explorerHasTitle("First review")).toBe(true));
    await waitFor(() => expect(activeExplorerTitleText()).toBe("First review"));
    await waitFor(() => expect(screen.getAllByText("notes").length).toBeGreaterThan(0));

    fireEvent.click(screen.getByLabelText("Plan"));
    await waitFor(() => expect(document.querySelector(".plan-tree-text")?.textContent).toBe("Warm Plan"));
    fireEvent.click(screen.getByLabelText("Review"));

    await waitFor(() => expect(explorerHasTitle("First review")).toBe(true));
    await waitFor(() => expect(activeExplorerTitleText()).toBe("First review"));
    expect(activeInspectorTabText()).toBe("UI");
    expect(screen.getAllByText("notes").length).toBeGreaterThan(0);
  });

  test("Plan controls write lifted state that survives page remount", async () => {
    render(<PlanStateHarness />);

    await waitFor(() => expect(document.querySelector(".plan-tree-text")?.textContent).toBe("Warm Plan"));
    clickPlanTreeLabel("Scope");
    await waitFor(() => expect(activePlanTreeText()).toBe("Scope"));

    fireEvent.click(screen.getByRole("button", { name: "Toggle Plan" }));
    await waitFor(() => expect(document.querySelector(".plan-tree")).toBeNull());
    fireEvent.click(screen.getByRole("button", { name: "Toggle Plan" }));

    await waitFor(() => expect(activePlanTreeText()).toBe("Scope"));
  });

  test("Timeline controls write lifted state that survives page remount", async () => {
    render(<TimelineStateHarness />);

    await waitFor(() => expect(explorerHasTitle("old event")).toBe(true));
    clickExplorerItem("old event");
    await waitFor(() => expect(activeExplorerTitleText()).toBe("old event"));
    fireEvent.click(screen.getByRole("tab", { name: "Input" }));
    await waitFor(() => expect(activeInspectorTabText()).toBe("Input"));

    fireEvent.click(screen.getByRole("button", { name: "Toggle Timeline" }));
    await waitFor(() => expect(document.querySelector(".explorer-list")).toBeNull());
    fireEvent.click(screen.getByRole("button", { name: "Toggle Timeline" }));

    await waitFor(() => expect(activeExplorerTitleText()).toBe("old event"));
    expect(activeInspectorTabText()).toBe("Input");
  });

  test("Review controls write lifted state that survives page remount", async () => {
    render(<ReviewStateHarness />);

    await waitFor(() => expect(explorerHasTitle("First review")).toBe(true));
    clickExplorerItem("First review");
    await waitFor(() => expect(activeExplorerTitleText()).toBe("First review"));
    fireEvent.click(screen.getByRole("tab", { name: "UI" }));
    await waitFor(() => expect(activeInspectorTabText()).toBe("UI"));
    fireEvent.click(screen.getByRole("button", { name: "Artifacts" }));
    await waitFor(() => expect(screen.getAllByText("notes").length).toBeGreaterThan(0));
    fireEvent.click(screen.getByRole("tab", { name: "trace" }));
    await waitFor(() => expect(activeArtifactTabText()).toBe("trace"));
    await waitFor(() => expect(activeArtifactBodyText()).toBe("trace payload"));

    fireEvent.click(screen.getByRole("button", { name: "Toggle Review" }));
    await waitFor(() => expect(document.querySelector(".explorer-list")).toBeNull());
    fireEvent.click(screen.getByRole("button", { name: "Toggle Review" }));

    await waitFor(() => expect(activeExplorerTitleText()).toBe("First review"));
    expect(activeInspectorTabText()).toBe("UI");
    expect(activeArtifactTabText()).toBe("trace");
    expect(activeArtifactBodyText()).toBe("trace payload");
  });

  test("falls back cleanly when remembered Plan ids are no longer present", async () => {
    window.history.pushState({}, "", "/workspace/wk_alpha/plan");
    render(
      <App
        initialPlanWorkspaceState={{
          selectedNodeId: "heading:missing",
          expandedNodeIds: ["document:docs/plans/active/warm.md", "heading:missing"],
        }}
      />,
    );

    await waitFor(() => expect(document.querySelector(".plan-tree-text")?.textContent).toBe("Warm Plan"));
    await waitFor(() => expect(activePlanTreeText()).toBe("Warm Plan"));
  });

  test("falls back cleanly when remembered Timeline ids are no longer present", async () => {
    window.history.pushState({}, "", "/workspace/wk_alpha/timeline");
    render(
      <App
        initialTimelineWorkspaceState={{
          selectedEventId: "missing-event",
          selectedTab: "missing-tab",
        }}
      />,
    );

    await waitFor(() => expect(explorerHasTitle("new event")).toBe(true));
    await waitFor(() => expect(activeExplorerTitleText()).toBe("new event"));
    await waitFor(() => expect(activeInspectorTabText()).toBe("Event"));
  });

  test("falls back cleanly when remembered Review ids are no longer present", async () => {
    window.history.pushState({}, "", "/workspace/wk_alpha/review");
    render(
      <App
        initialReviewWorkspaceState={{
          selectedRoundId: "missing-round",
          selectedDetailTab: "missing-tab",
          selectedArtifactKey: "missing-artifact",
          showArtifacts: true,
        }}
      />,
    );

    await waitFor(() => expect(explorerHasTitle("Second review")).toBe(true));
    await waitFor(() => expect(activeExplorerTitleText()).toBe("Second review"));
    await waitFor(() => expect(activeInspectorTabText()).toBe("Tests"));
  });

  test("falls back cleanly when a remembered review artifact id is no longer present", async () => {
    window.history.pushState({}, "", "/workspace/wk_alpha/review");
    render(
      <App
        initialReviewWorkspaceState={{
          selectedRoundId: "review-001-full",
          selectedDetailTab: "ui",
          selectedArtifactKey: "missing-artifact",
          showArtifacts: true,
        }}
      />,
    );

    await waitFor(() => expect(activeExplorerTitleText()).toBe("First review"));
    await waitFor(() => expect(screen.getAllByText("notes").length).toBeGreaterThan(0));
  });
});
