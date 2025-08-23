# FEAT-077: Agent Completion Protocol

## Overview
Implement a mandatory completion protocol for all agents to ensure they leave the git working directory clean after completing tasks. Currently, agents often report tasks as "complete" while leaving uncommitted changes, temporary files, or other artifacts that require manual cleanup.

## Problem Statement
Agents are not consistently cleaning up after themselves, leading to:
- Uncommitted changes left in the working directory
- Temporary files and directories not removed
- Confusion about whether tasks are truly complete
- Manual intervention required to clean up after agents
- Difficulty chaining agent tasks due to dirty workspace

## Requirements

### Functional Requirements
1. **Mandatory Completion Checklist**
   - All agents MUST verify git working directory is clean before reporting completion
   - Check for and remove temporary files created during task execution
   - Commit all valid changes with appropriate messages
   - Report as BLOCKED (not complete) if unable to achieve clean state

2. **Clear Definition of "Done"**
   - An agent task is ONLY complete when:
     - Git working directory is CLEAN (no modified/untracked files)
     - All temporary files/directories removed
     - All changes committed successfully
     - OR explicitly blocked with clear explanation of what prevents completion

3. **Completion Verification Commands**
   - Run `git status` to check for uncommitted changes
   - Check `tmp/` directory for temporary files
   - Verify no untracked files remain (except those in .gitignore)

4. **Blocking and Escalation**
   - If unable to achieve clean state, agent MUST:
     - Report specific reason for blockage
     - List uncommitted files and their status
     - Suggest resolution steps
     - NEVER claim task is complete with dirty workspace

### Non-Functional Requirements
1. **Consistency**: Apply protocol uniformly across all agents
2. **Clarity**: Clear status reporting (COMPLETE vs BLOCKED)
3. **Auditability**: Log completion verification steps
4. **Fail-safe**: Default to BLOCKED if uncertain about state

## Design Approach

### Implementation Strategy
1. Create shared completion verification module
2. Update all existing agents to use the protocol
3. Add completion verification to agent templates
4. Implement pre-completion checklist execution

### Affected Agents
- `spec-implementation-engineer`
- `product-doc-sync`
- `code-completion-verifier`
- `spec-review-engineer`
- All future agents created from templates

### Completion Protocol Flow
```
1. Task work completed
2. Run completion checklist:
   a. Check git status
   b. Scan for temporary files
   c. Run verification if changes exist
   d. Commit if verification passes
3. Verify clean state:
   - If clean → Report COMPLETE
   - If dirty → Report BLOCKED with details
```

## Tasks
- [ ] Define shared completion protocol module
- [ ] Update `spec-implementation-engineer` agent with completion protocol
- [ ] Update `product-doc-sync` agent with completion protocol
- [ ] Update `code-completion-verifier` agent with completion protocol
- [ ] Update `spec-review-engineer` agent with completion protocol
- [ ] Update agent templates with completion protocol
- [ ] Add completion verification logging
- [ ] Create tests for completion protocol
- [ ] Update documentation with new completion requirements
- [ ] Add completion status to TodoWrite integration

## Testing Requirements
1. **Unit Tests**
   - Test completion protocol with various workspace states
   - Verify correct COMPLETE/BLOCKED status reporting
   - Test temporary file detection and cleanup

2. **Integration Tests**
   - Test agent completion with uncommitted changes
   - Test agent completion with temporary files
   - Test agent blocking when unable to clean workspace
   - Test multi-agent workflows with clean handoffs

3. **Acceptance Tests**
   - Run agent on real task and verify clean completion
   - Test blocking behavior with intentional conflicts
   - Verify clear status reporting to users

## Acceptance Criteria
- [ ] All agents implement completion protocol
- [ ] No agent reports "complete" with dirty workspace
- [ ] Temporary files are always cleaned up
- [ ] Clear BLOCKED status with actionable details when cleanup fails
- [ ] Documentation updated with completion requirements
- [ ] 100% test coverage for completion protocol
- [ ] Successful multi-agent workflow with clean handoffs

## Dependencies
- None - this is a foundational improvement

## Priority
HIGH - This is critical for reliable agent operation and user experience

## Estimated Effort
- Implementation: 4-6 hours
- Testing: 2-3 hours
- Documentation: 1 hour
- Total: 7-10 hours

## Notes
- This is the foundation for reliable multi-agent workflows
- Essential for user trust in agent automation
- Reduces manual intervention and improves developer experience