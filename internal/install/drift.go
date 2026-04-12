package install

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RepoBootstrapDrift summarizes whether the default repo bootstrap assets for an
// agent appear stale relative to the currently packaged bootstrap assets.
type RepoBootstrapDrift struct {
	ManagedAssetsPresent      bool
	InstructionsStale         bool
	StaleManagedSkillPackages []string
	ExtraManagedSkillPackages []string
}

func (d RepoBootstrapDrift) Stale() bool {
	return d.InstructionsStale || len(d.StaleManagedSkillPackages) > 0 || len(d.ExtraManagedSkillPackages) > 0
}

// InspectRepoBootstrapDrift checks the default repo bootstrap targets for the
// selected agent without mutating any files.
func (s Service) InspectRepoBootstrapDrift(agent string) (RepoBootstrapDrift, error) {
	agent = normalizeAgent(agent)
	instructionsFile, err := s.resolveInstructionsFile(agent, ScopeRepo, "")
	if err != nil {
		return RepoBootstrapDrift{}, err
	}
	skillsDir, err := s.resolveSkillsDir(agent, ScopeRepo, "")
	if err != nil {
		return RepoBootstrapDrift{}, err
	}

	instructionsPresent, instructionsStale, err := s.inspectManagedInstructionsDrift(instructionsFile, skillsDir)
	if err != nil {
		return RepoBootstrapDrift{}, err
	}
	skillPresent, staleSkills, extraSkills, err := s.inspectManagedSkillsDrift(skillsDir)
	if err != nil {
		return RepoBootstrapDrift{}, err
	}

	sort.Strings(staleSkills)
	sort.Strings(extraSkills)
	return RepoBootstrapDrift{
		ManagedAssetsPresent:      instructionsPresent || skillPresent,
		InstructionsStale:         instructionsStale,
		StaleManagedSkillPackages: staleSkills,
		ExtraManagedSkillPackages: extraSkills,
	}, nil
}

func (s Service) inspectManagedInstructionsDrift(targetFile, skillsDir string) (present bool, stale bool, err error) {
	data, err := os.ReadFile(targetFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, false, nil
		}
		return false, false, err
	}

	existing := string(data)
	beginMatches := instructionsManagedBlockBeginPattern.FindAllStringIndex(existing, -1)
	endMatches := instructionsManagedBlockEndPattern.FindAllStringIndex(existing, -1)
	if err := validateManagedBlockLayout(beginMatches, endMatches); err != nil {
		return false, false, err
	}
	if len(beginMatches) == 0 {
		return false, false, nil
	}

	lineEnding := detectLineEnding(existing)
	expected := trimTrailingLineBreaks(renderManagedBlock(lineEnding, pathLabel(s.Workdir, skillsDir), s.versionTag()))
	current := trimTrailingLineBreaks(existing[beginMatches[0][0]:endMatches[0][1]])
	return true, normalizeText(current) != normalizeText(expected), nil
}

func (s Service) inspectManagedSkillsDrift(targetDir string) (present bool, stale []string, extra []string, err error) {
	installed, err := discoverInstalledSkills(targetDir)
	if err != nil {
		return false, nil, nil, err
	}
	canonical, err := s.renderCanonicalSkillFiles()
	if err != nil {
		return false, nil, nil, err
	}

	for skillName, state := range installed {
		if !state.Managed {
			continue
		}
		present = true
		expectedFiles, ok := canonical[skillName]
		if !ok {
			extra = append(extra, skillName)
			continue
		}
		match, err := managedSkillPackageMatches(state.Root, expectedFiles)
		if err != nil {
			return false, nil, nil, err
		}
		if !match {
			stale = append(stale, skillName)
		}
	}

	return present, stale, extra, nil
}

func managedSkillPackageMatches(root string, expectedFiles map[string]string) (bool, error) {
	existingFiles, err := walkFiles(root)
	if err != nil {
		return false, err
	}
	expectedPaths := make(map[string]string, len(expectedFiles))
	for relPath, content := range expectedFiles {
		expectedPaths[filepath.Join(root, filepath.FromSlash(relPath))] = content
	}

	if len(existingFiles) != len(expectedPaths) {
		return false, nil
	}
	for _, existing := range existingFiles {
		expected, ok := expectedPaths[existing]
		if !ok {
			return false, nil
		}
		current, err := os.ReadFile(existing)
		if err != nil {
			return false, err
		}
		if string(current) != expected {
			return false, nil
		}
	}
	return true, nil
}

func normalizeText(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return strings.TrimSpace(content)
}
