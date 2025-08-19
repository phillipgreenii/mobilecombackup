# BUG-070: The unprocessed section of contacts.yaml should be ordered by phone_number

## Status
- **Reported**: 2025-08-19
- **Fixed**: 2025-08-19
- **Priority**: medium
- **Severity**: minor

## Overview
The unprocessed section of contacts.yaml is not consistently ordered by phone_number. While the `GetUnprocessedEntries()` method correctly returns entries sorted by phone number, the `SaveContacts()` method writes entries in random order due to Go's map iteration behavior, causing inconsistent file output and poor user experience when reviewing unprocessed contacts.

## Reproduction Steps
1. Run the test script to import backup data with unprocessed contacts:
   ```bash
   ./full-test.sh
   ```
2. Examine the generated contacts.yaml file:
   ```bash
   cat tmp/full-test/contacts.yaml | grep -A 20 "unprocessed:"
   ```
3. Run the import command again to regenerate the file:
   ```bash
   ./mobilecombackup import --repo-root="tmp/full-test" testdata
   ```
4. Compare the unprocessed section ordering - phone numbers will appear in different order
5. Alternatively, run this test multiple times and observe inconsistent ordering:
   ```bash
   for i in {1..3}; do 
     rm -rf tmp/test-ordering
     ./mobilecombackup init --repo-root="tmp/test-ordering"
     ./mobilecombackup import --repo-root="tmp/test-ordering" testdata
     echo "Run $i:"
     grep -A 10 "unprocessed:" tmp/test-ordering/contacts.yaml | head -5
   done
   ```

## Expected Behavior
The unprocessed section in contacts.yaml should always be ordered by phone_number lexicographically, providing consistent output across multiple runs and making it easier for users to review and manage unprocessed contacts.

## Actual Behavior
The unprocessed entries appear in random order in the saved contacts.yaml file, making it difficult to review and potentially causing unnecessary file changes when the same data is saved multiple times.

## Environment
- Version: current main branch
- OS: Linux 6.12.41
- Go version: 1.21+ (as configured in devbox.json)
- File: pkg/contacts/reader.go
- Test data: testdata/ directory with sample XML backup files

## Root Cause Analysis
### Investigation Notes
Analysis of `/home/phillipgreenii/Projects/mobilecombackup/pkg/contacts/reader.go` reveals the discrepancy:

1. **GetUnprocessedEntries() method (lines 394-424)**: Correctly sorts phone numbers lexicographically and contact names alphabetically
2. **SaveContacts() method (lines 489-499)**: Uses `for phone, names := range cm.unprocessed` which has random iteration order in Go
3. **GetUnprocessedEntries() creates sorted output**: Uses `sort.Strings(phoneNumbers)` on line 404
4. **SaveContacts() ignores sorting**: Directly iterates over the internal map without utilizing the sorted logic

### Root Cause
The `SaveContacts()` method in `/home/phillipgreenii/Projects/mobilecombackup/pkg/contacts/reader.go` (lines 489-499) iterates directly over the internal `cm.unprocessed` map using `for phone, names := range cm.unprocessed`, which produces random iteration order. This bypasses the sorting logic already implemented in `GetUnprocessedEntries()`.

## Fix Approach
Modify the `SaveContacts()` method to use the sorted entries from `GetUnprocessedEntries()` instead of directly iterating over the internal unprocessed map. This will ensure consistent ordering in the saved YAML file that matches the ordering provided by the getter method.

### Implementation Details
Replace lines 489-499 in `SaveContacts()` method:
```go
// Current problematic code:
for phone, names := range cm.unprocessed {
    if len(names) > 0 {
        entry := UnprocessedEntry{
            PhoneNumber:  phone,
            ContactNames: make([]string, len(names)),
        }
        copy(entry.ContactNames, names)
        contactsData.Unprocessed = append(contactsData.Unprocessed, entry)
    }
}
```

With:
```go
// Fixed code using sorted entries:
contactsData.Unprocessed = cm.GetUnprocessedEntries()
```

This simple change:
1. Leverages existing sorting logic from `GetUnprocessedEntries()`
2. Ensures phone numbers are lexicographically ordered
3. Ensures contact names within each phone are alphabetically sorted
4. Maintains consistency across multiple save operations
5. Reduces code duplication

## Tasks
- [ ] Reproduce the bug using full-test.sh and verify inconsistent ordering
- [ ] Create failing test case in reader_test.go demonstrating the issue
- [ ] Implement fix in SaveContacts method (lines 489-499) using GetUnprocessedEntries()
- [ ] Add test `TestContactsManager_SaveContacts_ConsistentOrdering` to verify deterministic output
- [ ] Update existing test `TestContactsManager_SaveContacts_NewFile` to check ordering
- [ ] Verify fix with multiple runs of full-test.sh
- [ ] Run full test suite: `devbox run test`
- [ ] Ensure no performance regression with benchmark if needed

## Testing
### New Test Cases Required
```go
// Test to add in reader_test.go
func TestContactsManager_SaveContacts_ConsistentOrdering(t *testing.T) {
    // Test that multiple saves produce identical output
    // even when entries are added in different order
}

func TestContactsManager_SaveContacts_OrderingMatchesGetter(t *testing.T) {
    // Verify SaveContacts output order matches GetUnprocessedEntries
}
```

### Regression Tests
- Test that multiple calls to SaveContacts produce byte-identical file output
- Test that saved unprocessed entries maintain lexicographic phone number ordering
- Test that contact names within each phone number remain alphabetically sorted
- Test edge cases: international formats (+1), different lengths, special characters
- Test with large number of unprocessed entries (100+) for performance

### Verification Commands
```bash
# Unit tests
devbox run test-unit pkg/contacts

# Integration test
./full-test.sh

# Manual verification
diff <(./mobilecombackup import --repo-root="tmp/test1" testdata && cat tmp/test1/contacts.yaml) \
     <(./mobilecombackup import --repo-root="tmp/test2" testdata && cat tmp/test2/contacts.yaml)
# Should show no differences
```

## Workaround
Currently, users can rely on the `GetUnprocessedEntries()` method to view sorted unprocessed contacts, but saved files will have inconsistent ordering until this is fixed.

## Related Issues
- Related features: FEAT-046-sort-unprocessed-number-list-by-phone-number (already implemented the sorting logic)
- Related features: FEAT-047-add-reprocess-contacts-subcommand (uses the affected functionality)
- Code locations: /home/phillipgreenii/Projects/mobilecombackup/pkg/contacts/reader.go:489-499 (SaveContacts method)
- Code locations: /home/phillipgreenii/Projects/mobilecombackup/pkg/contacts/reader.go:394-424 (GetUnprocessedEntries method with correct sorting)

## Notes
This issue demonstrates a common Go programming pitfall where map iteration order is non-deterministic. The sorting logic already exists and works correctly in GetUnprocessedEntries(), so the fix is straightforward by reusing that existing functionality.

The issue affects:
- **Data consistency**: Same data produces different file outputs
- **User experience**: Difficult to review unprocessed contacts
- **Version control**: Unnecessary diffs when contacts.yaml is committed
- **Testing**: Hard to verify expected output in tests

While the reading/parsing logic can handle unordered entries correctly, consistent ordering is important for maintainability and user experience.

### Additional Context
- The existing test suite (`TestContactsManager_GetUnprocessedEntries_Sorting`) verifies that GetUnprocessedEntries returns sorted data
- However, no test currently verifies that SaveContacts maintains this ordering in the file
- This gap in test coverage allowed the bug to remain undetected