import type { DashboardWorkspace } from "./types";

export function buildWorkspaceUnwatchRequest(workspace: DashboardWorkspace): { url: string; init: RequestInit } {
  return {
    url: `/api/workspace/${workspace.workspace_key}/unwatch`,
    init: {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ workspace_path: workspace.workspace_path }),
    },
  };
}

export function canUnwatchWorkspaceFromDegradedRoute(workspace: DashboardWorkspace | null | undefined): boolean {
  return workspace != null && workspace.invalid_reason !== "route_key_collision";
}
