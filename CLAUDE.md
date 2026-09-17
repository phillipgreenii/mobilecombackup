# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go command-line tool for processing mobile phone backup files (Call and SMS logs in XML format). It coalesces multiple backup files, removes duplicates, extracts attachments, and organizes data by year.

## Quick Commands Reference

### Most Common Development Commands

```bash
# Environment
flox activate              # Enter development environment (or rely on direnv auto-activation)
just ci                    # Run full CI pipeline (format + test + lint + build)

# Testing & Quality
just tests                 # Run all tests
just formatter             # Format code (nix fmt / treefmt+gofumpt)
just linter                # Run linter

# Building
just build-cli              # Build CLI with version info
```

### Issue Workflow Commands

Issue tracking is `bd` (beads), not the retired `issues/` markdown tracker (tc-5lxy.22).

```bash
# Create issues
bd create -t feature "description"   # Create new feature issue
bd create -t bug "description"       # Create new bug issue

# Groom / prepare issues
# bead-grooming skill              # Single-pass backlog quality sweep
/prepare-issue <bd-id>              # Deeper 5-stage review pipeline (spec/tech-design/test-strategy/impl-plan/final)

# Implement issues
# /drain-beads                     # Autonomous claim -> implement -> validate -> land -> close loop
```

### File Locations Quick Reference

- **Source code**: `pkg/` (Go packages) and `cmd/mobilecombackup/` (CLI)
- **Documentation**: `docs/` (specialized docs) and `README.md` (overview)
- **Issues**: tracked in `bd` (beads); `issues/completed/` holds the pre-beads historical archive only (read-only, do not edit)
- **Scripts**: `scripts/`

## Environment Context

### Development Environment

**All development and agent work assumes you are in the activated Flox environment.**

```bash
# Enter the development environment
flox activate
# (or just cd into the repo -- .envrc runs `use flox`, so direnv
# auto-activates after a one-time `direnv allow`)

# You'll see this message when environment loads:
📋 Setting up development environment...
```

**Key Facts:**

- Flox provides a reproducible development environment via Nix
- All tools are automatically available once the environment is activated
- Commands like `just tests` work from any directory within the repo
- Environment is isolated - doesn't affect your global system
- Activating the environment automatically runs `go mod tidy` (via the manifest's `[hook]`) and sets up the project-specific Neovim config; git hooks are installed separately via nix (see [Git Workflow](docs/GIT_WORKFLOW.md#git-hooks))

### Tools Provided by Flox

These tools are defined in `.flox/env/manifest.toml` and automatically available once the environment is activated:

**Go Development:**

- `go` - Go compiler and toolchain (version pinned by `go.version` in `.flox/env/manifest.toml`, currently 1.26.5)
- `gopls@latest` - Go language server for editor integration
- `golangci-lint@latest` - Comprehensive Go linter
- `gotestsum@latest` - Enhanced test output formatter

**Code Analysis:**

- `ast-grep@latest` - Structural code search and refactoring for Go
- `fd@latest` - Fast file finder (alternative to find)
- `ripgrep@latest` - Fast text search (alternative to grep)

**Data Processing:**

- `jq@latest` - JSON query and manipulation tool
- `yq@latest` - YAML query and manipulation tool

**Utilities:**

- `viu@latest` - Terminal image viewer
- `deno@2` - JavaScript/TypeScript runtime
- `uv@latest` - Fast Python package installer
- `claude-code` - Claude Code CLI (unpinned/floats to latest -- see `.flox/env/manifest.toml`'s `[install]` comment for why)

**Available via MCP (Claude Code extension):**

- Serena MCP tools - Semantic code analysis
- All `mcp__serena__*` functions for Go code manipulation

### just Commands

All `just` recipes are defined in the repo-root `justfile` (translated from the previous
manifest's `shell.scripts`, tc-5lxy.8):

**Core Development:**

```bash
just formatter        # Run nix fmt (treefmt/gofumpt -- stricter than plain go fmt)
just builder           # Build all packages
just tests             # Run all tests with gotestsum
just test-unit         # Run unit tests only (skip integration)
just test-integration  # Run integration tests only
just linter            # Run golangci-lint
just linter-fix        # Run linter with auto-fix
just build-cli         # Build CLI with version info
just ci                # Full CI pipeline: format, test, lint, build
```

**Quality & Validation:**

```bash
just validate-docs     # Validate documentation health
just update-doc-health # Update dashboard metrics
just coverage          # Generate HTML coverage report
just coverage-summary  # Show coverage summary
```

**Development Workflow:**

Pre-commit hooks are managed by nix (tc-5lxy.5) — see
[Git Workflow](docs/GIT_WORKFLOW.md#git-hooks): `nix run .#install-pre-commit-hooks`.

```bash
just validate-version  # Validate version strings
just ccusage            # Monitor Claude Code usage
```

(The old `list-issues` script has no replacement — the `issues/` markdown tracker was retired
in favor of `bd`; see [Issue Development Workflow](#issue-development-workflow) below.)

### Tool Availability Rules

**✅ Available in the activated Flox environment:**

- All 13 tools listed above with specified versions
- All `just` recipes
- Git operations (system git)
- Standard shell commands (bash, etc.)

**❌ NOT available outside the activated Flox environment:**

- `ast-grep`, `fd`, `ripgrep`, `gopls`, `golangci-lint`, `gotestsum`
- The pinned Go version (see `go.version` in `.flox/env/manifest.toml`)
- `jq`, `yq`, `viu`, `deno`, `uv`, `claude-code`
- Project-specific `just` recipes that call flox-provided tools directly

**⚠️ May vary if used outside the Flox environment:**

- `go` - System version likely different from the version pinned in `.flox/env/manifest.toml`
- `jq`, `yq` - May be installed globally but different versions

### Common Environment Issues

**Issue: `command not found: flox`**

- **Cause**: Flox not installed on system
- **Solution**: Install Flox (see https://flox.dev/docs/install-flox/) or use manual setup (see [Development Guide](docs/DEVELOPMENT.md#manual-setup))

**Issue: `command not found: ast-grep` (or other Flox-provided tool)**

- **Cause**: Not in the activated Flox environment
- **Solution**: Run `flox activate` first (or let direnv auto-activate)

**Issue: Wrong Go version (system Go instead of the version pinned in `.flox/env/manifest.toml`)**

- **Cause**: Using system Go instead of the Flox-provided Go
- **Solution**: Ensure you're in the activated Flox environment, verify with `go version`

**Issue: `just tests` doesn't work**

- **Cause**: Not in project directory or subdirectory
- **Solution**: `cd` to project root where `justfile` exists

**Issue: Changes to `.flox/env/manifest.toml` not taking effect**

- **Cause**: Need to reactivate the Flox environment
- **Solution**: Exit and re-enter: `flox deactivate` then `flox activate`

### Environment Verification Commands

**Check if you're in the activated Flox environment:**

```bash
# Method 1: Check environment variable (flox exports this while active)
echo $FLOX_ENV  # Should output a /nix/store/... path, not empty

# Method 2: Check Go version
go version  # Should match the `go.version` pin in .flox/env/manifest.toml

# Method 3: Check tool availability
which ast-grep  # Should show path in /nix/store/...
```

**Verify specific tools:**

```bash
# Check all Flox-provided tools are available
ast-grep --version
fd --version
ripgrep --version
gopls version
golangci-lint --version
gotestsum --version
jq --version
yq --version
deno --version
go version  # Should match the `go.version` pin in .flox/env/manifest.toml
```

**List all available just recipes:**

```bash
just --list  # Shows all recipes defined in the justfile
```

### Activation Hook Automation

When you run `flox activate` (or direnv auto-activates), these commands run automatically
(the manifest's `[hook]` on-activate, plus `[profile]` for interactive-shell aliases):

1. `go mod tidy` - Ensures Go dependencies are clean
2. Neovim config setup - Exports `NVIM_PROJECT_CONFIG` and (in an interactive shell) aliases `vim`/`nvim` to load the project-specific Neovim config (if `.config/nvim` exists)

**This means:**

- Dependencies are always up-to-date when entering the environment

Git hooks are separately managed by nix (tc-5lxy.5) — see
[Git Workflow](docs/GIT_WORKFLOW.md#git-hooks). They install/refresh automatically on `nix develop`
devShell entry, or via `nix run .#install-pre-commit-hooks`.

### Assumptions in Documentation

**When you see `just <recipe>`:**

- Assumes `just` is on PATH (it's one of the Flox-installed packages)
- Assumes you're in project directory (where `justfile` exists)
- Can be run from any subdirectory of the project

**When you see `ast-grep`, `jq`, `yq`, etc:**

- Assumes the Flox environment is activated
- These are NOT system commands, they're Flox-provided

**When you see `go build`, `go test`, etc:**

- Assumes the Flox environment is activated (using the Go version pinned in `.flox/env/manifest.toml`)
- Assumes `go mod tidy` has run (automatic in the activation hook)

**When you see scripts like `bash scripts/something.sh`:**

- Assumes script has executable permissions
- Assumes bash is available (standard on Linux/macOS)
- Assumes running from project root

### Quick Reference Table

| What                     | Where            | Command                                                               |
| ------------------------ | ---------------- | --------------------------------------------------------------------- |
| **Enter Flox env**       | Any directory    | `flox activate` (or direnv auto-activation)                           |
| **Exit Flox env**        | In Flox env      | `flox deactivate` or Ctrl+D                                           |
| **Check if in Flox env** | In shell         | `echo $FLOX_ENV`                                                      |
| **Verify Go version**    | In Flox env      | `go version` (should match `go.version` in `.flox/env/manifest.toml`) |
| **Run tests**            | In Flox env      | `just tests`                                                          |
| **Validate docs**        | In Flox env      | `just validate-docs`                                                  |
| **Full CI pipeline**     | In Flox env      | `just ci`                                                             |
| **List all recipes**     | In Flox env      | `just --list`                                                         |
| **Update environment**   | Outside Flox env | `flox update`                                                         |

### Why Flox?

**Benefits:**

- **Reproducible**: Exact same environment on every machine
- **Isolated**: Doesn't pollute global system with project tools
- **Declarative**: Environment defined in `.flox/env/manifest.toml`
- **Versioned**: Specific tool versions guaranteed (pins declared in the manifest)
- **Fast**: Nix caching makes environment activation quick
- **Comprehensive**: All 13 tools plus `just` in one `flox activate` command
- **Workspace-native**: Nix-based, so it composes with the rest of this repo's shared `pn-workspace.toml` infrastructure (see [ADR-0006](docs/adr/0006-flox-based-development-environment.md))

**Alternative:** If Flox isn't available, see [Development Guide - Manual Setup](docs/DEVELOPMENT.md#manual-setup) for installing tools individually.

## Development Commands

For verification workflow and quality commands, see [Verification Workflow](docs/VERIFICATION_WORKFLOW.md).

```bash
# Development shortcuts
flox activate        # Enter development environment (or rely on direnv auto-activation)
just builder         # Build all packages

# Git hooks (quality enforcement) -- managed by nix (tc-5lxy.5)
nix run .#install-pre-commit-hooks  # Install/refresh pre-commit + pre-push hooks
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

Issue tracking is `bd` (beads) -- the git-based `issues/{backlog,ready,active}/` markdown
tracker and its supporting commands/scripts were retired by tc-5lxy.22.
`issues/completed/` still holds the pre-beads historical archive on disk until tc-5lxy.24
imports it; it is read-only until then.

### Quick Reference

1. **Create issue**: `bd create -t feature "title"` or `bd create -t bug "title"`
2. **Groom issue**: bead-grooming skill (single-pass) or `/prepare-issue <bd-id>` (deeper
   5-stage review pipeline) once enough detail exists to work from
3. **Implement**: `/drain-beads` (autonomous claim -> implement -> validate -> land -> close),
   or claim and work a bead directly
4. **Complete**: `bd close <bd-id>` as part of landing the change

## Documentation Rules

### Living Documentation

- **docs/SPECIFICATION.md** is a **living representation** of the project
- **MUST be updated** whenever documentation changes to match current system state
- Serves as the single source of truth for current architecture and capabilities

### Completed Issues Policy

- **Completed issues** (in `issues/completed/`) should **NOT be updated**
- They serve as historical records of what was implemented
- **Allowed exceptions only**:
  - Adding cross-references to newer issues that modified the functionality
  - Minor text improvements (typos, readability) that don't change interpretation

### Documentation Update Workflow

1. **Always review** `docs/SPECIFICATION.md` when updating docs
2. **Verify** code state matches documentation
3. **Update** specification.md if system has evolved
4. **Preserve** completed issues as historical records
5. **Cross-reference** when newer issues supersede older ones

### Documentation Health Dashboard Maintenance

The project maintains a documentation health dashboard in `docs/INDEX.md` that tracks metrics and quality indicators.

**Automated Updates** (No Agent Action Required):

- Metrics are automatically updated by `scripts/update-doc-health.sh` via pre-commit hooks
- Automated metrics include: file counts, line counts, broken links, freshness, validation status
- These update on every commit that touches documentation files

**Agent Responsibilities** (Update When Needed):
Agents MUST update the qualitative sections in `docs/INDEX.md` dashboard when:

1. **After completing documentation tasks** that:
   - Add new documentation files (update Coverage by Category)
   - Close documentation gaps (remove from Known Gaps, update coverage status)
   - Identify new gaps (add to Known Gaps)
   - Significantly restructure existing docs (update Recent Significant Changes)

2. **When health status changes**:
   - Coverage drops below 90% in any category (update status to ⚠️ or 🟠)
   - All gaps in a category are closed (update status to ✅)
   - Critical issues arise (update Overall Health assessment)

3. **During periodic reviews**:
   - Weekly: Review and update Action Items based on automated metrics
   - When running `/review-and-update-documentation` command
   - When stale docs are identified (>45 days old)

**What to Update** (Agent-Maintained Sections):

```markdown
### Overall Health: 🟢/🟡/🟠/🔴

<!-- Update if assessment changes -->

### Coverage by Category

<!-- Update when new docs added or gaps closed -->

### Known Gaps

<!-- Add when gaps identified, remove when closed -->

### Action Items

<!-- Update priorities based on automated metrics -->

### Recent Significant Changes

<!-- Add entry for major documentation work -->
```

**How to Update**:

1. Read current dashboard state from `docs/INDEX.md`
2. Update only the agent-maintained sections (marked with comments)
3. Do NOT modify auto-generated sections (timestamps, metrics, freshness)
4. Include brief explanation in commit message
5. Dashboard will auto-update metrics on commit

**Example Agent Update**:

```markdown
# Agent completed Docker troubleshooting documentation

Updates to docs/INDEX.md dashboard:

1. Coverage by Category: Troubleshooting ✅ Complete (was ⚠️ Good)
2. Known Gaps: Removed "Docker troubleshooting"
3. Action Items: Marked "Add Docker troubleshooting" as complete
4. Recent Changes: Added entry about new Docker guide
```

**When NOT to Update**:

- Minor typo fixes that don't affect coverage
- Formatting/style changes only
- Updating metadata (Last Updated dates)
- Changes to automated metrics sections

## Documentation Architecture & Guidelines

**CRITICAL**: README.md MUST stay under 300 lines to prevent bloat and ensure discoverability.

### Documentation Structure

The project follows a hierarchical documentation structure designed to optimize user experience:

```
README.md (<300 lines)     # Project overview, quick install, basic usage, navigation
├── docs/INSTALLATION.md   # Comprehensive installation methods & troubleshooting
├── docs/CLI_REFERENCE.md  # Complete command documentation & examples
├── docs/DEVELOPMENT.md    # Development setup, testing, CI/CD workflows
├── docs/DEPLOYMENT.md     # Production deployment & Docker usage
├── docs/INDEX.md          # Documentation navigation guide
└── docs/                  # Specialized documentation (existing structure)
    ├── ARCHITECTURE.md    # System design & architectural decisions
    ├── GIT_WORKFLOW.md    # Git standards & commit rules
    ├── VERIFICATION_WORKFLOW.md  # Quality verification commands
    └── [other specialized docs]
```

### Documentation Placement Decision Tree

When adding or updating documentation, use this decision tree:

**Step 1: Is this essential for new users?**

- **YES** → Add to README.md (if under 300 line limit)
- **NO** → Continue to Step 2

**Step 2: What type of content is this?**

- **Installation methods/troubleshooting** → docs/INSTALLATION.md
- **CLI commands/usage examples** → docs/CLI_REFERENCE.md
- **Development workflows/setup** → docs/DEVELOPMENT.md
- **Production deployment** → docs/DEPLOYMENT.md
- **System architecture/design** → docs/ARCHITECTURE.md
- **Git/commit standards** → docs/GIT_WORKFLOW.md
- **Testing/quality workflows** → docs/VERIFICATION_WORKFLOW.md
- **Issue management workflows** → `bd` (beads); see this file's "Issue Development Workflow" section
- **Other specialized topics** → Create appropriate docs/[TOPIC].md

**Step 3: README.md Content Rules**
README.md should ONLY contain:

1. Project overview & badges (10-15 lines)
2. Quick installation (basic method only) (15-20 lines)
3. Essential usage examples (2-3 basic commands) (30-40 lines)
4. Documentation navigation (clear links to detailed docs) (20-30 lines)
5. Contributing quick start (basic info only) (10-15 lines)
6. License & essential links (5-10 lines)

**Step 4: Content Migration Strategy**
When README.md approaches 280 lines:

1. Identify non-essential content for migration
2. Move detailed examples to appropriate docs/ files
3. Replace with summary + link to detailed documentation
4. Test all navigation links work correctly

### Agent Documentation Guidelines

**For ALL agents working on this project:**

1. **NEVER add detailed content to README.md**
   - README.md is for overview and navigation only
   - Detailed information belongs in specialized docs/ files

2. **Always check README.md line count**
   - Use `wc -l README.md` to verify line count
   - If approaching 280 lines, migrate content before adding

3. **Use appropriate documentation files**
   - Follow the decision tree above for placement
   - Create new docs/ files only when necessary
   - Update docs/INDEX.md when adding new documentation
   - Update docs/INDEX.md dashboard after significant documentation changes

4. **Maintain cross-references**
   - Update all related documentation when making changes
   - Ensure links between documents remain valid
   - Use absolute paths for all documentation links

5. **Content quality standards**
   - Keep each documentation file focused on single responsibility
   - Use clear headings and navigation
   - Include examples relevant to the specific topic
   - Avoid duplication between files

### Documentation Validation Requirements

Before completing any documentation task:

1. **Line Count Verification**

   ```bash
   wc -l README.md  # Must be < 300 lines
   ```

2. **Link Validation**
   - Test all internal links work correctly
   - Verify cross-references between documents
   - Ensure navigation flows logically

3. **Content Completeness**
   - All information preserved in appropriate locations
   - No gaps in documentation coverage
   - Clear navigation between related topics

4. **User Experience Testing**
   - New user can find installation in <30 seconds
   - Developer can find contribution info in <1 minute
   - Documentation flows from high-level to detailed

### Memory Files for Documentation Architecture

The following memory files preserve documentation architecture decisions:

- **Documentation Architecture Standards**: Core principles and structure
- **README Content Limits**: Specific content rules and line count requirements
- **Content Migration Patterns**: Examples of what content goes where
- **FEAT-076 Implementation**: Rationale and goals for documentation restructuring

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

- `just test-unit`: Fast unit tests only (uses `gotestsum` for better output)
- `just test-integration`: Integration tests only (CLI and file I/O tests)
- `just tests`: Full test suite (both unit and integration tests with enhanced output)

#### Test Development Workflow

1. **During development**: Use `just test-unit` for rapid feedback
2. **Before committing**: Run `just tests` to ensure all tests pass
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

### `extends:` / `additional-tools:` Are Not Real Frontmatter Fields

Confirmed 2026-09-17 (tc-ijhxa): Claude Code's subagent loader does not support `extends:` or
`additional-tools:` in agent frontmatter -- they are silently ignored, not an error. The only real
tool-control fields are `tools:` (allowlist) and `disallowedTools:` (denylist); if `tools:` is
omitted entirely, the agent inherits the full default tool set (including unrestricted Bash). Do
not use `extends`/`additional-tools` to compose a tool list from `.claude/agents/templates/`; flatten
the full resolved list directly into the agent's own `tools:` field instead.

### Required Agent Permissions

The following agents require full access to all Serena MCP tools:

**Agents Requiring Serena MCP Access:**

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

**Optional Serena MCP access (degrades gracefully):** `technical-design-reviewer`,
`test-strategy-reviewer`, and `implementation-planner` also carry a handful of read-only
`mcp__serena__*` tools in their `tools:` allowlist (from the deleted `base-review-agent`
template, flattened directly into each file's frontmatter by tc-ijhxa). Unlike the three
agents above, these are review-only agents for which Serena access is a preference, not a
requirement -- if the Serena MCP is absent, they fall back to `Read`/`Grep` per the general
Code Analysis Workflow guidance below.

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

### Project Documentation

- **Architecture**: `docs/ARCHITECTURE.md` - System architecture and design decisions
- **Troubleshooting**: `docs/TROUBLESHOOTING.md` - Test/lint failures and fixes
- **Version Management**: `docs/VERSION_MANAGEMENT.md` - Release workflow
- **Session Learnings**: `docs/SESSION_LEARNINGS.md` - Implementation insights
- **Specification**: `docs/SPECIFICATION.md` - Detailed technical specs

## Common Pitfalls and Anti-Patterns

This section documents frequent mistakes agents make and how to avoid them.

### Git and Staging Mistakes

**❌ NEVER DO**: `git add .` (stages everything including unrelated files)
**✅ DO INSTEAD**: `git add pkg/specific/file.go pkg/specific/file_test.go` (stage only modified files)

**❌ NEVER DO**: `git commit --no-verify` (skips pre-commit hooks)
**✅ DO INSTEAD**: Fix the issues causing hook failures, then commit normally

**❌ NEVER DO**: Commit files you didn't modify for this task
**✅ DO INSTEAD**: Use `git status` before and after to identify only your changes

### Verification Mistakes

**❌ NEVER DO**: Mark task complete before running verification workflow
**✅ DO INSTEAD**: Run formatter → tests → linter → build-cli, THEN mark complete

**❌ NEVER DO**: Skip verification because "it's just documentation"
**✅ DO INSTEAD**: Always run full verification - markdown commits skip tests automatically

**❌ NEVER DO**: Mark task complete with failing tests/linter
**✅ DO INSTEAD**: Fix ALL issues before marking complete

### TodoWrite Mistakes

**❌ NEVER DO**: Have multiple tasks as `in_progress` simultaneously
**✅ DO INSTEAD**: Only ONE task `in_progress` at a time

**❌ NEVER DO**: Batch multiple task completions at once
**✅ DO INSTEAD**: Mark each task complete immediately after finishing it

**❌ NEVER DO**: Use TodoWrite for trivial single-step tasks
**✅ DO INSTEAD**: Reserve TodoWrite for complex multi-step tasks (3+ steps)

### Code Analysis Mistakes

**❌ NEVER DO**: Use `grep` or `ripgrep` for finding Go functions/types
**✅ DO INSTEAD**: Use Serena MCP tools (`mcp__serena__find_symbol`) for semantic search

**❌ NEVER DO**: Modify code without understanding structure first
**✅ DO INSTEAD**: Use `mcp__serena__get_symbols_overview` before making changes

**❌ NEVER DO**: Use `xml.NewDecoder` directly for XML parsing
**✅ DO INSTEAD**: Always use `security.NewSecureXMLDecoder` to prevent XXE vulnerabilities

### Documentation Mistakes

**❌ NEVER DO**: Add detailed content to README.md (line limit: 300)
**✅ DO INSTEAD**: Add to appropriate docs/ file and link from README

**❌ NEVER DO**: Update completed issues in `issues/completed/`
**✅ DO INSTEAD**: Cross-reference from new issues if functionality changed

**❌ NEVER DO**: Create documentation files without updating docs/INDEX.md
**✅ DO INSTEAD**: Update INDEX.md when adding new documentation

**❌ NEVER DO**: Complete documentation tasks without updating docs/INDEX.md dashboard
**✅ DO INSTEAD**: Update dashboard qualitative sections after significant doc changes

### Quick Fixes

When stuck on common issues:

1. **Tests failing**: Run `just formatter` first - formatting fixes many test issues
2. **Linter errors**: Check if it's an import issue - run `go mod tidy`
3. **Build failing**: Verify all imports use full paths: `github.com/phillipgreenii/mobilecombackup/pkg/...`
4. **Git hook blocking**: Run `nix run .#install-pre-commit-hooks` to reinstall/refresh, or `prek run` to see hook output directly (see [Git Workflow](docs/GIT_WORKFLOW.md#git-hooks))
5. **Can't find package**: Verify it exists in `pkg/` directory with correct name

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
