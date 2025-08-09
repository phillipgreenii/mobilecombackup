# BUG-017: Import summary incomplete - missing calls and attachments

## Status
- **Reported**: 2025-08-08
- **Fixed**: 2025-08-09
- **Priority**: high
- **Severity**: major

## Overview
The import summary is incomplete when importing data. It only displays SMS statistics while missing calls, attachments, and rejection information, making it unclear what was actually imported.

## Reproduction Steps
1. Run the import command with both SMS and calls data files
2. Wait for import to complete
3. Check the summary output
4. Observe that only SMS year summary is shown, calls summary is missing

## Expected Behavior
The import summary should show complete statistics for all imported data types:
```
Import Summary:
Calls:
  2014: 10 entries (8 new, 2 duplicates)
  2015: 15 entries (15 new, 0 duplicates)
  Total: 25 entries (23 new, 2 duplicates)
SMS:
  2014: 25 entries (20 new, 5 duplicates)
  2015: 30 entries (25 new, 5 duplicates)
  Total: 55 entries (45 new, 10 duplicates)
Attachments:
  Total: 12 files (10 new, 2 duplicates)
Rejections:
  Calls: 1 (missing-timestamp: 1)
  SMS: 2 (malformed-attachment: 2)
  Total: 3
```

## Actual Behavior
Only the SMS summary is displayed:
```
Summary:
SMS:
  2014: 25 messages
  2015: 30 messages
```
The calls summary, attachments summary, and rejection information are completely missing from the output.

## Environment
- Version: Current main branch
- OS: All platforms affected
- Go version: 1.24

## Root Cause Analysis
### Investigation Notes
Potential root causes to investigate:
- Summary aggregation logic may be overwriting instead of merging results
- Missing call to display the calls summary after processing
- Order of processing causing earlier summaries to be lost
- Summary data structure may not support multiple entity types

### Root Cause
The ImportSummary struct was aggregating all statistics into a single Total field instead of tracking statistics separately by entity type (calls, SMS, attachments). The displaySummary function was attempting to show separate statistics but the data structure didn't support this separation.

## Fix Approach
1. Verify summary data structure supports all entity types (calls, SMS, attachments, rejections)
2. Ensure summary aggregation merges results instead of overwriting
3. Standardize terminology (use "entries" for both calls and SMS)
4. Add new/duplicate breakdown for each type and year
5. Include attachment processing statistics
6. Add rejection counts with breakdown by rejection reason

## Tasks
- [x] Reproduce the bug with test data containing calls, SMS, and attachments
- [x] Identify where summary is generated in import command
- [x] Check if summary data structure supports multiple entity types
- [x] Fix summary aggregation to merge instead of overwrite
- [x] Standardize terminology to use "entries" for both calls and SMS
- [x] Add new/duplicate tracking for each type and year
- [ ] Add attachment statistics to summary (TODO: requires FEAT-012 implementation)
- [ ] Add rejection counts with reason breakdown (TODO: requires rejection tracking)
- [x] Write unit tests for summary generation with all entity types
- [ ] Update integration tests for complete summary output
- [x] Add test for summary format consistency
- [x] Verify summary persistence between processing phases

## Testing
### Regression Tests
- Test importing only calls - verify calls summary with new/duplicate breakdown
- Test importing only SMS - verify SMS summary with new/duplicate breakdown
- Test importing calls, SMS, and attachments - verify all summaries shown
- Test with multiple years of data
- Test with empty files for one type but not others
- Test with errors/rejections in one type but not others
- Test with only MMS messages (to verify attachment counting)
- Test with large datasets to ensure summary still works
- Test that terminology is consistent ("entries" not "messages")

### Verification Steps
1. Import test data with calls, SMS, and attachments
2. Verify summary shows all data types
3. Verify new/duplicate counts are accurate for each type and year
4. Verify attachment statistics are included
5. Verify rejection counts with reason breakdown
6. Verify totals match sum of yearly counts
7. Run import again and verify duplicate counts increase appropriately

## Workaround
Users can run separate imports for calls and SMS to see individual summaries, though this is not ideal.

## Related Issues
- FEAT-010: Add Import subcommand (where summary is generated)
- FEAT-008: Import calls functionality
- FEAT-009: Import SMS functionality

## Notes
- This is a critical user experience issue that makes it unclear what was actually imported
- The fix should ensure all entity types are displayed even if counts are zero
- Consider using a table format for better readability if output becomes too long
- Rejection reasons should use the standard format: missing-timestamp, malformed-attachment, parse-error
- The summary should be generated after all processing is complete to ensure accuracy