# FEAT-086: wire pre-commit gate and CI to the existing cached test-runner, and path-filter CI

## Status
- **Priority**: medium

## Overview
Follow-up from a workspace-wide testing-efficiency audit (2026-09-05). The pre-commit hook
and CI workflow both run the full uncached test suite on every code commit/push, even though
this repo already built a caching test runner (`cmd/mobilecombackup/cmd/test_runner.go`) that
neither one calls.

## Background
FEAT-072 already optimized the pre-commit hook to skip tests entirely for markdown-only
commits (`.githooks/pre-commit`, file-type detection via `git diff --cached --name-only`).
But any commit touching a single `.go` file still runs `devbox run tests` — plain
`gotestsum -- -covermode=set ./...`, the entire package tree, uncached.

Separately, `cmd/mobilecombackup/cmd/test_runner.go` implements a `test-runner` subcommand
with content-hash caching and `--mode=fast/smart`, exposed via `devbox run test-cached` /
`test-smart` / `test-fast` in `devbox.json` — but nothing in the commit/push path calls it.

CI (`.github/workflows/test.yml`) has no `paths-ignore` filter either, so a docs-only or
`issues/`-only push still runs the full `go test -v -covermode=set ... ./...` job.

## Requirements
### Functional Requirements
- [ ] `.githooks/pre-commit`'s code-commit path calls `devbox run test-cached` (or
      `test-smart`) instead of plain `devbox run tests`, so an unchanged package's tests are
      skipped via the existing content-hash cache rather than re-run every commit.
- [ ] `.github/workflows/test.yml` adds a `paths-ignore` (or equivalent) so a push touching
      only `**.md`, `issues/**`, or `docs/**` doesn't trigger the full Go test job.

### Non-Functional Requirements
- [ ] No reduction in actual coverage — the cached/smart mode must still catch a regression
      in a changed package; only unchanged packages should be skipped.

## Design
### Approach
`test_runner.go` and its `--cache-dir`/content-hash design already exist and are exercised
via `devbox run test-cached`/`test-smart`/`test-fast` — this is wiring, not new test
infrastructure. Read `test_runner.go`'s existing modes before changing the hook, since
`--mode=fast` vs `--mode=smart` may have different tradeoffs for a commit-time gate vs. a
pre-push gate.

## Tasks
- [ ] Read `cmd/mobilecombackup/cmd/test_runner.go` and `devbox.json`'s `test-cached`/
      `test-smart`/`test-fast` scripts to confirm which mode fits the commit-time gate.
- [ ] Update `.githooks/pre-commit`'s code-commit branch to call the cached/smart runner.
- [ ] Add a `paths-ignore` filter to `.github/workflows/test.yml`.
- [ ] Verify a commit touching only one package skips the others (timing/log evidence).

## References
- Code locations: `.githooks/pre-commit`, `cmd/mobilecombackup/cmd/test_runner.go`,
  `devbox.json` (`scripts.tests`, `test-cached`, `test-smart`, `test-fast`),
  `.github/workflows/test.yml`
- Related: FEAT-072 (markdown-only pre-commit optimization) — this extends the same
  file-type-detection idea to code commits via the runner's own cache instead of a
  markdown/code binary split.

## Notes
Found during a workspace-wide agent-validation-efficiency audit, not by working in this repo
directly — verify the exact `test_runner.go` cache-key/invalidation behavior before wiring it
into the hook, since a wrong cache key would silently skip a package that actually changed.

Cross-tracked as `tc-5lxy.28` in beads (this repo's `issues/` tracker is itself slated for
migration to beads per tc-5lxy's scope correction) — that bead also records that `devbox` is
completely absent on this machine, which is why this very commit could not run the pre-commit
hook's quality checks and needed an explicit operator-approved `--no-verify`.
