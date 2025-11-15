# Verification Workflow

This document defines the standard verification workflow that must be followed before marking any task complete or committing code changes.

---

**Last Updated**: 2025-01-15
**Related Documents**: [Git Workflow](GIT_WORKFLOW.md) | [Task Completion](TASK_COMPLETION.md) | [Development Guide](DEVELOPMENT.md)
**Prerequisites**: Devbox shell environment, pre-commit hooks installed

---

## Overview

The verification workflow ensures code quality, consistency, and functionality through a series of mandatory checks. **All verification commands must pass before any task can be marked complete.**

## Verification Commands

The verification workflow consists of four commands that **MUST** be run in this exact order:

```bash
# 1. Format code first (ALWAYS run first)
devbox run formatter

# 2. Run all tests (ALL must pass)
devbox run tests

# 3. Check code quality (ZERO violations)
devbox run linter

# 4. Build the CLI (must succeed)
devbox run build-cli
```

### Command Details

#### 1. Format Code (`devbox run formatter`)
- **Purpose**: Ensures consistent code formatting across the project
- **Requirement**: MUST be run first, before any other verification
- **Success Criteria**: Command completes without errors
- **Notes**: Formats all Go code according to project standards

#### 2. Run Tests (`devbox run tests`)
- **Purpose**: Validates all functionality works correctly
- **Requirement**: ALL tests must pass (not some, ALL)
- **Success Criteria**: Zero test failures, zero compilation errors
- **Notes**: Runs complete test suite including unit and integration tests

#### 3. Run Linter (`devbox run linter`)
- **Purpose**: Enforces code quality and Go best practices
- **Requirement**: ZERO lint violations allowed
- **Success Criteria**: No warnings, no errors, clean output
- **Notes**: Checks for code quality issues, unused variables, missing docs

#### 4. Build CLI (`devbox run build-cli`)
- **Purpose**: Ensures the application compiles and builds successfully
- **Requirement**: Build must succeed without errors
- **Success Criteria**: Executable is created without compilation errors
- **Notes**: Final check that all code integrates properly

## Development vs Completion Verification

### During Development (Optional for Efficiency)
During active development, you MAY use targeted commands for faster feedback:

```bash
# Test specific package
go test ./pkg/specific

# Lint specific area
golangci-lint run ./pkg/specific

# Quick build check
go build ./pkg/specific
```

### Before Task Completion (MANDATORY)
Before marking any TodoWrite task complete, you **MUST** run the full verification workflow:

```bash
devbox run formatter  # MUST be first
devbox run tests     # ALL tests MUST pass
devbox run linter    # ZERO violations
devbox run build-cli # MUST succeed
```

**CRITICAL**: Task is NOT complete without running and passing all four commands.

## Success Criteria

All verification commands must meet these criteria:

| Command | Success Criteria |
|---------|------------------|
| `formatter` | Completes without errors, code is formatted |
| `tests` | Zero failures, zero compilation errors |
| `linter` | Zero violations, clean output |
| `build-cli` | Builds successfully, executable created |

## Failure Handling

### If ANY Command Fails:
1. **Fix the issues** identified by the failing command
2. **Re-run the complete verification workflow** from the beginning
3. **Repeat until ALL commands pass**
4. **Only then** proceed with task completion or commit

### Common Failure Patterns:

**Test Failures:**
- Import errors → Fix imports or add missing dependencies
- Type mismatches → Add proper type conversions
- Unused variables → Remove or use the variables
- Missing test data → Create required files in `testdata/`

**Lint Violations:**
- Unused code → Remove unused variables/imports/functions
- Missing error checks → Add proper error handling
- Missing documentation → Add comments for exported functions
- Formatting issues → Ensure formatter ran successfully

**Build Failures:**
- Missing imports → Add required import statements
- Syntax errors → Fix Go syntax issues
- Dependency issues → Run `go mod tidy`

## Integration with Other Workflows

This verification workflow is referenced by:
- [Task Completion Requirements](TASK_COMPLETION.md)
- [Git Workflow](GIT_WORKFLOW.md) (before commits)
- All agent implementations
- Issue implementation commands

## Notes

- **Never skip verification steps** - they prevent production issues
- **Order matters** - formatting must be done first
- **All or nothing** - partial success is not acceptable
- **Commit requirement** - verification is prerequisite for commits
- See [Common Fixes](COMMON_FIXES.md) for detailed fix patterns