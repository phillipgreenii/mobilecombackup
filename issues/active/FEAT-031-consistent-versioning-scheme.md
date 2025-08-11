# FEAT-031: Consistent Versioning Scheme

## Status
- **Completed**: 
- **Priority**: high
- **Ready for Implementation**: 2025-08-11

## Overview
Establish and document a consistent versioning scheme that minimizes maintenance overhead while providing version information across all project touchpoints: code, SonarQube properties, GitHub Actions, and CLI output. The scheme should provide a single source of truth for version information and simple process for version updates.

## Background
The project currently has versioning scattered across multiple locations with inconsistent approaches:

1. **Code**: `main.go` uses ldflags injection with `Version = "dev"` fallback
2. **GitHub Actions**: `release.yml` triggers on git tags but doesn't consistently inject version
3. **DevBox**: `build-cli` script uses `git describe --tags` for version extraction  
4. **SonarQube**: FEAT-030 hardcodes `sonar.projectVersion=1.0` in properties file
5. **Current State**: No clear versioning process, no git tags, inconsistent version information

This fragmentation creates maintenance burden when releasing new versions and potential inconsistencies across deployment artifacts. A unified approach will:

- Reduce version update overhead from multiple files to single action
- Ensure consistency across all project artifacts and configurations
- Provide clear developer workflow for version management
- Support semantic versioning principles (major.minor.patch)
- Integrate seamlessly with existing devbox and CI workflows

## Requirements
### Functional Requirements
- [ ] Single source of truth for version information
- [ ] Automated version injection into CLI binary during build
- [ ] Version consistency across SonarQube properties, GitHub Actions, and code
- [ ] Simple developer workflow to update project version
- [ ] Support for semantic versioning (2.0.0 format)
- [ ] Integration with existing devbox build commands
- [ ] Git tag-based versioning for releases
- [ ] Development version handling (pre-release identifiers)

### Non-Functional Requirements
- [ ] Version update process should take less than 2 minutes
- [ ] Version information should be available at runtime via `--version` flag
- [ ] Build system should work offline (not dependent on network for version)
- [ ] Version scheme should support CI/CD automation
- [ ] Backward compatibility with existing `devbox run build-cli` command

## Design
### Approach
**Git Tag-Based Versioning with VERSION File Fallback**

Primary approach uses git tags as the authoritative source for release versions, with a `VERSION` file providing fallback for development and CI environments where git history might not be available.

**Version Sources (Priority Order):**
1. Git tags: `git describe --tags --always --dirty` for release builds
2. VERSION file: Plain text file containing current version with "-dev" suffix for development
3. Hardcoded fallback: `"dev"` when neither git nor VERSION file available

**Version Format Examples:**
- Release version (from git tag): `v2.0.0`
- Development version (from VERSION file + git): `2.0.0-dev-g1234567`
- VERSION file content: `2.0.0-dev`
- Git tag format: `v2.0.0` (with "v" prefix)

**Integration Points:**
- **CLI Binary**: Version injected via ldflags during `go build`
- **SonarQube**: Properties file reads from VERSION file or uses templating
- **GitHub Actions**: Uses git tags for release builds, VERSION file for other builds
- **Development**: VERSION file updated manually when starting new version cycle

### API/Interface
```go
// No new interfaces needed - leverages existing version injection in main.go
var (
    // Version is set via ldflags during build
    Version   = "dev"        // Fallback when no version injection
    BuildTime = "unknown"    // Existing build time injection
)

// Version information accessible via:
// 1. CLI: `mobilecombackup --version`  
// 2. Runtime: cmd.SetVersion(Version) in root.go
```

### Data Structures
```bash
# VERSION file format (plain text with -dev suffix)
2.0.0-dev

# Git tag format (semantic versioning with v prefix)
v2.0.0
v2.1.0-rc1
v2.1.0

# Development version string format
# VERSION-dev-gHASH where:
# - VERSION: base version from VERSION file (without -dev)
# - dev: development indicator
# - g: git prefix (standard git describe convention)
# - HASH: 7-character git commit hash
# Example: 2.0.0-dev-g1234567
```

### Version Extraction Logic
```bash
# Updated devbox.json build-cli script
# Extract base version from VERSION file (removing -dev suffix)
BASE_VERSION=$(cat VERSION 2>/dev/null | sed 's/-dev$//' || echo "dev")

# For development builds (default):
# Get git hash for dev suffix
GIT_HASH=$(git rev-parse --short=7 HEAD 2>/dev/null || echo "unknown")
if [ "$GIT_HASH" != "unknown" ]; then
    VERSION="${BASE_VERSION}-dev-g${GIT_HASH}"
else
    VERSION="${BASE_VERSION}-dev"
fi

# For release builds (when on exact tag):
GIT_TAG=$(git describe --tags --exact-match 2>/dev/null)
if [ -n "$GIT_TAG" ]; then
    # Remove v prefix for version string
    VERSION=${GIT_TAG#v}
fi

# Build with version injection
go build -ldflags "-X main.Version=$VERSION" -o mobilecombackup ./cmd/mobilecombackup
```

### Implementation Notes
**VERSION File Management:**
- Plain text file in repository root containing semantic version with "-dev" suffix (e.g., "2.0.0-dev")
- The "-dev" suffix indicates active development; removed only when creating release tags
- Updated manually by developers when starting new version development cycle
- Checked into git for consistency across development environments
- Used as base for development version strings (VERSION-dev-gHASH format)
- Standard practice in Go projects for version tracking alongside git tags

**Git Tag Integration:**
- Release process creates git tags in format `v2.0.0` (with "v" prefix, following Go conventions)
- Development builds append git commit hash to VERSION file content
- Release builds use exact tag matches via `git describe --tags --exact-match`
- When on a tagged commit, the tag version takes precedence over VERSION file

**Build System Updates:**
- Update `devbox.json` build-cli script to handle VERSION file with "-dev" suffix
- Extract git commit hash for development version format (VERSION-dev-gHASH)
- Maintain existing ldflags injection pattern
- Add version validation to ensure VERSION file contains valid semantic version with optional "-dev" suffix
- Handle both development and release build scenarios seamlessly

**SonarQube Integration:**
- Dynamic version injection using GitHub Actions workflow steps
- Extract version before SonarQube scan and pass via `--define sonar.projectVersion=<version>`
- For development builds: use base version without "-dev" suffix (e.g., "2.0.0")
- For release builds: use git tag version
- Avoid using build numbers; use semantic versions as per SonarQube best practices
- Implementation in FEAT-030 should account for dynamic version extraction

## Tasks
- [ ] Create VERSION file with initial version 2.0.0-dev
- [ ] Update devbox.json build-cli script with complete version extraction logic:
  - Extract base version from VERSION file (remove -dev suffix)
  - Append git hash for dev builds (VERSION-dev-gHASH format)
  - Use git tag for release builds when available
- [ ] Update GitHub Actions workflows:
  - Ensure `fetch-depth: 0` for full git history in release workflow
  - Add version extraction step before SonarQube scan
  - Configure go-releaser-action to use git tags for version detection
- [ ] Document version update workflow for developers:
  - When to update VERSION file
  - How to create release tags
  - Development vs release version formats
- [ ] Update FEAT-030 SonarQube integration:
  - Add dynamic version extraction in GitHub workflow
  - Use `--define sonar.projectVersion=<version>` parameter
  - Strip -dev suffix for SonarQube reporting
- [ ] Add version validation script to verify VERSION file format (semantic version with optional -dev)
- [ ] Create git tag v2.0.0 for initial release
- [ ] Test version injection in all build scenarios:
  - Local development builds (with git hash)
  - CI builds (shallow clone handling)
  - Release builds (exact tag match)
- [ ] Update project documentation with versioning scheme
- [ ] Add version update checklist to development workflow

## Testing
### Unit Tests
- Test version injection with various VERSION file contents (with and without -dev suffix)
- Test -dev suffix removal logic
- Test git hash extraction and formatting (7 characters)
- Test fallback behavior when VERSION file missing
- Test git tag extraction in different repository states
- Validate semantic version format parsing (including -dev variants)
- Test version string assembly (VERSION-dev-gHASH format)

### Integration Tests
- Build CLI binary and verify `--version` output:
  - Development build shows VERSION-dev-gHASH format
  - Release build shows clean semantic version
- Test devbox build-cli command:
  - With VERSION file containing -dev suffix
  - On exact git tag (should use tag version)
  - In detached HEAD state
  - Without git available
- Verify SonarQube version injection:
  - Development builds use base version (no -dev)
  - Release builds use tag version
- Test GitHub Actions workflows:
  - go-releaser-action with git tag triggers
  - SonarQube scan with dynamic version
  - Proper fetch-depth configuration

### Edge Cases
- VERSION file contains invalid format (non-semantic version)
- VERSION file missing -dev suffix (handle gracefully)
- Git repository in detached HEAD state (use commit hash)
- Shallow git clone without tag history (use VERSION file fallback)
- VERSION file contains trailing whitespace or newlines (trim)
- Build environment without git command available (VERSION-dev only)
- Concurrent version updates in different branches
- Git hash extraction fails (use "unknown" fallback)
- Building directly on a tag vs building from tag + commits
- VERSION file has version that doesn't match any git tags

## Risks and Mitigations
- **Risk**: VERSION file gets out of sync with actual release tags
  - **Mitigation**: Document clear workflow for version updates; add validation in CI to ensure VERSION file matches latest tag during releases

- **Risk**: Developers forget to update VERSION file when starting new version
  - **Mitigation**: Include VERSION file update in issue implementation workflow; add reminder in development documentation

- **Risk**: Git tag extraction fails in CI environments
  - **Mitigation**: Use VERSION file as reliable fallback; test CI scenarios extensively

- **Risk**: SonarQube integration becomes complex with dynamic versioning
  - **Mitigation**: Use GitHub Actions workflow steps for version extraction; pass version via --define parameter; follow SonarQube best practices avoiding build numbers

## References
- Related features: FEAT-030 (SonarQube Cloud Integration) - version in properties file
- Related features: FEAT-026 (Update CI to Use Devbox) - CI build integration
- Code locations: cmd/mobilecombackup/main.go (version injection)
- Code locations: devbox.json (build-cli script)
- Code locations: .github/workflows/release.yml (release builds)
- External docs: [Go Build Constraints](https://pkg.go.dev/go/build#hdr-Build_Constraints)
- External docs: [Semantic Versioning 2.0.0](https://semver.org/)

## Workflow Examples

### Developer Workflow for Version Updates
```bash
# 1. Start new version development (after v2.0.0 release)
echo "2.1.0-dev" > VERSION
git add VERSION
git commit -m "Start v2.1.0 development"

# 2. During development, builds show: 2.1.0-dev-g1234567

# 3. Create release
git tag v2.1.0
git push origin v2.1.0
# Release builds now show: 2.1.0

# 4. Post-release, update VERSION for next cycle
echo "2.2.0-dev" > VERSION
git add VERSION
git commit -m "Start v2.2.0 development"
```

### GitHub Actions Configuration
```yaml
# .github/workflows/release.yml
name: Release
on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Required for version detection
      
      - name: Extract version for SonarQube
        id: version
        run: |
          # For tagged releases, use tag version
          VERSION=${GITHUB_REF#refs/tags/v}
          echo "version=$VERSION" >> $GITHUB_OUTPUT
      
      - name: SonarQube Scan
        uses: SonarSource/sonarqube-scan-action@v2
        with:
          args: >
            -Dsonar.projectVersion=${{ steps.version.outputs.version }}
      
      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v5
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

# .github/workflows/ci.yml
name: CI
on:
  push:
    branches: [ main ]
  pull_request:

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # For git describe
      
      - name: Extract version
        id: version
        run: |
          # Extract base version and build dev version string
          BASE_VERSION=$(cat VERSION | sed 's/-dev$//')
          GIT_HASH=$(git rev-parse --short=7 HEAD)
          DEV_VERSION="${BASE_VERSION}-dev-g${GIT_HASH}"
          # For SonarQube, use base version only
          echo "dev_version=$DEV_VERSION" >> $GITHUB_OUTPUT
          echo "sonar_version=$BASE_VERSION" >> $GITHUB_OUTPUT
      
      - name: Build with version
        run: |
          devbox run go build -ldflags "-X main.Version=${{ steps.version.outputs.dev_version }}" \
            -o mobilecombackup ./cmd/mobilecombackup
      
      - name: SonarQube Scan
        uses: SonarSource/sonarqube-scan-action@v2
        with:
          args: >
            -Dsonar.projectVersion=${{ steps.version.outputs.sonar_version }}
```

## Notes
- Initial version set to 2.0.0-dev as starting point for development
- Development version format: VERSION-dev-gHASH (e.g., 2.0.0-dev-g1234567)
- Release version format: clean semantic version from git tags (e.g., 2.0.0)
- VERSION file always contains -dev suffix except in release tags
- Approach follows Go community standards for version management
- Git hash uses 7 characters following git describe conventions
- SonarQube receives clean semantic versions (no -dev or hash suffixes)
- GitHub Actions requires fetch-depth: 0 for proper git history access
- go-releaser-action automatically detects versions from git tags
- Design maintains backward compatibility with existing devbox commands
- Consider automating VERSION file updates post-release in future improvements