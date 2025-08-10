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
- [ ] Update GitHub Actions workflow to use `jetify-com/devbox-install-action`
- [ ] Remove direct dependency installation from GitHub Actions
- [ ] Ensure all existing CI functionality is preserved

### Non-Functional Requirements
- [ ] CI build time should not significantly increase
- [ ] CI configuration should be simpler and more maintainable

## Design
### Approach
1. Define a `ci` script in devbox.json that encapsulates all CI steps
2. Update the GitHub Actions workflow to install devbox and run the ci script
3. Remove explicit dependency installations from the workflow

### Implementation Notes
- The `jetify-com/devbox-install-action` GitHub Action should be used
- The ci script should include building, testing, and linting
- Consider caching devbox installations for faster CI runs

## Tasks
- [ ] Add `ci` script to devbox.json
- [ ] Update GitHub Actions workflow to use devbox-install-action
- [ ] Remove direct dependency installations from workflow
- [ ] Test CI pipeline with the new configuration
- [ ] Update documentation if needed

## Testing
### Integration Tests
- Verify CI passes with the new configuration
- Ensure all build artifacts are created correctly
- Confirm test results are reported properly

### Edge Cases
- Handle CI failures gracefully
- Ensure devbox installation errors are clearly reported

## Risks and Mitigations
- **Risk**: CI build time might increase due to devbox installation
  - **Mitigation**: Implement proper caching of devbox installations
- **Risk**: Devbox action might have compatibility issues
  - **Mitigation**: Pin to a stable version of the action

## Dependencies
None - This is a foundational improvement that other features can build upon.

## References
- Current CI configuration: .github/workflows/
- Devbox documentation: https://www.jetpack.io/devbox/docs/
- jetify-com/devbox-install-action: https://github.com/jetify-com/devbox-install-action

## Notes
This change will improve the development workflow by ensuring CI uses the same environment as local development.