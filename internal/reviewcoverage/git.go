package reviewcoverage

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Candidate is the immutable Git boundary used by one formal review round.
type Candidate struct {
	HeadSHA string
}

// CaptureCandidate requires a Git-backed, committed, clean candidate. Git's
// ordinary ignore rules apply, so the default ignored runtime root stays out of
// the candidate while a custom unignored runtime path correctly makes it dirty.
func CaptureCandidate(workdir string) (Candidate, error) {
	head, err := gitOutput(workdir, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return Candidate{}, fmt.Errorf("review requires a Git repository with a committed HEAD: %w", err)
	}
	head = strings.TrimSpace(head)
	if head == "" {
		return Candidate{}, fmt.Errorf("review requires a Git repository with a committed HEAD")
	}

	status, err := gitBytes(workdir, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return Candidate{}, fmt.Errorf("inspect candidate worktree: %w", err)
	}
	dirty := dirtyCandidateEntries(workdir, status)
	if len(dirty) > 0 {
		return Candidate{}, fmt.Errorf("review requires a clean candidate worktree; commit or remove: %s", strings.Join(dirty, ", "))
	}
	return Candidate{HeadSHA: head}, nil
}

// ResolveCommit returns the canonical commit object for a revision expression.
func ResolveCommit(workdir, revision string) (string, error) {
	resolved, err := gitOutput(workdir, "rev-parse", "--verify", strings.TrimSpace(revision)+"^{commit}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resolved), nil
}

func IsAncestor(workdir, ancestor, descendant string) (bool, error) {
	cmd := exec.Command("git", "-C", workdir, "merge-base", "--is-ancestor", ancestor, descendant)
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

func gitOutput(workdir string, args ...string) (string, error) {
	data, err := gitBytes(workdir, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func gitBytes(workdir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", workdir}, args...)...)
	data, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(data))
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), message)
	}
	return data, nil
}

func dirtyCandidateEntries(workdir string, status []byte) []string {
	entries := bytes.Split(status, []byte{0})
	dirty := make([]string, 0, len(entries))
	for _, entry := range entries {
		if len(entry) < 4 {
			continue
		}
		code := string(entry[:2])
		path := filepath.ToSlash(strings.TrimSpace(string(entry[3:])))
		dirty = append(dirty, code+" "+path)
	}
	return dirty
}
