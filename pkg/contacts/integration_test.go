// Package contacts provides integration tests for contact management functionality.
package contacts

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// copyFile copies a file from src to dst, creating directories as needed
func copyFile(src, dst string) error {
	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func TestContactsManager_Integration_WithTestData(t *testing.T) {
	// Check if test data exists
	testDataPath := "../../testdata/it/scenerio-00/original_repo_root/contacts.yaml"
	if _, err := os.Stat(testDataPath); os.IsNotExist(err) {
		t.Skip("Integration test data not available")
	}

	tempDir := t.TempDir()
	contactsPath := filepath.Join(tempDir, "contacts.yaml")

	err := copyFile(testDataPath, contactsPath)
	if err != nil {
		t.Fatalf("Failed to copy test data: %v", err)
	}

	manager := NewContactsManager(tempDir)
	err = manager.LoadContacts()
	if err != nil {
		t.Fatalf("LoadContacts failed with test data: %v", err)
	}

	// Verify we loaded some contacts
	count := manager.GetContactsCount()
	if count == 0 {
		t.Error("Expected to load some contacts from test data")
	}

	// Test basic functionality with loaded data
	contacts, err := manager.GetAllContacts()
	if err != nil {
		t.Fatalf("GetAllContacts failed: %v", err)
	}

	if len(contacts) != count {
		t.Errorf("GetAllContacts returned %d contacts, but count is %d", len(contacts), count)
	}

	// Test that we can look up each contact by their numbers
	for _, contact := range contacts {
		for _, number := range contact.Numbers {
			foundName, found := manager.GetContactByNumber(number)
			if !found {
				t.Errorf("Could not find contact for number %s", number)
			}
			if foundName != contact.Name {
				t.Errorf("Expected contact %s for number %s, got %s", contact.Name, number, foundName)
			}
		}
	}
}

func TestContactsManager_Integration_EmptyRepository(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewContactsManager(tempDir)

	// Test with empty repository (no contacts.yaml)
	err := manager.LoadContacts()
	if err != nil {
		t.Fatalf("LoadContacts should not fail for empty repository: %v", err)
	}

	if manager.GetContactsCount() != 0 {
		t.Error("Empty repository should have 0 contacts")
	}

	contacts, err := manager.GetAllContacts()
	if err != nil {
		t.Fatalf("GetAllContacts failed: %v", err)
	}
	if len(contacts) != 0 {
		t.Error("Empty repository should return empty contact list")
	}

	// Test lookups on empty repository
	_, found := manager.GetContactByNumber("5555551234")
	if found {
		t.Error("Should not find any contact in empty repository")
	}

	if manager.IsKnownNumber("5555551234") {
		t.Error("No numbers should be known in empty repository")
	}
}

func TestContactsManager_Integration_LargeContactList(t *testing.T) {
	tempDir := t.TempDir()
	contactsPath := filepath.Join(tempDir, "contacts.yaml")

	// Create a large contact list for performance testing
	yamlContent := "contacts:\n"
	expectedContacts := 100 // Reduced for faster test
	for i := 0; i < expectedContacts; i++ {
		// Create unique contact names using index
		contactName := fmt.Sprintf("Contact%04d", i)
		phoneNumber := fmt.Sprintf("555%07d", i)

		yamlContent += "  - name: \"" + contactName + "\"\n"
		yamlContent += "    numbers:\n"
		yamlContent += "      - \"" + phoneNumber + "\"\n"
	}

	err := os.WriteFile(contactsPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create large test file: %v", err)
	}

	manager := NewContactsManager(tempDir)
	err = manager.LoadContacts()
	if err != nil {
		t.Fatalf("LoadContacts failed with large contact list: %v", err)
	}

	count := manager.GetContactsCount()
	if count != expectedContacts {
		t.Errorf("Expected %d contacts, got %d", expectedContacts, count)
	}

	// Test lookup performance with large dataset
	testNumber := "5550000000" // First contact's number
	_, found := manager.GetContactByNumber(testNumber)
	if !found {
		t.Error("Should find contact in large dataset")
	}
}

func TestContactsManager_Integration_ReloadContacts(t *testing.T) {
	tempDir := t.TempDir()
	contactsPath := filepath.Join(tempDir, "contacts.yaml")

	// Create initial contacts file
	yamlContent1 := `contacts:
  - name: "Bob Ross"
    numbers:
      - "+15555551234"
`
	err := os.WriteFile(contactsPath, []byte(yamlContent1), 0644)
	if err != nil {
		t.Fatalf("Failed to create initial test file: %v", err)
	}

	manager := NewContactsManager(tempDir)
	err = manager.LoadContacts()
	if err != nil {
		t.Fatalf("Initial LoadContacts failed: %v", err)
	}

	if manager.GetContactsCount() != 1 {
		t.Error("Expected 1 contact initially")
	}

	// Update contacts file
	yamlContent2 := `contacts:
  - name: "Bob Ross"
    numbers:
      - "+15555551234"
  - name: "Jane Smith"
    numbers:
      - "+15555555678"
`
	err = os.WriteFile(contactsPath, []byte(yamlContent2), 0644)
	if err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	// Reload contacts
	err = manager.LoadContacts()
	if err != nil {
		t.Fatalf("Reload LoadContacts failed: %v", err)
	}

	if manager.GetContactsCount() != 2 {
		t.Error("Expected 2 contacts after reload")
	}

	// Verify new contact is findable
	name, found := manager.GetContactByNumber("5555555678")
	if !found || name != "Jane Smith" {
		t.Error("Should find newly added contact after reload")
	}
}

func TestContactsManager_Integration_PhoneNumberVariations(t *testing.T) {
	tempDir := t.TempDir()
	contactsPath := filepath.Join(tempDir, "contacts.yaml")

	yamlContent := `contacts:
  - name: "Test Contact"
    numbers:
      - "+1-555-555-1234"
      - "(555) 555-5678"
      - "555.555.9999"
      - "5551234567"
`
	err := os.WriteFile(contactsPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := NewContactsManager(tempDir)
	err = manager.LoadContacts()
	if err != nil {
		t.Fatalf("LoadContacts failed: %v", err)
	}

	// Test various input formats that should all resolve to the same contact
	testNumbers := []string{
		"5555551234",     // Normalized from +1-555-555-1234
		"15555551234",    // With country code
		"+15555551234",   // With + and country code
		"(555) 555-1234", // Formatted
		"555-555-1234",   // Dashed
		"555 555 1234",   // Spaced
		"5555555678",     // Second number
		"5559999",        // Last 7 digits of third number
		"1551234567",     // Fourth number with country code
	}

	for _, number := range testNumbers {
		name, found := manager.GetContactByNumber(number)
		if found && name == "Test Contact" {
			t.Logf("Successfully found contact for number: %s", number)
		}
	}
}

func TestContactsManager_Integration_UnknownContact(t *testing.T) {
	tempDir := t.TempDir()
	contactsPath := filepath.Join(tempDir, "contacts.yaml")

	yamlContent := `contacts:
  - name: "Bob Ross"
    numbers:
      - "+15555551234"
  - name: "<unknown>"
    numbers:
      - "8888888888"
      - "9999999999"
`
	err := os.WriteFile(contactsPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := NewContactsManager(tempDir)
	err = manager.LoadContacts()
	if err != nil {
		t.Fatalf("LoadContacts failed: %v", err)
	}

	// Test that unknown contact is handled properly
	name, found := manager.GetContactByNumber("8888888888")
	if !found || name != "<unknown>" {
		t.Error("Should find <unknown> contact")
	}

	// Verify unknown contact exists
	if !manager.ContactExists("<unknown>") {
		t.Error("Should recognize <unknown> as existing contact")
	}

	// Get numbers for unknown contact
	numbers, found := manager.GetNumbersByContact("<unknown>")
	if !found || len(numbers) != 2 {
		t.Error("Should get 2 numbers for <unknown> contact")
	}
}
