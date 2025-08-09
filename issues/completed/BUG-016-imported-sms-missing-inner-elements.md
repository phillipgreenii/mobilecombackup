# BUG-016: Imported SMS only has outer smses element and none of the inner elements

## Status
- **Reported**: 2025-08-08
- **Fixed**: 2025-08-09
- **Priority**: high
- **Severity**: major

## Overview
When importing SMS data, the resulting XML files only contain the outer `<smses>` element but are missing all the inner SMS and MMS message elements, resulting in empty SMS files after import.

## Reproduction Steps
1. Run the import command with SMS data files
2. Check the generated sms-YYYY.xml files in the repository
3. Observe that files only contain `<smses count="X">` and `</smses>` tags
4. Notice all actual SMS/MMS message elements are missing

## Expected Behavior
The imported SMS files should contain all SMS and MMS message elements from the source files, properly organized by year with the structure:
```xml
<smses count="X">
  <sms .../>
  <mms .../>
  ...
</smses>
```

## Actual Behavior
The SMS files only contain the outer wrapper element:
```xml
<smses count="X">
</smses>
```
All inner message elements are missing.

## Environment
- Version: Current main branch
- OS: All platforms affected
- Go version: 1.24

## Root Cause Analysis
### Investigation Notes
The issue was found to have two root causes:

1. **Incorrect writer path**: In `pkg/importer/sms.go` line 265, the SMS writer was being created with `si.options.RepoRoot` instead of the SMS directory path (`filepath.Join(si.options.RepoRoot, "sms")`).

2. **Type assertion mismatch**: The SMS writer expected pointer types (`*SMS` and `*MMS`) in the type switch, but the messages from the reader were non-pointer values (`SMS` and `MMS`). This caused all messages to be skipped during the type assertion, resulting in empty XML files.

### Root Cause
1. SMS writer was initialized with wrong directory path (repository root instead of SMS subdirectory)
2. Type assertion in writer only handled pointer types but received non-pointer Message interface values

## Fix Approach
1. Fixed the SMS writer initialization to use the correct SMS subdirectory path
2. Updated the type switch in the SMS writer to handle both pointer and non-pointer types for SMS and MMS messages

## Tasks
- [x] Reproduce the bug with test data
- [x] Identify root cause in SMS writer or import flow
- [x] Implement fix to properly write SMS/MMS elements
- [x] Write/update tests to verify all elements are written
- [x] Verify fix with various SMS/MMS data files
- [x] Update integration tests

## Testing
### Regression Tests
- Added `TestSMSImporter_BUG016_MessagesNotWritten` to verify the fix
- Test importing SMS files with both SMS and MMS messages
- Verify all elements are preserved in output
- Test with large files to ensure no elements are dropped

### Verification Steps
1. Import SMS test data ✓
2. Verify output files contain all expected SMS/MMS elements ✓
3. Compare element counts between source and destination ✓

All tests now pass:
- `TestSMSImporter_BUG016_MessagesNotWritten` - specifically tests this bug
- `TestSMSImporter_ImportFile` - verifies SMS import functionality
- All SMS package tests continue to pass

## Workaround
None available - this is a critical data loss issue that requires a fix.

## Related Issues
- FEAT-009: Import SMS functionality (where bug likely exists)
- FEAT-010: Add Import subcommand

## Notes
This is a critical bug as it results in data loss during the import process.