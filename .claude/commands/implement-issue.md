Use agent spec-implementation-engineer to implement issue $ARGUMENTS following the established development workflow.

**Workflow Overview:**
1. **Move issue to active** using `git mv issues/ready/[ISSUE].md issues/active/[ISSUE].md`
2. **Create TodoWrite list** from the issue's task section  
3. **Implement one task at a time** ensuring quality at each step
4. **Auto-commit after each completed task**
5. **Update documentation** when implementation is complete

**Implementation Requirements:**
- **One task at a time**: Complete each TodoWrite task fully before moving to next
- **Verification before completion**: ALL commands must succeed before marking task complete:
  ```bash
  devbox run formatter  # Code must be formatted
  devbox run tests     # All tests must pass  
  devbox run linter    # Zero lint violations
  devbox run build-cli # Build must succeed
  ```
- **Code quality**: Ensure code compiles and all tests pass before marking any task complete
- **Task completion**: Only mark TodoWrite tasks complete when ALL verification steps pass

**Auto-Commit Workflow (After Each Task):**
1. **Check git status** before and after task work to identify modified files
2. **Run verification** - ensure all quality checks pass:
   - `devbox run formatter` (format code first)
   - `devbox run tests` (all tests must pass)  
   - `devbox run linter` (zero violations)
   - `devbox run build-cli` (successful build)
3. **Stage only modified files** - never use `git add .`, only stage files you changed
4. **Commit with proper format**:
   ```
   [ISSUE-ID]: [Brief task description]

   [Optional: Implementation details]

   🤖 Generated with [Claude Code](https://claude.ai/code)

   Co-Authored-By: Claude <noreply@anthropic.com>
   ```

**Issue Completion:**
When all tasks are complete, use agent product-doc-sync to:
- Update the issue document with final implementation details
- Update `issues/specification.md` to match the implementation
- Move completed issue from `active/` to `completed/`
- Commit the final documentation updates

**Important Notes:**
- If verification fails, fix issues before marking task complete
- Never skip quality checks - they prevent production issues
- Reference issue ID in all commit messages for traceability
- Ask for guidance if implementation conflicts with specification