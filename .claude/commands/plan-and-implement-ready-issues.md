Use agent spec-implementation-engineer to plan and implement all issues in the ready/ directory.

**Overall Workflow:**
1. **Review all ready issues** to understand scope and dependencies
2. **Create implementation plan** considering issue priorities and dependencies  
3. **Implement each issue sequentially** using separate agent instances
4. **Follow complete implementation workflow** for each issue

**Implementation Strategy:**
- **Process issues one at a time** - complete each fully before starting next
- **Consider dependencies** - implement prerequisite issues before dependent ones
- **Use separate agent instance** for each issue implementation
- **Follow standard quality workflow** for each task within each issue

**Per-Issue Implementation Workflow:**
Each issue should follow the complete `/implement-issue` workflow:

1. **Move to active**: `git mv issues/ready/[ISSUE].md issues/active/[ISSUE].md`
2. **Create TodoWrite list** from issue tasks
3. **Implement one task at a time** with full verification:
   - `devbox run formatter` (format code first)
   - `devbox run tests` (all tests must pass)
   - `devbox run linter` (zero violations) 
   - `devbox run build-cli` (successful build)
4. **Auto-commit after each completed task**
5. **Use agent product-doc-sync** when issue complete to:
   - Update issue document with implementation details
   - Update `issues/specification.md` to match implementation  
   - Move completed issue from `active/` to `completed/`

**Quality Requirements:**
- **Code must compile** and all tests pass before marking any task complete
- **One task at a time**: Complete each TodoWrite task fully before moving to next
- **Commit only modified files** - never use `git add .`
- **Reference issue ID** in all commit messages for traceability

**Planning Considerations:**
- **Dependencies**: Review issue references and related features sections
- **Priority**: High-priority bugs before medium-priority features
- **Complexity**: Consider breaking large features into smaller chunks if needed
- **Testing**: Ensure comprehensive test coverage for all implementations

**Important Notes:**
- Ask for guidance if there are conflicts between specification and implementation
- If implementation reveals issues with specification, update documentation accordingly
- Ensure each issue results in working, tested, documented code
- Follow project conventions and coding standards throughout