package reviewdimensions

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/repoconfig"
	"github.com/catu-ai/easyharness/internal/reviewguidance"
)

const (
	SourceBuiltin = "builtin"
	SourceRepo    = "repo"
	SourcePlan    = "plan"
)

type Dimension struct {
	Name         string
	Sources      []string
	Description  string
	Instructions string
	Path         string
	PlanPath     string
}

type Catalog struct {
	Dimensions []Dimension
	Warnings   []string
	Errors     []contracts.ErrorDetail
}

type Service struct {
	Workdir string
}

func (s Service) List() contracts.ReviewDimensionsListResult {
	catalog := s.loadCurrent()
	if len(catalog.Errors) > 0 {
		return contracts.ReviewDimensionsListResult{
			OK:         false,
			Command:    "review dimensions list",
			Summary:    "Review dimension catalog is invalid.",
			Dimensions: []contracts.ReviewDimensionMetadata{},
			Warnings:   catalog.Warnings,
			Errors:     catalog.Errors,
		}
	}
	return contracts.ReviewDimensionsListResult{
		OK:       true,
		Command:  "review dimensions list",
		Summary:  fmt.Sprintf("Found %d review dimensions.", len(catalog.Dimensions)),
		Warnings: catalog.Warnings,
		Dimensions: func() []contracts.ReviewDimensionMetadata {
			items := make([]contracts.ReviewDimensionMetadata, 0, len(catalog.Dimensions))
			for _, dimension := range catalog.Dimensions {
				items = append(items, contracts.ReviewDimensionMetadata{
					Name:        dimension.Name,
					Sources:     append([]string(nil), dimension.Sources...),
					Path:        dimension.Path,
					PlanPath:    dimension.PlanPath,
					Description: dimension.Description,
				})
			}
			return items
		}(),
	}
}

func (s Service) Instructions(name string) (string, []string, []contracts.ErrorDetail) {
	name = strings.TrimSpace(name)
	if !ValidName(name) {
		return "", nil, []contracts.ErrorDetail{{Path: "name", Message: "must use lowercase alphanumeric segments separated by single hyphens"}}
	}
	catalog := s.loadCurrent()
	if len(catalog.Errors) > 0 {
		return "", catalog.Warnings, catalog.Errors
	}
	for _, dimension := range catalog.Dimensions {
		if dimension.Name == name {
			return dimension.Instructions, catalog.Warnings, nil
		}
	}
	return "", catalog.Warnings, []contracts.ErrorDetail{{Path: "name", Message: fmt.Sprintf("unknown review dimension %q", name)}}
}

func ValidName(name string) bool {
	return reviewguidance.ValidName(name)
}

// ResolveForPlan resolves one guidance dimension against an exact plan path.
// Review start uses this API so plan selection cannot change between catalog
// discovery and round creation.
func (s Service) ResolveForPlan(planPath, name string) (Dimension, []string, []contracts.ErrorDetail) {
	name = strings.TrimSpace(name)
	if !ValidName(name) {
		return Dimension{}, nil, []contracts.ErrorDetail{{Path: "name", Message: "must use lowercase alphanumeric segments separated by single hyphens"}}
	}
	catalog := s.CatalogForPlan(planPath)
	if len(catalog.Errors) > 0 {
		return Dimension{}, catalog.Warnings, catalog.Errors
	}
	for _, dimension := range catalog.Dimensions {
		if dimension.Name == name {
			return dimension, catalog.Warnings, nil
		}
	}
	return Dimension{}, catalog.Warnings, []contracts.ErrorDetail{{Path: "name", Message: fmt.Sprintf("unknown review dimension %q", name)}}
}

// CatalogForPlan returns the resolved catalog for one exact active, archived,
// or reopened plan. An empty plan path returns the repository catalog alone.
func (s Service) CatalogForPlan(planPath string) Catalog {
	result := repoconfig.Load(s.Workdir)
	catalog := Catalog{
		Dimensions: builtinDimensions(),
		Warnings:   result.Warnings,
	}
	root := filepath.Join(s.Workdir, filepath.FromSlash(result.Config.Paths.Review.Dimensions))
	repoDimensions, errors := loadRepoDimensions(root, result.Config.Paths.Review.Dimensions)
	if len(errors) > 0 {
		catalog.Errors = errors
		return catalog
	}
	catalog.Dimensions = mergeDimensions(catalog.Dimensions, repoDimensions)
	if strings.TrimSpace(planPath) == "" {
		return catalog
	}
	absPlanPath, relPlanPath, err := s.resolvePlanPath(planPath)
	if err != nil {
		catalog.Errors = []contracts.ErrorDetail{{Path: "plan_path", Message: err.Error()}}
		return catalog
	}
	planDimensions, errors := loadPlanDimensions(plan.ReviewGuidanceDirForPlanPath(absPlanPath), relPlanPath, s.Workdir)
	if len(errors) > 0 {
		catalog.Errors = errors
		return catalog
	}
	catalog.Dimensions = appendPlanDimensions(catalog.Dimensions, planDimensions)
	return catalog
}

func (s Service) loadCurrent() Catalog {
	currentPath, err := plan.DetectCurrentPath(s.Workdir)
	if errors.Is(err, plan.ErrNoCurrentPlan) {
		return s.CatalogForPlan("")
	}
	if err != nil {
		catalog := s.CatalogForPlan("")
		catalog.Errors = append(catalog.Errors, contracts.ErrorDetail{Path: "current_plan", Message: err.Error()})
		return catalog
	}
	return s.CatalogForPlan(currentPath)
}

func (s Service) resolvePlanPath(planPath string) (string, string, error) {
	workdir, err := filepath.Abs(s.Workdir)
	if err != nil {
		return "", "", fmt.Errorf("unable to resolve workdir: %v", err)
	}
	absPath := filepath.Clean(planPath)
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(workdir, filepath.FromSlash(planPath))
	}
	absPath, err = filepath.Abs(absPath)
	if err != nil {
		return "", "", fmt.Errorf("unable to resolve plan path: %v", err)
	}
	relPath, err := filepath.Rel(workdir, absPath)
	if err != nil || relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("must resolve inside the repository workdir")
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", "", fmt.Errorf("unable to read exact plan: %v", err)
	}
	if info.IsDir() {
		return "", "", fmt.Errorf("exact plan path must be a Markdown file")
	}
	return absPath, filepath.ToSlash(relPath), nil
}

func loadRepoDimensions(root, repoRoot string) ([]Dimension, []contracts.ErrorDetail) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []contracts.ErrorDetail{{Path: repoRoot, Message: fmt.Sprintf("unable to read review dimensions root: %v", err)}}
	}
	var dimensions []Dimension
	var errors []contracts.ErrorDetail
	seen := map[string]string{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		absPath := filepath.Join(root, entry.Name())
		repoPath := filepath.ToSlash(filepath.Join(repoRoot, entry.Name()))
		definition, err := reviewguidance.ParseFile(absPath)
		if err != nil {
			errors = append(errors, contracts.ErrorDetail{Path: repoPath, Message: err.Error()})
			continue
		}
		if previous, ok := seen[definition.Name]; ok {
			errors = append(errors, contracts.ErrorDetail{
				Path:    repoPath,
				Message: fmt.Sprintf("duplicates review dimension name %q already defined in %s", definition.Name, previous),
			})
			continue
		}
		seen[definition.Name] = repoPath
		dimensions = append(dimensions, Dimension{
			Name: definition.Name, Sources: []string{SourceRepo}, Description: definition.Description,
			Instructions: definition.Instructions, Path: repoPath,
		})
	}
	sort.Slice(dimensions, func(i, j int) bool {
		return dimensions[i].Name < dimensions[j].Name
	})
	return dimensions, errors
}

func loadPlanDimensions(root, planPath, workdir string) ([]Dimension, []contracts.ErrorDetail) {
	relRoot, err := filepath.Rel(workdir, root)
	if err != nil {
		relRoot = root
	}
	relRoot = filepath.ToSlash(relRoot)
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []contracts.ErrorDetail{{Path: relRoot, Message: fmt.Sprintf("unable to read plan review guidance root: %v", err)}}
	}
	var dimensions []Dimension
	var resultErrors []contracts.ErrorDetail
	seen := map[string]string{}
	for _, entry := range entries {
		repoPath := filepath.ToSlash(filepath.Join(relRoot, entry.Name()))
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			resultErrors = append(resultErrors, contracts.ErrorDetail{Path: repoPath, Message: "plan review guidance root accepts only Markdown files directly under it"})
			continue
		}
		definition, err := reviewguidance.ParseFile(filepath.Join(root, entry.Name()))
		if err != nil {
			resultErrors = append(resultErrors, contracts.ErrorDetail{Path: repoPath, Message: err.Error()})
			continue
		}
		if previous, ok := seen[definition.Name]; ok {
			resultErrors = append(resultErrors, contracts.ErrorDetail{Path: repoPath, Message: fmt.Sprintf("duplicates plan review guidance name %q already defined in %s", definition.Name, previous)})
			continue
		}
		seen[definition.Name] = repoPath
		dimensions = append(dimensions, Dimension{
			Name: definition.Name, Sources: []string{SourcePlan}, Description: definition.Description,
			Instructions: definition.Instructions, Path: repoPath, PlanPath: planPath,
		})
	}
	sort.Slice(dimensions, func(i, j int) bool { return dimensions[i].Name < dimensions[j].Name })
	return dimensions, resultErrors
}

func appendPlanDimensions(base, planDimensions []Dimension) []Dimension {
	byName := make(map[string]Dimension, len(base)+len(planDimensions))
	for _, dimension := range base {
		byName[dimension.Name] = dimension
	}
	for _, fragment := range planDimensions {
		if existing, ok := byName[fragment.Name]; ok {
			fragment.Description = strings.TrimSpace(existing.Description + " Plan guidance: " + fragment.Description)
			fragment.Instructions = strings.TrimSpace(existing.Instructions + "\n\n## Plan-scoped guidance\n\n" + fragment.Instructions)
			fragment.Sources = append(append([]string(nil), existing.Sources...), SourcePlan)
		}
		byName[fragment.Name] = fragment
	}
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	merged := make([]Dimension, 0, len(names))
	for _, name := range names {
		merged = append(merged, byName[name])
	}
	return merged
}

func mergeDimensions(builtin, repo []Dimension) []Dimension {
	byName := map[string]Dimension{}
	for _, dimension := range builtin {
		byName[dimension.Name] = dimension
	}
	for _, dimension := range repo {
		byName[dimension.Name] = dimension
	}
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	merged := make([]Dimension, 0, len(names))
	for _, name := range names {
		merged = append(merged, byName[name])
	}
	return merged
}

func builtinDimensions() []Dimension {
	return []Dimension{
		{
			Name:        "agent-ux",
			Sources:     []string{SourceBuiltin},
			Description: "Use when reviewing whether another agent can understand, resume, and safely act on the workflow state and command output.",
			Instructions: strings.TrimSpace(`
Review the change from the perspective of the next controller or reviewer agent.

Check whether command output, plan notes, review prompts, and handoff guidance are concise, accurate, and actionable. Look for hidden assumptions, missing next actions, ambiguous slot ownership, or wording that would make a future agent guess instead of act from repository evidence.
`),
		},
		{
			Name:        "correctness",
			Sources:     []string{SourceBuiltin},
			Description: "Use when reviewing implementation logic, workflow state transitions, command contracts, or negative-path behavior.",
			Instructions: strings.TrimSpace(`
Review the change for correctness.

Check implementation logic, command contracts, state transitions, validation behavior, and error paths. Look for stale assumptions, missing negative-path handling, mismatches between persisted artifacts and command output, and behavior that would break normal harness workflow progression.
`),
		},
		{
			Name:        "docs-consistency",
			Sources:     []string{SourceBuiltin},
			Description: "Use when changes touch README, AGENTS, specs, skills, schemas, examples, or other durable workflow guidance.",
			Instructions: strings.TrimSpace(`
Review the change for documentation and contract consistency.

Check whether README, AGENTS.md, managed skills, specs, schemas, examples, and plan text tell the same story as the implementation. Look for stale command names, mismatched field semantics, missing bootstrap asset updates, and guidance that would drift from the actual CLI behavior.
`),
		},
		{
			Name:        "evidence-validity",
			Sources:     []string{SourceBuiltin},
			Description: "Use when reviewing whether conclusions, syntheses, or decisions are supported by the scorecard, probes, evidence, residuals, and follow-up handling.",
			Instructions: strings.TrimSpace(`
Review the change for evidence validity.

Check whether accepted conclusions, syntheses, and decision artifacts are supported by the approved scorecard, tracked checkpoint reports, probes or experiments, durable evidence, and validation results. Look for weak evidence claims, missing comparisons, untested assumptions, rejected hypotheses that were not actually ruled out, residual uncertainty that is hidden or understated, and follow-up handling that leaves the decision less supported than the plan claims.
`),
		},
		{
			Name:        "risk-scan",
			Sources:     []string{SourceBuiltin},
			Description: "Use when reviewing unresolved blockers, leaked deferred scope, unsafe defaults, or release-sensitive workflow risks.",
			Instructions: strings.TrimSpace(`
Review the change for risk and unresolved scope.

Look for blockers that should stop closeout, deferred items that accidentally leaked into the current slice, unsafe defaults, brittle assumptions, migration or compatibility surprises, and release-sensitive behavior that needs stronger validation before the branch waits for merge approval.
`),
		},
		{
			Name:        "tests",
			Sources:     []string{SourceBuiltin},
			Description: "Use when reviewing whether coverage, fixtures, smoke tests, or validation claims prove the changed behavior.",
			Instructions: strings.TrimSpace(`
Review the change's test and validation coverage.

Check whether unit, integration, smoke, and manual validation evidence matches the behavioral risk. Look for missing assertions, weak fixtures, overbroad smoke claims, untested negative paths, and generated artifacts or bootstrap sync steps that should have been validated.
`),
		},
	}
}
