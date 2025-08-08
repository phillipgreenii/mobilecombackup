# BUG-022: Only generate rejected directory if there are rejected entries

## Status
- **Reported**: 2025-08-08
- **Fixed**: 
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
The `rejected/` directory should only be created if there are actually rejected entries to write. Clean imports should not create this directory.

## Actual Behavior
An empty `rejected/` directory is always created during import, regardless of whether any entries were rejected.

## Environment
- Version: Current main branch
- OS: All platforms affected
- Go version: 1.24

## Root Cause Analysis
### Investigation Notes
The import command likely creates the directory upfront rather than on-demand when needed.

### Root Cause
To be determined during investigation.

## Fix Approach
Modify the import logic to:
1. Track if any entries are rejected during processing
2. Only create the rejected/ directory when writing the first rejected entry
3. Clean up any empty rejected/ directory at the end if no rejections occurred

## Tasks
- [ ] Identify where rejected/ directory is created
- [ ] Modify to create directory only when needed
- [ ] Ensure thread-safety if concurrent writes
- [ ] Add cleanup for empty directory
- [ ] Write tests for both cases
- [ ] Update documentation

## Testing
### Regression Tests
- Test import with no rejections - no directory created
- Test import with rejections - directory created with files
- Test concurrent rejection writes
- Test directory permissions

### Verification Steps
1. Import clean data
2. Verify no rejected/ directory exists
3. Import data with known issues
4. Verify rejected/ directory exists with rejection files

## Workaround
Users can manually delete empty rejected/ directories after import.

## Related Issues
- FEAT-010: Add Import subcommand (where directory is created)
- Design principle: Avoid creating unnecessary artifacts

## Notes
This is a minor issue but follows the principle of not creating unnecessary files or directories. The fix should be straightforward.