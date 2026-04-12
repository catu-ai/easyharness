# easyharness

Harnesses matter. Building one shouldn't be the project.

`easyharness` is a git-native, agent-first harness for human-steered,
agent-executed work. It packages repo-local instructions, plans, workflow
state, review, and archive paths into a thin system that coding agents can
actually follow, so humans can steer work without maintaining a heavy custom
harness from scratch.

The project is named `easyharness`. The CLI executable remains `harness`.

## Why easyharness

Long-running agent work gets fragile when plans, state, and rules live only in
chat history or ad hoc shell scripts. Agents lose coherence, humans end up
micromanaging implementation details, and the workflow becomes hard to inspect
or teach to the next run.

`easyharness` keeps the important parts of the workflow in the repository:

- repo-local instructions and skills that tell the agent how to work here
- git-tracked plans and durable summaries that survive context loss
- command-owned workflow state, review artifacts, and evidence under `.local/`
- a steering surface built around plans, status, and execution summaries

The goal is not to make humans read every diff by default. The goal is to make
it easy for humans to set direction, approve intent, inspect high-signal
artifacts, and intervene when judgment is needed.

## Quickstart

Install `easyharness` with Homebrew:

```bash
brew tap catu-ai/tap
brew install easyharness
```

Bootstrap a repository:

```bash
cd /path/to/your-repo
harness init
```

`harness init` installs the managed `AGENTS.md` block and repo-local
`.agents/skills/` pack that tell your coding agent how to work in that
repository. After running it, restart your coding agent so it picks up the new
instructions and skills cleanly. In practice, that is the point where the
repository starts telling the agent what it needs to know.

When you or the agent need the current workflow position, use:

```bash
harness status
```

## Stability

`easyharness` is evolving quickly, and breaking changes may happen between
releases.

That does not mean the human operator needs to track every internal workflow
detail by hand. The harness is designed so agents can recover the relevant
context from repo-local instructions, plans, workflow state, and skills, then
continue the work intentionally.

The product should keep evolving in the same direction:

- reduce agent cognitive load
- improve execution quality and legibility
- help humans steer the work without micromanaging it

## How Humans Steer

In an `easyharness` repository, the human role is to steer the work, not to
micromanage every implementation step.

In practice that means:

- define intent, scope, constraints, and non-goals
- approve or adjust plans before execution starts
- review execution summaries, outcomes, and high-signal artifacts
- step in when product, risk, or judgment calls matter
- avoid treating success as manually reviewing every changed line by default

The repository workflow is built around that posture:

1. Discovery
2. Plan
3. Execute
4. Archive / publish / await merge approval
5. Land

## Workflow Surface

The current v0.2 harness surface centers on a few core ideas:

- tracked plans live under `docs/plans/`
- command-owned runtime state, reviews, and evidence live under
  `.local/harness/`
- the CLI reports one canonical `state.current_node`
- agents use repo-local skills instead of reconstructing workflow from shell
  history

The root CLI currently ships:

- `harness plan template`
- `harness plan lint`
- `harness init`
- `harness skills install`
- `harness skills uninstall`
- `harness instructions install`
- `harness instructions uninstall`
- `harness execute start`
- `harness evidence submit`
- `harness status`
- `harness ui`
- `harness review start`
- `harness review submit`
- `harness review aggregate`
- `harness archive`
- `harness reopen --mode <finalize-fix|new-step>`
- `harness land --pr <url> [--commit <sha>]`
- `harness land complete`

The root CLI also exposes `harness --version` as a plain-text binary identity
probe.

## Releases

`easyharness` ships through GitHub Releases and a dedicated Homebrew tap.
Supported release targets are:

- `darwin/amd64`
- `darwin/arm64`
- `linux/amd64`
- `linux/arm64`

Typical verification flow:

- macOS: `shasum -a 256 -c SHA256SUMS`
- Linux: `sha256sum -c SHA256SUMS`

To inspect a release archive directly:

```bash
unzip easyharness_<version>_<goos>_<goarch>.zip
cd easyharness_<version>_<goos>_<goarch>
./harness --version
./harness --help
```

Maintainers cut releases from a dedicated release PR that updates the root
`VERSION` file plus any related release docs. `VERSION` stores the unprefixed
release version such as `0.2.1`; after that PR merges to `main`, automation
creates the matching `v*` tag and dispatches the `Release` workflow, which
publishes the release assets for that tag and updates the Homebrew formula when
the tap token is configured.

## For Contributors

This repository is developed primarily through agents.

Repo-specific operating guidance lives in [AGENTS.md](./AGENTS.md). Detailed
local development and maintainer setup lives in
[docs/development.md](./docs/development.md). Durable CLI and workflow
contracts live in [docs/specs/index.md](./docs/specs/index.md), and the
checked-in schema registry lives in [schema/index.json](./schema/index.json).

## Background

These essays are good context for why harnesses matter and why
`easyharness` exists:

- [Harness design for long-running apps](https://www.anthropic.com/engineering/harness-design-long-running-apps)
- [Harness engineering: leveraging Codex in an agent-first world](https://openai.com/index/harness-engineering/)
