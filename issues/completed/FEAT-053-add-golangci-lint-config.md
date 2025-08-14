# FEAT-053: Add Golangci-lint Configuration

## Status
- **Priority**: high

## Overview
Add a comprehensive `.golangci.yml` configuration file to ensure consistent code quality checks and catch potential issues early in the development process.

## Background
The project currently uses golangci-lint but lacks a configuration file, which means it's using default settings that may not be optimal for this codebase. A proper configuration will catch more issues and enforce consistent code quality.

## Requirements
### Functional Requirements
- [ ] Create comprehensive .golangci.yml configuration
- [ ] Enable relevant linters for Go best practices
- [ ] Configure appropriate thresholds and settings
- [ ] Exclude test files from certain checks where appropriate

### Non-Functional Requirements
- [ ] Configuration should not produce excessive false positives
- [ ] Build process should complete in reasonable time
- [ ] Linting rules should align with project coding standards

## Design
### Approach
Create a comprehensive configuration that enables important linters while maintaining practical usability.

### Concrete Configuration
**Target Configuration File (.golangci.yml):**
```yaml
linters:
  enable:
    # Code quality and correctness
    - gofmt          # Checks whether code was gofmt-ed
    - govet          # Reports suspicious constructs
    - staticcheck    # Staticcheck is a go vet on steroids
    - gosimple       # Linter for Go source code that specializes in simplifying code
    - errcheck       # Check for unchecked errors
    - ineffassign    # Detects unused assignments
    - typecheck      # Type-checking errors
    
    # Security
    - gosec          # Inspects source code for security problems
    
    # Performance  
    - prealloc       # Finds slice declarations that could potentially be preallocated
    - bodyclose      # Checks whether HTTP response body is closed successfully
    - rowserrcheck   # Checks whether Err of rows is checked successfully
    
    # Style and consistency
    - stylecheck     # Stylecheck is a replacement for golint
    - misspell       # Finds commonly misspelled English words in comments
    - goconst        # Finds repeated strings that could be replaced by a constant
    - dupl           # Tool for code clone detection
    - lll            # Reports long lines
    - nakedret       # Finds naked returns in functions greater than a specified function length
    
    # Code complexity
    - gocyclo        # Computes and checks the cyclomatic complexity of functions
    - funlen         # Tool for detection of long functions
    - nestif         # Reports deeply nested if statements

linters-settings:
  lll:
    line-length: 120
  gocyclo:
    min-complexity: 15
  dupl:
    threshold: 100
  funlen:
    lines: 60
    statements: 40
  gosec:
    severity: medium
  nestif:
    min-complexity: 5

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - dupl
        - gosec
        - funlen
        - gocyclo
    - path: example_test\.go
      linters:
        - lll
```

### Performance Requirements
- Linting must complete within 60 seconds on CI
- Memory usage should not exceed 1GB during linting
- Compatible with golangci-lint version 1.50.0 or higher

## Tasks
### Phase 1: Baseline Assessment and Configuration
- [ ] Run proposed configuration on current codebase to assess violations
- [ ] Document baseline violation count by linter type
- [ ] Create .golangci.yml with specified configuration
- [ ] Test that golangci-lint runs successfully with new config
- [ ] Measure linting performance (time and memory usage)

### Phase 2: Issue Resolution Strategy
- [ ] Categorize violations: critical, important, nice-to-have
- [ ] Fix critical violations (security, correctness issues)
- [ ] Add `//nolint` directives for acceptable violations (temporary)
- [ ] Create issue tracking plan for remaining violations
- [ ] Verify devbox linter command works with new configuration

### Phase 3: Integration and Documentation  
- [ ] Update linting workflow documentation
- [ ] Add linter explanation comments to configuration file
- [ ] Create developer guidelines for common linter violations
- [ ] Test CI integration (if applicable)
- [ ] Schedule follow-up tasks for remaining violations

## Testing
### Configuration Validation
- Test .golangci.yml syntax is valid (`golangci-lint config path`)
- Verify all enabled linters are available in specified version
- Test exclude patterns work correctly for test files
- Validate performance benchmarks (< 60s, < 1GB memory)

### Baseline Testing
- Run `golangci-lint run --config .golangci.yml` on entire codebase
- Document violation counts: total, by package, by linter type
- Test with and without --fast mode for performance comparison
- Verify existing `devbox run linter` command still works

### Violation Resolution Testing
- Test that critical violations are fixed (no errors remain)
- Verify `//nolint` directives work correctly
- Test that new violations are caught in modified code
- Validate configuration works in different development environments

### Example Test Cases
- Test detection of unchecked errors (`errcheck`)
- Test security issue detection (`gosec`) 
- Test cyclomatic complexity limits (`gocyclo`)
- Test line length enforcement (`lll`)
- Test duplicate code detection (`dupl`)

## Risks and Mitigations
- **Risk**: New configuration reveals many existing issues
  - **Mitigation**: Gradually enable linters, fix issues incrementally
- **Risk**: Configuration is too strict and hinders development
  - **Mitigation**: Start with moderate settings, adjust based on feedback

## References
- Source: CODE_IMPROVEMENT_REPORT.md item #5

## Notes
This will significantly improve code quality consistency and help catch issues early. The configuration should be evolved over time based on team feedback and project needs.