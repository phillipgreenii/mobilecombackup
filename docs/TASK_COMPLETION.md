# Task Completion Requirements

This document defines what it means for a task to be "complete" and the mandatory requirements that must be met before marking any TodoWrite task as finished.

## Definition of Complete

A task is **COMPLETE** only when:

1. **All work is finished** according to the task specification
2. **All verification commands pass** (see [Verification Workflow](VERIFICATION_WORKFLOW.md))
3. **A successful commit is made** that passes all pre-commit hooks
4. **No quality issues remain** (tests, linting, build)

**CRITICAL**: A task is NOT complete until a successful commit is made.

## Mandatory Requirements

### Before Marking Task Complete

**ALL** of the following must succeed:

```bash
devbox run formatter  # Code must be formatted
devbox run tests     # All tests must pass
devbox run linter    # Zero lint violations
devbox run build-cli # Build must succeed
```

### Success Criteria

| Requirement | Success Criteria |
|-------------|------------------|
| **Formatting** | Code is consistently formatted, no formatting errors |
| **Tests** | ALL tests pass (not some, ALL), zero compilation errors |
| **Linting** | ZERO violations, clean output |
| **Build** | Successful build, executable created |
| **Commit** | Successful commit that passes all pre-commit hooks |

## Task Completion Workflow

### Step-by-Step Process

1. **Complete the work** specified in the task
2. **Run verification workflow**:
   ```bash
   devbox run formatter  # Always format first
   devbox run tests     # All must pass
   devbox run linter    # Zero violations
   devbox run build-cli # Must succeed
   ```
3. **Fix any issues** found by verification
4. **Re-run verification** until ALL commands pass
5. **Commit changes** using [Git Workflow](GIT_WORKFLOW.md)
6. **Mark task complete** in TodoWrite only after successful commit

### Fix-Retry Cycle

If ANY verification command fails:

1. **Fix the issues** identified by the failing command
2. **Re-run the COMPLETE verification workflow** from the beginning
3. **Repeat until ALL commands pass**
4. **Do NOT mark task complete** until all checks pass

## TodoWrite Task Management

### Task Status Rules

- **pending**: Task not yet started
- **in_progress**: Currently working on (limit to ONE task at a time)
- **completed**: Task finished successfully with all requirements met

### Task Status Updates

- **Only ONE task** should be `in_progress` at any time
- **Mark task complete IMMEDIATELY** after successful commit
- **Don't batch completions** - update status in real-time
- **Remove irrelevant tasks** entirely from the list

### Task Completion Requirements

**ONLY mark a task as completed when:**

- ✅ You have FULLY accomplished the task requirements
- ✅ ALL verification commands pass
- ✅ Successful commit is made
- ✅ No errors, blockers, or partial implementation remains

**NEVER mark a task as completed if:**

- ❌ Tests are failing
- ❌ Implementation is partial
- ❌ You encountered unresolved errors
- ❌ You couldn't find necessary files or dependencies
- ❌ Verification commands fail
- ❌ Commit failed due to pre-commit hooks

## Blocker Escalation Process

### When to Stop and Ask for Help

Stop working and ask for guidance when:

- **Repeated failures** after multiple fix attempts
- **Unfamiliar error patterns** not covered in [Common Fixes](COMMON_FIXES.md)
- **Multiple valid approaches** to fix an issue
- **Fix would significantly change** program behavior or business logic
- **Pre-commit hooks consistently fail** despite following all guidelines
- **Unable to understand** test failure or lint violation

### How to Report Blockers

When blocked, provide:

1. **What you were trying to accomplish**
2. **What commands you ran**
3. **Complete error messages**
4. **What you've tried to fix it**
5. **Why you're unsure how to proceed**

## Integration with Development Workflow

### Related Documentation

This document integrates with:
- [Verification Workflow](VERIFICATION_WORKFLOW.md) - Commands to run
- [Git Workflow](GIT_WORKFLOW.md) - Commit requirements
- [Common Fixes](COMMON_FIXES.md) - Fix patterns for issues
- [Issue Workflow](ISSUE_WORKFLOW.md) - Overall development process

### Agent Integration

All agents must follow these completion requirements:
- **spec-implementation-engineer**: Implements features with completion verification
- **code-completion-verifier**: Specializes in ensuring completion requirements
- **product-doc-sync**: Updates documentation with same completion rules

## Incremental Development

### During Development (Optional)

For faster feedback during active development, you MAY use:
- `go test ./pkg/specific` for targeted testing
- `golangci-lint run ./pkg/specific` for focused linting
- Quick builds with `go build ./pkg/specific`

### Final Verification (MANDATORY)

Before task completion, you MUST run the complete verification workflow regardless of incremental testing done during development.

## Quality Standards

### Zero Tolerance Policy

- **No test failures** - ALL tests must pass
- **No lint violations** - ZERO warnings or errors
- **No build errors** - Clean, successful builds only
- **No bypass mechanisms** - Never use `--no-verify`

### Performance Considerations

Balance thoroughness with efficiency:
- Use incremental testing during development
- Run full verification before completion
- Fix issues immediately when found
- Don't accumulate technical debt

## Examples

### ✅ Correct Task Completion

```
1. Implement feature X
2. Run devbox run formatter - ✅ Success
3. Run devbox run tests - ✅ All pass
4. Run devbox run linter - ✅ Zero violations
5. Run devbox run build-cli - ✅ Build succeeds
6. Commit changes - ✅ Hooks pass
7. Mark task complete in TodoWrite
```

### ❌ Incorrect Task Completion

```
1. Implement feature X
2. Run devbox run tests - ❌ 2 tests fail
3. Mark task complete anyway ← WRONG!
```

The task should remain `in_progress` until ALL verification passes and commit succeeds.

## Important Reminders

- **Task completion REQUIRES successful commit** with ALL checks passing
- **No exceptions** - partial success is not acceptable
- **Fix issues immediately** - don't accumulate problems
- **One task at a time** - focus on complete quality
- **Ask for help** when blocked rather than compromising quality