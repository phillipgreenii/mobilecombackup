# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go command-line tool for processing mobile phone backup files (Call and SMS logs in XML format). It coalesces multiple backup files, removes duplicates, extracts attachments, and organizes data by year.

## Development Commands

For verification workflow and quality commands, see [Verification Workflow](docs/VERIFICATION_WORKFLOW.md).

```bash
# Development shortcuts
devbox shell         # Enter development environment
devbox run builder   # Build all packages

# Git hooks (quality enforcement)
devbox run install-hooks  # Install pre-commit hooks
devbox run test-hooks     # Test hooks without committing
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

For complete issue development workflow, see [Issue Workflow](docs/ISSUE_WORKFLOW.md).

### Quick Reference
1. **Create issue**: Use `/create-feature` or `/create-bug` commands
2. **Plan issue**: Fill details in `issues/backlog/FEAT-XXX.md`
3. **Ready issue**: Move to `issues/ready/` when planned
4. **Implement**: Use `/implement-issue FEAT-XXX` command
5. **Complete**: Updates move to `issues/completed/`

## Task Completion Requirements

For detailed task completion requirements and verification workflow, see [Task Completion](docs/TASK_COMPLETION.md).

## Git Workflow and Commit Rules

For complete git workflow and commit rules, see [Git Workflow](docs/GIT_WORKFLOW.md).

**CRITICAL**: Every task MUST end with a commit that passes ALL quality checks. NEVER use `git commit --no-verify`.

## Development Tools

### Code Analysis
- **ast-grep**: Structural code search and refactoring (available in devbox)
- **fd**: Fast file finding
- **ripgrep**: Fast text search (via claude-code)

#### Common ast-grep Patterns
```bash
# Find function definitions
ast-grep --pattern 'func $NAME($$$) $RET { $$$ }'

# Find error handling patterns
ast-grep --pattern 'if err != nil { $$$ }'

# Find test functions
ast-grep --pattern 'func Test$_($$$) { $$$ }'
```

## Common Patterns

### Testing
- Target 80%+ coverage
- Use `testdata/` for test files
- Test both success and failure paths
- Create `example_test.go` for usage docs

#### Test Commands
- `devbox run test-unit`: Fast unit tests only (uses `gotestsum` for better output)
- `devbox run test-integration`: Integration tests only (CLI and file I/O tests)
- `devbox run test`: Full test suite (both unit and integration tests with enhanced output)

#### Test Development Workflow
1. **During development**: Use `devbox run test-unit` for rapid feedback
2. **Before committing**: Run `devbox run test` to ensure all tests pass
3. **Integration tests**: Use `testing.Short()` to skip in unit-only runs
4. **Unit tests**: Add `t.Parallel()` to pure logic tests for performance

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
See [Git Workflow](docs/GIT_WORKFLOW.md) for complete commit standards and staging practices.

## Additional Documentation

### Core Workflow Documentation
- **Verification Workflow**: `docs/VERIFICATION_WORKFLOW.md` - Quality verification commands
- **Git Workflow**: `docs/GIT_WORKFLOW.md` - Commit rules and standards
- **Task Completion**: `docs/TASK_COMPLETION.md` - Task completion requirements
- **Common Fixes**: `docs/COMMON_FIXES.md` - Fix patterns for common issues
- **Issue Workflow**: `docs/ISSUE_WORKFLOW.md` - Complete development lifecycle

### Project Documentation
- **Troubleshooting**: `docs/TROUBLESHOOTING.md` - Test/lint failures and fixes
- **Version Management**: `docs/VERSION_MANAGEMENT.md` - Release workflow
- **Session Learnings**: `docs/SESSION_LEARNINGS.md` - Implementation insights
- **Slash Commands**: `docs/SLASH_COMMANDS.md` - Available CLI commands
- **Specification**: `issues/specification.md` - Detailed technical specs
- **Next Steps**: `issues/next_steps.md` - Current priorities

## Important Reminders

- ALWAYS format before testing or committing
- NEVER skip verification steps
- NEVER use `git commit --no-verify`
- Task completion REQUIRES successful commit with ALL checks passing
- Use full import paths: `github.com/phillipgreen/mobilecombackup/pkg/...`
- Timestamps are milliseconds (divide by 1000 for Unix time)
- XML "null" values should be treated as empty
- Create temp files in `tmp/` directory (not `/tmp`)
- Test data has intentional quirks (count mismatches, mixed years)