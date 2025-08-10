# FEAT-024: Extract Contact Names to Unprocessed Section

## Status
- **Completed**: 2025-01-08
- **Priority**: medium

## Overview
Extract contact names from SMS/MMS XML files during import and add them to an "unprocessed" section in contacts.yaml for manual review. This allows users to build their contact list from imported messages without modifying the message data itself.

## Background
When importing SMS/MMS backups, messages often contain contact_name attributes that represent the sender/recipient names as they appeared in the phone at the time of the backup. These names are valuable for building a comprehensive contacts.yaml file, but should be reviewed before being added to the processed contact list. This feature extracts these names during import and stores them in a special "unprocessed" section of contacts.yaml, allowing users to manually review and promote them to the main contact list.

## Requirements
### Functional Requirements
- [x] Extract contact_name from SMS/MMS messages during import phase
- [x] Store extracted names in ContactsManager's unprocessed section
- [x] Support both SMS (single address) and MMS (multiple addresses) extraction
- [x] Handle phone number normalization for consistent storage
- [x] Process extraction after validation and deduplication
- [x] Create ContactsWriter interface for saving contacts.yaml
- [x] Preserve existing unprocessed entries when loading contacts.yaml
- [x] Support multiple names for the same phone number
- [x] Handle corrupted or missing contacts.yaml gracefully

### Non-Functional Requirements
- [x] Minimal performance impact - load contacts once at start, save once at end
- [x] Clear YAML format for manual review
- [x] Preserve all existing contact data
- [x] Atomic file operations to prevent data loss

## Design
### Approach
Two-phase process:
1. **Extract Phase**: During import, extract contact_name attributes from messages
2. **Write Phase**: After processing complete, save extracted names to contacts.yaml

### API/Interface

#### ContactsWriter Interface
```go
// ContactsWriter handles writing contact information to repository
type ContactsWriter interface {
    // SaveContacts writes the current state to contacts.yaml
    SaveContacts(path string) error
}
```

#### ContactsManager Enhancement
```go
// AddUnprocessedContact adds a contact to the unprocessed section
func (cm *ContactsManager) AddUnprocessedContact(phone, name string) {
    // Normalize phone number
    // Add to unprocessed map
    // Support multiple names per number
}

// GetUnprocessedContacts returns all unprocessed contacts
func (cm *ContactsManager) GetUnprocessedContacts() map[string][]string {
    // Return copy of unprocessed map
}
```

#### contacts.yaml Format
```yaml
# Main contact list
contacts:
  - name: "John Doe"
    phones:
      - "5551234567"
      - "5559876543"
  
  - name: "Jane Smith"
    phones:
      - "5555555555"

# Extracted names pending review
unprocessed:
  - "5551234567: John"        # Different name for existing contact
  - "5551234567: Johnny Doe"  # Another variation
  - "5559999999: Bob Smith"   # New contact
  - "5558888888: Alice Jones" # New contact
```

### Processing Flow
1. Load contacts.yaml at start (including any existing unprocessed section)
2. Process imports normally (validation, deduplication)
3. For each message with contact_name:
   - Extract phone number and contact_name
   - Normalize phone number
   - Add to ContactsManager's unprocessed section
4. After all processing complete, save contacts.yaml once

### Implementation Notes
- Load contacts.yaml once at program start
- Keep unprocessed entries in memory during processing
- Only save contacts.yaml after all imports complete
- Support multiple names for same phone number (all variations saved)
- Use simple "phone: name" format in unprocessed section for easy manual editing
- Preserve existing unprocessed entries when loading
- Handle missing/corrupted contacts.yaml gracefully (start with empty)

## Tasks
- [x] Add unprocessed field to ContactsManager struct
- [x] Implement AddUnprocessedContact method with phone normalization
- [x] Implement GetUnprocessedContacts method
- [x] Update ContactsManager Load to parse unprocessed section
- [x] Implement ContactsWriter interface and SaveContacts method
- [x] Add contact extraction to SMS import processor
- [x] Add contact extraction to MMS import processor (handle multiple addresses)
- [x] Integrate contact saving into main import command
- [x] Write unit tests for unprocessed contact management
- [x] Write integration tests for full import→extract→save flow
- [x] Update documentation with manual review workflow

## Testing
### Unit Tests
- Test AddUnprocessedContact with various phone formats
- Test multiple names for same phone number
- Test loading contacts.yaml with existing unprocessed section
- Test saving contacts.yaml with atomic operations
- Test phone number normalization edge cases
- Test empty/missing contacts.yaml handling

### Integration Tests
- Import SMS with contact_name attributes
- Import MMS with multiple addresses and contact names
- Verify extraction and storage in unprocessed section
- Test full workflow: load → import → extract → save
- Test with large message sets (performance validation)

### Edge Cases
- Empty contact_name attribute
- International phone numbers (+1, +44, etc.)
- Phone numbers with/without country codes
- Malformed phone numbers
- Same name appearing with different phone numbers
- Corrupted contacts.yaml file
- Write permissions issues
- Concurrent access scenarios

## Examples

### Example 1: Simple SMS Import
Input message:
```xml
<sms address="5551234567" contact_name="John Doe" body="Hello" date="1234567890000"/>
```

Result in contacts.yaml:
```yaml
unprocessed:
  - "5551234567: John Doe"
```

### Example 2: MMS with Multiple Recipients
Input message:
```xml
<mms address="5551234567" contact_name="John Doe">
  <addresses>
    <address address="5559876543" contact_name="Jane Smith"/>
    <address address="5555555555" contact_name="Bob Jones"/>
  </addresses>
</mms>
```

Result in contacts.yaml:
```yaml
unprocessed:
  - "5551234567: John Doe"
  - "5559876543: Jane Smith"
  - "5555555555: Bob Jones"
```

### Example 3: Multiple Names for Same Number
After importing multiple messages:
```yaml
unprocessed:
  - "5551234567: John"
  - "5551234567: Johnny"
  - "5551234567: John Doe"
  - "5551234567: J. Doe"
```

## Risks and Mitigations
- **Risk**: Data loss if contacts.yaml write fails
  - **Mitigation**: Use atomic file operations (write to temp, then rename)
- **Risk**: Memory usage with large unprocessed lists
  - **Mitigation**: Efficient in-memory storage, process in batches if needed
- **Risk**: Corrupted contacts.yaml preventing imports
  - **Mitigation**: Graceful handling, backup before write, start fresh if corrupted
- **Risk**: Phone number normalization inconsistencies
  - **Mitigation**: Reuse proven normalization from ContactsReader

## References
- FEAT-005: Read contacts from repository (ContactsReader implementation)
- FEAT-009: Import SMS functionality
- FEAT-010: Add Import subcommand

## Notes
- This feature extracts names FROM messages TO contacts.yaml, not the other way around
- The unprocessed section is for manual review - users decide which names to promote to main contacts
- Names are extracted during import but only saved to contacts.yaml at the end
- Multiple names for the same phone number are preserved for user review
- Simple "phone: name" format makes manual editing easy
- Future enhancement could add a dedicated command to promote unprocessed entries to main contacts

## Manual Review Workflow

After importing messages with contact names, users should follow this workflow to review and organize extracted contacts:

### 1. Review Extracted Contacts
After running import, check `contacts.yaml` for the `unprocessed` section:
```yaml
unprocessed:
  - "5551234567: John Doe"
  - "5551234567: Johnny"
  - "5551234567: J. Doe"
  - "5559876543: Jane Smith"
```

### 2. Identify Duplicate Variations
Look for the same phone number with different name variations:
- Multiple entries for `5551234567` suggest the same person with different name formats
- Decide which name is the canonical/preferred version

### 3. Manual Editing Process
Edit `contacts.yaml` to promote names from `unprocessed` to the main `contacts` section:

**Before:**
```yaml
contacts:
  - name: "Existing Contact"
    numbers: ["5550000000"]

unprocessed:
  - "5551234567: John Doe"
  - "5551234567: Johnny"
  - "5559876543: Jane Smith"
```

**After manual review:**
```yaml
contacts:
  - name: "Existing Contact"
    numbers: ["5550000000"]
  - name: "John Doe"  # Promoted from unprocessed
    numbers: ["5551234567"]
  - name: "Jane Smith"  # Promoted from unprocessed
    numbers: ["5559876543"]

unprocessed:
  # Removed processed entries, keep any that need further review
```

### 4. Best Practices
- **Consolidate variations**: Choose one canonical name per person
- **Verify phone numbers**: Ensure numbers are correct before promoting
- **Keep context**: Use message timestamps to help identify contacts
- **Incremental review**: Process a few contacts at a time to avoid overwhelm
- **Backup first**: Keep a backup of contacts.yaml before major edits

### 5. Common Scenarios

**Scenario 1: Same person, multiple name formats**
```yaml
unprocessed:
  - "5551234567: John Doe"
  - "5551234567: Johnny"
  - "5551234567: J. Doe"
```
**Resolution**: Choose "John Doe" as canonical, remove duplicates

**Scenario 2: Unknown contacts**
```yaml
unprocessed:
  - "5559999999: Unknown"
  - "5558888888: Mom"
```
**Resolution**: Keep "Mom", research or skip "Unknown"

**Scenario 3: Business vs personal**
```yaml
unprocessed:
  - "5551234567: John Doe"
  - "5551234567: Acme Corp"
```
**Resolution**: Determine if John works at Acme, choose appropriate name

### 6. Future Import Behavior
- Subsequent imports will add new unprocessed entries
- Existing processed contacts remain in main section
- New variations of existing numbers will still appear in unprocessed
- Manual review process repeats for new entries only