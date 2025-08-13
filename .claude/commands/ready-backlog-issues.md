Process all issues in the backlog directory to assess readiness for implementation. Use agent spec-review-engineer to review each issue individually, following the same workflow as the `/ready-issue` command.

**Processing Order:**
1. **Bugs first, then features** (BUG-XXX before FEAT-XXX)
2. **Alphabetical within each type** (BUG-001, BUG-002, then FEAT-001, FEAT-002)
3. **Dependency awareness**: If an issue depends on another issue, the dependency must be ready first
   - Example: If FEAT-100 depends on FEAT-099, process FEAT-099 first
   - If FEAT-099 is not ready but FEAT-100 is, leave FEAT-100 in backlog until FEAT-099 is ready
   - Check issue references, related features, and dependencies sections

**Workflow for each issue:**

1. **Check dependencies** - Review if this issue depends on other backlog issues that aren't ready yet
2. **Review the issue** using spec-review-engineer to determine if it has enough detail for implementation
3. **If the issue is ready for implementation AND has no unmet dependencies:**
   - Use `git mv issues/backlog/[ISSUE].md issues/ready/[ISSUE].md` to move it
   - Commit with message format: `marked [ISSUE-ID] ready`
4. **If the issue needs more work OR has unmet dependencies:**
   - Update the issue document with any gathered information, suggestions, or missing details
   - If blocked by dependencies, note this in the issue
   - Commit the improvements with message format: `Update [ISSUE-ID] with review feedback`
   - Leave the issue in backlog/ for further planning

**Progress Reporting:**
- Provide simple progress updates: "Processing [X] of [Y]: [ISSUE-ID]..."
- Show summary at end: "Ready: [list], Updated: [list], Dependencies Blocked: [list]"

**Important Notes:**
- Ask questions whenever guidance is needed (fundamental problems, unclear dependencies, etc.)
- Follow the same git mv and commit workflow as `/ready-issue` to prevent file handling issues
- Each issue should result in exactly one commit (either move to ready, or update in backlog)
- Do not skip issues - review every file in the backlog/ directory