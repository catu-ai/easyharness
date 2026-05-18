package remote

import (
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	PRStatusRecorded    = "recorded"
	PRStatusMissing     = "missing"
	PRStatusUnsupported = "unsupported"

	DegradedMissingPRURL      = "missing_pr_url"
	DegradedUnsupportedPRURL  = "unsupported_pr_url"
	DegradedNotGitRepository  = "not_git_repository"
	DegradedDetachedHead      = "detached_head"
	DegradedMissingRemote     = "missing_remote"
	DegradedUnsupportedRemote = "unsupported_remote"
	DegradedAmbiguousRemote   = "ambiguous_remote"
)

var scpLikeGitHubRemote = regexp.MustCompile(`^git@github\.com:([^/]+)/(.+?)(?:\.git)?$`)

type Service struct {
	Workdir string
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
		if match[1] == "" || repo == "" {
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
