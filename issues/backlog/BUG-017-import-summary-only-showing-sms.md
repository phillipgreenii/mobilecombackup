# BUG-017: Import summary only shows SMS when importing both SMS and calls

## Status
- **Reported**: 2025-08-08
- **Fixed**: 
- **Priority**: high
- **Severity**: major

## Overview
When importing both SMS and calls data, the import summary only displays the year summary for SMS messages. The calls summary is missing, making it appear that calls were not imported even when they were successfully processed.

## Reproduction Steps
1. Run the import command with both SMS and calls data files
2. Wait for import to complete
3. Check the summary output
4. Observe that only SMS year summary is shown, calls summary is missing

## Expected Behavior
The import summary should show year breakdowns for both SMS and calls when both are imported:
```
Summary:
Calls:
  2014: 10 entries
  2015: 15 entries
SMS:
  2014: 25 messages
  2015: 30 messages
```

## Actual Behavior
Only the SMS summary is displayed:
```
Summary:
SMS:
  2014: 25 messages
  2015: 30 messages
```
The calls summary is completely missing from the output.

## Environment
- Version: Current main branch
- OS: All platforms affected
- Go version: 1.24

## Root Cause Analysis
### Investigation Notes
To be investigated - likely issue in the import command's summary generation or display logic.

### Root Cause
To be determined during investigation.

## Fix Approach
To be determined after root cause analysis. Likely involves fixing the summary display logic to include all imported data types.

## Tasks
- [ ] Reproduce the bug with test data containing both SMS and calls
- [ ] Identify where summary is generated in import command
- [ ] Fix summary generation to include all data types
- [ ] Write/update tests for multi-type imports
- [ ] Verify fix shows correct summaries
- [ ] Update integration tests

## Testing
### Regression Tests
- Test importing only calls - verify calls summary shown
- Test importing only SMS - verify SMS summary shown  
- Test importing both - verify both summaries shown
- Test with multiple years of data

### Verification Steps
1. Import test data with both calls and SMS
2. Verify summary shows both data types
3. Verify counts are accurate for each type and year

## Workaround
Users can run separate imports for calls and SMS to see individual summaries, though this is not ideal.

## Related Issues
- FEAT-010: Add Import subcommand (where summary is generated)
- FEAT-008: Import calls functionality
- FEAT-009: Import SMS functionality

## Notes
This is a user experience issue that makes it unclear what was actually imported.