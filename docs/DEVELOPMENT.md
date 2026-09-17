# Development Guide

Complete guide to setting up, developing, testing, and contributing to MobileComBackup.

---

**Last Updated**: 2025-01-15
**Related Documents**: [Git Workflow](GIT_WORKFLOW.md) | [Verification Workflow](VERIFICATION_WORKFLOW.md) | [Architecture](ARCHITECTURE.md)
**Prerequisites**: Nix with flakes, Git

---

## Quick Start for Contributors

```bash
# 1. Clone repository
git clone https://github.com/phillipgreenii/mobilecombackup.git
cd mobilecombackup

# 2. Enter development environment
flox activate
# (or rely on direnv auto-activation: `direnv allow` once, then cd in)

# 3. Run tests
just tests

# 4. Make changes and test
just ci  # Full CI pipeline
```

## Development Environment Setup

### Prerequisites

- **Nix with flakes**: For reproducible development environment
- **Git**: Version control and contribution workflow

### Using Flox (Recommended)

Flox provides a consistent development environment with all required tools, declared in
`.flox/env/manifest.toml` (locked in `.flox/env/manifest.lock`):

```bash
# Install flox (if not already installed) -- see https://flox.dev/docs/install-flox/

# Enter development environment
flox activate

# Or, with direnv (the repo's .envrc contains `use flox`): run `direnv allow`
# once, and the environment auto-activates whenever you cd into the repo.

# Available tools in the environment:
# - Go 1.26.5
# - just (task runner for the commands below)
# - golangci-lint (code linting)
# - gotestsum (enhanced test output)
# - claude-code (AI development assistant)
```

### Manual Setup

If you prefer manual setup:

```bash
# Install Go 1.24+
# Install golangci-lint for linting
# Install gotestsum for enhanced test output (optional)
```

## Development Workflows

### Core Development Commands

```bash
# Build all packages
just builder

# Run tests with enhanced output
just tests

# Run linting
just linter

# Format code
just formatter

# Build CLI with version information
just build-cli

# Run complete CI pipeline locally
just ci
```

### Nix packaging (gomod2nix)

Day-to-day development uses flox (with `just` as the task runner); the Nix flake exists for packaged
distribution (`nix build`, `nix run`). The flake builds the CLI with
`mkGoBinary` from `nix-repo-base`, which resolves Go dependencies from the
committed `gomod2nix.toml` instead of a hand-maintained `vendorHash`.

`gomod2nix.toml` MUST be regenerated and committed whenever `go.mod` or `go.sum`
changes (adding, removing, or bumping a dependency):

```bash
# Regenerate the Nix dependency lockfile after any go.mod/go.sum change
nix run github:nix-community/gomod2nix -- generate

# Then commit the result
git add gomod2nix.toml
```

Regeneration needs network access, so it cannot run inside a `nix flake check`
derivation. The `gomod2nix-drift` job in `.github/workflows/test.yml` regenerates
the file in CI and fails if it differs from the committed copy.

Package-level checks run through the flake:

```bash
# Build the package and run the packaging/version/help checks
nix flake check

# Build just the CLI
nix build .#mobilecombackup
```

`nix build` produces the binary plus a man page, bash/zsh/fish completions, and
the tldr page from `docs/tldr/mobilecombackup.md`; `checks.packaging` asserts all
of them are present, because the generators only warn on failure.

The nix-built `--version` string is `<semver>-<8hex>`, e.g.
`mobilecombackup version 2.0.0-7719e3bf`. The semver half comes from the
`VERSION` file and the 8-hex suffix is a digest of the package's own source tree
(ADR 0006 digest versioning in `nix-repo-base`), so the version changes when the
source changes rather than on every commit. This is distinct from the
`just build-cli` version string, which still embeds the git description.

### Git Hooks and Quality Enforcement

Pre-commit hooks are managed by the nix-repo-base `flakeModules.pre-commit` module (tc-5lxy.5).
See [Git Workflow](GIT_WORKFLOW.md#git-hooks) for installation, the one-time
`core.hooksPath` migration for older clones, and the hook set (treefmt, statix, deadnix,
shellcheck, trailing-whitespace/end-of-file fixers, etc.).

**CRITICAL**: Every commit MUST pass ALL quality checks. NEVER use `git commit --no-verify`.

## Testing Strategy

### Test Types and Commands

The project has comprehensive testing with different scopes:

```bash
# Fast unit tests only (uses gotestsum for better output)
just test-unit

# Integration tests only (CLI and file I/O tests)
just test-integration

# Full test suite (both unit and integration tests with enhanced output)
just tests

# Run tests with coverage
go test -v -covermode=set ./...
```

### Test Development Guidelines

#### Test Organization

- **Unit Tests**: Fast, isolated logic testing
  - Add `t.Parallel()` to pure logic tests for performance
  - Target 80%+ coverage
  - Test both success and failure paths

- **Integration Tests**: CLI and file I/O testing
  - Use `testing.Short()` to skip in unit-only runs
  - Test real file system interactions
  - Validate CLI command integration

#### Test Data Management

- Use `testdata/` directories for test files
- Test data has intentional quirks (count mismatches, mixed years) to verify robustness
- Create realistic test scenarios that mirror production data

#### Test Development Workflow

1. **During development**: Use `just test-unit` for rapid feedback
2. **Before committing**: Run `just tests` to ensure all tests pass
3. **Create examples**: Add `example_test.go` files for usage documentation

### Example Test Structure

```go
func TestProcessCalls(t *testing.T) {
    t.Parallel() // For unit tests

    // Test success path
    t.Run("valid input", func(t *testing.T) {
        // Test implementation
    })

    // Test failure path
    t.Run("invalid input", func(t *testing.T) {
        // Error handling test
    })
}

func TestCLIIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
    // Integration test implementation
}
```

## Code Standards and Guidelines

### Architecture Principles

- **Streaming APIs**: Process large files without loading into memory
- **Error resilience**: Continue on individual failures, collect errors for reporting
- **Hash-based storage**: SHA-256 for content addressing
- **Interface-first**: Define APIs before implementation
- **UTC-based**: All timestamps and year partitioning use UTC

### Error Handling Patterns

```go
// Good: Return errors, include context
if err != nil {
    return fmt.Errorf("failed to process calls: %w", err)
}

// Bad: Exit from libraries
if err != nil {
    os.Exit(1) // Don't do this in library code
}

// Good: Collect errors for batch reporting
var errors []error
for _, item := range items {
    if err := processItem(item); err != nil {
        errors = append(errors, err)
        continue // Keep processing
    }
}
```

### XML Security Requirements

**CRITICAL**: Always use secure XML parsing to prevent XXE attacks:

```go
// Good: Use security wrapper
decoder := security.NewSecureXMLDecoder(reader)

// Bad: Direct XML decoder usage (prohibited)
decoder := xml.NewDecoder(reader) // XXE vulnerability
```

### File Organization Standards

- `types.go`: Structs and interfaces
- `reader.go`: Main implementation
- `*_test.go`: Unit and integration tests
- `example_test.go`: Usage examples

## Continuous Integration

### Local CI Pipeline

Run the complete CI pipeline locally before pushing:

```bash
just ci
```

This executes:

1. `just formatter` (`nix fmt` -- treefmt/gofumpt; stricter than plain `go fmt ./...`)
2. `just tests` (full test suite with coverage)
3. `just linter` (golangci-lint run)
4. `just build-cli` (versioned binary build)

### CI Environment

The same CI pipeline runs automatically on:

- Pull requests to main branch
- Pushes to main branch
- Manual workflow dispatch
- Release builds (tags)

All CI workflows install flox (via `flox/install-flox-action`) and run commands as
`flox activate -- <command>` to ensure consistency between local development and CI environments.

### Code Quality Analysis

The project integrates with SonarQube Cloud for automated code quality analysis:

- **Quality Gate**: Ensures code meets maintainability and reliability standards
- **Coverage Tracking**: Monitors test coverage trends and identifies untested code
- **Security Analysis**: Scans for potential security vulnerabilities
- **Code Smells Detection**: Identifies maintainability issues and technical debt
- **Duplication Analysis**: Tracks code duplication across the codebase

Quality metrics are automatically updated on every push and pull request. View the [SonarCloud dashboard](https://sonarcloud.io/project/overview?id=phillipgreenii_mobilecombackup) for detailed analysis reports.

## Version Management

### Version System

The project uses git tag-based versioning with fallback to VERSION file:

- **Development builds**: `2.0.0-dev-g1234567` (base version + git hash)
- **Release builds**: `2.0.0` (clean semantic version from git tags)

### Version Sources (Priority Order)

1. **Git tags**: For release builds (e.g., `v2.0.0` → `2.0.0`)
2. **VERSION file + git hash**: For development builds
3. **VERSION file only**: When git is unavailable
4. **Fallback**: `dev` when no version source available

### Version Validation

```bash
# Check the version of built binary
$ mobilecombackup --version
mobilecombackup version 2.0.0-dev-g1234567

# Validate version file format
$ just validate-version
```

## Contribution Workflow

### Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/yourusername/mobilecombackup.git
   cd mobilecombackup
   ```
3. **Set up development environment**:
   ```bash
   flox activate
   nix run .#install-pre-commit-hooks
   ```

### Development Process

1. **Create feature branch**:

   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make changes** following code standards:
   - Write tests for new functionality
   - Update documentation if needed
   - Follow existing code patterns

3. **Test thoroughly**:

   ```bash
   just ci  # Full CI pipeline
   ```

4. **Commit with quality checks**:

   ```bash
   git add .
   git commit -m "feat: add your feature description"
   # Pre-commit hooks run automatically
   ```

5. **Push and create pull request**:
   ```bash
   git push origin feature/your-feature-name
   # Create PR on GitHub
   ```

### Pull Request Guidelines

- **Clear description**: Explain what the PR does and why
- **Test coverage**: Include tests for new functionality
- **Documentation**: Update relevant docs if needed
- **Quality checks**: Ensure CI passes
- **Small focused changes**: Easier to review and merge

## Issue Development Workflow

Issue tracking is `bd` (beads); the git-based `issues/` markdown tracker was retired by
tc-5lxy.22. See the project [CLAUDE.md](../CLAUDE.md#issue-development-workflow) for the
current workflow.

### Quick Reference

1. **Create issue**: `bd create -t feature "title"` or `bd create -t bug "title"`
2. **Groom issue**: bead-grooming skill, or `/prepare-issue <bd-id>` for the deeper 5-stage
   review pipeline
3. **Implement**: `/drain-beads`, or claim and work a bead directly
4. **Complete**: `bd close <bd-id>` as part of landing the change

## Development Tools and Analysis

### Preferred Tool Hierarchy

For code analysis tasks, prefer this hierarchy:

1. **Serena MCP tools** - Semantic symbol search and code structure analysis
2. **ast-grep** - Structural patterns when Serena MCP insufficient
3. **ripgrep/grep** - Only for non-code text search

### Serena MCP Workflow Examples

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

### Common ast-grep Patterns

```bash
# Find function definitions
ast-grep --pattern 'func $NAME($$$) $RET { $$$ }'

# Find error handling patterns
ast-grep --pattern 'if err != nil { $$$ }'

# Find test functions
ast-grep --pattern 'func Test$_($$$) { $$$ }'
```

## Troubleshooting Development Issues

### Common Problems

#### Build Failures

```bash
# Fix: Ensure clean environment
flox activate
just builder
```

#### Test Failures

```bash
# Check specific test output
go test -v ./path/to/package

# Run with race detection
go test -race ./...
```

#### Linting Issues

```bash
# See specific linting problems
golangci-lint run

# Auto-fix some issues
golangci-lint run --fix
```

#### Git Hook Issues

```bash
# Reinstall/refresh hooks if they're not working
nix run .#install-pre-commit-hooks

# Run hooks manually against staged files
prek run
```

### Getting Help

- **Review documentation**: [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
- **Check existing issues**: [GitHub Issues](https://github.com/phillipgreenii/mobilecombackup/issues)
- **Create detailed issue**: Include environment info and error messages

## Architecture and Design

For detailed architecture information, see:

- **[Architecture Overview](ARCHITECTURE.md)** - System design and technical decisions
- **[ADR Index](adr/index.md)** - Architecture Decision Records
- **[Session Learnings](SESSION_LEARNINGS.md)** - Implementation insights

## Next Steps

After setting up your development environment:

- **[Complete CLI Reference](CLI_REFERENCE.md)** - Understand all available commands
- **[Architecture Overview](ARCHITECTURE.md)** - Learn system design principles
- **[Issue Development Workflow](#issue-development-workflow)** - Understand development process
- **[Troubleshooting Guide](TROUBLESHOOTING.md)** - Fix common development issues

---

📖 **[Documentation Index](INDEX.md)** | 🏠 **[Back to README](../README.md)**
