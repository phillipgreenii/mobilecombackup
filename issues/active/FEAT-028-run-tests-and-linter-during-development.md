# FEAT-028: Run Tests and Linter During Development

## Status
- **Completed**: Not started
- **Ready for Implementation**: 2025-08-11
- **Priority**: high

## Overview
Ensure that before completing any development task, agents verify clean state by running all tests and linting, fixing any failures or violations. Agents can use incremental testing during development for efficiency, but must ensure zero test failures and zero lint violations before marking tasks complete.

## Background
Currently, tests and linting might not be consistently run before task completion, leading to issues being discovered during commits or reviews. By requiring agents to verify clean state before completing tasks and fix any issues found, we can maintain higher code quality and reduce review cycles. Agents can develop efficiently using incremental testing, but must ensure quality at completion.

## Requirements
### Functional Requirements
- [ ] Before marking any task complete, agents must run `devbox run tests` 
- [ ] Before marking any task complete, agents must run `devbox run linter`
- [ ] Before marking any task complete, agents must run `devbox run build-cli`
- [ ] Agents must fix all test failures found during completion verification
- [ ] Agents must fix all lint violations found during completion verification
- [ ] Agents must fix all build failures found during completion verification
- [ ] Agents should ask user when unsure how to fix an issue
- [ ] Agents can use incremental testing during development for efficiency
- [ ] Zero test failures, zero lint violations, and successful build required for task completion

### Non-Functional Requirements
- [ ] Auto-fixes should be safe and not introduce new issues
- [ ] Completion verification should provide clear feedback on what was fixed

## Design
### Completion Workflow
1. Agent completes code changes for a task
2. Agent runs `devbox run tests` to verify all tests pass
3. If test failures found, agent fixes them and re-runs tests
4. Agent runs `devbox run linter` to verify no lint violations
5. If lint violations found, agent fixes them and re-runs linter
6. Agent runs `devbox run build-cli` to verify clean build
7. If build failures found, agent fixes them and re-runs build
8. Only after tests, linting, and build are all clean, agent marks task complete

### Development Process
- Agents MAY use incremental testing during development (`go test ./pkg/specific` or similar)
- Agents MAY run targeted lints on specific files during development
- Final verification MUST run complete test suite and linter
- Agents ask user when unsure how to fix test failures or lint violations

### Commands
- **Full Test Suite**: `devbox run tests` or `go test -v -covermode=set ./...`
- **Full Linter**: `devbox run linter` or `golangci-lint run`
- **CLI Build**: `devbox run build-cli` or `go build -o mobilecombackup ./cmd/mobilecombackup`
- **Acceptable Development Commands**: Any subset for efficiency during coding

### Auto-Fix Strategy
- Fix all test compilation errors (missing imports, typos, etc.)
- Fix all lint violations (formatting, unused variables, etc.)
- Fix all build compilation errors (missing dependencies, syntax errors, etc.)
- Ask user when fix might change business logic or when multiple valid approaches exist

## Common Fix Patterns
### Test Failure Patterns
- **`undefined: functionName`**: Add missing import or fix typo in function name
- **`cannot use x (type A) as type B`**: Add type conversion or fix function signature
- **`declared but not used`**: Remove unused variable or add usage
- **Missing test files**: Create required test data files or directories
- **Permission errors**: Fix file/directory permissions in tests

### Lint Violation Patterns  
- **`declared but not used`**: Remove unused variables, imports, or functions
- **`Error return value is not checked`**: Add proper error handling
- **`should have comment or be unexported`**: Add documentation comments
- **Formatting issues**: Run `gofmt` or equivalent formatting
- **Import ordering**: Use `goimports` to fix import organization

### When to Ask User
- Test logic appears incorrect (wrong expected values)
- Multiple valid approaches to fix a lint violation
- Fix would significantly change program behavior
- Unfamiliar error patterns not covered by common fixes

## Tasks
- [ ] Update CLAUDE.md with completion verification requirements
- [ ] Update .claude/agents/ with completion workflow instructions
- [ ] Document common test failure patterns and their fixes
- [ ] Document common lint violations and their fixes  
- [ ] Create agent instruction templates for completion verification
- [ ] Test completion workflow with intentionally broken code
- [ ] Integrate with TodoWrite tool to enforce completion verification

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
  - **Mitigation**: Only auto-fix well-understood, safe patterns; ask user for complex issues
- **Risk**: Agents might get stuck in fix/retry loops
  - **Mitigation**: Limit retry attempts; escalate to user after repeated failures
- **Risk**: Full test runs might be slow on large codebases
  - **Mitigation**: Accept the cost for quality; agents can use incremental testing during development

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