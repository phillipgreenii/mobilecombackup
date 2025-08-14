# FEAT-046: Sort Unprocessed Number List by Phone Number

## Status
- **Completed**: 2025-08-14 (Already implemented - contacts sorted lexicographically)
- **Priority**: medium

## Overview
Sort the unprocessed numbers list in contacts.yaml by phone number to provide consistent, predictable ordering that makes it easier for users to review and manage unprocessed contacts.

## Background
Currently, the unprocessed numbers in contacts.yaml appear in the order they were encountered during processing, which can be random and difficult to navigate. Sorting by phone number will provide a consistent experience and make it easier for users to find specific numbers when manually updating contacts.

## Requirements
### Functional Requirements
- [ ] Sort unprocessed entries by phone number in ascending order
- [ ] Maintain the existing structure and format of contacts.yaml
- [ ] Apply sorting during contact file generation/update operations
- [ ] Preserve all existing data (contact names, phone numbers) while sorting

### Non-Functional Requirements
- [ ] Sorting should be stable and consistent across runs
- [ ] No performance impact on contact processing
- [ ] Maintain backwards compatibility with existing repositories

## Design
### Approach
Implement sorting in the contacts writing/serialization logic to ensure the unprocessed section is always sorted by phone number when the contacts.yaml file is generated or updated.

### API/Interface
```go
// Update contact writing logic to sort unprocessed entries
type UnprocessedContact struct {
    PhoneNumber   string   `yaml:"phone_number"`
    ContactNames  []string `yaml:"contact_names"`
}

// Sort interface implementation
func (contacts UnprocessedContacts) Len() int
func (contacts UnprocessedContacts) Less(i, j int) bool  // compare phone numbers
func (contacts UnprocessedContacts) Swap(i, j int)
```

### Implementation Notes
- Sort should happen just before writing the YAML file
- Use Go's standard sort package for consistent behavior
- Phone number comparison should be lexicographic (string-based)
- Maintain the exact same YAML structure and formatting

## Tasks
- [ ] Identify where contacts.yaml is written/updated
- [ ] Implement sorting logic for unprocessed contacts slice
- [ ] Add sorting call before YAML serialization
- [ ] Write tests to verify sorting behavior
- [ ] Test with various phone number formats (international, domestic, etc.)
- [ ] Update documentation if needed

## Testing
### Unit Tests
- Test sorting with various phone number formats
- Test sorting with empty unprocessed list
- Test sorting with single entry (should not break)
- Test that contact names are preserved during sorting

### Integration Tests
- Test that import operations produce sorted unprocessed lists
- Test that manual contact updates maintain sorting
- Verify existing repositories work correctly after sorting implementation

### Edge Cases
- Empty phone numbers (should sort first or handle gracefully)
- Identical phone numbers with different contact names
- Various phone number formats (with/without country codes, formatting)

## Risks and Mitigations
- **Risk**: Changing order may confuse users familiar with current ordering
  - **Mitigation**: Document the change and provide clear benefit (easier navigation)
- **Risk**: Performance impact on large unprocessed lists
  - **Mitigation**: Sorting is typically O(n log n) and should be negligible for contact lists

## References
- Related features: FEAT-005-read-contacts-from-repository.md
- Code locations: pkg/contacts/ (wherever contacts.yaml is written)
- See NEW_ISSUES item #5 for original requirement

## Notes
This is a user experience improvement that makes the contacts.yaml file more manageable and predictable. The sorting should be applied consistently across all operations that generate or update the contacts file.