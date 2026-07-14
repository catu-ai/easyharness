package plan

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	templateassets "github.com/catu-ai/easyharness/assets/templates"
	"gopkg.in/yaml.v3"
)

var (
	stepHeadingPattern     = regexp.MustCompile(`^### Step [1-9][0-9]*: .+$`)
	planFilenamePattern    = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-([a-z0-9]+(?:-[a-z0-9]+)*)\.md$`)
	templateVersionPattern = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)$`)
	checkboxPattern        = regexp.MustCompile(`^- \[( |x|X)\] .+`)
	donePattern            = regexp.MustCompile(`^- Done:\s*\[( |x|X)\]\s*$`)
	stepContinuationHeader = regexp.MustCompile(`^#{1,6}(?:[ \t]|$)`)
	stepContinuationNumber = regexp.MustCompile(`^[0-9]+[.)][ \t]`)
	stepContinuationBullet = regexp.MustCompile(`^[-+*][ \t]`)
	stepContinuationSetext = regexp.MustCompile(`^[=-]+[ \t]*$`)
	stepContinuationRule   = regexp.MustCompile(`^(?:\*[ \t]*){3,}$|^(?:_[ \t]*){3,}$|^(?:-[ \t]*){3,}$`)
	issueURLPattern        = regexp.MustCompile(`https?://[^\s<>()]+/issues/[1-9][0-9]*(?:$|[/?#\s),.;\]])`)
	issueShorthandPattern  = regexp.MustCompile(`(?:^|[\s(\[])#[1-9][0-9]*(?:$|[\s),.;\]])`)
)

var (
	requiredTopSections = []string{
		"Goal",
		"Scope",
		"Acceptance Criteria",
		"Review Focus",
		"Deferred Items",
		"Work Breakdown",
		"Validation Strategy",
		"Closeout",
	}
	closeoutLabels = []string{"Validation", "Review", "Delivered", "Not Delivered", "Follow-Up Issues"}
)

type Frontmatter struct {
	TemplateVersion string   `yaml:"template_version"`
	CreatedAt       string   `yaml:"created_at"`
	ApprovedAt      string   `yaml:"approved_at,omitempty"`
	SourceType      string   `yaml:"source_type"`
	SourceRefs      []string `yaml:"source_refs"`
	Size            string   `yaml:"size"`
	WorkflowProfile string   `yaml:"workflow_profile,omitempty"`
}

type LintIssue struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type LintResult struct {
	OK                       bool          `json:"ok"`
	Command                  string        `json:"command"`
	Summary                  string        `json:"summary"`
	Artifacts                lintArtifacts `json:"artifacts,omitempty"`
	SupportedTemplateVersion string        `json:"supported_template_version,omitempty"`
	Errors                   []LintIssue   `json:"errors,omitempty"`
}

type lintArtifacts struct {
	PlanPath string `json:"plan_path"`
}

type lintContext struct {
	path           string
	frontmatter    Frontmatter
	rawFrontmatter map[string]any
	title          string
	sections       map[string]*section
	sectionOrder   []string
	steps          []step
	pathKind       string
}

type section struct {
	name  string
	lines []string
}

type step struct {
	title          string
	done           bool
	usesDoneMarker bool
	status         string
	outcome        string
	covers         string
	check          string
}

type checkboxItem struct {
	Checked bool
}

func LintFile(path string) LintResult {
	result := LintResult{
		Command:   "plan lint",
		Artifacts: lintArtifacts{PlanPath: path},
	}

	ctx, issues := parseAndValidate(path)
	if len(issues) > 0 {
		result.OK = false
		result.Summary = fmt.Sprintf("Plan is invalid with %d issue(s).", len(issues))
		result.Errors = issues
		if version, err := templateassets.PlanTemplateVersion(); err == nil {
			result.SupportedTemplateVersion = version
		}
		return result
	}

	version, err := templateassets.PlanTemplateVersion()
	if err == nil {
		result.SupportedTemplateVersion = version
	}
	result.OK = true
	result.Summary = fmt.Sprintf("Plan %q is valid.", ctx.title)
	return result
}

func parseAndValidate(path string) (*lintContext, []LintIssue) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, []LintIssue{{Path: "file", Message: err.Error()}}
	}

	issues := make([]LintIssue, 0)
	ctx := &lintContext{}
	ctx.path = path
	ctx.pathKind = inferPathKind(path)

	rawFrontmatter, body, err := splitFrontmatter(string(content))
	if err != nil {
		return nil, []LintIssue{{Path: "frontmatter", Message: err.Error()}}
	}

	var rawMap map[string]any
	if err := yaml.Unmarshal([]byte(rawFrontmatter), &rawMap); err != nil {
		issues = append(issues, LintIssue{Path: "frontmatter", Message: fmt.Sprintf("invalid YAML: %v", err)})
	} else {
		ctx.rawFrontmatter = rawMap
	}

	if err := yaml.Unmarshal([]byte(rawFrontmatter), &ctx.frontmatter); err != nil {
		issues = append(issues, LintIssue{Path: "frontmatter", Message: fmt.Sprintf("invalid YAML structure: %v", err)})
	}

	ctx.title, ctx.sections, ctx.sectionOrder = parseTopSections(body)
	if strings.TrimSpace(ctx.title) == "" {
		issues = append(issues, LintIssue{Path: "title", Message: "missing H1 title"})
	}

	issues = append(issues, validateFrontmatter(ctx)...)
	issues = append(issues, validateSectionOrder(ctx)...)
	issues = append(issues, validateGoal(ctx)...)
	issues = append(issues, validateScope(ctx)...)
	issues = append(issues, validateAcceptanceCriteria(ctx)...)
	issues = append(issues, validateReviewFocus(ctx)...)
	issues = append(issues, validateCloseout(ctx)...)

	steps, stepIssues := parseSteps(ctx)
	ctx.steps = steps
	issues = append(issues, stepIssues...)
	issues = append(issues, validateStepMarkers(ctx)...)

	issues = append(issues, validatePathRules(ctx)...)
	issues = append(issues, validateArchivedRules(ctx)...)

	return ctx, issues
}

func splitFrontmatter(content string) (string, string, error) {
	lines := strings.Split(content, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return "", "", errors.New("file must start with YAML frontmatter delimited by ---")
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(lines[1:i], "\n"), strings.Join(lines[i+1:], "\n"), nil
		}
	}
	return "", "", errors.New("frontmatter is missing a closing --- delimiter")
}

func parseTopSections(body string) (string, map[string]*section, []string) {
	lines := strings.Split(body, "\n")
	title := ""
	sections := map[string]*section{}
	order := make([]string, 0)
	var current *section

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")
		switch {
		case strings.HasPrefix(line, "# "):
			if title == "" {
				title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
			} else if current != nil {
				current.lines = append(current.lines, line)
			}
		case strings.HasPrefix(line, "## "):
			name := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			current = &section{name: name}
			sections[name] = current
			order = append(order, name)
		default:
			if current != nil {
				current.lines = append(current.lines, line)
			}
		}
	}

	return title, sections, order
}

func validateFrontmatter(ctx *lintContext) []LintIssue {
	issues := make([]LintIssue, 0)
	requiredKeys := []string{
		"template_version",
		"created_at",
		"source_type",
		"source_refs",
		"size",
	}
	for _, key := range requiredKeys {
		if _, ok := ctx.rawFrontmatter[key]; !ok {
			issues = append(issues, LintIssue{Path: "frontmatter." + key, Message: "missing required field"})
		}
	}

	if _, err := time.Parse(time.RFC3339, ctx.frontmatter.CreatedAt); err != nil {
		issues = append(issues, LintIssue{Path: "frontmatter.created_at", Message: "must be RFC3339"})
	}
	if strings.TrimSpace(ctx.frontmatter.CreatedAt) == "" {
		issues = append(issues, LintIssue{Path: "frontmatter.created_at", Message: "must not be empty"})
	}
	if _, ok := ctx.rawFrontmatter["approved_at"]; ok {
		if strings.TrimSpace(ctx.frontmatter.ApprovedAt) == "" {
			issues = append(issues, LintIssue{Path: "frontmatter.approved_at", Message: "must not be empty when provided"})
		} else if _, err := time.Parse(time.RFC3339, ctx.frontmatter.ApprovedAt); err != nil {
			issues = append(issues, LintIssue{Path: "frontmatter.approved_at", Message: "must be RFC3339"})
		}
	}
	if strings.TrimSpace(ctx.frontmatter.SourceType) == "" {
		issues = append(issues, LintIssue{Path: "frontmatter.source_type", Message: "must not be empty"})
	}
	if strings.TrimSpace(ctx.frontmatter.Size) == "" {
		issues = append(issues, LintIssue{Path: "frontmatter.size", Message: "must not be empty"})
	} else if !isSupportedPlanSize(ctx.frontmatter.Size) {
		issues = append(issues, LintIssue{Path: "frontmatter.size", Message: fmt.Sprintf("must be one of %s", strings.Join(supportedPlanSizes, ", "))})
	}
	for _, legacyKey := range []string{"status", "lifecycle", "revision", "updated_at"} {
		if _, ok := ctx.rawFrontmatter[legacyKey]; ok {
			issues = append(issues, LintIssue{
				Path:    "frontmatter." + legacyKey,
				Message: "legacy runtime field is no longer allowed in v0.2 tracked plans",
			})
		}
	}
	if rawProfile, ok := ctx.rawFrontmatter["workflow_profile"]; ok {
		value, ok := rawProfile.(string)
		if !ok || !slices.Contains([]string{WorkflowProfileStandard, WorkflowProfileLightweight}, strings.TrimSpace(value)) {
			issues = append(issues, LintIssue{
				Path:    "frontmatter.workflow_profile",
				Message: "must be standard or lightweight when provided",
			})
		}
	}
	if normalizeWorkflowProfile(ctx.frontmatter.WorkflowProfile) == WorkflowProfileLightweight && ctx.frontmatter.Size != PlanSizeXXS {
		issues = append(issues, LintIssue{
			Path:    "frontmatter.size",
			Message: "lightweight plans must use size XXS",
		})
	}
	supportedVersion, err := templateassets.PlanTemplateVersion()
	if err != nil {
		issues = append(issues, LintIssue{Path: "frontmatter.template_version", Message: err.Error()})
	} else if err := validateTemplateVersion(ctx.frontmatter.TemplateVersion, supportedVersion); err != nil {
		issues = append(issues, LintIssue{
			Path:    "frontmatter.template_version",
			Message: err.Error(),
		})
	}

	return issues
}

func validateSectionOrder(ctx *lintContext) []LintIssue {
	issues := make([]LintIssue, 0)
	if !slices.Equal(ctx.sectionOrder, requiredTopSections) {
		issues = append(issues, LintIssue{
			Path:    "sections",
			Message: fmt.Sprintf("top-level sections must appear in order: %s", strings.Join(requiredTopSections, " -> ")),
		})
	}
	return issues
}

func validateGoal(ctx *lintContext) []LintIssue {
	goal := ctx.sections["Goal"]
	if goal == nil {
		return []LintIssue{{Path: "section.Goal", Message: "missing Goal section"}}
	}
	for _, line := range goal.lines {
		if strings.TrimSpace(line) == "### Decisions and Constraints" {
			return nil
		}
	}
	return []LintIssue{{Path: "section.Goal", Message: "missing ### Decisions and Constraints"}}
}

func validateStepMarkers(ctx *lintContext) []LintIssue {
	issues := make([]LintIssue, 0, len(ctx.steps))
	for _, step := range ctx.steps {
		if step.usesDoneMarker {
			continue
		}
		issues = append(issues, LintIssue{
			Path:    "step." + step.title,
			Message: "step must use '- Done: [ ]' or '- Done: [x]'; legacy '- Status: ...' is no longer allowed",
		})
	}
	return issues
}

func validateScope(ctx *lintContext) []LintIssue {
	scope := ctx.sections["Scope"]
	if scope == nil {
		return []LintIssue{{Path: "section.Scope", Message: "missing Scope section"}}
	}
	content := strings.Join(scope.lines, "\n")
	issues := make([]LintIssue, 0)
	if !strings.Contains(content, "### In Scope") {
		issues = append(issues, LintIssue{Path: "section.Scope", Message: "missing ### In Scope"})
	}
	if !strings.Contains(content, "### Out of Scope") {
		issues = append(issues, LintIssue{Path: "section.Scope", Message: "missing ### Out of Scope"})
	}
	return issues
}

func validateAcceptanceCriteria(ctx *lintContext) []LintIssue {
	section := ctx.sections["Acceptance Criteria"]
	if section == nil {
		return []LintIssue{{Path: "section.Acceptance Criteria", Message: "missing Acceptance Criteria section"}}
	}
	issues := make([]LintIssue, 0)
	items, err := parseCheckboxList(section.lines)
	if err != nil {
		issues = append(issues, LintIssue{Path: "section.Acceptance Criteria", Message: err.Error()})
		return issues
	}
	if len(items) == 0 {
		issues = append(issues, LintIssue{Path: "section.Acceptance Criteria", Message: "must contain at least one checkbox"})
	}
	if ctx.pathKind == "archived" {
		for _, item := range items {
			if !item.Checked {
				issues = append(issues, LintIssue{Path: "section.Acceptance Criteria", Message: "archived plans must have all acceptance criteria checked"})
				break
			}
		}
	}
	return issues
}

func validateReviewFocus(ctx *lintContext) []LintIssue {
	section := ctx.sections["Review Focus"]
	if section == nil {
		return []LintIssue{{Path: "section.Review Focus", Message: "missing Review Focus section"}}
	}
	if strings.TrimSpace(strings.Join(section.lines, "\n")) == "" {
		return []LintIssue{{Path: "section.Review Focus", Message: "must not be empty"}}
	}
	return nil
}

func validateCloseout(ctx *lintContext) []LintIssue {
	section := ctx.sections["Closeout"]
	if section == nil {
		return []LintIssue{{Path: "section.Closeout", Message: "missing Closeout section"}}
	}
	_, issues := parseLabeledBullets("section.Closeout", section.lines, closeoutLabels)
	return issues
}

func parseSteps(ctx *lintContext) ([]step, []LintIssue) {
	workBreakdown := ctx.sections["Work Breakdown"]
	if workBreakdown == nil {
		return nil, []LintIssue{{Path: "section.Work Breakdown", Message: "missing Work Breakdown section"}}
	}

	lines := workBreakdown.lines
	steps := make([]step, 0)
	issues := make([]LintIssue, 0)
	var current *step
	var buffer []string

	flush := func() {
		if current == nil {
			return
		}
		parsed, errs := finalizeStep(*current, buffer)
		steps = append(steps, parsed)
		issues = append(issues, errs...)
	}

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")
		if strings.HasPrefix(line, "### ") {
			flush()
			if !stepHeadingPattern.MatchString(line) {
				issues = append(issues, LintIssue{
					Path:    "section.Work Breakdown",
					Message: fmt.Sprintf("invalid step heading %q; use ### Step N: Title", strings.TrimSpace(strings.TrimPrefix(line, "### "))),
				})
			}
			current = &step{title: strings.TrimSpace(strings.TrimPrefix(line, "### "))}
			buffer = nil
			continue
		}
		if current != nil {
			buffer = append(buffer, line)
		}
	}
	flush()

	if len(steps) == 0 {
		issues = append(issues, LintIssue{Path: "section.Work Breakdown", Message: "must contain at least one step"})
	}
	return steps, issues
}

func finalizeStep(base step, lines []string) (step, []LintIssue) {
	issues := make([]LintIssue, 0)
	stepPath := "step." + base.title
	trimmedIndex := -1
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			trimmedIndex = i
			break
		}
	}
	if trimmedIndex == -1 {
		return base, []LintIssue{{Path: stepPath, Message: "step body is empty"}}
	}
	if doneMatches := donePattern.FindStringSubmatch(strings.TrimSpace(lines[trimmedIndex])); len(doneMatches) == 2 {
		base.done = strings.EqualFold(doneMatches[1], "x")
		base.usesDoneMarker = true
		base.status = stepStatusFromDone(base.done)
	} else {
		return base, []LintIssue{{Path: stepPath, Message: "step must start with '- Done: [ ]' or '- Done: [x]'"}}
	}

	seen := map[string]bool{}
	values := map[string][]string{}
	previousOrder := -1
	fieldOrder := map[string]int{"Outcome": 0, "Covers": 1, "Check": 2}
	currentField := ""
	separatedFromField := false
	for _, rawLine := range lines[trimmedIndex+1:] {
		line := strings.TrimRight(rawLine, "\r")
		if strings.TrimSpace(line) == "" {
			if currentField != "" {
				separatedFromField = true
			}
			continue
		}

		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			if !strings.HasPrefix(line, "  ") {
				issues = append(issues, LintIssue{Path: stepPath, Message: "step field continuation lines must be indented by at least two spaces"})
				continue
			}
			continuation := strings.TrimSpace(line)
			if currentField == "" {
				issues = append(issues, LintIssue{Path: stepPath, Message: "step field continuation line must follow Outcome, Covers, or Check"})
				continue
			}
			if separatedFromField {
				issues = append(issues, LintIssue{Path: stepPath + "." + strings.ToLower(currentField), Message: "continuation lines must directly follow the field without a blank line"})
				continue
			}
			if isUnsupportedStepContinuation(continuation) {
				issues = append(issues, LintIssue{Path: stepPath + "." + strings.ToLower(currentField), Message: "continuation lines may contain ordinary wrapped prose only, not headings, thematic breaks, blockquotes, fences, or nested lists"})
				continue
			}
			values[currentField] = append(values[currentField], continuation)
			continue
		}

		currentField = ""
		separatedFromField = false
		trimmedLine := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmedLine, "- ") || !strings.Contains(trimmedLine, ":") {
			issues = append(issues, LintIssue{Path: stepPath, Message: "steps may contain only Outcome, Covers, optional Check, and their indented continuation lines"})
			continue
		}
		parts := strings.SplitN(strings.TrimPrefix(trimmedLine, "- "), ":", 2)
		name := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		order, ok := fieldOrder[name]
		if !ok {
			issues = append(issues, LintIssue{Path: stepPath, Message: fmt.Sprintf("unknown step field %q", name)})
			continue
		}
		if order < previousOrder {
			issues = append(issues, LintIssue{Path: stepPath, Message: "step fields must appear in Outcome, Covers, Check order"})
		}
		previousOrder = order
		if seen[name] {
			issues = append(issues, LintIssue{Path: stepPath + "." + strings.ToLower(name), Message: "must appear only once"})
			continue
		}
		seen[name] = true
		currentField = name
		if value != "" {
			values[name] = append(values[name], value)
		}
	}

	base.outcome = strings.Join(values["Outcome"], "\n")
	base.covers = strings.Join(values["Covers"], "\n")
	base.check = strings.Join(values["Check"], "\n")
	for _, name := range []string{"Outcome", "Covers", "Check"} {
		if seen[name] && len(values[name]) == 0 {
			issues = append(issues, LintIssue{Path: stepPath + "." + strings.ToLower(name), Message: "must not be empty"})
		}
	}

	if base.outcome == "" && !seen["Outcome"] {
		issues = append(issues, LintIssue{Path: stepPath + ".outcome", Message: "missing Outcome field"})
	}
	if base.covers == "" && !seen["Covers"] {
		issues = append(issues, LintIssue{Path: stepPath + ".covers", Message: "missing Covers field"})
	}

	return base, issues
}

func isUnsupportedStepContinuation(line string) bool {
	if stepContinuationHeader.MatchString(line) ||
		stepContinuationSetext.MatchString(line) ||
		stepContinuationRule.MatchString(line) ||
		strings.HasPrefix(line, ">") ||
		strings.HasPrefix(line, "```") ||
		strings.HasPrefix(line, "~~~") {
		return true
	}
	return stepContinuationBullet.MatchString(line) || stepContinuationNumber.MatchString(line)
}

func validatePathRules(ctx *lintContext) []LintIssue {
	issues := make([]LintIssue, 0)
	switch ctx.pathKind {
	case "active":
	case "archived":
	default:
		issues = append(issues, LintIssue{Path: "path", Message: "plan must live under a configured active or archived plan root"})
	}
	relativeWithinRoot := relativePathWithinPlanRoot(ctx.path)
	if strings.HasPrefix(relativeWithinRoot, SupplementsDirName+"/") {
		issues = append(issues, LintIssue{Path: "path", Message: "plan markdown must not live inside a supplements directory"})
	}

	if filenameErr := validatePlanFilename(filepath.Base(ctx.path)); filenameErr != nil {
		issues = append(issues, LintIssue{Path: "path", Message: filenameErr.Error()})
	}

	pathProfile := inferWorkflowProfileFromPath(ctx.path)
	declaredProfile := strings.TrimSpace(ctx.frontmatter.WorkflowProfile)
	if declaredProfile == WorkflowProfileStandard {
		issues = append(issues, LintIssue{Path: "frontmatter.workflow_profile", Message: "omit workflow_profile for standard plans; only lightweight plans should declare it"})
	}
	switch pathProfile {
	case WorkflowProfileStandard:
		switch declaredProfile {
		case WorkflowProfileLightweight:
			issues = append(issues, LintIssue{Path: "frontmatter.workflow_profile", Message: "standard archived paths must omit workflow_profile"})
		}
	case WorkflowProfileLightweight:
		if declaredProfile != WorkflowProfileLightweight {
			issues = append(issues, LintIssue{Path: "frontmatter.workflow_profile", Message: "lightweight archived paths require workflow_profile: lightweight"})
		}
	default:
		if ctx.pathKind == "active" && declaredProfile != "" && declaredProfile != WorkflowProfileLightweight {
			issues = append(issues, LintIssue{Path: "frontmatter.workflow_profile", Message: "tracked active plans must omit workflow_profile unless they explicitly use lightweight"})
		}
	}
	issues = append(issues, validateSupplementsRules(ctx)...)
	return issues
}

func validateSupplementsRules(ctx *lintContext) []LintIssue {
	issues := make([]LintIssue, 0)
	supplementsPath := SupplementsDirForPlanPath(ctx.path)
	for _, root := range candidateSupplementsRoots(ctx.path) {
		info, err := os.Stat(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return []LintIssue{{Path: "supplements", Message: err.Error()}}
		}
		if !info.IsDir() {
			return []LintIssue{{
				Path:    "supplements",
				Message: fmt.Sprintf("supplements parent path must be a directory when present: %s", filepath.ToSlash(filepath.Clean(root))),
			}}
		}
	}
	info, err := os.Stat(supplementsPath)
	if err != nil && !os.IsNotExist(err) {
		return []LintIssue{{Path: "supplements", Message: err.Error()}}
	}
	if err == nil && !info.IsDir() {
		return []LintIssue{{Path: "supplements", Message: "supplements path must be a directory when present"}}
	}
	if err == nil {
		cleanPlan := filepath.ToSlash(filepath.Clean(ctx.path))
		cleanSupplements := filepath.ToSlash(filepath.Clean(supplementsPath))
		planStem := strings.TrimSuffix(filepath.Base(cleanPlan), filepath.Ext(cleanPlan))
		if filepath.Base(cleanSupplements) != planStem {
			return []LintIssue{{Path: "supplements", Message: "supplements directory name must match the markdown plan stem"}}
		}
	}
	for _, alternate := range AlternateSupplementsDirsForPlanPath(ctx.path) {
		if _, err := os.Stat(alternate); err == nil {
			return []LintIssue{{
				Path:    "supplements",
				Message: fmt.Sprintf("supplements for this plan stem must live only under the matching root; conflicting path present at %s", filepath.ToSlash(filepath.Clean(alternate))),
			}}
		} else if err != nil && !os.IsNotExist(err) {
			return []LintIssue{{Path: "supplements", Message: err.Error()}}
		}
	}

	return issues
}

func candidateSupplementsRoots(path string) []string {
	roots := []string{filepath.Dir(SupplementsDirForPlanPath(path))}
	for _, alternate := range AlternateSupplementsDirsForPlanPath(path) {
		roots = append(roots, filepath.Dir(alternate))
	}
	return slices.Compact(roots)
}

func validateArchivedRules(ctx *lintContext) []LintIssue {
	if ctx.pathKind != "archived" {
		return nil
	}

	issues := make([]LintIssue, 0)
	for _, step := range ctx.steps {
		if !step.done {
			issues = append(issues, LintIssue{Path: "step." + step.title + ".done", Message: "archived plans require every step to be done"})
		}
	}

	closeout := ctx.sections["Closeout"]
	if closeout == nil {
		issues = append(issues, LintIssue{Path: "section.Closeout", Message: "missing Closeout section"})
	} else {
		content := strings.Join(closeout.lines, "\n")
		if containsArchivePlaceholderToken(content) {
			issues = append(issues, LintIssue{Path: "section.Closeout", Message: "archived plans must not keep archive-time placeholder tokens"})
		}
		values, parseIssues := parseLabeledBullets("section.Closeout", closeout.lines, append([]string{"Archived At", "Revision"}, closeoutLabels...))
		issues = append(issues, parseIssues...)
		if deferredItemsSection := ctx.sections["Deferred Items"]; deferredItemsSection != nil && hasRealDeferredItems(strings.Join(deferredItemsSection.lines, "\n")) {
			if !hasConcreteFollowUpIssueReference(values["Follow-Up Issues"]) {
				issues = append(issues, LintIssue{Path: "section.Closeout.Follow-Up Issues", Message: "archived plans with deferred items must include a concrete issue URL or #number reference"})
			}
		}
	}

	return issues
}

func parseLabeledBullets(path string, lines, required []string) (map[string]string, []LintIssue) {
	partsByLabel := map[string][]string{}
	issues := make([]LintIssue, 0)
	currentLabel := ""
	separatedFromField := false
	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")
		if strings.TrimSpace(line) == "" {
			if currentLabel != "" {
				separatedFromField = true
			}
			continue
		}
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			if !strings.HasPrefix(line, "  ") {
				issues = append(issues, LintIssue{Path: path, Message: "field continuation lines must be indented by at least two spaces"})
				continue
			}
			continuation := strings.TrimSpace(line)
			if currentLabel == "" {
				issues = append(issues, LintIssue{Path: path, Message: "field continuation line must follow a labeled bullet"})
				continue
			}
			if separatedFromField {
				issues = append(issues, LintIssue{Path: path + "." + currentLabel, Message: "continuation lines must directly follow the field without a blank line"})
				continue
			}
			if isUnsupportedStepContinuation(continuation) {
				issues = append(issues, LintIssue{Path: path + "." + currentLabel, Message: "continuation lines may contain ordinary wrapped prose only, not headings, thematic breaks, blockquotes, fences, or nested lists"})
				continue
			}
			partsByLabel[currentLabel] = append(partsByLabel[currentLabel], continuation)
			continue
		}

		currentLabel = ""
		separatedFromField = false
		trimmedLine := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmedLine, "- ") || !strings.Contains(trimmedLine, ":") {
			issues = append(issues, LintIssue{Path: path, Message: "must contain only labeled bullet lines and their indented prose continuations"})
			continue
		}
		parts := strings.SplitN(strings.TrimPrefix(trimmedLine, "- "), ":", 2)
		label := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if _, exists := partsByLabel[label]; exists {
			issues = append(issues, LintIssue{Path: path + "." + label, Message: "must appear only once"})
			continue
		}
		partsByLabel[label] = nil
		currentLabel = label
		if value != "" {
			partsByLabel[label] = append(partsByLabel[label], value)
		}
	}
	values := make(map[string]string, len(partsByLabel))
	for label, parts := range partsByLabel {
		values[label] = strings.Join(parts, "\n")
		if len(parts) == 0 {
			issues = append(issues, LintIssue{Path: path + "." + label, Message: "must not be empty"})
		}
	}
	for _, label := range required {
		if _, ok := values[label]; !ok {
			issues = append(issues, LintIssue{Path: path + "." + label, Message: "missing required closeout field"})
		}
	}
	return values, issues
}

func hasConcreteFollowUpIssueReference(value string) bool {
	return issueURLPattern.MatchString(value) || issueShorthandPattern.MatchString(value)
}

func parseCheckboxList(lines []string) ([]checkboxItem, error) {
	items := make([]checkboxItem, 0)
	hasItem := false
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		if checkboxPattern.MatchString(line) {
			hasItem = true
			items = append(items, checkboxItem{Checked: strings.HasPrefix(line, "- [x]") || strings.HasPrefix(line, "- [X]")})
			continue
		}
		if strings.HasPrefix(line, "- ") {
			return nil, fmt.Errorf("must use markdown checkboxes")
		}
		if !hasItem {
			return nil, fmt.Errorf("must start with a markdown checkbox")
		}
	}
	return items, nil
}

func parseLevelThreeSections(lines []string) (map[string]*section, []string) {
	sections := map[string]*section{}
	order := make([]string, 0)
	var current *section
	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")
		if strings.HasPrefix(line, "### ") {
			name := strings.TrimSpace(strings.TrimPrefix(line, "### "))
			current = &section{name: name}
			sections[name] = current
			order = append(order, name)
			continue
		}
		if current != nil {
			current.lines = append(current.lines, line)
		}
	}
	return sections, order
}

func hasPlaceholder(section *section, token string) bool {
	if section == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(strings.Join(section.lines, "\n")), token)
}

func hasRealDeferredItems(content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return false
	}
	candidates := []string{
		"NONE",
		"None.",
		"- None.",
		"- NONE",
	}
	for _, candidate := range candidates {
		if trimmed == candidate {
			return false
		}
	}
	return true
}

func validatePlanFilename(filename string) error {
	matches := planFilenamePattern.FindStringSubmatch(filename)
	if len(matches) != 3 {
		return fmt.Errorf("plan filename must match YYYY-MM-DD-short-topic.md")
	}
	if _, err := time.Parse("2006-01-02", matches[1]); err != nil {
		return fmt.Errorf("plan filename must start with a valid date")
	}
	return nil
}

func validateTemplateVersion(planVersion, supportedVersion string) error {
	planSemver, err := parseTemplateVersion(planVersion)
	if err != nil {
		return fmt.Errorf("template_version must be semver-like (for example 0.1.0)")
	}
	supportedSemver, err := parseTemplateVersion(supportedVersion)
	if err != nil {
		return fmt.Errorf("supported template version is invalid: %v", err)
	}
	if compareTemplateVersions(planSemver, supportedSemver) > 0 {
		return fmt.Errorf("template_version %q is newer than this harness supports (%q)", planVersion, supportedVersion)
	}
	return nil
}

func parseTemplateVersion(version string) ([3]int, error) {
	matches := templateVersionPattern.FindStringSubmatch(strings.TrimSpace(version))
	if len(matches) != 4 {
		return [3]int{}, fmt.Errorf("invalid version %q", version)
	}

	var parsed [3]int
	for i := 1; i < len(matches); i++ {
		value, err := strconv.Atoi(matches[i])
		if err != nil {
			return [3]int{}, err
		}
		parsed[i-1] = value
	}
	return parsed, nil
}

func compareTemplateVersions(left, right [3]int) int {
	for i := 0; i < len(left); i++ {
		switch {
		case left[i] < right[i]:
			return -1
		case left[i] > right[i]:
			return 1
		}
	}
	return 0
}

func stepStatusFromDone(done bool) string {
	if done {
		return "completed"
	}
	return "pending"
}
