# BUG-044: Contacts Parsing Multiple Contacts Not Split Correctly

## Status
- **Reported**: 2025-08-13
- **Fixed**: 
- **Priority**: high
- **Severity**: major

## Overview
The contact parsing logic is incorrectly handling multiple contacts, creating concatenated phone numbers and grouped contact names instead of creating separate contact entries for each person.

## Reproduction Steps
1. Import backup data that contains multiple contacts with different phone numbers
2. Check the generated contacts.yaml file
3. Observe that multiple contacts are incorrectly grouped together

## Expected Behavior
Each contact should be parsed into separate entries in contacts.yaml:

```yaml
contacts:
  - phone_number: "15555550001"
    contact_names:
      - Ted Turner
  - phone_number: "15555550003"
    contact_names:
      - Nancy Edgar
  - phone_number: "15555550005"
    contact_names:
      - Antonio Scott
  - phone_number: "15555550004"
    contact_names:
      - Max Hall
```

## Actual Behavior
Multiple contacts are incorrectly concatenated into single entries:

```yaml
contacts:
  - phone_number: "15555550001155555500031555555000515555550004"
    contact_names:
      - Ted Turner, Nancy Edgar, Antonio Scott, Max Hall
```

## Environment
- All versions affected
- Occurs when processing backup files with multiple contact entries
- Affects contact resolution for calls and SMS

## Root Cause Analysis
### Investigation Notes
The contact parsing logic appears to be concatenating data from multiple contact records instead of processing them individually.

### Root Cause
The issue is likely in the contact parsing/import logic where:
1. Phone numbers from multiple contacts are being concatenated instead of processed separately
2. Contact names are being joined with commas instead of creating separate contact entries
3. The parsing loop may not be properly separating individual contact records

## Fix Approach
1. Investigate the contact import/parsing code to understand how contacts are extracted from backup files
2. Identify where the concatenation is happening (likely in XML parsing or data processing)
3. Fix the logic to properly separate and create individual contact entries
4. Ensure each contact gets its own phone number and name entry
5. Update contact processing to handle the corrected format

## Tasks
- [ ] Reproduce the bug with test contact data
- [ ] Investigate contact parsing logic in pkg/contacts or pkg/importer
- [ ] Identify where phone number concatenation occurs
- [ ] Identify where contact name grouping occurs  
- [ ] Fix parsing logic to create separate contact entries
- [ ] Write/update tests with multiple contact scenarios
- [ ] Verify fix works with existing repositories

## Testing
### Regression Tests
- Test contact parsing with single contact entries (should still work)
- Test contact parsing with multiple contact entries (main fix)
- Test contact parsing with duplicate phone numbers
- Test contact parsing with empty/missing contact names

### Verification Steps
1. Create test data with known multiple contacts
2. Run import command and verify contacts.yaml structure
3. Ensure each contact has separate phone_number and contact_names entries
4. Verify that contact resolution works correctly for calls and SMS
5. Test backwards compatibility with existing repositories

## Workaround
Users can manually edit the contacts.yaml file to separate the concatenated entries, but this requires manual parsing of the combined data.

## Related Issues
- Related features: FEAT-005-read-contacts-from-repository.md, FEAT-024-contact-name-processing.md
- Code locations: pkg/contacts/, pkg/importer/
- This affects contact resolution accuracy for all calls and SMS

## Notes
This is a critical bug that affects the core functionality of contact management. Without proper contact separation, users cannot get accurate contact resolution for their communications data.