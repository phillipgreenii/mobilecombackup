# FEAT-029: Auto-commit Code Changes in Claude Commands and Agents

## Status
- **Completed**: Not started
- **Priority**: medium

## Overview
Configure all Claude commands and agents to automatically commit code changes when they complete a task, unless there's a specific need for user response. This will streamline the development workflow and reduce manual steps.

## Background
Currently, Claude commands and agents may complete tasks but leave uncommitted changes, requiring manual intervention to commit. This adds unnecessary steps to the workflow and can lead to confusion about what changes have been made. Automating commits will improve efficiency and maintain a clear history of changes.

## Requirements
### Functional Requirements
- [ ] Update all Claude commands to auto-commit after each task completion
- [ ] Track modified files during execution to stage only changed files
- [ ] Generate meaningful commit messages based on task context
- [ ] Enable agents to commit directly without user review
- [ ] Provide configuration option to disable auto-commit when needed

### Non-Functional Requirements
- [ ] Commits should follow project commit message conventions
- [ ] Auto-commit should not interfere with user's git workflow
- [ ] Only stage files that were actually modified during task execution
- [ ] Generate commit messages with FEAT-XXX/BUG-XXX references and task descriptions

## Design
### Approach
1. Review all Claude commands for commit behavior
2. Implement file modification tracking during task execution
3. Update commands to auto-commit after each completed task
4. Generate commit messages using task context and issue references
5. Add configuration options to control auto-commit behavior

### File Staging Strategy
Simple git status comparison to determine changed files:
```bash
# Before starting task
git status --porcelain > /tmp/before_task

# After completing task
git status --porcelain > /tmp/after_task

# Stage only the files that changed
diff /tmp/before_task /tmp/after_task | grep '^>' | cut -c4- | xargs git add
```

- Use git status to detect changes before and after task execution
- Stage only files that actually changed during the task
- Never use `git add .` or stage unrelated changes

### Commit Message Templates
**Task Completion Format:**
```
[ISSUE-ID]: [Task description from issue]

[Optional: Brief details about what was implemented]

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

**Examples:**
- `FEAT-029: Implement file modification tracking for auto-commit`
- `BUG-025: Fix timestamp parsing error in XML reader`
- `FEAT-030: Add validation for phone number format in contacts`

### Implementation Details
**Agent Behavior:**
- Agents commit when they finish their work
- Issue ID and task description already in agent prompt - no context passing needed
- Use simple git status comparison to find changed files
- If commit fails and agent doesn't know why, stop and ask user

**No complex configuration needed** - this is about changing agent behavior, not technical capabilities.

### Technical Approach
This is primarily a **behavioral change**, not a technical implementation:

1. **Agents can already commit** - the issue is getting them to do it consistently
2. **Simple file detection** - git status before/after comparison
3. **Context already available** - issue ID and task description are in the agent prompt
4. **Error handling** - if commit fails and agent unsure why, stop and ask user

## Tasks
- [ ] Update agent prompts to commit after completing tasks
- [ ] Implement simple git status comparison for file detection
- [ ] Update implement-issue command to instruct agents to commit
- [ ] Update create-feature and create-bug commands for auto-commit
- [ ] Test that agents consistently commit when they should
- [ ] Update documentation in CLAUDE.md about expected auto-commit behavior

### Command Inventory
**Commands that should auto-commit:**
- `/implement-issue` - After each task completion
- `/create-feature` - After creating feature document
- `/create-bug` - After creating bug document
- `/ready-issue` - After moving issue to ready (if modifications made)

**Commands that should NOT auto-commit:**
- `/review-issue` - Read-only operation
- `/review-and-update-documentation` - May need user review

## Testing
### Unit Tests
- Verify commit message generation
- Test git operations

### Integration Tests
- Test that agents commit after completing tasks in implement-issue
- Test that agents commit after create-feature and create-bug
- Verify only files changed during task are staged
- Test that agents stop and ask user when commits fail

### Edge Cases
- Handle cases where no files were actually modified
- Handle cases where working directory has unrelated changes
- Handle uncommittable states (merge conflicts) - agent should ask user
- Handle tasks that only read files (no commit needed)

## Risks and Mitigations
- **Risk**: Auto-commits might include unintended changes
  - **Mitigation**: Use git status comparison to stage only files changed during task
- **Risk**: Commits might have poor messages
  - **Mitigation**: Agents use issue ID and task description from their prompt
- **Risk**: Commit failures might block progress
  - **Mitigation**: Agent stops and asks user when commit fails unexpectedly

## Dependencies
- Works best with: FEAT-028 (commits should happen after successful tests/linting)
  - Auto-commit is more valuable when we know the code passes all checks

## References
- Claude commands: .claude/commands/
- Agent configurations: .claude/agents/
- Commit conventions: CLAUDE.md Git Workflow section
- Related: FEAT-028 (run tests and linter during development)

## Notes
If agents need specific permissions for git operations, the implementation should clearly document what configuration changes are required. Consider creating a dedicated command for testing auto-commit behavior.