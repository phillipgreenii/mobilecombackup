Use agent product-doc-sync to create a new bug report: $ARGUMENTS

**Bug Report Creation Workflow:**

1. **Determine issue number**:
   - Find the highest issue number across all FEAT-XXX and BUG-XXX files in issues/ directory
   - Use the next sequential number for the new bug (e.g., if highest is BUG-023, use BUG-024)

2. **Create bug document**:
   - Copy `issues/bug_template.md` to `issues/backlog/BUG-XXX-descriptive-name.md`
   - Use kebab-case naming convention for the descriptive name
   - Fill out the template sections based on the provided bug description

3. **Gather complete information**:
   - Ensure all critical bug report sections are completed
   - If missing information, ask for:
     - **Reproduction steps**: Clear step-by-step instructions
     - **Environment details**: Version, OS, specific conditions
     - **Expected vs actual behavior**: What should happen vs what happens
     - **Impact assessment**: How severe is the bug?
   - Don't commit until you have sufficient detail for investigation

4. **Auto-commit the new bug report**:
   - Check git status to confirm only the new bug file is staged
   - Commit with proper message format:
   ```
   Create BUG-XXX: [Brief bug description]

   Added new bug report to backlog for investigation and resolution.

   🤖 Generated with [Claude Code](https://claude.ai/code)

   Co-Authored-By: Claude <noreply@anthropic.com>
   ```

**Template Completion Requirements:**
- **Overview**: Clear description of the bug and its impact
- **Reproduction Steps**: Detailed, step-by-step instructions
- **Expected Behavior**: What should happen normally
- **Actual Behavior**: What actually happens (the bug)
- **Environment**: Version, OS, relevant configuration
- **Severity/Priority**: Assessment of bug impact and urgency
- **Root Cause Analysis**: Initial investigation section (can be "TBD")
- **Fix Approach**: Initial thoughts on resolution (can be "TBD")

**Important Notes:**
- Bug starts in `backlog/` directory for investigation
- Use `/ready-issue` command once bug is fully investigated and ready for fixing
- Prioritize high-severity bugs that affect core functionality
- Ask follow-up questions to ensure complete bug information