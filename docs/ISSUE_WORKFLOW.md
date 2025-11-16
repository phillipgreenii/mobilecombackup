# Issue Development Workflow

This document defines the complete lifecycle for developing features and fixes using the project's issue-based workflow.

## Overview

The issue workflow provides structure for planning, implementing, and completing development tasks. It integrates TodoWrite task management with quality verification and documentation updates.

Issues move through four states: **backlog** → **ready** → **active** → **completed**.

## Issue Lifecycle

### Quick Reference
1. **Create issue**: Use `/create-feature` or `/create-bug` → `backlog/`
2. **Plan issue**: Fill details in `issues/backlog/FEAT-XXX.md`
3. **Ready issue**: Move to `issues/ready/` when fully planned (use `/ready-issue`)
4. **Implement**: Use `/implement-issue FEAT-XXX` → moves to `active/`
5. **Complete**: Auto-moves to `issues/completed/` when finished

### Issue States and Directory Structure

```
issues/
├── backlog/     # Being planned, incomplete specifications
├── ready/       # Fully planned, ready for implementation
├── active/      # Currently being implemented (work in progress)
└── completed/   # Finished, tested, documented, and committed
```

**State Transitions:**
- `backlog/` → `ready/` : When specification is complete (manual or `/ready-issue`)
- `ready/` → `active/` : When implementation starts (`/implement-issue`)
- `active/` → `completed/` : When all tasks done, tested, and committed (automatic)

## Issue Creation

### Creating New Issues

**Feature Issues:**
```bash
/create-feature "Feature description"
```

**Bug Issues:**
```bash
/create-bug "Bug description"
```

### Initial Issue Location
- New issues start in `issues/backlog/`
- Format: `FEAT-XXX.md` or `BUG-XXX.md`
- Contains specification template to fill out

## Issue Planning Phase

### Planning Requirements

Before moving to `ready/`, ensure the issue contains:

- **Clear problem statement** or feature description
- **Functional requirements** with specific acceptance criteria
- **Non-functional requirements** (performance, security, scalability)
- **Technical constraints** and dependencies
- **API contracts** or interface definitions where applicable
- **Data models** and schemas if needed
- **Error handling** and edge cases
- **Testing requirements** and strategies
- **Implementation tasks** broken into specific steps

### Moving to Ready

**Manual Process:**
```bash
git mv issues/backlog/FEAT-XXX.md issues/ready/FEAT-XXX.md
git commit -m "FEAT-XXX: Move to ready for implementation"
```

**Command Process:**
```bash
/ready-issue FEAT-XXX
```

## Implementation Phase

### Starting Implementation

**Recommended Command:**
```bash
/implement-issue FEAT-XXX
```

This command will:
1. Move issue from `ready/` to `active/`
2. Create TodoWrite list from issue tasks
3. Begin implementation with agent guidance

### Implementation Requirements

During implementation:

- **One task at a time**: Complete each TodoWrite task fully before moving to next
- **Verification before completion**: Follow [Task Completion](TASK_COMPLETION.md) requirements
- **Quality standards**: All code must pass [Verification Workflow](VERIFICATION_WORKFLOW.md)
- **Commit after each task**: Follow [Git Workflow](GIT_WORKFLOW.md) for commits

### Task Breakdown

Create TodoWrite list from issue specification:
1. **Extract tasks** from issue's task section
2. **Make tasks specific** and actionable
3. **Work sequentially** - one task at a time
4. **Verify completion** - all checks must pass
5. **Commit progress** - after each completed task

### Example TodoWrite List
```
1. Create core data structures for feature X
2. Implement parsing logic with error handling  
3. Add comprehensive test suite with edge cases
4. Update documentation and examples
5. Run full verification and fix any issues
```

## Task Completion Requirements

For each TodoWrite task:

1. **Complete the work** according to task specification
2. **Run verification workflow**:
   ```bash
   devbox run formatter  # Format code first
   devbox run tests     # All tests must pass
   devbox run linter    # Zero lint violations
   devbox run build-cli # Build must succeed
   ```
3. **Fix any issues** found by verification
4. **Commit changes** using proper format
5. **Mark task complete** only after successful commit

See [Task Completion](TASK_COMPLETION.md) for detailed requirements.

## Issue Completion

### When All Tasks Are Complete

Use the `product-doc-sync` agent to:

1. **Update issue document** with final implementation details
2. **Update specifications** to match actual implementation  
3. **Move completed issue** from `active/` to `completed/`
4. **Commit documentation updates**

### Completion Checklist

- ✅ All TodoWrite tasks completed and committed
- ✅ All verification checks pass
- ✅ Issue document updated with implementation notes
- ✅ Specifications reflect actual implementation
- ✅ Issue moved to `completed/` directory
- ✅ Final documentation commit made

## Directory Movement Commands

### Manual Git Commands
```bash
# Backlog to Ready
git mv issues/backlog/FEAT-XXX.md issues/ready/FEAT-XXX.md

# Ready to Active  
git mv issues/ready/FEAT-XXX.md issues/active/FEAT-XXX.md

# Active to Completed
git mv issues/active/FEAT-XXX.md issues/completed/FEAT-XXX.md
```

### Slash Commands
```bash
/ready-issue FEAT-XXX          # Move backlog -> ready
/implement-issue FEAT-XXX       # Move ready -> active + implement
# Note: Completion typically handled by product-doc-sync agent
```

## Integration with Development Tools

### TodoWrite Integration

- **Task Creation**: Extract tasks from issue specifications
- **Status Tracking**: Use pending/in_progress/completed states
- **One Task Rule**: Only one task in_progress at a time
- **Completion Verification**: All checks must pass before marking complete

### Agent Integration

**spec-implementation-engineer**:
- Primary agent for implementing issues
- Follows task completion requirements
- Integrates with verification workflow

**product-doc-sync**:
- Updates documentation after implementation
- Handles issue completion and movement
- Ensures specifications match implementation

**code-completion-verifier**:
- Ensures all quality checks pass
- Provides auto-fixes for common issues
- Enforces task completion requirements

## Commit Message Integration

Reference issue ID in all related commits:

```
FEAT-055: Implement core parsing logic

Added streaming XML parser with error resilience
as specified in FEAT-055 requirements.

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

## Implementation Best Practices

### Planning Phase
- **Be specific** in requirements and acceptance criteria
- **Include examples** of expected inputs/outputs
- **Consider edge cases** and error scenarios
- **Define clear testing strategy**
- **Break into manageable tasks**

### Implementation Phase
- **Follow task sequence** - don't skip ahead
- **Verify each step** - don't accumulate issues
- **Commit frequently** - after each completed task
- **Fix issues immediately** - don't defer quality problems
- **Ask for help** when blocked

### Completion Phase
- **Update documentation** to match implementation
- **Review against original requirements**
- **Ensure all acceptance criteria met**
- **Clean up any temporary code or files**

## Quality Assurance

### Throughout Implementation

All implementation must follow:
- [Verification Workflow](VERIFICATION_WORKFLOW.md) - mandatory quality checks
- [Git Workflow](GIT_WORKFLOW.md) - commit standards and pre-commit hooks
- [Task Completion](TASK_COMPLETION.md) - completion verification requirements
- [Common Fixes](COMMON_FIXES.md) - fix patterns for issues

### No Shortcuts

- **Never skip verification** steps
- **Never use `git commit --no-verify`**
- **Never mark tasks complete** with failing checks
- **Never accumulate technical debt**

## Troubleshooting

### Common Issues

**Issue stuck in backlog**:
- Complete the specification requirements
- Ensure all sections are filled out
- Add specific acceptance criteria

**Implementation blocked**:
- Check [Common Fixes](COMMON_FIXES.md) for solutions
- Ask for help rather than compromising quality
- Break large tasks into smaller pieces

**Verification failures**:
- Follow [Verification Workflow](VERIFICATION_WORKFLOW.md)
- Use [Common Fixes](COMMON_FIXES.md) patterns
- Don't proceed until all checks pass

### Getting Help

When blocked:
1. **Document the specific issue** you're facing
2. **Include error messages** and commands run
3. **Explain what you've tried** to fix it
4. **Ask for specific guidance** on next steps

## Related Documentation

- [Task Completion](TASK_COMPLETION.md) - Requirements for task completion
- [Verification Workflow](VERIFICATION_WORKFLOW.md) - Quality checks
- [Git Workflow](GIT_WORKFLOW.md) - Commit and branching standards
- [Common Fixes](COMMON_FIXES.md) - Fix patterns for common issues

## Issue Types

### Feature Issues (FEAT-XXX)
- New functionality or capabilities
- Enhancement to existing features
- API additions or modifications

### Bug Issues (BUG-XXX)
- Incorrect behavior fixes
- Performance improvements
- Security vulnerability fixes

### Documentation Issues (DOC-XXX)
- Documentation updates
- README improvements
- API documentation changes

Each type follows the same workflow but may have different planning requirements and acceptance criteria.