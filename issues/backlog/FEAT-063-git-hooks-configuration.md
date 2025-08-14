# FEAT-063: Add Git Hooks Configuration

## Status
- **Priority**: low

## Overview
Implement pre-commit hooks to automatically run code quality checks (formatting, testing, linting) before commits, catching issues early in the development cycle.

## Background
Issues are currently caught late in the development cycle, after code is committed. Pre-commit hooks would catch formatting, testing, and linting issues before they enter the repository.

## Requirements
### Functional Requirements
- [ ] Create pre-commit hook script that runs quality checks
- [ ] Run formatter, tests, and linter before allowing commits
- [ ] Provide easy setup mechanism for developers
- [ ] Allow bypassing hooks for emergency commits
- [ ] Support both local development and CI environments

### Non-Functional Requirements
- [ ] Hooks should run quickly to not slow down development
- [ ] Should provide clear error messages when checks fail
- [ ] Should be easy to install and configure
- [ ] Should work across different development environments

## Design
### Approach
Create git hooks that integrate with existing devbox scripts and provide clear setup instructions.

### Implementation Notes
- Use existing devbox scripts for consistency
- Provide installation script for easy setup
- Include bypass mechanism for urgent commits
- Consider using tools like pre-commit or simple shell scripts

## Tasks
### Phase 1: Hook Script Development
- [ ] Create `.githooks/pre-commit` script with specific checks:
  - Run `devbox run formatter` (must pass)
  - Run `devbox run tests` (must pass)  
  - Run `devbox run linter` (must pass)
  - Performance target: complete within 30 seconds
- [ ] Add bypass mechanism: `git commit --no-verify` for emergencies
- [ ] Handle partial commits correctly (only check staged files)

### Phase 2: Installation and Integration
- [ ] Create `scripts/install-hooks.sh` installation script
- [ ] Update `.devbox.json` to include hook installation in init_hook
- [ ] Add git config setup: `git config core.hooksPath .githooks`
- [ ] Test hook behavior across different development environments (devbox, direct)

### Phase 3: Documentation and Guidelines
- [ ] Add hook documentation to CLAUDE.md development commands section
- [ ] Create troubleshooting guide for common hook failures
- [ ] Document bypass procedures for emergency commits
- [ ] Add hooks to new developer setup checklist

## Implementation Specifications
**Hook Script Requirements:**
```bash
#!/bin/sh
# Pre-commit hook - runs quality checks
set -e

echo "Running pre-commit checks..."

# Run formatter (fast, should complete in ~5s)
echo "1/3 Running formatter..."
devbox run formatter

# Run tests (slower, target ~20s)  
echo "2/3 Running tests..."
devbox run tests

# Run linter (fast, should complete in ~5s)
echo "3/3 Running linter..."
devbox run linter

echo "✓ All pre-commit checks passed"
```

**Performance Targets:**
- Total execution time: < 30 seconds
- Formatter: < 5 seconds
- Tests: < 20 seconds  
- Linter: < 5 seconds

## Testing
### Unit Tests
- Test hook script functionality
- Test bypass mechanisms

### Integration Tests
- Test hooks in real development workflow
- Test behavior with failing checks

### Edge Cases
- Hook behavior with partial commits
- Performance with large commits
- Hook behavior in CI environments

## Risks and Mitigations
- **Risk**: Hooks slowing down development workflow
  - **Mitigation**: Optimize hook performance and provide bypass options
- **Risk**: Developers bypassing hooks routinely
  - **Mitigation**: Make hooks fast and provide clear value

## References
- Source: CODE_IMPROVEMENT_REPORT.md item #15

## Notes
This will help maintain code quality consistency and catch issues before they enter the repository. Keep hooks lightweight and focused on essential quality checks.