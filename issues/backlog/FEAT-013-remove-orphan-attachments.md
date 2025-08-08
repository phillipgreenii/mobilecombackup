# FEAT-013: Remove orphan attachments

## Status
- **Completed**: -
- **Priority**: low

## Overview
Violations related to orphan attachments should only be removed explicitly, so a specific argument should be used.

## Background
Not likely, but it is possible for attachments to become orphaned, if that happens, `--remove-orphan-attachments` can be used to remove them.

## Dependencies
- FEAT-001: Repository Validation (core validation infrastructure)
- FEAT-003: Read SMS from Repository (to build reference map)
- FEAT-004: Read Attachments from Repository (to scan existing files)
- FEAT-007: Validate Subcommand (CLI integration)

## Requirements
### Functional Requirements
- [ ] Only operate within `validate` subcommand with explicit `--remove-orphan-attachments` flag
- [ ] Scan entire repository to build complete reference map from all SMS/MMS files
- [ ] Remove both attachment file and corresponding `.metadata.yaml` file for orphans
- [ ] Clean up empty attachment subdirectories after removal
- [ ] Support `--dry-run` flag to show what would be deleted and check permissions
- [ ] Continue processing if individual file removal fails, but log failures
- [ ] Report detailed statistics: scanned attachments, referenced attachments, removed count
- [ ] Can be combined with `--autofix`

### Non-Functional Requirements
- [ ] Handle large datasets efficiently
- [ ] No backup creation - removal is permanent and opt-in only

## Implementation Design

### Algorithm
1. **Build Reference Map**: Scan all SMS files in repository, extract attachment path references
2. **Scan Attachment Directory**: Walk `attachments/` tree, identify all files
3. **Find Orphans**: Compare actual files vs. referenced files
4. **Safe Removal**: Delete orphaned files and their `.metadata.yaml` files
5. **Cleanup**: Remove empty subdirectories

### Error Handling
- Continue processing if individual files fail to read
- Log all removal failures but don't abort operation
- Provide detailed error reporting
- With `--dry-run`, perform permission checks and report potential issues

## Edge Cases
- Handle same attachment referenced multiple times (should not be considered orphan)
- Handle malformed attachment paths in SMS files (log but continue)
- Consider validating attachment hashes before removal
- Handle permission issues gracefully

