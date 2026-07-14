package reviewcoverage

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	baselineComparable, err := planWithMaskedCloseoutBody(baseline)
	if err != nil {
		return fmt.Errorf("read reviewed plan structure: %w", err)
	}
	currentComparable, err := planWithMaskedCloseoutBody(current)
	if err != nil {
		return fmt.Errorf("read current plan structure: %w", err)
	}
	if !bytes.Equal(baselineComparable, currentComparable) {
		return fmt.Errorf("current plan changed outside the allowed closeout summary bodies")
	}
	return nil
}

// ValidateArchivedCandidate binds publish and merge handoff to the reviewed
// candidate. It permits only the mechanical archive move of the reviewed plan
// and its supplements, plus Closeout-body changes that were already allowed at
// the archive boundary.
func ValidateArchivedCandidate(workdir, archivedPlanPath string, chain *Chain) error {
	_, _, allowed, err := validateArchivedCandidateStructure(workdir, archivedPlanPath, chain)
	if err != nil {
		return err
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

// ValidateArchivedCandidateAgainstBase preserves review coverage when the
// current candidate differs from the reviewed candidate only because its base
// advanced. The candidate-owned committed delta must remain byte-for-byte and
// mode-for-mode identical, and an upstream edit to any candidate-owned path
// fails closed even when Git could merge it automatically.
//
// Callers must pass the current base revision explicitly. The ordinary
// ValidateArchivedCandidate remains the strict fallback when no trustworthy
// base revision is available.
func ValidateArchivedCandidateAgainstBase(workdir, archivedPlanPath string, chain *Chain, currentBaseRevision string) error {
	_, _, allowed, err := validateArchivedCandidateStructure(workdir, archivedPlanPath, chain)
	if err != nil {
		return err
	}
	candidate, err := CaptureCandidate(workdir)
	if err != nil {
		return fmt.Errorf("validate archived candidate worktree: %w", err)
	}
	currentBase, err := ResolveCommit(workdir, currentBaseRevision)
	if err != nil {
		return fmt.Errorf("resolve current candidate base %q: %w", currentBaseRevision, err)
	}
	baseIncluded, err := IsAncestor(workdir, currentBase, candidate.HeadSHA)
	if err != nil {
		return fmt.Errorf("validate current candidate base ancestry: %w", err)
	}
	if !baseIncluded {
		return fmt.Errorf("current candidate HEAD does not contain base %s; refresh or repair the candidate before preserving review coverage", currentBase)
	}
	reviewedBase, err := gitOutput(workdir, "merge-base", chain.CoveredHeadSHA, currentBase)
	if err != nil {
		return fmt.Errorf("resolve reviewed candidate base: %w", err)
	}
	if strings.TrimSpace(reviewedBase) == "" {
		return fmt.Errorf("reviewed candidate and current base have no merge base")
	}

	reviewedPaths, err := changedPathsBetween(workdir, reviewedBase, chain.CoveredHeadSHA)
	if err != nil {
		return fmt.Errorf("inspect reviewed candidate delta: %w", err)
	}
	upstreamPaths, err := changedPathsBetween(workdir, reviewedBase, currentBase)
	if err != nil {
		return fmt.Errorf("inspect upstream base delta: %w", err)
	}
	upstream := make(map[string]bool, len(upstreamPaths))
	for _, path := range upstreamPaths {
		upstream[path] = true
	}
	for _, path := range reviewedPaths {
		if upstream[path] {
			return fmt.Errorf("upstream base synchronization overlaps reviewed candidate path %s; reopen and review the synchronized candidate", path)
		}
	}

	reviewedDelta, err := candidateDelta(workdir, reviewedBase, chain.CoveredHeadSHA, allowed)
	if err != nil {
		return fmt.Errorf("capture reviewed candidate delta: %w", err)
	}
	currentDelta, err := candidateDelta(workdir, currentBase, candidate.HeadSHA, allowed)
	if err != nil {
		return fmt.Errorf("capture current candidate delta: %w", err)
	}
	if path := firstDeltaDifference(reviewedDelta, currentDelta); path != "" {
		return fmt.Errorf("candidate-owned delta changed after review; reopen and review the synchronized candidate: %s", path)
	}
	return nil
}

// ValidatePublishedCandidate validates the immutable candidate commit recorded
// by publish evidence without depending on the caller's current checkout. This
// is the land-time counterpart to ValidateArchivedCandidate: after a squash or
// rebase merge the worktree may be on a landed main commit that cannot descend
// from the reviewed feature commit, while the published candidate itself must
// still be the reviewed candidate plus the mechanical archive move.
func ValidatePublishedCandidate(workdir, archivedPlanPath string, chain *Chain, publishedRevision string) error {
	publishedHead, allowed, err := validatePublishedCandidateStructure(workdir, archivedPlanPath, chain, publishedRevision)
	if err != nil {
		return err
	}
	ancestor, err := IsAncestor(workdir, chain.CoveredHeadSHA, publishedHead)
	if err != nil {
		return fmt.Errorf("validate published candidate ancestry: %w", err)
	}
	if !ancestor {
		return fmt.Errorf("published candidate %s is not descended from reviewed head %s", publishedHead, chain.CoveredHeadSHA)
	}
	changed, err := changedPathsBetween(workdir, chain.CoveredHeadSHA, publishedHead)
	if err != nil {
		return fmt.Errorf("inspect published candidate changes: %w", err)
	}
	for _, path := range changed {
		if !allowed[path] {
			return fmt.Errorf("unreviewed product change in published candidate: %s", path)
		}
	}
	return nil
}

// ValidatePublishedCandidateAgainstBase validates a published candidate whose
// commits were rewritten by an otherwise review-preserving base sync. The
// immutable base and head commits come from the same fresh sync observation;
// the reviewed and published candidate-owned deltas must remain identical.
func ValidatePublishedCandidateAgainstBase(workdir, archivedPlanPath string, chain *Chain, publishedRevision, baseRevision string) error {
	publishedHead, allowed, err := validatePublishedCandidateStructure(workdir, archivedPlanPath, chain, publishedRevision)
	if err != nil {
		return err
	}
	currentBase, err := ResolveCommit(workdir, baseRevision)
	if err != nil {
		return fmt.Errorf("resolve published candidate base %q: %w", baseRevision, err)
	}
	baseIncluded, err := IsAncestor(workdir, currentBase, publishedHead)
	if err != nil {
		return fmt.Errorf("validate published candidate base ancestry: %w", err)
	}
	if !baseIncluded {
		return fmt.Errorf("published candidate %s does not contain recorded base %s", publishedHead, currentBase)
	}
	reviewedBase, err := gitOutput(workdir, "merge-base", chain.CoveredHeadSHA, currentBase)
	if err != nil {
		return fmt.Errorf("resolve reviewed candidate base: %w", err)
	}
	reviewedPaths, err := changedPathsBetween(workdir, reviewedBase, chain.CoveredHeadSHA)
	if err != nil {
		return fmt.Errorf("inspect reviewed candidate delta: %w", err)
	}
	upstreamPaths, err := changedPathsBetween(workdir, reviewedBase, currentBase)
	if err != nil {
		return fmt.Errorf("inspect recorded base delta: %w", err)
	}
	upstream := make(map[string]bool, len(upstreamPaths))
	for _, path := range upstreamPaths {
		upstream[path] = true
	}
	for _, path := range reviewedPaths {
		if upstream[path] {
			return fmt.Errorf("recorded base overlaps reviewed candidate path %s; published candidate requires fresh review", path)
		}
	}
	reviewedDelta, err := candidateDelta(workdir, reviewedBase, chain.CoveredHeadSHA, allowed)
	if err != nil {
		return fmt.Errorf("capture reviewed candidate delta: %w", err)
	}
	publishedDelta, err := candidateDelta(workdir, currentBase, publishedHead, allowed)
	if err != nil {
		return fmt.Errorf("capture published candidate delta: %w", err)
	}
	if path := firstDeltaDifference(reviewedDelta, publishedDelta); path != "" {
		return fmt.Errorf("published candidate-owned delta changed after review: %s", path)
	}
	return nil
}

func validatePublishedCandidateStructure(workdir, archivedPlanPath string, chain *Chain, publishedRevision string) (string, map[string]bool, error) {
	if chain == nil || strings.TrimSpace(chain.CoveredHeadSHA) == "" {
		return "", nil, fmt.Errorf("published candidate has no reviewed head coverage")
	}
	reviewedPlan, err := repoRelativePath(workdir, chain.ReviewedPlanPath)
	if err != nil {
		return "", nil, fmt.Errorf("resolve reviewed plan path: %w", err)
	}
	archivedPlan, err := repoRelativePath(workdir, archivedPlanPath)
	if err != nil {
		return "", nil, fmt.Errorf("resolve archived plan path: %w", err)
	}
	publishedHead, err := ResolveCommit(workdir, publishedRevision)
	if err != nil {
		return "", nil, fmt.Errorf("resolve published candidate %q: %w", publishedRevision, err)
	}

	baseline, err := gitBytes(workdir, "show", chain.CoveredHeadSHA+":"+reviewedPlan)
	if err != nil {
		return "", nil, fmt.Errorf("read reviewed plan content: %w", err)
	}
	currentMode, current, err := readGitRevisionFile(workdir, publishedHead, archivedPlan)
	if err != nil {
		return "", nil, fmt.Errorf("read published archived plan content: %w", err)
	}
	if currentMode != "100644" {
		return "", nil, fmt.Errorf("published archived plan has git mode %s, expected command-rendered regular file mode 100644", currentMode)
	}
	expected, err := commandRenderedArchivePlan(baseline)
	if err != nil {
		return "", nil, fmt.Errorf("prepare reviewed plan for archive comparison: %w", err)
	}
	expectedComparable, err := planWithMaskedCloseoutBody(expected)
	if err != nil {
		return "", nil, fmt.Errorf("read command-rendered plan structure: %w", err)
	}
	reviewedComparable, err := planWithMaskedCloseoutBody(baseline)
	if err != nil {
		return "", nil, fmt.Errorf("read reviewed plan structure: %w", err)
	}
	currentComparable, err := planWithMaskedCloseoutBody(current)
	if err != nil {
		return "", nil, fmt.Errorf("read published archived plan structure: %w", err)
	}
	if !bytes.Equal(expectedComparable, currentComparable) && !bytes.Equal(reviewedComparable, currentComparable) {
		return "", nil, fmt.Errorf("published archived plan differs from the command-rendered reviewed plan outside the allowed Closeout body near %s", firstComparableDifference(expectedComparable, currentComparable))
	}
	if entry, err := treeEntry(workdir, publishedHead, reviewedPlan); err != nil {
		return "", nil, err
	} else if entry != "" {
		return "", nil, fmt.Errorf("reviewed active plan path still exists in published candidate: %s", reviewedPlan)
	}

	allowed := map[string]bool{reviewedPlan: true, archivedPlan: true}
	if err := validatePublishedSupplements(workdir, chain.CoveredHeadSHA, publishedHead, reviewedPlan, archivedPlan, allowed); err != nil {
		return "", nil, err
	}
	return publishedHead, allowed, nil
}

func validateArchivedCandidateStructure(workdir, archivedPlanPath string, chain *Chain) (string, string, map[string]bool, error) {
	if chain == nil || strings.TrimSpace(chain.CoveredHeadSHA) == "" {
		return "", "", nil, fmt.Errorf("archived candidate has no reviewed head coverage")
	}
	reviewedPlan, err := repoRelativePath(workdir, chain.ReviewedPlanPath)
	if err != nil {
		return "", "", nil, fmt.Errorf("resolve reviewed plan path: %w", err)
	}
	archivedPlan, err := repoRelativePath(workdir, archivedPlanPath)
	if err != nil {
		return "", "", nil, fmt.Errorf("resolve archived plan path: %w", err)
	}
	if reviewedPlan == archivedPlan {
		return "", "", nil, fmt.Errorf("archived plan path still equals reviewed active plan path")
	}

	baseline, err := gitBytes(workdir, "show", chain.CoveredHeadSHA+":"+reviewedPlan)
	if err != nil {
		return "", "", nil, fmt.Errorf("read reviewed plan content: %w", err)
	}
	archivedPlanFSPath := filepath.Join(workdir, filepath.FromSlash(archivedPlan))
	currentMode, current, err := readGitWorktreeFile(archivedPlanFSPath)
	if err != nil {
		return "", "", nil, fmt.Errorf("read archived plan content: %w", err)
	}
	if currentMode != "100644" {
		return "", "", nil, fmt.Errorf("archived plan has git mode %s, expected command-rendered regular file mode 100644", currentMode)
	}
	expected, err := commandRenderedArchivePlan(baseline)
	if err != nil {
		return "", "", nil, fmt.Errorf("prepare reviewed plan for archive comparison: %w", err)
	}
	expectedComparable, err := planWithMaskedCloseoutBody(expected)
	if err != nil {
		return "", "", nil, fmt.Errorf("read command-rendered plan structure: %w", err)
	}
	reviewedComparable, err := planWithMaskedCloseoutBody(baseline)
	if err != nil {
		return "", "", nil, fmt.Errorf("read reviewed plan structure: %w", err)
	}
	currentComparable, err := planWithMaskedCloseoutBody(current)
	if err != nil {
		return "", "", nil, fmt.Errorf("read archived plan structure: %w", err)
	}
	if !bytes.Equal(expectedComparable, currentComparable) && !bytes.Equal(reviewedComparable, currentComparable) {
		return "", "", nil, fmt.Errorf("archived plan differs from the command-rendered reviewed plan outside the allowed Closeout body near %s", firstComparableDifference(expectedComparable, currentComparable))
	}
	if _, err := os.Lstat(filepath.Join(workdir, filepath.FromSlash(reviewedPlan))); !os.IsNotExist(err) {
		if err != nil {
			return "", "", nil, fmt.Errorf("inspect reviewed active plan path: %w", err)
		}
		return "", "", nil, fmt.Errorf("reviewed active plan path still exists after archive: %s", reviewedPlan)
	}

	allowed := map[string]bool{reviewedPlan: true, archivedPlan: true}
	if err := validateArchivedSupplements(workdir, chain.CoveredHeadSHA, reviewedPlan, archivedPlan, allowed); err != nil {
		return "", "", nil, err
	}
	return reviewedPlan, archivedPlan, allowed, nil
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

func planWithMaskedCloseoutBody(content []byte) ([]byte, error) {
	envelope, body, err := splitPlanEnvelope(content)
	if err != nil {
		return nil, err
	}
	comparable := make([]byte, 0, len(envelope)+len(body))
	comparable = append(comparable, envelope...)
	comparable = append(comparable, maskCloseoutBodies(body)...)
	return comparable, nil
}

func splitPlanEnvelope(content []byte) ([]byte, []byte, error) {
	lineStart := 0
	lineNumber := 0
	for lineStart <= len(content) {
		lineEnd := len(content)
		if relative := bytes.IndexByte(content[lineStart:], '\n'); relative >= 0 {
			lineEnd = lineStart + relative
		}
		line := content[lineStart:lineEnd]
		if lineNumber == 0 && strings.TrimSpace(string(line)) != "---" {
			return nil, nil, fmt.Errorf("missing opening frontmatter delimiter")
		}
		if lineNumber > 0 && strings.TrimSpace(string(line)) == "---" {
			bodyStart := lineEnd
			if bodyStart < len(content) && content[bodyStart] == '\n' {
				bodyStart++
			}
			return content[:bodyStart], content[bodyStart:], nil
		}
		if lineEnd == len(content) {
			break
		}
		lineStart = lineEnd + 1
		lineNumber++
	}
	return nil, nil, fmt.Errorf("missing closing frontmatter delimiter")
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

func validatePublishedSupplements(workdir, coveredHead, publishedHead, reviewedPlan, archivedPlan string, allowed map[string]bool) error {
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
		currentMode, current, err := readGitRevisionFile(workdir, publishedHead, target)
		if err != nil {
			return fmt.Errorf("read published archived supplement %s: %w", target, err)
		}
		if currentMode != expectedMode {
			return fmt.Errorf("published archived supplement git mode differs from reviewed content: %s (%s -> %s)", target, expectedMode, currentMode)
		}
		if !bytes.Equal(baseline, current) {
			return fmt.Errorf("published archived supplement differs from reviewed content: %s", target)
		}
		if sourceEntry, err := treeEntry(workdir, publishedHead, source); err != nil {
			return err
		} else if sourceEntry != "" {
			return fmt.Errorf("reviewed supplement still exists in published candidate: %s", source)
		}
		allowed[source] = true
		allowed[target] = true
	}
	return nil
}

func readGitRevisionFile(workdir, revision, path string) (string, []byte, error) {
	entry, err := treeEntry(workdir, revision, path)
	if err != nil {
		return "", nil, err
	}
	if entry == "" {
		return "", nil, os.ErrNotExist
	}
	fields := strings.Fields(entry)
	if len(fields) < 3 {
		return "", nil, fmt.Errorf("parse tree entry for %s at %s", path, revision)
	}
	content, err := gitBytes(workdir, "show", revision+":"+path)
	if err != nil {
		return "", nil, err
	}
	return fields[0], content, nil
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

func changedPathsBetween(workdir, from, to string) ([]string, error) {
	data, err := gitBytes(workdir, "diff", "--name-status", "-z", "--no-renames", from, to, "--")
	if err != nil {
		return nil, err
	}
	parts := bytes.Split(data, []byte{0})
	paths := make([]string, 0, len(parts)/2)
	for i := 0; i+1 < len(parts); i += 2 {
		path := filepath.ToSlash(string(parts[i+1]))
		if path != "" {
			paths = append(paths, path)
		}
	}
	return uniqueStrings(paths), nil
}

type deltaEntry struct {
	Before string
	After  string
}

func candidateDelta(workdir, from, to string, excluded map[string]bool) (map[string]deltaEntry, error) {
	paths, err := changedPathsBetween(workdir, from, to)
	if err != nil {
		return nil, err
	}
	delta := make(map[string]deltaEntry, len(paths))
	for _, path := range paths {
		if excluded[path] {
			continue
		}
		before, err := treeEntry(workdir, from, path)
		if err != nil {
			return nil, err
		}
		after, err := treeEntry(workdir, to, path)
		if err != nil {
			return nil, err
		}
		delta[path] = deltaEntry{Before: before, After: after}
	}
	return delta, nil
}

func treeEntry(workdir, revision, path string) (string, error) {
	data, err := gitBytes(workdir, "ls-tree", "-z", revision, "--", ":(literal)"+path)
	if err != nil {
		return "", fmt.Errorf("inspect %s at %s: %w", path, revision, err)
	}
	entry := strings.TrimSuffix(string(data), "\x00")
	if entry == "" {
		return "", nil
	}
	metadata, _, ok := strings.Cut(entry, "\t")
	if !ok {
		return "", fmt.Errorf("parse tree entry for %s at %s", path, revision)
	}
	return metadata, nil
}

func firstDeltaDifference(reviewed, current map[string]deltaEntry) string {
	paths := make([]string, 0, len(reviewed)+len(current))
	for path := range reviewed {
		paths = append(paths, path)
	}
	for path := range current {
		if _, ok := reviewed[path]; !ok {
			paths = append(paths, path)
		}
	}
	paths = uniqueStrings(paths)
	sort.Strings(paths)
	for _, path := range paths {
		if reviewed[path] != current[path] {
			return path
		}
	}
	return ""
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
