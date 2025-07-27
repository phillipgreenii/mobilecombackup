package contacts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestContactsManager_LoadContacts_ValidFile(t *testing.T) {
	tempDir := t.TempDir()
	contactsPath := filepath.Join(tempDir, "contacts.yaml")

	// Create a valid contacts.yaml file
	yamlContent := `contacts:
  - name: "Bob Ross"
    numbers:
      - "+15555551234"
      - "5555551234"
  - name: "Jane Smith"
    numbers:
      - "+15555555678"
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

	if !manager.loaded {
		t.Error("Manager should be marked as loaded")
	}

	if manager.GetContactsCount() != 3 {
		t.Errorf("Expected 3 contacts, got %d", manager.GetContactsCount())
	}
}

func TestContactsManager_LoadContacts_MissingFile(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewContactsManager(tempDir)

	err := manager.LoadContacts()
	if err != nil {
		t.Fatalf("LoadContacts should not fail for missing file: %v", err)
	}

	if !manager.loaded {
		t.Error("Manager should be marked as loaded even when file is missing")
	}

	if manager.GetContactsCount() != 0 {
		t.Errorf("Expected 0 contacts, got %d", manager.GetContactsCount())
	}
}

func TestContactsManager_LoadContacts_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	contactsPath := filepath.Join(tempDir, "contacts.yaml")

	// Create invalid YAML
	yamlContent := `contacts:
  - name: "Bob Ross
    numbers:
      - "+15555551234"
`
	err := os.WriteFile(contactsPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := NewContactsManager(tempDir)
	err = manager.LoadContacts()
	if err == nil {
		t.Error("LoadContacts should fail for invalid YAML")
	}
}

func TestContactsManager_LoadContacts_EmptyContactName(t *testing.T) {
	tempDir := t.TempDir()
	contactsPath := filepath.Join(tempDir, "contacts.yaml")

	yamlContent := `contacts:
  - name: ""
    numbers:
      - "+15555551234"
`
	err := os.WriteFile(contactsPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := NewContactsManager(tempDir)
	err = manager.LoadContacts()
	if err == nil {
		t.Error("LoadContacts should fail for empty contact name")
	}
}

func TestContactsManager_LoadContacts_DuplicateNumbers(t *testing.T) {
	tempDir := t.TempDir()
	contactsPath := filepath.Join(tempDir, "contacts.yaml")

	yamlContent := `contacts:
  - name: "Bob Ross"
    numbers:
      - "+15555551234"
  - name: "Jane Smith"
    numbers:
      - "5555551234"  # Same number as Bob, normalized
`
	err := os.WriteFile(contactsPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := NewContactsManager(tempDir)
	err = manager.LoadContacts()
	if err == nil {
		t.Error("LoadContacts should fail for duplicate phone numbers")
	}
}

func TestContactsManager_GetContactByNumber(t *testing.T) {
	tempDir := t.TempDir()
	contactsPath := filepath.Join(tempDir, "contacts.yaml")

	yamlContent := `contacts:
  - name: "Bob Ross"
    numbers:
      - "+15555551234"
      - "5555555678"
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

	tests := []struct {
		number   string
		expected string
		found    bool
	}{
		{"+15555551234", "Bob Ross", true},
		{"5555551234", "Bob Ross", true},        // Normalized lookup
		{"15555551234", "Bob Ross", true},       // Without +
		{"(555) 555-1234", "Bob Ross", true},    // Formatted number
		{"555-555-5678", "Bob Ross", true},      // Second number
		{"+15555559999", "", false},             // Not found
	}

	for _, test := range tests {
		name, found := manager.GetContactByNumber(test.number)
		if found != test.found {
			t.Errorf("GetContactByNumber(%s): expected found=%v, got %v", test.number, test.found, found)
		}
		if name != test.expected {
			t.Errorf("GetContactByNumber(%s): expected %s, got %s", test.number, test.expected, name)
		}
	}
}

func TestContactsManager_GetNumbersByContact(t *testing.T) {
	tempDir := t.TempDir()
	contactsPath := filepath.Join(tempDir, "contacts.yaml")

	yamlContent := `contacts:
  - name: "Bob Ross"
    numbers:
      - "+15555551234"
      - "5555555678"
  - name: "Jane Smith"
    numbers:
      - "+15555559999"
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

	// Test existing contact
	numbers, found := manager.GetNumbersByContact("Bob Ross")
	if !found {
		t.Error("GetNumbersByContact should find Bob Ross")
	}
	if len(numbers) != 2 {
		t.Errorf("Expected 2 numbers for Bob Ross, got %d", len(numbers))
	}

	// Test non-existing contact
	_, found = manager.GetNumbersByContact("Unknown Person")
	if found {
		t.Error("GetNumbersByContact should not find Unknown Person")
	}
}

func TestContactsManager_ContactExists(t *testing.T) {
	tempDir := t.TempDir()
	contactsPath := filepath.Join(tempDir, "contacts.yaml")

	yamlContent := `contacts:
  - name: "Bob Ross"
    numbers:
      - "+15555551234"
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

	if !manager.ContactExists("Bob Ross") {
		t.Error("ContactExists should return true for Bob Ross")
	}
	if manager.ContactExists("Unknown Person") {
		t.Error("ContactExists should return false for Unknown Person")
	}
}

func TestContactsManager_IsKnownNumber(t *testing.T) {
	tempDir := t.TempDir()
	contactsPath := filepath.Join(tempDir, "contacts.yaml")

	yamlContent := `contacts:
  - name: "Bob Ross"
    numbers:
      - "+15555551234"
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

	if !manager.IsKnownNumber("5555551234") {
		t.Error("IsKnownNumber should return true for known number")
	}
	if manager.IsKnownNumber("9999999999") {
		t.Error("IsKnownNumber should return false for unknown number")
	}
}

func TestContactsManager_GetAllContacts(t *testing.T) {
	tempDir := t.TempDir()
	contactsPath := filepath.Join(tempDir, "contacts.yaml")

	yamlContent := `contacts:
  - name: "Bob Ross"
    numbers:
      - "+15555551234"
  - name: "Jane Smith"
    numbers:
      - "+15555559999"
`
	err := os.WriteFile(contactsPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := NewContactsManager(tempDir)
	contacts, err := manager.GetAllContacts()
	if err != nil {
		t.Fatalf("GetAllContacts failed: %v", err)
	}

	if len(contacts) != 2 {
		t.Errorf("Expected 2 contacts, got %d", len(contacts))
	}

	// Verify we get copies, not references
	if len(contacts) > 0 {
		contacts[0].Name = "Modified"
		// Original should be unchanged
		originalName, _ := manager.GetContactByNumber("+15555551234")
		if originalName == "Modified" {
			t.Error("GetAllContacts should return copies, not references")
		}
	}
}

func TestNormalizePhoneNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"+15555551234", "5555551234"},
		{"15555551234", "5555551234"},
		{"5555551234", "5555551234"},
		{"(555) 555-1234", "5555551234"},
		{"555-555-1234", "5555551234"},
		{"555.555.1234", "5555551234"},
		{"555 555 1234", "5555551234"},
		{"+1-555-555-1234", "5555551234"},
		{"12345678901", "2345678901"}, // 11 digits starting with 1
		{"22345678901", "22345678901"}, // 11 digits not starting with 1
	}

	for _, test := range tests {
		result := normalizePhoneNumber(test.input)
		if result != test.expected {
			t.Errorf("normalizePhoneNumber(%s): expected %s, got %s", test.input, test.expected, result)
		}
	}
}

func TestContactsManager_UnloadedState(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewContactsManager(tempDir)

	// Test methods on unloaded manager
	if manager.GetContactsCount() != 0 {
		t.Error("GetContactsCount should return 0 for unloaded manager")
	}

	if manager.ContactExists("Anyone") {
		t.Error("ContactExists should return false for unloaded manager")
	}

	if manager.IsKnownNumber("5555551234") {
		t.Error("IsKnownNumber should return false for unloaded manager")
	}

	_, found := manager.GetContactByNumber("5555551234")
	if found {
		t.Error("GetContactByNumber should return false for unloaded manager")
	}

	_, found = manager.GetNumbersByContact("Anyone")
	if found {
		t.Error("GetNumbersByContact should return false for unloaded manager")
	}
}

func TestContactsManager_EmptyContacts(t *testing.T) {
	tempDir := t.TempDir()
	contactsPath := filepath.Join(tempDir, "contacts.yaml")

	yamlContent := `contacts: []`
	err := os.WriteFile(contactsPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := NewContactsManager(tempDir)
	err = manager.LoadContacts()
	if err != nil {
		t.Fatalf("LoadContacts failed: %v", err)
	}

	if manager.GetContactsCount() != 0 {
		t.Errorf("Expected 0 contacts, got %d", manager.GetContactsCount())
	}

	contacts, err := manager.GetAllContacts()
	if err != nil {
		t.Fatalf("GetAllContacts failed: %v", err)
	}
	if len(contacts) != 0 {
		t.Errorf("Expected 0 contacts, got %d", len(contacts))
	}
}

func TestContactsManager_ContactsWithEmptyNumbers(t *testing.T) {
	tempDir := t.TempDir()
	contactsPath := filepath.Join(tempDir, "contacts.yaml")

	yamlContent := `contacts:
  - name: "Bob Ross"
    numbers:
      - "+15555551234"
      - ""  # Empty number should be skipped
      - "5555555678"
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

	// Should find contact by both non-empty numbers
	name, found := manager.GetContactByNumber("5555551234")
	if !found || name != "Bob Ross" {
		t.Error("Should find contact by first number")
	}

	name, found = manager.GetContactByNumber("5555555678")
	if !found || name != "Bob Ross" {
		t.Error("Should find contact by second number")
	}
}