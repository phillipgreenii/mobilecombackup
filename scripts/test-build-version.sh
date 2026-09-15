#!/bin/bash
# Unit test for scripts/build-version.sh
#
# Exercises the four branches of the version-string algorithm:
#   1. exact git tag                (release build)
#   2. git hash + non-empty BASE_VERSION  (development build)
#   3. git hash + empty BASE_VERSION      (development build, no VERSION file)
#   4. no git available                   (fallback build)
#
# Every scenario is constructed in a fresh, isolated TEMP git repo -- never
# against the real checkout (CLAUDE.md unit-test isolation rule) -- so this is
# safe to run from any working directory, including a real clone with local
# changes.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_VERSION_SH="$SCRIPT_DIR/build-version.sh"

failures=0

# assert_eq <label> <expected> <actual>
assert_eq() {
  local label="$1" expected="$2" actual="$3"
  if [ "$expected" = "$actual" ]; then
    echo "✅ $label: got '$actual'"
  else
    echo "❌ $label: expected '$expected', got '$actual'"
    failures=$((failures + 1))
  fi
}

# assert_match <label> <regex> <actual>
assert_match() {
  local label="$1" regex="$2" actual="$3"
  if [[ $actual =~ $regex ]]; then
    echo "✅ $label: got '$actual' (matches $regex)"
  else
    echo "❌ $label: '$actual' does not match $regex"
    failures=$((failures + 1))
  fi
}

new_temp_repo() {
  local dir
  dir=$(mktemp -d)
  (
    cd "$dir"
    git init -q
    git config user.email "build-version-test@mobilecombackup.local"
    git config user.name "build-version-test"
  )
  echo "$dir"
}

echo "🧪 Testing scripts/build-version.sh"
echo "===================================="

# --- Scenario 1: exact git tag (release build) ---------------------------
echo ""
echo "Scenario 1: exact git tag"
repo1=$(new_temp_repo)
trap_dirs=("$repo1")
(
  cd "$repo1"
  echo "1.2.3-dev" >VERSION
  git add VERSION
  git commit -q -m "init"
  git tag v1.2.3
  out=$(bash "$BUILD_VERSION_SH")
  echo "$out"
) >"$repo1/out.txt"
scenario1_out=$(tail -n1 "$repo1/out.txt")
assert_eq "exact tag" "1.2.3" "$scenario1_out"

# --- Scenario 2: hash + non-empty BASE_VERSION (dev build) ---------------
echo ""
echo "Scenario 2: git hash + non-empty BASE_VERSION"
repo2=$(new_temp_repo)
trap_dirs+=("$repo2")
(
  cd "$repo2"
  echo "2.5.0-dev" >VERSION
  git add VERSION
  git commit -q -m "init"
  out=$(bash "$BUILD_VERSION_SH")
  echo "$out"
) >"$repo2/out.txt"
scenario2_out=$(tail -n1 "$repo2/out.txt")
assert_match "hash + BASE_VERSION" '^2\.5\.0-dev-g[0-9a-f]{7}$' "$scenario2_out"

# --- Scenario 3: hash + empty BASE_VERSION (dev build, no VERSION file) --
echo ""
echo "Scenario 3: git hash + empty BASE_VERSION"
repo3=$(new_temp_repo)
trap_dirs+=("$repo3")
(
  cd "$repo3"
  # No VERSION file at all -> BASE_VERSION resolves empty.
  git commit -q --allow-empty -m "init"
  out=$(bash "$BUILD_VERSION_SH")
  echo "$out"
) >"$repo3/out.txt"
scenario3_out=$(tail -n1 "$repo3/out.txt")
assert_match "hash + empty BASE_VERSION" '^dev-g[0-9a-f]{7}$' "$scenario3_out"

# --- Scenario 4: no git available (fallback build) ------------------------
echo ""
echo "Scenario 4: no git available"
repo4=$(mktemp -d)
trap_dirs+=("$repo4")
(
  cd "$repo4"
  echo "3.0.0-dev" >VERSION
  # Force every git invocation to fail regardless of any ancestor
  # repository, without relying on the temp dir's location in the
  # filesystem.
  out=$(GIT_DIR=/nonexistent-dir-force-no-git bash "$BUILD_VERSION_SH")
  echo "$out"
) >"$repo4/out.txt"
scenario4_out=$(tail -n1 "$repo4/out.txt")
assert_eq "no git" "3.0.0-dev" "$scenario4_out"

# --- Bonus: --base mode ----------------------------------------------------
echo ""
echo "Scenario 5 (bonus): --base mode"
(
  cd "$repo2"
  out=$(bash "$BUILD_VERSION_SH" --base)
  echo "$out"
) >"$repo2/out_base.txt"
scenario5_out=$(tail -n1 "$repo2/out_base.txt")
assert_eq "--base mode" "2.5.0" "$scenario5_out"

# --- Cleanup ----------------------------------------------------------------
for d in "${trap_dirs[@]}"; do
  rm -rf "$d"
done

echo ""
if [ "$failures" -eq 0 ]; then
  echo "🎉 All build-version.sh scenarios passed!"
  exit 0
else
  echo "❌ $failures scenario(s) failed"
  exit 1
fi
