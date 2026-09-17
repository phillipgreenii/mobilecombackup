# mobilecombackup justfile — translated from the previous package manifest's
# shell.scripts (tc-5lxy.8). Recipes stay thin; heavy logic stays in scripts/*.
#
# just's default shell is `sh -cu`, which several recipes below rely on
# bash for (e.g. full-test.sh's bash-only `[[ ]]`). Force bash for every
# recipe.
set shell := ["bash", "-cu"]

default:
    @just --list

# --- format / build / test / lint ---

# Upgraded from `go fmt ./...` to `nix fmt` (treefmt, which runs the
# stricter gofumpt) now that tc-5lxy.5 gives the flake a formatter output.
# This is the one permitted difference from the previous manifest's
# `formatter` script.
formatter:
    nix fmt

builder:
    go build -v ./...

tests:
    gotestsum --format testname -- -covermode=set ./...

test-unit:
    gotestsum --format testname -- -short ./...

test-integration:
    gotestsum --format testname -- -run Integration ./...

linter:
    golangci-lint run ./...

# Parameterized recipe backing `mobilecombackup smart-verify`'s targeted-test
# path (cmd/mobilecombackup/cmd/smart_verify.go's runTargetedTests). The
# prior implementation invoked the old package manager's arbitrary-command
# passthrough with a runtime-computed ./pkg/... list; `just` has no
# passthrough verb, so this recipe exists solely to give that call a named
# target (tc-5lxy.14). `pkgs` is one space-separated string of package
# globs (e.g. "./pkg/calls/... ./pkg/sms/...") that this recipe's own bash
# body re-splits on whitespace -- pass it quoted as a single argument.
test-packages pkgs:
    gotestsum --format testname -- {{pkgs}}

linter-fix:
    golangci-lint run --fix ./...

build-cli:
    VERSION=$(bash scripts/build-version.sh) && go build -ldflags "-X main.Version=$VERSION" -o mobilecombackup github.com/phillipgreenii/mobilecombackup/cmd/mobilecombackup

validate-version:
    bash scripts/validate-version.sh

# End-to-end smoke test (builds its own CLI internally via `just build-cli`
# at full-test.sh:6). Invoked via bash explicitly so its bash-only `[[ ]]`
# tests run correctly regardless of the script's own `#!/bin/sh` shebang.
full-test:
    bash full-test.sh

# Composed via recipe dependencies (not by shelling out to `just`), in the
# same order the previous manifest's ci ran them: formatter, tests, linter, build-cli.
ci: formatter tests linter build-cli

# --- coverage ---

coverage:
    echo '📊 Running test coverage analysis...'
    gotestsum --format testname -- -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    echo '✅ Coverage report generated: coverage.html'

coverage-summary:
    echo '📈 Test Coverage Summary:'
    gotestsum --format testname -- -cover ./...

# --- docs ---

validate-docs:
    bash scripts/validate-docs.sh

update-doc-health:
    bash scripts/update-doc-health.sh

# --- misc ---

ccusage:
    deno run -E -R=$HOME/.claude/ -R=$HOME/.config/claude/ -R=/tmp -S=homedir -N='raw.githubusercontent.com:443' npm:ccusage@latest blocks --live

# --- binary-dependent recipes ---
# Each depends on build-cli so it never runs against a stale ./mobilecombackup.

smart-verify: build-cli
    ./mobilecombackup smart-verify

test-fast: build-cli
    ./mobilecombackup test-runner --mode=fast

test-smart: build-cli
    ./mobilecombackup test-runner --mode=smart --order=fail-fast

test-cached: build-cli
    ./mobilecombackup test-runner --cache-dir=.test-cache

test-clear-cache: build-cli
    ./mobilecombackup test-runner --clear-cache
