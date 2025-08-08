# Issue Documentation

This directory contains planning and documentation for all issues (features and bugs) in the mobilecombackup project.

## Structure

- `feature_template.md` - Template for new feature documents
- `bug_template.md` - Template for bug reports
- `next_steps.md` - Place to track temporary notes related to active issues or plan of which issues to work on next
- `specification.md` - Provides the full picture of the entire application
- `active/` - Issues currently being implemented/fixed
- `ready/` - Issues which are ready to be implemented/fixed
- `backlog/` - Issues which are still in the planning/investigation phase; they aren't yet ready to start
- `completed/` - Resolved issues (reference documentation)

## Workflow

### For Features
1. **Create Feature**: Copy `feature_template.md` to `backlog/FEAT-XXX-name.md`
2. **Plan**: Fill out requirements and design sections; once ready, move to `ready/`
3. **Implement**: When starting to implement a feature, move it to `active/`
4. **Complete**: Move to `completed/` and update with final implementation details; update `specification.md` to align with changes

### For Bugs
1. **Create Bug**: Copy `bug_template.md` to `backlog/BUG-XXX-name.md`
2. **Investigate**: Reproduce issue, identify root cause; once ready, move to `ready/`
3. **Fix**: When starting to fix a bug, move it to `active/`
4. **Complete**: Move to `completed/` and update with verification steps

## Issue Numbering

All issues (features and bugs) share a single sequential numbering system. The number is permanent and helps with tracking dependencies and references.

Examples:
- FEAT-001
- FEAT-002
- BUG-003
- FEAT-004
- BUG-005

When creating a new issue, always use the next number after the highest existing FEAT or BUG number across all directories.