# BUG-069: Contacts appear in both processed and unprocessed lists

## Status
- **Reported**: 2025-08-19
- **Fixed**: 2025-08-19
- **Priority**: medium
- **Severity**: major

## Overview
Phone numbers that exist in the main contacts list also appear in the unprocessed contacts list, violating the expected behavior that processed contacts should not appear in unprocessed. This creates data inconsistency and confuses users about which contacts are actually processed.

**Note**: The code contains comments indicating "no normalization per requirement" which suggests this may have been intentional behavior at some point. This requirement should be re-evaluated as it causes data integrity issues.

## Reproduction Steps
1. Run `./full-test.sh` in the project root
2. Navigate to `./tmp/full-test` directory
3. Examine the generated `contacts.yaml` file
4. Observe the duplication issue:
   - Initially: "5555550004" is set as Jim Henson's number in contacts (line 27 of full-test.sh)
   - After import: "5555550004" also appears in unprocessed with "Oscar Wilde" from test data
   - Result: The same normalized number exists in both the contacts and unprocessed sections

## Expected Behavior
When a phone number is associated with a contact in the main `contacts` section, it should not appear in the `unprocessed` section. The unprocessed section should only contain phone numbers that are not yet associated with any known contact.

## Actual Behavior
The same phone number "5555550004" appears in both sections:
```yaml
contacts:
    - name: Jim Henson
      numbers:
        - "5555550004"
unprocessed:
    # ... other entries ...
    - phone_number: "5555550004"
      contact_names:
        - Oscar Wilde
```
This happens because the import process encounters "Oscar Wilde" associated with the same number in the test data but fails to recognize it's already a known contact due to normalization inconsistency.

## Environment
- Version: Current main branch (commit: ef65023)
- OS: Linux 6.12.41
- Go version: As specified in go.mod
- Test environment: full-test.sh integration test

## Root Cause Analysis
### Investigation Notes
The bug occurs in the `AddUnprocessedContacts` method in `pkg/contacts/reader.go`. The method has an inconsistency between how it checks for known contacts versus how it stores unprocessed contacts:

1. **Line 382**: The method calls `IsKnownNumber(address)` to check if a contact should be skipped
2. **Line 387**: If not skipped, it calls `addUnprocessedEntry(address, name)` 
3. **Line 428**: `addUnprocessedEntry` normalizes the phone number using `normalizePhoneNumber(phone)`

### Root Cause
The issue is a normalization inconsistency that appears to be intentional based on code comments:
- `IsKnownNumber` internally calls `GetContactByNumber` which normalizes the phone number before checking
- However, `AddUnprocessedContacts` passes the raw address to `IsKnownNumber` (line 382 comment: "no normalization per requirement")
- The raw address is also passed to `addUnprocessedEntry` (line 387 comment: "no normalization per requirement")
- Inside `addUnprocessedEntry`, the phone number IS normalized (line 428), creating the inconsistency

**Important**: The comments suggest this was intentional, but it causes data integrity issues and should be fixed.

The bug manifests when:
1. A contact is stored with a normalized number (e.g., "5555550004")
2. The import process encounters the same number in a different format 
3. The raw format doesn't match the stored format, so `IsKnownNumber` check passes
4. `addUnprocessedEntry` normalizes the number and adds it to unprocessed, creating a duplicate

## Fix Approach
The fix should ensure consistent normalization in the duplicate detection logic:

1. **Option A (Recommended)**: Remove the "no normalization per requirement" restriction and normalize the address before calling `IsKnownNumber` in `AddUnprocessedContacts`. This ensures consistency with how the rest of the system handles phone number matching.
2. **Option B**: Modify `IsKnownNumber` to handle both raw and normalized versions
3. **Option C**: Change `addUnprocessedEntry` to not normalize if the intent is to keep raw numbers

**Recommendation**: Option A is preferred as it maintains consistency throughout the codebase. The "no normalization" requirement appears to be outdated and causes data integrity issues.

**Implementation for Option A**:
```go
// In AddUnprocessedContacts method, around line 382
normalized := normalizePhoneNumber(address)
if cm.IsKnownNumber(normalized) {
    continue // Skip known contacts
}
```

## Tasks
- [ ] Create unit test `TestContactsManager_AddUnprocessedContacts_NormalizationConsistency` to reproduce the bug
- [ ] Review and remove the "no normalization per requirement" restriction if outdated
- [ ] Implement fix to normalize addresses before checking IsKnownNumber
- [ ] Update existing tests that may rely on the old behavior
- [ ] Verify fix resolves issue without breaking existing functionality
- [ ] Test with various phone formats (+1 prefix, formatting characters, etc.)
- [ ] Verify `reprocess-contacts` command handles normalization correctly
- [ ] Update documentation about phone number normalization behavior
- [ ] Run full-test.sh to confirm the fix

## Testing
### Regression Tests
- Test that contacts in the main section don't appear in unprocessed
- Test with various phone number formats:
  - US numbers with +1 prefix: "+15555550004"
  - US numbers with 1 prefix: "15555550004" 
  - Plain 10-digit numbers: "5555550004"
  - Formatted numbers: "(555) 555-0004", "555-555-0004"
- Test the reprocess-contacts command to ensure it doesn't create duplicates
- Test that normalization is consistent across all contact operations

### Verification Steps
1. Run `./full-test.sh` and verify that "5555550004" only appears once in contacts.yaml (under Jim Henson)
2. Verify Oscar Wilde appears in unprocessed with a different number or not at all if already known
3. Run unit tests that specifically test the `AddUnprocessedContacts` method with normalization scenarios
4. Verify that the `reprocess-contacts` command works correctly with various phone number formats
5. Ensure no regression in existing contact management functionality

## Workaround
Manually edit the `contacts.yaml` file to remove duplicate entries from the unprocessed section. However, this is temporary as re-running import or reprocess-contacts commands may recreate the duplicates.

## Related Issues
- Related features: FEAT-024-contact-name-processing.md, FEAT-005-read-contacts-from-repository.md, FEAT-047-add-reprocess-contacts-subcommand.md
- Code locations: 
  - pkg/contacts/reader.go:361-391 (AddUnprocessedContacts method)
  - pkg/contacts/reader.go:427-469 (addUnprocessedEntry method)
  - pkg/contacts/reader.go:284-287 (IsKnownNumber method)

## Notes
This bug affects data integrity and user experience. While not critical, it should be fixed to ensure consistent behavior and prevent confusion about which contacts are processed versus unprocessed. The full-test.sh integration test serves as a good reproduction case and should continue to be used for verification.

**Architecture Note**: The "no normalization per requirement" comments in the code suggest this behavior may have been intentional at some point. However, this requirement appears outdated as it conflicts with the normalization that happens in `addUnprocessedEntry` and causes data integrity issues. The fix should prioritize consistency over preserving this outdated requirement.