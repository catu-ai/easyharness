package reviewdimensions

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/repoconfig"
	"gopkg.in/yaml.v3"
)

const (
	SourceBuiltin = "builtin"
	SourceRepo    = "repo"
)

var namePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Dimension struct {
	Name         string
	Source       string
	Description  string
	Instructions string
	Path         string
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
	catalog := s.load()
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
					Source:      dimension.Source,
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
	catalog := s.load()
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
	return namePattern.MatchString(strings.TrimSpace(name))
}

func (s Service) load() Catalog {
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
	return catalog
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
		dimension, err := parseDimensionFile(absPath, repoPath)
		if err != nil {
			errors = append(errors, contracts.ErrorDetail{Path: repoPath, Message: err.Error()})
			continue
		}
		if previous, ok := seen[dimension.Name]; ok {
			errors = append(errors, contracts.ErrorDetail{
				Path:    repoPath,
				Message: fmt.Sprintf("duplicates review dimension name %q already defined in %s", dimension.Name, previous),
			})
			continue
		}
		seen[dimension.Name] = repoPath
		dimensions = append(dimensions, dimension)
	}
	sort.Slice(dimensions, func(i, j int) bool {
		return dimensions[i].Name < dimensions[j].Name
	})
	return dimensions, errors
}

func parseDimensionFile(absPath, repoPath string) (Dimension, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return Dimension{}, fmt.Errorf("unable to read dimension file: %v", err)
	}
	frontmatter, body, err := splitFrontmatter(string(data))
	if err != nil {
		return Dimension{}, err
	}
	var metadata struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(frontmatter), &node); err != nil {
		return Dimension{}, fmt.Errorf("malformed YAML frontmatter: %v", err)
	}
	if len(node.Content) != 1 || node.Content[0].Kind != yaml.MappingNode {
		return Dimension{}, fmt.Errorf("frontmatter must be a YAML object")
	}
	for i := 0; i+1 < len(node.Content[0].Content); i += 2 {
		key := strings.TrimSpace(node.Content[0].Content[i].Value)
		if key != "name" && key != "description" {
			return Dimension{}, fmt.Errorf("unsupported frontmatter field %q", key)
		}
	}
	if err := yaml.Unmarshal([]byte(frontmatter), &metadata); err != nil {
		return Dimension{}, fmt.Errorf("malformed YAML frontmatter: %v", err)
	}
	metadata.Name = strings.TrimSpace(metadata.Name)
	metadata.Description = strings.TrimSpace(metadata.Description)
	body = strings.TrimSpace(body)
	if !ValidName(metadata.Name) {
		return Dimension{}, fmt.Errorf("field name must use lowercase alphanumeric segments separated by single hyphens")
	}
	if metadata.Description == "" {
		return Dimension{}, fmt.Errorf("field description must not be empty")
	}
	if body == "" {
		return Dimension{}, fmt.Errorf("instruction body must not be empty")
	}
	return Dimension{
		Name:         metadata.Name,
		Source:       SourceRepo,
		Description:  metadata.Description,
		Instructions: body,
		Path:         repoPath,
	}, nil
}

func splitFrontmatter(content string) (string, string, error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return "", "", fmt.Errorf("must start with YAML frontmatter")
	}
	rest := strings.TrimPrefix(content, "---\n")
	index := strings.Index(rest, "\n---")
	if index < 0 {
		return "", "", fmt.Errorf("missing closing YAML frontmatter marker")
	}
	frontmatter := rest[:index]
	body := rest[index+len("\n---"):]
	body = strings.TrimPrefix(body, "\n")
	return frontmatter, body, nil
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
			Source:      SourceBuiltin,
			Description: "Use when reviewing whether another agent can understand, resume, and safely act on the workflow state and command output.",
			Instructions: strings.TrimSpace(`
Review the change from the perspective of the next controller or reviewer agent.

Check whether command output, plan notes, review prompts, and handoff guidance are concise, accurate, and actionable. Look for hidden assumptions, missing next actions, ambiguous slot ownership, or wording that would make a future agent guess instead of act from repository evidence.
`),
		},
		{
			Name:        "correctness",
			Source:      SourceBuiltin,
			Description: "Use when reviewing implementation logic, workflow state transitions, command contracts, or negative-path behavior.",
			Instructions: strings.TrimSpace(`
Review the change for correctness.

Check implementation logic, command contracts, state transitions, validation behavior, and error paths. Look for stale assumptions, missing negative-path handling, mismatches between persisted artifacts and command output, and behavior that would break normal harness workflow progression.
`),
		},
		{
			Name:        "docs-consistency",
			Source:      SourceBuiltin,
			Description: "Use when changes touch README, AGENTS, specs, skills, schemas, examples, or other durable workflow guidance.",
			Instructions: strings.TrimSpace(`
Review the change for documentation and contract consistency.

Check whether README, AGENTS.md, managed skills, specs, schemas, examples, and plan text tell the same story as the implementation. Look for stale command names, mismatched field semantics, missing bootstrap asset updates, and guidance that would drift from the actual CLI behavior.
`),
		},
		{
			Name:        "evidence-validity",
			Source:      SourceBuiltin,
			Description: "Use when reviewing whether conclusions, syntheses, or decisions are supported by the scorecard, probes, evidence, residuals, and follow-up handling.",
			Instructions: strings.TrimSpace(`
Review the change for evidence validity.

Check whether accepted conclusions, syntheses, and decision artifacts are supported by the approved scorecard, tracked checkpoint reports, probes or experiments, durable evidence, and validation results. Look for weak evidence claims, missing comparisons, untested assumptions, rejected hypotheses that were not actually ruled out, residual uncertainty that is hidden or understated, and follow-up handling that leaves the decision less supported than the plan claims.
`),
		},
		{
			Name:        "risk-scan",
			Source:      SourceBuiltin,
			Description: "Use when reviewing unresolved blockers, leaked deferred scope, unsafe defaults, or release-sensitive workflow risks.",
			Instructions: strings.TrimSpace(`
Review the change for risk and unresolved scope.

Look for blockers that should stop closeout, deferred items that accidentally leaked into the current slice, unsafe defaults, brittle assumptions, migration or compatibility surprises, and release-sensitive behavior that needs stronger validation before the branch waits for merge approval.
`),
		},
		{
			Name:        "tests",
			Source:      SourceBuiltin,
			Description: "Use when reviewing whether coverage, fixtures, smoke tests, or validation claims prove the changed behavior.",
			Instructions: strings.TrimSpace(`
Review the change's test and validation coverage.

Check whether unit, integration, smoke, and manual validation evidence matches the behavioral risk. Look for missing assertions, weak fixtures, overbroad smoke claims, untested negative paths, and generated artifacts or bootstrap sync steps that should have been validated.
`),
		},
	}
}
