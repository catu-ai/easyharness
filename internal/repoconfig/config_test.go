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
