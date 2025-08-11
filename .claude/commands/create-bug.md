Use agent product-doc-sync.  Using the defined issue workflow, create a new bug report: $ARGUMENTS. 

1. Find the highest issue number across all FEAT-XXX and BUG-XXX files in the issues/ directory
2. Use the next sequential number for the new bug
3. Copy issues/bug_template.md to issues/backlog/BUG-XXX-descriptive-name.md
4. Fill out the template based on the bug description
5. If the user hasn't provided all the necessary information (reproduction steps, environment, etc.), ask for those details
6. **Auto-commit the created bug document**: After creating the new bug document, commit it with a message like "Create BUG-XXX: [brief description]"
