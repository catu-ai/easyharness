import { cleanup, render, screen } from "@testing-library/preact";
import { afterEach, describe, expect, test, vi } from "vitest";

import { dashboardRowKey } from "./helpers";
import { DashboardHome, WorkspaceDegradedPage } from "./pages";
import type { DashboardWorkspace, WorkspaceRouteResult } from "./types";

function dashboardWorkspace(overrides: Partial<DashboardWorkspace> = {}): DashboardWorkspace {
  return {
    workspace_key: "wk_alpha",
    workspace_name: "alpha",
    workspace_path: "/tmp/alpha",
    last_seen_at: "2026-04-23T15:00:00Z",
    dashboard_state: "active",
    current_node: "execution/step-2/implement",
    summary: "Alpha summary",
    progress: {
      nodes: [
        { label: "execution/step-1/implement · Prepare data", state: "done" },
        { label: "execution/step-1/review · Prepare data", state: "done" },
        { label: "execution/step-2/implement · Build UI", state: "current" },
      ],
    },
    ...overrides,
  };
}

describe("dashboard helpers and pages", () => {
  afterEach(() => {
    cleanup();
  });

  test("dashboard row keys stay unique for route-key collisions", () => {
    const left = dashboardWorkspace({ workspace_key: "wk_same", workspace_path: "/tmp/alpha" });
    const right = dashboardWorkspace({ workspace_key: "wk_same", workspace_path: "/tmp/beta" });

    expect(dashboardRowKey(left, 0)).not.toBe(dashboardRowKey(right, 1));
  });

  test("dashboard home renders one progress node per workflow node", () => {
    render(
      <DashboardHome
        loading={false}
        error={null}
        workspaces={[dashboardWorkspace()]}
        onOpenWorkspace={vi.fn()}
        onUnwatch={vi.fn()}
      />,
    );

    expect(document.querySelectorAll(".dashboard-progress-node")).toHaveLength(3);
    const currentNode = document.querySelector(".dashboard-progress-node.is-current");
    expect(currentNode?.getAttribute("title")).toBe("execution/step-2/implement · Build UI");
    expect(currentNode?.getAttribute("data-label")).toBe("execution/step-2/implement · Build UI");
    expect(currentNode?.getAttribute("tabindex")).toBe("0");
    expect(currentNode?.getAttribute("role")).toBe("img");
    expect(screen.getByText("alpha")).toBeTruthy();
    expect(screen.getByText("Open")).toBeTruthy();
    expect(screen.queryByText("execution/step-2/implement")).toBeNull();
  });

  test("degraded page keeps the return path and only shows unwatch for watched routes", () => {
    const watchedResult: WorkspaceRouteResult = {
      ok: true,
      resource: "workspace",
      summary: "Workspace is invalid.",
      watched: true,
      workspace: dashboardWorkspace({ dashboard_state: "invalid", invalid_reason: "missing_plan" }),
    };

    const { rerender } = render(
      <WorkspaceDegradedPage
        loading={false}
        error={null}
        result={watchedResult}
        onReturnDashboard={vi.fn()}
        onUnwatch={vi.fn()}
      />,
    );

    expect(screen.getByText("Return to dashboard")).toBeTruthy();
    expect(screen.getByText("Unwatch")).toBeTruthy();

    rerender(
      <WorkspaceDegradedPage
        loading={false}
        error={null}
        result={{
          ok: true,
          resource: "workspace",
          summary: "Workspace is not currently watched.",
          watched: false,
          workspace: null,
        }}
        onReturnDashboard={vi.fn()}
        onUnwatch={vi.fn()}
      />,
    );

    expect(screen.getByText("Return to dashboard")).toBeTruthy();
    expect(screen.queryByText("Unwatch")).toBeNull();

    rerender(
      <WorkspaceDegradedPage
        loading={false}
        error={null}
        result={{
          ok: true,
          resource: "workspace",
          summary: "Workspace route key collides.",
          watched: true,
          workspace: dashboardWorkspace({ invalid_reason: "route_key_collision" }),
        }}
        onReturnDashboard={vi.fn()}
        onUnwatch={vi.fn()}
      />,
    );

    expect(screen.getByText("Return to dashboard")).toBeTruthy();
    expect(screen.queryByText("Unwatch")).toBeNull();
  });
});
