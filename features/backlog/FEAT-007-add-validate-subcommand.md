# FEAT-007: Add Validate subcommand

## Status
- **Completed**: -
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
- Display format: table or tree structure optimized for readability
- Priority: clear visibility of violations when found
- Progress indicators shown for all validation phases

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
- Concurrent execution where possible for performance

### Progress Indication
- Progress indicators shown for all subcommands except help/usage
- Indicates current validation phase and progress within phase
- Updates in real-time during long operations (e.g., checksum validation)

## References
- Pre-req: `FEAT-006: Enable CLI`
- Related: `FEAT-001: Repository Validation` (provides validation logic)
- Future: `FEAT-011: Validation Autofix` (will add --autofix flag)

