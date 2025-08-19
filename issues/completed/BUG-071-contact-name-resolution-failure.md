# BUG-071: Contact Name Resolution Failure During Re-Import

## Status
- **Reported**: 2025-08-19
- **Fixed**: 2025-08-19
- **Priority**: high
- **Severity**: major

## Overview
During re-import operations, existing contacts listed in contacts.yaml are not being resolved and applied to imported calls and SMS records. This results in backup contact names being preserved instead of using the known contact names from the repository, causing data inconsistency and poor user experience.

## Reproduction Steps
1. Create a repository: `./mobilecombackup init --repo-root=./tmp/test`
2. Add a contact to contacts.yaml:
   ```yaml
   contacts: 
       - name: "Jim Henson"
         numbers:
           - "5555550004"
   ```
3. Run reprocess-contacts: `./mobilecombackup reprocess-contacts --repo-root=./tmp/test`
4. Import test data: `./mobilecombackup import --repo-root=./tmp/test testdata`
5. Check the imported calls: examine calls files in the repository
6. Check contacts.yaml for unprocessed contacts
7. Observe that phone number "5555550004" appears in unprocessed contacts despite having a known contact

Alternatively, run: `./full-test.sh` and observe the test failure.

## Expected Behavior
- During import, when processing a call/SMS with phone number "5555550004", the system should:
  1. Look up the existing contact "Jim Henson" for this number
  2. Apply "Jim Henson" as the contact name in the imported record
  3. NOT add "5555550004" to unprocessed contacts since it's already known
- The imported calls should show contact_name="Jim Henson" instead of contact_name="Oscar Wilde"
- Phone number "5555550004" should NOT appear in the unprocessed contacts section

## Actual Behavior
- The system only extracts contact names from backup XML files during import
- Existing contacts in contacts.yaml are completely ignored during the import process
- Contact names from backup files (e.g., "Oscar Wilde") are preserved in imported records
- Known phone numbers are still added to unprocessed contacts
- Contact resolution only occurs during contact reprocessing, not during import

## Environment
- Version: current main branch (commit 8dd2c54)
- OS: Linux 6.12.41
- Go version: (as configured in devbox)
- Reproducible via full-test.sh

## Root Cause Analysis
### Investigation Notes
1. **Contact Loading**: The importer correctly loads existing contacts via `contactsManager.LoadContacts()` in `loadExistingRepository()` (pkg/importer/importer.go:413)

2. **Missing Resolution Logic**: During import processing, contact names are only **extracted** from backup XML files, but existing contacts are never **resolved**:
   - In `pkg/importer/calls.go:324`, `extractContact()` is called which only adds to unprocessed contacts
   - There is no corresponding `resolveContact()` call that uses `GetContactByNumber()` to apply existing contact names

3. **Import Process Flow**: 
   - XML contact names (e.g., "Oscar Wilde") are preserved as-is in call.ContactName
   - `extractContact()` adds these to unprocessed contacts regardless of existing known contacts
   - No step applies existing contact names from contacts.yaml to imported records

4. **Interface Available**: The `ContactsManager` interface provides `GetContactByNumber()` method, but it's never used during import

### Root Cause
The import process lacks contact name resolution logic. While existing contacts are loaded, they are never consulted to override backup contact names during the import process.

## Fix Approach
Add contact name resolution logic to the import process:

1. **Add resolveContact function**: Create a new function that uses `GetContactByNumber()` to resolve contact names
2. **Modify extractContact function**: Skip adding to unprocessed contacts if the contact was already known and resolved
3. **Apply during processing**: Call contact resolution during call/SMS processing to override backup contact names BEFORE extraction
4. **Update both importers**: Apply the fix to both calls and SMS importers

Implementation approach:
```go
// In calls.go, add a new function to resolve contacts
func (ci *CallsImporter) resolveContact(call *calls.Call) bool {
    if ci.contactsManager != nil && call.Number != "" {
        if knownName, exists := ci.contactsManager.GetContactByNumber(call.Number); exists {
            call.ContactName = knownName
            return true // Contact was resolved from existing data
        }
    }
    return false // Contact was not found in existing data
}

// Modify extractContact to skip already-known contacts
func (ci *CallsImporter) extractContact(call *calls.Call, wasResolved bool) {
    if ci.contactsManager == nil {
        return
    }
    
    // Skip extraction if contact was already resolved from existing data
    if wasResolved {
        return
    }
    
    // Only extract if both number and contact name are present
    if call.Number != "" && call.ContactName != "" {
        // Existing extraction logic...
    }
}

// In processCalls, resolve BEFORE extraction (around line 323-324)
wasResolved := ci.resolveContact(call)
ci.extractContact(call, wasResolved)

// Apply the same pattern to SMS importer in sms.go
```

## Acceptance Criteria
- [ ] When importing data with phone number "5555550004", if "Jim Henson" exists in contacts.yaml for that number, all imported records MUST show contact_name="Jim Henson" (not "Oscar Wilde")
- [ ] Phone numbers that already exist in contacts.yaml MUST NOT appear in the unprocessed section after import
- [ ] Phone numbers with no match in contacts.yaml MUST still be added to unprocessed contacts
- [ ] The fix MUST work for both calls and SMS/MMS imports
- [ ] Running `./full-test.sh` MUST pass without errors (specifically line 47's check for "Jim Henson")
- [ ] Re-importing the same data multiple times MUST produce identical results (idempotent)

## Tasks
- [ ] Reproduce the bug with full-test.sh
- [ ] Add contact resolution logic to calls importer
- [ ] Add contact resolution logic to SMS importer  
- [ ] Write unit tests for contact resolution during import
- [ ] Write integration tests for the bug scenario
- [ ] Verify fix resolves issue in full-test.sh
- [ ] Update documentation if needed

## Testing
### Regression Tests
- Test that existing contacts are properly resolved during import
- Test that re-import uses existing contacts instead of backup names
- Test that resolved contacts are NOT added to unprocessed list
- Test that unknown contacts are still added to unprocessed list
- Test that contact resolution works with phone number normalization
- Test that both calls and SMS respect existing contact resolution

### Verification Steps
1. Run full-test.sh - should pass all checks
2. Verify imported calls show resolved contact names (Jim Henson vs Oscar Wilde)
3. Verify known phone numbers don't appear in unprocessed contacts
4. Verify contact resolution works for both calls and SMS
5. Verify contact extraction still works for truly unknown contacts

## Workaround
Use the `reprocess-contacts` command after import to extract contact names from repository files, but this doesn't resolve the fundamental issue of backup contact names taking precedence over known contacts.

## Impact on Metrics
When fixed, this should affect the following observable behaviors:
- **Unprocessed contacts count**: Should decrease significantly on re-imports
- **Contact resolution rate**: New metric showing resolved vs extracted contacts
- **Data consistency**: Contact names should be consistent across all years for the same phone number
- **Import idempotency**: Multiple imports should produce identical results

## Related Issues
- Related features: FEAT-024-contact-name-processing
- Related features: FEAT-047-add-reprocess-contacts-subcommand
- Code locations: 
  - pkg/importer/calls.go:324 (extractContact function)
  - pkg/importer/calls.go:299-349 (processCalls function)
  - pkg/importer/sms.go (similar issue in SMS processing)
  - pkg/importer/importer.go:413 (contact loading)

## Notes
This bug affects data quality significantly as users expect their known contacts to be used instead of potentially outdated or incorrect contact names from backup files. The bug is particularly problematic during re-import scenarios where the repository already contains well-maintained contact information.

The fix should ensure that:
1. Contact resolution happens BEFORE contact extraction
2. Known contacts take precedence over backup contact names
3. Already-resolved contacts are NOT added to the unprocessed list
4. Genuinely new/unknown contacts are still extracted for processing
5. Both calls and SMS importers follow the same resolution pattern

### Implementation Notes
- The `extractContact` function signature needs to change to accept a `wasResolved` flag
- Both `pkg/importer/calls.go` and `pkg/importer/sms.go` need parallel changes
- Consider adding a metric to track how many contacts were resolved vs extracted