package reviewcoverage

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/catu-ai/easyharness/internal/plan"
	"gopkg.in/yaml.v3"
)

const allowedCloseoutSection = "Closeout"

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

// ValidateArchivedCandidate binds publish and merge handoff to the reviewed
// candidate. It permits only the mechanical archive move of the reviewed plan
// and its supplements, plus Closeout-body changes that were already allowed at
// the archive boundary.
func ValidateArchivedCandidate(workdir, archivedPlanPath string, chain *Chain) error {
	if chain == nil || strings.TrimSpace(chain.CoveredHeadSHA) == "" {
		return fmt.Errorf("archived candidate has no reviewed head coverage")
	}
	reviewedPlan, err := repoRelativePath(workdir, chain.ReviewedPlanPath)
	if err != nil {
		return fmt.Errorf("resolve reviewed plan path: %w", err)
	}
	archivedPlan, err := repoRelativePath(workdir, archivedPlanPath)
	if err != nil {
		return fmt.Errorf("resolve archived plan path: %w", err)
	}
	if reviewedPlan == archivedPlan {
		return fmt.Errorf("archived plan path still equals reviewed active plan path")
	}

	currentHead, err := ResolveCommit(workdir, "HEAD")
	if err != nil {
		return fmt.Errorf("resolve archived candidate head: %w", err)
	}
	ancestor, err := IsAncestor(workdir, chain.CoveredHeadSHA, currentHead)
	if err != nil {
		return fmt.Errorf("validate reviewed candidate ancestry: %w", err)
	}
	if !ancestor {
		return fmt.Errorf("archived candidate HEAD is not descended from reviewed head %s; reopen and review the current candidate", chain.CoveredHeadSHA)
	}

	baseline, err := gitBytes(workdir, "show", chain.CoveredHeadSHA+":"+reviewedPlan)
	if err != nil {
		return fmt.Errorf("read reviewed plan content: %w", err)
	}
	archivedPlanFSPath := filepath.Join(workdir, filepath.FromSlash(archivedPlan))
	currentMode, current, err := readGitWorktreeFile(archivedPlanFSPath)
	if err != nil {
		return fmt.Errorf("read archived plan content: %w", err)
	}
	if currentMode != "100644" {
		return fmt.Errorf("archived plan has git mode %s, expected command-rendered regular file mode 100644", currentMode)
	}
	expected, err := commandRenderedArchivePlan(baseline)
	if err != nil {
		return fmt.Errorf("prepare reviewed plan for archive comparison: %w", err)
	}
	expectedComparable := maskCloseoutBodies(expected)
	reviewedComparable := maskCloseoutBodies(baseline)
	currentComparable := maskCloseoutBodies(current)
	if !bytes.Equal(expectedComparable, currentComparable) && !bytes.Equal(reviewedComparable, currentComparable) {
		return fmt.Errorf("archived plan differs from the command-rendered reviewed plan outside the allowed Closeout body near %s", firstComparableDifference(expectedComparable, currentComparable))
	}
	if _, err := os.Lstat(filepath.Join(workdir, filepath.FromSlash(reviewedPlan))); !os.IsNotExist(err) {
		if err != nil {
			return fmt.Errorf("inspect reviewed active plan path: %w", err)
		}
		return fmt.Errorf("reviewed active plan path still exists after archive: %s", reviewedPlan)
	}

	allowed := map[string]bool{reviewedPlan: true, archivedPlan: true}
	if err := validateArchivedSupplements(workdir, chain.CoveredHeadSHA, reviewedPlan, archivedPlan, allowed); err != nil {
		return err
	}
	changed, err := changedPaths(workdir, chain.CoveredHeadSHA)
	if err != nil {
		return err
	}
	for _, path := range changed {
		if !allowed[path] {
			return fmt.Errorf("unreviewed product change after archive; reopen and review the current candidate: %s", path)
		}
	}
	return nil
}

func commandRenderedArchivePlan(content []byte) ([]byte, error) {
	rawFrontmatter, body, err := splitPlanContent(content)
	if err != nil {
		return nil, err
	}
	var frontmatter plan.Frontmatter
	if err := yaml.Unmarshal([]byte(rawFrontmatter), &frontmatter); err != nil {
		return nil, fmt.Errorf("parse reviewed frontmatter: %w", err)
	}
	canonical, err := yaml.Marshal(frontmatter)
	if err != nil {
		return nil, fmt.Errorf("render reviewed frontmatter: %w", err)
	}
	return []byte(fmt.Sprintf("---\n%s---\n\n%s", string(canonical), strings.TrimLeft(string(body), "\n"))), nil
}

func splitPlanContent(content []byte) (string, []byte, error) {
	lines := strings.Split(string(content), "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return "", nil, fmt.Errorf("missing opening frontmatter delimiter")
	}
	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == "---" {
			return strings.Join(lines[1:index], "\n"), []byte(strings.Join(lines[index+1:], "\n")), nil
		}
	}
	return "", nil, fmt.Errorf("missing closing frontmatter delimiter")
}

func firstComparableDifference(left, right []byte) string {
	leftLines := strings.Split(string(left), "\n")
	rightLines := strings.Split(string(right), "\n")
	limit := len(leftLines)
	if len(rightLines) < limit {
		limit = len(rightLines)
	}
	for index := 0; index < limit; index++ {
		if leftLines[index] != rightLines[index] {
			return fmt.Sprintf("line %d (%q -> %q)", index+1, leftLines[index], rightLines[index])
		}
	}
	return fmt.Sprintf("end of document (%d lines -> %d lines)", len(leftLines), len(rightLines))
}

func validateArchivedSupplements(workdir, coveredHead, reviewedPlan, archivedPlan string, allowed map[string]bool) error {
	reviewedDir := repoSlashPath(plan.SupplementsDirForPlanPath(filepath.FromSlash(reviewedPlan)))
	archivedDir := repoSlashPath(plan.SupplementsDirForPlanPath(filepath.FromSlash(archivedPlan)))
	data, err := gitBytes(workdir, "ls-tree", "-r", "-z", coveredHead, "--", reviewedDir)
	if err != nil {
		return fmt.Errorf("inspect reviewed plan supplements: %w", err)
	}
	for _, raw := range bytes.Split(data, []byte{0}) {
		entry := string(raw)
		if entry == "" {
			continue
		}
		metadata, source, ok := strings.Cut(entry, "\t")
		fields := strings.Fields(metadata)
		if !ok || len(fields) < 1 || source == "" {
			return fmt.Errorf("parse reviewed supplement tree entry %q", entry)
		}
		expectedMode := fields[0]
		source = filepath.ToSlash(source)
		rel, err := filepath.Rel(filepath.FromSlash(reviewedDir), filepath.FromSlash(source))
		if err != nil || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
			return fmt.Errorf("reviewed supplement escaped its expected directory: %s", source)
		}
		target := filepath.ToSlash(filepath.Join(filepath.FromSlash(archivedDir), rel))
		baseline, err := gitBytes(workdir, "show", coveredHead+":"+source)
		if err != nil {
			return fmt.Errorf("read reviewed supplement %s: %w", source, err)
		}
		currentMode, current, err := readGitWorktreeFile(filepath.Join(workdir, filepath.FromSlash(target)))
		if err != nil {
			return fmt.Errorf("read archived supplement %s: %w", target, err)
		}
		if currentMode != expectedMode {
			return fmt.Errorf("archived supplement git mode differs from reviewed content: %s (%s -> %s)", target, expectedMode, currentMode)
		}
		if !bytes.Equal(baseline, current) {
			return fmt.Errorf("archived supplement differs from reviewed content: %s", target)
		}
		if _, err := os.Lstat(filepath.Join(workdir, filepath.FromSlash(source))); !os.IsNotExist(err) {
			if err != nil {
				return fmt.Errorf("inspect reviewed supplement path %s: %w", source, err)
			}
			return fmt.Errorf("reviewed supplement still exists after archive: %s", source)
		}
		allowed[source] = true
		allowed[target] = true
	}
	return nil
}

func readGitWorktreeFile(path string) (string, []byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		if err != nil {
			return "", nil, err
		}
		return "120000", []byte(target), nil
	}
	if !info.Mode().IsRegular() {
		return "", nil, fmt.Errorf("unsupported file type %s", info.Mode().Type())
	}
	mode := "100644"
	if info.Mode().Perm()&0o111 != 0 {
		mode = "100755"
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}
	return mode, content, nil
}

func repoRelativePath(workdir, path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("path is empty")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(workdir, filepath.FromSlash(path))
	}
	path = filepath.Clean(path)
	rel, err := filepath.Rel(workdir, path)
	if err != nil || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return "", fmt.Errorf("path is outside the repository")
	}
	return filepath.ToSlash(rel), nil
}

func repoSlashPath(path string) string {
	return filepath.ToSlash(filepath.Clean(path))
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
	var fence markdownFence
	for _, line := range lines {
		if fence.active() {
			if !masking {
				out = append(out, line)
			}
			if fence.closes(line) {
				fence = markdownFence{}
			}
			continue
		}
		if opened, ok := openingMarkdownFence(line); ok {
			fence = opened
			if !masking {
				out = append(out, line)
			}
			continue
		}
		if strings.HasPrefix(line, "## ") {
			name := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			masking = name == allowedCloseoutSection
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

// markdownFence tracks CommonMark-style fenced code blocks closely enough to
// keep heading-shaped code from changing the plan section currently being
// masked. Only a real level-two heading outside a fence can start or stop a
// closeout body.
type markdownFence struct {
	marker byte
	length int
}

func (f markdownFence) active() bool {
	return f.length > 0
}

func (f markdownFence) closes(line string) bool {
	trimmed := trimMarkdownFenceIndent(line)
	if trimmed == "" || trimmed[0] != f.marker {
		return false
	}
	markerLength := countLeadingByte(trimmed, f.marker)
	return markerLength >= f.length && strings.TrimSpace(trimmed[markerLength:]) == ""
}

func openingMarkdownFence(line string) (markdownFence, bool) {
	trimmed := trimMarkdownFenceIndent(line)
	if len(trimmed) < 3 || (trimmed[0] != '`' && trimmed[0] != '~') {
		return markdownFence{}, false
	}
	marker := trimmed[0]
	markerLength := countLeadingByte(trimmed, marker)
	if markerLength < 3 {
		return markdownFence{}, false
	}
	// A backtick fence's info string cannot itself contain a backtick.
	if marker == '`' && strings.Contains(trimmed[markerLength:], "`") {
		return markdownFence{}, false
	}
	return markdownFence{marker: marker, length: markerLength}, true
}

func trimMarkdownFenceIndent(line string) string {
	indent := 0
	for indent < len(line) && indent < 4 && line[indent] == ' ' {
		indent++
	}
	if indent > 3 {
		return ""
	}
	return line[indent:]
}

func countLeadingByte(value string, marker byte) int {
	count := 0
	for count < len(value) && value[count] == marker {
		count++
	}
	return count
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
