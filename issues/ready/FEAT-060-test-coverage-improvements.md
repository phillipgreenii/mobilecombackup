# FEAT-060: Test Coverage Improvements

## Status
- **Priority**: low

## Overview
Improve test coverage across packages with comprehensive testing strategies including table-driven tests, benchmarks, and fuzzing tests for critical components.

## Background
While the project has good test coverage in many areas, some packages lack comprehensive testing, and there are opportunities to add more sophisticated testing approaches like fuzzing and benchmarking.

## Requirements
### Functional Requirements
- [ ] Increase test coverage to 85%+ across all packages
- [ ] Add table-driven tests for edge cases
- [ ] Implement benchmark tests for performance-critical paths
- [ ] Add fuzz testing for parsers and input validation
- [ ] Create integration tests for complex workflows

### Non-Functional Requirements
- [ ] Tests should run efficiently in CI/CD pipeline
- [ ] Fuzz tests should be comprehensive but time-bounded
- [ ] Benchmarks should track performance over time
- [ ] Tests should be maintainable and well-documented

## Design
### Approach
Systematically analyze test coverage and add missing tests using modern Go testing approaches.

### Implementation Notes
- Use Go's built-in testing tools (testing, benchmarking, fuzzing)
- Focus on edge cases and error conditions
- Add performance regression detection through benchmarks
- Include property-based testing where appropriate

## Tasks
### Phase 1: Coverage Analysis and Prioritization
- [ ] Run `go test -coverprofile=coverage.out ./...` to establish baseline
- [ ] Identify packages below 80% coverage (likely: pkg/autofix, pkg/validation)
- [ ] Document specific functions lacking test coverage in priority packages
- [ ] Focus on critical paths: XML parsing, validation logic, error handling

### Phase 2: Targeted Test Implementation  
- [ ] Add fuzz tests for `pkg/calls/xml_reader.go` and `pkg/sms/xml_reader.go`
- [ ] Create table-driven tests for validation rules in `pkg/validation`
- [ ] Add benchmark tests for `pkg/attachments` storage operations
- [ ] Implement error path testing for `pkg/importer` rejection handling
- [ ] Add integration tests for full import workflow with various file sizes

### Phase 3: Infrastructure and Standards
- [ ] Set up coverage reporting in devbox scripts (`devbox run coverage`)
- [ ] Create testing guidelines document with patterns and standards
- [ ] Add coverage gates: minimum 85% for new packages, 80% for existing
- [ ] Document fuzzing strategy for security-critical parsers

## Specific Coverage Targets
**Packages needing attention (based on complexity):**
- `pkg/validation`: Complex validation logic, multiple error paths
- `pkg/autofix`: File manipulation, error recovery scenarios  
- `pkg/sms/xml_reader.go`: Complex MMS parsing with attachments
- `pkg/importer`: Integration logic with multiple failure modes

**Critical functions to test:**
- XML parsing error handling in streaming operations
- Attachment extraction and hash validation
- Coalescer deduplication logic
- Validation violation detection and reporting

## Testing
### Unit Tests
- Comprehensive coverage of all public APIs
- Edge case and boundary condition testing
- Error path testing

### Integration Tests
- Full import workflows with various input types
- Validation workflows with different repository states
- Cross-package integration scenarios

### Edge Cases
- Malformed input handling
- Resource exhaustion scenarios
- Concurrent access patterns

## References
- Source: CODE_IMPROVEMENT_REPORT.md item #12

## Notes
Focus on areas that will provide the most value - parsers, validation logic, and core business logic should be prioritized over simple utility functions.