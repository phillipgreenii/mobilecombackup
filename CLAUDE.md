# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go command-line tool for processing mobile phone backup files (Call and SMS logs in XML format). It coalesces multiple backup files, removes duplicates, extracts attachments, and organizes data by year.

## Quick Commands Reference

### Most Common Development Commands
```bash
# Environment
devbox shell              # Enter development environment
devbox run ci             # Run full CI pipeline (format + test + lint + build)

# Testing & Quality
devbox run test           # Run all tests
devbox run formatter      # Format code
devbox run linter         # Run linter

# Building
devbox run build-cli      # Build CLI with version info
```

### Issue Workflow Commands
```bash
# Create issues
/create-feature "description"    # Create new feature issue
/create-bug "description"        # Create new bug issue

# Prepare issues
/review-issue FEAT-XXX           # Review issue specification
/ready-issue FEAT-XXX            # Move to ready/ when complete

# Implement issues
/implement-issue FEAT-XXX        # Start implementation (moves to active/)

# Batch operations
/ready-backlog-issues            # Process all backlog issues
/plan-and-implement-ready-issues # Implement all ready issues
```

### File Locations Quick Reference
- **Source code**: `pkg/` (Go packages) and `cmd/mobilecombackup/` (CLI)
- **Documentation**: `docs/` (specialized docs) and `README.md` (overview)
- **Issues**: `issues/{backlog,ready,active,completed}/`
- **Templates**: `issues/{feature_template,bug_template}.md`
- **Scripts**: `scripts/` and `issues/create-issue.sh`

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
1. **Create issue**: Use automation script or slash commands
   - **Preferred**: `./issues/create-issue.sh FEATURE "title"` or `./issues/create-issue.sh BUG "title"`
   - **Alternative**: `/create-feature` or `/create-bug` commands
2. **Plan issue**: Fill details in `issues/backlog/FEAT-XXX.md`
3. **Ready issue**: Move to `issues/ready/` when planned
4. **Implement**: Use `/implement-issue FEAT-XXX` command
5. **Complete**: Updates move to `issues/completed/`

### Issue Creation Automation (FEAT-075)
The project includes an automated issue creation script that reduces manual overhead:

```bash
# Create new feature issue
./issues/create-issue.sh FEATURE "implement user authentication"
# Creates: issues/backlog/FEAT-076-implement-user-authentication.md

# Create new bug issue  
./issues/create-issue.sh BUG "validation fails on empty input"
# Creates: issues/backlog/BUG-077-validation-fails-on-empty-input.md
```

**Benefits:**
- Automatic sequential numbering across all issue types
- Kebab-case title conversion with comprehensive error handling
- Template copying and title replacement
- Comprehensive validation and colorized feedback
- Executes in under 1 second

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
- **Issue management workflows** → docs/ISSUE_WORKFLOW.md
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

1. **Tests failing**: Run `devbox run formatter` first - formatting fixes many test issues
2. **Linter errors**: Check if it's an import issue - run `go mod tidy`
3. **Build failing**: Verify all imports use full paths: `github.com/phillipgreenii/mobilecombackup/pkg/...`
4. **Git hook blocking**: Check `.githooks/pre-commit` - it auto-detects markdown-only commits
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