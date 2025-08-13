# FEAT-035: Fix Contacts Processing and Format Issues

## Status
- **Completed**: 2025-08-13
- **Priority**: high
- **Ready for Implementation**: 2025-08-13

## Overview
Fix multiple issues with contacts processing including format, parsing, matching, and sorting problems.

## Background
Several issues were identified with contacts processing:
- contacts.yaml unprocessed format is incorrect (needs phone_number and contact_names properties)
- Unprocessed entries are not sorted by phone number
- SMS address field may contain multiple phone numbers separated by `~` with corresponding contacts separated by `,`
- Numbers that exist in contacts.yaml are being added to unprocessed instead of being matched

## Requirements
### Functional Requirements
- [x] Fix contacts.yaml unprocessed format to use structured phone_number and contact_names properties
- [x] Sort unprocessed entries by raw phone number (lexicographic)
- [x] Parse multiple phone numbers from SMS address field (separated by `~`)
- [x] Parse corresponding contact names (separated by `,`)
- [x] Validate equal counts of phone numbers and contact names (reject if mismatch)
- [x] Match against existing contacts.yaml entries (exclude known numbers from unprocessed)
- [x] Combine contact names for same phone number into single unprocessed entry

### Non-Functional Requirements
- [x] Contact matching should be efficient for large contact lists
- [x] No phone number normalization required (use raw numbers for matching)
- [x] No data migration needed (repositories can be recreated)

## Design
### Unprocessed Format Change

#### Current (Problematic) Format
```yaml
unprocessed:
  - "5551234567: John Doe"
  - "5551234567: Johnny"  # Duplicate number, separate entries
  - "5559876543: Jane Smith"
```

#### New Structured Format
```yaml
unprocessed:
  - phone_number: "5551234567"
    contact_names: ["John Doe", "Johnny"]  # Combined under one number
  - phone_number: "5559876543"
    contact_names: ["Jane Smith"]
```

### Multi-Address Parsing Algorithm

#### Input Example
```
SMS address: "5551234567~5559876543"
Contact names: "John Doe,Jane Smith"
```

#### Parsing Logic
1. Split address by `~` → `["5551234567", "5559876543"]`
2. Split contact names by `,` → `["John Doe", "Jane Smith"]`
3. Validate counts match (2 == 2) ✓
4. Pair: `[("5551234567", "John Doe"), ("5559876543", "Jane Smith")]`

#### Validation Violations
```
# Mismatched counts - REJECT ENTIRE ENTRY
address: "5551234567~5559876543~5551111111"  # 3 numbers
contacts: "John,Jane"                        # 2 names
# Result: Entry rejected, no contacts processed

# Contact name contains comma - REJECT
address: "5551234567"
contacts: "Smith, John Jr."  # Would split to ["Smith", " John Jr."]
# Result: Count mismatch (1 vs 2), entry rejected
```

### Contact Matching Logic

#### Known vs Unknown Numbers
```go
// Phone number is "known" if:
// 1. Exists in contacts.yaml main section (not unprocessed)
// 2. Use raw phone number comparison (no normalization)

func isKnownContact(phoneNumber string, contacts *ContactsData) bool {
    for _, contact := range contacts.Contacts {
        if contact.PhoneNumber == phoneNumber {
            return true  // Found in main contacts
        }
    }
    return false  // Not found, add to unprocessed
}
```

### Data Structures
```go
// New unprocessed entry structure
type UnprocessedEntry struct {
    PhoneNumber  string   `yaml:"phone_number"`
    ContactNames []string `yaml:"contact_names"`
}

// Updated contacts structure
type ContactsData struct {
    Contacts    []Contact           `yaml:"contacts"`
    Unprocessed []UnprocessedEntry  `yaml:"unprocessed"`
}
```

### Implementation Notes
- Sort unprocessed entries by phone_number field (raw string comparison)
- No whitespace trimming around delimiters (keep strict parsing)
- Reject entire SMS entry if address/contact count mismatch
- Combine multiple contact names for same phone number
- No data migration required (repositories recreated from source)

## Tasks
- [x] Update UnprocessedEntry struct with phone_number and contact_names fields
- [x] Implement multi-address parsing with count validation
- [x] Add contact matching against main contacts.yaml entries
- [x] Implement contact name combining for duplicate phone numbers
- [x] Add sorting of unprocessed entries by raw phone number
- [x] Update contacts.yaml reading/writing to use new format
- [x] Add validation rejection for count mismatches
- [x] Write comprehensive parsing tests for edge cases
- [x] Update documentation for new contacts.yaml format

## Testing
### Unit Tests
- Test multi-address parsing with matching counts ("123~456", "John,Jane")
- Test validation rejection for count mismatch ("123~456", "John")
- Test contact name combining for duplicate phone numbers
- Test contact matching against existing contacts.yaml entries
- Test sorting of unprocessed entries by phone number
- Test structured YAML output format

### Integration Tests
- Test SMS import with multi-address creates correct unprocessed entries
- Test known contacts are excluded from unprocessed section
- Test contacts.yaml file written with new structured format
- Test unprocessed entries are sorted in output file

### Edge Cases
- SMS with single address and single contact (no delimiters)
- SMS with empty address or contact fields
- Contact names containing special characters (but not commas)
- Multiple SMS entries with same phone numbers
- SMS with address but no contacts, or contacts but no address
- Very long phone numbers or contact names

## Risks and Mitigations
- **Risk**: Rejecting SMS entries due to validation violations may lose contact data
  - **Mitigation**: Log validation failures for user review; strict validation prevents data corruption
- **Risk**: No normalization may miss contact matches due to format differences
  - **Mitigation**: Document that phone numbers must match exactly; future enhancement can add normalization
- **Risk**: Format change breaks existing tools expecting old unprocessed format
  - **Mitigation**: No migration needed since repositories are recreated; document format change

## References
- Related features: FEAT-005 (Contacts reader implementation)
- Code locations: pkg/contacts/types.go (UnprocessedEntry struct)
- Code locations: pkg/contacts/reader.go (contacts.yaml processing)
- Code locations: pkg/sms/ (SMS address parsing during import)
- Dependencies: YAML parsing for new structured format

## Implementation Notes

### Completed Implementation
All functional requirements have been successfully implemented:

1. **UnprocessedEntry Struct**: Added new struct with `phone_number` and `contact_names` fields to replace the old string format.

2. **Multi-Address Parsing**: Implemented `AddUnprocessedContacts()` method that splits addresses by `~` and contact names by `,`, validates equal counts, and processes each pair.

3. **Contact Matching**: Integrated with existing `IsKnownNumber()` to exclude numbers that already exist in the main contacts section.

4. **Contact Combining**: Uses internal helper `addUnprocessedEntry()` that prevents duplicate name entries for the same phone number.

5. **Sorting**: Implemented lexicographic sorting by raw phone number in `GetUnprocessedEntries()` method.

6. **New Format Support**: Updated both loading and saving logic to handle the new structured YAML format while maintaining backward compatibility for reading old-format files.

7. **Validation**: Returns error if address count doesn't match contact name count, preventing data corruption.

8. **Comprehensive Testing**: Added extensive unit tests covering all edge cases including count mismatches, empty values, known contact filtering, duplicate combining, and sorting.

9. **Documentation**: Added example tests demonstrating all new functionality including multi-address parsing, validation errors, and the new structured format.

### Key Design Decisions
- **Backward Compatibility**: Old-format contacts.yaml files are automatically converted during loading
- **Raw Phone Numbers**: No normalization applied per requirements (uses exact string matching)
- **Error Handling**: Count mismatches cause entire entry rejection rather than partial processing
- **Memory Efficiency**: Uses efficient sorting algorithm and map-based deduplication

### Files Modified
- `pkg/contacts/types.go`: Added UnprocessedEntry struct and new interface methods
- `pkg/contacts/reader.go`: Implemented new methods and updated reading/writing logic
- `pkg/contacts/reader_test.go`: Updated existing tests and added comprehensive new tests
- `pkg/contacts/example_test.go`: Added documentation examples for new functionality

## Notes
Implementation completed successfully with all requirements met and comprehensive test coverage achieved.