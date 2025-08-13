# Slash Commands for Issue Development

The repository includes custom slash commands in `.claude/commands/` to streamline issue development workflows.

## Available Commands

### `/implement-issue FEAT-XXX or BUG-XXX`
Start implementing an issue following the established workflow:
- Moves issue from `ready/` to `active/`
- Creates TodoWrite list from issue tasks
- Ensures code compilation and test passing before task completion
- **Auto-commits after each completed task** using git status comparison
- Commits only modified files with proper issue references
- Updates both issue document and `specification.md` on completion

### `/ready-issue FEAT-XXX or BUG-XXX`
Validate if an issue has enough detail for implementation:
- Reviews issue document completeness
- Moves from `backlog/` to `ready/` if sufficiently detailed

### `/review-issue FEAT-XXX or BUG-XXX`
Review an issue specification:
- Provides feedback and suggestions for improvements
- Asks clarifying questions about requirements
- **Auto-commits any improvements made to the issue document**

### `/create-feature <description>`
Create a new feature issue:
- Finds next sequential issue number
- Creates FEAT-XXX document from template
- Places in `backlog/` for planning
- **Auto-commits the created feature document**

### `/create-bug <description>`
Create a new bug report:
- Finds next sequential issue number
- Creates BUG-XXX document from template
- Places in `backlog/` for investigation
- **Auto-commits the created bug document**

### `/remember-anything-learned-this-session`
Update CLAUDE.md with session learnings:
- Captures development workflow improvements
- Documents new patterns and best practices

## Using Slash Commands

These commands are invoked by the user and provide structured prompts for common issue development tasks. They help ensure consistency across issue implementations and reduce the cognitive load of remembering all workflow steps.

## Auto-Commit Behavior

All Claude commands and agents are configured to automatically commit code changes when they complete tasks.

### When Auto-Commit Occurs
- After completing each TodoWrite task in `/implement-issue`
- After creating feature documents with `/create-feature`
- After creating bug documents with `/create-bug`
- After completing issue reviews with `/review-issue` (if changes were made)
- After completing documentation updates

### File Detection Strategy
Agents use git status comparison to identify only files they modified:
```bash
# Before starting task
git status --porcelain > /tmp/before_task

# After completing task
git status --porcelain > /tmp/after_task

# Stage only changed files
comm -13 /tmp/before_task /tmp/after_task | cut -c4- | xargs -r git add
```

### Commit Message Format
```
[ISSUE-ID]: [Brief task description]

[Optional: Details about implementation]

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

### Important Notes
- Agents never use `git add .` - they stage only files they actually modified
- Auto-commit only occurs after successful verification (code formatted, tests pass, code compiles, linting clean)
- If commit fails unexpectedly, agents will stop and ask for user guidance
- Commands that only read files without making changes do not auto-commit