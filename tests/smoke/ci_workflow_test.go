package smoke_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

func TestCIWorkflowBuildsEmbeddedUIBeforeGoTests(t *testing.T) {
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
	support.RequireContains(t, workflow, `run: scripts/build-embedded-ui`)
	support.RequireContains(t, workflow, `run: go test ./...`)
	support.RequireSubstringOrder(t, workflow, `uses: pnpm/action-setup@v5`, `uses: actions/setup-node@v6`)
	support.RequireSubstringOrder(t, workflow, `run: scripts/build-embedded-ui`, `run: go test ./...`)
}
