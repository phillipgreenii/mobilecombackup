# FEAT-011: Validation Autofix

## Status
- **Completed**: -
- **Priority**: low

## Overview
Some of the validation violations can be resolved in an automatic fashion.

## Background
Some of the validation rules are recoverable.  For those, the project can fix the violations without any other intervention.

## Requirements
### Functional Requirements
- [ ] Enabled during `validate` subcommand with `--autofix` parameter.
- [ ] Report which violations were fixed
- [ ] Report remaining violations

### Non-Functional Requirements
- [ ] Handle large datasets efficiently

### Auto Fixes

- `calls/`
  - generate if missing
- `sms/`
  - generate if missing
- `attachments/`
  - generate if missing
  - DO NOT REMOVE ophan files
- `contacts.yaml`
  - generate if missing
- `summary.yaml`
  - generate if missing
  - regenerate if any violations in it
- `files.yaml`
  - DO NOT RECALCULATE `sha256` if it exists but does not match
  - generate if missing
  - remove duplicates
  - clean up path names (convert absolute to relative; resolve the paths)
  - add missing files
  - remove files which don't exist
  - update `size_bytes` if not correct
  - generate `sha256` if missing
- `files.yaml.sha256`
  - DO NOT RECALCULATE `sha256` if it exists but does not match (incorrect SHA-256 indicates potential corruption/tampering)
  - generate if missing

## Design

### Implementation Approach
- Each autofix operation should be independent and continue even if others fail
- Use concurrent execution by default for independent operations
- Sequential execution required for dependent operations:
  1. Directory structure must exist before file operations
  2. All data files must be processed before `summary.yaml` can be regenerated
  3. `files.yaml` must be complete before `files.yaml.sha256` can be generated

### Error Handling
- Individual autofix failures should not stop the process
- Collect all errors and report at the end
- Distinguish between "couldn't fix" and "error while fixing"
- Preserve incorrect SHA-256 values to avoid masking corruption/tampering issues

### Backup Strategy
- Before modifying any file, create a backup with timestamp
- Backup location: `{filename}.backup.{timestamp}`
- Use atomic file operations (write to temp, then rename)

### Reporting Format
```
Validation Autofix Report
========================

Fixed Violations:
✓ Created missing directory: calls/
✓ Created missing directory: sms/
✓ Generated missing file: summary.yaml
✓ Updated size_bytes in files.yaml (3 entries)
✓ Added missing files to files.yaml (2 entries)

Remaining Violations:
✗ SHA-256 mismatch: calls/calls-2023.xml (autofix not available for checksum mismatches)
✗ Orphan attachment: attachments/ab/ab12345 (use --remove-orphans flag)

Errors During Autofix:
⚠ Failed to create contacts.yaml: Permission denied

Summary: 6 violations fixed, 2 remaining, 1 error
```

## Usage Examples

### Basic autofix
```bash
mobilecombackup validate --repo-root /path/to/repo --autofix
```

### Dry-run to preview changes
```bash
mobilecombackup validate --repo-root /path/to/repo --autofix --dry-run
# Shows what would be fixed without making changes
```

### Autofix with verbose output
```bash
mobilecombackup validate --repo-root /path/to/repo --autofix --verbose
# Shows detailed progress of each autofix operation
```

## Exit Codes
- `0`: All violations were fixed successfully
- `1`: Some violations remain after autofix
- `2`: Errors occurred during autofix (but process continued)
- `3`: Fatal error prevented autofix from running

## Implementation Notes
- Autofix should be conservative - only fix what is safe to fix automatically
- SHA-256 mismatches are never auto-corrected to avoid hiding data integrity issues
- Operations should be idempotent - running autofix multiple times should be safe
- Consider adding `--force-checksum` flag in future if users need to regenerate checksums

## References
- Pre-req: `FEAT-007: Add Validate subcommand`
- Related: `FEAT-001: Repository Validation` (defines validation rules)
- Future: `FEAT-013: Remove Orphan Attachments` (handles orphan cleanup)

