# FEAT-005: Read Contacts from Repository

## Status
- **Completed**: -
- **Priority**: high

## Overview
Implement functionality to read contact information from the repository's `contacts.yaml` file. This feature provides access to the mapping between phone numbers and contact names, validates the YAML structure, and supports efficient lookup operations.

## Background
The repository maintains a `contacts.yaml` file that maps phone numbers to contact names. This centralized mapping ensures consistency across all records and allows for contact information updates without modifying the actual call/SMS records. The format supports multiple numbers per contact and handles unknown numbers.

## Requirements
### Functional Requirements
- [ ] Read and parse `contacts.yaml` file
- [ ] Validate `contacts.yaml` against expected schema
- [ ] Support multiple phone numbers per contact
- [ ] Handle special "<unknown>" contact designation
- [ ] Provide efficient lookup by phone number
- [ ] Support reverse lookup (numbers by contact name)
- [ ] Validate YAML structure and format
- [ ] Handle missing contacts.yaml gracefully
- [ ] Detect duplicate phone numbers across contacts

### Non-Functional Requirements
- [ ] Fast lookup performance (O(1) for phone number lookup)
- [ ] Memory efficient for large contact lists
- [ ] Clear error messages for malformed YAML
- [ ] Support for phone number normalization

## Design
### Approach
Create a contacts reader that:
1. Parses the YAML structure
2. Builds efficient lookup maps
3. Handles phone number variations
4. Provides both forward and reverse lookups
5. Validates contact data integrity

### API/Interface
```go
// Contact represents a contact with associated phone numbers
type Contact struct {
    Name    string
    Numbers []string
}

// ContactsReader reads contact information from repository
type ContactsReader interface {
    // LoadContacts loads all contacts from contacts.yaml
    LoadContacts() error
    
    // GetContactByNumber returns contact name for a phone number
    GetContactByNumber(number string) (string, bool)
    
    // GetNumbersByContact returns all numbers for a contact name
    GetNumbersByContact(name string) ([]string, bool)
    
    // GetAllContacts returns all contacts
    GetAllContacts() ([]*Contact, error)
    
    // ContactExists checks if a contact name exists
    ContactExists(name string) bool
    
    // IsKnownNumber checks if a number has a contact
    IsKnownNumber(number string) bool
    
    // GetContactsCount returns total number of contacts
    GetContactsCount() int
}

// ContactsData represents the YAML structure
type ContactsData struct {
    Contacts []Contact `yaml:"contacts"`
}

// ContactsManager provides contact management functionality
type ContactsManager struct {
    repoPath        string
    contacts        map[string]*Contact  // name -> Contact
    numberToName    map[string]string    // number -> name
    loaded          bool
}
```

### Implementation Notes
- Build lookup maps on load for performance
- Normalize phone numbers for consistent lookup
- Handle variations: +1XXXXXXXXXX, XXXXXXXXXX, etc.
- Special handling for "<unknown>" contact
- Consider caching parsed data

## Tasks
- [ ] Define Contact struct and interfaces
- [ ] Create ContactsReader interface
- [ ] Implement YAML parsing logic
- [ ] Build efficient lookup maps
- [ ] Add phone number normalization
- [ ] Implement all query methods
- [ ] Add validation for YAML structure
- [ ] Write comprehensive unit tests
- [ ] Add integration tests
- [ ] Document phone number formats

## Testing
### Unit Tests
- Parse valid contacts.yaml
- Handle empty contacts file
- Test duplicate number detection
- Verify phone number normalization
- Test unknown contact handling
- Lookup performance tests

### Integration Tests
- Load contacts from repository
- Cross-reference with calls/SMS data
- Handle missing contacts.yaml
- Test with large contact lists

### Edge Cases
- Empty contacts list
- Missing contacts.yaml file
- Duplicate phone numbers
- Invalid YAML syntax
- Special characters in names
- International phone formats
- Malformed phone numbers

## Risks and Mitigations
- **Risk**: Inconsistent phone number formats
  - **Mitigation**: Implement robust normalization
- **Risk**: Large contact lists impacting memory
  - **Mitigation**: Efficient data structures, lazy loading
- **Risk**: YAML parsing errors
  - **Mitigation**: Validation and clear error messages

## References
- Related features: FEAT-002 (Calls), FEAT-003 (SMS)
- Specification: See "contacts.yaml" section
- Code location: pkg/contacts/reader.go (to be created)

## Notes
- Contact names in records are informational only
- This is the authoritative source for number->name mapping
- Consider adding contact merge functionality later
- May need export/import features in future