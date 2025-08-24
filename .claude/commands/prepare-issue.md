Execute a comprehensive 5-stage multi-agent review pipeline to prepare an issue for implementation.

**Usage**: `/prepare-issue FEAT-XXX or BUG-XXX [options]`

**Options:**
- `--skip-stage <stage>` - Skip specific review stage (spec, tech-design, test-strategy, implementation, final)
- `--fast` - Run minimal reviews with reduced validation criteria
- `--strict` - Enhanced review criteria with additional validation steps  
- `--resume` - Resume pipeline from last successful stage (auto-detected from state)

## Pipeline Overview

This command orchestrates a 5-stage pipeline using specialized agents to ensure issues are thoroughly reviewed and implementation-ready:

```
Backlog → [Stage 1] → [Stage 2] → [Stage 3] → [Stage 4] → [Stage 5] → Ready
              ↓           ↓           ↓           ↓           ↓
         Spec Review  Tech Design  Test Strategy  Implementation  Final Check
```

## Stage Execution Process

### Stage 1: Specification Review (spec-review-engineer)
1. **Load issue from backlog/** directory
2. **Use spec-review-engineer agent** to review completeness and clarity:
   - Validate all requirements are clearly defined
   - Check for ambiguities and missing details
   - Ensure acceptance criteria are measurable
   - Add clarifications and improvements to the specification
3. **Commit improvements** with message format:
   ```
   [ISSUE-ID] Stage 1: Specification review improvements

   - Enhanced requirements clarity
   - Added missing acceptance criteria  
   - Resolved specification ambiguities

   🤖 Generated with [Claude Code](https://claude.ai/code)
   Co-Authored-By: Claude <noreply@anthropic.com>
   ```
4. **Update pipeline state** (`.pipeline-state/[ISSUE-ID].json`) with stage completion

### Stage 2: Technical Design Review (technical-design-reviewer)  
1. **Use technical-design-reviewer agent** to validate architecture and approach:
   - Review system design and component relationships
   - Validate architectural patterns and scalability
   - Assess security considerations and integration points
   - Add technical implementation details and design clarifications
2. **Commit improvements** with standardized stage 2 message format
3. **Update pipeline state** with technical review completion

### Stage 3: Test Strategy Review (test-strategy-reviewer)
1. **Use test-strategy-reviewer agent** to ensure comprehensive test coverage:
   - Analyze test scenarios for completeness
   - Add specific test cases and edge conditions
   - Validate testing approach and coverage targets
   - Enhance testing requirements with detailed scenarios
2. **Commit improvements** with standardized stage 3 message format
3. **Update pipeline state** with test strategy completion

### Stage 4: Implementation Planning (implementation-planner)
1. **Use implementation-planner agent** for detailed task breakdown:
   - Decompose feature into concrete implementation tasks
   - Add effort estimates and complexity assessments
   - Identify dependencies and implementation order
   - Create detailed task breakdown with acceptance criteria
2. **Commit improvements** with standardized stage 4 message format
3. **Update pipeline state** with implementation planning completion

### Stage 5: Final Readiness Validation
1. **Validate all pipeline stages completed successfully**
2. **Verify issue contains all required sections and details**
3. **Move issue to ready/** directory using `git mv`
4. **Commit final transition** with message:
   ```
   [ISSUE-ID] Ready: Completed 5-stage preparation pipeline

   Pipeline completed successfully:
   ✓ Stage 1: Specification review  
   ✓ Stage 2: Technical design review
   ✓ Stage 3: Test strategy review
   ✓ Stage 4: Implementation planning
   ✓ Stage 5: Final validation

   Issue is now implementation-ready.

   🤖 Generated with [Claude Code](https://claude.ai/code)
   Co-Authored-By: Claude <noreply@anthropic.com>
   ```
5. **Clean up pipeline state** after successful completion

## Implementation Guidelines

### Pipeline State Management
Create and maintain pipeline state in `.pipeline-state/[ISSUE-ID].json`:
```json
{
  "issue_id": "FEAT-084",
  "start_time": "2025-01-15T10:30:00Z",
  "current_stage": 2,
  "completed_stages": ["spec-review"],
  "stage_results": {
    "spec-review": {
      "completed_at": "2025-01-15T10:35:00Z",
      "agent": "spec-review-engineer",
      "changes_made": 15,
      "commit_hash": "abc123"
    }
  },
  "options": {
    "mode": "default",
    "skip_stages": []
  }
}
```

### Error Handling and Recovery
- **Interruption Handling**: Save state after each stage completion
- **Resume Capability**: Use `--resume` flag to continue from last successful stage
- **Stage Failures**: Report specific failure reasons and resolution steps
- **Rollback Support**: Maintain git history for safe rollback if needed

### Agent Integration
- **Clean Handoffs**: Each agent commits improvements before next stage
- **Completion Protocol**: Use FEAT-077 completion protocol for each stage
- **Quality Verification**: Run verification steps before stage commits
- **Error Reporting**: Agents report BLOCKED status with resolution steps

### Progress Reporting
Provide real-time progress updates:
```
🚀 Starting issue preparation pipeline for FEAT-084
📋 Issue: Automated Documentation Synchronization System

Stage 1/5: Specification Review
  ✓ Loading issue from backlog/
  ⏳ Running spec-review-engineer...
  ✓ Specification enhanced (12 improvements)
  ✓ Committed stage 1 changes
  
Stage 2/5: Technical Design Review
  ⏳ Running technical-design-reviewer...
```

### Options Implementation
- **--skip-stage**: Skip specified stages (with warnings about reduced quality)
- **--fast**: Use expedited review with minimal validation criteria
- **--strict**: Enhanced validation with additional quality checks  
- **--resume**: Auto-detect state and resume from last successful stage

## Quality Assurance

### Verification Requirements
- All agents must complete successfully before stage advancement
- Git working directory must be clean after each stage commit
- Pipeline state must be consistent and recoverable
- Final issue must pass readiness validation

### Safety Mechanisms
- Atomic stage operations (complete success or clean rollback)
- State persistence for interruption recovery
- Git history preservation for manual recovery
- Validation checkpoints between stages

This command implements the comprehensive issue preparation pipeline designed in FEAT-079, providing automated quality assurance and thorough review for all implementation-bound issues.