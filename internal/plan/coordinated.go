package plan

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const SubplanResultPending = "PENDING"

var subplanIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

var (
	subplanSections     = []string{"Outcome", "Work Breakdown", "Result"}
	subplanResultLabels = []string{"Validation", "Delivered"}
	readSubplanSnapshot = loadSubplanSnapshot
)

type SubplanFrontmatter struct {
	DependsOn []string `yaml:"depends_on,omitempty"`
}

type SubplanResult struct {
	Validation string
	Delivered  string
}

type SubplanDocument struct {
	Path         string
	ID           string
	Title        string
	DependsOn    []string
	Steps        []DocumentStep
	Result       SubplanResult
	PathKind     string
	RootPlanPath string
}

func (d *SubplanDocument) CurrentStep() *DocumentStep {
	if d == nil {
		return nil
	}
	return currentStep(d.Steps)
}

func (d *SubplanDocument) AllStepsCompleted() bool {
	if d == nil {
		return false
	}
	return allStepsCompleted(d.Steps)
}

func (d *SubplanDocument) Completed() bool {
	if d == nil || !d.AllStepsCompleted() {
		return false
	}
	return completedSubplanResult(d.Result.Validation) &&
		completedSubplanResult(d.Result.Delivered)
}

type CoordinatedPackage struct {
	Root     *Document
	Subplans []*SubplanDocument
}

type CoordinatedProgress struct {
	Total     int
	Completed int
	Runnable  int
	Waiting   int
}

func LoadSubplanFile(path string) (*SubplanDocument, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	doc, issues := parseAndValidateSubplanContent(path, content)
	if len(issues) > 0 {
		return nil, fmt.Errorf("%s: %s", issues[0].Path, issues[0].Message)
	}
	return doc, nil
}

func LoadCoordinatedPackage(rootPath string) (*CoordinatedPackage, error) {
	root, err := LoadFile(rootPath)
	if err != nil {
		return nil, err
	}
	if !root.UsesCoordinatedProfile() {
		return nil, fmt.Errorf("plan %s does not use workflow_profile: coordinated", filepath.ToSlash(rootPath))
	}

	initial, err := readSubplanSnapshot(rootPath)
	if err != nil {
		return nil, err
	}

	pkg := &CoordinatedPackage{Root: root}
	for _, file := range initial {
		subplan, issues := parseAndValidateSubplanContent(file.path, file.content)
		if len(issues) > 0 {
			return nil, fmt.Errorf("load subplan %q: %s: %s", file.name, issues[0].Path, issues[0].Message)
		}
		pkg.Subplans = append(pkg.Subplans, subplan)
	}
	verified, err := readSubplanSnapshot(rootPath)
	if err != nil {
		return nil, err
	}
	if !equalSubplanSnapshots(initial, verified) {
		return nil, fmt.Errorf("subplan package changed while it was being read; retry")
	}
	sort.Slice(pkg.Subplans, func(i, j int) bool {
		return pkg.Subplans[i].ID < pkg.Subplans[j].ID
	})
	return pkg, nil
}

type subplanSnapshotFile struct {
	name    string
	path    string
	content []byte
}

func loadSubplanSnapshot(rootPath string) ([]subplanSnapshotFile, error) {
	supplementsDir := SupplementsDirForPlanPath(rootPath)
	if err := rejectSymlinkPath(supplementsDir, "supplements"); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	subplansDir := SubplansDirForPlanPath(rootPath)
	if err := rejectSymlinkPath(subplansDir, "subplans"); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	entries, err := os.ReadDir(subplansDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	snapshot := make([]subplanSnapshotFile, 0, len(entries))
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("subplans must be in-package regular files; symlink %q is not allowed", entry.Name())
		}
		if entry.IsDir() {
			return nil, fmt.Errorf("subplans must be flat; nested directory %q is not allowed", entry.Name())
		}
		if filepath.Ext(entry.Name()) != ".md" {
			return nil, fmt.Errorf("subplans directory may contain only Markdown subplan files; found %q", entry.Name())
		}
		path := filepath.Join(subplansDir, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("subplans must be in-package regular files; found %q", entry.Name())
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		snapshot = append(snapshot, subplanSnapshotFile{
			name:    entry.Name(),
			path:    path,
			content: content,
		})
	}
	return snapshot, nil
}

func rejectSymlinkPath(path, label string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s path must be a tracked directory, not a symlink", label)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s path must be a directory", label)
	}
	return nil
}

func equalSubplanSnapshots(left, right []subplanSnapshotFile) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].name != right[index].name ||
			!bytes.Equal(left[index].content, right[index].content) {
			return false
		}
	}
	return true
}

func (p *CoordinatedPackage) Subplan(id string) *SubplanDocument {
	if p == nil {
		return nil
	}
	for _, subplan := range p.Subplans {
		if subplan.ID == id {
			return subplan
		}
	}
	return nil
}

func (p *CoordinatedPackage) AllSubplansCompleted() bool {
	if p == nil {
		return false
	}
	for _, subplan := range p.Subplans {
		if !subplan.Completed() {
			return false
		}
	}
	return true
}

func (p *CoordinatedPackage) CompletionIssues() []DocumentIssue {
	if p == nil {
		return []DocumentIssue{{Path: "subplans", Message: "coordinated package is unavailable"}}
	}
	issues := append([]DocumentIssue(nil), p.DependencyIssues()...)
	for _, subplan := range p.Subplans {
		if !subplan.Completed() {
			issues = append(issues, DocumentIssue{
				Path:    "subplan." + subplan.ID,
				Message: "all subplan steps and Result fields must be complete",
			})
		}
	}
	return issues
}

func (p *CoordinatedPackage) Progress() CoordinatedProgress {
	progress := CoordinatedProgress{}
	if p == nil {
		return progress
	}
	progress.Total = len(p.Subplans)
	for _, subplan := range p.Subplans {
		if subplan.Completed() {
			progress.Completed++
			continue
		}
		ready := true
		for _, dependency := range subplan.DependsOn {
			dependencyPlan := p.Subplan(dependency)
			if dependencyPlan == nil || !dependencyPlan.Completed() {
				ready = false
				break
			}
		}
		if ready {
			progress.Runnable++
		} else {
			progress.Waiting++
		}
	}
	return progress
}

func (p *CoordinatedPackage) DependencyIssues() []DocumentIssue {
	if p == nil {
		return nil
	}
	issues := make([]DocumentIssue, 0)
	byID := make(map[string]*SubplanDocument, len(p.Subplans))
	for _, subplan := range p.Subplans {
		if _, exists := byID[subplan.ID]; exists {
			issues = append(issues, DocumentIssue{
				Path:    "subplan." + subplan.ID,
				Message: "duplicate subplan identifier",
			})
			continue
		}
		byID[subplan.ID] = subplan
	}
	for _, subplan := range p.Subplans {
		for _, dependency := range subplan.DependsOn {
			switch {
			case dependency == subplan.ID:
				issues = append(issues, DocumentIssue{
					Path:    "subplan." + subplan.ID + ".depends_on",
					Message: "subplan must not depend on itself",
				})
			case byID[dependency] == nil:
				issues = append(issues, DocumentIssue{
					Path:    "subplan." + subplan.ID + ".depends_on",
					Message: fmt.Sprintf("missing dependency %q", dependency),
				})
			}
		}
	}

	state := make(map[string]uint8, len(byID))
	stack := make([]string, 0, len(byID))
	reportedCycles := map[string]bool{}
	var visit func(string)
	visit = func(id string) {
		switch state[id] {
		case 1:
			start := 0
			for start < len(stack) && stack[start] != id {
				start++
			}
			cycle := append(append([]string{}, stack[start:]...), id)
			key := strings.Join(cycle, " -> ")
			if !reportedCycles[key] {
				reportedCycles[key] = true
				issues = append(issues, DocumentIssue{
					Path:    "subplan." + id + ".depends_on",
					Message: "dependency cycle: " + key,
				})
			}
			return
		case 2:
			return
		}
		state[id] = 1
		stack = append(stack, id)
		for _, dependency := range byID[id].DependsOn {
			if byID[dependency] != nil && dependency != id {
				visit(dependency)
			}
		}
		stack = stack[:len(stack)-1]
		state[id] = 2
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		visit(id)
	}
	return issues
}

func parseAndValidateSubplan(path string) (*SubplanDocument, []LintIssue) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, []LintIssue{{Path: "file", Message: err.Error()}}
	}
	return parseAndValidateSubplanContent(path, content)
}

func parseAndValidateSubplanContent(path string, content []byte) (*SubplanDocument, []LintIssue) {
	rawFrontmatter, body, err := splitFrontmatter(string(content))
	if err != nil {
		return nil, []LintIssue{{Path: "frontmatter", Message: err.Error()}}
	}

	issues := make([]LintIssue, 0)
	var raw map[string]any
	if err := yaml.Unmarshal([]byte(rawFrontmatter), &raw); err != nil {
		issues = append(issues, LintIssue{Path: "frontmatter", Message: fmt.Sprintf("invalid YAML: %v", err)})
	}
	for key := range raw {
		if key != "depends_on" {
			issues = append(issues, LintIssue{Path: "frontmatter." + key, Message: "subplans allow only the optional depends_on field"})
		}
	}
	var frontmatter SubplanFrontmatter
	if err := yaml.Unmarshal([]byte(rawFrontmatter), &frontmatter); err != nil {
		issues = append(issues, LintIssue{Path: "frontmatter", Message: fmt.Sprintf("invalid YAML structure: %v", err)})
	}

	rootPath, pathErr := RootPathForSubplanPath(path)
	if pathErr != nil {
		issues = append(issues, LintIssue{Path: "path", Message: pathErr.Error()})
	}
	id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if err := validateSubplanID(id); err != nil {
		issues = append(issues, LintIssue{Path: "path", Message: err.Error()})
	}
	seenDependencies := map[string]bool{}
	for _, dependency := range frontmatter.DependsOn {
		if err := validateSubplanID(dependency); err != nil {
			issues = append(issues, LintIssue{Path: "frontmatter.depends_on", Message: fmt.Sprintf("invalid dependency %q: %v", dependency, err)})
		}
		if seenDependencies[dependency] {
			issues = append(issues, LintIssue{Path: "frontmatter.depends_on", Message: fmt.Sprintf("duplicate dependency %q", dependency)})
		}
		seenDependencies[dependency] = true
	}

	title, sections, order := parseTopSections(body)
	if strings.TrimSpace(title) == "" {
		issues = append(issues, LintIssue{Path: "title", Message: "missing H1 title"})
	}
	if !equalStrings(order, subplanSections) {
		issues = append(issues, LintIssue{
			Path:    "sections",
			Message: "subplan sections must appear in order: " + strings.Join(subplanSections, " -> "),
		})
	}
	if outcome := sections["Outcome"]; outcome == nil || strings.TrimSpace(strings.Join(outcome.lines, "\n")) == "" {
		issues = append(issues, LintIssue{Path: "section.Outcome", Message: "must not be empty"})
	}
	stepContext := &lintContext{sections: sections}
	parsedSteps, stepIssues := parseSteps(stepContext)
	issues = append(issues, stepIssues...)
	stepContext.steps = parsedSteps
	issues = append(issues, validateStepMarkers(stepContext)...)

	resultValues := map[string]string{}
	if resultSection := sections["Result"]; resultSection == nil {
		issues = append(issues, LintIssue{Path: "section.Result", Message: "missing Result section"})
	} else {
		var resultIssues []LintIssue
		resultValues, resultIssues = parseLabeledBullets("section.Result", resultSection.lines, subplanResultLabels)
		issues = append(issues, resultIssues...)
		for label := range resultValues {
			if label != "Validation" && label != "Delivered" {
				issues = append(issues, LintIssue{Path: "section.Result." + label, Message: "unknown result field"})
			}
		}
	}

	doc := &SubplanDocument{
		Path:         path,
		ID:           id,
		Title:        title,
		DependsOn:    append([]string(nil), frontmatter.DependsOn...),
		PathKind:     inferPathKind(path),
		RootPlanPath: rootPath,
		Result: SubplanResult{
			Validation: resultValues["Validation"],
			Delivered:  resultValues["Delivered"],
		},
	}
	for _, parsedStep := range parsedSteps {
		doc.Steps = append(doc.Steps, documentStep(parsedStep))
	}
	return doc, issues
}

func validateSubplanID(id string) error {
	if !subplanIDPattern.MatchString(id) {
		return fmt.Errorf("subplan identifier must be a lowercase kebab-case slug")
	}
	return nil
}

func completedSubplanResult(value string) bool {
	trimmed := strings.TrimSpace(value)
	return trimmed != "" && !strings.EqualFold(trimmed, SubplanResultPending)
}

func documentStep(parsed step) DocumentStep {
	return DocumentStep{
		Title:          parsed.title,
		Done:           parsed.done,
		UsesDoneMarker: parsed.usesDoneMarker,
		Status:         parsed.status,
		Outcome:        parsed.outcome,
		Covers:         parsed.covers,
		Check:          parsed.check,
	}
}

func currentStep(steps []DocumentStep) *DocumentStep {
	for i := range steps {
		if !steps[i].Done {
			return &steps[i]
		}
	}
	return nil
}

func allStepsCompleted(steps []DocumentStep) bool {
	if len(steps) == 0 {
		return false
	}
	for _, step := range steps {
		if !step.Done {
			return false
		}
	}
	return true
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
