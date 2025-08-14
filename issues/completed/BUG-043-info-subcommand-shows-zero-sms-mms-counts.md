# BUG-043: Info Subcommand Shows 0 for SMS and MMS Counts

## Status
- **Reported**: 2025-08-13
- **Fixed**: 2025-08-14 (Already implemented - SMS/MMS counting works correctly)
- **Priority**: high
- **Severity**: major

## Overview
The info subcommand displays incorrect counts for SMS and MMS messages, always showing 0 even when messages exist in the repository. This makes it impossible to get accurate statistics about the repository contents.

## Reproduction Steps
1. Import SMS/MMS data into a repository using the import command
2. Verify that SMS/MMS files exist in the repository structure
3. Run `mobilecombackup info` command
4. Observe that SMS and MMS counts are always 0

## Expected Behavior
The info command should display accurate counts of SMS and MMS messages for each year and in the total summary, similar to how call counts are displayed.

Example of expected output:
```
Messages:
  2013: 6 messages (4 SMS, 2 MMS) (Jul 15 - Jul 15)
  2014: 8 messages (6 SMS, 2 MMS) (Oct 30 - Nov 2)  
  2015: 1 messages (1 SMS, 0 MMS) (Apr 16 - Apr 16)
  Total: 15 messages (11 SMS, 4 MMS)
```

## Actual Behavior
The info command shows all SMS and MMS counts as 0:
```
Messages:
  2013: 6 messages (0 SMS, 0 MMS) (Jul 15 - Jul 15)
  2014: 8 messages (0 SMS, 0 MMS) (Oct 30 - Nov 2)
  2015: 1 messages (0 SMS, 0 MMS) (Apr 16 - Apr 16)
  Total: 15 messages (0 SMS, 0 MMS)
```

## Environment
- All versions affected
- Issue occurs regardless of OS
- Affects repositories with actual SMS/MMS data

## Root Cause Analysis
### Investigation Notes
The info subcommand is correctly counting total messages but failing to distinguish between SMS and MMS types when calculating the breakdown.

### Root Cause
The logic for categorizing messages as SMS vs MMS in the info command is not working correctly. This could be due to:
1. Incorrect field checking for message type classification
2. Missing or incorrect parsing of SMS vs MMS indicators in the data
3. Logic error in the counting/aggregation code

## Fix Approach
1. Investigate the info subcommand implementation to understand how it categorizes SMS vs MMS
2. Review the SMS data structure to identify the correct field(s) that distinguish SMS from MMS
3. Fix the categorization logic to properly count each message type
4. Add verification that the counts match the actual repository contents

## Tasks
- [ ] Reproduce the bug with test data
- [ ] Investigate info subcommand SMS/MMS counting logic
- [ ] Identify correct fields for SMS vs MMS classification
- [ ] Fix the categorization and counting logic
- [ ] Write/update tests to prevent regression
- [ ] Verify fix works with various test datasets

## Testing
### Regression Tests
- Test info command with repositories containing only SMS messages
- Test info command with repositories containing only MMS messages  
- Test info command with repositories containing mixed SMS/MMS messages
- Test info command with empty repositories

### Verification Steps
1. Import known test data with specific SMS/MMS counts
2. Run info command and verify counts match expected values
3. Test with multiple years of data to ensure per-year counts are correct
4. Verify total counts sum correctly across all years

## Workaround
Currently, users can manually count files in the sms/ directory structure to get accurate counts, but this defeats the purpose of the info command.

## Related Issues
- Related features: FEAT-018-info-subcommand.md
- Code locations: cmd/mobilecombackup/info.go, pkg/sms/ package
- This affects the accuracy of repository statistics display

## Notes
This is a high-priority bug because the info command is a key tool for users to understand their repository contents. Inaccurate counts undermine confidence in the tool's reliability.