package reviewcoverage

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var allowedCloseoutSections = map[string]bool{
	"Validation Summary": true,
	"Review Summary":     true,
	"Archive Summary":    true,
	"Outcome Summary":    true,
}

// ValidateArchiveWorktree permits only closeout-body edits in the current plan
// after the reviewed head. Every product, source, test, specification,
// supplement, plan-structure, and unrelated documentation change is rejected.
func ValidateArchiveWorktree(workdir, planPath, coveredHead string) error {
	currentHead, err := ResolveCommit(workdir, "HEAD")
	if err != nil {
		return fmt.Errorf("resolve current candidate head: %w", err)
	}
	ancestor, err := IsAncestor(workdir, coveredHead, currentHead)
	if err != nil {
		return fmt.Errorf("validate covered candidate ancestry: %w", err)
	}
	if !ancestor {
		return fmt.Errorf("current HEAD is not descended from reviewed head %s", coveredHead)
	}

	if !filepath.IsAbs(planPath) {
		planPath = filepath.Join(workdir, filepath.FromSlash(planPath))
	}
	planPath = filepath.Clean(planPath)
	relPlan, err := filepath.Rel(workdir, planPath)
	if err != nil || strings.HasPrefix(relPlan, ".."+string(filepath.Separator)) || relPlan == ".." {
		return fmt.Errorf("current plan path is outside the repository")
	}
	relPlan = filepath.ToSlash(relPlan)

	changed, err := changedPaths(workdir, coveredHead)
	if err != nil {
		return err
	}
	for _, path := range changed {
		if path != relPlan {
			return fmt.Errorf("unreviewed candidate change after %s: %s", coveredHead, path)
		}
	}
	if !contains(changed, relPlan) {
		return nil
	}

	modeSummary, err := gitOutput(workdir, "diff", "--summary", coveredHead, "--", relPlan)
	if err != nil {
		return fmt.Errorf("inspect plan file mode: %w", err)
	}
	if strings.TrimSpace(modeSummary) != "" {
		return fmt.Errorf("current plan path or file mode changed after review")
	}
	baseline, err := gitBytes(workdir, "show", coveredHead+":"+relPlan)
	if err != nil {
		return fmt.Errorf("read reviewed plan content: %w", err)
	}
	current, err := os.ReadFile(planPath)
	if err != nil {
		return fmt.Errorf("read current plan content: %w", err)
	}
	if !bytes.Equal(maskCloseoutBodies(baseline), maskCloseoutBodies(current)) {
		return fmt.Errorf("current plan changed outside the allowed closeout summary bodies")
	}
	return nil
}

func changedPaths(workdir, coveredHead string) ([]string, error) {
	data, err := gitBytes(workdir, "diff", "--name-status", "-z", "--no-renames", coveredHead, "--")
	if err != nil {
		return nil, fmt.Errorf("inspect reviewed candidate changes: %w", err)
	}
	parts := bytes.Split(data, []byte{0})
	paths := make([]string, 0)
	for i := 0; i+1 < len(parts); i += 2 {
		status := string(parts[i])
		path := filepath.ToSlash(string(parts[i+1]))
		if status != "M" {
			paths = append(paths, path)
			continue
		}
		paths = append(paths, path)
	}
	untracked, err := gitBytes(workdir, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, fmt.Errorf("inspect untracked candidate changes: %w", err)
	}
	for _, raw := range bytes.Split(untracked, []byte{0}) {
		path := filepath.ToSlash(string(raw))
		if path == "" {
			continue
		}
		paths = append(paths, path)
	}
	return uniqueStrings(paths), nil
}

func maskCloseoutBodies(content []byte) []byte {
	lines := strings.Split(string(content), "\n")
	out := make([]string, 0, len(lines))
	masking := false
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			name := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			masking = allowedCloseoutSections[name]
			out = append(out, line)
			if masking {
				out = append(out, "<EASYHARNESS_CLOSEOUT_BODY>")
			}
			continue
		}
		if !masking {
			out = append(out, line)
		}
	}
	return []byte(strings.Join(out, "\n"))
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}
