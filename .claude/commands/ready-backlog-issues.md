Process all issues in the backlog directory to assess readiness for implementation. Use agent spec-review-engineer to review each issue individually, following the same workflow as the `/ready-issue` command.

**Workflow for each issue:**

1. **Review the issue** using spec-review-engineer to determine if it has enough detail for implementation
2. **If the issue is ready for implementation:**
   - Use `git mv issues/backlog/[ISSUE].md issues/ready/[ISSUE].md` to move it
   - Commit with message format: `marked [ISSUE-ID] ready`
3. **If the issue needs more work:**
   - Update the issue document with any gathered information, suggestions, or missing details
   - Commit the improvements with message format: `Update [ISSUE-ID] with review feedback`
   - Leave the issue in backlog/ for further planning

**Processing Requirements:**
- Process **one issue at a time** in sequence
- Complete all work on each issue (review, update, move/commit) before moving to the next
- Use the same readiness criteria and review standards as `/ready-issue`
- Provide a summary at the end showing which issues were moved to ready/ and which need more work

**Important Notes:**
- Follow the same git mv and commit workflow as `/ready-issue` to prevent file handling issues
- Each issue should result in exactly one commit (either move to ready, or update in backlog)
- Do not skip issues - review every file in the backlog/ directory
- If an issue has fundamental problems that make it unimplementable, note this in the review and ask for guidance