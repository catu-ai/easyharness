package repoconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingConfigUsesDefaults(t *testing.T) {
	result := Load(t.TempDir())
	if result.Exists {
		t.Fatalf("expected missing config, got %#v", result)
	}
	if !result.Valid || len(result.Warnings) != 0 {
		t.Fatalf("expected missing config to be valid defaults, got %#v", result)
	}
	if result.Config.Version != CurrentVersion {
		t.Fatalf("expected default version %d, got %#v", CurrentVersion, result)
	}
	if result.Config.Paths.Plans.Active != DefaultActivePlansRoot ||
		result.Config.Paths.Plans.Archived != DefaultArchivedPlansRoot ||
		result.Config.Paths.LocalRuntime != DefaultLocalRuntimeRoot {
		t.Fatalf("expected default paths, got %#v", result.Config.Paths)
	}
}

func TestLoadValidConfig(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, DefaultContent)

	result := Load(root)
	if !result.Exists || !result.Valid {
		t.Fatalf("expected valid existing config, got %#v", result)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", result.Warnings)
	}
}

func TestLoadValidCustomPathRoots(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `version: 1
paths:
  plans:
    active: workflow/plans/open
    archived: workflow/plans/done
  local_runtime: tmp/harness-runtime
`)

	result := Load(root)
	if !result.Exists || !result.Valid {
		t.Fatalf("expected valid custom config, got %#v", result)
	}
	if got, want := result.Config.Paths.Plans.Active, "workflow/plans/open"; got != want {
		t.Fatalf("active root = %q, want %q", got, want)
	}
	if got, want := result.Config.Paths.Plans.Archived, "workflow/plans/done"; got != want {
		t.Fatalf("archived root = %q, want %q", got, want)
	}
	if got, want := result.Config.Paths.LocalRuntime, "tmp/harness-runtime"; got != want {
		t.Fatalf("local runtime root = %q, want %q", got, want)
	}
}

func TestLoadPartiallySpecifiedPathsUseDefaults(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `version: 1
paths:
  plans:
    active: workflow/plans/open
`)

	result := Load(root)
	if !result.Valid {
		t.Fatalf("expected valid partial paths config, got %#v", result)
	}
	if got, want := result.Config.Paths.Plans.Active, "workflow/plans/open"; got != want {
		t.Fatalf("active root = %q, want %q", got, want)
	}
	if got, want := result.Config.Paths.Plans.Archived, DefaultArchivedPlansRoot; got != want {
		t.Fatalf("archived root = %q, want %q", got, want)
	}
	if got, want := result.Config.Paths.LocalRuntime, DefaultLocalRuntimeRoot; got != want {
		t.Fatalf("local runtime root = %q, want %q", got, want)
	}
}

func TestLoadInvalidConfigWarnsAndUsesDefaults(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		want    string
	}{
		{name: "malformed", content: "version: [\n", want: "malformed YAML"},
		{name: "non object", content: "- version\n", want: "YAML object"},
		{name: "missing version", content: "name: repo\n", want: "missing required field version"},
		{name: "unsupported version", content: "version: 2\n", want: "unsupported version 2"},
		{name: "string version", content: "version: \"1\"\n", want: "field version must be the integer 1"},
		{name: "unknown field", content: "version: 1\nreview: {}\n", want: "unsupported field"},
		{name: "paths non object", content: "version: 1\npaths: []\n", want: "field paths must be a YAML object"},
		{name: "unknown paths field", content: "version: 1\npaths:\n  hooks: {}\n", want: "unsupported field \"paths.hooks\""},
		{name: "unknown plans field", content: "version: 1\npaths:\n  plans:\n    current: docs/current\n", want: "unsupported field \"paths.plans.current\""},
		{name: "absolute path", content: "version: 1\npaths:\n  plans:\n    active: /tmp/plans\n", want: "paths.plans.active must be repo-relative"},
		{name: "escaping path", content: "version: 1\npaths:\n  plans:\n    active: ../plans\n", want: "paths.plans.active must stay within the repository"},
		{name: "empty path", content: "version: 1\npaths:\n  local_runtime: \"\"\n", want: "paths.local_runtime must not be empty"},
		{name: "overlapping plan roots", content: "version: 1\npaths:\n  plans:\n    active: workflow/plans\n    archived: workflow/plans/archived\n", want: "configured path roots must not overlap"},
		{name: "runtime inside plan root", content: "version: 1\npaths:\n  plans:\n    active: workflow\n  local_runtime: workflow/.local\n", want: "configured path roots must not overlap"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeConfig(t, root, tc.content)

			result := Load(root)
			if !result.Exists || result.Valid {
				t.Fatalf("expected invalid existing config, got %#v", result)
			}
			if result.Config.Version != CurrentVersion {
				t.Fatalf("expected defaults after invalid config, got %#v", result)
			}
			warnings := strings.Join(result.Warnings, "\n")
			if !strings.Contains(warnings, "Ignoring") || !strings.Contains(warnings, tc.want) || !strings.Contains(warnings, "using built-in defaults") {
				t.Fatalf("unexpected warnings %#v", result.Warnings)
			}
		})
	}
}

func TestGetScalarReturnsResolvedValues(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `version: 1
paths:
  plans:
    active: workflow/plans/open
  local_runtime: tmp/harness-runtime
`)

	result := Load(root)
	for _, tc := range []struct {
		key  string
		want string
	}{
		{key: "paths.plans.active", want: "workflow/plans/open"},
		{key: "paths.plans.archived", want: DefaultArchivedPlansRoot},
		{key: "paths.local_runtime", want: "tmp/harness-runtime"},
	} {
		got, err := result.Config.GetScalar(tc.key)
		if err != nil {
			t.Fatalf("GetScalar(%q) returned error: %v", tc.key, err)
		}
		if got != tc.want {
			t.Fatalf("GetScalar(%q) = %q, want %q", tc.key, got, tc.want)
		}
	}
}

func TestGetScalarRejectsObjectsAndUnknownKeys(t *testing.T) {
	config := DefaultConfig()

	if _, err := config.GetScalar("paths"); err == nil || !strings.Contains(err.Error(), "object") || !strings.Contains(err.Error(), "harness repo config list paths") {
		t.Fatalf("expected object error with list hint, got %v", err)
	}
	if _, err := config.GetScalar("paths.plans"); err == nil || !strings.Contains(err.Error(), "object") || !strings.Contains(err.Error(), "harness repo config list paths.plans") {
		t.Fatalf("expected nested object error with list hint, got %v", err)
	}
	if _, err := config.GetScalar("paths.missing"); err == nil || !strings.Contains(err.Error(), "unknown repo config key") {
		t.Fatalf("expected unknown key error, got %v", err)
	}
}

func TestListResolvedEntries(t *testing.T) {
	config := DefaultConfig()

	entries, err := config.ListResolved("")
	if err != nil {
		t.Fatalf("ListResolved returned error: %v", err)
	}
	want := []ConfigEntry{
		{Key: "paths.plans.active", Value: DefaultActivePlansRoot},
		{Key: "paths.plans.archived", Value: DefaultArchivedPlansRoot},
		{Key: "paths.local_runtime", Value: DefaultLocalRuntimeRoot},
	}
	if len(entries) != len(want) {
		t.Fatalf("expected %d entries, got %#v", len(want), entries)
	}
	for i := range want {
		if entries[i] != want[i] {
			t.Fatalf("entry %d = %#v, want %#v", i, entries[i], want[i])
		}
	}

	entries, err = config.ListResolved("paths.plans")
	if err != nil {
		t.Fatalf("ListResolved(paths.plans) returned error: %v", err)
	}
	if len(entries) != 2 || entries[0].Key != "paths.plans.active" || entries[1].Key != "paths.plans.archived" {
		t.Fatalf("unexpected paths.plans entries: %#v", entries)
	}

	entries, err = config.ListResolved("paths.local_runtime")
	if err != nil {
		t.Fatalf("ListResolved(paths.local_runtime) returned error: %v", err)
	}
	if len(entries) != 1 || entries[0].Key != "paths.local_runtime" {
		t.Fatalf("unexpected scalar-prefixed entries: %#v", entries)
	}

	if _, err := config.ListResolved("missing"); err == nil || !strings.Contains(err.Error(), "unknown repo config prefix") {
		t.Fatalf("expected unknown prefix error, got %v", err)
	}
}

func writeConfig(t *testing.T, root, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(File))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
