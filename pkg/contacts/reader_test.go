package contacts

import (
	"os"
	"path/filepath"
	"strings"
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
		{"5555551234", "Bob Ross", true},     // Normalized lookup
		{"15555551234", "Bob Ross", true},    // Without +
		{"(555) 555-1234", "Bob Ross", true}, // Formatted number
		{"555-555-5678", "Bob Ross", true},   // Second number
		{"+15555559999", "", false},          // Not found
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
		{"12345678901", "2345678901"},  // 11 digits starting with 1
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

// Test unprocessed contact functionality
func TestContactsManager_AddUnprocessedContact(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewContactsManager(tempDir)

	// Test adding valid contacts
	manager.AddUnprocessedContact("5551234567", "John Doe")
	manager.AddUnprocessedContact("+1-555-987-6543", "Jane Smith")
	manager.AddUnprocessedContact("(555) 111-2222", "Bob Ross")

	unprocessed := manager.GetUnprocessedContacts()
	if len(unprocessed) != 3 {
		t.Errorf("Expected 3 unprocessed contacts, got %d", len(unprocessed))
	}

	// Check normalization worked correctly
	if _, exists := unprocessed["5551234567"]; !exists {
		t.Error("Phone number 5551234567 should exist after normalization")
	}
	if _, exists := unprocessed["5559876543"]; !exists {
		t.Error("Phone number 5559876543 should exist after normalization")
	}
	if _, exists := unprocessed["5551112222"]; !exists {
		t.Error("Phone number 5551112222 should exist after normalization")
	}
}

func TestContactsManager_AddUnprocessedContact_EmptyValues(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewContactsManager(tempDir)

	// Test adding empty/invalid values - should be ignored
	manager.AddUnprocessedContact("", "John Doe")
	manager.AddUnprocessedContact("5551234567", "")
	manager.AddUnprocessedContact("", "")

	unprocessed := manager.GetUnprocessedContacts()
	if len(unprocessed) != 0 {
		t.Errorf("Expected 0 unprocessed contacts for empty values, got %d", len(unprocessed))
	}
}

func TestContactsManager_AddUnprocessedContact_Duplicates(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewContactsManager(tempDir)

	// Add the same contact multiple times
	manager.AddUnprocessedContact("5551234567", "John Doe")
	manager.AddUnprocessedContact("5551234567", "John Doe")   // Exact duplicate
	manager.AddUnprocessedContact("+15551234567", "John Doe") // Different format, same number and name

	unprocessed := manager.GetUnprocessedContacts()
	if len(unprocessed) != 1 {
		t.Errorf("Expected 1 unprocessed entry, got %d", len(unprocessed))
	}

	names := unprocessed["5551234567"]
	if len(names) != 1 {
		t.Errorf("Expected 1 name for phone number, got %d", len(names))
	}
	if names[0] != "John Doe" {
		t.Errorf("Expected 'John Doe', got '%s'", names[0])
	}
}

func TestContactsManager_AddUnprocessedContact_MultipleNames(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewContactsManager(tempDir)

	// Add multiple names for the same number
	manager.AddUnprocessedContact("5551234567", "John Doe")
	manager.AddUnprocessedContact("5551234567", "Johnny")
	manager.AddUnprocessedContact("5551234567", "J. Doe")
	manager.AddUnprocessedContact("5551234567", "John") // Different name

	unprocessed := manager.GetUnprocessedContacts()
	if len(unprocessed) != 1 {
		t.Errorf("Expected 1 phone number entry, got %d", len(unprocessed))
	}

	names := unprocessed["5551234567"]
	if len(names) != 4 {
		t.Errorf("Expected 4 different names, got %d", len(names))
	}

	// Check all names are present
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}

	expectedNames := []string{"John Doe", "Johnny", "J. Doe", "John"}
	for _, expected := range expectedNames {
		if !nameSet[expected] {
			t.Errorf("Expected name '%s' not found in results", expected)
		}
	}
}

func TestContactsManager_GetUnprocessedContacts_Copy(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewContactsManager(tempDir)

	manager.AddUnprocessedContact("5551234567", "John Doe")
	manager.AddUnprocessedContact("5551234567", "Johnny")

	// Get unprocessed contacts
	unprocessed1 := manager.GetUnprocessedContacts()
	unprocessed2 := manager.GetUnprocessedContacts()

	// Modify first result
	unprocessed1["5551234567"][0] = "Modified"
	unprocessed1["9999999999"] = []string{"New Contact"}

	// Verify second result is unaffected (deep copy)
	if unprocessed2["5551234567"][0] != "John Doe" {
		t.Error("GetUnprocessedContacts should return deep copies")
	}
	if _, exists := unprocessed2["9999999999"]; exists {
		t.Error("Modifications to returned map should not affect internal state")
	}
}

func TestContactsManager_LoadContacts_WithUnprocessed(t *testing.T) {
	tempDir := t.TempDir()
	contactsPath := filepath.Join(tempDir, "contacts.yaml")

	yamlContent := `contacts:
  - name: "Bob Ross"
    numbers:
      - "+15555551234"
      - "5555551234"
unprocessed:
  - "5551234567: John Doe"
  - "5551234567: Johnny"
  - "5559876543: Jane Smith"
  - "malformed entry"  # Should be ignored
  - ": Empty Phone"    # Should be ignored
  - "5558888888: "     # Should be ignored
  - "5557777777:   "   # Should be ignored (whitespace only)
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

	// Check processed contacts
	if manager.GetContactsCount() != 1 {
		t.Errorf("Expected 1 processed contact, got %d", manager.GetContactsCount())
	}

	// Check unprocessed contacts
	unprocessed := manager.GetUnprocessedContacts()
	if len(unprocessed) != 2 {
		t.Errorf("Expected 2 unprocessed phone numbers, got %d", len(unprocessed))
	}

	// Verify first phone number with multiple names
	names := unprocessed["5551234567"]
	if len(names) != 2 {
		t.Errorf("Expected 2 names for 5551234567, got %d", len(names))
	}
	if names[0] != "John Doe" || names[1] != "Johnny" {
		t.Errorf("Expected 'John Doe' and 'Johnny', got %v", names)
	}

	// Verify second phone number
	names = unprocessed["5559876543"]
	if len(names) != 1 {
		t.Errorf("Expected 1 name for 5559876543, got %d", len(names))
	}
	if names[0] != "Jane Smith" {
		t.Errorf("Expected 'Jane Smith', got '%s'", names[0])
	}
}

func TestContactsManager_SaveContacts_NewFile(t *testing.T) {
	tempDir := t.TempDir()
	contactsPath := filepath.Join(tempDir, "contacts.yaml")
	manager := NewContactsManager(tempDir)

	// Add some test data
	manager.AddUnprocessedContact("5551234567", "John Doe")
	manager.AddUnprocessedContact("5551234567", "Johnny")
	manager.AddUnprocessedContact("5559876543", "Jane Smith")

	// Save contacts
	err := manager.SaveContacts(contactsPath)
	if err != nil {
		t.Fatalf("SaveContacts failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(contactsPath); os.IsNotExist(err) {
		t.Fatal("contacts.yaml should exist after save")
	}

	// Read and verify content
	content, err := os.ReadFile(contactsPath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	yamlStr := string(content)
	if !strings.Contains(yamlStr, "5551234567: John Doe") {
		t.Error("Saved file should contain '5551234567: John Doe'")
	}
	if !strings.Contains(yamlStr, "5551234567: Johnny") {
		t.Error("Saved file should contain '5551234567: Johnny'")
	}
	if !strings.Contains(yamlStr, "5559876543: Jane Smith") {
		t.Error("Saved file should contain '5559876543: Jane Smith'")
	}
	if !strings.Contains(yamlStr, "contacts: []") {
		t.Error("Saved file should contain empty contacts array")
	}
}

func TestContactsManager_SaveContacts_ExistingContacts(t *testing.T) {
	tempDir := t.TempDir()
	contactsPath := filepath.Join(tempDir, "contacts.yaml")

	// Create initial file with contacts
	yamlContent := `contacts:
  - name: "Bob Ross"
    numbers:
      - "+15555551234"
      - "5555551234"
  - name: "Jane Smith"
    numbers:
      - "+15555555678"
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

	// Add unprocessed contacts
	manager.AddUnprocessedContact("5551234567", "John Doe")
	manager.AddUnprocessedContact("5559876543", "New Contact")

	// Save
	err = manager.SaveContacts(contactsPath)
	if err != nil {
		t.Fatalf("SaveContacts failed: %v", err)
	}

	// Read and verify both processed and unprocessed are saved
	content, err := os.ReadFile(contactsPath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	yamlStr := string(content)
	// Should have existing contacts
	if !strings.Contains(yamlStr, `name: Bob Ross`) {
		t.Error("Should preserve existing contacts")
	}
	if !strings.Contains(yamlStr, `name: Jane Smith`) {
		t.Error("Should preserve existing contacts")
	}
	// Should have new unprocessed contacts
	if !strings.Contains(yamlStr, "5551234567: John Doe") {
		t.Error("Should include new unprocessed contacts")
	}
	if !strings.Contains(yamlStr, "5559876543: New Contact") {
		t.Error("Should include new unprocessed contacts")
	}
}

func TestContactsManager_SaveContacts_AtomicOperation(t *testing.T) {
	tempDir := t.TempDir()
	contactsPath := filepath.Join(tempDir, "contacts.yaml")
	manager := NewContactsManager(tempDir)

	manager.AddUnprocessedContact("5551234567", "John Doe")

	// Save contacts
	err := manager.SaveContacts(contactsPath)
	if err != nil {
		t.Fatalf("SaveContacts failed: %v", err)
	}

	// Verify temp file was cleaned up
	tempPath := contactsPath + ".tmp"
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Error("Temp file should be cleaned up after successful save")
	}
}
