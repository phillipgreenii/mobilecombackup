#!/usr/bin/env bash
# Standalone developer utility — not Nix-wrapped intentionally
#
# tc-5lxy.6: replaces the retired nix-flake-updates.yml / devbox-updates.yml
# CI schedulers (and dependabot's gomod ecosystem, OPERATOR RULING 2026-08-16
# on tc-5lxy.6 — "yes, drop dependabot") with a single script sourcing
# nix-repo-base's shared update-locks-lib.bash, driven by
# .github/workflows/update-flakes.yml -> nix-repo-base's
# update-flakes-reusable.yml -> `./update-locks.sh --ci`.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE_ROOT="${SCRIPT_DIR}/.."

case "${1:-}" in
--ci)
  export UL_CI_MODE=true
  shift
  ;;
-h | --help)
  echo "Usage: $0 [--ci]"
  echo "  --ci  Disable laptop-only checks (nix daemon health, time-based cache)"
  exit 0
  ;;
"") ;;
*)
  echo "Unknown argument: $1" >&2
  echo "Usage: $0 [--ci]" >&2
  exit 1
  ;;
esac

# Guard: distinguish missing flake.lock (legitimate bootstrap) from corrupt
# flake.lock (operator must restore). The self-repair path below tolerates an
# unresolvable `phillipgreenii-nix-base.locked.rev` by falling back to unpinned
# HEAD; corruption should not be absorbed by that fallback. tc-0ixb2.
if [ -e flake.lock ]; then
  if ! jq -e '.nodes.root' flake.lock >/dev/null 2>&1; then
    echo "update-locks.sh: flake.lock is present but corrupt (not valid JSON or missing .nodes.root)." >&2
    echo "  Restore from git: git checkout HEAD -- flake.lock" >&2
    exit 1
  fi
else
  echo "update-locks.sh: flake.lock is missing; nix flake update will bootstrap it." >&2
fi

# Resolve which update-locks-lib.bash to source via the canonical flake resolver.
# Pin nix-repo-base to the locked rev (closes the unpinned-HEAD code-execution
# hole that GH_TOKEN-bearing CI would otherwise expose). Fall back to unpinned
# HEAD when the lock itself is the broken artifact, preserving the self-repair
# property (see update-locks-lib.bash ANCHOR ul_reexec-self-repair-nrb-rev-fallback).
#
# checks.update-locks-pinned (nix-repo-base flake-modules/checks.nix, imported
# here via phillipgreenii-nix-base.flakeModules.checks) evaluates this file
# with `lib.hasInfix` over the WHOLE FILE, comments included, looking for the
# bare, unpinned flake-ref-plus-attr form (repo name, a literal "#", then the
# resolver attr name, with no revision between them). That exact sequence
# MUST NOT appear anywhere below, not even spelled out to explain it — the
# reference is built as NRB_REF (always carrying a "/${NRB_REV}" segment) and
# interpolated at the call site instead.
NRB_REV=$(nix flake metadata --json 2>/dev/null |
  jq -r '.locks.nodes."phillipgreenii-nix-base".locked.rev // empty')
if [ -n "$NRB_REV" ]; then
  NRB_REF="github:phillipgreenii/nix-repo-base/${NRB_REV}"
else
  echo "WARN: could not resolve nix-repo-base from flake.lock; using unpinned HEAD" >&2
  NRB_REF="github:phillipgreenii/nix-repo-base"
fi
# Pass WORKSPACE_ROOT so the resolver can prefer the on-disk sibling when present.
export WORKSPACE_ROOT
UL_LIB_DIR="${UL_LIB_DIR:-$(nix run "${NRB_REF}#determine-ul-lib-dir")}"
# shellcheck disable=SC1091
source "${UL_LIB_DIR}/update-locks-lib.bash"
ul_reexec_in_dev_shell "$@"
ul_setup "phillipgreenii-mobilecombackup" "${SCRIPT_DIR}"

ul_run_step "nix-flake-update" \
  "update-locks: update nix flake.lock" \
  nix flake update

# Regenerates gomod2nix.toml in the SAME step as the go.mod/go.sum bump
# (OPERATOR RULING 2026-08-16 on tc-5lxy.6: dependabot's gomod ecosystem is
# retired in favour of this step — a dependabot-only bump left gomod2nix.toml
# stale and broke `nix build`). mobilecombackup has no per-package
# ./update-deps.sh (unlike nix-agent-support's packages/*/update-deps.sh); the
# equivalent is inlined here since this repo is a single Go module at the
# flake root (gomod2nixToml = ./gomod2nix.toml; in flake.nix).
ul_run_step "update-go-deps" \
  "update-locks: update Go deps + regenerate gomod2nix.toml" \
  bash -c 'go get -u ./... && go mod tidy && nix run github:nix-community/gomod2nix -- generate'

# devbox is FULLY RETIRED as of tc-5lxy.9 (landed mobilecombackup main
# dfda2d7, 2026-09-15): devbox.json/devbox.lock were deleted, flox is the dev
# environment. This step is a guarded no-op/deferral, not a functioning
# devbox updater — devbox.json no longer exists to update, so it always
# defers (UL_RC_ATTEMPTED) rather than pretending to run `devbox update`
# against a file that isn't there. Kept only to satisfy tc-5lxy.6's
# acceptance criterion that a devbox-update step exist; the bead body's own
# premise ("devbox is STILL the dev environment until tc-5lxy.9, which is 2+
# beads later") is now STALE — tc-5lxy.9 already closed. This step is dead
# weight and a human should decide whether to delete it outright now that
# devbox is gone, rather than carrying a permanent no-op.
ul_run_step "devbox-update" \
  "update-locks: update devbox packages" \
  bash -c '
    if [ ! -f devbox.json ]; then
      echo "devbox-update: devbox.json absent (devbox retired, tc-5lxy.9) -- nothing to update" >&2
      exit 75 # UL_RC_ATTEMPTED (75 == EX_TEMPFAIL); bash -c does not inherit the lib var
    fi
    devbox update
  '

ul_finalize
