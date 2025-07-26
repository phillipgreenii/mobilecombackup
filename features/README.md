# Feature Documentation

This directory contains planning and documentation for features in the mobilecombackup project.

## Structure

- `template.md` - Template for new feature documents
- `next_steps.md` - Place to track temporary notes related to the active features or plan of which features to work on next.
- `specification.md` - Provides the full picture of the entire application.
- `active/` - Features currently being implemented.
- `ready/` - Features which are ready to be implemented.
- `backlog/` - Features which are still in the planning phase; they aren't yet ready to start.
- `completed/` - Implemented features (reference documentation)

## Workflow

1. **Create Feature**: Copy `template.md` to `backlog/FEAT-XXX-name.md`
2. **Plan**: Fill out requirements and design sections; once ready, move to `ready/`
3. **Implement**: When starting to implement a task, put it into `active/`.
4. **Complete**: Move to `completed/` and update with final implementation details; update `specification.md` to align with changes.

## Feature Numbering

Features are numbered sequentially (FEAT-001, FEAT-002, etc.). The number is permanent and helps with tracking dependencies and references.
