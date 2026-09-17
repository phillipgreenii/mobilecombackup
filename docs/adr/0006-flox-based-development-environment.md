# ADR-0006: Flox-Based Development Environment

**Status:** Accepted
**Date:** 2026-09-15
**Author:** Development Team
**Deciders:** Core development team / operator

## Context

ADR-0005 chose a JSON-manifest-based tool as this project's development environment manager
(see that ADR for the full historical record of what it was and why). Separately, the
project adopted the shared `pn-workspace.toml` workspace of `phillipgreenii` nix-\* repos
(2026-08-16) and undertook a broader effort (epic tc-5lxy) to bring mobilecombackup onto that
shared Nix infrastructure: `flakeModules.pre-commit`/`checks`/`devshell` for git hooks,
formatting and CI checks (tc-5lxy.5), a pinned lock-update script (tc-5lxy.6), and a
`justfile` task runner replacing the old manifest's `scripts` (tc-5lxy.8).

The prior tool itself does not compose the same way as the rest of that shared infrastructure
— it is not Nix-flake-native, and the reuse review behind epic tc-5lxy identified Flox as the
intended replacement (reuse item 12). Flox is Nix-based (so it fits the same underlying
toolchain as the rest of the workspace) but exposes a declarative TOML manifest and its own
catalog/search workflow rather than the old JSON manifest's script-based interface.

## Decision

We replace the prior environment manager with **Flox** (`.flox/env/manifest.toml` +
`.flox/env/manifest.lock`) as the development environment manager, and **just**
(repo-root `justfile`) as the task runner for the recipes the old manifest previously
exposed as inline scripts (tc-5lxy.9, tc-5lxy.8).

### Environment Management

- **Flox**: reproducible development environments, Nix-backed, activated via `flox activate`
  or automatically via direnv (`.envrc` contains `use flox`)
- All 13 previously-pinned tools plus `just` are declared in `.flox/env/manifest.toml`'s
  `[install]` section, resolved directly from Flox's own nixpkgs-backed catalog

### Task Runner

- **just**: thin recipes in the repo-root `justfile` (formatter, builder, tests, test-unit,
  test-integration, linter, linter-fix, build-cli, ci, coverage, coverage-summary,
  validate-docs, update-doc-health, validate-version, full-test, and the `test_runner.go`-backed
  smart-verify/test-fast/test-smart/test-cached/test-clear-cache recipes) — heavy logic stays
  in `scripts/*`, per the workspace convention (nix-repo-base's own justfile states the same
  rule)

### Git Hooks / Quality Gates

- Git hooks, formatting (`nix fmt` / treefmt+gofumpt) and CI checks moved to the shared
  `flakeModules.pre-commit`/`checks`/`devshell` (tc-5lxy.5), replacing `.githooks/` and
  `scripts/install-hooks.sh` (both deleted). This is a separate decision from the
  environment-manager swap covered here, but the two were sequenced together as part of the
  same epic and this ADR references it for completeness — see
  [Git Workflow](../GIT_WORKFLOW.md#git-hooks).

## Rationale

### Flox for Environment Management

- **Fits the shared workspace**: Nix-based, so it composes with the rest of the
  `pn-workspace.toml` infrastructure (flake inputs, `flakeModules`) the way the prior
  JSON-manifest tool did not
- **Reproducible environments**: same underlying guarantee ADR-0005 valued, now delivered via
  Flox's own catalog/lockfile instead
- **Cross-platform support**: works on macOS and Linux (Flox's manifest declares no explicit
  platform restriction, defaulting to Flox's own multi-system resolution — see the manifest's
  `[options]` comment for one caveat: `go` currently has no `x86_64-darwin` build in Flox's
  catalog)
- **Package parity**: all 13 previously-pinned packages plus `just` resolved 1:1 from Flox's
  catalog with no behavior change (verified per-package via `flox search`/`flox show` before
  the manifest was authored) — see `.flox/env/manifest.toml`'s own comments for the two
  documented exceptions (an unpinned `claude-code` version, and the `go` version pin
  cross-checked against `quality-dashboard.yml`'s `actions/setup-go` pin)

### just for Task Running

- Replaces the old manifest's inline `scripts` section — Flox's manifest has no equivalent
  named-script mechanism (`[install]`/`[vars]`/`[hook]`/`[profile]`/`[services]`/`[options]`
  sections only), so task running needed a new home
- Matches the workspace convention already used by the other `pn-workspace.toml` repos
- Recipes stay thin (a call to `scripts/*.sh`, `go build`/`go test`, or `nix fmt`); nothing
  reimplements logic that used to live in the old manifest's own script bodies

### Alternatives Considered

Re-litigating the alternatives ADR-0005 already weighed (Docker, Make, plain GOPATH, CI-only
quality checks) is out of scope here — none of that reasoning changed. The only new
alternative considered was staying on the pre-existing environment manager and NOT adopting
the shared workspace infrastructure; that was rejected because it would have left
mobilecombackup unable to reuse `nix-repo-base`'s `flakeModules` (pre-commit, checks, devshell,
the Go builder factory) and its pinned lock-update tooling, duplicating maintenance the rest
of the `phillipgreenii` nix-\* workspace already centralizes.

## Consequences

### Positive Consequences

- **Workspace fit**: mobilecombackup now composes with `pn-workspace.toml` tooling the same
  way sibling repos do
- **Consistency**: all developers use identical tool versions, resolved from one catalog
- **Onboarding**: `flox activate` (or direnv auto-activation) gives the full environment; no
  separate task-runner install beyond what the manifest itself provides (`just` is one of the
  installed packages)
- **Quality automation unchanged in kind**: git hooks / CI quality gates continue to exist,
  now via the nix `flakeModules.pre-commit` framework instead of the old manifest's own hook
  wiring

### Negative Consequences

- **Second learning curve**: developers already having learned the prior tool's concepts now
  need Flox's (manifest sections, `flox activate`, catalog search) — the same category of cost
  ADR-0005 accepted for its own tool, paid again
- **Migration cost**: every doc, script and CI workflow referencing the old commands needed
  updating (this ADR's own motivating epic, tc-5lxy, and specifically tc-5lxy.25 for the
  documentation sweep)
- **No named-script mechanism in Flox's manifest**: unlike the prior tool, Flox has no
  `scripts` section, which is why a separate task runner (`just`) was needed at all — one more
  moving part than a single all-in-one manifest

## Implementation

### Flox Manifest (excerpt)

```toml
[install]
go.pkg-path = "go"
go.version = "1.26.5"
just.pkg-path = "just"
# ...11 more packages

[hook]
on-activate = '''
  go mod tidy
  # exports NVIM_PROJECT_CONFIG for the project-specific Neovim config
'''

[profile]
bash = '''
  # vim/nvim aliases -- defined here (not [hook]) so they persist into the
  # caller's actual interactive shell
'''
```

### Development Workflow

1. **Environment activation**: `flox activate` (or `cd` into the repo with direnv installed
   and `.envrc` allowed once)
2. **Development iteration**: write code with immediate linting/formatting feedback
3. **Testing**: `just test-unit` for fast feedback, `just tests` for the full suite
4. **Pre-commit**: nix-managed hooks run formatter/tests/linter/build automatically
5. **Commit**: quality-assured commits only, per [Git Workflow](../GIT_WORKFLOW.md)

### Quality Gates

Unchanged in substance from ADR-0005: formatter (`nix fmt`, now treefmt/gofumpt rather than
plain `gofumpt -l -w .`), linter (`golangci-lint run`, via `just linter`), tests, and build
must all pass before commit.

## Related Decisions

- **Supersedes**: [ADR-0005](0005-development-tool-choices.md) (Development Tool Ecosystem
  Choices) — ADR-0005's own content is left as-is (an executed decision must stay legible as
  history); only its Status is updated to Superseded, per this ADR.
- **ADR-0001**: Streaming Processing — development tools still support streaming architecture
  testing
- **ADR-0003**: XML Security — security linting remains integrated into the development
  workflow, now via the nix `flakeModules.checks`/`pre-commit` gates
