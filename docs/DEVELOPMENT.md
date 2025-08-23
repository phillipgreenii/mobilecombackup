# Development Guide

Complete guide to setting up, developing, testing, and contributing to MobileComBackup.

## Quick Start for Contributors

```bash
# 1. Clone repository
git clone https://github.com/phillipgreenii/mobilecombackup.git
cd mobilecombackup

# 2. Enter development environment
devbox shell

# 3. Run tests
devbox run test

# 4. Make changes and test
devbox run ci  # Full CI pipeline
```

## Development Environment Setup

### Prerequisites

- **Nix with flakes**: For reproducible development environment
- **Git**: Version control and contribution workflow

### Using Devbox (Recommended)

Devbox provides a consistent development environment with all required tools:

```bash
# Install devbox (if not already installed)
curl -fsSL https://get.jetify.com/devbox | bash

# Enter development environment
devbox shell

# Available tools in the environment:
# - Go 1.24
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
devbox run builder

# Run tests with enhanced output
devbox run test

# Run linting
devbox run linter

# Format code
devbox run formatter

# Build CLI with version information
devbox run build-cli

# Run complete CI pipeline locally
devbox run ci
```

### Git Hooks and Quality Enforcement

The project uses pre-commit hooks to enforce code quality:

```bash
# Install git hooks
devbox run install-hooks

# Test hooks without committing
devbox run test-hooks
```

#### Pre-commit Hook Optimization

The pre-commit hook is optimized for different workflow types:

- **Markdown-only commits**: Skip tests, run formatter + linter only (~6s, target <10s)
- **Code/mixed commits**: Run full checks (formatter + tests + linter, target <30s)
- **Automatic detection**: Uses `git diff --cached --name-only` to analyze staged files
- **Clear feedback**: Shows optimization decisions and performance metrics

**CRITICAL**: Every commit MUST pass ALL quality checks. NEVER use `git commit --no-verify`.

## Testing Strategy

### Test Types and Commands

The project has comprehensive testing with different scopes:

```bash
# Fast unit tests only (uses gotestsum for better output)
devbox run test-unit

# Integration tests only (CLI and file I/O tests)
devbox run test-integration  

# Full test suite (both unit and integration tests with enhanced output)
devbox run test

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

1. **During development**: Use `devbox run test-unit` for rapid feedback
2. **Before committing**: Run `devbox run test` to ensure all tests pass
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
devbox run ci
```

This executes:
1. `devbox run formatter` (go fmt ./...)
2. `devbox run test` (full test suite with coverage)
3. `devbox run linter` (golangci-lint run)  
4. `devbox run build-cli` (versioned binary build)

### CI Environment

The same CI pipeline runs automatically on:
- Pull requests to main branch
- Pushes to main branch
- Manual workflow dispatch  
- Release builds (tags)

All CI workflows use devbox to ensure consistency between local development and CI environments.

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
$ devbox run validate-version
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
   devbox shell
   devbox run install-hooks
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
   devbox run ci  # Full CI pipeline
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

For complete issue development workflow, see [Issue Workflow](ISSUE_WORKFLOW.md).

### Quick Reference

1. **Create issue**: Use `/create-feature` or `/create-bug` commands
2. **Plan issue**: Fill details in `issues/backlog/FEAT-XXX.md`
3. **Ready issue**: Move to `issues/ready/` when planned
4. **Implement**: Use `/implement-issue FEAT-XXX` command
5. **Complete**: Updates move to `issues/completed/`

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
devbox shell --pure
devbox run builder
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
# Reinstall hooks if they're not working
devbox run install-hooks

# Test hooks manually
devbox run test-hooks
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
- **[Issue Workflow](ISSUE_WORKFLOW.md)** - Understand development process
- **[Troubleshooting Guide](TROUBLESHOOTING.md)** - Fix common development issues

---

📖 **[Documentation Index](INDEX.md)** | 🏠 **[Back to README](../README.md)**