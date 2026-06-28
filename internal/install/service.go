package install

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	bootstrapassets "github.com/catu-ai/easyharness/assets/bootstrap"
	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/repoconfig"
	versioninfo "github.com/catu-ai/easyharness/internal/version"
	"gopkg.in/yaml.v3"
)

const (
	ScopeRepo = "repo"
	ScopeUser = "user"

	ResourceBootstrap    = "bootstrap"
	ResourceSkills       = "skills"
	ResourceInstructions = "instructions"
	ResourceConfig       = "config"

	OperationInstall   = "install"
	OperationUninstall = "uninstall"
	OperationRefresh   = "refresh"

	ActionCreate = "create"
	ActionUpdate = "update"
	ActionDelete = "delete"
	ActionNoop   = "noop"

	defaultAgent = "codex"

	instructionsManagedBlockEnd = "<!-- easyharness:end -->"

	skillMetadataManaged = "easyharness-managed"
	skillMetadataVersion = "easyharness-version"
)

var (
	instructionsManagedBlockBeginPattern = regexp.MustCompile(`(?m)^<!-- easyharness:begin(?: version="[^"]*")? -->[ \t]*\r?$`)
	instructionsManagedBlockEndPattern   = regexp.MustCompile(`(?m)^<!-- easyharness:end -->[ \t]*\r?$`)
)

type Service struct {
	Workdir     string
	Version     versioninfo.Info
	LookupEnv   func(string) (string, bool)
	UserHomeDir func() (string, error)
}

type Options struct {
	Scope            string
	Agent            string
	SkillsDir        string
	InstructionsFile string
	DryRun           bool
}

type Result = contracts.BootstrapResult
type Action = contracts.BootstrapAction
type NextAction = contracts.NextAction
type CommandError = contracts.ErrorDetail

type plannedWrite struct {
	path    string
	kind    string
	details string
	content []byte
	pruneTo string
}

func (s Service) Init(opts Options) Result {
	resolvedScope := normalizeScope(opts.Scope)
	if resolvedScope == "" {
		resolvedScope = ScopeRepo
	}
	if resolvedScope != ScopeRepo {
		return s.errorResult("repo init", ResourceBootstrap, OperationInstall, resolvedScope, normalizeAgent(opts.Agent), opts.DryRun, "Unsupported init scope.", []CommandError{{
			Path:    "scope",
			Message: fmt.Sprintf("supported init scope is %q", ScopeRepo),
		}})
	}

	agent := normalizeAgent(opts.Agent)
	instructionsFile, err := s.resolveInstructionsFile(agent, ScopeRepo, opts.InstructionsFile)
	if err != nil {
		return s.errorResult("repo init", ResourceBootstrap, OperationInstall, resolvedScope, agent, opts.DryRun, "Unable to resolve init targets.", []CommandError{{Path: "instructions.file", Message: err.Error()}})
	}
	skillsDir, err := s.resolveSkillsDir(agent, ScopeRepo, opts.SkillsDir)
	if err != nil {
		return s.errorResult("repo init", ResourceBootstrap, OperationInstall, resolvedScope, agent, opts.DryRun, "Unable to resolve init targets.", []CommandError{{Path: "skills.dir", Message: err.Error()}})
	}

	writes, errs := s.planInstructionsInstall(instructionsFile, skillsDir)
	skillWrites, skillErrs := s.planSkillsInstall(skillsDir)
	writes = append(writes, skillWrites...)
	errs = append(errs, skillErrs...)
	configWrites, configWarnings, configErrs := s.planRepoConfigInit()
	writes = append(writes, configWrites...)
	errs = append(errs, configErrs...)
	if len(errs) > 0 {
		return s.errorResult("repo init", ResourceBootstrap, OperationInstall, resolvedScope, agent, opts.DryRun, "Unable to prepare the bootstrap targets.", errs)
	}
	if err := s.applyWrites(writes, opts.DryRun); err != nil {
		return s.errorResult("repo init", ResourceBootstrap, OperationInstall, resolvedScope, agent, opts.DryRun, "Unable to write the bootstrap targets.", []CommandError{*err})
	}

	return s.successResultWithWarnings("repo init", ResourceBootstrap, OperationInstall, resolvedScope, agent, opts.DryRun, writes, configWarnings)
}

func (s Service) InstallSkills(opts Options) Result {
	return s.runSkillCommand("repo skills install", OperationInstall, opts)
}

func (s Service) UninstallSkills(opts Options) Result {
	return s.runSkillCommand("repo skills uninstall", OperationUninstall, opts)
}

func (s Service) InstallInstructions(opts Options) Result {
	return s.runInstructionsCommand("repo instructions install", OperationInstall, opts)
}

func (s Service) UninstallInstructions(opts Options) Result {
	return s.runInstructionsCommand("repo instructions uninstall", OperationUninstall, opts)
}

func (s Service) InitConfig(opts Options) Result {
	scope := ScopeRepo
	agent := normalizeAgent(opts.Agent)
	writes, warnings, errs := s.planRepoConfigInit()
	if len(errs) > 0 {
		return s.errorResult("repo config init", ResourceConfig, OperationInstall, scope, agent, opts.DryRun, "Unable to prepare the repo config target.", errs)
	}
	if err := s.applyWrites(writes, opts.DryRun); err != nil {
		return s.errorResult("repo config init", ResourceConfig, OperationInstall, scope, agent, opts.DryRun, "Unable to write the repo config target.", []CommandError{*err})
	}
	return s.successResultWithWarnings("repo config init", ResourceConfig, OperationInstall, scope, agent, opts.DryRun, writes, warnings)
}

func (s Service) RefreshConfig(opts Options) Result {
	scope := ScopeRepo
	agent := normalizeAgent(opts.Agent)
	writes, errs := s.planRepoConfigRefresh()
	if len(errs) > 0 {
		return s.errorResult("repo config refresh", ResourceConfig, OperationRefresh, scope, agent, opts.DryRun, "Unable to refresh the repo config target.", errs)
	}
	if err := s.applyWrites(writes, opts.DryRun); err != nil {
		return s.errorResult("repo config refresh", ResourceConfig, OperationRefresh, scope, agent, opts.DryRun, "Unable to write the repo config target.", []CommandError{*err})
	}
	return s.successResult("repo config refresh", ResourceConfig, OperationRefresh, scope, agent, opts.DryRun, writes)
}

func (s Service) PlanConfigRefreshDiff() (string, []CommandError) {
	writes, errs := s.planRepoConfigRefresh()
	if len(errs) > 0 {
		return "", errs
	}
	if len(writes) == 0 || writes[0].kind == ActionNoop {
		return "", nil
	}
	write := writes[0]
	switch write.kind {
	case ActionCreate:
		return renderUnifiedFileDiff(pathLabel(s.Workdir, write.path), nil, false, write.content), nil
	case ActionUpdate:
		current, err := os.ReadFile(write.path)
		if err != nil {
			return "", []CommandError{{
				Path:    pathLabel(s.Workdir, write.path),
				Message: fmt.Sprintf("read repo config target: %v", err),
			}}
		}
		return renderUnifiedFileDiff(pathLabel(s.Workdir, write.path), current, true, write.content), nil
	default:
		return "", nil
	}
}

func (s Service) RefreshConfigPreviewError(errs []CommandError) Result {
	return s.errorResult("repo config refresh", ResourceConfig, OperationRefresh, ScopeRepo, defaultAgent, true, "Unable to refresh the repo config target.", errs)
}

func (s Service) runSkillCommand(command, operation string, opts Options) Result {
	scope := normalizeScope(opts.Scope)
	if scope == "" {
		scope = ScopeRepo
	}
	if !isValidScope(scope) {
		return s.errorResult(command, ResourceSkills, operation, scope, normalizeAgent(opts.Agent), opts.DryRun, "Unsupported skills scope.", []CommandError{{
			Path:    "scope",
			Message: fmt.Sprintf("supported scopes are %q and %q", ScopeRepo, ScopeUser),
		}})
	}
	agent := normalizeAgent(opts.Agent)
	targetDir, err := s.resolveSkillsDir(agent, scope, opts.SkillsDir)
	if err != nil {
		return s.errorResult(command, ResourceSkills, operation, scope, agent, opts.DryRun, "Unable to resolve the skills target.", []CommandError{{Path: "skills.dir", Message: err.Error()}})
	}

	var (
		writes []plannedWrite
		errs   []CommandError
	)
	if operation == OperationInstall {
		writes, errs = s.planSkillsInstall(targetDir)
	} else {
		writes, errs = s.planSkillsUninstall(targetDir)
	}
	if len(errs) > 0 {
		return s.errorResult(command, ResourceSkills, operation, scope, agent, opts.DryRun, "Unable to prepare the skills target.", errs)
	}
	if err := s.applyWrites(writes, opts.DryRun); err != nil {
		return s.errorResult(command, ResourceSkills, operation, scope, agent, opts.DryRun, "Unable to write the skills target.", []CommandError{*err})
	}
	return s.successResultWithWarnings(command, ResourceSkills, operation, scope, agent, opts.DryRun, writes, s.repoConfigWarnings(scope))
}

func (s Service) runInstructionsCommand(command, operation string, opts Options) Result {
	scope := normalizeScope(opts.Scope)
	if scope == "" {
		scope = ScopeRepo
	}
	if !isValidScope(scope) {
		return s.errorResult(command, ResourceInstructions, operation, scope, normalizeAgent(opts.Agent), opts.DryRun, "Unsupported instructions scope.", []CommandError{{
			Path:    "scope",
			Message: fmt.Sprintf("supported scopes are %q and %q", ScopeRepo, ScopeUser),
		}})
	}
	agent := normalizeAgent(opts.Agent)
	targetFile, err := s.resolveInstructionsFile(agent, scope, opts.InstructionsFile)
	if err != nil {
		return s.errorResult(command, ResourceInstructions, operation, scope, agent, opts.DryRun, "Unable to resolve the instructions target.", []CommandError{{Path: "instructions.file", Message: err.Error()}})
	}

	var (
		writes []plannedWrite
		errs   []CommandError
	)
	if operation == OperationInstall {
		skillsDir, skillsErr := s.resolveSkillsDir(agent, scope, opts.SkillsDir)
		if skillsErr != nil {
			return s.errorResult(command, ResourceInstructions, operation, scope, agent, opts.DryRun, "Unable to resolve the instructions target.", []CommandError{{Path: "skills.dir", Message: skillsErr.Error()}})
		}
		writes, errs = s.planInstructionsInstall(targetFile, skillsDir)
	} else {
		writes, errs = s.planInstructionsUninstall(targetFile)
	}
	if len(errs) > 0 {
		return s.errorResult(command, ResourceInstructions, operation, scope, agent, opts.DryRun, "Unable to prepare the instructions target.", errs)
	}
	if err := s.applyWrites(writes, opts.DryRun); err != nil {
		return s.errorResult(command, ResourceInstructions, operation, scope, agent, opts.DryRun, "Unable to write the instructions target.", []CommandError{*err})
	}
	return s.successResultWithWarnings(command, ResourceInstructions, operation, scope, agent, opts.DryRun, writes, s.repoConfigWarnings(scope))
}

func (s Service) resolveSkillsDir(agent, scope, override string) (string, error) {
	if strings.TrimSpace(override) != "" {
		return s.resolvePath(override), nil
	}
	switch agent {
	case defaultAgent:
		if scope == ScopeRepo {
			return filepath.Join(s.Workdir, ".agents", "skills"), nil
		}
		home, err := s.codexHome()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "skills"), nil
	default:
		return "", fmt.Errorf("agent %q requires an explicit --dir override for skills", agent)
	}
}

func (s Service) resolveInstructionsFile(agent, scope, override string) (string, error) {
	if strings.TrimSpace(override) != "" {
		return s.resolvePath(override), nil
	}
	switch agent {
	case defaultAgent:
		if scope == ScopeRepo {
			return filepath.Join(s.Workdir, "AGENTS.md"), nil
		}
		home, err := s.codexHome()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "AGENTS.md"), nil
	default:
		return "", fmt.Errorf("agent %q requires an explicit --file override for instructions", agent)
	}
}

func (s Service) codexHome() (string, error) {
	if lookup := s.LookupEnv; lookup != nil {
		if value, ok := lookup("CODEX_HOME"); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value), nil
		}
	} else if value := strings.TrimSpace(os.Getenv("CODEX_HOME")); value != "" {
		return value, nil
	}

	var (
		home string
		err  error
	)
	if s.UserHomeDir != nil {
		home, err = s.UserHomeDir()
	} else {
		home, err = os.UserHomeDir()
	}
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	if strings.TrimSpace(home) == "" {
		return "", fmt.Errorf("resolve user home: empty path")
	}
	return filepath.Join(home, ".codex"), nil
}

func (s Service) planInstructionsInstall(targetFile, skillsDir string) ([]plannedWrite, []CommandError) {
	skillsDisplay := pathLabel(s.Workdir, skillsDir)
	data, err := os.ReadFile(targetFile)
	if err != nil {
		if os.IsNotExist(err) {
			content := defaultInstructionScaffold(targetFile, "\n") + renderManagedBlock("\n", skillsDisplay, s.versionTag())
			return []plannedWrite{{
				path:    targetFile,
				kind:    ActionCreate,
				details: "Create the instructions file with the easyharness-managed bootstrap block.",
				content: []byte(content),
			}}, nil
		}
		return nil, []CommandError{{Path: pathLabel(s.Workdir, targetFile), Message: err.Error()}}
	}

	existing := string(data)
	lineEnding := detectLineEnding(existing)
	managedBlock := renderManagedBlock(lineEnding, skillsDisplay, s.versionTag())
	beginMatches := instructionsManagedBlockBeginPattern.FindAllStringIndex(existing, -1)
	endMatches := instructionsManagedBlockEndPattern.FindAllStringIndex(existing, -1)
	if err := validateManagedBlockLayout(beginMatches, endMatches); err != nil {
		return nil, []CommandError{{Path: pathLabel(s.Workdir, targetFile), Message: err.Error()}}
	}

	var next string
	var kind string
	var details string
	switch len(beginMatches) {
	case 0:
		trimmed := strings.TrimSpace(existing)
		if trimmed == "" {
			next = defaultInstructionScaffold(targetFile, lineEnding) + managedBlock
			kind = ActionUpdate
			details = "Populate the empty instructions file with the easyharness-managed bootstrap block."
		} else {
			next = trimTrailingLineBreaks(existing) + lineEnding + lineEnding + managedBlock
			kind = ActionUpdate
			details = "Append the easyharness-managed bootstrap block to the instructions file."
		}
	default:
		begin := beginMatches[0][0]
		end := endMatches[0][1]
		before := trimTrailingLineBreaks(existing[:begin])
		after := trimSurroundingLineBreaks(existing[end:])
		parts := []string{}
		if before != "" {
			parts = append(parts, before)
		}
		parts = append(parts, trimTrailingLineBreaks(managedBlock))
		if after != "" {
			parts = append(parts, after)
		}
		next = strings.Join(parts, lineEnding+lineEnding) + lineEnding
		kind = ActionUpdate
		details = "Refresh the existing easyharness-managed bootstrap block in place."
	}

	if next == existing {
		kind = ActionNoop
		details = "Instructions file already contains the current easyharness-managed bootstrap block."
	}

	return []plannedWrite{{
		path:    targetFile,
		kind:    kind,
		details: details,
		content: []byte(next),
	}}, nil
}

func (s Service) planInstructionsUninstall(targetFile string) ([]plannedWrite, []CommandError) {
	data, err := os.ReadFile(targetFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []plannedWrite{{
				path:    targetFile,
				kind:    ActionNoop,
				details: "Instructions file is already absent.",
			}}, nil
		}
		return nil, []CommandError{{Path: pathLabel(s.Workdir, targetFile), Message: err.Error()}}
	}

	existing := string(data)
	beginMatches := instructionsManagedBlockBeginPattern.FindAllStringIndex(existing, -1)
	endMatches := instructionsManagedBlockEndPattern.FindAllStringIndex(existing, -1)
	if err := validateManagedBlockLayout(beginMatches, endMatches); err != nil {
		return nil, []CommandError{{Path: pathLabel(s.Workdir, targetFile), Message: err.Error()}}
	}
	if len(beginMatches) == 0 {
		return []plannedWrite{{
			path:    targetFile,
			kind:    ActionNoop,
			details: "Instructions file does not contain an easyharness-managed bootstrap block.",
		}}, nil
	}

	lineEnding := detectLineEnding(existing)
	begin := beginMatches[0][0]
	end := endMatches[0][1]
	before := trimTrailingLineBreaks(existing[:begin])
	after := trimSurroundingLineBreaks(existing[end:])
	parts := []string{}
	if before != "" {
		parts = append(parts, before)
	}
	if after != "" {
		parts = append(parts, after)
	}
	remaining := ""
	if len(parts) > 0 {
		remaining = strings.Join(parts, lineEnding+lineEnding) + lineEnding
	}

	trimmedRemaining := strings.TrimSpace(remaining)
	defaultHeading := strings.TrimSpace(strings.TrimRight(defaultInstructionScaffold(targetFile, lineEnding), lineEnding))
	if trimmedRemaining == "" || trimmedRemaining == defaultHeading {
		return []plannedWrite{{
			path:    targetFile,
			kind:    ActionDelete,
			details: "Remove the instructions file because it only contained easyharness-managed bootstrap content.",
			pruneTo: filepath.Dir(targetFile),
		}}, nil
	}

	return []plannedWrite{{
		path:    targetFile,
		kind:    ActionUpdate,
		details: "Remove the easyharness-managed bootstrap block while preserving user-owned instructions.",
		content: []byte(remaining),
	}}, nil
}

func (s Service) planSkillsInstall(targetDir string) ([]plannedWrite, []CommandError) {
	canonical, err := s.renderCanonicalSkillFiles()
	if err != nil {
		return nil, []CommandError{{Path: pathLabel(s.Workdir, targetDir), Message: err.Error()}}
	}

	writes := []plannedWrite{}
	errs := []CommandError{}

	installed, err := discoverInstalledSkills(targetDir)
	if err != nil {
		return nil, []CommandError{{Path: pathLabel(s.Workdir, targetDir), Message: err.Error()}}
	}

	canonicalNames := make(map[string]struct{}, len(canonical))
	for name := range canonical {
		canonicalNames[name] = struct{}{}
	}

	for skillName, state := range installed {
		if _, ok := canonicalNames[skillName]; ok {
			continue
		}
		if !state.Managed {
			continue
		}
		deleteWrites, deleteErr := planDeleteTree(state.Root, "Remove stale easyharness-managed skill that is no longer part of the packaged bootstrap set.")
		if deleteErr != nil {
			errs = append(errs, CommandError{Path: pathLabel(s.Workdir, state.Root), Message: deleteErr.Error()})
			continue
		}
		writes = append(writes, deleteWrites...)
	}

	for skillName, files := range canonical {
		targetRoot := filepath.Join(targetDir, skillName)
		if _, ok := installed[skillName]; !ok {
			existingFiles, existingErr := walkFiles(targetRoot)
			if existingErr != nil && !os.IsNotExist(existingErr) {
				errs = append(errs, CommandError{Path: pathLabel(s.Workdir, targetRoot), Message: existingErr.Error()})
				continue
			}
			if len(existingFiles) > 0 {
				errs = append(errs, CommandError{
					Path:    pathLabel(s.Workdir, targetRoot),
					Message: "target skill directory already exists and is not recognized as easyharness-managed",
				})
				continue
			}
		}
		if state, ok := installed[skillName]; ok && !state.Managed {
			legacyManaged, legacyErr := isLegacyManagedSkill(targetRoot, skillName, files)
			if legacyErr != nil {
				errs = append(errs, CommandError{Path: pathLabel(s.Workdir, targetRoot), Message: legacyErr.Error()})
				continue
			}
			if !legacyManaged {
				errs = append(errs, CommandError{
					Path:    pathLabel(s.Workdir, targetRoot),
					Message: "target skill already exists and is not recognized as easyharness-managed",
				})
				continue
			}
		}

		existingFiles, err := walkFiles(targetRoot)
		if err != nil {
			errs = append(errs, CommandError{Path: pathLabel(s.Workdir, targetRoot), Message: err.Error()})
			continue
		}
		expectedPaths := make(map[string]struct{}, len(files))
		for rel, content := range files {
			targetPath := filepath.Join(targetRoot, filepath.FromSlash(rel))
			expectedPaths[targetPath] = struct{}{}
			current, readErr := os.ReadFile(targetPath)
			if readErr != nil {
				if os.IsNotExist(readErr) {
					writes = append(writes, plannedWrite{
						path:    targetPath,
						kind:    ActionCreate,
						details: "Create the easyharness-managed skill file from packaged bootstrap assets.",
						content: []byte(content),
					})
					continue
				}
				errs = append(errs, CommandError{Path: pathLabel(s.Workdir, targetPath), Message: readErr.Error()})
				continue
			}
			kind := ActionUpdate
			details := "Refresh the easyharness-managed skill file from packaged bootstrap assets."
			if string(current) == content {
				kind = ActionNoop
				details = "Skill file already matches the current easyharness-managed bootstrap asset."
			}
			writes = append(writes, plannedWrite{
				path:    targetPath,
				kind:    kind,
				details: details,
				content: []byte(content),
			})
		}
		for _, existing := range existingFiles {
			if _, ok := expectedPaths[existing]; ok {
				continue
			}
			writes = append(writes, plannedWrite{
				path:    existing,
				kind:    ActionDelete,
				details: "Remove stale file from the easyharness-managed skill directory.",
				pruneTo: targetDir,
			})
		}
	}

	return writes, errs
}

func (s Service) planSkillsUninstall(targetDir string) ([]plannedWrite, []CommandError) {
	installed, err := discoverInstalledSkills(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []plannedWrite{{
				path:    targetDir,
				kind:    ActionNoop,
				details: "Skills directory is already absent.",
			}}, nil
		}
		return nil, []CommandError{{Path: pathLabel(s.Workdir, targetDir), Message: err.Error()}}
	}

	writes := []plannedWrite{}
	for _, state := range installed {
		if !state.Managed {
			continue
		}
		deleteWrites, deleteErr := planDeleteTree(state.Root, "Remove the easyharness-managed skill package.")
		if deleteErr != nil {
			return nil, []CommandError{{Path: pathLabel(s.Workdir, state.Root), Message: deleteErr.Error()}}
		}
		writes = append(writes, deleteWrites...)
	}
	if len(writes) == 0 {
		return []plannedWrite{{
			path:    targetDir,
			kind:    ActionNoop,
			details: "No easyharness-managed skill packages were found in the target directory.",
		}}, nil
	}
	return writes, nil
}

func (s Service) planRepoConfigInit() ([]plannedWrite, []string, []CommandError) {
	configPath := filepath.Join(s.Workdir, filepath.FromSlash(repoconfig.File))
	info, statErr := os.Stat(configPath)
	if statErr != nil && !os.IsNotExist(statErr) {
		return nil, nil, []CommandError{{
			Path:    pathLabel(s.Workdir, configPath),
			Message: fmt.Sprintf("inspect repo config target: %v", statErr),
		}}
	}
	if statErr == nil && info.IsDir() {
		return nil, nil, []CommandError{{
			Path:    pathLabel(s.Workdir, configPath),
			Message: "repo config target is a directory",
		}}
	}

	result := repoconfig.Load(s.Workdir)
	if result.Exists {
		return []plannedWrite{{
			path:    result.Path,
			kind:    ActionNoop,
			details: "Repo config already exists and will not be overwritten.",
		}}, repoConfigWarningsForWorkdir(s.Workdir, result.Warnings), nil
	}
	return []plannedWrite{{
		path:    result.Path,
		kind:    ActionCreate,
		details: "Create the repo customization manifest with the current canonical config.",
		content: []byte(repoconfig.DefaultContent),
	}}, repoConfigWarningsForWorkdir(s.Workdir, result.Warnings), nil
}

func (s Service) planRepoConfigRefresh() ([]plannedWrite, []CommandError) {
	configPath := filepath.Join(s.Workdir, filepath.FromSlash(repoconfig.File))
	info, statErr := os.Stat(configPath)
	if statErr != nil && !os.IsNotExist(statErr) {
		return nil, []CommandError{{
			Path:    pathLabel(s.Workdir, configPath),
			Message: fmt.Sprintf("inspect repo config target: %v", statErr),
		}}
	}
	if statErr == nil && info.IsDir() {
		return nil, []CommandError{{
			Path:    pathLabel(s.Workdir, configPath),
			Message: "repo config target is a directory",
		}}
	}

	result := repoconfig.Load(s.Workdir)
	if result.Exists && !result.Valid {
		return nil, []CommandError{{
			Path:    pathLabel(s.Workdir, result.Path),
			Message: result.InvalidReason,
		}}
	}
	content := []byte(repoconfig.DefaultContent)
	if result.Exists {
		content = []byte(repoconfig.Render(result.Config))
		current, err := os.ReadFile(result.Path)
		if err != nil {
			return nil, []CommandError{{
				Path:    pathLabel(s.Workdir, result.Path),
				Message: fmt.Sprintf("read repo config target: %v", err),
			}}
		}
		if string(current) == string(content) {
			return []plannedWrite{{
				path:    result.Path,
				kind:    ActionNoop,
				details: "Repo config already matches the current canonical config.",
			}}, nil
		}
		return []plannedWrite{{
			path:    result.Path,
			kind:    ActionUpdate,
			details: "Refresh the repo config to the current canonical config while preserving configured values.",
			content: content,
		}}, nil
	}
	return []plannedWrite{{
		path:    result.Path,
		kind:    ActionCreate,
		details: "Create the repo customization manifest with the current canonical config.",
		content: content,
	}}, nil
}

func (s Service) renderCanonicalSkillFiles() (map[string]map[string]string, error) {
	files, err := bootstrapassets.SkillFiles()
	if err != nil {
		return nil, err
	}
	skills := map[string]map[string]string{}
	for relPath, content := range files {
		parts := strings.SplitN(relPath, "/", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("unexpected packaged skill path %q", relPath)
		}
		root := parts[0]
		inner := parts[1]
		if _, ok := skills[root]; !ok {
			skills[root] = map[string]string{}
		}
		if inner == "SKILL.md" {
			rendered, renderErr := renderManagedSkill(content, s.versionTag())
			if renderErr != nil {
				return nil, fmt.Errorf("render managed skill %s: %w", root, renderErr)
			}
			content = rendered
		}
		skills[root][inner] = content
	}
	return skills, nil
}

func (s Service) applyWrites(writes []plannedWrite, dryRun bool) *CommandError {
	if dryRun {
		return nil
	}
	for _, write := range writes {
		switch write.kind {
		case ActionNoop:
			continue
		case ActionDelete:
			if err := os.Remove(write.path); err != nil && !os.IsNotExist(err) {
				return &CommandError{Path: pathLabel(s.Workdir, write.path), Message: fmt.Sprintf("remove path: %v", err)}
			}
			pruneEmptyParents(filepath.Dir(write.path), write.pruneTo)
		default:
			if err := os.MkdirAll(filepath.Dir(write.path), 0o755); err != nil {
				return &CommandError{Path: pathLabel(s.Workdir, write.path), Message: fmt.Sprintf("create parent directory: %v", err)}
			}
			if err := os.WriteFile(write.path, write.content, 0o644); err != nil {
				return &CommandError{Path: pathLabel(s.Workdir, write.path), Message: fmt.Sprintf("write file: %v", err)}
			}
		}
	}
	return nil
}

func (s Service) successResult(command, resource, operation, scope, agent string, dryRun bool, writes []plannedWrite) Result {
	return s.successResultWithWarnings(command, resource, operation, scope, agent, dryRun, writes, nil)
}

func (s Service) successResultWithWarnings(command, resource, operation, scope, agent string, dryRun bool, writes []plannedWrite, warnings []string) Result {
	return Result{
		OK:         true,
		Command:    command,
		Summary:    summarizeWrites(resource, operation, dryRun, writes),
		Mode:       modeName(dryRun),
		Resource:   resource,
		Operation:  operation,
		Scope:      scope,
		Agent:      agent,
		Actions:    toActions(s.Workdir, writes),
		NextAction: []NextAction{},
		Warnings:   warnings,
	}
}

func (s Service) errorResult(command, resource, operation, scope, agent string, dryRun bool, summary string, errs []CommandError) Result {
	return Result{
		OK:         false,
		Command:    command,
		Summary:    summary,
		Mode:       modeName(dryRun),
		Resource:   resource,
		Operation:  operation,
		Scope:      scope,
		Agent:      agent,
		Actions:    []Action{},
		NextAction: []NextAction{},
		Errors:     errs,
	}
}

func summarizeWrites(resource, operation string, dryRun bool, writes []plannedWrite) string {
	creates, updates, deletes := countWrites(writes)
	if creates == 0 && updates == 0 && deletes == 0 {
		if operation == OperationUninstall {
			if dryRun {
				return fmt.Sprintf("Dry run complete. No easyharness-managed %s assets would be removed.", resource)
			}
			return fmt.Sprintf("No easyharness-managed %s assets were removed.", resource)
		}
		if dryRun {
			return fmt.Sprintf("Dry run complete. %s assets are already up to date.", strings.Title(resource))
		}
		return fmt.Sprintf("%s assets are already up to date.", strings.Title(resource))
	}
	if dryRun {
		return fmt.Sprintf("Dry run complete. %d path(s) would be created, %d updated, and %d deleted.", creates, updates, deletes)
	}
	return fmt.Sprintf("Applied bootstrap changes. %d path(s) created, %d updated, and %d deleted.", creates, updates, deletes)
}

func defaultInstructionScaffold(targetFile, lineEnding string) string {
	return "# " + filepath.Base(targetFile) + lineEnding + lineEnding
}

func renderManagedBlock(lineEnding, skillsDisplay, version string) string {
	body := normalizeLineEndings(strings.TrimSpace(bootstrapassets.AgentsManagedBlock()), lineEnding)
	body = strings.ReplaceAll(body, ".agents/skills/", skillsDisplay)
	begin := fmt.Sprintf(`<!-- easyharness:begin version="%s" -->`, version)
	return begin + lineEnding + body + lineEnding + instructionsManagedBlockEnd + lineEnding
}

func renderManagedSkill(content, version string) (string, error) {
	rawFrontmatter, body, err := splitFrontmatter(content)
	if err != nil {
		return "", err
	}
	var frontmatter skillFrontmatter
	if err := yaml.Unmarshal([]byte(rawFrontmatter), &frontmatter); err != nil {
		return "", fmt.Errorf("parse frontmatter: %w", err)
	}
	if frontmatter.Metadata == nil {
		frontmatter.Metadata = map[string]string{}
	}
	frontmatter.Metadata[skillMetadataManaged] = "true"
	frontmatter.Metadata[skillMetadataVersion] = version
	renderedFrontmatter, err := yaml.Marshal(frontmatter)
	if err != nil {
		return "", fmt.Errorf("render frontmatter: %w", err)
	}
	return "---\n" + string(renderedFrontmatter) + "---\n\n" + strings.TrimLeft(body, "\n"), nil
}

func (s Service) versionTag() string {
	info := s.Version
	if info.Mode == "" && info.Commit == "" && info.Version == "" {
		info = versioninfo.Current()
	}
	if strings.TrimSpace(info.Mode) == "dev" {
		return "dev"
	}
	if strings.TrimSpace(info.Version) != "" {
		return strings.TrimSpace(info.Version)
	}
	return "dev"
}

func (s Service) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(s.Workdir, path))
}

func normalizeAgent(agent string) string {
	agent = strings.TrimSpace(strings.ToLower(agent))
	if agent == "" {
		return defaultAgent
	}
	return agent
}

func normalizeScope(scope string) string {
	return strings.TrimSpace(strings.ToLower(scope))
}

func isValidScope(scope string) bool {
	switch scope {
	case ScopeRepo, ScopeUser:
		return true
	default:
		return false
	}
}

func modeName(dryRun bool) string {
	if dryRun {
		return "dry_run"
	}
	return "apply"
}

func toActions(workdir string, writes []plannedWrite) []Action {
	actions := make([]Action, 0, len(writes))
	for _, write := range writes {
		actions = append(actions, Action{
			Path:    pathLabel(workdir, write.path),
			Kind:    write.kind,
			Details: write.details,
		})
	}
	return actions
}

func countWrites(writes []plannedWrite) (int, int, int) {
	var creates, updates, deletes int
	for _, write := range writes {
		switch write.kind {
		case ActionCreate:
			creates++
		case ActionUpdate:
			updates++
		case ActionDelete:
			deletes++
		}
	}
	return creates, updates, deletes
}

func detectLineEnding(content string) string {
	if strings.Contains(content, "\r\n") {
		return "\r\n"
	}
	return "\n"
}

func normalizeLineEndings(content, lineEnding string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return strings.ReplaceAll(content, "\n", lineEnding)
}

func trimTrailingLineBreaks(content string) string {
	return strings.TrimRight(content, "\r\n")
}

func trimSurroundingLineBreaks(content string) string {
	return strings.Trim(content, "\r\n")
}

func pathLabel(workdir, path string) string {
	if rel, err := filepath.Rel(workdir, path); err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != ".." {
		return filepath.ToSlash(rel)
	}
	return filepath.Clean(path)
}

func renderUnifiedFileDiff(path string, oldContent []byte, oldExists bool, newContent []byte) string {
	if oldExists && string(oldContent) == string(newContent) {
		return ""
	}
	oldLines := splitDiffLines(string(oldContent))
	newLines := splitDiffLines(string(newContent))
	var b strings.Builder
	if oldExists {
		fmt.Fprintf(&b, "--- a/%s\n", path)
	} else {
		b.WriteString("--- /dev/null\n")
	}
	fmt.Fprintf(&b, "+++ b/%s\n", path)
	fmt.Fprintf(&b, "@@ -%s +%s @@\n", unifiedRange(len(oldLines), oldExists), unifiedRange(len(newLines), true))
	for _, line := range oldLines {
		writeDiffLine(&b, '-', line)
	}
	for _, line := range newLines {
		writeDiffLine(&b, '+', line)
	}
	return b.String()
}

func splitDiffLines(content string) []string {
	if content == "" {
		return nil
	}
	lines := strings.SplitAfter(content, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func unifiedRange(lineCount int, exists bool) string {
	if !exists || lineCount == 0 {
		return "0,0"
	}
	return fmt.Sprintf("1,%d", lineCount)
}

func writeDiffLine(b *strings.Builder, prefix byte, line string) {
	b.WriteByte(prefix)
	b.WriteString(line)
	if !strings.HasSuffix(line, "\n") {
		b.WriteByte('\n')
	}
}

func repoConfigWarningsForWorkdir(workdir string, warnings []string) []string {
	if len(warnings) == 0 {
		return nil
	}
	replacer := strings.NewReplacer(filepath.ToSlash(workdir)+"/", "")
	result := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		result = append(result, replacer.Replace(warning))
	}
	return result
}

func (s Service) repoConfigWarnings(scope string) []string {
	if scope != ScopeRepo {
		return nil
	}
	return repoConfigWarningsForWorkdir(s.Workdir, repoconfig.Load(s.Workdir).Warnings)
}

func walkFiles(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}

	files := []string{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func planDeleteTree(root, details string) ([]plannedWrite, error) {
	files, err := walkFiles(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	writes := make([]plannedWrite, 0, len(files))
	pruneTo := filepath.Dir(root)
	for i := len(files) - 1; i >= 0; i-- {
		writes = append(writes, plannedWrite{
			path:    files[i],
			kind:    ActionDelete,
			details: details,
			pruneTo: pruneTo,
		})
	}
	return writes, nil
}

func pruneEmptyParents(dir, stop string) {
	stop = filepath.Clean(stop)
	for {
		cleanDir := filepath.Clean(dir)
		if cleanDir == stop || cleanDir == filepath.Dir(cleanDir) {
			return
		}
		entries, err := os.ReadDir(cleanDir)
		if err != nil || len(entries) > 0 {
			return
		}
		if err := os.Remove(cleanDir); err != nil {
			return
		}
		dir = filepath.Dir(cleanDir)
	}
}

func validateManagedBlockLayout(beginMatches, endMatches [][]int) error {
	if len(beginMatches) != len(endMatches) || len(beginMatches) > 1 {
		return fmt.Errorf("expected zero or one easyharness managed block; found an ambiguous marker layout")
	}
	if len(beginMatches) == 1 && beginMatches[0][0] >= endMatches[0][0] {
		return fmt.Errorf("expected zero or one easyharness managed block; found an ambiguous marker layout")
	}
	return nil
}

func splitFrontmatter(content string) (string, string, error) {
	var rest string
	switch {
	case strings.HasPrefix(content, "---\r\n"):
		rest = content[len("---\r\n"):]
	case strings.HasPrefix(content, "---\n"):
		rest = content[len("---\n"):]
	default:
		return "", "", fmt.Errorf("file must start with YAML frontmatter delimited by ---")
	}

	for i := 0; i < len(rest); {
		lineEnd := strings.IndexAny(rest[i:], "\r\n")
		if lineEnd < 0 {
			if rest[i:] == "---" {
				return strings.Trim(rest[:i], "\r\n"), "", nil
			}
			break
		}
		lineEnd += i
		if rest[i:lineEnd] == "---" {
			sepLen := 1
			if rest[lineEnd] == '\r' && lineEnd+1 < len(rest) && rest[lineEnd+1] == '\n' {
				sepLen = 2
			}
			return strings.Trim(rest[:i], "\r\n"), rest[lineEnd+sepLen:], nil
		}
		if rest[lineEnd] == '\r' && lineEnd+1 < len(rest) && rest[lineEnd+1] == '\n' {
			i = lineEnd + 2
			continue
		}
		i = lineEnd + 1
	}
	return "", "", fmt.Errorf("frontmatter is missing a closing --- delimiter")
}

type skillFrontmatter struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license,omitempty"`
	Compatibility string            `yaml:"compatibility,omitempty"`
	Metadata      map[string]string `yaml:"metadata,omitempty"`
	AllowedTools  string            `yaml:"allowed-tools,omitempty"`
}

type installedSkillState struct {
	Root    string
	Managed bool
}

func discoverInstalledSkills(targetDir string) (map[string]installedSkillState, error) {
	info, err := os.Stat(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]installedSkillState{}, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", targetDir)
	}
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return nil, err
	}
	states := map[string]installedSkillState{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		root := filepath.Join(targetDir, entry.Name())
		skillPath := filepath.Join(root, "SKILL.md")
		managed, managedErr := isManagedSkill(skillPath)
		if managedErr != nil {
			if os.IsNotExist(managedErr) {
				continue
			}
			return nil, managedErr
		}
		states[entry.Name()] = installedSkillState{Root: root, Managed: managed}
	}
	return states, nil
}

func isManagedSkill(skillPath string) (bool, error) {
	data, err := os.ReadFile(skillPath)
	if err != nil {
		return false, err
	}
	rawFrontmatter, _, err := splitFrontmatter(string(data))
	if err != nil {
		return false, err
	}
	var frontmatter skillFrontmatter
	if err := yaml.Unmarshal([]byte(rawFrontmatter), &frontmatter); err != nil {
		return false, err
	}
	return frontmatter.Metadata != nil && strings.TrimSpace(frontmatter.Metadata[skillMetadataManaged]) == "true", nil
}

func isLegacyManagedSkill(root, skillName string, canonicalFiles map[string]string) (bool, error) {
	existingFiles, err := walkFiles(root)
	if err != nil {
		return false, err
	}
	expected := make(map[string]struct{}, len(canonicalFiles))
	for rel := range canonicalFiles {
		expected[filepath.Join(root, filepath.FromSlash(rel))] = struct{}{}
	}
	for _, existing := range existingFiles {
		if _, ok := expected[existing]; !ok {
			return false, nil
		}
	}

	for rel, canonicalContent := range canonicalFiles {
		targetPath := filepath.Join(root, filepath.FromSlash(rel))
		existing, readErr := os.ReadFile(targetPath)
		if readErr != nil {
			return false, readErr
		}
		expected := canonicalContent
		if rel == "SKILL.md" {
			expected, err = stripManagedSkillMetadata(canonicalContent)
			if err != nil {
				return false, err
			}
		}
		if normalizeSkillContent(string(existing)) != normalizeSkillContent(expected) {
			return false, nil
		}
	}
	return true, nil
}

func stripManagedSkillMetadata(content string) (string, error) {
	rawFrontmatter, body, err := splitFrontmatter(content)
	if err != nil {
		return "", err
	}
	var frontmatter skillFrontmatter
	if err := yaml.Unmarshal([]byte(rawFrontmatter), &frontmatter); err != nil {
		return "", err
	}
	if frontmatter.Metadata != nil {
		delete(frontmatter.Metadata, skillMetadataManaged)
		delete(frontmatter.Metadata, skillMetadataVersion)
		if len(frontmatter.Metadata) == 0 {
			frontmatter.Metadata = nil
		}
	}
	renderedFrontmatter, err := yaml.Marshal(frontmatter)
	if err != nil {
		return "", err
	}
	return "---\n" + string(renderedFrontmatter) + "---\n\n" + strings.TrimLeft(body, "\n"), nil
}

func normalizeSkillContent(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return strings.TrimSpace(content)
}
