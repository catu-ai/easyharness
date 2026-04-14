---
template_version: 0.2.0
created_at: "2026-04-14T21:04:51+08:00"
approved_at: "2026-04-14T21:21:32+08:00"
source_type: issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/32
size: XS
---

# Make `harness --version` JSON-First With Tiered Build Metadata

## Goal

Close `#32` by replacing the current plain-text `harness --version` debug
surface with a JSON-first version probe that better serves agent and script
consumers. This is an intentional contract change: the root `--version` flag
should now default to machine-readable JSON instead of human-oriented labeled
text.

This slice should also make a deliberate call on which richer build metadata
to expose now and how that differs between release and dev binaries. The
preferred direction is to expose only metadata that the running binary can
report reliably from Go build info or existing ldflags, keep release output
consumer-facing and concise, and allow dev output to carry additional
debug-friendly fields such as the resolved binary path.

## Scope

### In Scope

- Change `harness --version` so the default output is JSON.
- Remove the assumption that the root version probe remains plain text by
  default, and update tests/docs/help accordingly.
- Extend the internal version-info model so the CLI can emit one stable JSON
  contract with tiered field visibility for release and dev builds.
- Surface a narrowly chosen set of richer build metadata that is already
  available or cheap to derive from Go build info:
  `version`, `mode`, `commit`, with optional `go_version` and `build_time` for
  both modes when available, plus dev-only debug fields such as `modified` and
  `path`.
- Keep release output intentionally concise for consumers while allowing dev
  output to expose extra debug-helpful fields from the same source of truth.
- Update tests and docs to pin the new JSON default and the chosen
  release-versus-dev metadata contract.

### Out of Scope

- Replacing `harness --version` with a `harness version` subcommand.
- Preserving the old plain-text default or adding a permanent `--text`
  compatibility escape hatch.
- Adding installer-wrapper provenance, Homebrew-specific metadata, or other
  extra provenance that is not already available from the running binary.
- Reworking unrelated command-output contracts away from the existing
  JSON-first workflow design.

## Acceptance Criteria

- [ ] `harness --version` exits zero and returns parseable JSON by default.
- [ ] The default JSON contract is stable and intentionally tiered:
      release output includes the concise consumer-facing subset, while dev
      output may include extra debug-oriented fields such as `modified` and
      `path`.
- [ ] The richer metadata set is deliberate and documented: `go_version` and
      `build_time` may appear in both modes when available from build info,
      while `modified` is dev-only and unavailable data is omitted rather than
      fabricated.
- [ ] Unit and smoke coverage verify the JSON default, reject regressions back
      to plain text, and cover release-versus-dev visibility rules for `path`
      and the richer metadata fields.
- [ ] Docs and help text describe `harness --version` as a JSON-first binary
      identity probe so this slice can reasonably close `#32`.

## Deferred Items

- Add wrapper/install provenance only if a later slice decides that binary
  identity is insufficient without distribution-context metadata.
- Add a dedicated `version` subcommand only if future UX work finds a real
  need beyond the root flag.

## Work Breakdown

### Step 1: Replace the plain-text version contract with tiered JSON output

- Done: [x]

#### Objective

Define one shared internal version-info shape and emit path so the CLI can
replace the current plain-text version view with one stable JSON contract.

#### Details

Keep the current trust boundary: report what the running binary can actually
know about itself. Prefer Go build info and existing build variables over new
plumbing. The JSON contract should use the same field names in both modes, but
release builds should only expose the concise consumer-facing subset, while
dev builds may expose extra debug-oriented fields such as `modified` and
`path`. If some build-info fields are not available in a given execution
context, omit them cleanly instead of introducing fake placeholders or
compatibility shims.

Use the following examples as the intended contract shape. They are not
byte-for-byte golden fixtures for every environment, but they define the field
names, release-versus-dev visibility rules, and omission semantics this slice
is expected to implement.

Release example:

```json
{
  "version": "v0.2.1",
  "mode": "release",
  "commit": "abc1234",
  "go_version": "go1.25.0",
  "build_time": "2026-04-14T12:34:56Z"
}
```

Dev example:

```json
{
  "version": "v0.2.1-dev",
  "mode": "dev",
  "commit": "abc1234",
  "go_version": "go1.25.0",
  "build_time": "2026-04-14T12:34:56Z",
  "modified": true,
  "path": "/Users/example/src/easyharness/.local/bin/harness"
}
```

For both examples:

- `path` is omitted outside dev mode.
- `modified` is omitted outside dev mode.
- If `build_time` is unavailable from the running binary metadata, omit it
  instead of inventing a placeholder.
- If `go_version` is unavailable, omit it instead of fabricating one.
- In dev mode, `modified` should reflect binary metadata when available; if a
  build context cannot report it reliably, omit it rather than guessing.

#### Expected Files

- `internal/version/info.go`
- `internal/version/info_test.go`
- `internal/cli/app.go`
- `internal/cli/app_test.go`

#### Validation

- Unit tests cover the shared version-info struct for release and dev builds.
- CLI tests cover `--version` and `--version --help` without ambiguity in
  root-flag parsing.
- Tests pin the JSON field set and ensure the command no longer falls back to
  plain-text labeled output.

#### Execution Notes

Expanded `internal/version.Info` into the shared JSON contract source for
`harness --version`, adding `go_version` and `build_time` for both modes when
available plus dev-only `modified` and `path`. Switched the root flag output
from labeled plain text to indented JSON, updated the CLI help text, and
retired text-parsing consumers in the dev installer wrapper and Homebrew
formula generator so they now consume the JSON contract instead of grepping
plain-text labels.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 1 and Step 2 landed as one tightly coupled slice,
so a single full finalize review gives a more truthful read than isolated
step-closeout review.

### Step 2: Pin the JSON default in smoke coverage and docs

- Done: [x]

#### Objective

Protect the new JSON default and tiered metadata contract with repo-level
coverage and durable docs so later changes do not accidentally drift back to
plain text or blur the dev-versus-release boundary.

#### Details

Update the CLI contract and user-facing docs to explain that `harness --version`
is now a JSON-first binary identity probe in this agent-first repository.
Repo-level tests should prove the built binary emits JSON by default, keep the
release-mode path omission, and verify that richer metadata only appears when
the binary can report it honestly and when the chosen mode should expose it.

#### Expected Files

- `tests/smoke/smoke_test.go`
- `README.md`
- `docs/specs/cli-contract.md`

#### Validation

- Smoke tests assert the default built-binary output is valid JSON.
- Smoke or integration coverage asserts release and dev binaries expose the
  agreed field tiers, especially for `modified` and `path`.
- Docs and help text stay aligned about the JSON default and the richer
  machine-readable contract.

#### Execution Notes

Updated the smoke suite, release-package checks, Homebrew formula checks, and
stable-wrapper fallback tests to validate JSON output and the new release/dev
field tiers. Refreshed `README.md`, `docs/specs/cli-contract.md`,
`docs/development.md`, and `docs/releasing.md` so the checked-in docs all
describe `harness --version` as a JSON-first binary identity probe. Validated
the slice with `bash -n scripts/install-dev-harness`, focused unit/smoke runs,
and a final `go test ./... -count=1`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 stayed coupled to Step 1's contract flip and was
validated together; the candidate will receive one full finalize review.

## Validation Strategy

- Run focused unit tests for `internal/version` and `internal/cli`.
- Run repo-level smoke coverage for version-command behavior.
- Run the full Go test suite if the targeted changes stay green and do not
  expose unrelated failures.

## Risks

- Risk: The richer metadata may differ across dev, release, and test-built
  binaries in ways that make the contract flaky.
  - Mitigation: Source metadata from `runtime/debug` build info where possible,
    omit unavailable fields instead of inventing values, and pin tests to
    field presence rules rather than unstable local timestamps or paths.
- Risk: Replacing the long-standing plain-text default could leave stale docs
  or tests that still assume labeled text.
  - Mitigation: Update help text, README, CLI contract, and smoke assertions in
    the same slice so the repository only documents one default behavior.

## Validation Summary

PENDING_UNTIL_ARCHIVE

## Review Summary

PENDING_UNTIL_ARCHIVE

## Archive Summary

PENDING_UNTIL_ARCHIVE

## Outcome Summary

### Delivered

PENDING_UNTIL_ARCHIVE

### Not Delivered

PENDING_UNTIL_ARCHIVE

### Follow-Up Issues

NONE
