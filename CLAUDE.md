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

## Documentation Rules

### Living Documentation
- **issues/specification.md** is a **living representation** of the project
- **MUST be updated** whenever documentation changes to match current system state
- Serves as the single source of truth for current architecture and capabilities

### Completed Issues Policy
- **Completed issues** (in `issues/completed/`) should **NOT be updated**
- They serve as historical records of what was implemented
- **Allowed exceptions only**:
  - Adding cross-references to newer issues that modified the functionality
  - Minor text improvements (typos, readability) that don't change interpretation
  
### Documentation Update Workflow
1. **Always review** `issues/specification.md` when updating docs
2. **Verify** code state matches documentation
3. **Update** specification.md if system has evolved
4. **Preserve** completed issues as historical records
5. **Cross-reference** when newer issues supersede older ones

## Task Completion Requirements

For detailed task completion requirements and verification workflow, see [Task Completion](docs/TASK_COMPLETION.md).

## Git Workflow and Commit Rules

For complete git workflow and commit rules, see [Git Workflow](docs/GIT_WORKFLOW.md).

**CRITICAL**: Every task MUST end with a commit that passes ALL quality checks. NEVER use `git commit --no-verify`.

### Pre-commit Hook Optimization (FEAT-072)
The pre-commit hook is optimized for documentation-focused workflows:
- **Markdown-only commits**: Skip tests, run formatter + linter only (~6s, target <10s)
- **Code/mixed commits**: Run full checks (formatter + tests + linter, target <30s)
- **Automatic detection**: Uses `git diff --cached --name-only` to analyze staged files
- **Clear feedback**: Shows optimization decisions and performance metrics

## Development Tools

### Code Analysis (Preferred - Semantic Tools)
- **Serena MCP**: Advanced semantic code analysis and symbol manipulation
  - `mcp__serena__find_symbol` - Semantic symbol search (prefer over grep for code)
  - `mcp__serena__search_for_pattern` - Advanced pattern matching with code awareness
  - `mcp__serena__get_symbols_overview` - Understand file structure before editing
  - `mcp__serena__find_referencing_symbols` - Find symbol usage across codebase
  - `mcp__serena__replace_symbol_body` - Precise code modifications
  - `mcp__serena__insert_after_symbol` / `mcp__serena__insert_before_symbol` - Structured code insertion

### Code Analysis (Fallback - Structural Tools)
- **ast-grep**: Structural code search and refactoring (when Serena MCP insufficient)
- **fd**: Fast file finding
- **ripgrep**: Fast text search (use only for non-code content)

#### Tool Selection Guidelines
**For code analysis tasks, prefer this hierarchy:**
1. **Serena MCP tools** - For symbol finding, code structure analysis, precise modifications
2. **ast-grep** - For structural patterns when Serena MCP insufficient  
3. **ripgrep/grep** - Only for non-code text search or when semantic tools fail

#### Common ast-grep Patterns
```bash
# Find function definitions
ast-grep --pattern 'func $NAME($$$) $RET { $$$ }'

# Find error handling patterns
ast-grep --pattern 'if err != nil { $$$ }'

# Find test functions
ast-grep --pattern 'func Test$_($$$) { $$$ }'
```

#### Serena MCP Workflow Examples
```bash
# Recommended workflow for code analysis:
# 1. Get file overview before editing
mcp__serena__get_symbols_overview

# 2. Find specific symbols semantically
mcp__serena__find_symbol --name_path "functionName"

# 3. Find usage/references
mcp__serena__find_referencing_symbols

# 4. Make precise modifications
mcp__serena__replace_symbol_body
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

### XML Security
- Always use `security.NewSecureXMLDecoder` for XML parsing
- Direct `xml.NewDecoder` usage is prohibited (XXE vulnerability)

### File Organization
- `types.go`: Structs and interfaces
- `reader.go`: Main implementation
- `*_test.go`: Unit and integration tests
- `example_test.go`: Usage examples

### Git Workflow
See [Git Workflow](docs/GIT_WORKFLOW.md) for complete commit standards and staging practices.

## Agent Tool Preferences

### Required Agent Permissions
The following agents require full access to all Serena MCP tools:

**Agents Requiring Serena MCP Access:**
- `code-completion-verifier`
- `spec-implementation-engineer` 
- `spec-review-engineer`

**Required Serena MCP Tools:**
- `mcp__serena__get_symbols_overview`
- `mcp__serena__find_symbol`
- `mcp__serena__find_referencing_symbols`
- `mcp__serena__search_for_pattern`
- `mcp__serena__replace_symbol_body`
- `mcp__serena__insert_after_symbol`
- `mcp__serena__insert_before_symbol`
- `mcp__serena__list_dir`
- `mcp__serena__find_file`
- `mcp__serena__write_memory`
- `mcp__serena__read_memory`
- `mcp__serena__list_memories`
- `mcp__serena__delete_memory`
- `mcp__serena__check_onboarding_performed`
- `mcp__serena__onboarding`
- `mcp__serena__think_about_collected_information`
- `mcp__serena__think_about_task_adherence`
- `mcp__serena__think_about_whether_you_are_done`

### Code Analysis Workflow
When working with Go code, agents should:

1. **Start with Serena MCP** for all code analysis:
   - `mcp__serena__get_symbols_overview` to understand file structure
   - `mcp__serena__find_symbol` to locate specific functions/types
   - `mcp__serena__find_referencing_symbols` to understand usage

2. **Use Serena MCP for modifications**:
   - `mcp__serena__replace_symbol_body` for function/method changes
   - `mcp__serena__insert_after_symbol` for adding new code
   - Ensure changes are semantically correct within code structure

3. **Fallback to basic tools only when**:
   - Serena MCP tools fail or are insufficient
   - Working with non-code files (documentation, configs)
   - Simple text-based operations

### Tool Selection Examples
```bash
# ✅ PREFERRED: Semantic analysis for Go code
mcp__serena__find_symbol --name_path "ProcessCalls"

# ❌ AVOID: Text search for code symbols
grep "func ProcessCalls"

# ✅ PREFERRED: Understanding code structure  
mcp__serena__get_symbols_overview --relative_path "pkg/calls"

# ❌ AVOID: Basic file reading for code analysis
cat pkg/calls/reader.go

# ✅ PREFERRED: Finding symbol references
mcp__serena__find_referencing_symbols --name_path "Call" 

# ❌ AVOID: Text-based reference search
grep -r "Call" .
```

## Additional Documentation

### Core Workflow Documentation
- **Verification Workflow**: `docs/VERIFICATION_WORKFLOW.md` - Quality verification commands
- **Git Workflow**: `docs/GIT_WORKFLOW.md` - Commit rules and standards
- **Task Completion**: `docs/TASK_COMPLETION.md` - Task completion requirements
- **Common Fixes**: `docs/COMMON_FIXES.md` - Fix patterns for common issues
- **Issue Workflow**: `docs/ISSUE_WORKFLOW.md` - Complete development lifecycle

### Project Documentation
- **Architecture**: `docs/ARCHITECTURE.md` - System architecture and design decisions
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
- Use full import paths: `github.com/phillipgreenii/mobilecombackup/pkg/...`
- Timestamps are milliseconds (divide by 1000 for Unix time)
- XML "null" values should be treated as empty
- Create temp files in `tmp/` directory (not `/tmp`)
- Test data has intentional quirks (count mismatches, mixed years)