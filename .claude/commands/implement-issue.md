Use agent spec-implementation-engineer to implement issue $ARGUMENTS following the established development workflow.

**Workflow Overview:**
1. **Move issue to active** using `git mv issues/ready/[ISSUE].md issues/active/[ISSUE].md`
2. **Create TodoWrite list** from the issue's task section  
3. **Implement one task at a time** ensuring quality at each step
4. **Auto-commit after each completed task**
5. **Update documentation** when implementation is complete

**Implementation Requirements:**
- **One task at a time**: Complete each TodoWrite task fully before moving to next
- **Verification before completion**: Follow [Verification Workflow](docs/VERIFICATION_WORKFLOW.md) and [Task Completion](docs/TASK_COMPLETION.md) requirements
- **Code quality**: All verification commands must pass before marking any task complete

**Auto-Commit Workflow (After Each Task):**
Follow [Git Workflow](docs/GIT_WORKFLOW.md) for commit standards and [Verification Workflow](docs/VERIFICATION_WORKFLOW.md) requirements.

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