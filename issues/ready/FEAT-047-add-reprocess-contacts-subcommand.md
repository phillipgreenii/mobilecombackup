# FEAT-047: Add Reprocess-Contacts Subcommand

## Status
- **Completed**: 
- **Priority**: high

## Overview
Add a new `reprocess-contacts` subcommand that updates all call and SMS entries to use updated contact information from contacts.yaml. This allows users to modify contacts.yaml and then synchronize all historical data to use the updated contact mappings.

## Background
When users manually update contacts.yaml (typically by removing unprocessed numbers and adding proper contacts), the repository becomes inconsistent because existing call and SMS entries still reference old contact information. The reprocess-contacts command will resolve this by updating all historical data to use the current contact mappings.

## Requirements
### Functional Requirements
- [ ] Add new `reprocess-contacts` subcommand to CLI
- [ ] Load and validate repository (suppress contacts.yaml validation issues)
- [ ] Reprocess all call and SMS entries to match current contacts.yaml
- [ ] Regenerate contacts.yaml with updated unprocessed values
- [ ] Validate no duplicate phone numbers in contacts
- [ ] Remove unprocessed phone numbers that exist in contacts (autofix)
- [ ] Write updated repository back to disk
- [ ] Display summary of contact resolution statistics
- [ ] Update import command to show same contact resolution statistics

### Non-Functional Requirements
- [ ] Handle large repositories efficiently
- [ ] Provide clear progress indication for long-running operations
- [ ] Maintain data integrity throughout the process
- [ ] Support repositories with years of historical data

## Design
### Approach
1. **Load repository with relaxed validation** - Skip contacts.yaml specific validations
2. **Reprocess all entries** - Update contact_names in calls and SMS to match contacts.yaml
3. **Regenerate contacts.yaml** - Update unprocessed section based on reprocessed data
4. **Validate final state** - Ensure repository is consistent
5. **Generate statistics** - Show contact resolution breakdown by year and type

### API/Interface
```go
// New subcommand
func NewReprocessContactsCmd() *cobra.Command

// Contact resolution statistics
type ContactResolutionStats struct {
    Known    int // from contacts.yaml, not unprocessed
    Guessed  int // from contacts.yaml, but in unprocessed section  
    Unknown  int // "(Unknown)" or other unknown indicators
}

// Summary by year and type
type ProcessingSummary struct {
    Calls map[int]ContactResolutionStats // by year
    SMS   map[int]ContactResolutionStats // by year
}
```

### Implementation Notes
- Use existing repository loading but suppress specific validation errors
- Iterate through all call and SMS files by year
- Apply current contact resolution logic to update contact_names
- Rebuild contacts.yaml from scratch based on reprocessed data
- Implement duplicate detection and autofix for contacts.yaml

## Tasks
- [ ] Create reprocess-contacts subcommand structure
- [ ] Implement repository loading with relaxed validation
- [ ] Create contact reprocessing logic for calls
- [ ] Create contact reprocessing logic for SMS
- [ ] Implement contacts.yaml regeneration
- [ ] Add validation for duplicate phone numbers in contacts
- [ ] Add autofix for unprocessed numbers that exist in contacts
- [ ] Implement contact resolution statistics tracking
- [ ] Add statistics display functionality
- [ ] Update import command to show contact resolution statistics
- [ ] Write comprehensive tests
- [ ] Update full-test.sh to use new subcommand

## Testing
### Unit Tests
- Test contact reprocessing logic with various scenarios
- Test statistics calculation and display
- Test duplicate detection and autofix logic
- Test validation suppression during loading

### Integration Tests
- Test complete reprocess-contacts workflow with real repository
- Test that import command shows updated statistics
- Test that full-test.sh passes with new subcommand
- Verify repository consistency before and after reprocessing

### Edge Cases
- Empty repositories
- Repositories with only calls or only SMS
- Repositories with complex contact scenarios
- Large repositories with multiple years of data

## Risks and Mitigations
- **Risk**: Data corruption during reprocessing
  - **Mitigation**: Validate repository state before and after processing
- **Risk**: Performance issues with large repositories
  - **Mitigation**: Process data in chunks, provide progress indicators
- **Risk**: Breaking existing workflows
  - **Mitigation**: Thorough testing with full-test.sh integration

## References
- Related features: FEAT-024-contact-name-processing.md, FEAT-010-add-import-subcommand.md
- Code locations: cmd/mobilecombackup/, pkg/contacts/, pkg/calls/, pkg/sms/
- Updates required: full-test.sh script integration

## Notes
### Usage Flow
1. User modifies contacts.yaml (removes unprocessed entries, adds proper contacts)
2. Repository is in invalid state due to inconsistent contact references
3. User runs `mobilecombackup reprocess-contacts`
4. All historical data is updated to use current contact mappings
5. Repository becomes valid and consistent

### Contact Resolution Categories
- **Known contact**: Contact from contacts.yaml not in unprocessed section
- **Guessed contact**: Contact from contacts.yaml that IS in unprocessed section  
- **Unknown contact**: Associated with "(Unknown)" or other unknown indicators

### Validation Rules to Implement
- No duplicate phone numbers across different contacts
- No phone number should exist in both contacts and unprocessed sections
- Repository should be fully consistent after reprocessing

### Statistics Display
The command should show breakdown by year for both calls and SMS:
```
Call Resolution Summary:
  2013: 15 calls (10 known, 3 guessed, 2 unknown)
  2014: 23 calls (18 known, 4 guessed, 1 unknown)
  Total: 38 calls (28 known, 7 guessed, 3 unknown)

SMS Resolution Summary:
  2013: 8 messages (5 known, 2 guessed, 1 unknown)
  2014: 12 messages (9 known, 2 guessed, 1 unknown)  
  Total: 20 messages (14 known, 4 guessed, 2 unknown)
```