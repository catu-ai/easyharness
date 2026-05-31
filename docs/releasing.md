# Releasing

`easyharness` ships as GitHub Release archives built from the tracked release
workflow at
[catu-ai/easyharness](https://github.com/catu-ai/easyharness).

The release archive name follows the project name, while the unpacked
executable remains `harness`. Tagged releases can also update the dedicated
Homebrew tap formula `easyharness` in `catu-ai/homebrew-tap`, which users
install as `catu-ai/tap/easyharness`.

## Release Policy

`easyharness` currently ships a public 0.x release line. Releases are
change-driven rather than calendar-driven: maintainers cut a release when the
current `main` branch contains a coherent user-facing improvement, a useful
same-theme fast-follow, or a repair worth shipping to users.

Ordinary feature, documentation, and fix PRs do not bump `VERSION`. Version
changes happen in a dedicated release PR after the intended scope is already
merged or ready to publish. That release PR should stay narrow: update the
root `VERSION` file, include any directly related release documentation, and
let the VERSION-driven automation create the tag and publish assets after the
PR merges.

Minor releases such as `v0.5.0` represent a coherent user promise. They do not
need to finish every possible idea in a feature family, but they should deliver
enough of the promise that users can understand and start relying on it. A
minor release must be selected through a concrete GitHub milestone whose issues
are release-critical for that promise.

Minor releases are selected release promises, not automatic version bumps for
every public, user-facing, or agent-facing contract change. A contract change
outside the selected minor milestone can still be release-worthy, but it should
wait for a later selected promise or ship as a patch when it fits the patch
criteria below.

Patch releases such as `v0.5.1` are smaller. They may ship bug fixes, release
repairs, documentation or CI corrections, and low-risk follow-ups that extend
the same already-shipped promise. Routine patch releases do not need a GitHub
milestone. Create a patch milestone only when the patch needs an explicit
bucket, such as a small coordinated set of related fixes.

Milestones are release promises, not feature-family backlogs. Put only the
issues that must be done before that version should ship into the milestone.
Related follow-up issues can remain normal triaged issues until maintainers
select them for a later release. When useful, an umbrella or delivery issue can
use its body to distinguish must-deliver work from likely fast-follows and
later/not-now ideas; do not add priority labels or tracker issues just to carry
that context.

Release readiness depends on both automation and maintainer judgment. Required
CI must be green for the release PR, and the release workflow must publish the
expected GitHub Release and Homebrew artifacts after merge. Maintainers also
decide whether the release promise is complete enough, whether known blockers
remain, and whether a post-publication repair is needed.

## Release Checklist

1. Decide the next release version, such as `0.0.0`, and update the
   root `VERSION` file in a dedicated release PR.
2. Make sure `main` is up to date, run `scripts/build-embedded-ui`, and then
   run `go test ./...` in the release PR before merge.
3. If you want an extra local packaging check before merge, run
   `scripts/build-release --version "v$(cat VERSION)"`.
4. Merge the release PR to `main`.
5. Confirm the `Tag Release From VERSION` workflow created the matching git
   tag, for example `v0.0.0`, and then dispatched the `Release`
   workflow for that tag.
6. Confirm the `Release` workflow uploaded the release archives and
   `SHA256SUMS` file.
7. If the Homebrew tap token is configured, confirm the workflow updated
   `Formula/easyharness.rb` in `catu-ai/homebrew-tap`.
8. Confirm the release workflow's Homebrew verification job passed.
   It should stage a local `catu-ai/tap` checkout from the rendered formula,
   install an earlier compatible release when one exists, upgrade to the
   current release with `brew upgrade catu-ai/tap/easyharness`, and pass
   `brew test easyharness`.

`VERSION` intentionally stores the bare release version without the leading
`v`. The auto-tag workflow adds that prefix when it creates the git tag, so
`VERSION=0.0.0` maps to the release tag `v0.0.0`.

Release PR separation is a team convention rather than a repository-enforced
rule. The expected path is that a release PR contains the `VERSION` bump and
any release-doc updates, while ordinary feature PRs leave `VERSION` alone.

You can still use the `Release` workflow-dispatch path to republish assets for
an existing `v*` tag without creating a second tag. The workflow rejects
branch names or other non-tag refs.

Release archives intentionally derive packaged file mtimes from the tagged
commit timestamp in UTC, subject to ZIP's 2-second timestamp precision. That
keeps repeated builds of the same commit deterministic without making unpacked
files look like they came from `2000-01-01 00:00`.

## What Gets Published

- Prebuilt `darwin/amd64`, `darwin/arm64`, `linux/amd64`, and `linux/arm64`
  archives.
- A `SHA256SUMS` file for checksum verification.
- Tags with a prerelease suffix such as `-alpha.1` publish as GitHub
  prereleases rather than stable releases.
- The release binary reports JSON build metadata through `harness --version`,
  including the release version and mode, plus the build commit, `go_version`,
  or `build_time` when the binary can report them.
- Archive entry timestamps are derived from the source commit time for the
  tagged revision, subject to ZIP's 2-second precision, rather than the
  wall-clock publish time.
- When the tap token is configured, tagged releases update the default
  Homebrew formula `easyharness`, including prerelease tags. Stable tags make
  the default tap formula stable again; prerelease tags can also move the tap
  forward when maintainers choose to publish them.

## Homebrew Tap Publishing

Homebrew publishing uses the separate public repository
`catu-ai/homebrew-tap`. Because Homebrew lets users omit the `homebrew-`
prefix in tap commands, that repository is installed as `catu-ai/tap`.

Tagged releases update the tap on GitHub alone once these prerequisites are in
place:

1. Create `catu-ai/homebrew-tap` with an initial commit on its default branch.
   The workflow assumes that branch is `main`.
2. Add a repository secret named `EASYHARNESS_HOMEBREW_TAP_TOKEN` to
   `catu-ai/easyharness`.
3. Give that token contents-write access to `catu-ai/homebrew-tap`.

The release workflow renders `Formula/easyharness.rb` from the staged
`dist/release/SHA256SUMS` file after the GitHub Release assets are published,
then pushes the updated formula into the tap when the secret is available.

If the secret is missing, the release workflow emits a warning and skips the
tap update instead of blocking the archive upload. The repair path is:

1. Configure or fix `EASYHARNESS_HOMEBREW_TAP_TOKEN`.
2. Confirm `catu-ai/homebrew-tap` still has a writable default branch.
3. Re-run the Release workflow with `workflow_dispatch` for the same `v*` tag.

The formula name remains `easyharness`, while the installed binary remains
`harness`.

## Contributor Baseline

Release and CI jobs use the Go version recorded in `go.mod`, which is currently
`go 1.25.0`. They also install Node.js/pnpm so the embedded UI assets are
built before Go tests and release packaging consume them.
