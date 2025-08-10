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
- [ ] Update all Claude commands to auto-commit unless user input is needed
- [ ] Update agent configurations to enable auto-commit
- [ ] Ensure meaningful commit messages are generated
- [ ] Provide option to disable auto-commit when needed
- [ ] Handle cases where user permission is required

### Non-Functional Requirements
- [ ] Commits should follow project commit message conventions
- [ ] Auto-commit should not interfere with user's git workflow
- [ ] Changes should be properly staged before committing

## Design
### Approach
1. Review all Claude commands for commit behavior
2. Update commands to include auto-commit logic
3. Configure agents with appropriate permissions
4. Implement smart commit message generation
5. Add flags/options to control auto-commit behavior

### Implementation Notes
- Commit messages should reference the issue being worked on (FEAT-XXX, BUG-XXX)
- Use git's staging area properly (avoid `git add .`)
- Consider interactive rebase for cleaning up commits if needed
- Respect existing commit message format with Co-Authored-By

### Manual Configuration Steps
Since agents may not have automatic git commit permissions, users will need to configure their environment:

1. **For Claude Desktop/Browser**:
   - Agents cannot directly commit due to security restrictions
   - Provide clear instructions in command output for manual commit steps
   - Generate complete commit commands that users can copy/paste

2. **For CLI environments**:
   - Ensure git user.name and user.email are configured
   - Verify SSH keys or credentials are properly set up
   - Test with: `git config user.name` and `git config user.email`

3. **Alternative Approach**:
   - Commands should generate a commit script that users can review and execute
   - Example: `./commit-changes.sh` with proper staging and commit message
   - Include safety checks in the script (no uncommitted changes, etc.)

## Tasks
- [ ] Audit all Claude commands in .claude/commands/
- [ ] Update commands to include auto-commit logic
- [ ] Review agent configuration requirements
- [ ] Update agent configurations if permissions are needed
- [ ] Implement commit message generation logic
- [ ] Add documentation about auto-commit behavior
- [ ] Test auto-commit in various scenarios

## Testing
### Unit Tests
- Verify commit message generation
- Test git operations

### Integration Tests
- Test auto-commit with various commands
- Verify proper file staging
- Test error handling when commits fail

### Edge Cases
- Handle uncommittable states (merge conflicts)
- Handle empty commits
- Handle cases where working directory is dirty

## Risks and Mitigations
- **Risk**: Auto-commits might include unintended changes
  - **Mitigation**: Carefully stage only intended files, never use `git add .`
- **Risk**: Users might lose control over commit history
  - **Mitigation**: Provide clear option to disable auto-commit
- **Risk**: Commits might have poor messages
  - **Mitigation**: Generate meaningful messages based on context

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