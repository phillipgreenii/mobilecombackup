# FEAT-027: Ensure Code Formatting in Development Workflow

## Status
- **Completed**: Not started
- **Priority**: medium
- **Ready for Implementation**: Yes

## Overview
Update agent behavior to ensure that code is automatically formatted before committing. Use the existing `devbox run formatter` script and integrate formatting into the quality workflow before tests/linting.

## Background
Consistent code formatting is essential for maintainability and reducing merge conflicts. Currently, there's no guarantee that code is formatted before completion of development tasks. This feature will ensure all code follows the project's formatting standards.

## Requirements
### Functional Requirements
- [ ] Use existing `devbox run formatter` script for code formatting
- [ ] Agents automatically format code before committing changes
- [ ] Run formatting before tests/linting to prevent formatting from affecting tests
- [ ] Ensure all agents apply formatting when they complete code changes

### Non-Functional Requirements
- [ ] Formatting should be fast and not impact development workflow
- [ ] Formatting rules should be consistent with project standards

## Design
### Approach
1. Verify existing `devbox run formatter` script works correctly
2. Update agent behavior to auto-format before committing
3. Integrate formatting into quality workflow: format → test → lint
4. Update CLAUDE.md to document formatting expectations for agents

### Implementation Notes
- Use existing `devbox run formatter` (runs `go fmt ./...`)
- Format before running tests/linting to ensure formatting doesn't negatively affect code
- Auto-format and continue (don't fail if code wasn't formatted)
- Agents should format once they're done working and before committing

## Tasks
- [ ] Verify `devbox run formatter` works correctly
- [ ] Update agent behavior to auto-format before committing
- [ ] Integrate formatting into quality workflow (format → test → lint)
- [ ] Update CLAUDE.md with agent formatting expectations
- [ ] Test that agents consistently format code before commits
- [ ] Document formatting step in development workflow

## Testing
### Unit Tests
- Verify formatting script works correctly
- Ensure all Go files are properly formatted

### Integration Tests
- Test that agents automatically format code before committing
- Verify formatting runs before tests/linting in workflow
- Test formatting doesn't break existing functionality

## Risks and Mitigations
- **Risk**: Formatting might conflict with developer preferences
  - **Mitigation**: Use standard Go formatting tools that are widely accepted
- **Risk**: Formatting might break code in edge cases
  - **Mitigation**: Ensure tests pass after formatting

## Dependencies
- Works with FEAT-028: Formatting should run before tests/linting
- Relates to FEAT-029: Agents need to format before auto-committing

## References
- Claude commands: .claude/commands/
- Devbox configuration: devbox.json
- Go formatting tools: gofmt, goimports
- Related: FEAT-028 (formatting should be run alongside tests and linting)

## Notes
This will improve code quality and consistency across the project.