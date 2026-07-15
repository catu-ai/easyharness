export type Page = "status" | "plan" | "timeline" | "review";

export type Tone = "good" | "danger" | "warning" | "muted";
export type LiveFreshnessKind = "idle" | "connecting" | "updating" | "live" | "stale" | "disconnected";
export type LiveFreshness = {
  kind: LiveFreshnessKind;
  label: string;
  detail: string;
  tone: Tone;
  lastSuccessAt: string | null;
};

export type PageDef = { id: Page; label: string; href: string };

export type SectionLink = { id: string; label: string; meta?: string };

export type NextAction = {
  command: string | null;
  description: string;
};

export type ErrorDetail = {
  path: string;
  message: string;
};

export type StatusRemoteDegradation = {
  code: string;
  message?: string;
};

export type StatusFacts = {
  current_step?: string;
  current_step_number?: number;
  step_completed?: number;
  step_total?: number;
  acceptance_completed?: number;
  acceptance_total?: number;
  revision?: number;
  reopen_mode?: string;
  review_kind?: string;
  review_status?: string;
  reviewed_head_sha?: string;
  archive_blocker_count?: number;
  evidence?: {
    recorded?: {
      publish?: {
        status?: string;
        pr_url?: string;
      } | null;
      ci?: {
        status?: string;
      } | null;
      sync?: {
        status?: string;
      } | null;
    } | null;
    remote?: {
      observation?: string;
      assessment?: string;
      message?: string;
      pr?: {
        state?: string;
        draft: boolean;
      } | null;
      ci?: {
        status?: string;
      } | null;
      sync?: {
        status?: string;
      } | null;
      degraded?: StatusRemoteDegradation[] | null;
    } | null;
  } | null;
  land_pr_url?: string;
  land_commit?: string;
};

export type StatusResult = {
  ok: boolean;
  command: string;
  summary: string;
  state?: {
    current_node?: string;
  };
  facts?: StatusFacts | null;
  artifacts?: Record<string, unknown> | null;
  next_actions?: NextAction[] | null;
  blockers?: ErrorDetail[] | null;
  warnings?: string[] | null;
  errors?: ErrorDetail[] | null;
};

export type PlanHeading = {
  id: string;
  label: string;
  level: number;
  anchor: string;
  children?: PlanHeading[] | null;
};

export type PlanPreview = {
  status: string;
  content_type?: string;
  content?: string;
  reason?: string;
  byte_size?: number;
  extension?: string;
};

export type PlanNode = {
  id: string;
  kind: "directory" | "file";
  label: string;
  path?: string;
  children?: PlanNode[] | null;
  preview?: PlanPreview | null;
};

export type PlanDocument = {
  title: string;
  path: string;
  markdown: string;
  headings: PlanHeading[];
};

export type PlanResult = {
  ok: boolean;
  resource: string;
  summary: string;
  artifacts?: {
    plan_path?: string;
    supplements_path?: string;
  } | null;
  document?: PlanDocument | null;
  supplements?: PlanNode | null;
  warnings?: string[] | null;
  errors?: ErrorDetail[] | null;
};

export type PlanWorkspaceState = {
  selectedNodeId: string | null;
  expandedNodeIds: string[] | null;
};

export type TimelineDetail = {
  key: string;
  value: string;
};

export type TimelineArtifactRef = {
  label: string;
  value: string;
  path?: string;
  content_type?: string;
  content?: unknown;
};

export type TimelineEvent = {
  event_id: string;
  sequence: number;
  recorded_at: string;
  kind: string;
  command: string;
  summary: string;
  synthetic?: boolean;
  plan_path?: string;
  plan_stem: string;
  revision?: number;
  from_node?: string;
  to_node?: string;
  details?: TimelineDetail[] | null;
  artifact_refs?: TimelineArtifactRef[] | null;
  input?: unknown;
  output?: unknown;
  artifacts?: unknown;
  payload?: unknown;
  raw_input?: unknown;
  raw_output?: unknown;
  raw_artifacts?: unknown;
  [key: string]: unknown;
};

export type TimelineResult = {
  ok: boolean;
  resource: string;
  summary: string;
  artifacts?: {
    plan_path?: string;
  } | null;
  events?: TimelineEvent[] | null;
  errors?: ErrorDetail[] | null;
};

export type TimelineWorkspaceState = {
  selectedEventId: string | null;
  selectedTab: string;
};

export type ReviewFinding = {
  finding_id?: string;
  area: string;
  severity: string;
  title: string;
  details: string;
  locations?: string[] | null;
};

export type ReviewReviewer = {
  name?: string;
  status?: string;
  submitted_at?: string;
  aborted_at?: string;
  summary?: string;
  warnings?: string[] | null;
};

export type ReviewRound = {
  round_id: string;
  kind?: string;
  anchor_sha?: string;
  reviewed_head_sha?: string;
  repairs_round_id?: string;
  repair_finding_ids?: string[] | null;
  step?: number;
  revision?: number;
  review_title?: string;
  status?: string;
  status_summary?: string;
  decision?: string;
  created_at?: string;
  updated_at?: string;
  decided_at?: string;
  is_active?: boolean;
  reviewer?: ReviewReviewer | null;
  blocking_findings?: ReviewFinding[] | null;
  non_blocking_findings?: ReviewFinding[] | null;
  resolved_finding_ids?: string[] | null;
  unresolved_finding_ids?: string[] | null;
  coverage_status?: string;
  warnings?: string[] | null;
};

export type ReviewResult = {
  ok: boolean;
  resource: string;
  summary: string;
  rounds?: ReviewRound[] | null;
  warnings?: string[] | null;
  errors?: ErrorDetail[] | null;
};

export type ReviewWorkspaceState = {
  selectedRoundId: string | null;
};

export type DashboardProgressNode = {
  label: string;
  state: "pending" | "current" | "done";
};

export type DashboardProgress = {
  nodes?: DashboardProgressNode[] | null;
  step_completed?: number;
  step_total?: number;
  acceptance_completed?: number;
  acceptance_total?: number;
};

export type DashboardWorkspace = {
  workspace_key: string;
  workspace_name?: string;
  workspace_path: string;
  watched_at?: string;
  last_seen_at?: string;
  dashboard_state: "active" | "completed" | "idle" | "missing" | "invalid" | string;
  invalid_reason?: string;
  current_node?: string;
  plan_title?: string;
  facts?: StatusFacts | null;
  progress?: DashboardProgress | null;
  summary: string;
  next_actions?: NextAction[] | null;
  warnings?: string[] | null;
  blockers?: ErrorDetail[] | null;
  errors?: ErrorDetail[] | null;
  artifacts?: {
    plan_path?: string;
    supplements_path?: string;
    review_round_id?: string;
    review_submission_path?: string;
    last_landed_at?: string;
  } | null;
};

export type DashboardGroup = {
  state: string;
  workspaces?: DashboardWorkspace[] | null;
};

export type DashboardResult = {
  ok: boolean;
  resource: string;
  summary: string;
  groups?: DashboardGroup[] | null;
  errors?: ErrorDetail[] | null;
};

export type WorkspaceRouteResult = {
  ok: boolean;
  resource: string;
  summary: string;
  watched: boolean;
  workspace?: DashboardWorkspace | null;
  errors?: ErrorDetail[] | null;
};

declare global {
  interface Window {
    __HARNESS_UI__?: {
      workdir?: string;
      repoName?: string;
      productName?: string;
    };
  }
}
