package support

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestRunCommandWithTimeoutReportsCommandAndOutput(t *testing.T) {
	switch {
	case os.Getenv("EASYHARNESS_TIMEOUT_HELPER_SLEEP") == "1":
		fmt.Println("helper stdout before sleep")
		fmt.Fprintln(os.Stderr, "helper stderr before sleep")
		time.Sleep(5 * time.Second)
		return
	case os.Getenv("EASYHARNESS_TIMEOUT_HELPER_RUN_WITH_OPTIONS") == "1":
		RunWithOptions(t, RunOptions{
			Workdir: RepoRoot(t),
			Args:    []string{"ui", "--port", "0", "--no-open"},
			Timeout: 500 * time.Millisecond,
		})
		return
	case os.Getenv("EASYHARNESS_TIMEOUT_HELPER_CALL") == "1":
		RunCommandWithTimeout(
			t,
			500*time.Millisecond,
			"",
			append(os.Environ(), "EASYHARNESS_TIMEOUT_HELPER_SLEEP=1"),
			os.Args[0],
			"-test.run=TestRunCommandWithTimeoutReportsCommandAndOutput",
		)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRunCommandWithTimeoutReportsCommandAndOutput")
	cmd.Env = append(os.Environ(), "EASYHARNESS_TIMEOUT_HELPER_CALL=1")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected helper test process to fail on timeout")
	}

	text := string(output)
	for _, fragment := range []string{
		"timed out after 500ms",
		"helper stdout before sleep",
		"helper stderr before sleep",
		"-test.run=TestRunCommandWithTimeoutReportsCommandAndOutput",
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("expected timeout output to contain %q, got:\n%s", fragment, text)
		}
	}
}

func TestRunWithOptionsReportsTimeout(t *testing.T) {
	if os.Getenv("EASYHARNESS_TIMEOUT_HELPER_RUN_WITH_OPTIONS") == "1" {
		TestRunCommandWithTimeoutReportsCommandAndOutput(t)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRunWithOptionsReportsTimeout")
	cmd.Env = append(os.Environ(), "EASYHARNESS_TIMEOUT_HELPER_RUN_WITH_OPTIONS=1")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected helper test process to fail on harness timeout")
	}

	text := string(output)
	for _, fragment := range []string{
		"run harness [ui --port 0 --no-open] timed out after 500ms",
		"stdout:",
		"stderr:",
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("expected timeout output to contain %q, got:\n%s", fragment, text)
		}
	}
}
