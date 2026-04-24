package contracts

// DashboardResult is the read-only UI resource for the machine-local dashboard
// home.
type DashboardResult struct {
	// OK reports whether the machine-local watchlist was loaded.
	OK bool `json:"ok"`

	// Resource is the stable UI resource identifier.
	Resource string `json:"resource"`

	// Summary is the concise human-readable explanation of the loaded
	// dashboard model.
	Summary string `json:"summary"`

	// Groups lists watched workspaces grouped by dashboard lifecycle state.
	Groups []DashboardGroup `json:"groups"`

	// Errors lists hard failures that prevented dashboard loading.
	Errors []ErrorDetail `json:"errors,omitempty"`
}

// DashboardGroup is one dashboard lifecycle group.
type DashboardGroup struct {
	// State is the dashboard lifecycle state for every workspace in the group.
	State string `json:"state"`

	// Workspaces lists watched workspace entries in dashboard recency order.
	Workspaces []DashboardWorkspace `json:"workspaces"`
}

// DashboardWorkspace is one watched workspace summary for the dashboard home.
type DashboardWorkspace struct {
	// WorkspaceKey is the opaque deterministic route key derived from the
	// watched workspace path.
	WorkspaceKey string `json:"workspace_key"`

	// WorkspaceName is the human-readable folder/workspace name derived from the
	// watched workspace path.
	WorkspaceName string `json:"workspace_name"`

	// WorkspacePath is the canonical watched path from the watchlist record.
	WorkspacePath string `json:"workspace_path"`

	// WatchedAt is the timestamp when the workspace first entered the
	// watchlist.
	WatchedAt string `json:"watched_at,omitempty"`

	// LastSeenAt is the dashboard recency timestamp from the watchlist record.
	LastSeenAt string `json:"last_seen_at,omitempty"`

	// DashboardState is the read-time dashboard lifecycle state.
	DashboardState string `json:"dashboard_state"`

	// InvalidReason refines invalid entries without expanding the dashboard
	// lifecycle state enum.
	InvalidReason string `json:"invalid_reason,omitempty"`

	// CurrentNode is the raw harness workflow node for readable status entries.
	CurrentNode string `json:"current_node,omitempty"`

	// PlanTitle is the current or last-relevant tracked plan title when one can
	// be resolved from status artifacts.
	PlanTitle string `json:"plan_title,omitempty"`

	// Facts carries selected status facts that help the dashboard render compact
	// progress and metadata.
	Facts *StatusFacts `json:"facts,omitempty"`

	// Progress is the compact progress model for the dashboard list item.
	Progress *DashboardProgress `json:"progress,omitempty"`

	// Summary is the compact workspace summary for dashboard rows or cards.
	Summary string `json:"summary"`

	// NextAction lists the most relevant status follow-up steps for readable
	// entries.
	NextAction []NextAction `json:"next_actions,omitempty"`

	// Warnings lists non-fatal degraded-state notes for this workspace.
	Warnings []string `json:"warnings,omitempty"`

	// Blockers lists state issues that block ordinary progression for this
	// workspace.
	Blockers []ErrorDetail `json:"blockers,omitempty"`

	// Errors lists hard failures for this workspace entry.
	Errors []ErrorDetail `json:"errors,omitempty"`

	// Artifacts points to stable status artifact handles for navigation or
	// display.
	Artifacts *StatusArtifacts `json:"artifacts,omitempty"`
}

// DashboardProgress is the compact progress signal rendered on the dashboard
// home.
type DashboardProgress struct {
	// Nodes are the ordered progress nodes for this workspace. The node count
	// varies with the underlying tracked plan and workflow phase structure.
	Nodes []DashboardProgressNode `json:"nodes,omitempty"`
}

// DashboardProgressNode is one progress node in the dashboard progress signal.
type DashboardProgressNode struct {
	// Label is the tooltip/focus text for this node.
	Label string `json:"label"`

	// State is one of pending, current, or done.
	State string `json:"state"`
}

// DashboardWorkspaceResult is the route-level lookup result for one watched
// workspace key.
type DashboardWorkspaceResult struct {
	// OK reports whether the lookup completed without a top-level read failure.
	OK bool `json:"ok"`

	// Resource is the stable UI resource identifier.
	Resource string `json:"resource"`

	// Summary is the concise human-readable explanation of the lookup result.
	Summary string `json:"summary"`

	// Watched reports whether the route key currently resolves to a watched
	// workspace entry.
	Watched bool `json:"watched"`

	// Workspace is the watched workspace entry when the key resolves.
	Workspace *DashboardWorkspace `json:"workspace,omitempty"`

	// Errors lists top-level lookup errors.
	Errors []ErrorDetail `json:"errors,omitempty"`
}
