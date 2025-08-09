# BUG-022: Only generate rejected directory if there are rejected entries

## Status
- **Reported**: 2025-08-08
- **Fixed**: 2025-08-09
- **Priority**: medium
- **Severity**: minor

## Overview
The import command creates a `rejected/` directory even when there are no rejected entries during import, resulting in unnecessary empty directories in the repository.

## Reproduction Steps
1. Initialize a new repository
2. Import clean data files with no errors or rejections
3. Check the repository structure
4. Observe that an empty `rejected/` directory was created

## Expected Behavior
The `rejected/` directory should only be created if there are actually rejected entries to write. Clean imports should not create this directory. When created, it should mirror the repository structure:
```
rejected/
├── calls/
│   └── calls-2024.xml     # Rejected call entries
├── sms/
│   └── sms-2024.xml       # Rejected SMS entries
└── rejection-reasons.yaml  # Detailed rejection reasons
```

## Actual Behavior
An empty `rejected/` directory is always created during import, regardless of whether any entries were rejected.

## Environment
- Version: Current main branch
- OS: All platforms affected
- Go version: 1.24

## Root Cause Analysis
### Investigation Notes
The import command likely creates the directory upfront rather than on-demand when needed. Since import is multi-threaded, proper synchronization is required.

### Root Cause
The XMLRejectionWriter was creating the rejected/ directory structure in its constructor, before knowing if any rejections would actually be written. This resulted in empty directories being created even for clean imports with no rejected entries.

## Fix Approach
Modify the import logic to:
1. Use atomic flag to track if any rejections occurred
2. Use sync.Once for thread-safe directory creation on first rejection
3. Mirror repository structure in rejected/ directory
4. Write rejected entries in importable format (same as original)
5. Write rejection reasons to separate `rejection-reasons.yaml` file
6. Batch all rejections per file and write once when file processing completes
7. Update import summary to include "Rejected entries saved to: rejected/" when applicable

## Tasks
- [x] Identify where rejected/ directory is created in import command
- [x] Implement atomic flag to track rejection occurrence
- [x] Use sync.Once for thread-safe directory creation
- [x] Create mirrored directory structure (rejected/calls/, rejected/sms/)
- [x] Implement batching of rejections per source file
- [x] Write rejected entries in original XML format (importable)
- [ ] Create rejection-reasons.yaml with detailed rejection information
- [x] Update import summary to mention rejected directory when created
- [x] Write unit tests for rejection handling
- [x] Write integration tests for multi-threaded scenarios
- [ ] Test with read-only repository permissions
- [ ] Update documentation about rejected directory structure

## Testing
### Regression Tests
- Test import with no rejections - no directory created
- Test import with rejections - directory created with correct structure
- Test concurrent rejection writes from multiple threads
- Test directory permissions and read-only repository
- Test multiple import runs with existing rejected directory
- Test rejection in different entity types (calls vs SMS)
- Test that rejected files are in importable format
- Test rejection-reasons.yaml contains all rejection details
- Test import summary mentions rejected directory when created

### Verification Steps
1. Import clean data
2. Verify no rejected/ directory exists
3. Import data with known issues (missing timestamps, malformed attachments)
4. Verify rejected/ directory exists with:
   - Mirrored structure (calls/, sms/)
   - Rejected entries in original XML format
   - rejection-reasons.yaml with detailed reasons
5. Re-import rejected files to verify format
6. Verify import summary includes "Rejected entries saved to: rejected/"

## Workaround
Users can manually delete empty rejected/ directories after import.

## Related Issues
- FEAT-010: Add Import subcommand (where directory is created)
- Design principle: Avoid creating unnecessary artifacts

## Notes
- This follows the principle of not creating unnecessary files or directories
- Rejected entries must be in importable format for potential reprocessing
- Thread safety is critical due to multi-threaded import process
- The rejection-reasons.yaml provides detailed diagnostics without modifying the original XML
- Example rejection-reasons.yaml format:
  ```yaml
  rejections:
    - file: "backup-2024-01-15.xml"
      type: "call"
      timestamp: "2024-01-15T10:30:00Z"
      reason: "missing-timestamp"
      details: "Call entry missing required date field"
    - file: "backup-2024-01-15.xml"
      type: "sms"
      timestamp: "2024-01-15T10:31:00Z"
      reason: "malformed-attachment"
      details: "Failed to decode base64 attachment data"
  ```