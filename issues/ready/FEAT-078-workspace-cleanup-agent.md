# FEAT-078: Workspace Cleanup Agent

## Overview
Create a dedicated agent specializing in ensuring clean git working directories and workspace cleanup. This agent can be invoked standalone or by other agents to guarantee proper cleanup after task completion.

## Problem Statement
Currently, workspace cleanup is inconsistent across agents, leading to:
- Residual temporary files after task completion
- Uncommitted changes blocking subsequent tasks
- No standardized cleanup procedure
- Difficulty recovering from partially completed tasks
- Manual intervention required to clean workspace

## Requirements

### Functional Requirements
1. **Workspace Analysis**
   - Detect all uncommitted changes (staged and unstaged)
   - Identify temporary files and directories
   - Find untracked files not in .gitignore
   - Detect merge conflicts or rebase states

2. **Intelligent Cleanup**
   - Categorize changes by type (code, docs, tests, config)
   - Run appropriate verification for change types
   - Commit valid changes with descriptive messages
   - Remove temporary files safely
   - Handle special git states (merge, rebase, cherry-pick)

3. **Recovery Capabilities**
   - Recover from interrupted agent tasks
   - Complete partial commits
   - Resolve simple conflicts automatically
   - Stash or discard invalid changes based on context

4. **Integration Support**
   - Callable by other agents as final step
   - Standalone slash command `/cleanup-workspace`
   - Pre-task workspace validation
   - Post-task cleanup verification

### Non-Functional Requirements
1. **Safety**: Never lose important work
2. **Transparency**: Log all cleanup actions
3. **Idempotency**: Safe to run multiple times
4. **Performance**: Quick workspace assessment

## Design Approach

### Agent Architecture
```yaml
name: workspace-cleanup
description: Ensures clean git working directory after any task
tools: Bash, Read, Grep, LS, TodoWrite
model: sonnet
color: blue
```

### Core Responsibilities
1. **Assessment Phase**
   - Run comprehensive git status check
   - Scan for temporary files/directories
   - Identify change categories
   - Determine cleanup strategy

2. **Cleanup Phase**
   - Run verification on valid changes
   - Commit changes with appropriate messages
   - Remove temporary artifacts
   - Handle error states

3. **Reporting Phase**
   - Report cleanup actions taken
   - List any remaining issues
   - Provide resolution suggestions
   - Update task status appropriately

### Integration Points
- Called automatically by other agents before completion
- Available as standalone command
- Integrated with TodoWrite for status updates
- Works with verification workflow

## Tasks
- [ ] Create workspace-cleanup agent definition
- [ ] Implement workspace assessment logic
- [ ] Add intelligent change categorization
- [ ] Implement safe cleanup procedures
- [ ] Add recovery from interrupted states
- [ ] Create `/cleanup-workspace` slash command
- [ ] Integrate with existing agents
- [ ] Add comprehensive logging
- [ ] Create cleanup strategy templates
- [ ] Implement safety checks and confirmations

## Testing Requirements
1. **Unit Tests**
   - Test change detection and categorization
   - Verify safe cleanup procedures
   - Test recovery mechanisms

2. **Integration Tests**
   - Test with various workspace states
   - Verify integration with other agents
   - Test conflict resolution
   - Test temporary file cleanup

3. **Edge Cases**
   - Large uncommitted changes
   - Binary files and generated code
   - Merge conflicts
   - Interrupted rebase operations
   - Permission issues

## Acceptance Criteria
- [ ] Agent correctly identifies all workspace issues
- [ ] Safe cleanup without data loss
- [ ] Clear reporting of actions taken
- [ ] Integration with existing agents
- [ ] `/cleanup-workspace` command works standalone
- [ ] Handles all common git states
- [ ] Comprehensive test coverage
- [ ] Documentation complete

## Dependencies
- FEAT-077: Agent Completion Protocol (recommended to implement first)

## Priority
HIGH - Essential for reliable automation

## Estimated Effort
- Implementation: 6-8 hours
- Testing: 3-4 hours
- Integration: 2 hours
- Documentation: 1 hour
- Total: 12-15 hours

## Notes
- Consider implementing safeguards against accidental data loss
- May need user confirmation for destructive operations
- Should maintain audit log of all cleanup actions
- Could be extended to handle git worktree cleanup