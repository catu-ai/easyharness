package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/catu-ai/easyharness/internal/dashboard"
	"github.com/catu-ai/easyharness/internal/evidence"
	"github.com/catu-ai/easyharness/internal/helptopics"
	"github.com/catu-ai/easyharness/internal/install"
	"github.com/catu-ai/easyharness/internal/lifecycle"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/remote"
	"github.com/catu-ai/easyharness/internal/repoconfig"
	"github.com/catu-ai/easyharness/internal/review"
	"github.com/catu-ai/easyharness/internal/runstate"
	"github.com/catu-ai/easyharness/internal/status"
	"github.com/catu-ai/easyharness/internal/ui"
	versioninfo "github.com/catu-ai/easyharness/internal/version"
	"github.com/catu-ai/easyharness/internal/watchlist"
)

type App struct {
	Stdout      io.Writer
	Stderr      io.Writer
	Stdin       io.Reader
	Now         func() time.Time
	Getwd       func() (string, error)
	LookupEnv   func(string) (string, bool)
	UserHomeDir func() (string, error)
	Version     func() versioninfo.Info
	RunUIServer func(context.Context, ui.Server) error
	RunCommand  remote.CommandRunner

	StatusSettleTimeout      time.Duration
	StatusSettlePollInterval time.Duration
}

func New(stdout, stderr io.Writer) *App {
	return &App{
		Stdout:      stdout,
		Stderr:      stderr,
		Stdin:       os.Stdin,
		Now:         time.Now,
		Getwd:       os.Getwd,
		LookupEnv:   os.LookupEnv,
		UserHomeDir: os.UserHomeDir,
		Version:     versioninfo.Current,
		RunUIServer: func(ctx context.Context, server ui.Server) error {
			return server.Run(ctx)
		},
	}
}

func (a *App) Run(args []string) int {
	if len(args) == 0 {
		a.printRootUsage()
		return 2
	}

	switch args[0] {
	case "--version":
		return a.runVersion(args[1:])
	case "plan":
		return a.runPlan(args[1:])
	case "execute":
		return a.runExecute(args[1:])
	case "evidence":
		return a.runEvidence(args[1:])
	case "review":
		return a.runReview(args[1:])
	case "land":
		return a.runLand(args[1:])
	case "archive":
		return a.runArchive(args[1:])
	case "reopen":
		return a.runReopen(args[1:])
	case "status":
		return a.runStatus(args[1:])
	case "repo":
		return a.runRepo(args[1:])
	case "dashboard":
		return a.runDashboard(args[1:])
	case "ui":
		return a.runUI(args[1:])
	case "help":
		return a.runHelp(args[1:])
	case "-h", "--help":
		a.printRootUsage()
		return 0
	default:
		fmt.Fprintf(a.Stderr, "unknown command %q\n\n", args[0])
		a.printRootUsage()
		return 2
	}
}

func (a *App) runHelp(args []string) int {
	if len(args) == 1 {
		switch args[0] {
		case "-h", "--help", "help":
			a.printHelpUsage()
			return 0
		}
	}
	if err := helptopics.Render(a.Stdout, args); err != nil {
		var unknown helptopics.UnknownTopicError
		if errors.As(err, &unknown) {
			helptopics.WriteUnknown(a.Stderr, unknown)
			return 2
		}
		fmt.Fprintf(a.Stderr, "%v\n", err)
		return 1
	}
	return 0
}

func (a *App) runVersion(args []string) int {
	fs := flag.NewFlagSet("harness --version", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness --version")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Print JSON build information for the running harness binary.")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return 2
	}
	if a.Version == nil {
		a.Version = versioninfo.Current
	}
	encoder := json.NewEncoder(a.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(a.Version()); err != nil {
		fmt.Fprintf(a.Stderr, "write version output: %v\n", err)
		return 1
	}
	return 0
}

func (a *App) runReview(args []string) int {
	if len(args) == 0 {
		a.printReviewUsage()
		return 2
	}
	switch args[0] {
	case "start":
		return a.runReviewStart(args[1:])
	case "submit":
		return a.runReviewSubmit(args[1:])
	case "-h", "--help", "help":
		a.printReviewUsage()
		return 0
	default:
		fmt.Fprintf(a.Stderr, "unknown review subcommand %q\n\n", args[0])
		a.printReviewUsage()
		return 2
	}
}

func (a *App) runPlan(args []string) int {
	if len(args) == 0 {
		a.printPlanUsage()
		return 2
	}
	switch args[0] {
	case "template":
		return a.runPlanTemplate(args[1:])
	case "lint":
		return a.runPlanLint(args[1:])
	case "approve":
		return a.runPlanApprove(args[1:])
	case "-h", "--help", "help":
		a.printPlanUsage()
		return 0
	default:
		fmt.Fprintf(a.Stderr, "unknown plan subcommand %q\n\n", args[0])
		a.printPlanUsage()
		return 2
	}
}

func (a *App) runExecute(args []string) int {
	if len(args) == 0 {
		a.printExecuteUsage()
		return 2
	}
	switch args[0] {
	case "start":
		return a.runExecuteStart(args[1:])
	case "-h", "--help", "help":
		a.printExecuteUsage()
		return 0
	default:
		fmt.Fprintf(a.Stderr, "unknown execute subcommand %q\n\n", args[0])
		a.printExecuteUsage()
		return 2
	}
}

func (a *App) runEvidence(args []string) int {
	if len(args) == 0 {
		a.printEvidenceUsage()
		return 2
	}
	switch args[0] {
	case "submit":
		return a.runEvidenceSubmit(args[1:])
	case "refresh":
		return a.runEvidenceRefresh(args[1:])
	case "-h", "--help", "help":
		a.printEvidenceUsage()
		return 0
	default:
		fmt.Fprintf(a.Stderr, "unknown evidence subcommand %q\n\n", args[0])
		a.printEvidenceUsage()
		return 2
	}
}

func (a *App) runLand(args []string) int {
	if len(args) > 0 {
		switch args[0] {
		case "complete":
			return a.runLandComplete(args[1:])
		case "-h", "--help", "help":
			a.printLandUsage()
			return 0
		}
	}
	return a.runLandEntry(args)
}

func (a *App) runEvidenceSubmit(args []string) int {
	fs := flag.NewFlagSet("harness evidence submit", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	kind := fs.String("kind", "", "Evidence kind: ci, publish, or sync.")
	inputPath := fs.String("input", "", "Read the evidence payload JSON from this path. Defaults to stdin.")
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness evidence submit --kind <ci|publish|sync> [--input <path>]")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Record append-only CI, publish, or sync evidence for the current archived candidate.")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Schemas:")
		fmt.Fprintln(a.Stderr, `  ci:      {"status":"pending|success|failed|not_applied","provider":"optional","url":"optional","reason":"required when status=not_applied"}`)
		fmt.Fprintln(a.Stderr, `  publish: {"status":"recorded|not_applied","pr_url":"required when status=recorded","branch":"optional","base":"optional","commit":"optional","reason":"required when status=not_applied"}`)
		fmt.Fprintln(a.Stderr, `  sync:    {"status":"fresh|stale|conflicted|not_applied","base_ref":"optional","head_ref":"optional","reason":"required when status=not_applied"}`)
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Examples:")
		fmt.Fprintln(a.Stderr, `  harness evidence submit --kind ci <<'EOF'`)
		fmt.Fprintln(a.Stderr, `  {"status":"success","provider":"github-actions","url":"https://github.com/org/repo/actions/runs/123"}`)
		fmt.Fprintln(a.Stderr, `  EOF`)
		fmt.Fprintln(a.Stderr, `  harness evidence submit --kind sync <<'EOF'`)
		fmt.Fprintln(a.Stderr, `  {"status":"not_applied","reason":"repository has no shared merge target in this environment"}`)
		fmt.Fprintln(a.Stderr, `  EOF`)
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 || strings.TrimSpace(*kind) == "" {
		fs.Usage()
		return 2
	}
	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	inputBytes, err := a.readInput(*inputPath)
	if err != nil {
		fmt.Fprintf(a.Stderr, "read evidence input: %v\n", err)
		return 1
	}
	recordedAt := a.Now().Format(time.RFC3339)
	beforeStatus := readStatusSnapshot(workdir)
	result := evidence.Service{
		Workdir: workdir,
		Now:     a.Now,
		AfterMutation: evidenceTimelineHook(workdir, beforeStatus, recordedAt, *kind, map[string]any{
			"kind":  *kind,
			"input": json.RawMessage(inputBytes),
		}),
	}.Submit(*kind, inputBytes)
	return a.writeJSONResultForWorkdir(workdir, result)
}

func (a *App) runEvidenceRefresh(args []string) int {
	fs := flag.NewFlagSet("harness evidence refresh", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness evidence refresh")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Observe the recorded PR URL and append CI and sync evidence when remote facts are clear.")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return 2
	}
	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	recordedAt := a.Now().Format(time.RFC3339)
	beforeStatus := readStatusSnapshot(workdir)
	result := evidence.Service{
		Workdir:      workdir,
		Now:          a.Now,
		RunCommand:   a.RunCommand,
		AfterRefresh: evidenceRefreshTimelineHook(workdir, beforeStatus, recordedAt),
	}.Refresh()
	return a.writeJSONResultForWorkdir(workdir, result)
}

func (a *App) runLandEntry(args []string) int {
	fs := flag.NewFlagSet("harness land", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	prURL := fs.String("pr", "", "Merged PR URL.")
	commit := fs.String("commit", "", "Optional landed commit SHA.")
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness land --pr <url> [--commit <sha>]")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Record merge confirmation for the current archived candidate and enter required post-merge bookkeeping.")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 || strings.TrimSpace(*prURL) == "" {
		fs.Usage()
		return 2
	}
	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	recordedAt := a.Now().Format(time.RFC3339)
	beforeStatus := readStatusSnapshot(workdir)
	result := lifecycle.Service{
		Workdir: workdir,
		Now:     a.Now,
		AfterMutation: lifecycleTimelineHook(workdir, beforeStatus, recordedAt, map[string]any{
			"pr":     *prURL,
			"commit": strings.TrimSpace(*commit),
		}),
	}.Land(*prURL, *commit)
	return a.writeJSONResultForWorkdir(workdir, result)
}

func (a *App) runRepo(args []string) int {
	if len(args) == 0 {
		a.printRepoUsage()
		return 2
	}
	switch args[0] {
	case "init":
		return a.runRepoInit(args[1:])
	case "skills":
		return a.runRepoSkills(args[1:])
	case "instructions":
		return a.runRepoInstructions(args[1:])
	case "config":
		return a.runRepoConfig(args[1:])
	case "-h", "--help", "help":
		a.printRepoUsage()
		return 0
	default:
		fmt.Fprintf(a.Stderr, "unknown repo subcommand %q\n\n", args[0])
		a.printRepoUsage()
		return 2
	}
}

func (a *App) runRepoInit(args []string) int {
	fs := flag.NewFlagSet("harness repo init", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	agent := fs.String("agent", "", "Agent profile name used for default targets. Defaults to codex.")
	skillsDir := fs.String("skills-dir", "", "Override the skills target directory.")
	instructionsFile := fs.String("instructions-file", "", "Override the instructions target file.")
	dryRun := fs.Bool("dry-run", false, "Show the planned repository changes without writing files.")
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness repo init [--agent <name>] [--skills-dir <path>] [--instructions-file <path>] [--dry-run]")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Install or refresh the managed repo instructions, skill pack, and config manifest.")
		fmt.Fprintln(a.Stderr)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return 2
	}
	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	result := install.Service{Workdir: workdir}.Init(install.Options{
		Agent:            *agent,
		SkillsDir:        *skillsDir,
		InstructionsFile: *instructionsFile,
		DryRun:           *dryRun,
	})
	return a.writeJSONResult(result)
}

func (a *App) runRepoSkills(args []string) int {
	if len(args) == 0 {
		a.printRepoSkillsUsage()
		return 2
	}
	switch args[0] {
	case "install":
		return a.runRepoSkillsInstall(args[1:])
	case "uninstall":
		return a.runRepoSkillsUninstall(args[1:])
	case "-h", "--help", "help":
		a.printRepoSkillsUsage()
		return 0
	default:
		fmt.Fprintf(a.Stderr, "unknown repo skills subcommand %q\n\n", args[0])
		a.printRepoSkillsUsage()
		return 2
	}
}

func (a *App) runRepoInstructions(args []string) int {
	if len(args) == 0 {
		a.printRepoInstructionsUsage()
		return 2
	}
	switch args[0] {
	case "install":
		return a.runRepoInstructionsInstall(args[1:])
	case "uninstall":
		return a.runRepoInstructionsUninstall(args[1:])
	case "-h", "--help", "help":
		a.printRepoInstructionsUsage()
		return 0
	default:
		fmt.Fprintf(a.Stderr, "unknown repo instructions subcommand %q\n\n", args[0])
		a.printRepoInstructionsUsage()
		return 2
	}
}

func (a *App) runRepoConfig(args []string) int {
	if len(args) == 0 {
		a.printRepoConfigUsage()
		return 2
	}
	switch args[0] {
	case "init":
		return a.runRepoConfigInit(args[1:])
	case "refresh":
		return a.runRepoConfigRefresh(args[1:])
	case "get":
		return a.runRepoConfigGet(args[1:])
	case "list":
		return a.runRepoConfigList(args[1:])
	case "-h", "--help", "help":
		a.printRepoConfigUsage()
		return 0
	default:
		fmt.Fprintf(a.Stderr, "unknown repo config subcommand %q\n\n", args[0])
		a.printRepoConfigUsage()
		return 2
	}
}

func (a *App) runRepoSkillsInstall(args []string) int {
	return a.runSkillsCommand("harness repo skills install", args, true)
}

func (a *App) runRepoSkillsUninstall(args []string) int {
	return a.runSkillsCommand("harness repo skills uninstall", args, false)
}

func (a *App) runSkillsCommand(name string, args []string, installOp bool) int {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	scope := fs.String("scope", install.ScopeRepo, "Skills scope: repo or user.")
	agent := fs.String("agent", "", "Agent profile name used for default targets. Defaults to codex.")
	dir := fs.String("dir", "", "Override the skills target directory.")
	dryRun := fs.Bool("dry-run", false, "Show the planned changes without writing files.")
	fs.Usage = func() {
		fmt.Fprintf(a.Stderr, "Usage: %s [--scope <repo|user>] [--agent <name>] [--dir <path>] [--dry-run]\n", name)
		fmt.Fprintln(a.Stderr)
		if installOp {
			fmt.Fprintln(a.Stderr, "Install or refresh the managed bootstrap skill pack.")
		} else {
			fmt.Fprintln(a.Stderr, "Remove easyharness-managed skill packages from the resolved target directory.")
		}
		fmt.Fprintln(a.Stderr)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return 2
	}
	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	service := install.Service{Workdir: workdir}
	opts := install.Options{Scope: *scope, Agent: *agent, SkillsDir: *dir, DryRun: *dryRun}
	if installOp {
		return a.writeJSONResult(service.InstallSkills(opts))
	}
	return a.writeJSONResult(service.UninstallSkills(opts))
}

func (a *App) runRepoInstructionsInstall(args []string) int {
	return a.runInstructionsCommand("harness repo instructions install", args, true)
}

func (a *App) runRepoInstructionsUninstall(args []string) int {
	return a.runInstructionsCommand("harness repo instructions uninstall", args, false)
}

func (a *App) runInstructionsCommand(name string, args []string, installOp bool) int {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	scope := fs.String("scope", install.ScopeRepo, "Instructions scope: repo or user.")
	agent := fs.String("agent", "", "Agent profile name used for default targets. Defaults to codex.")
	file := fs.String("file", "", "Override the instructions target file.")
	dir := fs.String("dir", "", "Override the paired skills directory used when rendering the managed block.")
	dryRun := fs.Bool("dry-run", false, "Show the planned changes without writing files.")
	fs.Usage = func() {
		if installOp {
			fmt.Fprintf(a.Stderr, "Usage: %s [--scope <repo|user>] [--agent <name>] [--file <path>] [--dir <path>] [--dry-run]\n", name)
		} else {
			fmt.Fprintf(a.Stderr, "Usage: %s [--scope <repo|user>] [--agent <name>] [--file <path>] [--dry-run]\n", name)
		}
		fmt.Fprintln(a.Stderr)
		if installOp {
			fmt.Fprintln(a.Stderr, "Install or refresh the easyharness-managed bootstrap block in the target instructions file.")
		} else {
			fmt.Fprintln(a.Stderr, "Remove the easyharness-managed bootstrap block from the target instructions file.")
		}
		fmt.Fprintln(a.Stderr)
		fmt.Fprintf(a.Stderr, "  -agent string\n")
		fmt.Fprintln(a.Stderr, "        Agent profile name used for default targets. Defaults to codex.")
		fmt.Fprintf(a.Stderr, "  -dry-run\n")
		fmt.Fprintln(a.Stderr, "        Show the planned changes without writing files.")
		fmt.Fprintf(a.Stderr, "  -file string\n")
		fmt.Fprintln(a.Stderr, "        Override the instructions target file.")
		if installOp {
			fmt.Fprintf(a.Stderr, "  -dir string\n")
			fmt.Fprintln(a.Stderr, "        Override the paired skills directory used when rendering the managed block.")
		}
		fmt.Fprintf(a.Stderr, "  -scope string\n")
		fmt.Fprintf(a.Stderr, "        Instructions scope: %s or %s. (default %q)\n", install.ScopeRepo, install.ScopeUser, install.ScopeRepo)
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return 2
	}
	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	service := install.Service{Workdir: workdir}
	opts := install.Options{Scope: *scope, Agent: *agent, SkillsDir: *dir, InstructionsFile: *file, DryRun: *dryRun}
	if installOp {
		return a.writeJSONResult(service.InstallInstructions(opts))
	}
	return a.writeJSONResult(service.UninstallInstructions(opts))
}

func (a *App) runRepoConfigInit(args []string) int {
	fs := flag.NewFlagSet("harness repo config init", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	dryRun := fs.Bool("dry-run", false, "Show the planned repo config change without writing files.")
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness repo config init [--dry-run]")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Create the canonical .harness/config.yaml manifest when it is missing.")
		fmt.Fprintln(a.Stderr)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return 2
	}
	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	result := install.Service{Workdir: workdir}.InitConfig(install.Options{DryRun: *dryRun})
	return a.writeJSONResult(result)
}

func (a *App) runRepoConfigRefresh(args []string) int {
	fs := flag.NewFlagSet("harness repo config refresh", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	diff := fs.Bool("diff", false, "Print the unified diff refresh would apply without writing files.")
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness repo config refresh [--diff]")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Create or refresh .harness/config.yaml to the current canonical shape.")
		fmt.Fprintln(a.Stderr)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return 2
	}
	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	if *diff {
		service := install.Service{Workdir: workdir}
		diffText, errs := service.PlanConfigRefreshDiff()
		if len(errs) > 0 {
			return a.writeJSONResult(service.RefreshConfigPreviewError(errs))
		}
		fmt.Fprint(a.Stdout, diffText)
		return 0
	}
	result := install.Service{Workdir: workdir}.RefreshConfig(install.Options{})
	return a.writeJSONResult(result)
}

func (a *App) runRepoConfigGet(args []string) int {
	fs := flag.NewFlagSet("harness repo config get", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness repo config get <key>")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Print one resolved scalar repo config value.")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}
	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	result := repoconfig.Load(workdir)
	a.printRepoConfigWarnings(result.Warnings)
	value, err := result.Config.GetScalar(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(a.Stderr, err)
		return 1
	}
	fmt.Fprintln(a.Stdout, value)
	return 0
}

func (a *App) runRepoConfigList(args []string) int {
	fs := flag.NewFlagSet("harness repo config list", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness repo config list [prefix]")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Print resolved repo config leaf entries as key=value lines.")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fs.Usage()
		return 2
	}
	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	prefix := ""
	if fs.NArg() == 1 {
		prefix = fs.Arg(0)
	}
	result := repoconfig.Load(workdir)
	a.printRepoConfigWarnings(result.Warnings)
	entries, err := result.Config.ListResolved(prefix)
	if err != nil {
		fmt.Fprintln(a.Stderr, err)
		return 1
	}
	for _, entry := range entries {
		fmt.Fprintf(a.Stdout, "%s=%s\n", entry.Key, entry.Value)
	}
	return 0
}

func (a *App) printRepoConfigWarnings(warnings []string) {
	for _, warning := range warnings {
		fmt.Fprintln(a.Stderr, warning)
	}
}

func (a *App) runDashboard(args []string) int {
	return a.runUIServerCommand("dashboard", args, func() (string, error) {
		return "/dashboard", nil
	})
}

func (a *App) runUI(args []string) int {
	return a.runUIServerCommand("ui", args, func() (string, error) {
		workdir, err := a.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve working directory: %w", err)
		}
		if err := (watchlist.Service{
			LookupEnv:   a.LookupEnv,
			UserHomeDir: a.UserHomeDir,
		}).Touch(workdir); err != nil {
			return "", fmt.Errorf("touch watchlist for current workspace: %w", err)
		}
		canonicalPath, err := watchlist.CanonicalWorkspacePath(workdir)
		if err != nil {
			return "", fmt.Errorf("resolve canonical workspace path: %w", err)
		}
		return fmt.Sprintf("/workspace/%s/status", dashboard.WorkspaceKey(canonicalPath)), nil
	})
}

func (a *App) runUIServerCommand(command string, args []string, resolveOpenPath func() (string, error)) int {
	fs := flag.NewFlagSet("harness "+command, flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	host := fs.String("host", "127.0.0.1", "Bind the local UI server to this host.")
	port := fs.Int("port", 0, "Bind the local UI server to this port. Use 0 to auto-select an available port.")
	noOpen := fs.Bool("no-open", false, "Start the local UI server without opening a browser.")
	fs.Usage = func() {
		fmt.Fprintf(a.Stderr, "Usage: harness %s [--host <host>] [--port <port>] [--no-open]\n", command)
		fmt.Fprintln(a.Stderr)
		switch command {
		case "dashboard":
			fmt.Fprintln(a.Stderr, "Start the local machine-local dashboard home.")
		default:
			fmt.Fprintln(a.Stderr, "Start the local read-only harness UI workbench for the current repository.")
		}
		fmt.Fprintln(a.Stderr)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return 2
	}
	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	openPath, err := resolveOpenPath()
	if err != nil {
		fmt.Fprintf(a.Stderr, "%v\n", err)
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err = a.RunUIServer(ctx, ui.Server{
		Workdir:     workdir,
		Host:        *host,
		Port:        *port,
		Stdout:      a.Stdout,
		Stderr:      a.Stderr,
		OpenBrowser: !*noOpen,
		OpenPath:    openPath,
	})
	if err != nil {
		fmt.Fprintf(a.Stderr, "run harness %s: %v\n", command, err)
		return 1
	}
	return 0
}

func (a *App) runPlanTemplate(args []string) int {
	fs := flag.NewFlagSet("harness plan template", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)

	var refs stringListFlag
	title := fs.String("title", "", "Seed the H1 title.")
	output := fs.String("output", "", "Write the rendered template to this file instead of stdout.")
	lightweight := fs.Bool("lightweight", false, "Render the lightweight variant and seed workflow_profile: lightweight.")
	dateValue := fs.String("date", "", "Seed timestamps using this YYYY-MM-DD date with the current local time-of-day.")
	timestampValue := fs.String("timestamp", "", "Seed timestamps using this RFC3339 timestamp.")
	sourceType := fs.String("source-type", "direct_request", "Seed the frontmatter source_type field.")
	size := fs.String("size", "", "Seed the required frontmatter size field (XXS, XS, S, M, L, XL, or XXL).")
	fs.Var(&refs, "source-ref", "Seed one source_refs entry. Repeat to add multiple refs.")
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness plan template [flags]")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Render the packaged plan template with seeded title, timestamp, source metadata, and size.")
		fmt.Fprintln(a.Stderr)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(a.Stderr, "harness plan template does not accept positional arguments")
		return 2
	}
	ts, err := a.resolveTimestamp(*timestampValue, *dateValue)
	if err != nil {
		fmt.Fprintln(a.Stderr, err.Error())
		return 2
	}

	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      *title,
		Timestamp:  ts,
		SourceType: *sourceType,
		SourceRefs: refs,
		Size:       *size,
		WorkflowProfile: func() string {
			if *lightweight {
				return plan.WorkflowProfileLightweight
			}
			return ""
		}(),
	})
	if err != nil {
		fmt.Fprintf(a.Stderr, "render template: %v\n", err)
		return 1
	}

	if *output == "" {
		_, _ = io.WriteString(a.Stdout, rendered)
		return 0
	}

	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fmt.Fprintf(a.Stderr, "create parent directory: %v\n", err)
		return 1
	}
	if err := os.WriteFile(*output, []byte(rendered), 0o644); err != nil {
		fmt.Fprintf(a.Stderr, "write template: %v\n", err)
		return 1
	}
	fmt.Fprintf(a.Stdout, "Wrote plan template to %s\n", *output)
	return 0
}

func (a *App) runPlanLint(args []string) int {
	fs := flag.NewFlagSet("harness plan lint", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness plan lint <plan-path>")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Validate a tracked plan and emit compact machine-readable results.")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}

	result := plan.LintFile(fs.Arg(0))
	encoder := json.NewEncoder(a.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(a.Stderr, "encode lint result: %v\n", err)
		return 1
	}
	if result.OK {
		return 0
	}
	return 1
}

func (a *App) runPlanApprove(args []string) int {
	fs := flag.NewFlagSet("harness plan approve", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	by := fs.String("by", "", "Approval source. Use human after the human explicitly approves execution.")
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness plan approve --by human")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Record the explicit human approval boundary for the current active plan before execution starts.")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 || strings.TrimSpace(*by) == "" {
		fs.Usage()
		return 2
	}

	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	recordedAt := a.Now().Format(time.RFC3339)
	beforeStatus := readStatusSnapshot(workdir)
	result := lifecycle.Service{
		Workdir: workdir,
		Now:     a.Now,
		AfterMutation: lifecycleTimelineHook(workdir, beforeStatus, recordedAt, map[string]any{
			"by": strings.TrimSpace(*by),
		}),
	}.PlanApprove(*by)
	return a.writeJSONResultForWorkdir(workdir, result)
}

func (a *App) runStatus(args []string) int {
	fs := flag.NewFlagSet("harness status", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness status")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Summarize the current plan plus local execution state for the current worktree.")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return 2
	}

	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}

	if result, settled := a.settleStatusSnapshot(workdir); !settled {
		return a.writeJSONResultForWorkdir(workdir, result)
	}
	result := status.Service{
		Workdir:       workdir,
		ObserveRemote: true,
		RunCommand:    a.RunCommand,
	}.Snapshot()
	return a.writeJSONResultForWorkdir(workdir, result)
}

func (a *App) settleStatusSnapshot(workdir string) (status.Result, bool) {
	planPath, err := plan.DetectCurrentPath(workdir)
	if err != nil {
		return status.Result{}, true
	}
	planStem := strings.TrimSuffix(filepath.Base(planPath), filepath.Ext(planPath))
	timeout := a.StatusSettleTimeout
	if timeout == 0 {
		timeout = 2 * time.Second
	}
	pollInterval := a.StatusSettlePollInterval
	if pollInterval == 0 {
		pollInterval = 25 * time.Millisecond
	}
	held, err := runstate.WaitForStateMutationLockRelease(workdir, planStem, timeout, pollInterval)
	if err != nil {
		return status.Result{
			OK:      false,
			Command: "status",
			Summary: "Unable to inspect local state mutation lock.",
			Artifacts: &status.Artifacts{
				ProjectRoot: workdir,
				PlanPath:    normalizeRepoPath(workdir, planPath),
			},
			Errors: []status.StatusError{{Path: "state", Message: "Unable to inspect local state mutation lock."}},
		}, false
	}
	if held {
		return status.Result{
			OK:      false,
			Command: "status",
			Summary: "Another local state mutation is still in progress.",
			Artifacts: &status.Artifacts{
				ProjectRoot: workdir,
				PlanPath:    normalizeRepoPath(workdir, planPath),
			},
			Errors: []status.StatusError{{Path: "state", Message: "Another local state mutation is still in progress."}},
		}, false
	}
	return status.Result{}, true
}

func (a *App) runReviewStart(args []string) int {
	fs := flag.NewFlagSet("harness review start", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	forceFull := fs.Bool("full", false, "Start a new full coverage root instead of the inferred linked delta.")
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness review start [--full]")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Create the mandatory integrated finalize review round.")
		fmt.Fprintln(a.Stderr, "The first round is full; later rounds infer a linked delta unless --full deliberately resets coverage.")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return 2
	}
	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	recordedAt := a.Now().Format(time.RFC3339)
	beforeStatus := readStatusSnapshot(workdir)
	result := review.Service{
		Workdir:    workdir,
		Now:        a.Now,
		AfterStart: reviewStartTimelineHook(workdir, beforeStatus, recordedAt, map[string]any{"force_full": *forceFull}),
	}.Start(review.StartOptions{ForceFull: *forceFull})
	return a.writeJSONResultForWorkdir(workdir, result)
}

func (a *App) runReviewSubmit(args []string) int {
	fs := flag.NewFlagSet("harness review submit", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	roundID := fs.String("round", "", "Review round ID.")
	by := fs.String("by", "", "Reviewer identity label for this submission.")
	inputPath := fs.String("input", "", "Read the reviewer submission JSON from this path. Defaults to stdin.")
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness review submit --round <round-id> --by <reviewer-name> [--input <path>]")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Record the integrated reviewer submission and complete the selected review round.")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 || strings.TrimSpace(*roundID) == "" || strings.TrimSpace(*by) == "" {
		fs.Usage()
		return 2
	}
	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	inputBytes, err := a.readInput(*inputPath)
	if err != nil {
		fmt.Fprintf(a.Stderr, "read reviewer submission: %v\n", err)
		return 1
	}
	recordedAt := a.Now().Format(time.RFC3339)
	beforeStatus := readStatusSnapshot(workdir)
	result := review.Service{
		Workdir: workdir,
		Now:     a.Now,
		AfterSubmit: reviewSubmitTimelineHook(workdir, beforeStatus, recordedAt, map[string]any{
			"by":    strings.TrimSpace(*by),
			"input": json.RawMessage(inputBytes),
		}),
	}.Submit(*roundID, *by, inputBytes)
	return a.writeJSONResultForWorkdir(workdir, result)
}

func (a *App) runExecuteStart(args []string) int {
	fs := flag.NewFlagSet("harness execute start", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness execute start")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Record the explicit execution-start milestone for the current active plan.")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return 2
	}
	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	recordedAt := a.Now().Format(time.RFC3339)
	beforeStatus := readStatusSnapshot(workdir)
	result := lifecycle.Service{
		Workdir:       workdir,
		Now:           a.Now,
		AfterMutation: lifecycleTimelineHook(workdir, beforeStatus, recordedAt, nil),
	}.ExecuteStart()
	return a.writeJSONResultForWorkdir(workdir, result)
}

func (a *App) runLandComplete(args []string) int {
	fs := flag.NewFlagSet("harness land complete", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness land complete")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Record that required post-merge bookkeeping is complete and restore idle worktree state.")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return 2
	}
	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	recordedAt := a.Now().Format(time.RFC3339)
	beforeStatus := readStatusSnapshot(workdir)
	result := lifecycle.Service{
		Workdir:       workdir,
		Now:           a.Now,
		AfterMutation: lifecycleTimelineHook(workdir, beforeStatus, recordedAt, nil),
	}.LandComplete()
	return a.writeJSONResultForWorkdir(workdir, result)
}

func (a *App) runArchive(args []string) int {
	fs := flag.NewFlagSet("harness archive", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness archive")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Freeze the current active plan for merge handoff.")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return 2
	}
	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	recordedAt := a.Now().Format(time.RFC3339)
	beforeStatus := readStatusSnapshot(workdir)
	result := lifecycle.Service{
		Workdir:       workdir,
		Now:           a.Now,
		AfterMutation: lifecycleTimelineHook(workdir, beforeStatus, recordedAt, nil),
	}.Archive()
	return a.writeJSONResultForWorkdir(workdir, result)
}

func (a *App) runReopen(args []string) int {
	fs := flag.NewFlagSet("harness reopen", flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	mode := fs.String("mode", "", "Reopen mode: finalize-fix or new-step.")
	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: harness reopen --mode <finalize-fix|new-step>")
		fmt.Fprintln(a.Stderr)
		fmt.Fprintln(a.Stderr, "Restore the current archived plan to active execution.")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 || strings.TrimSpace(*mode) == "" {
		fs.Usage()
		return 2
	}
	workdir, err := a.Getwd()
	if err != nil {
		fmt.Fprintf(a.Stderr, "resolve working directory: %v\n", err)
		return 1
	}
	recordedAt := a.Now().Format(time.RFC3339)
	beforeStatus := readStatusSnapshot(workdir)
	result := lifecycle.Service{
		Workdir:       workdir,
		Now:           a.Now,
		AfterMutation: lifecycleTimelineHook(workdir, beforeStatus, recordedAt, map[string]any{"mode": *mode}),
	}.Reopen(*mode)
	return a.writeJSONResultForWorkdir(workdir, result)
}

func (a *App) resolveTimestamp(timestampValue, dateValue string) (time.Time, error) {
	if strings.TrimSpace(timestampValue) != "" {
		ts, err := time.Parse(time.RFC3339, timestampValue)
		if err != nil {
			return time.Time{}, fmt.Errorf("--timestamp must be RFC3339: %w", err)
		}
		return ts, nil
	}
	if strings.TrimSpace(dateValue) != "" {
		now := a.Now()
		location := now.Location()
		day, err := time.ParseInLocation("2006-01-02", dateValue, location)
		if err != nil {
			return time.Time{}, fmt.Errorf("--date must be YYYY-MM-DD: %w", err)
		}
		return time.Date(day.Year(), day.Month(), day.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), location), nil
	}
	return a.Now(), nil
}

func (a *App) printRootUsage() {
	fmt.Fprintln(a.Stderr, "Usage: harness <command> [subcommand] [flags]")
	fmt.Fprintln(a.Stderr)
	fmt.Fprintln(a.Stderr, "Flags:")
	fmt.Fprintln(a.Stderr, "  --version       Print JSON build information for the running harness binary")
	fmt.Fprintln(a.Stderr)
	fmt.Fprintln(a.Stderr, "Commands:")
	fmt.Fprintln(a.Stderr, "  plan template   Render the packaged plan template")
	fmt.Fprintln(a.Stderr, "  plan lint       Validate a tracked plan")
	fmt.Fprintln(a.Stderr, "  plan approve    Record explicit human approval for the current plan")
	fmt.Fprintln(a.Stderr, "  execute start   Record the execution-start milestone")
	fmt.Fprintln(a.Stderr, "  evidence submit Record append-only CI, publish, or sync evidence")
	fmt.Fprintln(a.Stderr, "  evidence refresh Refresh CI and sync evidence from a recorded PR")
	fmt.Fprintln(a.Stderr, "  review start    Create a deterministic review round")
	fmt.Fprintln(a.Stderr, "  review submit   Record the integrated reviewer decision")
	fmt.Fprintln(a.Stderr, "  land            Record merge confirmation and start required post-merge bookkeeping")
	fmt.Fprintln(a.Stderr, "  land complete   Record required post-merge bookkeeping completion")
	fmt.Fprintln(a.Stderr, "  archive         Freeze the current active plan")
	fmt.Fprintln(a.Stderr, "  reopen          Restore the current archived plan")
	fmt.Fprintln(a.Stderr, "  status          Summarize the current plan and local execution state")
	fmt.Fprintln(a.Stderr, "  repo            Manage repo-level easyharness resources")
	fmt.Fprintln(a.Stderr, "  help            Read agent-facing product guidance topics")
	fmt.Fprintln(a.Stderr, "  dashboard       Start the local machine-local dashboard home")
	fmt.Fprintln(a.Stderr, "  ui              Start the local read-only harness UI workbench")
}

func (a *App) printHelpUsage() {
	fmt.Fprintln(a.Stderr, "Usage: harness help [topic ...]")
	fmt.Fprintln(a.Stderr)
	fmt.Fprintln(a.Stderr, "Print plain-text agent-facing product guidance topics.")
	fmt.Fprintln(a.Stderr)
	helptopics.Render(a.Stderr, nil)
}

func (a *App) printPlanUsage() {
	fmt.Fprintln(a.Stderr, "Usage: harness plan <subcommand> [flags]")
	fmt.Fprintln(a.Stderr)
	fmt.Fprintln(a.Stderr, "Subcommands:")
	fmt.Fprintln(a.Stderr, "  template   Render the packaged plan template")
	fmt.Fprintln(a.Stderr, "  lint       Validate a tracked plan")
	fmt.Fprintln(a.Stderr, "  approve    Record explicit human approval for the current plan")
}

func (a *App) printReviewUsage() {
	fmt.Fprintln(a.Stderr, "Usage: harness review <subcommand> [flags]")
	fmt.Fprintln(a.Stderr)
	fmt.Fprintln(a.Stderr, "Subcommands:")
	fmt.Fprintln(a.Stderr, "  start      Create a deterministic review round")
	fmt.Fprintln(a.Stderr, "  submit     Record the integrated reviewer decision")
}

func (a *App) printExecuteUsage() {
	fmt.Fprintln(a.Stderr, "Usage: harness execute <subcommand> [flags]")
	fmt.Fprintln(a.Stderr)
	fmt.Fprintln(a.Stderr, "Subcommands:")
	fmt.Fprintln(a.Stderr, "  start      Record the explicit execution-start milestone")
}

func (a *App) printEvidenceUsage() {
	fmt.Fprintln(a.Stderr, "Usage: harness evidence <subcommand> [flags]")
	fmt.Fprintln(a.Stderr)
	fmt.Fprintln(a.Stderr, "Subcommands:")
	fmt.Fprintln(a.Stderr, "  submit     Record append-only CI, publish, or sync evidence")
	fmt.Fprintln(a.Stderr, "  refresh    Refresh CI and sync evidence from a recorded PR")
}

func (a *App) printRepoUsage() {
	fmt.Fprintln(a.Stderr, "Usage: harness repo <subcommand> [flags]")
	fmt.Fprintln(a.Stderr)
	fmt.Fprintln(a.Stderr, "Subcommands:")
	fmt.Fprintln(a.Stderr, "  init          Install or refresh repo instructions, skills, and config")
	fmt.Fprintln(a.Stderr, "  skills        Manage easyharness-managed skill packages")
	fmt.Fprintln(a.Stderr, "  instructions  Manage easyharness instruction files and managed blocks")
	fmt.Fprintln(a.Stderr, "  config        Manage the .harness/config.yaml manifest")
	fmt.Fprintln(a.Stderr)
	fmt.Fprintln(a.Stderr, "See also:")
	fmt.Fprintln(a.Stderr, "  harness help repo config")
}

func (a *App) printRepoSkillsUsage() {
	fmt.Fprintln(a.Stderr, "Usage: harness repo skills <subcommand> [flags]")
	fmt.Fprintln(a.Stderr)
	fmt.Fprintln(a.Stderr, "Subcommands:")
	fmt.Fprintln(a.Stderr, "  install    Install or refresh easyharness-managed skill packages")
	fmt.Fprintln(a.Stderr, "  uninstall  Remove easyharness-managed skill packages")
}

func (a *App) printRepoInstructionsUsage() {
	fmt.Fprintln(a.Stderr, "Usage: harness repo instructions <subcommand> [flags]")
	fmt.Fprintln(a.Stderr)
	fmt.Fprintln(a.Stderr, "Subcommands:")
	fmt.Fprintln(a.Stderr, "  install    Install or refresh the easyharness-managed bootstrap block")
	fmt.Fprintln(a.Stderr, "  uninstall  Remove the easyharness-managed bootstrap block")
}

func (a *App) printRepoConfigUsage() {
	fmt.Fprintln(a.Stderr, "Usage: harness repo config <subcommand> [flags]")
	fmt.Fprintln(a.Stderr)
	fmt.Fprintln(a.Stderr, "Subcommands:")
	fmt.Fprintln(a.Stderr, "  init       Create the canonical .harness/config.yaml manifest when missing")
	fmt.Fprintln(a.Stderr, "  refresh    Refresh .harness/config.yaml to the current canonical shape")
	fmt.Fprintln(a.Stderr, "  get        Print one resolved scalar repo config value")
	fmt.Fprintln(a.Stderr, "  list       Print resolved repo config leaf entries")
	fmt.Fprintln(a.Stderr)
	fmt.Fprintln(a.Stderr, "See also:")
	fmt.Fprintln(a.Stderr, "  harness help repo config")
}

func (a *App) printLandUsage() {
	fmt.Fprintln(a.Stderr, "Usage: harness land --pr <url> [--commit <sha>]")
	fmt.Fprintln(a.Stderr, "   or: harness land complete")
	fmt.Fprintln(a.Stderr)
	fmt.Fprintln(a.Stderr, "Commands:")
	fmt.Fprintln(a.Stderr, "  land            Record merge confirmation and enter required post-merge bookkeeping")
	fmt.Fprintln(a.Stderr, "  land complete   Record required post-merge bookkeeping completion and restore idle")
}

type stringListFlag []string

func (f *stringListFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *stringListFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func (a *App) touchWatchlist(workdir string) {
	service := watchlist.Service{
		LookupEnv:   a.LookupEnv,
		UserHomeDir: a.UserHomeDir,
		Now:         a.Now,
	}
	_ = service.Touch(workdir)
}

func (a *App) writeJSONResultForWorkdir(workdir string, value any) int {
	exitCode := a.writeJSONResult(value)
	if exitCode == 0 && watchlistTouchEnabled(value) {
		a.touchWatchlist(workdir)
	}
	return exitCode
}

func (a *App) readInput(path string) ([]byte, error) {
	if strings.TrimSpace(path) != "" {
		return os.ReadFile(path)
	}
	if a.Stdin == nil {
		return nil, fmt.Errorf("stdin is unavailable")
	}
	return io.ReadAll(a.Stdin)
}

func (a *App) writeJSONResult(value any) int {
	encoder := json.NewEncoder(a.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintf(a.Stderr, "encode JSON result: %v\n", err)
		return 1
	}

	switch result := value.(type) {
	case plan.LintResult:
		if result.OK {
			return 0
		}
	case status.Result:
		if result.OK {
			return 0
		}
	case review.StartResult:
		if result.OK {
			return 0
		}
	case review.SubmitResult:
		if result.OK {
			return 0
		}
	case evidence.Result:
		if result.OK {
			return 0
		}
	case evidence.RefreshResult:
		if result.OK {
			return 0
		}
	case lifecycle.Result:
		if result.OK {
			return 0
		}
	case install.Result:
		if result.OK {
			return 0
		}
	}
	return 1
}

func watchlistTouchEnabled(value any) bool {
	switch value.(type) {
	case evidence.Result, evidence.RefreshResult, lifecycle.Result, review.StartResult, review.SubmitResult, status.Result:
		return true
	default:
		return false
	}
}
