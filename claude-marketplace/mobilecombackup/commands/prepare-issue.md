Execute a comprehensive 5-stage multi-agent review pipeline to prepare a `bd` issue for implementation.

**Usage**: `/prepare-issue <bd-issue-id> [options]`

**Options:**

- `--skip-stage <stage>` - Skip specific review stage (spec, tech-design, test-strategy, implementation, final)
- `--fast` - Run minimal reviews with reduced validation criteria
- `--strict` - Enhanced review criteria with additional validation steps
- `--resume` - Resume pipeline from last successful stage (auto-detected from `bd show <id> --json`'s `metadata.prepare_pipeline`)

## Pipeline Overview

This command orchestrates a 5-stage pipeline using specialized agents to ensure a `bd` issue is thoroughly reviewed and implementation-ready. It exists for issues that need MORE than the bead-grooming skill's single-pass backlog sweep: a full architecture review, a dedicated test-strategy pass, and a detailed task breakdown, each performed by a specialized agent with a clean handoff to the next.

```
deep-review-requested → [Stage 1] → [Stage 2] → [Stage 3] → [Stage 4] → [Stage 5] → prepared
                            ↓           ↓           ↓           ↓           ↓
                       Spec Review  Tech Design  Test Strategy  Implementation  Final Check
```

**Entry condition**: the target issue carries the `deep-review-requested` label (applied by whoever wants this pipeline run, instead of the lighter bead-grooming pass). If the issue doesn't have it, add it first: `bd update <id> --add-label deep-review-requested`.

## Pipeline State

Unlike the retired file-tracker version (which used `.pipeline-state/[ISSUE-ID].json` alongside `git mv` between `issues/{backlog,ready}/`), pipeline state lives directly on the `bd` issue itself, in its `metadata` field (arbitrary JSON, preserved verbatim by `bd`):

```json
{
  "prepare_pipeline": {
    "start_time": "2026-09-17T10:30:00Z",
    "current_stage": 2,
    "completed_stages": ["spec-review"],
    "stage_results": {
      "spec-review": {
        "completed_at": "2026-09-17T10:35:00Z",
        "agent": "spec-review-engineer",
        "changes_made": 15
      }
    },
    "options": {
      "mode": "default",
      "skip_stages": []
    }
  }
}
```

Read it with `bd show <id> --json | jq '.data[0].metadata.prepare_pipeline'`. Write it with `bd update <id> --metadata '<full JSON object, all top-level metadata keys, not just prepare_pipeline>'` -- `--metadata` REPLACES the whole field, so merge with any other metadata keys already present before writing.

## Stage Execution Process

### Stage 1: Specification Review (spec-review-engineer)

1. **Load the issue**: `bd show <id>` -- read `description`, `design`, `acceptance_criteria`, `notes`.
2. **Use spec-review-engineer agent** to review completeness and clarity:
   - Validate all requirements are clearly defined
   - Check for ambiguities and missing details
   - Ensure acceptance criteria are measurable
   - Add clarifications and improvements
3. **Apply improvements** with `bd update <id> --description "..." --acceptance "..." --append-notes "Stage 1: Specification review improvements"` as needed (only the fields that actually changed).
4. **Update pipeline state** (see above) with stage 1 completion.

### Stage 2: Technical Design Review (technical-design-reviewer)

1. **Use technical-design-reviewer agent** to validate architecture and approach:
   - Review system design and component relationships
   - Validate architectural patterns and scalability
   - Assess security considerations and integration points
   - Add technical implementation details and design clarifications
2. **Apply improvements** to the issue's `design` field via `bd update <id> --design "..."`.
3. **Update pipeline state** with technical review completion.

### Stage 3: Test Strategy Review (test-strategy-reviewer)

1. **Use test-strategy-reviewer agent** to ensure comprehensive test coverage:
   - Analyze test scenarios for completeness
   - Add specific test cases and edge conditions
   - Validate testing approach and coverage targets
   - Enhance testing requirements with detailed scenarios
2. **Apply improvements** to `acceptance_criteria` or `notes` (whichever already holds testing content for this issue) via `bd update`.
3. **Update pipeline state** with test strategy completion.

### Stage 4: Implementation Planning (implementation-planner)

1. **Use implementation-planner agent** for detailed task breakdown:
   - Decompose the issue into concrete implementation tasks
   - Add effort estimates and complexity assessments (t-shirt sizes, per this repo's estimate conventions -- never calendar-time estimates)
   - Identify dependencies and implementation order
   - Create a detailed task breakdown as `acceptance_criteria` checkboxes
2. **Apply improvements** via `bd update <id> --acceptance "..."`.
3. **Update pipeline state** with implementation planning completion.

### Stage 5: Final Readiness Validation

1. **Validate all pipeline stages completed successfully** (all 5 present in `metadata.prepare_pipeline.completed_stages`, unless explicitly skipped via `--skip-stage`).
2. **Verify the issue contains all required sections**: non-empty `description`, `design`, and `acceptance_criteria`.
3. **Mark the issue prepared**: `bd update <id> --remove-label deep-review-requested --add-label has-acceptance-criteria --actor "<session-id>"` (matching this project's existing `has-acceptance-criteria` labeling convention).
4. **Clean up pipeline state** (optional): leave `metadata.prepare_pipeline` in place as a completion record, or clear it -- either is fine since the label transition is the authoritative "done" signal.

## Implementation Guidelines

### Error Handling and Recovery

- **Interruption Handling**: Write pipeline state to the issue's `metadata` after each stage completion, not just at the end.
- **Resume Capability**: `--resume` reads `metadata.prepare_pipeline.current_stage`/`completed_stages` and continues from there.
- **Stage Failures**: Report specific failure reasons and resolution steps; do not advance `current_stage` on failure.
- **Rollback Support**: Any actual repo file edits an agent makes during a stage (e.g. updating a design doc referenced by the issue) still go through the normal git workflow and are recoverable via git history; the issue's own field history is NOT versioned by `bd` beyond its `notes` append log, so stage-by-stage `--append-notes` entries are the audit trail for what changed and when.

### Agent Integration

- **Clean Handoffs**: Each agent's changes land via `bd update` before the next stage starts.
- **Quality Verification**: If a stage's agent edits actual repository files (not just the `bd` issue), the normal verification workflow (formatter/tests/linter/build) and commit standards apply before that stage is marked complete.
- **Error Reporting**: Agents report BLOCKED status with resolution steps.

### Progress Reporting

Provide real-time progress updates, e.g.:

```
🚀 Starting issue preparation pipeline for tc-5lxy.31
📋 Issue: <issue title>

Stage 1/5: Specification Review
  ✓ Loaded tc-5lxy.31 from bd
  ⏳ Running spec-review-engineer...
  ✓ Specification enhanced (12 improvements)
  ✓ Applied via bd update

Stage 2/5: Technical Design Review
  ⏳ Running technical-design-reviewer...
```

### Options Implementation

- **--skip-stage**: Skip specified stages (with warnings about reduced quality); record the skip in `metadata.prepare_pipeline.options.skip_stages`.
- **--fast**: Use expedited review with minimal validation criteria.
- **--strict**: Enhanced validation with additional quality checks.
- **--resume**: Auto-detect state from `metadata.prepare_pipeline` and resume from the last completed stage.

## Quality Assurance

### Verification Requirements

- All agents must complete successfully before stage advancement.
- Pipeline state in `metadata` must stay consistent and recoverable across stages.
- The final issue must pass the Stage 5 readiness validation before the `deep-review-requested` label is removed.

### Safety Mechanisms

- State persistence in `bd` metadata for interruption recovery (survives across sessions, unlike the old `.pipeline-state/*.json` which lived only in the worktree).
- Validation checkpoints between stages.

This command ports the multi-agent issue preparation pipeline originally designed in FEAT-079 (see `bd show` on the imported historical record once tc-5lxy.24 lands) from the retired git-based `issues/` tracker onto `bd`, per tc-5lxy.22.
