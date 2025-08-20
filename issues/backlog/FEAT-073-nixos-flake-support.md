# FEAT-073: NixOS Flake Support

## Status
- **Completed**: 
- **Priority**: medium

## Overview
Add Nix flake support to the mobilecombackup project to enable easy installation and usage on NixOS systems and other systems using the Nix package manager. This feature will provide an alternative installation method that integrates with the Nix ecosystem while maintaining compatibility with existing build systems.

## Prerequisites
- **Nix Knowledge**: Basic understanding of Nix expressions and flakes
- **Go Module System**: Understanding of Go module vendoring and dependencies
- **Version Management**: Familiarity with project's versioning system (FEAT-031)
- **Release Process**: Understanding of current GitHub release workflow
- **Testing Access**: Access to at least Linux system with Nix installed
- **Repository Access**: Ability to push to main branch and create tags

## Business Value
- **User Reach**: Expands access to NixOS and Nix package manager community (~30k+ active users)
- **Distribution**: Enables zero-configuration installation on NixOS systems
- **Reproducibility**: Provides deterministic builds across all environments
- **User Experience**: Simplifies installation with single-command deployment
- **Adoption**: Lowers barriers for organizations using Nix for infrastructure management

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
#### Core Package Requirements
- [ ] Create flake.nix file defining the package with proper schema and metadata
- [ ] Expose mobilecombackup binary as default package (`packages.default`)
- [ ] Implement version injection matching `scripts/build-version.sh` behavior exactly
- [ ] Support all binary subcommands (import, export, validate, version, etc.)
- [ ] Preserve CLI flags and configuration file support
- [ ] Ensure `nix flake check` validates the flake successfully

#### Installation Methods
- [ ] Support `nix run github:phillipgreen/mobilecombackup` usage pattern (unstable/main)
- [ ] Support `nix run github:phillipgreen/mobilecombackup?ref=vX.Y.Z` usage pattern (stable/release)
- [ ] Support `nix profile install github:phillipgreen/mobilecombackup` usage pattern (unstable/main)
- [ ] Support `nix profile install github:phillipgreen/mobilecombackup?ref=vX.Y.Z` usage pattern (stable/release)
- [ ] Enable NixOS module integration via `environment.systemPackages`
- [ ] Support home-manager integration for user-level installation

#### Documentation Requirements
- [ ] Update README.md with Nix installation instructions for both stable and unstable
- [ ] Document GitHub release workflow for Nix users
- [ ] Provide NixOS configuration examples
- [ ] Include troubleshooting guide for common Nix issues
- [ ] Add flake usage examples to docs/INSTALLATION.md (create if needed)
- [ ] Clearly document that devbox remains the primary development environment

### Non-Functional Requirements
#### Build & Reproducibility
- [ ] Flake builds must be reproducible and deterministic (fixed vendorHash)
- [ ] Version information must match existing build-version.sh output exactly
- [ ] Build time should be comparable to devbox builds (<60 seconds)
- [ ] Support offline builds after initial dependency fetch
- [ ] Binary size should match or improve upon current builds (~15-20MB)

#### Compatibility
- [ ] Flake should use standard Nix Go build patterns (buildGoModule)
- [ ] Support multiple architectures: x86_64-linux, aarch64-linux, x86_64-darwin, aarch64-darwin
- [ ] Maintain compatibility with Nix 2.4+ (flakes experimental feature)
- [ ] Work with both standalone Nix and NixOS systems
- [ ] Flake must pass `nix flake check` validation
- [ ] Package must be installable and runnable without errors

#### Performance & Resources
- [ ] Installation size should be reasonable (<50MB including dependencies)
- [ ] Runtime dependencies should be minimal (no unnecessary packages)
- [ ] Memory usage during build should be reasonable (<2GB)
- [ ] Support parallel builds where applicable

#### Version Management
- [ ] Support both tag-based (stable) and branch-based (unstable) installations
- [ ] Version output format must be identical to current implementation
- [ ] Handle missing git information gracefully (e.g., in tarball downloads)
- [ ] Support version extraction from both git tags and VERSION file
- [ ] Leverage existing VERSION file and release workflow without modification
- [ ] Minimize version update locations (ideally just VERSION file)

## Design
### Approach
Create a standard Nix flake that:
1. **Package Definition**: Uses `buildGoModule` to create the mobilecombackup package
2. **Version Injection**: Implements equivalent of build-version.sh logic using Nix
3. **Package-Only Focus**: Provides installable package without development shell
4. **GitHub Integration**: Works when referenced directly from GitHub URL
5. **Validation**: Ensures flake passes `nix flake check` for quality assurance

**Note**: Development continues to use devbox exclusively. The Nix flake is for package distribution only.

### Technical Architecture
#### Flake Structure
- **inputs**: nixpkgs (unstable), flake-utils for multi-platform support
- **outputs**: packages.default (binary) - no development shell
- **overlays**: Optional overlay for integration with existing Nix configurations
- **checks**: Build verification, flake validation, and version format tests

#### Version Detection Strategy
```
Priority Order (matching build-version.sh):
1. Git tag (exact match) → Clean version (e.g., "2.0.0")
2. VERSION file + git hash → Dev version (e.g., "2.1.0-dev-g1234567")
3. VERSION file only → Dev version without hash (e.g., "2.1.0-dev")
4. Fallback → "dev" (when no version information available)
```

#### Dependency Management
- Use `vendorHash` for reproducible Go module fetching
- Pin nixpkgs input to specific revision for stability
- Document process for updating vendorHash when dependencies change

### API/Interface
```nix
# flake.nix structure (complete example)
{
  description = "A tool for processing mobile phone backup files";
  
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };
  
  outputs = { self, nixpkgs, flake-utils }: 
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        version = # Version detection logic here
      in {
        packages = {
          default = pkgs.buildGoModule { 
            # Package definition
          };
          mobilecombackup = self.packages.${system}.default;
        };
        
        devShells.default = pkgs.mkShell {
          # Development environment
        };
        
        apps.default = flake-utils.lib.mkApp {
          drv = self.packages.${system}.default;
        };
      }
    );
}
```

### Data Structures
```nix
# Package definition attributes
{
  pname = "mobilecombackup";
  version = detectVersion { inherit src; };  # Custom version detection
  src = ./.;
  vendorHash = "sha256-XXXXXX...";  # Computed from go.sum
  
  # Version injection matching current build
  ldflags = [ 
    "-X main.Version=${version}"
    "-s -w"  # Strip debug info for smaller binary
  ];
  
  # Build configuration
  CGO_ENABLED = 0;  # Static binary
  
  # Metadata
  meta = with lib; {
    description = "Tool for processing mobile phone backup files";
    homepage = "https://github.com/phillipgreen/mobilecombackup";
    license = licenses.mit;  # Update to actual license
    maintainers = with maintainers; [ ];  # Add maintainer info
    platforms = platforms.unix;
  };
}

# Development shell configuration
{
  buildInputs = with pkgs; [
    go_1_24
    gopls
    golangci-lint
    gotestsum
    ast-grep
    jq
    yq
    # Special handling for devbox-specific tools
  ];
  
  shellHook = ''
    echo "📋 Nix development environment loaded"
    echo "Run 'go test ./...' to run tests"
    echo "Run 'go build ./cmd/mobilecombackup' to build"
  '';
}
```

### Implementation Details
#### Version Injection Implementation
```nix
# Pseudo-code for version detection in Nix
version = let
  gitTag = builtins.tryEval (builtins.readFile ./.git/refs/tags/...);
  versionFile = builtins.readFile ./VERSION;
  baseVersion = lib.removeSuffix "-dev" versionFile;
  gitHash = if (builtins.pathExists ./.git) 
    then builtins.substring 0 7 (builtins.readFile ./.git/HEAD)
    else "unknown";
in
  if gitTag.success then
    lib.removePrefix "v" gitTag.value
  else if gitHash != "unknown" then
    "${baseVersion}-dev-g${gitHash}"
  else if versionFile != "" then
    "${baseVersion}-dev"
  else
    "dev";
```

#### Go Module Handling
- Use standard `buildGoModule` with computed `vendorHash`
- Document vendorHash update process: `nix-prefetch-git` or `nix build --impure`
- Consider using `vendorSha256 = null` during development, fixed hash for releases
- Ensure `go.mod` and `go.sum` are included in source

#### Development Environment Strategy
- **No Nix development shell** - Development uses devbox exclusively
- **Single configuration source** - All development tools configured in devbox.json only
- **Clear separation** - Nix for package distribution, devbox for development
- **Documentation clarity** - All development docs point to devbox commands

#### Cross-Platform Support
- Use `flake-utils.lib.eachDefaultSystem` for multi-architecture support
- Test on available platforms, document any platform-specific limitations
- Consider Darwin-specific handling for macOS builds
- Ensure CGO_ENABLED=0 for static binary builds (matching current build)

#### CI/CD Integration
- Add `nix flake check` to GitHub Actions workflow for validation
- Ensure `nix flake check` verifies the flake can be installed and run
- Document in release checklist: "Verify Nix flake check passes before tagging"
- Consider caching Nix store in CI for faster builds
- Add smoke test: `nix run .#mobilecombackup -- --version`
- Flake check should validate version format and basic functionality

#### Documentation Strategy
- Update README.md with prominent Nix installation section
- Create docs/NIX_INSTALLATION.md for detailed package installation instructions
- Clearly state that devbox is the only supported development environment
- Add troubleshooting for common issues (flakes not enabled, etc.)
- Provide examples for both end-users and NixOS system administrators
- Remove any references to `nix develop` from all documentation

### Git Repository & GitHub Configuration
#### Repository Requirements
- **Git Tags**: Ensure semantic versioning tags (e.g., `v1.2.3`) are pushed to GitHub for stable releases
- **Release Workflow**: Continue using existing GitHub releases with proper tags
- **Branch Protection**: Main branch remains the unstable/development branch
- **Flake File**: `flake.nix` must be committed to the repository root

#### Simplified Release Process
The existing release process works perfectly with Nix flakes:

1. **Single Version Source**: VERSION file is the only place to update version
2. **Simple Release Command**:
   ```bash
   # One-step release (could be scripted)
   git tag v2.1.0 && git push origin v2.1.0
   ```
3. **Automatic Processing**:
   - GitHub Actions builds release binaries automatically
   - Nix users can immediately install via `?ref=v2.1.0`
   - Version shows clean semantic version (e.g., "2.1.0")

4. **Post-Release**:
   ```bash
   # Update VERSION for next cycle
   echo "2.2.0-dev" > VERSION
   git add VERSION && git commit -m "Start v2.2.0 development"
   ```

**Key Benefits**:
- Version only updated in VERSION file
- Single command can trigger release
- No changes needed to existing workflow
- Fully automated after tag push

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

#### Complete Example flake.nix
```nix
# Example complete flake.nix implementation
{
  description = "Tool for processing mobile phone backup files";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        
        # Version detection matching build-version.sh
        detectVersion = src: let
          versionFile = builtins.readFile (src + "/VERSION");
          baseVersion = pkgs.lib.removeSuffix "-dev" (pkgs.lib.removeSuffix "\n" versionFile);
          gitDir = src + "/.git";
          hasGit = builtins.pathExists gitDir;
          
          # Try to get git information
          gitInfo = if hasGit then
            let
              headFile = gitDir + "/HEAD";
              headContent = if builtins.pathExists headFile 
                then builtins.readFile headFile 
                else "";
              isTag = pkgs.lib.hasPrefix "ref: refs/tags/" headContent;
              tagName = if isTag 
                then pkgs.lib.removePrefix "ref: refs/tags/" headContent
                else "";
            in {
              isTag = isTag;
              tagName = pkgs.lib.removeSuffix "\n" tagName;
              shortRev = if !isTag && headContent != ""
                then builtins.substring 0 7 (pkgs.lib.removeSuffix "\n" headContent)
                else "unknown";
            }
          else {
            isTag = false;
            tagName = "";
            shortRev = "unknown";
          };
        in
          if gitInfo.isTag && gitInfo.tagName != "" then
            # Release build: use git tag version (remove v prefix)
            pkgs.lib.removePrefix "v" gitInfo.tagName
          else if gitInfo.shortRev != "unknown" then
            # Development build: append git hash
            "${baseVersion}-dev-g${gitInfo.shortRev}"
          else if baseVersion != "" then
            # Fallback to VERSION file
            "${baseVersion}-dev"
          else
            # Final fallback
            "dev";
            
      in {
        packages = {
          default = pkgs.buildGoModule rec {
            pname = "mobilecombackup";
            version = detectVersion ./.;
            
            src = ./.;
            
            # Update this hash when go.mod/go.sum changes
            vendorHash = "sha256-XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX";
            
            # Match current build flags
            ldflags = [
              "-X main.Version=${version}"
              "-s -w"
            ];
            
            # Static binary
            CGO_ENABLED = 0;
            
            # Skip tests during build (run separately)
            doCheck = false;
            
            meta = with pkgs.lib; {
              description = "Tool for processing mobile phone backup files";
              homepage = "https://github.com/phillipgreen/mobilecombackup";
              license = licenses.mit;  # Update to actual license
              maintainers = [ ];
              platforms = platforms.unix;
            };
          };
          
          mobilecombackup = self.packages.${system}.default;
        };
        
        apps.default = flake-utils.lib.mkApp {
          drv = self.packages.${system}.default;
        };
        
        # No development shell - use devbox for development
        # This flake is for package distribution only
        
        # Checks for nix flake check validation
        checks = {
          # Verify package builds
          build = self.packages.${system}.default;
          
          # Verify binary runs and shows help
          runnable = pkgs.runCommand "check-runnable" {} ''
            ${self.packages.${system}.default}/bin/mobilecombackup --help > /dev/null
            echo "✓ Binary runs successfully"
            touch $out
          '';
          
          # Verify version format
          version = pkgs.runCommand "check-version" {} ''
            VERSION=$(${self.packages.${system}.default}/bin/mobilecombackup --version)
            echo "Version: $VERSION"
            
            # Check version format
            if echo "$VERSION" | grep -qE '^mobilecombackup version [0-9]+\.[0-9]+\.[0-9]+(-dev-g[0-9a-f]{7})?$'; then
              echo "✓ Version format is correct"
              touch $out
            else
              echo "✗ Version format is incorrect"
              exit 1
            fi
          '';
          
          # Verify all subcommands are accessible
          subcommands = pkgs.runCommand "check-subcommands" {} ''
            for cmd in import export validate info init version; do
              ${self.packages.${system}.default}/bin/mobilecombackup $cmd --help > /dev/null 2>&1
              echo "✓ Subcommand '$cmd' works"
            done
            touch $out
          '';
        };
      }
    ) // {
      # Overlay for advanced users
      overlays.default = final: prev: {
        mobilecombackup = self.packages.${final.system}.default;
      };
    };
}
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
### Research & Planning
- [ ] Research best practices for Go flakes with version injection
- [ ] Study similar Go projects with package-only Nix flakes
- [ ] Determine optimal nixpkgs channel (unstable vs stable)
- [ ] Decide on flake.lock commit strategy (commit vs .gitignore)
- [ ] Design simplified release process/script

### Implementation
- [ ] Create initial flake.nix with package definition (no dev shell)
- [ ] Implement version injection equivalent to build-version.sh
- [ ] Add proper flake metadata (description, maintainers)
- [ ] Compute and set correct vendorHash for Go modules
- [ ] Implement multi-platform support using flake-utils
- [ ] Add overlay output for advanced users
- [ ] Create comprehensive flake checks (build, run, version, subcommands)
- [ ] Ensure flake passes `nix flake check` validation
- [ ] Create release helper script for one-command releases (optional)

### Testing & Validation
- [ ] Test flake locally with `nix build` and `nix run`
- [ ] Verify `nix flake check` passes all validations
- [ ] Test that flake can be installed and run successfully
- [ ] Verify binary functionality (import, export, validate commands)
- [ ] Test GitHub URL usage pattern for main branch (unstable)
- [ ] Test GitHub URL usage pattern with ?ref= for tags (stable)
- [ ] Test on multiple platforms (Linux required, macOS if available)
- [ ] Verify flake.lock generation and reproducibility
- [ ] Test version selection strategies (main, tags, commits, branches)
- [ ] Verify version output matches existing build system format exactly
- [ ] Test offline builds after initial fetch
- [ ] Test `nix profile` installation and removal
- [ ] Validate all flake checks work correctly
- [ ] Test simplified release process end-to-end

### Documentation
- [ ] Update README.md with Nix package installation instructions
- [ ] Create docs/NIX_INSTALLATION.md for package installation only
- [ ] Update all docs to clarify devbox-only development
- [ ] Add Nix-specific notes to docs/VERSION_MANAGEMENT.md
- [ ] Create example NixOS system configuration snippets
- [ ] Document home-manager integration example
- [ ] Add troubleshooting section for common Nix issues
- [ ] Document vendorHash update process
- [ ] Create release checklist addition for `nix flake check` verification
- [ ] Document simplified release process/script
- [ ] Remove any `nix develop` references from documentation

### CI/CD Integration
- [ ] Add `nix flake check` to CI pipeline for validation
- [ ] Configure flake check to verify installation and runtime
- [ ] Document flake check requirement in release process
- [ ] Update release workflow documentation
- [ ] Consider adding Nix cache configuration
- [ ] Ensure flake check validates all critical functionality

## Testing
### Unit Tests
- Verify flake builds successfully with `nix build`
- Confirm version injection works correctly for all scenarios
- Validate development shell includes all required tools
- Test cross-platform builds where available
- Verify binary size and dependencies are reasonable

### Integration Tests
#### Installation Testing
- Test `nix run github:phillipgreen/mobilecombackup -- --help` (unstable/main)
- Test `nix run github:phillipgreen/mobilecombackup?ref=vX.Y.Z -- --help` (stable/tag)
- Test `nix profile install github:phillipgreen/mobilecombackup` (unstable)
- Test `nix profile install github:phillipgreen/mobilecombackup?ref=vX.Y.Z` (stable)
- Test `nix profile remove mobilecombackup` (cleanup)

#### Functionality Testing
- Verify binary works identically to devbox-built version
- Test all subcommands (import, export, validate, version)
- Verify configuration file loading works
- Test with sample backup files from testdata/
- Confirm version output format matches exactly

#### Flake Validation Testing
- Test `nix flake check` passes all checks
- Verify flake metadata is correct
- Test flake outputs are properly defined
- Verify checks validate installation capability
- Test checks validate runtime functionality

#### Reproducibility Testing
- Validate flake works without internet after initial fetch
- Test switching between stable and unstable versions
- Verify builds are reproducible (same hash for same input)
- Test with pinned and unpinned nixpkgs

### Edge Cases
- Handle missing git information gracefully (tarball downloads)
- Fallback version handling when VERSION file unavailable
- Network-less builds after initial flake fetch
- Clean handling of Go module vendoring
- Binary execution on systems without Nix
- Handling of shallow git clones
- Missing or corrupted flake.lock file
- Concurrent `nix profile` operations

### Acceptance Criteria
- [ ] Binary runs successfully on target platforms
- [ ] Version string matches format: "X.Y.Z" (stable) or "X.Y.Z-dev-gHASH" (unstable)
- [ ] All CLI commands function identically to devbox build
- [ ] Installation via GitHub URL works without local clone
- [ ] `nix flake check` passes all validations
- [ ] Flake can be installed and run without errors
- [ ] Builds are reproducible across different machines
- [ ] Documentation clearly states devbox-only development
- [ ] Release process is simplified (ideally single command)
- [ ] Version updates required in only one location (VERSION file)

## Risks and Mitigations
### Technical Risks
- **Risk**: Nix learning curve for maintenance
  - **Mitigation**: Use standard patterns, document decisions, provide examples
  - **Mitigation**: Create maintenance guide for non-Nix developers
  - **Contingency**: Maintain as optional feature, keep devbox as primary

- **Risk**: Version inconsistency between build methods
  - **Mitigation**: Careful implementation of version injection logic equivalence
  - **Mitigation**: Add version comparison tests in CI
  - **Mitigation**: Single source of truth (build-version.sh logic)

- **Risk**: CI complexity increase
  - **Mitigation**: Make flake checking optional and non-blocking
  - **Mitigation**: Provide clear documentation and troubleshooting
  - **Mitigation**: Use GitHub Actions matrix for Nix-specific jobs

### Dependency Risks
- **Risk**: User confusion about development environment
  - **Mitigation**: No Nix development shell provided
  - **Mitigation**: Clear documentation that devbox is the only dev environment
  - **Mitigation**: All development commands reference devbox only

- **Risk**: vendorHash maintenance burden
  - **Mitigation**: Document update process clearly
  - **Mitigation**: Add script to compute vendorHash automatically
  - **Mitigation**: Include in dependency update checklist

### Platform Risks
- **Risk**: Cross-platform build issues
  - **Mitigation**: Test on available platforms, use standard nixpkgs patterns
  - **Mitigation**: Document platform-specific limitations
  - **Mitigation**: Provide platform compatibility matrix

- **Risk**: Darwin (macOS) specific issues
  - **Mitigation**: Test on macOS if available
  - **Mitigation**: Document as "best effort" support
  - **Mitigation**: Engage community for testing/feedback

### User Experience Risks
- **Risk**: Confusion between stable and unstable versions
  - **Mitigation**: Clear documentation with examples
  - **Mitigation**: Explicit version strings in builds
  - **Mitigation**: Add "channel" info to version output (stable/unstable)

- **Risk**: Missing git tags breaking stable installations
  - **Mitigation**: Document release process requirements
  - **Mitigation**: Add CI validation for tag presence
  - **Mitigation**: Provide tag listing command in documentation

### Process Risks
- **Risk**: flake.lock causing merge conflicts
  - **Mitigation**: Document update strategy clearly
  - **Mitigation**: Consider .gitignore for lock file initially
  - **Mitigation**: Use automated dependency updates if committing

- **Risk**: Release process complexity
  - **Mitigation**: Keep existing simple workflow unchanged
  - **Mitigation**: Provide optional one-command release script
  - **Mitigation**: Version only updated in VERSION file
  - **Mitigation**: Add `nix flake check` to release checklist
  - **Mitigation**: Automate flake validation in CI

## References
### Related Features
- FEAT-026: Devbox CI integration (parallel build system)
- FEAT-031: Version management (version injection requirements)
- FEAT-072: Pre-commit hook optimization (development workflow)

### Code Locations
- `devbox.json`: Development tools to mirror
- `scripts/build-version.sh`: Version detection logic to replicate
- `VERSION`: Base version file
- `go.mod` / `go.sum`: Go module dependencies
- `.github/workflows/release.yml`: Release process integration
- `cmd/mobilecombackup/main.go`: Version variable injection point

### External Documentation
- [Nix Flakes Guide](https://nixos.wiki/wiki/Flakes)
- [buildGoModule documentation](https://nixos.org/manual/nixpkgs/stable/#sec-language-go)
- [Flake templates for Go projects](https://github.com/NixOS/templates)
- [Nix Pills](https://nixos.org/guides/nix-pills/) - Understanding Nix fundamentals
- [Zero to Nix](https://zero-to-nix.com/) - Modern Nix introduction
- [Example Go Flakes](https://github.com/topics/nix-flake?l=go) - Reference implementations

### Similar Projects for Reference
- [Hugo](https://github.com/gohugoio/hugo) - Static site generator with Nix support
- [Caddy](https://github.com/caddyserver/caddy) - Web server with flake
- [age](https://github.com/FiloSottile/age) - Encryption tool with clean flake

## Security Considerations
- **Supply Chain Security**: Pin nixpkgs to specific commits with hash verification
- **Reproducible Builds**: Use fixed vendorHash to ensure consistent Go module fetching
- **Binary Integrity**: Builds should be reproducible and verifiable
- **Dependency Auditing**: Document all runtime and build dependencies
- **Sandboxing**: Nix builds run in isolated environments by default
- **No Network Access**: Builds should work offline after initial fetch
- **Minimal Attack Surface**: Static binary with no runtime dependencies

## Success Metrics
- **Installation Success Rate**: >95% successful installations via Nix
- **Version Consistency**: 100% match with devbox build version strings
- **Build Reproducibility**: Identical binaries for same inputs
- **User Adoption**: Measurable increase in NixOS user base
- **Maintenance Burden**: <2 hours per month for Nix-specific issues
- **CI Integration**: No increase in CI failure rate

## Notes
This feature enhances project accessibility without disrupting existing workflows. The flake should be seen as an additional installation method rather than a replacement for devbox-based development, which remains the primary development environment. Consider this feature as expanding the project's reach to the NixOS/Nix ecosystem while maintaining the current development philosophy and tooling.

### Stable vs Unstable Installation Strategy
- **Unstable (main branch)**: Default when no ref specified, tracks latest development
- **Stable (release tags)**: Requires explicit `?ref=vX.Y.Z`, uses GitHub releases
- **Version string format**: Must match existing build system:
  - Stable: Clean semantic version (e.g., "2.0.0") from git tags
  - Unstable: Development format (e.g., "2.1.0-dev-g1234567") from VERSION file + git hash
- **No repository changes needed**: Works with existing git workflow and GitHub releases
- **flake.lock handling**: Initial recommendation - .gitignore to avoid conflicts, revisit after stability

### Integration with Existing Release Process
The Nix flake will integrate seamlessly with the current release workflow:
1. **Version detection**: Replicate `scripts/build-version.sh` logic in Nix
2. **Git tags**: Use existing `v*` tag format (e.g., v2.0.0)
3. **GitHub Actions**: No changes needed to `.github/workflows/release.yml`
4. **VERSION file**: Continue using for development version tracking
5. **Release testing**: Add Nix verification to release checklist

### Implementation Priority
1. **Phase 1**: Basic flake with package definition and version injection
2. **Phase 2**: Comprehensive flake checks and multi-platform support
3. **Phase 3**: Documentation (package-only) and CI integration
4. **Phase 4**: Simplified release tooling and community feedback

### Open Questions for Product Decision
1. Should flake.lock be committed initially or added to .gitignore?
2. Should `nix flake check` be required or informational in CI?
3. Should we support overlay output in initial version?
4. Should we create a release helper script for one-command releases?
5. Should flake check failures block releases?