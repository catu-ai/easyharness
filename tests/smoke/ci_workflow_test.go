package smoke_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

func TestCIWorkflowUsesDevelopmentValidationProfile(t *testing.T) {
	repoRoot := support.RepoRoot(t)

	workflowData, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}
	workflow := string(workflowData)

	support.RequireContains(t, workflow, `uses: actions/checkout@v6`)
	support.RequireContains(t, workflow, `uses: pnpm/action-setup@v5`)
	support.RequireContains(t, workflow, `version: 10.32.1`)
	support.RequireContains(t, workflow, `run_install: false`)
	support.RequireContains(t, workflow, `uses: actions/setup-node@v6`)
	support.RequireContains(t, workflow, `node-version: "22"`)
	support.RequireContains(t, workflow, `cache: pnpm`)
	support.RequireContains(t, workflow, `cache-dependency-path: web/pnpm-lock.yaml`)
	support.RequireContains(t, workflow, `run: corepack enable`)
	support.RequireContains(t, workflow, `uses: actions/setup-go@v6`)
	support.RequireContains(t, workflow, `run: scripts/validate`)
	support.RequireSubstringOrder(t, workflow, `uses: pnpm/action-setup@v5`, `uses: actions/setup-node@v6`)
}

func TestValidationScriptsDefineDevelopmentAndReleaseProfiles(t *testing.T) {
	repoRoot := support.RepoRoot(t)

	validatePath := filepath.Join(repoRoot, "scripts", "validate")
	validateInfo, err := os.Stat(validatePath)
	if err != nil {
		t.Fatalf("stat scripts/validate: %v", err)
	}
	if validateInfo.Mode()&0o111 == 0 {
		t.Fatalf("expected scripts/validate to be executable, mode %s", validateInfo.Mode())
	}
	validateData, err := os.ReadFile(validatePath)
	if err != nil {
		t.Fatalf("read scripts/validate: %v", err)
	}
	validate := string(validateData)
	support.RequireContains(t, validate, "scripts/build-embedded-ui")
	support.RequireContains(t, validate, "go test ./...")

	validateReleasePath := filepath.Join(repoRoot, "scripts", "validate-release")
	validateReleaseInfo, err := os.Stat(validateReleasePath)
	if err != nil {
		t.Fatalf("stat scripts/validate-release: %v", err)
	}
	if validateReleaseInfo.Mode()&0o111 == 0 {
		t.Fatalf("expected scripts/validate-release to be executable, mode %s", validateReleaseInfo.Mode())
	}
	validateReleaseData, err := os.ReadFile(validateReleasePath)
	if err != nil {
		t.Fatalf("read scripts/validate-release: %v", err)
	}
	validateRelease := string(validateReleaseData)
	support.RequireContains(t, validateRelease, "scripts/validate")
	support.RequireContains(t, validateRelease, "go test -tags installer_smoke ./tests/installer -count=1")
	support.RequireContains(t, validateRelease, "go test -tags release_smoke ./tests/release -count=1")
	support.RequireSubstringOrder(t, validateRelease, "scripts/validate", "installer_smoke")
	support.RequireSubstringOrder(t, validateRelease, "installer_smoke", "release_smoke")
}
