# Troubleshooting Guide

This document contains common test failures, lint violations, and their fixes.

## Common Test Failure Patterns and Fixes

### Compilation Errors in Tests

#### `undefined: functionName`
- **Cause**: Missing import or typo in function name
- **Fix**: Add missing import (`import "package/path"`) or correct function name
- **Example**: `undefined: filepath` → add `import "path/filepath"`

#### `cannot use x (type A) as type B`
- **Cause**: Type mismatch in function arguments or return values
- **Fix**: Add explicit type conversion or fix function signature
- **Example**: `string` to `[]byte` → use `[]byte(stringVar)`

#### `declared but not used`
- **Cause**: Variable declared but never referenced
- **Fix**: Remove unused variable or add usage (use `_` for intentionally unused)
- **Example**: `result := someFunc()` not used → `_ = someFunc()` or use result

### Test Logic Errors

#### Wrong expected values
- **Cause**: Test expects incorrect data or counts
- **Fix**: Verify actual test data content and adjust expectations
- **Example**: Test expects 56 entries but file has 12 → update expected count

#### Path resolution issues
- **Cause**: Relative paths incorrect from test directory
- **Fix**: Use correct relative paths or absolute paths with `filepath.Abs()`
- **Example**: `../../../../testdata/` → `../../../testdata/` from cmd/mobilecombackup/cmd/

#### Exit code mismatches
- **Cause**: Unit tests vs integration tests expect different error handling
- **Fix**: Use `os.Exit()` for integration tests, return errors for unit tests
- **Example**: Use `testing.Testing()` to detect test mode

### Test Data and File Issues

#### Missing test data files
- **Cause**: Tests reference non-existent files
- **Fix**: Create required files in `testdata/` or update test paths
- **Example**: Create `testdata/to_process/00/calls-test.xml`

#### Permission errors
- **Cause**: Test files have wrong permissions
- **Fix**: Fix file/directory permissions using `chmod` or `os.Chmod()`
- **Example**: `os.Chmod(dir, 0755)` for directories

#### Empty XML causing rejections
- **Cause**: Test uses empty XML files that get rejected during import
- **Fix**: Use `--no-error-on-rejects` flag or provide realistic test data
- **Example**: Add flag to test command or use actual call/SMS records

### Integration vs Unit Test Patterns

**Unit Tests**: Test command functions directly via `rootCmd.Execute()`
- Expect errors to be returned, not `os.Exit()` calls
- Mock external dependencies
- Focus on logic validation

**Integration Tests**: Test binary execution via `exec.Command`
- Expect specific exit codes via `os.Exit()` calls
- Use real file system and external dependencies
- Focus on end-to-end behavior

## Common Lint Violation Patterns and Fixes

### Error Handling Violations

#### `Error return value is not checked (errcheck)`
- **Cause**: Function returning error not handled
- **Fix**: Add proper error handling or use `_` to explicitly ignore
- **Examples**:
  - `file.Close()` → `_ = file.Close()` or `defer func() { _ = file.Close() }()`
  - `os.Setenv(k, v)` → `_ = os.Setenv(k, v)` in tests

### Unused Code Violations

#### `declared but not used (unused)`
- **Cause**: Variables, functions, or imports not referenced
- **Fix**: Remove unused code or add usage
- **Examples**:
  - Unused import → Remove from import statement
  - Unused variable → Remove declaration or add usage
  - Unused function → Remove or mark as used in tests

### Documentation Violations

#### `should have comment or be unexported (golint)`
- **Cause**: Exported functions/types missing documentation
- **Fix**: Add proper documentation comments
- **Example**: `// ProcessCalls processes call records from XML files`

### Static Analysis Violations

#### `empty branch (staticcheck)`
- **Cause**: Empty if/else branches that do nothing
- **Fix**: Add meaningful code or remove empty branch
- **Example**: Replace empty `if` with comment explaining why no action needed

#### `could use tagged switch (staticcheck)`
- **Cause**: Complex if/else chain that could be a switch
- **Fix**: Refactor to use switch statement for clarity
- **Example**: Convert `if result.Action == "extracted"` chain to switch

### Import and Formatting Issues

#### Import ordering
- **Cause**: Imports not in standard Go order
- **Fix**: Use `goimports` or `devbox run formatter` to fix
- **Standard order**: stdlib, third-party, local packages

#### Formatting inconsistencies
- **Cause**: Code not formatted according to `gofmt` standards
- **Fix**: Run `gofmt` or `devbox run formatter`

## Common Auto-Fix Patterns

### Test Failures
- `undefined: functionName` → Add missing import or fix typo
- `cannot use x (type A) as type B` → Add type conversion
- `declared but not used` → Remove unused variable or add usage
- Missing test data files → Create required files in testdata/

### Lint Violations
- `declared but not used` → Remove unused variables/imports/functions
- `Error return value is not checked` → Add proper error handling
- `should have comment or be unexported` → Add documentation comments
- Formatting issues → Run `gofmt` or use `devbox run formatter`

### Build Failures
- Missing imports → Add required imports
- Syntax errors → Fix code syntax
- Missing dependencies → Run `go mod tidy` or add dependencies

## When to Ask User

Ask for user guidance when:
- Test logic appears incorrect (wrong expected values)
- Multiple valid approaches to fix a lint violation
- Fix would significantly change program behavior
- Unfamiliar error patterns not covered by common fixes
- Repeated failures after multiple fix attempts

## Devbox Environment Issues

- **Problem**: Commands fail outside devbox environment
- **Solution**: Use `devbox run command` to run commands without entering the shell

## Test Data Quirks

- **Count mismatches**: Some files have `count="56"` but only 12 actual entries - these are expected and useful for validation testing
- **Mixed years**: Test data often contains mixed years - adjust tests accordingly
- **555 phone numbers**: Test data uses anonymized 555 phone numbers
- **Binary attachments**: MMS files contain real PNG data in base64
- **Special characters**: SMS bodies contain escaped XML entities (&amp; for &)