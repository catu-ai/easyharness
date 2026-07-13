package reviewcoverage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/runstate"
)

type Round struct {
	Manifest  contracts.ReviewManifest
	Aggregate contracts.ReviewAggregate
}

type Chain struct {
	RootRoundID             string
	TipRoundID              string
	CoveredHeadSHA          string
	ReviewedPlanPath        string
	Revision                int
	Decision                string
	UnresolvedBlockingCount int
	Rounds                  []string
}

func LoadRound(workdir, planStem, roundID string) (*Round, error) {
	dir := runstate.ReviewRoundDir(workdir, planStem, strings.TrimSpace(roundID))
	manifestData, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("read review manifest for %s: %w", roundID, err)
	}
	var manifest contracts.ReviewManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("parse review manifest for %s: %w", roundID, err)
	}
	aggregateData, err := os.ReadFile(filepath.Join(dir, "aggregate.json"))
	if err != nil {
		return nil, fmt.Errorf("read review aggregate for %s: %w", roundID, err)
	}
	var aggregate contracts.ReviewAggregate
	if err := json.Unmarshal(aggregateData, &aggregate); err != nil {
		return nil, fmt.Errorf("parse review aggregate for %s: %w", roundID, err)
	}
	if manifest.RoundID != roundID || aggregate.RoundID != roundID {
		return nil, fmt.Errorf("review round %s has inconsistent artifact identities", roundID)
	}
	return &Round{Manifest: manifest, Aggregate: aggregate}, nil
}

// Resolve validates the continuous finalize chain ending at tipRoundID.
func Resolve(workdir, planStem, tipRoundID string, currentRevision int) (*Chain, error) {
	tipRoundID = strings.TrimSpace(tipRoundID)
	if tipRoundID == "" {
		return nil, fmt.Errorf("finalize review coverage has no tip round")
	}
	seen := map[string]bool{}
	reverse := make([]string, 0)
	currentID := tipRoundID
	var tip *Round
	var child *Round
	for currentID != "" {
		if seen[currentID] {
			return nil, fmt.Errorf("finalize review coverage contains a cycle at %s", currentID)
		}
		seen[currentID] = true
		round, err := LoadRound(workdir, planStem, currentID)
		if err != nil {
			return nil, err
		}
		if round.Manifest.Step != nil || round.Aggregate.Step != nil {
			return nil, fmt.Errorf("review round %s is step-bound and cannot provide finalize coverage", currentID)
		}
		if err := validateRoundAggregate(round); err != nil {
			return nil, fmt.Errorf("review round %s: %w", currentID, err)
		}
		if strings.TrimSpace(round.Manifest.ReviewedHeadSHA) == "" || round.Manifest.ReviewedHeadSHA != round.Aggregate.ReviewedHeadSHA {
			return nil, fmt.Errorf("review round %s does not have one consistent reviewed head", currentID)
		}
		if tip == nil {
			tip = round
		}
		if child != nil {
			if child.Manifest.Repair == nil || strings.TrimSpace(child.Manifest.Repair.RoundID) != currentID {
				return nil, fmt.Errorf("review round %s does not directly link to parent %s", child.Manifest.RoundID, currentID)
			}
			if child.Manifest.AnchorSHA != round.Manifest.ReviewedHeadSHA {
				return nil, fmt.Errorf("review round %s anchor does not equal parent %s reviewed head", child.Manifest.RoundID, currentID)
			}
			ancestor, err := IsAncestor(workdir, round.Manifest.ReviewedHeadSHA, child.Manifest.ReviewedHeadSHA)
			if err != nil {
				return nil, fmt.Errorf("validate review ancestry: %w", err)
			}
			if !ancestor {
				return nil, fmt.Errorf("review round %s reviewed head is not descended from parent %s", child.Manifest.RoundID, currentID)
			}
			if child.Manifest.Revision < round.Manifest.Revision || child.Manifest.Revision > round.Manifest.Revision+1 {
				return nil, fmt.Errorf("review round %s revision does not continuously extend parent %s", child.Manifest.RoundID, currentID)
			}
			if child.Manifest.Revision == round.Manifest.Revision+1 && round.Aggregate.Decision != "pass" {
				return nil, fmt.Errorf("review round %s cannot extend unresolved prior revision %s", child.Manifest.RoundID, currentID)
			}
			if err := validateRepairCoverage(round, child); err != nil {
				return nil, fmt.Errorf("review round %s: %w", child.Manifest.RoundID, err)
			}
		}
		reverse = append(reverse, currentID)
		if round.Manifest.Kind == "full" {
			if round.Manifest.Repair != nil {
				return nil, fmt.Errorf("full review round %s cannot link to a repair parent", currentID)
			}
			if len(round.Aggregate.ResolvedFindingIDs) > 0 {
				return nil, fmt.Errorf("full review root %s cannot claim inherited finding resolutions", currentID)
			}
			if !sameStringSet(findingIDs(round.Aggregate.BlockingFindings), round.Aggregate.UnresolvedFindingIDs) {
				return nil, fmt.Errorf("full review root %s cumulative unresolved findings do not match its blocking findings", currentID)
			}
			blockingByID := findingMap(round.Aggregate.BlockingFindings)
			for _, unresolved := range round.Aggregate.UnresolvedBlockingFindings {
				if blocking, ok := blockingByID[unresolved.FindingID]; !ok || !sameFinding(blocking, unresolved) {
					return nil, fmt.Errorf("full review root %s cumulative finding %s does not preserve its blocking finding data", currentID, unresolved.FindingID)
				}
			}
			break
		}
		if round.Manifest.Kind != "delta" || round.Manifest.Repair == nil {
			return nil, fmt.Errorf("finalize coverage round %s is not a full root or linked repair delta", currentID)
		}
		child = round
		currentID = strings.TrimSpace(round.Manifest.Repair.RoundID)
	}
	if tip == nil {
		return nil, fmt.Errorf("finalize review coverage is empty")
	}
	if tip.Manifest.Revision != currentRevision {
		return nil, fmt.Errorf("coverage tip revision %d does not match current revision %d", tip.Manifest.Revision, currentRevision)
	}
	for left, right := 0, len(reverse)-1; left < right; left, right = left+1, right-1 {
		reverse[left], reverse[right] = reverse[right], reverse[left]
	}
	return &Chain{
		RootRoundID:             reverse[0],
		TipRoundID:              tip.Manifest.RoundID,
		CoveredHeadSHA:          tip.Manifest.ReviewedHeadSHA,
		ReviewedPlanPath:        tip.Manifest.PlanPath,
		Revision:                tip.Manifest.Revision,
		Decision:                tip.Aggregate.Decision,
		UnresolvedBlockingCount: len(tip.Aggregate.UnresolvedFindingIDs),
		Rounds:                  reverse,
	}, nil
}

func validateRoundAggregate(round *Round) error {
	if round.Manifest.Kind != round.Aggregate.Kind || round.Manifest.Revision != round.Aggregate.Revision {
		return fmt.Errorf("manifest and aggregate metadata disagree")
	}
	if !sameOptionalInt(round.Manifest.Step, round.Aggregate.Step) {
		return fmt.Errorf("manifest and aggregate step bindings disagree")
	}
	if !sameRepair(round.Manifest.Repair, round.Aggregate.Repair) {
		return fmt.Errorf("manifest and aggregate repair links disagree")
	}
	if round.Aggregate.Repair != nil {
		if duplicate := firstDuplicate(round.Aggregate.Repair.FindingIDs); duplicate != "" {
			return fmt.Errorf("aggregate repair link contains duplicate finding id %s", duplicate)
		}
	}
	if (len(round.Aggregate.UnresolvedFindingIDs) == 0) != (round.Aggregate.Decision == "pass") {
		return fmt.Errorf("aggregate decision is inconsistent with cumulative unresolved findings")
	}
	if round.Aggregate.Decision != "pass" && round.Aggregate.Decision != "changes_requested" {
		return fmt.Errorf("aggregate decision %q is unsupported", round.Aggregate.Decision)
	}
	if !sameStringSet(findingIDs(round.Aggregate.UnresolvedBlockingFindings), round.Aggregate.UnresolvedFindingIDs) {
		return fmt.Errorf("cumulative unresolved finding ids and objects disagree")
	}
	if err := validateFindingSet(round.Aggregate.BlockingFindings, true, "blocking findings"); err != nil {
		return err
	}
	if err := validateFindingSet(round.Aggregate.NonBlockingFindings, false, "non-blocking findings"); err != nil {
		return err
	}
	if err := validateFindingSet(round.Aggregate.UnresolvedBlockingFindings, true, "cumulative unresolved findings"); err != nil {
		return err
	}
	if duplicate := firstDuplicate(append(findingIDs(round.Aggregate.BlockingFindings), findingIDs(round.Aggregate.NonBlockingFindings)...)); duplicate != "" {
		return fmt.Errorf("finding id %s appears more than once in this round", duplicate)
	}
	if duplicate := firstDuplicate(round.Aggregate.ResolvedFindingIDs); duplicate != "" {
		return fmt.Errorf("resolved finding id %s appears more than once", duplicate)
	}
	if duplicate := firstDuplicate(round.Aggregate.UnresolvedFindingIDs); duplicate != "" {
		return fmt.Errorf("unresolved finding id %s appears more than once", duplicate)
	}
	resolved := stringSet(round.Aggregate.ResolvedFindingIDs)
	for _, findingID := range round.Aggregate.UnresolvedFindingIDs {
		if resolved[findingID] {
			return fmt.Errorf("finding %s is both resolved and unresolved", findingID)
		}
	}
	for _, finding := range round.Aggregate.UnresolvedBlockingFindings {
		if finding.Severity != "blocker" && finding.Severity != "important" {
			return fmt.Errorf("non-blocking finding %s appears in cumulative blocking set", finding.FindingID)
		}
	}
	return nil
}

func sameOptionalInt(left, right *int) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func validateRepairCoverage(parent, child *Round) error {
	if duplicate := firstDuplicate(child.Manifest.Repair.FindingIDs); duplicate != "" {
		return fmt.Errorf("repair reference contains duplicate finding id %s", duplicate)
	}
	targeted := stringSet(child.Manifest.Repair.FindingIDs)
	knownParentFindings := stringSet(append(
		append(findingIDs(parent.Aggregate.BlockingFindings), findingIDs(parent.Aggregate.NonBlockingFindings)...),
		findingIDs(parent.Aggregate.UnresolvedBlockingFindings)...,
	))
	for findingID := range targeted {
		if findingID == "" || !knownParentFindings[findingID] {
			return fmt.Errorf("repair reference targets finding %q that is absent from parent %s", findingID, parent.Manifest.RoundID)
		}
	}
	for _, findingID := range parent.Aggregate.UnresolvedFindingIDs {
		if !targeted[findingID] {
			return fmt.Errorf("repair reference does not cover parent unresolved finding %s", findingID)
		}
	}
	resolved := stringSet(child.Aggregate.ResolvedFindingIDs)
	for findingID := range resolved {
		if !targeted[findingID] {
			return fmt.Errorf("resolved finding %s was not targeted by the repair reference", findingID)
		}
	}
	want := findingMap(parent.Aggregate.UnresolvedBlockingFindings)
	for findingID := range resolved {
		delete(want, findingID)
	}
	for _, finding := range child.Aggregate.BlockingFindings {
		if _, inherited := want[finding.FindingID]; inherited {
			return fmt.Errorf("new blocking finding %s collides with inherited parent debt", finding.FindingID)
		}
		want[finding.FindingID] = finding
	}
	if !sameStringSet(findingMapKeys(want), child.Aggregate.UnresolvedFindingIDs) {
		return fmt.Errorf("cumulative unresolved findings do not equal parent debt minus resolutions plus new blocking findings")
	}
	got := findingMap(child.Aggregate.UnresolvedBlockingFindings)
	for findingID, finding := range want {
		if current, ok := got[findingID]; !ok || !sameFinding(current, finding) {
			return fmt.Errorf("cumulative unresolved finding %s does not preserve its validated finding data", findingID)
		}
	}
	return nil
}

func validateFindingSet(findings []contracts.ReviewAggregateFinding, blocking bool, label string) error {
	seen := map[string]bool{}
	for _, finding := range findings {
		findingID := strings.TrimSpace(finding.FindingID)
		if findingID == "" {
			return fmt.Errorf("%s contain an empty finding id", label)
		}
		if seen[findingID] {
			return fmt.Errorf("%s contain duplicate finding id %s", label, findingID)
		}
		seen[findingID] = true
		isBlocking := finding.Severity == "blocker" || finding.Severity == "important"
		if blocking != isBlocking {
			return fmt.Errorf("%s contain finding %s with inconsistent severity %q", label, findingID, finding.Severity)
		}
	}
	return nil
}

func firstDuplicate(values []string) string {
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return "<empty>"
		}
		if seen[value] {
			return value
		}
		seen[value] = true
	}
	return ""
}

func findingMap(findings []contracts.ReviewAggregateFinding) map[string]contracts.ReviewAggregateFinding {
	out := make(map[string]contracts.ReviewAggregateFinding, len(findings))
	for _, finding := range findings {
		out[finding.FindingID] = finding
	}
	return out
}

func findingMapKeys(values map[string]contracts.ReviewAggregateFinding) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	return out
}

func sameFinding(left, right contracts.ReviewAggregateFinding) bool {
	if left.FindingID != right.FindingID || left.Slot != right.Slot || left.Role != right.Role ||
		left.Area != right.Area || left.Severity != right.Severity || left.Title != right.Title ||
		left.Details != right.Details || left.HasLocations != right.HasLocations ||
		len(left.Locations) != len(right.Locations) {
		return false
	}
	for index := range left.Locations {
		if left.Locations[index] != right.Locations[index] {
			return false
		}
	}
	return true
}

func sameRepair(left, right *contracts.ReviewRepairReference) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return strings.TrimSpace(left.RoundID) == strings.TrimSpace(right.RoundID) && sameStringSet(left.FindingIDs, right.FindingIDs)
}

func findingIDs(findings []contracts.ReviewAggregateFinding) []string {
	out := make([]string, 0, len(findings))
	for _, finding := range findings {
		out = append(out, finding.FindingID)
	}
	return out
}

func stringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[strings.TrimSpace(value)] = true
	}
	return out
}

func sameStringSet(left, right []string) bool {
	leftSet := stringSet(left)
	rightSet := stringSet(right)
	if len(leftSet) != len(rightSet) {
		return false
	}
	for value := range leftSet {
		if !rightSet[value] {
			return false
		}
	}
	return true
}

func StateFromChain(chain *Chain) *runstate.FinalizeCoverage {
	if chain == nil {
		return nil
	}
	return &runstate.FinalizeCoverage{
		RootRoundID:             chain.RootRoundID,
		TipRoundID:              chain.TipRoundID,
		CoveredHeadSHA:          chain.CoveredHeadSHA,
		Revision:                chain.Revision,
		UnresolvedBlockingCount: chain.UnresolvedBlockingCount,
	}
}
