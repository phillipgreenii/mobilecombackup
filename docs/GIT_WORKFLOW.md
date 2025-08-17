# Git Workflow

This document defines the standard Git workflow and commit rules that must be followed throughout the project.

## Core Principles

### ABSOLUTE COMMIT RULE

**CRITICAL**: Every task MUST end with a commit that passes ALL quality checks:

1. **NEVER** use `git commit --no-verify`
2. **ALL** tests MUST pass (not some, ALL)
3. **ZERO** linting errors allowed
4. Build MUST succeed
5. Code MUST be formatted

**ENFORCEMENT**:
- A task is NOT complete until a successful commit is made
- If ANY quality check fails, the task remains incomplete
- If there's a blocker preventing commit, STOP and ask for help
- Do NOT report task completion with failing tests/lint/build

## Pre-Commit Requirements

Before every commit, you MUST:

1. **Run verification workflow** (see [Verification Workflow](VERIFICATION_WORKFLOW.md))
2. **Pass all pre-commit hooks** (installed via `devbox run install-hooks`)
3. **Stage only relevant files** (never use `git add .`)

## File Staging Best Practices

### Stage Specific Files Only

```bash
# CORRECT: Stage specific files you modified
git add pkg/specific/file.go pkg/specific/file_test.go

# WRONG: Never stage everything
git add .
```

### File Detection Strategy

Use git status comparison to identify files you modified:

```bash
# Before starting task
git status --porcelain > /tmp/before_task

# After completing task  
git status --porcelain > /tmp/after_task

# Identify changed files and stage them
comm -13 /tmp/before_task /tmp/after_task | cut -c4- | xargs -r git add
```

**Important**: Only commit files you actually modified for the task - never stage unrelated changes.

## Commit Message Format

### Standard Template

```
[ISSUE-ID]: [Brief description]

[Optional: Detailed explanation if needed]

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

### Examples

**Feature Implementation:**
```
FEAT-055: Add context support to SMS parser

Implemented context passing through parser chain to enable
better error reporting and debugging capabilities.

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

**Bug Fix:**
```
BUG-123: Fix duplicate detection in call coalescer

Fixed edge case where calls with identical timestamps
were incorrectly identified as duplicates.

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

**Documentation Update:**
```
DOC: Update architecture overview in CLAUDE.md

Added new package descriptions and updated design
principles to reflect recent refactoring.

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

### Commit Message Guidelines

- **Issue Reference**: Always include issue ID when applicable
- **Brief Description**: Start with action verb (Add, Fix, Update, Remove)
- **Imperative Mood**: Use "Add feature" not "Added feature"
- **Detail Level**: Include details for complex changes
- **Attribution**: Always include Claude Code attribution footer

## Git Hooks

### Installation
```bash
devbox run install-hooks  # Install pre-commit hooks
devbox run test-hooks     # Test hooks without committing
```

### Hook Validation
Pre-commit hooks will automatically run:
- Code formatting checks
- Lint validation
- Test execution
- Build verification

**NEVER bypass hooks** with `--no-verify` flag.

## Commit Workflow

### Step-by-Step Process

1. **Complete your work** on a specific task
2. **Run verification workflow**:
   ```bash
   devbox run formatter  # Format first
   devbox run tests     # All tests pass
   devbox run linter    # Zero violations
   devbox run build-cli # Successful build
   ```
3. **Check git status** to see modified files
4. **Stage specific files** you modified
5. **Create commit** with proper message format
6. **Verify hooks pass** (never use `--no-verify`)

### Auto-Commit After Task Completion

After completing each TodoWrite task, ALWAYS commit your changes:

```bash
# Check what files changed
git status

# Stage only files you modified
git add path/to/modified/file1.go path/to/modified/file2_test.go

# Commit with proper format
git commit -m "$(cat <<'EOF'
FEAT-XXX: Brief task description

Optional implementation details

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

## Branch Management

### Main Branch
- **Name**: `main`
- **Protection**: All commits must pass pre-commit hooks
- **Usage**: Direct commits for features and fixes

### Working with Branches
- Check current branch: `git branch`
- Switch branches: `git checkout branch-name`
- Create new branch: `git checkout -b feature-branch`

## Integration with Development Workflow

This Git workflow integrates with:
- [Task Completion Requirements](TASK_COMPLETION.md)
- [Verification Workflow](VERIFICATION_WORKFLOW.md)
- [Issue Development Workflow](ISSUE_WORKFLOW.md)

## Error Handling

### If Commit Fails Due to Hooks:
1. **Read the hook error message** carefully
2. **Fix the identified issues**
3. **Re-run verification workflow**
4. **Try commit again**
5. **Ask for help** if issues persist

### If Verification Fails:
1. **Do NOT attempt to commit**
2. **Fix the failing checks** first
3. **Re-run complete verification**
4. **Only commit when ALL checks pass**

## Important Reminders

- **NEVER** use `git commit --no-verify`
- **NEVER** stage unrelated files with `git add .`
- **ALWAYS** run verification before committing
- **ALWAYS** include issue reference in commit messages
- **ALWAYS** use the standard commit message format
- Task completion REQUIRES successful commit with ALL checks passing