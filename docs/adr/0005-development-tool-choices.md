# ADR-0005: Development Tool Ecosystem Choices

**Status:** Accepted
**Date:** 2024-01-15
**Author:** Development Team
**Deciders:** Core development team

## Context

We needed to establish a development toolchain that supports:

1. **Reproducible builds**: Consistent development environments
2. **Code quality**: Automated formatting, linting, and testing
3. **Cross-platform development**: Works on different operating systems
4. **Dependency management**: Reliable package and tool versioning
5. **Developer productivity**: Fast feedback loops and easy setup

The toolchain affects all aspects of development workflow and long-term maintainability.

## Decision

We chose **Devbox** as the primary development environment manager with these core tools:

### Environment Management
- **Devbox**: Reproducible development environments with Nix
- **golangci-lint**: Comprehensive Go linting
- **gotestsum**: Enhanced test output and reporting

### Code Quality Tools
- **gofmt/gofumpt**: Code formatting
- **staticcheck**: Advanced static analysis
- **gosec**: Security-focused linting
- **govet**: Go standard linting

### Testing and Coverage
- **Go built-in testing**: Standard test framework
- **testdata/**: Test fixture organization
- **Coverage reporting**: Built-in Go coverage tools

## Rationale

### Devbox for Environment Management
- **Reproducible environments**: Nix-based package management ensures consistency
- **Cross-platform support**: Works on macOS, Linux, and Windows (WSL)
- **Version pinning**: Exact tool versions specified and reproducible
- **Zero-config sharing**: New developers get identical environment immediately
- **Isolation**: Project dependencies don't conflict with system tools

### Comprehensive Linting Strategy
- **golangci-lint**: Meta-linter running multiple linters efficiently
- **Static analysis**: Catches bugs before runtime
- **Security scanning**: gosec integration for security vulnerabilities
- **Performance hints**: Linters identify performance anti-patterns
- **Consistency enforcement**: Automated code style consistency

### Enhanced Testing Experience
- **gotestsum**: Better test output formatting and progress indication
- **Parallel testing**: Efficient test execution with `-parallel` flags
- **Coverage tracking**: Visibility into test coverage metrics
- **Integration testing**: Separate test categories for different test types

### Git Hooks Integration
- **Pre-commit validation**: Automated quality checks before commits
- **Fast feedback**: Immediate notification of quality issues
- **Consistent standards**: All commits meet quality requirements automatically

### Alternatives Considered

1. **Docker-based development environment**
   - Rejected: Slower filesystem performance on macOS
   - More complex volume mounting and networking setup
   - Devbox provides better developer experience with native performance

2. **Make-based build system**
   - Rejected: Platform-specific differences in Make implementations
   - Less sophisticated dependency management than Devbox
   - Devbox provides better reproducibility guarantees

3. **Traditional GOPATH/go mod only**
   - Rejected: No tool version management
   - Inconsistent linter and tool versions across developers
   - Manual environment setup complexity

4. **GitHub Actions only for quality**
   - Rejected: Slow feedback loop for developers
   - Requires push to get quality feedback
   - Local development should catch issues before CI

## Consequences

### Positive Consequences
- **Onboarding simplicity**: New developers run `devbox shell` and have full environment
- **Consistency**: All developers use identical tool versions
- **Quality automation**: Pre-commit hooks prevent quality regressions
- **Fast feedback**: Local quality checks before pushing to CI
- **Cross-platform**: Same workflow on all supported platforms
- **Maintenance**: Tool updates managed centrally in devbox configuration

### Negative Consequences
- **Learning curve**: Developers need to learn Devbox concepts
- **Nix dependency**: Underlying dependency on Nix package manager
- **Initial setup**: First-time Devbox installation requires setup steps
- **Tool lock-in**: Migration away from Devbox requires environment recreation

## Implementation

### Devbox Configuration
```json
{
  "packages": [
    "go_1_21",
    "golangci-lint",
    "gotestsum",
    "git"
  ],
  "scripts": {
    "test": "gotestsum --format testname",
    "test-unit": "gotestsum --format testname -- -short",
    "linter": "golangci-lint run",
    "formatter": "gofumpt -l -w ."
  }
}
```

### Development Workflow
1. **Environment activation**: `devbox shell`
2. **Development iteration**: Write code with immediate linting feedback
3. **Testing**: `devbox run test-unit` for fast feedback
4. **Pre-commit**: Automated formatting, linting, and testing
5. **Commit**: Quality-assured commits only

### Quality Gates
- **Formatter**: `gofumpt` ensures consistent code formatting
- **Linter**: `golangci-lint` catches bugs and style issues
- **Tests**: All tests must pass before commit
- **Build**: Code must compile successfully

### Git Hooks
- **Pre-commit**: Runs formatter, linter, tests, and build
- **Commit validation**: Ensures all quality gates pass
- **Fast execution**: Optimized for quick developer feedback

## Related Decisions

- **ADR-0001**: Streaming Processing - Development tools support streaming architecture testing
- **ADR-0003**: XML Security - Security linting integrated into development workflow