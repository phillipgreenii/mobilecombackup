Use agent product-doc-sync to comprehensively review and update all project documentation to ensure alignment with current codebase state.

**Documentation Review Scope:**
Review and update all project documentation including:

**Core Documentation:**
- **CLAUDE.md**: Project instructions and development guidelines
- **README.md**: User-facing project description and usage
- **issues/specification.md**: Technical specification document
- **docs/**: All documentation files in docs directory

**Issue Documentation:**
- **Completed issues**: Ensure issue documents reflect actual implementation
- **Active specifications**: Verify accuracy of ready/active issue details
- **Templates**: Update feature and bug templates with current standards

**Technical Documentation:**
- **API documentation**: Align with current code interfaces  
- **Architecture documentation**: Reflect current system design
- **Development guides**: Update with current workflow and standards
- **Configuration documentation**: Match current settings and options

**Documentation Update Workflow:**
1. **Compare code vs docs** - identify discrepancies between current implementation and documentation
2. **Update content** - modify documentation to accurately reflect current state
3. **Standardize formats** - ensure consistent formatting and structure across all docs
4. **Verify examples** - ensure all code examples work with current codebase
5. **Auto-commit updates** with proper message:
   ```
   Update documentation to align with current codebase

   Synchronized documentation with implementation including:
   - [List specific areas updated]

   🤖 Generated with [Claude Code](https://claude.ai/code)

   Co-Authored-By: Claude <noreply@anthropic.com>
   ```

**Quality Standards:**
- **Accuracy**: All documentation must reflect actual current behavior
- **Completeness**: Cover all significant features and workflows  
- **Clarity**: Clear, unambiguous language for intended audience
- **Examples**: Working, tested examples where applicable
- **Structure**: Consistent formatting and organization

**Important Notes:**
- Focus on high-impact documentation that users and developers rely on
- Don't create new documentation unless specifically needed
- Update existing content rather than duplicating information
- Ensure Claude-specific instructions remain current and effective