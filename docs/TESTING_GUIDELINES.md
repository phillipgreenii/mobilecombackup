# Testing Guidelines

This document outlines testing standards and practices for the mobilecombackup project.

## Testing Standards

### Coverage Requirements
- **New packages**: Minimum 85% test coverage
- **Existing packages**: Minimum 80% test coverage
- **Critical paths**: 90%+ coverage (XML parsing, validation, security)

### Test Organization
```
package_name/
├── types.go              # Core types and interfaces
├── implementation.go     # Main implementation
├── implementation_test.go # Unit tests
├── integration_test.go   # Integration tests (if needed)
├── example_test.go       # Usage examples and documentation
├── bench_test.go         # Benchmark tests (for performance-critical code)
└── fuzz_test.go          # Fuzz tests (for parsers and input validation)
```

## Test Types

### 1. Unit Tests (`*_test.go`)
**Purpose**: Test individual functions and methods in isolation.

**Standards**:
- Test both success and failure paths
- Use table-driven tests for multiple scenarios
- Include edge cases and boundary conditions
- Mock external dependencies

**Example**:
```go
func TestParseTimestamp(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected time.Time
        wantErr  bool
    }{
        {"valid timestamp", "1640995200000", time.Unix(1640995200, 0), false},
        {"empty string", "", time.Time{}, true},
        {"null value", "null", time.Time{}, false},
        {"invalid format", "invalid", time.Time{}, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := ParseTimestamp(tt.input)
            if tt.wantErr && err == nil {
                t.Error("Expected error but got none")
            }
            if !tt.wantErr && err != nil {
                t.Errorf("Unexpected error: %v", err)
            }
            if !result.Equal(tt.expected) {
                t.Errorf("Expected %v, got %v", tt.expected, result)
            }
        })
    }
}
```

### 2. Integration Tests (`*_integration_test.go`)
**Purpose**: Test component interactions and end-to-end workflows.

**Standards**:
- Use temporary directories for file operations
- Test real file I/O and system interactions
- Verify component integration
- Clean up resources with defer statements

### 3. Example Tests (`example_test.go`)
**Purpose**: Provide usage documentation and ensure API usability.

**Standards**:
- Include // Output: comments for verification
- Show realistic usage scenarios
- Keep examples simple and focused
- Use Example prefix for function names

### 4. Benchmark Tests (`*_bench_test.go`)
**Purpose**: Measure and track performance characteristics.

**Standards**:
- Use `b.SetBytes()` for throughput measurements
- Test different input sizes
- Include concurrent benchmarks for shared resources
- Reset timer after setup: `b.ResetTimer()`

**Example**:
```go
func BenchmarkAttachmentStorage(b *testing.B) {
    sizes := []int{1024, 10*1024, 100*1024, 1024*1024}
    
    for _, size := range sizes {
        b.Run(fmt.Sprintf("%dB", size), func(b *testing.B) {
            data := make([]byte, size)
            b.SetBytes(int64(size))
            b.ResetTimer()
            
            for i := 0; i < b.N; i++ {
                // Benchmark code here
            }
        })
    }
}
```

### 5. Fuzz Tests (`*_fuzz_test.go`)
**Purpose**: Discover edge cases and security vulnerabilities through random input testing.

**Standards**:
- Include for all parsers and input validation
- Seed with known good and bad inputs
- Never allow panics - always fail gracefully
- Set reasonable limits on resource usage

**Example**:
```go
func FuzzXMLParser(f *testing.F) {
    // Seed with valid inputs
    f.Add(`<valid>xml</valid>`)
    f.Add(`<edge case="value"/>`)
    
    f.Fuzz(func(t *testing.T, input string) {
        defer func() {
            if r := recover(); r != nil {
                t.Errorf("Parser panicked: %v", r)
            }
        }()
        
        result, err := ParseXML(input)
        // Error is acceptable, panic is not
        _ = result
        _ = err
    })
}
```

## Test Execution

### Running Tests
```bash
# Run all tests
devbox run tests

# Run with coverage
devbox run coverage

# Run coverage summary
devbox run coverage-summary

# Run benchmarks
go test -bench=. ./...

# Run fuzz tests (time-limited)
go test -fuzz=FuzzXMLParser -fuzztime=30s ./pkg/sms
```

### Coverage Analysis
```bash
# Generate detailed coverage report
devbox run coverage

# View coverage in browser
open coverage.html

# Check specific package coverage
go test -cover ./pkg/validation
```

## Best Practices

### Test Data Management
- Use `testdata/` directories for test files
- Keep test data minimal and focused
- Use temporary files/directories for file operations
- Clean up resources with `defer` statements

### Error Testing
- Test all error paths explicitly
- Use `errors.Is()` and `errors.As()` for error comparison
- Verify error messages contain useful information
- Test error recovery and cleanup

### Performance Testing
- Benchmark critical paths (parsing, I/O, validation)
- Track performance regressions over time
- Test with realistic data sizes
- Include memory allocation benchmarks

### Security Testing
- Fuzz test all input parsers
- Test boundary conditions and resource limits
- Verify input validation and sanitization
- Test for common vulnerabilities (path traversal, etc.)

### Concurrent Testing
- Test thread-safe operations with `go test -race`
- Use `testing.T.Parallel()` for independent tests
- Test resource contention scenarios
- Verify proper cleanup in concurrent scenarios

## Patterns to Avoid

❌ **Don't**:
- Skip testing error paths
- Use real files in unit tests
- Leave resource leaks in tests
- Test implementation details instead of behavior
- Use hardcoded paths or timing dependencies

✅ **Do**:
- Test behavior and contracts
- Use dependency injection for testability
- Mock external dependencies
- Test edge cases and error conditions
- Keep tests fast and reliable

## Coverage Monitoring

Current coverage targets by package:
- `pkg/coalescer`: 100% ✅
- `pkg/errors`: 100% ✅  
- `pkg/types`: 100% ✅
- `pkg/validation`: 88.9% ✅
- `pkg/contacts`: 87.7% ✅
- `pkg/security`: 91.7% ✅
- `pkg/logging`: 84.9% ✅
- `pkg/config`: 82.7% ✅
- `pkg/manifest`: 78.7% ⚠️
- `pkg/autofix`: 77.8% ⚠️
- `pkg/importer`: 74.0% ⚠️
- `pkg/metrics`: 70.4% ⚠️
- `pkg/calls`: 68.6% ⚠️
- `pkg/sms`: 65.7% ⚠️

Priority for improvement: `pkg/sms`, `pkg/calls`, `pkg/metrics`, `pkg/importer`

## Running the Full Test Suite

```bash
# Complete testing workflow
devbox run formatter  # Format code
devbox run tests      # Run all tests
devbox run linter     # Check code quality
devbox run coverage   # Generate coverage report
devbox run build-cli  # Verify builds
```

This ensures code quality, test coverage, and build integrity before deployment.