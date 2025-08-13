# BUG-045: Unknown Contact Handling Logic is Incorrect

## Status
- **Reported**: 2025-08-13
- **Fixed**: 
- **Priority**: high
- **Severity**: major

## Overview
The logic for handling unknown contacts in the unprocessed section of contacts.yaml is incorrect. When a known contact is found for a phone number that was previously marked as "(Unknown)", the unknown entry should be replaced rather than kept alongside the known contact.

## Reproduction Steps
1. Import data where a phone number initially has no contact information (gets marked as "(Unknown)")
2. Later, import or process data that provides a real contact name for the same phone number
3. Check the unprocessed section in contacts.yaml
4. Observe that both "(Unknown)" and the real contact name are present

## Expected Behavior
When a real contact name is found for a phone number:

```yaml
unprocessed:
  - phone_number: "5555550000"
    contact_names:
      - Jane Smith
```

The "(Unknown)" entry should be completely replaced by the known contact name.

## Actual Behavior
Both unknown and known contact names are kept:

```yaml
unprocessed:
  - phone_number: "5555550000"
    contact_names:
      - (Unknown)
      - Jane Smith
```

## Environment
- All versions affected
- Occurs during contact processing and import operations
- Affects contact resolution accuracy

## Root Cause Analysis
### Investigation Notes
The contact processing logic is treating "(Unknown)" as a regular contact name instead of a placeholder that should be replaced when real contact information becomes available.

### Root Cause
The issue is in the contact merging/updating logic where:
1. "(Unknown)" is being treated as a valid contact name rather than a placeholder
2. The logic doesn't check if "(Unknown)" should be replaced when a real contact is found
3. The system adds new contacts without removing placeholder entries

## Fix Approach
1. Implement logic to detect unknown contact placeholders (currently "(Unknown)", but designed to be extensible)
2. When a real contact name is found for a phone number, remove any unknown placeholders
3. Only add "(Unknown)" if no other contact names exist for that phone number
4. Ensure the logic is extensible for other unknown contact indicators in the future

## Tasks
- [ ] Identify where unknown contact handling occurs in the codebase
- [ ] Create a function to detect unknown contact placeholders
- [ ] Implement replacement logic that removes unknowns when real contacts are found
- [ ] Ensure "(Unknown)" is only added when no other contacts exist
- [ ] Make the unknown contact detection extensible for future placeholder types
- [ ] Write comprehensive tests for unknown contact scenarios
- [ ] Update existing repositories to clean up duplicate unknown entries

## Testing
### Regression Tests
- Test contact processing with initially unknown contacts that later get real names
- Test that "(Unknown)" is not added when real contacts already exist
- Test that multiple real contact names work correctly
- Test that unknown contacts are properly replaced, not just added to

### Verification Steps
1. Create test scenario with unknown contact that later gets a real name
2. Verify that only the real contact name appears in the final result
3. Test that "(Unknown)" is added only when no other contact exists
4. Verify backwards compatibility with existing contact data

## Workaround
Users can manually edit contacts.yaml to remove "(Unknown)" entries when real contact names are present.

## Related Issues
- Related features: FEAT-024-contact-name-processing.md, FEAT-005-read-contacts-from-repository.md
- Code locations: pkg/contacts/, contact processing logic
- This affects the accuracy and cleanliness of contact data

## Notes
This issue is part of a broader contact management improvement. The logic should be designed to handle future unknown contact indicators beyond just "(Unknown)" to make the system more flexible and maintainable.

### Implementation Design Notes
The unknown contact detection should be implemented as:
```go
func isUnknownContact(contactName string) bool {
    unknownIndicators := []string{"(Unknown)"} // extensible for future indicators
    for _, indicator := range unknownIndicators {
        if contactName == indicator {
            return true
        }
    }
    return false
}
```