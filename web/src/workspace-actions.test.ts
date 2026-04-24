import { describe, expect, test } from "vitest";

import { buildWorkspaceUnwatchRequest, canUnwatchWorkspaceFromDegradedRoute } from "./workspace-actions";
import type { DashboardWorkspace } from "./types";

function dashboardWorkspace(overrides: Partial<DashboardWorkspace> = {}): DashboardWorkspace {
  return {
    workspace_key: "wk_alpha",
    workspace_name: "alpha",
    workspace_path: "/tmp/alpha",
    dashboard_state: "invalid",
    summary: "Alpha summary",
    ...overrides,
  };
}

describe("workspace actions", () => {
  test("buildWorkspaceUnwatchRequest includes the selected workspace path", () => {
    const request = buildWorkspaceUnwatchRequest(dashboardWorkspace({ workspace_path: "/tmp/beta" }));

    expect(request.url).toBe("/api/workspace/wk_alpha/unwatch");
    expect(request.init.method).toBe("POST");
    expect((request.init.headers as Record<string, string>)["Content-Type"]).toBe("application/json");
    expect(JSON.parse(String(request.init.body))).toEqual({ workspace_path: "/tmp/beta" });
  });

  test("ambiguous degraded routes cannot unwatch directly", () => {
    expect(canUnwatchWorkspaceFromDegradedRoute(dashboardWorkspace({ invalid_reason: "route_key_collision" }))).toBe(false);
    expect(canUnwatchWorkspaceFromDegradedRoute(dashboardWorkspace({ invalid_reason: "missing_plan" }))).toBe(true);
    expect(canUnwatchWorkspaceFromDegradedRoute(null)).toBe(false);
  });
});
