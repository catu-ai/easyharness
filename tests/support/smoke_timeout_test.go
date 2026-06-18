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
	case os.Getenv("EASYHARNESS_TIMEOUT_HELPER_CALL") == "1":
		RunCommandWithTimeout(
			t,
			50*time.Millisecond,
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
		"timed out after 50ms",
		"helper stdout before sleep",
		"helper stderr before sleep",
		"-test.run=TestRunCommandWithTimeoutReportsCommandAndOutput",
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("expected timeout output to contain %q, got:\n%s", fragment, text)
		}
	}
}
