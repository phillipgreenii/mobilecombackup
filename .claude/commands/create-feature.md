Use agent product-doc-sync to create a new feature issue: $ARGUMENTS

**Feature Creation Workflow:**

1. **Determine issue number**:
   - Find the highest issue number across all FEAT-XXX and BUG-XXX files in issues/ directory
   - Use the next sequential number for the new feature (e.g., if highest is FEAT-039, use FEAT-040)

2. **Create feature document**:
   - Copy `issues/feature_template.md` to `issues/backlog/FEAT-XXX-descriptive-name.md`
   - Use kebab-case naming convention for the descriptive name
   - Fill out the template sections based on the provided description

3. **Complete specification**:
   - Fill in all template sections with appropriate detail
   - Ask clarifying questions as needed to properly complete the specification
   - Ensure the feature has clear requirements and acceptance criteria

4. **Auto-commit the new feature**:
   - Check git status to confirm only the new feature file is staged
   - Commit with proper message format:
   ```
   Create FEAT-XXX: [Brief feature description]

   Added new feature specification to backlog for planning and implementation.

   🤖 Generated with [Claude Code](https://claude.ai/code)

   Co-Authored-By: Claude <noreply@anthropic.com>
   ```

**Template Completion Guidelines:**
- **Overview**: Clear, concise description of what the feature does
- **Requirements**: Specific functional and non-functional requirements  
- **Design**: High-level approach and technical considerations
- **Tasks**: Actionable implementation tasks
- **Testing**: Comprehensive testing strategy
- **Priority**: Assign appropriate priority level

**Important Notes:**
- Feature starts in `backlog/` directory for further planning
- Use `/ready-issue` command once feature is fully planned
- Ask questions to ensure complete specification before committing
- Follow project conventions for naming and structure