package remote

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	PRStatusRecorded    = "recorded"
	PRStatusMissing     = "missing"
	PRStatusUnsupported = "unsupported"

	PRObservationAvailable   = "available"
	PRObservationUnavailable = "unavailable"

	HandoffObservationAvailable   = "available"
	HandoffObservationUnavailable = "unavailable"
	HandoffObservationDegraded    = "degraded"

	RemoteCIAvailable   = "available"
	RemoteCIUnavailable = "unavailable"

	RemoteSyncAvailable   = "available"
	RemoteSyncUnavailable = "unavailable"

	DegradedMissingPRURL      = "missing_pr_url"
	DegradedUnsupportedPRURL  = "unsupported_pr_url"
	DegradedNotGitRepository  = "not_git_repository"
	DegradedDetachedHead      = "detached_head"
	DegradedMissingRemote     = "missing_remote"
	DegradedUnsupportedRemote = "unsupported_remote"
	DegradedAmbiguousRemote   = "ambiguous_remote"
	DegradedGhMissing         = "gh_missing"
	DegradedGhAuthUnavailable = "gh_auth_unavailable"
	DegradedGhTimeout         = "gh_timeout"
	DegradedPRUnreadable      = "pr_unreadable"
	DegradedChecksUnreadable  = "checks_unreadable"
	DegradedMergeUnreadable   = "merge_unreadable"
	DegradedGhInvalidJSON     = "gh_invalid_json"
	DegradedGhCommandFailed   = "gh_command_failed"
)

var scpLikeGitHubRemote = regexp.MustCompile(`^git@github\.com:([^/]+)/(.+?)(?:\.git)?$`)
var defaultCommandTimeout = 30 * time.Second
var defaultCommandWaitDelay = 2 * time.Second

type Service struct {
	Workdir    string
	RunCommand CommandRunner
}

type CommandRunner func(name string, args ...string) CommandResult

type CommandResult struct {
	Stdout string
	Stderr string
	Err    error
}

type Snapshot struct {
	Local LocalContext `json:"local"`
	PR    PRIdentity   `json:"pr"`
}

type LocalContext struct {
	InGitRepo bool          `json:"in_git_repo"`
	Root      string        `json:"root,omitempty"`
	Branch    string        `json:"branch,omitempty"`
	Detached  bool          `json:"detached,omitempty"`
	Head      string        `json:"head,omitempty"`
	Remote    *Remote       `json:"remote,omitempty"`
	Degraded  []Degradation `json:"degraded,omitempty"`
}

type Remote struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Host      string `json:"host,omitempty"`
	Owner     string `json:"owner,omitempty"`
	Repo      string `json:"repo,omitempty"`
	Supported bool   `json:"supported"`
}

type PRIdentity struct {
	Status   string      `json:"status"`
	URL      string      `json:"url,omitempty"`
	Owner    string      `json:"owner,omitempty"`
	Repo     string      `json:"repo,omitempty"`
	Number   int         `json:"number,omitempty"`
	Degraded Degradation `json:"degraded,omitempty"`
}

type PRObservation struct {
	Status           string      `json:"status"`
	URL              string      `json:"url,omitempty"`
	Number           int         `json:"number,omitempty"`
	State            string      `json:"state,omitempty"`
	IsDraft          bool        `json:"is_draft,omitempty"`
	MergeStateStatus string      `json:"merge_state_status,omitempty"`
	Mergeable        string      `json:"mergeable,omitempty"`
	ReviewDecision   string      `json:"review_decision,omitempty"`
	HeadRefName      string      `json:"head_ref_name,omitempty"`
	HeadRefOID       string      `json:"head_ref_oid,omitempty"`
	BaseRefName      string      `json:"base_ref_name,omitempty"`
	BaseRefOID       string      `json:"base_ref_oid,omitempty"`
	Degraded         Degradation `json:"degraded,omitempty"`
}

type HandoffObservation struct {
	Status   string                `json:"status"`
	PR       PRObservation         `json:"pr"`
	CI       RemoteCIObservation   `json:"ci"`
	Sync     RemoteSyncObservation `json:"sync"`
	Degraded []Degradation         `json:"degraded,omitempty"`
}

type RemoteCIObservation struct {
	Status         string      `json:"status"`
	EvidenceStatus string      `json:"evidence_status,omitempty"`
	Checks         []CheckRun  `json:"checks,omitempty"`
	Degraded       Degradation `json:"degraded,omitempty"`
}

type RemoteSyncObservation struct {
	Status         string      `json:"status"`
	EvidenceStatus string      `json:"evidence_status,omitempty"`
	MergeState     string      `json:"merge_state,omitempty"`
	Freshness      string      `json:"freshness,omitempty"`
	Conflict       string      `json:"conflict,omitempty"`
	MergePolicy    string      `json:"merge_policy,omitempty"`
	Degraded       Degradation `json:"degraded,omitempty"`
}

type CheckRun struct {
	Name     string `json:"name,omitempty"`
	Workflow string `json:"workflow,omitempty"`
	Bucket   string `json:"bucket,omitempty"`
	State    string `json:"state,omitempty"`
	Link     string `json:"link,omitempty"`
}

type Degradation struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
}

func (s Service) Snapshot(recordedPRURL string) Snapshot {
	return Snapshot{
		Local: InspectLocal(s.Workdir),
		PR:    ParseRecordedPRURL(recordedPRURL),
	}
}

func (s Service) ObserveRecordedPR(identity PRIdentity) PRObservation {
	if identity.Status != PRStatusRecorded {
		return PRObservation{
			Status:   PRObservationUnavailable,
			Degraded: identity.Degraded,
		}
	}

	result := s.run("gh", "pr", "view", identity.URL, "--json", "url,number,state,isDraft,mergeStateStatus,mergeable,reviewDecision,headRefName,headRefOid,baseRefName,baseRefOid")
	if result.Err != nil {
		return PRObservation{
			Status:   PRObservationUnavailable,
			Degraded: classifyGhFailure(result),
		}
	}

	var parsed struct {
		URL              string `json:"url"`
		Number           int    `json:"number"`
		State            string `json:"state"`
		IsDraft          bool   `json:"isDraft"`
		MergeStateStatus string `json:"mergeStateStatus"`
		Mergeable        string `json:"mergeable"`
		ReviewDecision   string `json:"reviewDecision"`
		HeadRefName      string `json:"headRefName"`
		HeadRefOID       string `json:"headRefOid"`
		BaseRefName      string `json:"baseRefName"`
		BaseRefOID       string `json:"baseRefOid"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &parsed); err != nil {
		return PRObservation{
			Status: PRObservationUnavailable,
			Degraded: Degradation{
				Code:    DegradedGhInvalidJSON,
				Message: "gh returned invalid JSON for recorded PR observation",
			},
		}
	}

	return PRObservation{
		Status:           PRObservationAvailable,
		URL:              parsed.URL,
		Number:           parsed.Number,
		State:            parsed.State,
		IsDraft:          parsed.IsDraft,
		MergeStateStatus: parsed.MergeStateStatus,
		Mergeable:        parsed.Mergeable,
		ReviewDecision:   parsed.ReviewDecision,
		HeadRefName:      parsed.HeadRefName,
		HeadRefOID:       parsed.HeadRefOID,
		BaseRefName:      parsed.BaseRefName,
		BaseRefOID:       parsed.BaseRefOID,
	}
}

func (s Service) ObserveHandoff(identity PRIdentity) HandoffObservation {
	pr := s.ObserveRecordedPR(identity)
	if pr.Status != PRObservationAvailable {
		degradation := pr.Degraded
		return HandoffObservation{
			Status:   HandoffObservationUnavailable,
			PR:       pr,
			CI:       unavailableCI(degradation),
			Sync:     unavailableSync(degradation),
			Degraded: degradationList(degradation),
		}
	}

	ci := s.observeChecks(identity.URL)
	sync := s.observeSync(identity, pr)
	degraded := make([]Degradation, 0, 2)
	if ci.Status == RemoteCIUnavailable {
		degraded = append(degraded, ci.Degraded)
	}
	if sync.Status == RemoteSyncUnavailable {
		degraded = append(degraded, sync.Degraded)
	}

	status := HandoffObservationAvailable
	if len(degraded) > 0 {
		status = HandoffObservationDegraded
	}
	return HandoffObservation{
		Status:   status,
		PR:       pr,
		CI:       ci,
		Sync:     sync,
		Degraded: degraded,
	}
}

func (s Service) observeChecks(prURL string) RemoteCIObservation {
	result := s.run("gh", "pr", "checks", prURL, "--json", "name,workflow,bucket,state,link")
	if errors.Is(result.Err, context.DeadlineExceeded) || errors.Is(result.Err, exec.ErrWaitDelay) {
		return unavailableCI(classifyGhFailure(result))
	}
	if result.Err != nil && strings.TrimSpace(result.Stdout) == "" {
		return unavailableCI(classifyGhFailure(result))
	}

	var checks []CheckRun
	if err := json.Unmarshal([]byte(result.Stdout), &checks); err != nil {
		return unavailableCI(Degradation{
			Code:    DegradedGhInvalidJSON,
			Message: "gh returned invalid JSON for recorded PR checks",
		})
	}
	if len(checks) == 0 {
		return unavailableCI(Degradation{
			Code:    DegradedChecksUnreadable,
			Message: "recorded PR checks are unavailable",
		})
	}

	status, ok := classifyChecks(checks)
	if !ok {
		return unavailableCI(Degradation{
			Code:    DegradedChecksUnreadable,
			Message: "recorded PR checks could not be classified",
		})
	}
	return RemoteCIObservation{
		Status:         RemoteCIAvailable,
		EvidenceStatus: status,
		Checks:         checks,
	}
}

func classifyChecks(checks []CheckRun) (string, bool) {
	hasPending := false
	hasPassing := false
	for _, check := range checks {
		switch strings.ToLower(strings.TrimSpace(check.Bucket)) {
		case "fail", "cancel":
			return "failed", true
		case "pending":
			hasPending = true
		case "pass":
			hasPassing = true
		case "skipping":
		default:
			return "", false
		}
	}
	if hasPending {
		return "pending", true
	}
	if hasPassing {
		return "success", true
	}
	return "", false
}

func (s Service) observeSync(identity PRIdentity, pr PRObservation) RemoteSyncObservation {
	mergeState := strings.ToUpper(strings.TrimSpace(pr.MergeStateStatus))
	conflict := classifyConflict(pr.Mergeable, mergeState)
	policy := classifyMergePolicy(mergeState)
	baseOID := strings.TrimSpace(pr.BaseRefOID)
	headOID := strings.TrimSpace(pr.HeadRefOID)
	if baseOID == "" || headOID == "" {
		return unavailableSyncWithFacts(mergeState, conflict, policy, Degradation{
			Code:    DegradedMergeUnreadable,
			Message: "recorded PR base or head commit is unavailable",
		})
	}

	endpoint := fmt.Sprintf("repos/%s/%s/compare/%s...%s", identity.Owner, identity.Repo, baseOID, headOID)
	result := s.run("gh", "api", endpoint)
	if result.Err != nil {
		return unavailableSyncWithFacts(mergeState, conflict, policy, classifyGhFailure(result))
	}
	var comparison struct {
		BehindBy int `json:"behind_by"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &comparison); err != nil || comparison.BehindBy < 0 {
		return unavailableSyncWithFacts(mergeState, conflict, policy, Degradation{
			Code:    DegradedGhInvalidJSON,
			Message: "gh returned invalid JSON for recorded PR base comparison",
		})
	}

	freshness := "fresh"
	if comparison.BehindBy > 0 {
		freshness = "stale"
	}
	evidenceStatus := freshness
	if conflict == "conflicted" {
		evidenceStatus = "conflicted"
	}
	return RemoteSyncObservation{
		Status:         RemoteSyncAvailable,
		EvidenceStatus: evidenceStatus,
		MergeState:     mergeState,
		Freshness:      freshness,
		Conflict:       conflict,
		MergePolicy:    policy,
	}
}

func classifyConflict(mergeable, mergeState string) string {
	switch strings.ToUpper(strings.TrimSpace(mergeable)) {
	case "MERGEABLE":
		return "clear"
	case "CONFLICTING":
		return "conflicted"
	}
	if mergeState == "DIRTY" {
		return "conflicted"
	}
	return "unknown"
}

func classifyMergePolicy(mergeState string) string {
	switch mergeState {
	case "BLOCKED", "DRAFT", "HAS_HOOKS":
		return "blocked"
	case "CLEAN", "BEHIND", "DIRTY", "UNSTABLE":
		return "clear"
	default:
		return "unknown"
	}
}

func unavailableCI(degradation Degradation) RemoteCIObservation {
	return RemoteCIObservation{
		Status:   RemoteCIUnavailable,
		Degraded: degradation,
	}
}

func unavailableSync(degradation Degradation) RemoteSyncObservation {
	return RemoteSyncObservation{
		Status:   RemoteSyncUnavailable,
		Degraded: degradation,
	}
}

func unavailableSyncWithFacts(mergeState, conflict, policy string, degradation Degradation) RemoteSyncObservation {
	return RemoteSyncObservation{
		Status:      RemoteSyncUnavailable,
		MergeState:  mergeState,
		Conflict:    conflict,
		MergePolicy: policy,
		Degraded:    degradation,
	}
}

func degradationList(degradation Degradation) []Degradation {
	if degradation.Code == "" {
		return nil
	}
	return []Degradation{degradation}
}

func ParseRecordedPRURL(raw string) PRIdentity {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return PRIdentity{
			Status: PRStatusMissing,
			Degraded: Degradation{
				Code:    DegradedMissingPRURL,
				Message: "publish evidence has no recorded PR URL",
			},
		}
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme != "https" || parsed.Host != "github.com" {
		return unsupportedPRURL(trimmed)
	}
	parts := splitPath(parsed.Path)
	if len(parts) != 4 || parts[2] != "pull" {
		return unsupportedPRURL(trimmed)
	}
	if parts[0] == "" || parts[1] == "" {
		return unsupportedPRURL(trimmed)
	}
	number, err := strconv.Atoi(parts[3])
	if err != nil || number <= 0 {
		return unsupportedPRURL(trimmed)
	}

	return PRIdentity{
		Status: PRStatusRecorded,
		URL:    trimmed,
		Owner:  parts[0],
		Repo:   parts[1],
		Number: number,
	}
}

func InspectLocal(workdir string) LocalContext {
	root, err := gitOutput(workdir, "rev-parse", "--show-toplevel")
	if err != nil {
		return LocalContext{
			Degraded: []Degradation{{
				Code:    DegradedNotGitRepository,
				Message: "workdir is not inside a git repository",
			}},
		}
	}

	local := LocalContext{
		InGitRepo: true,
		Root:      root,
	}
	if head, err := gitOutput(root, "rev-parse", "--verify", "HEAD"); err == nil {
		local.Head = head
	}
	if branch, err := gitOutput(root, "symbolic-ref", "--quiet", "--short", "HEAD"); err == nil {
		local.Branch = branch
	} else {
		local.Detached = true
		local.Degraded = append(local.Degraded, Degradation{
			Code:    DegradedDetachedHead,
			Message: "worktree HEAD is detached",
		})
	}

	remote, degradation := inspectRemote(root, local.Branch)
	if remote != nil {
		local.Remote = remote
	}
	if degradation.Code != "" {
		local.Degraded = append(local.Degraded, degradation)
	}

	return local
}

func inspectRemote(root, branch string) (*Remote, Degradation) {
	remoteName := ""
	if branch != "" {
		if upstream, err := gitOutput(root, "config", "--get", "branch."+branch+".remote"); err == nil && upstream != "." {
			remoteName = upstream
		}
	}
	if remoteName == "" {
		if _, err := gitOutput(root, "remote", "get-url", "origin"); err == nil {
			remoteName = "origin"
		}
	}
	if remoteName == "" {
		remotes, err := gitOutput(root, "remote")
		if err != nil || strings.TrimSpace(remotes) == "" {
			return nil, Degradation{
				Code:    DegradedMissingRemote,
				Message: "no git remote is configured",
			}
		}
		names := splitLines(remotes)
		if len(names) == 1 {
			remoteName = names[0]
		} else {
			return nil, Degradation{
				Code:    DegradedAmbiguousRemote,
				Message: "multiple git remotes are configured and no unambiguous remote was selected",
			}
		}
	}

	rawURL, err := gitOutput(root, "remote", "get-url", remoteName)
	if err != nil {
		return nil, Degradation{
			Code:    DegradedMissingRemote,
			Message: fmt.Sprintf("unable to read git remote %q", remoteName),
		}
	}
	remote := parseGitHubRemote(remoteName, rawURL)
	if !remote.Supported {
		return &remote, Degradation{
			Code:    DegradedUnsupportedRemote,
			Message: fmt.Sprintf("git remote %q is not a supported GitHub remote", remoteName),
		}
	}
	return &remote, Degradation{}
}

func parseGitHubRemote(name, raw string) Remote {
	trimmed := strings.TrimSpace(raw)
	remote := Remote{Name: name, URL: trimmed}

	if match := scpLikeGitHubRemote.FindStringSubmatch(trimmed); len(match) == 3 {
		repo := strings.TrimSuffix(match[2], ".git")
		if match[1] == "" || repo == "" || strings.Contains(repo, "/") {
			return remote
		}
		remote.Host = "github.com"
		remote.Owner = match[1]
		remote.Repo = repo
		remote.Supported = true
		return remote
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host != "github.com" {
		return remote
	}
	parts := splitPath(parsed.Path)
	if len(parts) != 2 {
		return remote
	}
	repo := strings.TrimSuffix(parts[1], ".git")
	if parts[0] == "" || repo == "" {
		return remote
	}
	remote.Host = "github.com"
	remote.Owner = parts[0]
	remote.Repo = repo
	remote.Supported = true
	return remote
}

func unsupportedPRURL(raw string) PRIdentity {
	return PRIdentity{
		Status: PRStatusUnsupported,
		URL:    raw,
		Degraded: Degradation{
			Code:    DegradedUnsupportedPRURL,
			Message: "recorded PR URL is not a supported GitHub pull request URL",
		},
	}
}

func gitOutput(workdir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", filepath.Clean(workdir)}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (s Service) run(name string, args ...string) CommandResult {
	if s.RunCommand != nil {
		return s.RunCommand(name, args...)
	}
	ctx := context.Background()
	var cancel context.CancelFunc
	if defaultCommandTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, defaultCommandTimeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = defaultCommandWaitDelay
	output, err := cmd.Output()
	stderr := ""
	if exitErr := new(exec.ExitError); errors.As(err, &exitErr) {
		stderr = string(exitErr.Stderr)
	}
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	return CommandResult{
		Stdout: string(output),
		Stderr: stderr,
		Err:    err,
	}
}

func classifyGhFailure(result CommandResult) Degradation {
	if errors.Is(result.Err, context.DeadlineExceeded) || errors.Is(result.Err, exec.ErrWaitDelay) {
		return Degradation{
			Code:    DegradedGhTimeout,
			Message: "gh timed out while observing the recorded PR",
		}
	}
	text := strings.ToLower(result.Stderr + "\n" + result.Err.Error())
	switch {
	case errors.Is(result.Err, exec.ErrNotFound):
		return Degradation{
			Code:    DegradedGhMissing,
			Message: "gh is not available",
		}
	case strings.Contains(text, "auth") || strings.Contains(text, "authentication") || strings.Contains(text, "401"):
		return Degradation{
			Code:    DegradedGhAuthUnavailable,
			Message: "gh authentication is unavailable",
		}
	case strings.Contains(text, "could not resolve to a pullrequest") || strings.Contains(text, "not found"):
		return Degradation{
			Code:    DegradedPRUnreadable,
			Message: "recorded PR could not be read through gh",
		}
	default:
		return Degradation{
			Code:    DegradedGhCommandFailed,
			Message: "gh failed while observing the recorded PR",
		}
	}
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func splitLines(input string) []string {
	lines := strings.Split(input, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
