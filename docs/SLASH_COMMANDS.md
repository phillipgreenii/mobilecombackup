# Slash Commands for Issue Development

The repository includes custom slash commands in `.claude/commands/` to streamline issue development workflows with comprehensive quality assurance and auto-commit functionality.

## Available Commands

### `/implement-issue FEAT-XXX or BUG-XXX`
Implement an issue following the complete development workflow:
- **Moves issue from `ready/` to `active/`** using `git mv`
- **Creates TodoWrite list** from issue tasks for progress tracking
- **Implements one task at a time** with mandatory quality verification
- **Auto-commits after each completed task** with proper issue references
- **Mandatory verification before task completion**: All commands must succeed:
  - `devbox run formatter` (format code first)
  - `devbox run tests` (all tests must pass)
  - `devbox run linter` (zero violations)
  - `devbox run build-cli` (successful build)
- **Updates documentation** when complete using product-doc-sync agent

### `/ready-issue FEAT-XXX or BUG-XXX`
Validate if an issue has enough detail for implementation:
- **Reviews issue document completeness** using spec-review-engineer agent
- **Uses `git mv` to cleanly move from `backlog/` to `ready/`** if sufficiently detailed
- **Auto-commits the file movement** with standardized message format
- **Prevents file handling issues** by using proper git mv workflow

### `/review-issue FEAT-XXX or BUG-XXX`
Review an issue specification for completeness and clarity:
- **Uses spec-review-engineer agent** for thorough specification review
- **Provides feedback and suggestions** for improvements
- **Asks clarifying questions** about requirements and implementation details
- **Auto-commits any improvements** made to the issue document during review

### `/ready-backlog-issues`
Process all issues in backlog to assess readiness:
- **Reviews each issue individually** using same criteria as `/ready-issue`
- **Smart processing order**: Bugs first, then features, alphabetically within each type
- **Dependency awareness**: Ensures prerequisite issues are ready before dependent ones
- **Moves ready issues to `ready/` directory** with auto-commit
- **Updates and commits improvements** to issues that need more work
- **Progress reporting**: Shows processing status and final summary
- **Processes one issue at a time** to ensure quality

### `/create-feature <description>`
Create a new feature issue with comprehensive planning:
- **Finds next sequential issue number** across all FEAT-XXX and BUG-XXX files
- **Creates FEAT-XXX document** from template with kebab-case naming
- **Completes comprehensive template** with requirements, design, tasks, and testing
- **Asks clarifying questions** to ensure complete specification
- **Places in `backlog/` for planning** and further development
- **Auto-commits the created feature document** with standardized message

### `/create-bug <description>`
Create a new bug report with complete investigation details:
- **Finds next sequential issue number** across all FEAT-XXX and BUG-XXX files
- **Creates BUG-XXX document** from template with detailed bug information
- **Gathers complete information** including reproduction steps, environment, severity
- **Asks for missing details** before committing (reproduction steps, environment, impact)
- **Places in `backlog/` for investigation** and resolution planning
- **Auto-commits the created bug document** with standardized message

### `/plan-and-implement-ready-issues`
Plan and implement all issues in the ready directory:
- **Reviews all ready issues** to understand scope and dependencies
- **Creates implementation plan** considering priorities and dependencies
- **Implements each issue sequentially** using separate agent instances
- **Follows complete `/implement-issue` workflow** for each issue
- **Considers dependencies** - implements prerequisites before dependent issues
- **Uses quality verification** for each task within each issue

### `/remember-anything-learned-this-session`
Capture session learnings to improve future development:
- **Reviews session history** for valuable development insights
- **Categorizes learnings** by workflow, technical, process improvements
- **Updates CLAUDE.md** with specific, actionable guidance
- **Includes examples** and commands where helpful
- **Auto-commits updates** with description of improvements added

### `/review-and-update-documentation`
Comprehensively review and synchronize all project documentation:
- **Reviews all documentation** including CLAUDE.md, README.md, specifications
- **Compares code vs docs** to identify discrepancies
- **Updates content** to accurately reflect current implementation
- **Standardizes formats** across all documentation
- **Verifies examples** work with current codebase
- **Auto-commits updates** with list of areas synchronized

## Using Slash Commands

These commands provide structured, quality-assured workflows for all issue development tasks. They ensure consistency, maintain code quality, and provide comprehensive documentation throughout the development process.

## Auto-Commit Behavior

All slash commands include sophisticated auto-commit functionality with quality verification.

### When Auto-Commit Occurs
- **After completing each TodoWrite task** in `/implement-issue` (with full verification)
- **After creating feature documents** with `/create-feature`
- **After creating bug documents** with `/create-bug`
- **After moving issues to ready** with `/ready-issue`
- **After processing each issue** with `/ready-backlog-issues` (move to ready or update in backlog)
- **After completing issue reviews** with `/review-issue` (if changes were made)
- **After implementing each issue** with `/plan-and-implement-ready-issues`
- **After capturing session learnings** with `/remember-anything-learned-this-session`
- **After updating documentation** with `/review-and-update-documentation`

### Quality Verification (Implementation Commands)

Before any implementation task is marked complete, ALL verification commands must succeed:

```bash
devbox run formatter  # Format code first
devbox run tests     # All tests must pass
devbox run linter    # Zero lint violations
devbox run build-cli # Build must succeed
```

**No exceptions**: If any verification step fails, the task remains incomplete until issues are resolved.

### File Detection Strategy

Commands use git status comparison to stage only files actually modified:

```bash
# Before starting task
git status --porcelain > /tmp/before_task

# After completing task
git status --porcelain > /tmp/after_task

# Stage only changed files
comm -13 /tmp/before_task /tmp/after_task | cut -c4- | xargs -r git add
```

### Commit Message Formats

**Implementation Tasks:**
```
[ISSUE-ID]: [Brief task description]

[Optional: Implementation details]

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

**Issue Management:**
```
[Action] [ISSUE-ID]: [Brief description]

[Details about the action taken]

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

### Important Notes

- **Never use `git add .`** - Commands stage only files they actually modified
- **Auto-commit only after successful verification** - Prevents broken code from being committed
- **One task at a time** - Complete each task fully before moving to the next
- **Quality gates enforced** - No task completion without passing all verification steps
- **Issue ID traceability** - All commits reference the relevant issue for tracking
- **Commands ask for guidance** when encountering fundamental problems or conflicts
- **File movement uses `git mv`** - Prevents split states and commit issues