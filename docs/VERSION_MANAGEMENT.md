# Version Management Workflow

This document describes the versioning system used for mobilecombackup, including development workflows, release procedures, and CI/CD integration.

## Versioning Scheme Overview

The project uses a git tag-based versioning system with VERSION file fallback, providing a single source of truth for version information across CLI, SonarQube, and GitHub Actions.

### Version Formats
- **Development builds**: `2.0.0-dev-g1234567` (VERSION file base + git hash)
- **Release builds**: `2.0.0` (clean semantic version from git tags)
- **VERSION file content**: `2.0.0-dev` (base version with -dev suffix)
- **Git tag format**: `v2.0.0` (with "v" prefix following Go conventions)

## Developer Workflows

### Starting New Version Development
```bash
# After releasing v2.0.0, start development for next version
echo "2.1.0-dev" > VERSION
git add VERSION
git commit -m "Start v2.1.0 development"

# During development, builds show: 2.1.0-dev-g1234567
devbox run build-cli
./mobilecombackup --version  # Shows: mobilecombackup version 2.1.0-dev-g1234567
```

### Creating a Release
```bash
# 1. Ensure VERSION file contains correct base version (without -dev)
# 2. Create and push git tag
git tag v2.1.0
git push origin v2.1.0

# 3. Release builds now show: 2.1.0 (clean version from git tag)
# 4. GitHub Actions will trigger release build automatically

# 5. Post-release, update VERSION for next development cycle
echo "2.2.0-dev" > VERSION
git add VERSION
git commit -m "Start v2.2.0 development"
```

## Version Extraction Logic

The build system follows this priority order:

1. **Git tags** (release builds): Use `git describe --tags --exact-match` for tagged commits
2. **VERSION file + git hash** (development builds): Combine base version with git commit hash
3. **VERSION file only** (fallback): When git is unavailable or in CI shallow clones
4. **Hardcoded fallback**: `"dev"` when neither git nor VERSION file available

## Build System Integration

### Local Development
```bash
# Automatic version injection during build
devbox run build-cli  # Uses scripts/build-version.sh

# Manual version extraction for testing
bash scripts/build-version.sh  # Shows current version string
```

### GitHub Actions
- **CI builds**: Extract base version for SonarQube, full dev version for binaries
- **Release builds**: Use git tag version, trigger on `v*` tags
- **Version variables**: Available as `${{ steps.version.outputs.version }}` in workflows

### SonarQube Integration
- **Development**: Uses base version without `-dev` suffix (e.g., `2.0.0`)
- **Release**: Uses clean semantic version from git tags
- **Implementation**: Pass version via `--define sonar.projectVersion=<version>` parameter

## Version Validation

The `scripts/build-version.sh` script handles all edge cases:
- Detached HEAD state (uses commit hash)
- Shallow git clones (uses VERSION file fallback)
- Missing git command (VERSION-dev only)
- Invalid VERSION file format (falls back to "dev")

## Best Practices

- **Update VERSION file** when starting new version development cycle
- **Use semantic versioning** (major.minor.patch) for all versions
- **Test version output** with `--version` flag before releases
- **Verify GitHub Actions** trigger correctly on tag pushes
- **Keep -dev suffix** in VERSION file during development (removed only in git tags)

## Version Update Checklists

### Starting New Version Development
- [ ] Update VERSION file with new base version (e.g., `2.1.0-dev`)
- [ ] Commit VERSION file update with clear message
- [ ] Verify development builds show new version format
- [ ] Test `devbox run validate-version` passes

### Preparing for Release
- [ ] Ensure all planned features/fixes are complete
- [ ] Run full test suite and verify all tests pass
- [ ] Update CHANGELOG.md or release notes
- [ ] Verify VERSION file contains correct base version
- [ ] Test build and version extraction locally

### Creating Release
- [ ] Create git tag with `v` prefix: `git tag v2.1.0`
- [ ] Verify tag-based build shows clean version (no -dev suffix)
- [ ] Push tag to trigger GitHub Actions release: `git push origin v2.1.0`
- [ ] Monitor GitHub Actions release build completion
- [ ] Verify release artifacts are created correctly

### Post-Release
- [ ] Update VERSION file for next development cycle (e.g., `2.2.0-dev`)
- [ ] Commit VERSION file update
- [ ] Verify development builds resume with new -dev version format
- [ ] Update project documentation if needed

## Troubleshooting

### Version not updating
- Check git tag creation and push to remote
- Verify tag follows `v*` format

### CI version issues
- Ensure `fetch-depth: 0` in GitHub Actions checkout
- Check if running in shallow clone

### Build failures
- Verify `scripts/build-version.sh` is executable and in PATH
- Check VERSION file format is valid

### SonarQube version errors
- Check base version extraction strips `-dev` suffix correctly
- Verify version follows semantic versioning format