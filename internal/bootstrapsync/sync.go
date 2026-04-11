package bootstrapsync

import (
	"errors"
	"fmt"
	"strings"

	"github.com/catu-ai/easyharness/internal/install"
)

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
	result := install.Service{Workdir: workdir}.Init(install.Options{})
	if !result.OK {
		return result, resultError(result)
	}
	return result, nil
}

func Check(workdir string) (install.Result, error) {
	result := install.Service{Workdir: workdir}.Init(install.Options{DryRun: true})
	if !result.OK {
		return result, resultError(result)
	}

	drifted := make([]install.Action, 0, len(result.Actions))
	for _, action := range result.Actions {
		if action.Kind != install.ActionNoop {
			drifted = append(drifted, action)
		}
	}
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
