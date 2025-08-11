Use agent spec-implementation-engineer to start implementing $ARGUMENTS. Follow the development workflow on how to mark it as active. When working on the implementation, prioritize working on one task at a time. The code should be in a state where it can compile and all tests pass before marking a task as complete.

**IMPORTANT**: After completing each task, you MUST auto-commit your changes:
- Format code first: Run `devbox run formatter` to ensure consistent formatting
- Use git status comparison to identify files you modified during the task
- Stage only the files you actually changed (never use `git add .`)
- Commit with a descriptive message referencing the issue ID and task description
- Follow the commit message format specified in the agent instructions

When the issue is complete, be sure to update both the details of the issue and `issues/specification.md` to ensure they match the implementation. If there is a conflict, ask me.
