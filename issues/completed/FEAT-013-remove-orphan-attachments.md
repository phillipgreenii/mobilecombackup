# FEAT-013: Remove orphan attachments

## Status
- **Completed**: 2025-08-12
- **Priority**: low

## Overview
Add functionality to remove orphaned attachment files that are no longer referenced by any SMS/MMS messages. This operates as an explicit cleanup operation within the validate command.

## Background
Attachments can become orphaned through manual file operations or interrupted processing. When this happens, the `--remove-orphan-attachments` flag can be used to safely remove unreferenced files and reclaim disk space.

## Dependencies
- FEAT-001: Repository Validation (core validation infrastructure)
- FEAT-003: Read SMS from Repository (to build reference map)
- FEAT-004: Read Attachments from Repository (to scan existing files)
- FEAT-007: Validate Subcommand (CLI integration)

## Requirements
### Functional Requirements
- [ ] Only operate within `validate` subcommand with explicit `--remove-orphan-attachments` flag
- [ ] Can be combined with `--autofix` flag (both operations run independently)
- [ ] Scan entire repository to build complete reference map from all SMS/MMS files
- [ ] Remove orphaned attachment files (no metadata files exist in current system)
- [ ] Clean up empty attachment subdirectories after removal
- [ ] Support `--dry-run` flag to show what would be deleted and check permissions
- [ ] Continue processing if individual file removal fails, but log failures
- [ ] Report removal results as subsection within validation output format
- [ ] No confirmation prompt required - explicit flag serves as confirmation

### Non-Functional Requirements
- [ ] Process at least 10,000 attachments per second during scanning
- [ ] Memory usage O(n) where n is number of unique attachment references
- [ ] Use streaming for SMS/MMS scanning to avoid loading all messages in memory
- [ ] No backup creation - removal is permanent and opt-in only

## Implementation Design

### Algorithm
1. **Build Reference Map**: Scan all SMS files in repository, extract attachment path references using existing SMSReader functionality
2. **Scan Attachment Directory**: Walk `attachments/` tree, identify all files using AttachmentManager
3. **Find Orphans**: Compare actual files vs. referenced files to identify unreferenced attachments
4. **Safe Removal**: Delete orphaned attachment files (no metadata files to remove)
5. **Cleanup**: Remove empty subdirectories after file removal

### CLI Integration
- **Flag**: `--remove-orphan-attachments` (no short form)
- **Compatibility**: Works with `--autofix`, `--dry-run`, `--quiet`, `--json` flags
- **Usage**: `mobilecombackup validate --remove-orphan-attachments [--dry-run] [--quiet] [--json]`

### Error Handling
- **Permission Errors**: Log and continue processing, include in removal failure count
- **File In Use**: Log failure and continue, no retry attempts
- **Individual File Failures**: Continue processing remaining files, collect all failures for reporting
- **Operation Failures**: Handle same as validation operation failures (appropriate exit codes)
- **Dry-Run Mode**: Perform permission checks and report potential removal issues

### Output Format Integration
Extends existing validation output with orphan attachment removal subsection:

**Normal Mode**:
```
Repository validation completed:
  Calls: 12,345 processed, 0 violations found
  SMS: 23,456 processed, 0 violations found
  Attachments: 3,456 processed, 0 violations found
  
Orphan attachment removal:
  Attachments scanned: 3,456
  Orphans found: 23
  Orphans removed: 22 (44.4 MB freed)
  Removal failures: 1
    - attachments/ab/ab12345...: Permission denied
```

**JSON Mode**:
```json
{
  "validation": { ... },
  "orphan_removal": {
    "attachments_scanned": 3456,
    "orphans_found": 23,
    "orphans_removed": 22,
    "bytes_freed": 46545920,
    "removal_failures": 1,
    "failed_removals": [
      {
        "path": "attachments/ab/ab12345...",
        "error": "Permission denied"
      }
    ]
  }
}
```

## Tasks
- [x] Add `--remove-orphan-attachments` flag to validate command
- [x] Implement orphan attachment detection algorithm
- [x] Add safe file removal with error collection
- [x] Implement empty directory cleanup
- [x] Integrate with existing validation output formats
- [x] Add dry-run mode support
- [x] Write comprehensive tests

## Testing
### Unit Tests
- Reference map building from SMS files
- Orphan detection algorithm correctness
- File removal with various error conditions
- Output formatting in normal and JSON modes

### Integration Tests
- End-to-end orphan removal with real repository structure
- Combination with `--autofix` and other flags
- Large dataset performance testing
- Permission error handling

### Edge Cases
- Same attachment referenced multiple times (should not be considered orphan)
- Malformed attachment paths in SMS files (log but continue processing)
- Symbolic links and special files in attachment directory
- Files being written during orphan removal operation
- Empty attachment directory (no attachments to scan)
- All attachments are referenced (no orphans found)

## Risks and Mitigations
- **Risk**: Accidental removal of legitimate attachments due to parsing errors
  - **Mitigation**: Comprehensive testing with real data, conservative error handling
- **Risk**: Performance issues with large attachment directories
  - **Mitigation**: Streaming implementation, progress reporting
- **Risk**: Permission issues preventing removal
  - **Mitigation**: Graceful error handling, detailed failure reporting

## References
- FEAT-001: Repository Validation (validation output format)
- FEAT-003: Read SMS from Repository (attachment reference extraction)
- FEAT-004: Read Attachments from Repository (attachment scanning)
- FEAT-007: Validate Subcommand (CLI flag integration)
- FEAT-011: Autofix command (compatible operation design)

## Notes
- This feature provides explicit cleanup functionality for edge cases
- No automatic removal - requires explicit user action via flag
- Removal statistics contribute to overall validation violation counts
- Exit codes follow existing validation patterns (no special codes needed)