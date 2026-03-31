package bootstrapsync

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	bootstrapassets "github.com/catu-ai/easyharness/assets/bootstrap"
	"github.com/catu-ai/easyharness/internal/install"
)

const actionDelete = "delete"

type DriftError struct {
	Actions []install.Action
}

func (e *DriftError) Error() string {
	if len(e.Actions) == 0 {
		return "bootstrap dogfood outputs drifted from assets/bootstrap"
	}

	lines := make([]string, 0, len(e.Actions)+1)
	lines = append(lines, "bootstrap dogfood outputs drifted from assets/bootstrap:")
	for _, action := range e.Actions {
		lines = append(lines, fmt.Sprintf("- %s: %s (%s)", action.Path, action.Kind, action.Details))
	}
	return strings.Join(lines, "\n")
}

func Sync(workdir string) (install.Result, error) {
	result := install.Service{Workdir: workdir}.Install(install.Options{})
	if !result.OK {
		return result, resultError(result)
	}

	deletions, err := removeOrphanedSkillFiles(workdir)
	if err != nil {
		return result, err
	}
	if len(deletions) > 0 {
		result.Actions = append(result.Actions, deletions...)
		result.Summary = summarizeApply(result.Summary, len(deletions))
	}
	return result, nil
}

func Check(workdir string) (install.Result, error) {
	result := install.Service{Workdir: workdir}.Install(install.Options{DryRun: true})
	if !result.OK {
		return result, resultError(result)
	}

	drifted := make([]install.Action, 0, len(result.Actions))
	for _, action := range result.Actions {
		if action.Kind != install.ActionNoop {
			drifted = append(drifted, action)
		}
	}
	orphaned, err := findOrphanedSkillFiles(workdir)
	if err != nil {
		return result, err
	}
	drifted = append(drifted, orphaned...)
	if len(drifted) > 0 {
		return result, &DriftError{Actions: drifted}
	}
	return result, nil
}

func resultError(result install.Result) error {
	if len(result.Errors) == 0 {
		return errors.New(result.Summary)
	}

	lines := make([]string, 0, len(result.Errors)+1)
	lines = append(lines, result.Summary)
	for _, err := range result.Errors {
		lines = append(lines, fmt.Sprintf("- %s: %s", err.Path, err.Message))
	}
	return errors.New(strings.Join(lines, "\n"))
}

func findOrphanedSkillFiles(workdir string) ([]install.Action, error) {
	root := filepath.Join(workdir, ".agents", "skills")
	entries, err := existingSkillFiles(root)
	if err != nil {
		return nil, err
	}

	canonical, err := bootstrapassets.SkillFiles()
	if err != nil {
		return nil, err
	}

	orphaned := make([]install.Action, 0)
	for _, relPath := range entries {
		canonicalPath := filepath.ToSlash(relPath)
		if _, ok := canonical[canonicalPath]; ok {
			continue
		}
		orphaned = append(orphaned, install.Action{
			Path:    filepath.ToSlash(filepath.Join(".agents/skills", relPath)),
			Kind:    actionDelete,
			Details: "Remove stale materialized skill file that no longer exists in assets/bootstrap.",
		})
	}
	return orphaned, nil
}

func removeOrphanedSkillFiles(workdir string) ([]install.Action, error) {
	orphaned, err := findOrphanedSkillFiles(workdir)
	if err != nil {
		return nil, err
	}

	deletions := make([]install.Action, 0, len(orphaned))
	for _, action := range orphaned {
		absPath := filepath.Join(workdir, filepath.FromSlash(action.Path))
		if err := os.Remove(absPath); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("remove orphaned skill file %s: %w", action.Path, err)
		}
		deletions = append(deletions, action)
		pruneEmptyParents(filepath.Dir(absPath), filepath.Join(workdir, ".agents", "skills"))
	}
	return deletions, nil
}

func existingSkillFiles(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat .agents/skills: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf(".agents/skills is not a directory")
	}

	files := make([]string, 0)
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, relPath)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk .agents/skills: %w", err)
	}
	return files, nil
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

func summarizeApply(base string, deletions int) string {
	if deletions == 0 {
		return base
	}
	if base == "Harness-managed repository assets are already up to date." {
		return fmt.Sprintf("Refreshed bootstrap dogfood outputs. %d stale materialized skill file(s) removed.", deletions)
	}
	return fmt.Sprintf("%s Removed %d stale materialized skill file(s).", base, deletions)
}
