# FEAT-031: Consistent Versioning Scheme

## Status
- **Completed**: 
- **Priority**: high

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
2. VERSION file: Plain text file containing current version for development
3. Hardcoded fallback: `"dev"` when neither git nor VERSION file available

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
# VERSION file format (plain text)
2.0.0

# Git tag format (semantic versioning)
v2.0.0
v2.1.0-rc1
v2.1.0

# Version extraction logic in devbox.json
VERSION=$(cat VERSION 2>/dev/null || echo "dev")
# For release builds:
VERSION=$(git describe --tags --exact-match 2>/dev/null || cat VERSION)
```

### Implementation Notes
**VERSION File Management:**
- Plain text file in repository root containing semantic version (e.g., "2.0.0")
- Updated manually by developers when starting new version development
- Checked into git for consistency across development environments
- Used as fallback when git tags not available (CI environments, shallow clones)

**Git Tag Integration:**
- Release process creates git tags in format `v2.0.0`
- `git describe --tags` provides automatic version with commit distance for development builds
- Release builds use exact tag matches via `git describe --tags --exact-match`

**Build System Updates:**
- Update `devbox.json` build-cli script to use VERSION file fallback
- Maintain existing ldflags injection pattern
- Add version validation to ensure VERSION file contains valid semantic version

**SonarQube Integration:**
- Template or script-based approach to inject version into `sonar-project.properties`
- Alternative: Read VERSION file during SonarQube analysis setup

## Tasks
- [ ] Create VERSION file with initial version 2.0.0
- [ ] Update devbox.json build-cli script to use VERSION file fallback
- [ ] Document version update workflow for developers
- [ ] Update FEAT-030 SonarQube properties to use VERSION file
- [ ] Add version validation script to verify VERSION file format
- [ ] Create git tag v2.0.0 for current state
- [ ] Test version injection in all build scenarios (dev, CI, release)
- [ ] Update project documentation with versioning scheme
- [ ] Add version update checklist to development workflow

## Testing
### Unit Tests
- Test version injection with various VERSION file contents
- Test fallback behavior when VERSION file missing
- Test git tag extraction in different repository states
- Validate semantic version format parsing

### Integration Tests
- Build CLI binary and verify `--version` output matches VERSION file
- Test devbox build-cli command with and without git tags
- Verify SonarQube properties contain correct version
- Test GitHub Actions release workflow with version injection

### Edge Cases
- VERSION file contains invalid format (non-semantic version)
- Git repository in detached HEAD state
- Shallow git clone without tag history
- VERSION file contains trailing whitespace or newlines
- Build environment without git command available
- Concurrent version updates in different branches

## Risks and Mitigations
- **Risk**: VERSION file gets out of sync with actual release tags
  - **Mitigation**: Document clear workflow for version updates; add validation in CI to ensure VERSION file matches latest tag during releases

- **Risk**: Developers forget to update VERSION file when starting new version
  - **Mitigation**: Include VERSION file update in issue implementation workflow; add reminder in development documentation

- **Risk**: Git tag extraction fails in CI environments
  - **Mitigation**: Use VERSION file as reliable fallback; test CI scenarios extensively

- **Risk**: SonarQube integration becomes complex with dynamic versioning
  - **Mitigation**: Use simple file-based approach; template generation if needed; ensure FEAT-030 implementation accounts for this

## References
- Related features: FEAT-030 (SonarQube Cloud Integration) - version in properties file
- Related features: FEAT-026 (Update CI to Use Devbox) - CI build integration
- Code locations: cmd/mobilecombackup/main.go (version injection)
- Code locations: devbox.json (build-cli script)
- Code locations: .github/workflows/release.yml (release builds)
- External docs: [Go Build Constraints](https://pkg.go.dev/go/build#hdr-Build_Constraints)
- External docs: [Semantic Versioning 2.0.0](https://semver.org/)

## Notes
- Initial version set to 2.0.0 as specified in requirements
- Approach balances simplicity with robustness - git tags for releases, VERSION file for development
- VERSION file approach ensures version availability even in constrained CI environments
- Design maintains backward compatibility with existing devbox commands
- Consider automating VERSION file updates in future workflow improvements
- SonarQube integration in FEAT-030 should be implemented after this versioning scheme is established