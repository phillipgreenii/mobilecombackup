# FEAT-073: NixOS Flake Support

## Status
- **Completed**: 
- **Priority**: medium

## Overview
Add Nix flake support to the mobilecombackup project to enable easy installation and usage on NixOS systems and other systems using the Nix package manager. This feature will provide an alternative installation method that integrates with the Nix ecosystem while maintaining compatibility with existing build systems.

## Background
Currently, mobilecombackup requires building from source using devbox/Go toolchain or manual Go builds. NixOS users and Nix package manager users would benefit from:
- Declarative installation via Nix flakes
- Integration with Nix profiles and system configurations
- Reproducible builds with pinned dependencies
- Easy access via GitHub URL without local cloning
- Support for both direct running (`nix run`) and permanent installation (`nix profile install`)

This enhancement expands the project's accessibility to the growing NixOS and Nix ecosystem community.

## Requirements
### Functional Requirements
- [ ] Create flake.nix file defining the package
- [ ] Expose mobilecombackup binary as default package
- [ ] Support `nix run github:phillipgreen/mobilecombackup` usage pattern (unstable/main)
- [ ] Support `nix run github:phillipgreen/mobilecombackup?ref=vX.Y.Z` usage pattern (stable/release)
- [ ] Support `nix profile install github:phillipgreen/mobilecombackup` usage pattern (unstable/main)
- [ ] Support `nix profile install github:phillipgreen/mobilecombackup?ref=vX.Y.Z` usage pattern (stable/release)
- [ ] Maintain version information injection in Nix builds
- [ ] Update README.md with Nix installation instructions for both stable and unstable
- [ ] Support development environment setup via flake (`nix develop`)
- [ ] Document GitHub release workflow for Nix users

### Non-Functional Requirements
- [ ] Flake builds must be reproducible and deterministic
- [ ] Version information must match existing build-version.sh output
- [ ] Flake should use standard Nix Go build patterns
- [ ] Development environment should provide same tools as devbox
- [ ] Installation size should be reasonable (no unnecessary dependencies)
- [ ] Support both tag-based (stable) and branch-based (unstable) installations

## Design
### Approach
Create a standard Nix flake that:
1. **Package Definition**: Uses `buildGoModule` to create the mobilecombackup package
2. **Version Injection**: Implements equivalent of build-version.sh logic using Nix
3. **Development Shell**: Provides equivalent environment to devbox shell
4. **Multiple Outputs**: Supports both package installation and development workflows
5. **GitHub Integration**: Works when referenced directly from GitHub URL

### API/Interface
```nix
# flake.nix structure
{
  description = "A tool for processing mobile phone backup files";
  
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };
  
  outputs = { self, nixpkgs, flake-utils }: 
    # Package definition with version injection
    # Development shell with Go + tools
    # Default package export
}
```

### Data Structures
```nix
# Package attributes
{
  pname = "mobilecombackup";
  version = "..."; # from git or VERSION file
  src = ./.;
  vendorHash = "..."; # Go module hash
  ldflags = [ "-X main.Version=${version}" ];
  
  # Development shell tools
  devShell = pkgs.mkShell {
    buildInputs = [
      pkgs.go
      pkgs.golangci-lint
      pkgs.gotestsum
      # other devbox equivalents
    ];
  };
}
```

### Implementation Notes
- **Version Strategy**: Replicate build-version.sh logic in Nix to ensure version consistency
- **Go Module Handling**: Use standard `buildGoModule` with vendorHash for reproducibility  
- **Development Environment**: Mirror devbox.json package list in development shell
- **Cross-Platform**: Ensure flake works on multiple architectures (x86_64-linux, aarch64-linux, x86_64-darwin, aarch64-darwin)
- **CI Integration**: Consider adding flake check to CI pipeline
- **Documentation**: Clear examples for both end-users and NixOS system configurations

### Git Repository & GitHub Configuration
#### Repository Requirements
- **Git Tags**: Ensure semantic versioning tags (e.g., `v1.2.3`) are pushed to GitHub for stable releases
- **Release Workflow**: Continue using existing GitHub releases with proper tags
- **Branch Protection**: Main branch remains the unstable/development branch
- **Flake File**: `flake.nix` must be committed to the repository root

#### Current Release Process (from docs/VERSION_MANAGEMENT.md)
The project already has a well-defined release process that is compatible with Nix flake requirements:

1. **Development Phase**:
   - VERSION file contains development version (e.g., `2.1.0-dev`)
   - Builds show format: `2.1.0-dev-g1234567` (base + git hash)
   - Main branch serves as unstable/development channel

2. **Release Creation**:
   ```bash
   # Create and push git tag (triggers GitHub Actions release)
   git tag v2.1.0
   git push origin v2.1.0
   
   # GitHub Actions automatically builds release binaries on v* tags
   # Release builds show clean version: 2.1.0
   ```

3. **Post-Release**:
   ```bash
   # Update VERSION for next development cycle
   echo "2.2.0-dev" > VERSION
   git add VERSION
   git commit -m "Start v2.2.0 development"
   ```

4. **GitHub Actions Integration**:
   - Release workflow triggers on `v*` tags (see `.github/workflows/release.yml`)
   - Builds versioned binaries with clean semantic version
   - Uses `fetch-depth: 0` for full git history (required for version detection)

#### Flake Compatibility Requirements
- **No changes needed to existing release process** - Current workflow is fully compatible
- **Git tags format `v*`** already used and working
- **GitHub Actions** already configured for release builds
- **Version injection** via `scripts/build-version.sh` can be replicated in Nix

#### Version Selection Strategy
```bash
# Unstable (latest main branch)
nix run github:phillipgreen/mobilecombackup
nix profile install github:phillipgreen/mobilecombackup

# Stable (latest release tag)
nix run github:phillipgreen/mobilecombackup?ref=v1.2.3
nix profile install github:phillipgreen/mobilecombackup?ref=v1.2.3

# Specific commit
nix run github:phillipgreen/mobilecombackup?rev=abc123def

# Branch-based
nix run github:phillipgreen/mobilecombackup?ref=feature-branch
```

#### GitHub Configuration
- **No special GitHub configuration required** - Nix fetches directly from public repository
- **Release tags must be lightweight or annotated tags** pushed to GitHub
- **Consider adding Nix installation instructions to GitHub release notes**
- **Optional**: Add `flake.nix` and `flake.lock` to repository for pinned dependencies

#### Documentation Updates for README
```markdown
## Installation via Nix

### Unstable (Latest Development)
# Run directly from main branch (shows version like 2.1.0-dev-g1234567)
nix run github:phillipgreen/mobilecombackup -- --help

# Install to profile from main branch
nix profile install github:phillipgreen/mobilecombackup

### Stable (Latest Release)
# Run specific release version (shows clean version like 2.0.0)
nix run github:phillipgreen/mobilecombackup?ref=v2.0.0 -- --help

# Install specific release to profile
nix profile install github:phillipgreen/mobilecombackup?ref=v2.0.0

# List available releases
# Visit: https://github.com/phillipgreen/mobilecombackup/releases

### NixOS System Configuration
{ pkgs, ... }:
{
  environment.systemPackages = [
    (pkgs.callPackage (fetchGit {
      url = "https://github.com/phillipgreen/mobilecombackup";
      ref = "v2.0.0"; # Use specific release tag
      # ref = "main"; # Or use main for latest development
    }) {})
  ];
}

### Version Information
- Stable releases: Use git tags (e.g., v2.0.0) for reproducible builds
- Development builds: Use main branch for latest features
- Check version: mobilecombackup --version
```

#### Release Process Documentation for Flake Maintainers
When creating a new release that Nix users can install:

1. **Pre-release Checklist**:
   - Ensure all tests pass: `devbox run test`
   - Verify VERSION file has correct base version
   - Test flake locally: `nix build .#mobilecombackup`

2. **Create Release**:
   ```bash
   # Tag the release (this enables Nix stable installs)
   git tag v2.1.0
   git push origin v2.1.0
   
   # GitHub Actions will automatically:
   # - Build release binaries
   # - Create GitHub release
   # - Tag becomes available for Nix users immediately
   ```

3. **Verify Nix Installation Works**:
   ```bash
   # Test stable installation with new tag
   nix run github:phillipgreen/mobilecombackup?ref=v2.1.0 -- --version
   # Should show: mobilecombackup version 2.1.0
   
   # Test unstable still works
   nix run github:phillipgreen/mobilecombackup -- --version
   # Should show: mobilecombackup version 2.2.0-dev-g1234567
   ```

4. **Update Documentation**:
   - Update README examples if major version changed
   - Add Nix installation notes to GitHub release description
   - Consider announcing in Nix community channels

## Tasks
- [ ] Research best practices for Go flakes with version injection
- [ ] Create initial flake.nix with basic package definition
- [ ] Implement version injection equivalent to build-version.sh (matching current versioning)
- [ ] Add development shell with devbox-equivalent tools
- [ ] Test flake locally with `nix build`, `nix run`, `nix develop`
- [ ] Test GitHub URL usage pattern for main branch (unstable)
- [ ] Test GitHub URL usage pattern with ?ref= for tags (stable)
- [ ] Update README.md with comprehensive Nix installation instructions (stable vs unstable)
- [ ] Add Nix-specific notes to docs/VERSION_MANAGEMENT.md
- [ ] Create release checklist addition for Nix verification
- [ ] Add flake check to CI pipeline (optional)
- [ ] Test on multiple platforms (Linux, macOS if available)
- [ ] Create example NixOS system configuration snippets
- [ ] Verify flake.lock generation and commit strategy
- [ ] Test version selection strategies (main, tags, commits, branches)
- [ ] Verify version output matches existing build system format
- [ ] Document flake testing in release process

## Testing
### Unit Tests
- Verify flake builds successfully with `nix build`
- Confirm version injection works correctly
- Validate development shell includes all required tools
- Test cross-platform builds

### Integration Tests
- Test `nix run github:phillipgreen/mobilecombackup --help` (unstable/main)
- Test `nix run github:phillipgreen/mobilecombackup?ref=vX.Y.Z --help` (stable/tag)
- Test `nix profile install github:phillipgreen/mobilecombackup` (unstable)
- Test `nix profile install github:phillipgreen/mobilecombackup?ref=vX.Y.Z` (stable)
- Verify binary works identically to devbox-built version
- Test development workflow with `nix develop`
- Validate flake works without internet after initial fetch
- Test switching between stable and unstable versions
- Verify correct version string in both stable and unstable builds

### Edge Cases
- Handle missing git information gracefully
- Fallback version handling when VERSION file unavailable
- Network-less builds after initial flake fetch
- Clean handling of Go module vendoring

## Risks and Mitigations
- **Risk**: Nix learning curve for maintenance
  - **Mitigation**: Use standard patterns, document decisions, provide examples
- **Risk**: Version inconsistency between build methods
  - **Mitigation**: Careful implementation of version injection logic equivalence
- **Risk**: CI complexity increase
  - **Mitigation**: Make flake checking optional, provide clear documentation
- **Risk**: Dependency bloat in development shell
  - **Mitigation**: Mirror only essential tools from devbox.json
- **Risk**: Cross-platform build issues
  - **Mitigation**: Test on available platforms, use standard nixpkgs patterns
- **Risk**: Confusion between stable and unstable versions
  - **Mitigation**: Clear documentation, explicit version strings in builds
- **Risk**: Missing git tags breaking stable installations
  - **Mitigation**: Document release process requirements, add CI validation
- **Risk**: flake.lock causing merge conflicts
  - **Mitigation**: Document update strategy, consider .gitignore for lock file

## References
- Related features: FEAT-026 (devbox CI), FEAT-031 (versioning)
- Code locations: devbox.json, scripts/build-version.sh, VERSION file
- External docs: 
  - [Nix Flakes](https://nixos.wiki/wiki/Flakes)
  - [buildGoModule documentation](https://nixos.org/manual/nixpkgs/stable/#sec-language-go)
  - [Flake templates for Go projects](https://github.com/NixOS/templates)

## Notes
This feature enhances project accessibility without disrupting existing workflows. The flake should be seen as an additional installation method rather than a replacement for devbox-based development, which remains the primary development environment. Consider this feature as expanding the project's reach to the NixOS/Nix ecosystem while maintaining the current development philosophy and tooling.

### Stable vs Unstable Installation Strategy
- **Unstable (main branch)**: Default when no ref specified, tracks latest development
- **Stable (release tags)**: Requires explicit `?ref=vX.Y.Z`, uses GitHub releases
- **Version string format**: Must match existing build system:
  - Stable: Clean semantic version (e.g., "2.0.0") from git tags
  - Unstable: Development format (e.g., "2.1.0-dev-g1234567") from VERSION file + git hash
- **No repository changes needed**: Works with existing git workflow and GitHub releases
- **flake.lock handling**: Consider whether to commit for reproducibility vs .gitignore for flexibility

### Integration with Existing Release Process
The Nix flake will integrate seamlessly with the current release workflow:
1. **Version detection**: Replicate `scripts/build-version.sh` logic in Nix
2. **Git tags**: Use existing `v*` tag format (e.g., v2.0.0)
3. **GitHub Actions**: No changes needed to `.github/workflows/release.yml`
4. **VERSION file**: Continue using for development version tracking
5. **Release testing**: Add Nix verification to release checklist