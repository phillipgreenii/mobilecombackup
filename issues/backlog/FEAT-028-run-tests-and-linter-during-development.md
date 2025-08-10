# FEAT-028: Run Tests and Linter During Development

## Status
- **Completed**: Not started
- **Priority**: high

## Overview
Ensure that during development, both tests and the linter are run automatically. Agents should be configured to fix any errors from failed tests or lint violations before proceeding.

## Background
Currently, tests and linting might not be consistently run during development, leading to issues being discovered late in the process. By running these checks automatically and having agents fix issues proactively, we can maintain higher code quality and reduce review cycles.

## Requirements
### Functional Requirements
- [ ] Configure development workflow to run tests automatically
- [ ] Configure development workflow to run linter automatically
- [ ] Ensure agents fix test failures when possible
- [ ] Ensure agents fix lint violations
- [ ] Provide clear error messages when issues cannot be auto-fixed

### Non-Functional Requirements
- [ ] Tests and linting should not significantly slow down development
- [ ] Auto-fixes should be safe and not introduce new issues

## Design
### Approach
1. Update agent configurations to include test and lint steps
2. Configure agents to analyze and fix common test failures
3. Configure agents to fix lint violations automatically
4. Ensure proper error reporting when auto-fix is not possible

### Implementation Notes
- Leverage golangci-lint for comprehensive linting
- Use `go test` with appropriate flags for testing
- Consider running tests in watch mode during development
- Agents should understand common Go error patterns

## Tasks
- [ ] Update agent configurations to include test/lint steps
- [ ] Document common test failure patterns and fixes
- [ ] Document common lint violations and fixes
- [ ] Create examples of auto-fixable issues
- [ ] Update development workflow documentation
- [ ] Test agent behavior with failing tests/lint issues

## Testing
### Unit Tests
- Verify agents correctly identify test failures
- Verify agents correctly identify lint violations

### Integration Tests
- Test complete development workflow with intentional failures
- Verify agents fix issues appropriately

### Edge Cases
- Handle cases where auto-fix is not possible
- Handle conflicting lint rules
- Handle flaky tests

## Risks and Mitigations
- **Risk**: Auto-fixes might introduce bugs
  - **Mitigation**: Only auto-fix well-understood, safe patterns
- **Risk**: Constant test/lint runs might slow development
  - **Mitigation**: Run incrementally on changed files only

## Dependencies
- Benefits from: FEAT-027 (code formatting should be part of the automated checks)
  - While not strictly required, having formatting in place first makes the automated workflow more complete

## References
- Agent configurations: .claude/agents/
- Linter configuration: .golangci.yml
- Test documentation: CLAUDE.md testing section
- Related: FEAT-027 (ensure code formatting)
- Related: FEAT-029 (auto-commit after successful tests/linting)

## Notes
This will significantly improve code quality and reduce the feedback loop for developers.