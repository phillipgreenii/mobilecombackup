package importer

import (
	"testing"

	"github.com/phillipgreen/mobilecombackup/pkg/calls"
	"github.com/phillipgreen/mobilecombackup/pkg/contacts"
	"github.com/phillipgreen/mobilecombackup/pkg/sms"
)

func TestContactParsing_MultipleContactNames(t *testing.T) {
	tempDir := t.TempDir()

	// Create contacts manager
	contactsManager := contacts.NewContactsManager(tempDir)

	// Test calls importer with multiple contact names
	t.Run("Calls_MultipleContactNames", func(t *testing.T) {
		callsImporter := &CallsImporter{
			contactsManager: contactsManager,
		}

		// Test single contact name
		call1 := &calls.Call{Number: "15555551234", ContactName: "John Doe"}
		callsImporter.extractContact(call1)

		// Test multiple contact names separated by comma
		call2 := &calls.Call{Number: "15555556789", ContactName: "Jane Smith, Bob Wilson"}
		callsImporter.extractContact(call2)

		// Test multiple contact names with extra spaces
		call3 := &calls.Call{Number: "15555550000", ContactName: "Alice Brown,  Charlie Davis , Eve White"}
		callsImporter.extractContact(call3)

		// Get unprocessed entries
		entries := contactsManager.GetUnprocessedEntries()

		// Verify we have the correct number of entries (grouped by phone number)
		expectedEntries := 3 // 3 different phone numbers
		if len(entries) != expectedEntries {
			t.Errorf("Expected %d unprocessed entries, got %d", expectedEntries, len(entries))
		}

		// Verify the contacts were split correctly
		contactMap := make(map[string][]string)
		for _, entry := range entries {
			contactMap[entry.PhoneNumber] = entry.ContactNames
		}

		// Check single contact
		if names, exists := contactMap["5555551234"]; !exists || len(names) != 1 || names[0] != "John Doe" {
			t.Errorf("Expected single contact 'John Doe' for 5555551234, got %v", names)
		}

		// Check multiple contacts (two contacts for same number)
		if names, exists := contactMap["5555556789"]; !exists || len(names) != 2 {
			t.Errorf("Expected 2 contacts for 5555556789, got %v", names)
		} else {
			expectedNames := []string{"Jane Smith", "Bob Wilson"}
			for _, expected := range expectedNames {
				found := false
				for _, name := range names {
					if name == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected contact '%s' not found in %v", expected, names)
				}
			}
		}

		// Check three contacts with extra spaces trimmed
		if names, exists := contactMap["5555550000"]; !exists || len(names) != 3 {
			t.Errorf("Expected 3 contacts for 5555550000, got %v", names)
		} else {
			expectedNames := []string{"Alice Brown", "Charlie Davis", "Eve White"}
			for _, expected := range expectedNames {
				found := false
				for _, name := range names {
					if name == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected contact '%s' not found in %v", expected, names)
				}
			}
		}
	})

	// Test SMS importer with multiple contact names
	t.Run("SMS_MultipleContactNames", func(t *testing.T) {
		// Clear previous entries
		contactsManager = contacts.NewContactsManager(tempDir)

		smsImporter := &SMSImporter{
			contactsManager: contactsManager,
		}

		// Test SMS with multiple contact names
		smsMsg := sms.SMS{Address: "15555551111", ContactName: "Test User, Another User"}
		smsImporter.extractSMSContact(smsMsg)

		// Test MMS with multiple contact names
		mmsMsg := sms.MMS{Address: "15555552222", ContactName: "MMS User1,MMS User2,MMS User3"}
		smsImporter.extractMMSContacts(mmsMsg)

		// Get unprocessed entries
		entries := contactsManager.GetUnprocessedEntries()

		// Verify we have the correct number of entries
		expectedEntries := 2 // One for SMS number, one for MMS number
		if len(entries) != expectedEntries {
			t.Errorf("Expected %d unprocessed entries, got %d", expectedEntries, len(entries))
		}

		// Create contact map for verification
		contactMap := make(map[string][]string)
		for _, entry := range entries {
			contactMap[entry.PhoneNumber] = entry.ContactNames
		}

		// Check SMS contacts
		if names, exists := contactMap["5555551111"]; !exists || len(names) != 2 {
			t.Errorf("Expected 2 SMS contacts for 5555551111, got %v", names)
		}

		// Check MMS contacts
		if names, exists := contactMap["5555552222"]; !exists || len(names) != 3 {
			t.Errorf("Expected 3 MMS contacts for 5555552222, got %v", names)
		}
	})
}

func TestContactParsing_EmptyAndSingleContacts(t *testing.T) {
	tempDir := t.TempDir()
	contactsManager := contacts.NewContactsManager(tempDir)

	callsImporter := &CallsImporter{
		contactsManager: contactsManager,
	}

	// Test empty contact name
	call1 := &calls.Call{Number: "15555551234", ContactName: ""}
	callsImporter.extractContact(call1)

	// Test empty number
	call2 := &calls.Call{Number: "", ContactName: "John Doe"}
	callsImporter.extractContact(call2)

	// Test comma with empty parts
	call3 := &calls.Call{Number: "15555556789", ContactName: "John Doe, , "}
	callsImporter.extractContact(call3)

	// Get unprocessed entries
	entries := contactsManager.GetUnprocessedEntries()

	// Should only have one entry (for the call with valid number and contact)
	expectedEntries := 1
	if len(entries) != expectedEntries {
		t.Errorf("Expected %d unprocessed entry, got %d", expectedEntries, len(entries))
	}

	if len(entries) > 0 {
		entry := entries[0]
		if entry.PhoneNumber != "5555556789" || len(entry.ContactNames) != 1 || entry.ContactNames[0] != "John Doe" {
			t.Errorf("Expected single contact 'John Doe' for 5555556789, got %v", entry.ContactNames)
		}
	}
}
