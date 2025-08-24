# FEAT-079: Issue Preparation Pipeline

## Overview
Create a comprehensive orchestration command `/prepare-issue` that takes an issue from backlog to ready state through multiple agent reviews, ensuring the specification is complete, technically sound, and implementation-ready.

## Problem Statement
Currently, preparing an issue for implementation requires:
- Manual review of specification completeness
- Separate validation of technical design
- Individual assessment of test strategy
- No standardized readiness criteria
- Multiple manual steps to move from backlog to ready
- Inconsistent quality of issue specifications

## Requirements

### Functional Requirements
1. **Orchestrated Multi-Agent Review**
   - Sequential agent reviews with clean handoffs
   - Each agent commits improvements before passing to next
   - Automatic progression through review pipeline
   - Clear status reporting at each stage

2. **Review Pipeline Stages**
   - **Stage 1: Specification Review** (spec-review-engineer)
     - Validate completeness of requirements
     - Check for ambiguities and gaps
     - Ensure acceptance criteria are clear
     - Commit specification improvements
   
   - **Stage 2: Technical Design Review** (new: technical-design-reviewer)
     - Validate technical approach
     - Check architectural alignment
     - Verify feasibility and constraints
     - Commit design enhancements
   
   - **Stage 3: Test Strategy Review** (new: test-strategy-reviewer)
     - Ensure comprehensive test coverage plan
     - Validate test categories and approaches
     - Add specific test scenarios
     - Commit test plan updates
   
   - **Stage 4: Implementation Planning** (new: implementation-planner)
     - Break down into concrete tasks
     - Add effort estimates
     - Identify dependencies
     - Commit task breakdown
   
   - **Stage 5: Final Readiness Check**
     - Verify all criteria met
     - Move to ready/ directory
     - Commit the transition

3. **Interruption Handling**
   - Support for "needs-human-input" status
   - Clear reporting of blocking issues
   - Ability to resume from last successful stage
   - Rollback capability if major issues found

4. **Progress Tracking**
   - Real-time status updates during pipeline
   - Summary report at completion
   - Audit trail of all changes made
   - Time tracking for each stage

### Non-Functional Requirements
1. **Atomicity**: Each stage completes fully or rolls back
2. **Idempotency**: Safe to re-run on same issue
3. **Transparency**: Clear visibility into pipeline progress
4. **Flexibility**: Configurable review criteria

## Design Approach

### Command Structure
```markdown
/prepare-issue FEAT-XXX [options]
Options:
  --skip-stage <stage>  Skip specific review stage (spec|tech|test|plan)
  --fast               Run minimal reviews (basic validation only)
  --strict             Enhanced review criteria (additional quality checks)
  --resume             Resume from last successful stage (reads from .pipeline-state)
```

### Command Implementation
The slash command will be implemented in `.claude/commands/prepare-issue.md` with the following structure:

1. **Argument Parsing**: Extract issue ID and options from `$ARGUMENTS`
2. **State Management**: Check for existing `.pipeline-state/FEAT-XXX.json` for resume capability
3. **Sequential Agent Invocation**: Each stage uses Task tool to invoke specific agent
4. **Clean Handoff Protocol**: Verify git status clean between stages (per FEAT-077)
5. **Progress Reporting**: Real-time status updates using echo statements
6. **Final Transition**: Use `git mv` to move from backlog/ to ready/ on success

### Pipeline Architecture
```
┌─────────────┐     ┌──────────┐     ┌──────────┐
│   Backlog   │────▶│ Pipeline │────▶│  Ready   │
└─────────────┘     └──────────┘     └──────────┘
                          │
                    ┌─────▼─────┐
                    │  Stage 1  │
                    │ Spec Review│
                    └─────┬─────┘
                    ┌─────▼─────┐
                    │  Stage 2  │
                    │Tech Design│
                    └─────┬─────┘
                    ┌─────▼─────┐
                    │  Stage 3  │
                    │Test Strategy│
                    └─────┬─────┘
                    ┌─────▼─────┐
                    │  Stage 4  │
                    │Implementation│
                    └─────┬─────┘
                    ┌─────▼─────┐
                    │  Stage 5  │
                    │Final Check│
                    └───────────┘
```

### New Agent Requirements
1. **technical-design-reviewer** (`.claude/agents/technical-design-reviewer.md`)
   - Reviews technical approach and architecture
   - Validates against existing patterns
   - Ensures scalability and maintainability
   - **Required Tools**: Read, Write, Edit, Grep, mcp__serena__find_symbol, mcp__serena__get_symbols_overview
   - **Model**: sonnet
   - **Extends**: base-review-agent (from FEAT-082)
   - **Key Prompts**:
     ```
     You are a technical design reviewer specializing in architecture validation.
     Review the issue specification focusing on:
     1. Technical approach soundness
     2. Architectural alignment with existing patterns
     3. Scalability and performance considerations
     4. Security and error handling completeness
     5. Integration points and dependencies
     ```

2. **test-strategy-reviewer** (`.claude/agents/test-strategy-reviewer.md`)
   - Ensures comprehensive test coverage
   - Adds specific test scenarios
   - Validates testing approach
   - **Required Tools**: Read, Write, Edit, Grep, TodoWrite
   - **Model**: sonnet
   - **Extends**: base-review-agent (from FEAT-082)
   - **Key Prompts**:
     ```
     You are a test strategy specialist ensuring comprehensive test coverage.
     Review and enhance the testing requirements by:
     1. Identifying missing test categories (unit, integration, E2E)
     2. Adding specific test scenarios and edge cases
     3. Defining test data requirements
     4. Specifying expected coverage targets
     5. Ensuring testability of acceptance criteria
     ```

3. **implementation-planner** (`.claude/agents/implementation-planner.md`)
   - Creates detailed task breakdown
   - Adds time/complexity estimates
   - Identifies dependencies
   - **Required Tools**: Read, Write, Edit, TodoWrite
   - **Model**: sonnet
   - **Extends**: base-review-agent (from FEAT-082)
   - **Key Prompts**:
     ```
     You are an implementation planning specialist who breaks down features into actionable tasks.
     Enhance the issue with:
     1. Granular task breakdown (2-4 hour chunks)
     2. Clear task dependencies and ordering
     3. Effort estimates using T-shirt sizing (S=1-2h, M=2-4h, L=4-8h, XL=8+h)
     4. Risk identification and mitigation strategies
     5. Specific file/function targets for each task
     ```

## Implementation Details

### State Management
Pipeline state will be persisted in `.pipeline-state/` directory:
```json
{
  "issue": "FEAT-079",
  "started": "2024-01-15T10:00:00Z",
  "current_stage": 3,
  "completed_stages": ["spec", "tech", "test"],
  "pending_stages": ["plan", "final"],
  "stage_durations": {
    "spec": 120,
    "tech": 180,
    "test": 150
  },
  "commits": [
    "abc123: FEAT-079: Enhanced specification clarity",
    "def456: FEAT-079: Added technical design details",
    "ghi789: FEAT-079: Expanded test scenarios"
  ],
  "options": {
    "fast": false,
    "strict": true,
    "skipped": []
  }
}
```

### Commit Message Formats
Each stage will use standardized commit messages:
- **Stage 1**: `FEAT-XXX: Enhanced specification completeness`
- **Stage 2**: `FEAT-XXX: Added technical design details`
- **Stage 3**: `FEAT-XXX: Expanded test strategy and scenarios`
- **Stage 4**: `FEAT-XXX: Added implementation task breakdown`
- **Stage 5**: `marked FEAT-XXX ready`

### Error Recovery Strategy
1. **Soft Failures**: Continue to next stage with warnings
2. **Hard Failures**: Save state and exit with clear error message
3. **Rollback**: Use `git reset --hard HEAD~N` to undo stage commits
4. **Resume**: Read state file and continue from last successful stage

### Pipeline Orchestration Logic
```bash
# Main pipeline implementation in /prepare-issue command
ISSUE_ID=$1
STATE_DIR=".pipeline-state"
STATE_FILE="${STATE_DIR}/${ISSUE_ID}.json"
LOCK_FILE="${STATE_DIR}/${ISSUE_ID}.lock"

# Ensure state directory exists
mkdir -p "${STATE_DIR}"

# Check for concurrent execution
if [[ -f "${LOCK_FILE}" ]]; then
    echo "ERROR: Pipeline already running for ${ISSUE_ID}"
    exit 1
fi
touch "${LOCK_FILE}"
trap "rm -f ${LOCK_FILE}" EXIT

# Define pipeline stages with agents
declare -A STAGES=(
    [1]="spec:spec-review-engineer:Specification Review"
    [2]="tech:technical-design-reviewer:Technical Design"
    [3]="test:test-strategy-reviewer:Test Strategy"
    [4]="plan:implementation-planner:Implementation Planning"
)

# Main pipeline execution
TOTAL_STAGES=${#STAGES[@]}
for i in $(seq 1 ${TOTAL_STAGES}); do
    IFS=':' read -r stage_key agent_name stage_desc <<< "${STAGES[$i]}"
    
    # Skip if requested
    if [[ " ${SKIP_STAGES} " =~ " ${stage_key} " ]]; then
        echo "⏭️  Skipping: ${stage_desc}"
        continue
    fi
    
    echo ""
    echo "━━━ Stage ${i}/${TOTAL_STAGES}: ${stage_desc} ━━━"
    START=$(date +%s)
    
    # Save pre-stage git state
    git status --porcelain > "/tmp/pre_${stage_key}.state"
    
    # Invoke agent using Task tool
    echo "Invoking ${agent_name} agent..."
    Use agent ${agent_name} to review and enhance issues/backlog/${ISSUE_ID}.md
    
    # Check workspace is clean (FEAT-077 protocol)
    git status --porcelain > "/tmp/post_${stage_key}.state"
    if [[ -s "/tmp/post_${stage_key}.state" ]]; then
        echo "⚠️  Workspace not clean after ${stage_desc}"
        echo "Invoking workspace-cleanup agent..."
        Use agent workspace-cleanup to ensure clean git state
    fi
    
    # Record stage completion
    DURATION=$(($(date +%s) - START))
    echo "✓ ${stage_desc} completed in ${DURATION}s"
    update_state_file "${stage_key}" "completed" "${DURATION}"
done

# Final transition
echo ""
echo "━━━ Final Stage: Moving to ready/ ━━━"
git mv "issues/backlog/${ISSUE_ID}.md" "issues/ready/${ISSUE_ID}.md"
git commit -m "marked ${ISSUE_ID} ready via preparation pipeline

Issue has passed all review stages:
- Specification completeness verified
- Technical design validated
- Test strategy comprehensive
- Implementation tasks defined"

# Cleanup
rm -f "${STATE_FILE}" "${LOCK_FILE}"
echo ""
echo "✅ Issue ${ISSUE_ID} successfully prepared and ready for implementation!"
```

## Tasks
- [ ] Create `/prepare-issue` slash command
- [ ] Implement pipeline orchestration logic
- [ ] Create technical-design-reviewer agent
- [ ] Create test-strategy-reviewer agent
- [ ] Create implementation-planner agent
- [ ] Add stage status tracking
- [ ] Implement clean handoff between agents
- [ ] Add interruption and resume support
- [ ] Create progress reporting
- [ ] Add configuration options
- [ ] Implement rollback capability
- [ ] Create comprehensive tests
- [ ] Document pipeline workflow

## Testing Requirements
1. **Unit Tests**
   - Test each pipeline stage independently
   - Verify clean handoffs between stages (mock git status)
   - Test interruption handling and state persistence
   - Validate option parsing (--skip-stage, --fast, --strict)
   - Test state file creation and updates

2. **Integration Tests**
   - Full pipeline execution with sample issues
   - Test with various issue types (FEAT, BUG)
   - Verify commit history is clean and sequential
   - Test resume from interruption at each stage
   - Validate final issue placement in ready/
   - Verify metrics collection in metrics.json

3. **Edge Cases**
   - Issue already in ready state (should exit gracefully)
   - Malformed issue documents (should fail at stage 1)
   - Git conflicts during pipeline (should detect and halt)
   - Agent failures mid-pipeline (should save state for resume)
   - Missing dependencies (FEAT-077, FEAT-078 not implemented)
   - Concurrent pipeline runs on same issue (lock file mechanism)
   - Invalid stage names in --skip-stage option

4. **Performance Tests**
   - Measure stage durations for baseline
   - Test with large issue documents (>1000 lines)
   - Verify state file doesn't grow unbounded

## Acceptance Criteria
- [ ] Single command moves issue from backlog to ready
- [ ] Each stage commits improvements independently with standardized messages
- [ ] Clean git history with descriptive commits (verified via git log)
- [ ] Clear progress reporting throughout pipeline (stage N of M format)
- [ ] Handles interruptions gracefully (state file allows resume)
- [ ] All review agents provide value-adding improvements (measurable via diff size)
- [ ] Documentation includes pipeline workflow diagram and usage examples
- [ ] Success rate tracking: Log outcomes to `.pipeline-state/metrics.json` for analysis
- [ ] State cleanup: Successful completions remove state files automatically
- [ ] Workspace verification: Each stage ends with clean git status (FEAT-077)

## Dependencies
- **FEAT-077: Agent Completion Protocol** (ensures clean handoffs) - COMPLETED
  - Provides workspace verification between stages
  - Required for clean agent handoffs
- **FEAT-078: Workspace Cleanup Agent** (for recovery scenarios) - COMPLETED
  - Can be invoked if stage leaves dirty workspace
  - Provides recovery from failed stages
- **FEAT-082: Agent Template Inheritance** - COMPLETED
  - Provides base-review-agent template for new agents
  - Enables consistent agent behavior

**Note**: All dependencies have been completed. The implementation can proceed immediately.

## Priority
HIGH - Significantly improves issue preparation workflow and ensures consistent issue quality

## Estimated Effort
- Implementation: 10-12 hours
  - Slash command creation: 3-4 hours
  - Pipeline orchestration: 4-5 hours
  - State management: 3-4 hours
- New agents: 8-10 hours
  - Technical design reviewer: 2-3 hours
  - Test strategy reviewer: 2-3 hours
  - Implementation planner: 3-4 hours
  - Testing agents: 1 hour
- Testing: 4-5 hours
  - Unit tests: 2 hours
  - Integration tests: 2-3 hours
- Documentation: 2 hours
- Total: 24-29 hours

## Success Metrics and Validation

### Measuring "90% Success Rate"
Success rate will be calculated from `.pipeline-state/metrics.json`:
```json
{
  "total_runs": 100,
  "successful": 92,
  "failed": 5,
  "blocked": 3,
  "success_rate": 0.92,
  "average_duration": 420,
  "stage_success_rates": {
    "spec": 0.98,
    "tech": 0.95,
    "test": 0.93,
    "plan": 0.96
  }
}
```

### Validation Methods
1. **Automated tracking**: Each pipeline run updates metrics
2. **Weekly reports**: Generate success rate reports
3. **Issue quality scores**: Track post-implementation defects
4. **Time-to-implementation**: Measure how quickly prepared issues are completed

## Notes
- Consider adding quality metrics for each stage (diff size, issues found)
- May want to add skip conditions for simple issues (< 50 lines)
- Could extend to support batch processing of multiple issues
- Future enhancement: ML-based review suggestions
- Pipeline state files should be added to .gitignore
- Consider adding --dry-run option to preview changes without commits
- Integration with existing `/ready-issue` command for final transition
- New agents should be created with clear specifications before implementation

## Integration with Existing Systems

### Relationship to `/ready-issue`
- `/prepare-issue` is a comprehensive pipeline that includes `/ready-issue` as final step
- `/ready-issue` remains available for simple, manual issue promotion
- `/prepare-issue` adds multi-stage review and enhancement before promotion

### Coordination with TodoWrite
- Pipeline creates master todo list for tracking overall progress
- Each agent may create sub-todos for their specific tasks
- Final stage consolidates todos into issue document

### Git Workflow Integration
- Follows project's commit message standards (see docs/GIT_WORKFLOW.md)
- Uses pre-commit hooks for validation (never --no-verify)
- Creates atomic commits for each stage

### Metrics Collection
- Stage durations logged to `.pipeline-state/metrics.json`
- Success/failure rates tracked per issue type
- Agent effectiveness measured by diff size and issues resolved