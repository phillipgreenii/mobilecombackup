# FEAT-007: Add Validate subcommand

## Status
- **Completed**: 2025-08-08
- **Priority**: high

## Overview
The validation added in FEAT-001, will be triggered via the `validate` subcommand.

## Background
Users need a simple interface to validate a repository

## Requirements
### Functional Requirements
- [ ] Parse command-line arguments (repo-root)
- [ ] Display summary with statistics
- [ ] Display violations (if any found)
- [ ] Handle validation errors gracefully
- [ ] Support verbose/quiet output modes
- [ ] Return appropriate exit codes (0 for valid, non-zero for violations)
- [ ] Support JSON output format for CI/CD integration
- [ ] Show progress indication for all validation phases
- [ ] Run concurrent validation checks where possible

### Non-Functional Requirements
- [ ] Clear error messages
- [ ] Progress indication for all validation phases
- [ ] Distinct exit codes for validation violations vs runtime errors

### Usage Examples

#### `repo-root` specifies the repository

```
mobilecombackup validate --repo-root /path/to/repo
```
- repository: `/path/to/repo`

#### `MB_REPO_ROOT` specifies the repository

```
MB_REPO_ROOT=/path/to/repo
mobilecombackup validate
```
- repository: `/path/to/repo`

#### when no repository is specified, use current directory as the repository

```
mobilecombackup validate
```
- repository: `.`

#### Verbose output

```
mobilecombackup validate --repo-root /path/to/repo --verbose
```
- Shows detailed progress for each validation step

#### JSON output for scripting

```
mobilecombackup validate --repo-root /path/to/repo --output=json
```
- Returns validation results in JSON format for CI/CD integration

#### Quiet mode (only show errors)

```
mobilecombackup validate --repo-root /path/to/repo --quiet
```
- Only displays violations, suppresses progress and success messages

## Design

### Output Format
- Default output shows all validation checks being performed
- Display format: simple list of violations found
- Priority: clear visibility of violations when found
- Single progress indicator for entire validation process

### Exit Codes
- `0`: Repository is valid (no violations found)
- `1`: Validation violations detected
- `2`: Runtime error (e.g., invalid repository path, I/O errors)

### Validation Scope
- Always performs full repository validation
- All validation phases from FEAT-001 are executed:
  1. Structure validation
  2. Manifest validation
  3. Checksum validation
  4. Content validation
  5. Consistency validation
- Sequential execution (concurrent validation deferred for future enhancement)

### Progress Indication
- Single progress indicator for entire validation operation
- Updates in real-time during long operations (e.g., checksum validation)

### JSON Output Schema
When `--output=json` is specified:
```json
{
  "valid": true,
  "violations": []
}
```

Or when violations are found:
```json
{
  "valid": false,
  "violations": [
    {
      "type": "missing-timestamp",
      "file": "calls/calls-2015.xml",
      "line": 42,
      "message": "Call entry missing required 'date' field"
    },
    {
      "type": "checksum-mismatch",
      "file": "attachments/ab/ab12345...",
      "expected": "ab12345...",
      "actual": "cd67890...",
      "message": "Attachment file content does not match expected hash"
    }
  ]
}
```

### Error Reporting
- Violations include detailed context (file path, line number when applicable)
- Messages provide clear explanation of what's wrong
- Where possible, include hints for fixing (preparation for FEAT-011 autofix)

## Tasks
- [x] Add validate subcommand to CLI parser in cmd/mobilecombackup
- [x] Create ValidateCommand struct with Run method
- [x] Implement progress reporting interface for validation phases
- [x] Create output formatters (text and JSON)
- [x] Wire up validation logic from FEAT-001
- [x] Handle MB_REPO_ROOT environment variable
- [x] Implement verbose and quiet output modes
- [x] Add proper exit code handling (0/1/2)
- [x] Write unit tests for command parsing
- [x] Write integration tests for validation scenarios
- [x] Update help documentation and usage examples

## Testing

### Unit Tests
- [x] Command parsing with various flag combinations
- [x] Repository path resolution (CLI arg, env var, current dir)
- [x] Output formatter tests (text and JSON formats)
- [x] Exit code handling for different scenarios

### Integration Tests
- [x] Valid repository (expect exit 0)
- [x] Repository with violations (expect exit 1)
- [x] Non-existent repository path (expect exit 2)
- [x] Empty repository (valid, no violations)
- [x] Large repository performance test
- [x] Corrupted data scenarios:
  - Missing required files
  - Invalid XML structure
  - Checksum mismatches
  - Count mismatches in manifests
- [x] Output format verification:
  - Default text output
  - JSON output structure
  - Quiet mode (only errors)
  - Verbose mode (detailed progress)

### Edge Cases
- [x] Repository on read-only filesystem
- [x] Interrupted validation (Ctrl+C handling)
- [x] Symbolic links in repository
- [x] Very long file paths
- [x] Unicode characters in violation messages

## References
- Pre-req: `FEAT-006: Enable CLI`
- Related: `FEAT-001: Repository Validation` (provides validation logic)
- Future: `FEAT-011: Validation Autofix` (will add --autofix flag)

