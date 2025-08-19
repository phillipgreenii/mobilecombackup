# BUG-056: Path validation bypasses across codebase - security audit needed

## Status
- **Reported**: 2025-08-18
- **Fixed**: 2025-08-19
- **Priority**: high
- **Severity**: major

## Overview
Multiple locations in the codebase use `filepath.Join` directly without leveraging the security path validator in `pkg/security/path.go`, potentially allowing path traversal attacks that could access files outside the repository.

## Reproduction Steps
1. Audit all `filepath.Join` usage across codebase (200+ instances found)
2. Identify instances that handle user-controllable or external input
3. Verify if they use `security.PathValidator` or bypass validation
4. Test for potential path traversal vulnerabilities

## Expected Behavior
All file operations handling external input should use the security path validator to prevent directory traversal attacks.

## Actual Behavior
Many `filepath.Join` calls bypass security validation, particularly in:
- Command-line interface handlers
- File import/export operations
- Repository structure management

## Environment
- Version: current main branch
- OS: All supported platforms
- Affects all file operation functionality

## Root Cause Analysis
### Investigation Notes
The codebase has a well-implemented security path validator (`pkg/security/path.go`) but many code paths don't use it consistently:
- Direct `filepath.Join` usage in 200+ locations
- Mix of validated and unvalidated path operations
- Inconsistent security practices across packages

### Root Cause
No enforced policy requiring security validator usage for file operations with external input.

## Fix Approach
1. Audit all `filepath.Join` usage to identify security-sensitive operations
2. Replace direct `filepath.Join` with `PathValidator.JoinAndValidate` where appropriate
3. Add linting rules to prevent future unsafe path operations
4. Create secure path operation guidelines

## Risk Categorization Criteria
Need to define clear criteria for categorizing 545 `filepath.Join` instances:

### High Risk (Must validate)
- User-provided paths from CLI arguments
- Paths from imported XML/YAML files
- Any path that crosses trust boundaries
- Paths used in file write operations

### Medium Risk (Should validate)
- Paths derived from user data (e.g., hash-based paths)
- Paths used in manifest generation
- Year-based directory creation from timestamps

### Low Risk (May skip validation)
- Hard-coded test data paths
- Internal package paths with no external input
- Temporary file creation in controlled environments

## Implementation Priority
1. **Critical** (implement first):
   - cmd/mobilecombackup/cmd/*.go (all CLI handlers)
   - pkg/importer/*.go (handles external data)

2. **High** (implement second):
   - pkg/attachments/*.go (user data storage)
   - pkg/autofix/*.go (file modifications)

3. **Medium** (implement third):
   - pkg/calls/writer.go, pkg/sms/writer.go (year-based paths)
   - pkg/manifest/*.go (file listings)

## Tasks
- [ ] Create audit tracking spreadsheet (545 instances across 88 files)
- [ ] Define systematic audit methodology
- [ ] Complete audit of all filepath.Join usage with risk categorization
- [ ] Replace high-risk instances with security validator
- [ ] Add performance benchmarks (baseline vs with validation)
- [ ] Update development guidelines for secure path handling
- [ ] Add regression tests for path validation
- [ ] Document secure path operation patterns

## Testing
### Regression Tests
- Test path traversal prevention with malicious inputs
- Verify all file operations stay within repository boundaries
- Test symlink resolution security
- Validate error handling for blocked paths

### Verification Steps
1. Run security tests with path traversal payloads
2. Verify no file operations can access parent directories
3. Test with various path traversal techniques (../, ..\, symlinks)
4. Confirm legitimate operations still work correctly

## Workaround
Currently no workaround available - avoid processing untrusted file paths until fixed.

## Related Issues
- Related to BUG-050 (previous path traversal fix)
- Follows security hardening initiative
- Part of comprehensive security audit

## Notes
This is a systematic security hardening effort. Priority should be on user-facing operations (CLI, import/export) before internal utility functions. Some `filepath.Join` usage may be legitimate (test setup, internal operations) and doesn't require security validation.