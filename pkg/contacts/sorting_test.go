package contacts

import (
	"testing"
)

func TestContactsManager_ImprovedSorting(t *testing.T) {
	manager := NewContactsManager("")

	// Add entries in random order to test sorting
	manager.AddUnprocessedContact("5559876543", "Zebra Contact")
	manager.AddUnprocessedContact("5551234567", "Beta Contact")
	manager.AddUnprocessedContact("5559876543", "Alpha Contact")
	manager.AddUnprocessedContact("5555555555", "Middle Contact")
	manager.AddUnprocessedContact("5551234567", "Charlie Contact")

	entries := manager.GetUnprocessedEntries()

	// Verify correct number of entries
	if len(entries) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(entries))
	}

	// Verify phone numbers are sorted lexicographically
	expectedPhones := []string{"5551234567", "5555555555", "5559876543"}
	for i, entry := range entries {
		if entry.PhoneNumber != expectedPhones[i] {
			t.Errorf("Expected phone number %s at position %d, got %s", expectedPhones[i], i, entry.PhoneNumber)
		}
	}

	// Verify contact names within each phone number are sorted alphabetically
	// First phone number should have "Beta Contact", "Charlie Contact"
	if len(entries[0].ContactNames) != 2 {
		t.Errorf("Expected 2 contacts for first phone number, got %d", len(entries[0].ContactNames))
	} else {
		expectedNames := []string{"Beta Contact", "Charlie Contact"}
		for i, name := range entries[0].ContactNames {
			if name != expectedNames[i] {
				t.Errorf("Expected contact name %s at position %d, got %s", expectedNames[i], i, name)
			}
		}
	}

	// Second phone number should have "Middle Contact"
	if len(entries[1].ContactNames) != 1 || entries[1].ContactNames[0] != "Middle Contact" {
		t.Errorf("Expected 'Middle Contact' for second phone number, got %v", entries[1].ContactNames)
	}

	// Third phone number should have "Alpha Contact", "Zebra Contact"
	if len(entries[2].ContactNames) != 2 {
		t.Errorf("Expected 2 contacts for third phone number, got %d", len(entries[2].ContactNames))
	} else {
		expectedNames := []string{"Alpha Contact", "Zebra Contact"}
		for i, name := range entries[2].ContactNames {
			if name != expectedNames[i] {
				t.Errorf("Expected contact name %s at position %d, got %s", expectedNames[i], i, name)
			}
		}
	}
}

func TestContactsManager_SortingEdgeCases(t *testing.T) {
	manager := NewContactsManager("")

	// Test with numbers that might sort differently as strings vs numbers
	manager.AddUnprocessedContact("555123", "Short Number")
	manager.AddUnprocessedContact("5551234567", "Long Number")
	manager.AddUnprocessedContact("555", "Very Short")
	manager.AddUnprocessedContact("55512345678", "Very Long")

	entries := manager.GetUnprocessedEntries()

	// Verify entries are sorted lexicographically (as strings, not numbers)
	expectedOrder := []string{"555", "555123", "5551234567", "55512345678"}
	if len(entries) != len(expectedOrder) {
		t.Fatalf("Expected %d entries, got %d", len(expectedOrder), len(entries))
	}

	for i, entry := range entries {
		if entry.PhoneNumber != expectedOrder[i] {
			t.Errorf("Expected phone number %s at position %d, got %s", expectedOrder[i], i, entry.PhoneNumber)
		}
	}
}
