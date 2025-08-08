# BUG-016: Imported SMS only has outer smses element and none of the inner elements

## Status
- **Reported**: 2025-08-08
- **Fixed**: 
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
To be investigated - likely issue in the SMS writer implementation or the import process flow.

### Root Cause
To be determined during investigation.

## Fix Approach
To be determined after root cause analysis. Likely involves fixing the SMS writer to properly include message elements.

## Tasks
- [ ] Reproduce the bug with test data
- [ ] Identify root cause in SMS writer or import flow
- [ ] Implement fix to properly write SMS/MMS elements
- [ ] Write/update tests to verify all elements are written
- [ ] Verify fix with various SMS/MMS data files
- [ ] Update integration tests

## Testing
### Regression Tests
- Test importing SMS files with both SMS and MMS messages
- Verify all elements are preserved in output
- Test with large files to ensure no elements are dropped

### Verification Steps
1. Import SMS test data
2. Verify output files contain all expected SMS/MMS elements
3. Compare element counts between source and destination

## Workaround
None available - this is a critical data loss issue that requires a fix.

## Related Issues
- FEAT-009: Import SMS functionality (where bug likely exists)
- FEAT-010: Add Import subcommand

## Notes
This is a critical bug as it results in data loss during the import process.