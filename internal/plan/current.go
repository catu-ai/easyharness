package plan

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yzhang1918/superharness/internal/runstate"
)

func DetectCurrentPath(workdir string) (string, error) {
	if current, err := runstate.LoadCurrentPlan(workdir); err != nil {
		return "", err
	} else if current != nil && strings.TrimSpace(current.PlanPath) != "" {
		return filepath.Join(workdir, current.PlanPath), nil
	}

	activeMatches, err := filepath.Glob(filepath.Join(workdir, "docs", "plans", "active", "*.md"))
	if err != nil {
		return "", err
	}
	sort.Strings(activeMatches)
	if len(activeMatches) == 1 {
		return activeMatches[0], nil
	}
	if len(activeMatches) > 1 {
		return "", fmt.Errorf("multiple active plans found; add .local/harness/current-plan.json to disambiguate")
	}

	archivedMatches, err := filepath.Glob(filepath.Join(workdir, "docs", "plans", "archived", "*.md"))
	if err != nil {
		return "", err
	}
	sort.Strings(archivedMatches)
	if len(archivedMatches) == 1 {
		return archivedMatches[0], nil
	}

	return "", fmt.Errorf("no current plan found")
}
