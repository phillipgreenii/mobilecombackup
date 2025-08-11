# FEAT-026: Update CI to Use Devbox

## Status
- **Completed**: Not started
- **Priority**: high

## Overview
Update the continuous integration (CI) pipeline to use devbox instead of directly installing dependencies. This will ensure consistency between local development and CI environments.

## Background
Currently, the CI pipeline installs dependencies directly in the GitHub Actions workflow. By switching to devbox, we can:
- Ensure the same versions and tools are used in both local development and CI
- Simplify CI configuration 
- Reduce maintenance burden when updating dependencies
- Improve build reproducibility

## Requirements
### Functional Requirements
- [ ] Add a `ci` script to devbox.json that runs the necessary CI steps
- [ ] Update both GitHub Actions workflows (test.yml and release.yml) to use `jetify-com/devbox-install-action`
- [ ] Remove direct dependency installation from GitHub Actions
- [ ] Ensure all existing CI functionality is preserved
- [ ] CI will use whatever Go version devbox has configured (currently 1.24)

### Non-Functional Requirements
- [ ] CI configuration should be simpler and more maintainable
- [ ] Build time increase is acceptable for consistency benefits

## Design
### Approach
1. Define a `ci` script in devbox.json that encapsulates all CI steps
2. Update both GitHub Actions workflows (test.yml and release.yml) to install devbox and run the ci script
3. Remove explicit dependency installations from both workflows

### CI Script Configuration
The `ci` script in devbox.json should include all current CI steps:
```json
{
  "scripts": {
    "ci": [
      "devbox run formatter",
      "devbox run tests", 
      "devbox run linter",
      "devbox run build-cli"
    ]
  }
}
```

### Implementation Notes
- The `jetify-com/devbox-install-action` GitHub Action should be used for both workflows
- CI will automatically use the Go version configured in devbox.json (currently 1.24)
- Replace direct Go setup and dependency installation with devbox installation
- No parallel old/new CI configuration needed - direct migration

## Tasks
- [ ] Add `ci` script to devbox.json with formatter, tests, linter, and build-cli steps
- [ ] Update test.yml workflow to use devbox-install-action and remove direct Go setup
- [ ] Update release.yml workflow to use devbox-install-action and remove direct Go setup
- [ ] Remove direct dependency installations from both workflows
- [ ] Test both CI workflows with the new configuration
- [ ] Update README.md and documentation with new CI approach

## Testing
### Integration Tests
- Verify both test.yml and release.yml workflows pass with devbox configuration
- Ensure all build artifacts are created correctly in both workflows
- Confirm test results and coverage reports are generated properly
- Verify Go 1.24 is used (upgraded from previous 1.16.x)

### Edge Cases
- Handle devbox installation failures gracefully
- Ensure devbox action errors are clearly reported in CI logs
- Test behavior when devbox.json is malformed

## Risks and Mitigations
- **Risk**: Go version upgrade from 1.16.x to 1.24 might cause compatibility issues
  - **Mitigation**: Test thoroughly; Go maintains backward compatibility
- **Risk**: Devbox action might have compatibility issues
  - **Mitigation**: Pin to a stable version of the action
- **Risk**: CI build time will increase due to devbox installation
  - **Mitigation**: Acceptable tradeoff for environment consistency

## Dependencies
None - This is a foundational improvement that other features can build upon.

## References
- Current CI configuration: .github/workflows/
- Devbox documentation: https://www.jetpack.io/devbox/docs/
- jetify-com/devbox-install-action: https://github.com/jetify-com/devbox-install-action

## Notes
This change will improve the development workflow by ensuring CI uses the same environment as local development.