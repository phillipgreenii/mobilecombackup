# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go command-line tool for processing mobile phone backup files (Call and SMS logs in XML format). It coalesces multiple backup files, removes duplicates, extracts attachments, and organizes data by year.

## Development Commands

```bash
# Quality workflow (ALWAYS run in this order)
devbox run formatter  # Format code first
devbox run tests     # Run all tests
devbox run linter    # Check code quality
devbox run build-cli # Build the CLI

# Development shortcuts
devbox shell         # Enter development environment
devbox run builder   # Build all packages

# Git hooks (quality enforcement)
devbox run install-hooks  # Install pre-commit hooks
devbox run test-hooks     # Test hooks without committing
git commit --no-verify   # Bypass hooks for emergencies
```

## Architecture Overview

### Core Packages
- **cmd/mobilecombackup**: CLI entry point with Cobra commands
- **pkg/calls**: Call log processing with streaming XML reader
- **pkg/sms**: SMS/MMS processing (handles complex MMS parts)
- **pkg/contacts**: Contact management with YAML support
- **pkg/attachments**: Hash-based attachment storage
- **pkg/manifest**: File manifest generation
- **pkg/importer**: Import orchestration with validation
- **pkg/coalescer**: Deduplication logic

### Key Design Principles
- **Streaming APIs**: Process large files without loading into memory
- **Error resilience**: Continue on individual failures, collect errors
- **Hash-based storage**: SHA-256 for content addressing
- **Interface-first**: Define APIs before implementation
- **UTC-based**: All timestamps and year partitioning use UTC

### Repository Structure
```
repository/
├── .mobilecombackup.yaml  # Repository marker
├── calls/                 # Call records by year
├── sms/                   # SMS/MMS records by year
├── attachments/           # Hash-based attachment storage
└── contacts.yaml          # Contact information
```

## Issue Development Workflow

### Quick Reference
1. **Create issue**: Use `/create-feature` or `/create-bug` commands
2. **Plan issue**: Fill details in `issues/backlog/FEAT-XXX.md`
3. **Ready issue**: Move to `issues/ready/` when planned
4. **Implement**: Use `/implement-issue FEAT-XXX` command
5. **Complete**: Updates move to `issues/completed/`

### Issue Structure
```
issues/
├── active/      # Currently being implemented
├── ready/       # Fully planned, ready to implement
├── backlog/     # Being planned
└── completed/   # Finished issues
```

### Implementation Best Practices
- Create TodoWrite list from issue tasks
- Work on one task at a time
- Run verification before marking tasks complete
- Commit after each completed task
- Reference issue in commit messages (e.g., "FEAT-XXX: Add validation")

## Task Completion Verification

**MANDATORY**: Before marking any task complete, ALL commands must succeed:
```bash
devbox run formatter  # Code must be formatted
devbox run tests     # All tests must pass
devbox run linter    # Zero lint violations
devbox run build-cli # Build must succeed
```

If ANY command fails:
1. Fix the issues
2. Re-run verification
3. Only mark complete when ALL pass

## Common Patterns

### Testing
- Target 80%+ coverage
- Use `testdata/` for test files
- Test both success and failure paths
- Create `example_test.go` for usage docs

### Error Handling
- Return errors, don't `os.Exit()` in libraries
- Include context in error messages
- Continue processing on individual failures
- Collect and report all errors at end

### File Organization
- `types.go`: Structs and interfaces
- `reader.go`: Main implementation
- `*_test.go`: Unit and integration tests
- `example_test.go`: Usage examples

### Git Workflow
```bash
# Stage specific files only (NEVER use git add .)
git add pkg/specific/file.go

# Commit with issue reference
git commit -m "FEAT-XXX: Brief description

Detailed explanation if needed

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

## Additional Documentation

For detailed information, see:
- **Troubleshooting**: `docs/TROUBLESHOOTING.md` - Test/lint failures and fixes
- **Version Management**: `docs/VERSION_MANAGEMENT.md` - Release workflow
- **Session Learnings**: `docs/SESSION_LEARNINGS.md` - Implementation insights
- **Slash Commands**: `docs/SLASH_COMMANDS.md` - Available CLI commands
- **Specification**: `issues/specification.md` - Detailed technical specs
- **Next Steps**: `issues/next_steps.md` - Current priorities

## Important Reminders

- ALWAYS format before testing or committing
- NEVER skip verification steps
- Use full import paths: `github.com/phillipgreen/mobilecombackup/pkg/...`
- Timestamps are milliseconds (divide by 1000 for Unix time)
- XML "null" values should be treated as empty
- Create temp files in `tmp/` directory (not `/tmp`)
- Test data has intentional quirks (count mismatches, mixed years)