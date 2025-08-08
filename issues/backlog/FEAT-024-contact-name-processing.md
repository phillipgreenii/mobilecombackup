# FEAT-024: Add contact_name to unprocessed property if in contacts.yaml

## Status
- **Completed**: 
- **Priority**: medium

## Overview
When importing SMS/MMS messages, if a contact_name is found in contacts.yaml, it should be added to an "unprocessed" property in the message for manual review and processing. If the contact already exists in contacts.yaml, no action is needed.

## Background
During import, phone numbers in messages can be matched against contacts.yaml to identify known contacts. This feature would help users manually review and update their messages with proper contact names by flagging messages that have matching contacts but haven't been processed yet.

## Requirements
### Functional Requirements
- [ ] During import, check each message's phone number against contacts.yaml
- [ ] If a matching contact is found, add an "unprocessed" property with the contact name
- [ ] Skip if contact_name already exists in the message
- [ ] Support both SMS and MMS messages
- [ ] Handle various phone number formats (normalized comparison)

### Non-Functional Requirements
- [ ] Minimal performance impact on import
- [ ] Clear indication of which messages need manual review
- [ ] Preserve all existing message data

## Design
### Approach
Integrate contact lookup into the import process, adding metadata for manual review without modifying core message content.

### API/Interface
Messages with matching contacts would have:
```xml
<sms address="+15551234567" unprocessed_contact_name="John Doe" .../>
```

### Implementation Notes
- Reuse ContactsReader for efficient lookups
- Normalize phone numbers before comparison
- Add unprocessed_contact_name attribute only if:
  1. Contact exists in contacts.yaml
  2. Message doesn't already have contact_name attribute
- Consider performance with large contact lists

## Tasks
- [ ] Add contact lookup to SMS import process
- [ ] Add contact lookup to MMS import process  
- [ ] Implement phone number normalization for matching
- [ ] Add unprocessed_contact_name attribute when appropriate
- [ ] Write tests for contact matching logic
- [ ] Test performance impact
- [ ] Update documentation

## Testing
### Unit Tests
- Test contact matching with various phone formats
- Test skipping when contact_name already exists
- Test with missing contacts.yaml
- Test with empty contacts

### Integration Tests
- Import with contacts.yaml present
- Verify unprocessed_contact_name added correctly
- Test with large contact lists

### Edge Cases
- International phone numbers
- Phone numbers with/without country codes
- Malformed phone numbers
- Duplicate contacts

## Risks and Mitigations
- **Risk**: Performance degradation with large contact lists
  - **Mitigation**: Use efficient lookup structure from ContactsReader
- **Risk**: Incorrect phone number matching
  - **Mitigation**: Robust normalization and testing

## References
- FEAT-005: Read contacts from repository (ContactsReader implementation)
- FEAT-009: Import SMS functionality
- FEAT-010: Add Import subcommand

## Notes
This feature aids manual review workflow. Users can search for "unprocessed_contact_name" to find messages needing attention. Future enhancement could add a dedicated command to process these messages.