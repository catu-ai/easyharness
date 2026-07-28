package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/catu-ai/easyharness/internal/repoconfig"
)

const (
	WorkflowProfileStandard     = "standard"
	WorkflowProfileLightweight  = "lightweight"
	WorkflowProfileCoordinated  = "coordinated"
	WorkflowProfileGoalOriented = "goal_oriented"
	SupplementsDirName          = "supplements"
	SubplansDirName             = "subplans"
)

func normalizeWorkflowProfile(value string) string {
	switch strings.TrimSpace(value) {
	case "", WorkflowProfileStandard:
		return WorkflowProfileStandard
	case WorkflowProfileLightweight:
		return WorkflowProfileLightweight
	case WorkflowProfileCoordinated:
		return WorkflowProfileCoordinated
	case WorkflowProfileGoalOriented:
		return WorkflowProfileGoalOriented
	default:
		return strings.TrimSpace(value)
	}
}

func inferWorkflowProfileFromPath(path string) string {
	paths := pathsForPath(path)
	if pathUnderRoot(path, paths.archivedPlansRoot) {
		return WorkflowProfileStandard
	}
	if pathUnderRoot(path, paths.lightweightArchivedRoot) {
		return WorkflowProfileLightweight
	}
	if paths.hasValidConfig {
		return ""
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	switch {
	case strings.Contains(clean, "/docs/plans/archived/") || strings.HasPrefix(clean, "docs/plans/archived/"):
		return WorkflowProfileStandard
	case strings.Contains(clean, "/.local/harness/plans/archived/") || strings.HasPrefix(clean, ".local/harness/plans/archived/"):
		return WorkflowProfileLightweight
	default:
		return ""
	}
}

func inferPathKind(path string) string {
	paths := pathsForPath(path)
	if pathUnderRoot(path, paths.activePlansRoot) {
		return "active"
	}
	if pathUnderRoot(path, paths.archivedPlansRoot) || pathUnderRoot(path, paths.lightweightArchivedRoot) {
		return "archived"
	}
	if paths.hasValidConfig {
		return ""
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	switch {
	case strings.Contains(clean, "/docs/plans/active/") || strings.HasPrefix(clean, "docs/plans/active/"):
		return "active"
	case strings.Contains(clean, "/docs/plans/archived/") || strings.HasPrefix(clean, "docs/plans/archived/"):
		return "archived"
	case strings.Contains(clean, "/.local/harness/plans/archived/") || strings.HasPrefix(clean, ".local/harness/plans/archived/"):
		return "archived"
	}
	return ""
}

func PathKindFor(path string) string {
	return inferPathKind(path)
}

func activeCandidatePaths(workdir string) ([]string, error) {
	layout := pathsForWorkdir(workdir)
	paths, err := filepath.Glob(filepath.Join(layout.activePlansRoot, "*.md"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func currentLooksArchived(path string) bool {
	return inferPathKind(path) == "archived"
}

func ArchivedPathFor(workdir, planStem, currentPath string, profile string) string {
	layout := pathsForWorkdir(workdir)
	switch normalizeWorkflowProfile(profile) {
	case WorkflowProfileLightweight:
		return filepath.Join(layout.lightweightArchivedRoot, filepath.Base(currentPath))
	default:
		return filepath.Join(layout.archivedPlansRoot, filepath.Base(currentPath))
	}
}

func ActivePathFor(workdir, planStem, currentPath string, profile string) string {
	return filepath.Join(pathsForWorkdir(workdir).activePlansRoot, filepath.Base(currentPath))
}

func SupplementsDirForPlanPath(path string) string {
	clean := filepath.Clean(path)
	dir := filepath.Dir(clean)
	stem := strings.TrimSuffix(filepath.Base(clean), filepath.Ext(clean))
	return filepath.Join(dir, SupplementsDirName, stem)
}

func SubplansDirForPlanPath(path string) string {
	return filepath.Join(SupplementsDirForPlanPath(path), SubplansDirName)
}

func SubplanPathForPlan(path, id string) (string, error) {
	if err := validateSubplanID(strings.TrimSpace(id)); err != nil {
		return "", err
	}
	return filepath.Join(SubplansDirForPlanPath(path), strings.TrimSpace(id)+".md"), nil
}

func RootPathForSubplanPath(path string) (string, error) {
	clean := filepath.Clean(path)
	if filepath.Ext(clean) != ".md" || filepath.Base(filepath.Dir(clean)) != SubplansDirName {
		return "", fmt.Errorf("subplan must be a direct Markdown child of a subplans directory")
	}
	packageDir := filepath.Dir(filepath.Dir(clean))
	if filepath.Base(filepath.Dir(packageDir)) != SupplementsDirName {
		return "", fmt.Errorf("subplan must live under supplements/<root-stem>/subplans")
	}
	rootStem := filepath.Base(packageDir)
	rootPath := filepath.Join(filepath.Dir(filepath.Dir(packageDir)), rootStem+".md")
	if inferPathKind(rootPath) == "" {
		return "", fmt.Errorf("subplan root must live under a configured active or archived plan root")
	}
	if err := validatePlanFilename(filepath.Base(rootPath)); err != nil {
		return "", fmt.Errorf("invalid coordinated root path: %w", err)
	}
	return rootPath, nil
}

func ResolveSubplanPath(rootPath, ref string) (string, error) {
	trimmed := strings.TrimSpace(ref)
	if trimmed == "" {
		return "", fmt.Errorf("subplan reference must not be empty")
	}
	if err := validateSubplanID(trimmed); err == nil {
		return SubplanPathForPlan(rootPath, trimmed)
	}

	candidate := filepath.Clean(filepath.FromSlash(trimmed))
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(pathsForPath(rootPath).workdir, candidate)
	}
	root, err := RootPathForSubplanPath(candidate)
	if err != nil {
		return "", err
	}
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return "", err
	}
	absCandidateRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if filepath.Clean(absRoot) != filepath.Clean(absCandidateRoot) {
		return "", fmt.Errorf("subplan does not belong to coordinated root %s", filepath.ToSlash(rootPath))
	}
	return candidate, nil
}

func IsSubplanPath(path string) bool {
	_, err := RootPathForSubplanPath(path)
	return err == nil
}

func AlternateSupplementsDirsForPlanPath(path string) []string {
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	paths := pathsForPath(path)
	candidates := []string{
		filepath.Join(paths.activePlansRoot, SupplementsDirName, stem),
		filepath.Join(paths.archivedPlansRoot, SupplementsDirName, stem),
		filepath.Join(paths.lightweightArchivedRoot, SupplementsDirName, stem),
	}

	expected := filepath.Clean(SupplementsDirForPlanPath(path))
	if absolute, err := filepath.Abs(expected); err == nil {
		expected = filepath.Clean(absolute)
	}
	filtered := make([]string, 0, len(candidates)-1)
	for _, candidate := range candidates {
		if filepath.Clean(candidate) == expected {
			continue
		}
		filtered = append(filtered, candidate)
	}
	return filtered
}

func relativePathWithinPlanRoot(path string) string {
	paths := pathsForPath(path)
	for _, root := range []string{paths.activePlansRoot, paths.archivedPlansRoot, paths.lightweightArchivedRoot} {
		if rel, ok := relPathUnderRoot(path, root); ok {
			return rel
		}
	}
	if paths.hasValidConfig {
		return ""
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	for _, marker := range []string{"/docs/plans/active/", "/docs/plans/archived/", "/.local/harness/plans/archived/"} {
		if idx := strings.Index(clean, marker); idx >= 0 {
			return strings.TrimPrefix(clean[idx+len(marker):], "/")
		}
	}
	for _, marker := range []string{"docs/plans/active/", "docs/plans/archived/", ".local/harness/plans/archived/"} {
		if strings.HasPrefix(clean, marker) {
			return strings.TrimPrefix(clean, marker)
		}
	}
	return ""
}

func ArchivedSupplementsDirFor(workdir, planStem, currentPath string, profile string) string {
	return SupplementsDirForPlanPath(ArchivedPathFor(workdir, planStem, currentPath, profile))
}

func ActiveSupplementsDirFor(workdir, planStem, currentPath string, profile string) string {
	return SupplementsDirForPlanPath(ActivePathFor(workdir, planStem, currentPath, profile))
}

type configuredPlanPaths struct {
	workdir                 string
	activePlansRoot         string
	archivedPlansRoot       string
	localRuntimeRoot        string
	lightweightArchivedRoot string
	hasValidConfig          bool
}

func pathsForWorkdir(workdir string) configuredPlanPaths {
	load := repoconfig.Load(workdir)
	config := load.Config
	return configuredPlanPaths{
		workdir:                 workdir,
		activePlansRoot:         filepath.Join(workdir, filepath.FromSlash(config.Paths.Plans.Active)),
		archivedPlansRoot:       filepath.Join(workdir, filepath.FromSlash(config.Paths.Plans.Archived)),
		localRuntimeRoot:        filepath.Join(workdir, filepath.FromSlash(config.Paths.LocalRuntime)),
		lightweightArchivedRoot: filepath.Join(workdir, filepath.FromSlash(config.Paths.LocalRuntime), "plans", "archived"),
		hasValidConfig:          load.Exists && load.Valid,
	}
}

func pathsForPath(path string) configuredPlanPaths {
	if workdir := inferWorkdirForPath(path); workdir != "" {
		return pathsForWorkdir(workdir)
	}
	return pathsForWorkdir("")
}

func inferWorkdirForPath(path string) string {
	if !filepath.IsAbs(path) {
		if cwd, err := os.Getwd(); err == nil {
			return cwd
		}
		return ""
	}
	if workdir := findWorkdirFromPath(path); workdir != "" {
		return workdir
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	for _, marker := range []string{"/docs/plans/active/", "/docs/plans/archived/", "/.local/harness/plans/archived/"} {
		if idx := strings.Index(clean, marker); idx >= 0 {
			return filepath.FromSlash(clean[:idx])
		}
	}
	return ""
}

func findWorkdirFromPath(path string) string {
	dir := filepath.Dir(filepath.Clean(path))
	for {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(repoconfig.File))); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func pathUnderRoot(path, root string) bool {
	_, ok := relPathUnderRoot(path, root)
	return ok
}

func relPathUnderRoot(path, root string) (string, bool) {
	cleanPath := filepath.Clean(path)
	cleanRoot := filepath.Clean(root)
	if filepath.IsAbs(cleanRoot) && !filepath.IsAbs(cleanPath) {
		if cwd, err := os.Getwd(); err == nil {
			cleanPath = filepath.Join(cwd, cleanPath)
		}
	}
	if cleanPath == cleanRoot {
		return "", false
	}
	rel, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil {
		return "", false
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}
