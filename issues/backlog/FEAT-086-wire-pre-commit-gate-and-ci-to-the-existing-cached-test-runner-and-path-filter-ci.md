# FEAT-086: wire pre-commit gate and CI to the existing cached test-runner, and path-filter CI

## Status
- **Priority**: medium

## Overview
Follow-up from a workspace-wide testing-efficiency audit (2026-09-05). The pre-commit hook
and CI workflow both run the full uncached test suite on every code commit/push, even though
this repo already built a caching test runner (`cmd/mobilecombackup/cmd/test_runner.go`) that
neither one calls.

## Background
NOTE (tc-5lxy.25, 2026-09-17): `.githooks/pre-commit` and `scripts/install-hooks.sh` — the
hook file this issue's Tasks section below proposed editing — were deleted by tc-5lxy.5, and
the dev environment migrated from the old package manager to flox/just (tc-5lxy.9, tc-5lxy.8) after this issue
was filed. Pre-commit hooks are now managed by the nix `flakeModules.pre-commit` framework
(see `docs/GIT_WORKFLOW.md#git-hooks`); this issue's premise of editing `.githooks/pre-commit`
directly no longer applies as written and needs re-scoping against that framework's own
hook-composition mechanism before it can be worked. `just`/`flox` equivalents are substituted
below for the commands this issue names, but the wiring approach itself is unverified against
the new hook framework.

FEAT-072 already optimized the pre-commit hook to skip tests entirely for markdown-only
commits (via the now-deleted `.githooks/pre-commit`, file-type detection via
`git diff --cached --name-only`). But any commit touching a single `.go` file still runs
`just tests` — plain `gotestsum -- -covermode=set ./...`, the entire package tree, uncached.

Separately, `cmd/mobilecombackup/cmd/test_runner.go` implements a `test-runner` subcommand
with content-hash caching and `--mode=fast/smart`, exposed via the `just test-cached` /
`test-smart` / `test-fast` recipes — but nothing in the commit/push path calls it.

CI (`.github/workflows/test.yml`) has no `paths-ignore` filter either, so a docs-only or
`issues/`-only push still runs the full `go test -v -covermode=set ... ./...` job.

## Requirements
### Functional Requirements
- [ ] The nix pre-commit framework's code-commit path calls `just test-cached` (or
      `test-smart`) instead of plain `just tests`, so an unchanged package's tests are
      skipped via the existing content-hash cache rather than re-run every commit. (The
      original `.githooks/pre-commit` this pointed at is gone — see the note above; the
      equivalent hook needs identifying in the nix framework first.)
- [ ] `.github/workflows/test.yml` adds a `paths-ignore` (or equivalent) so a push touching
      only `**.md`, `issues/**`, or `docs/**` doesn't trigger the full Go test job.

### Non-Functional Requirements
- [ ] No reduction in actual coverage — the cached/smart mode must still catch a regression
      in a changed package; only unchanged packages should be skipped.

## Design
### Approach
`test_runner.go` and its `--cache-dir`/content-hash design already exist and are exercised
via the `just test-cached`/`test-smart`/`test-fast` recipes — this is wiring, not new test
infrastructure. Read `test_runner.go`'s existing modes before changing the hook, since
`--mode=fast` vs `--mode=smart` may have different tradeoffs for a commit-time gate vs. a
pre-push gate.

## Tasks
- [ ] Read `cmd/mobilecombackup/cmd/test_runner.go` and the justfile's `test-cached`/
      `test-smart`/`test-fast` recipes to confirm which mode fits the commit-time gate.
- [ ] Identify the current nix-pre-commit-framework equivalent of the deleted
      `.githooks/pre-commit` and update its code-commit branch to call the cached/smart runner.
- [ ] Add a `paths-ignore` filter to `.github/workflows/test.yml`.
- [ ] Verify a commit touching only one package skips the others (timing/log evidence).

## References
- Code locations: the nix `flakeModules.pre-commit` hook set (successor to the deleted
  `.githooks/pre-commit`), `cmd/mobilecombackup/cmd/test_runner.go`,
  `justfile` (`tests`, `test-cached`, `test-smart`, `test-fast` recipes),
  `.github/workflows/test.yml`
- Related: FEAT-072 (markdown-only pre-commit optimization) — this extends the same
  file-type-detection idea to code commits via the runner's own cache instead of a
  markdown/code binary split.

## Notes
Found during a workspace-wide agent-validation-efficiency audit, not by working in this repo
directly — verify the exact `test_runner.go` cache-key/invalidation behavior before wiring it
into the hook, since a wrong cache key would silently skip a package that actually changed.

Cross-tracked as `tc-5lxy.28` in beads (this repo's `issues/` tracker is itself slated for
migration to beads per tc-5lxy's scope correction) — that bead also records that the old
package manager was completely absent on this machine, which is why this very commit could
not run the pre-commit hook's quality checks and needed an explicit operator-approved
`--no-verify`.
