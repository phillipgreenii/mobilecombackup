package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/phillipgreen/mobilecombackup/pkg/contacts"
)

// mockContactsReader implements ContactsReader for testing
type mockContactsReader struct {
	contacts         []*contacts.Contact
	loadError        error
	getAllError      error
	loadCalled       bool
	numberToContact  map[string]string
	contactToNumbers map[string][]string
	contactExists    map[string]bool
}

func (m *mockContactsReader) LoadContacts() error {
	m.loadCalled = true
	if m.loadError != nil {
		return m.loadError
	}

	// Build lookup maps from contacts
	m.numberToContact = make(map[string]string)
	m.contactToNumbers = make(map[string][]string)
	m.contactExists = make(map[string]bool)

	for _, contact := range m.contacts {
		m.contactExists[contact.Name] = true
		m.contactToNumbers[contact.Name] = contact.Numbers
		for _, number := range contact.Numbers {
			m.numberToContact[number] = contact.Name
		}
	}

	return nil
}

func (m *mockContactsReader) GetContactByNumber(number string) (string, bool) {
	if !m.loadCalled {
		return "", false
	}
	name, exists := m.numberToContact[number]
	return name, exists
}

func (m *mockContactsReader) GetNumbersByContact(name string) ([]string, bool) {
	if !m.loadCalled {
		return nil, false
	}
	numbers, exists := m.contactToNumbers[name]
	return numbers, exists
}

func (m *mockContactsReader) GetAllContacts() ([]*contacts.Contact, error) {
	if !m.loadCalled {
		return nil, fmt.Errorf("contacts not loaded")
	}
	if m.getAllError != nil {
		return nil, m.getAllError
	}
	return m.contacts, nil
}

func (m *mockContactsReader) ContactExists(name string) bool {
	if !m.loadCalled {
		return false
	}
	return m.contactExists[name]
}

func (m *mockContactsReader) IsKnownNumber(number string) bool {
	if !m.loadCalled {
		return false
	}
	_, exists := m.numberToContact[number]
	return exists
}

func (m *mockContactsReader) GetContactsCount() int {
	if !m.loadCalled {
		return 0
	}
	return len(m.contacts)
}

func (m *mockContactsReader) AddUnprocessedContacts(addresses, contactNames string) error {
	// Mock implementation - do nothing for tests
	return nil
}

func (m *mockContactsReader) GetUnprocessedEntries() []contacts.UnprocessedEntry {
	// Mock implementation - return empty slice for tests
	return []contacts.UnprocessedEntry{}
}

func TestContactsValidatorImpl_ValidateContactsStructure(t *testing.T) {
	tempDir := t.TempDir()

	mockReader := &mockContactsReader{}
	validator := NewContactsValidator(tempDir, mockReader)

	// Test missing contacts.yaml
	violations := validator.ValidateContactsStructure()
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation for missing contacts.yaml, got %d", len(violations))
	}

	if len(violations) > 0 {
		if violations[0].Type != MissingFile {
			t.Errorf("Expected MissingFile violation, got %s", violations[0].Type)
		}
		if violations[0].Severity != SeverityWarning {
			t.Errorf("Expected warning severity for missing contacts.yaml, got %s", violations[0].Severity)
		}
	}

	// Create contacts.yaml file
	contactsFile := filepath.Join(tempDir, "contacts.yaml")
	err := os.WriteFile(contactsFile, []byte("contacts: []"), 0644)
	if err != nil {
		t.Fatalf("Failed to create contacts.yaml: %v", err)
	}

	// Test with valid file
	violations = validator.ValidateContactsStructure()
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations with valid contacts.yaml, got %d: %v", len(violations), violations)
	}

	// Test with load error
	mockReader.loadError = fmt.Errorf("invalid YAML format")
	violations = validator.ValidateContactsStructure()
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation for load error, got %d", len(violations))
	}

	if len(violations) > 0 && violations[0].Type != InvalidFormat {
		t.Errorf("Expected InvalidFormat violation, got %s", violations[0].Type)
	}
}

func TestContactsValidatorImpl_ValidateContactsData(t *testing.T) {
	tempDir := t.TempDir()

	// Test with valid contacts data
	validContacts := []*contacts.Contact{
		{
			Name:    "John Doe",
			Numbers: []string{"+15551234567", "555-123-4567"},
		},
		{
			Name:    "Jane Smith",
			Numbers: []string{"+15559876543"},
		},
	}

	mockReader := &mockContactsReader{
		contacts: validContacts,
	}

	validator := NewContactsValidator(tempDir, mockReader)
	violations := validator.ValidateContactsData()

	if len(violations) != 0 {
		t.Errorf("Expected 0 violations with valid contacts data, got %d: %v", len(violations), violations)
	}
}

func TestContactsValidatorImpl_ValidateContactsData_ValidationErrors(t *testing.T) {
	tempDir := t.TempDir()

	// Test with various data problems
	problematicContacts := []*contacts.Contact{
		{
			Name:    "", // Empty name
			Numbers: []string{"+15551234567"},
		},
		{
			Name:    "John Doe",
			Numbers: []string{}, // No numbers
		},
		{
			Name:    "Jane Smith",
			Numbers: []string{""}, // Empty number
		},
		{
			Name:    "Bob Johnson",
			Numbers: []string{"invalid@phone"}, // Invalid format
		},
		{
			Name:    "Alice Brown",
			Numbers: []string{"+15551234567"}, // Duplicate number from first contact
		},
		{
			Name:    "John Doe", // Duplicate name
			Numbers: []string{"+15559999999"},
		},
	}

	mockReader := &mockContactsReader{
		contacts: problematicContacts,
	}

	validator := NewContactsValidator(tempDir, mockReader)
	violations := validator.ValidateContactsData()

	// Should have multiple violations
	if len(violations) < 4 {
		t.Errorf("Expected at least 4 violations, got %d: %v", len(violations), violations)
	}

	// Check for specific violation types
	foundEmptyName := false
	foundNoNumbers := false
	foundEmptyNumber := false
	foundInvalidFormat := false
	foundDuplicateNumber := false
	foundDuplicateName := false

	for _, violation := range violations {
		switch violation.Message {
		case "Contact name cannot be empty":
			foundEmptyName = true
		case "Contact 'John Doe' has no phone numbers":
			foundNoNumbers = true
		case "Empty phone number for contact 'Jane Smith'":
			foundEmptyNumber = true
		case "Phone number 'invalid@phone' has unusual format for contact 'Bob Johnson'":
			foundInvalidFormat = true
		case "Phone number '+15551234567' assigned to multiple contacts: '(unnamed contact 1)' and 'Alice Brown'":
			foundDuplicateNumber = true
		case "Duplicate contact name: John Doe":
			foundDuplicateName = true
		}
	}

	if !foundEmptyName {
		t.Error("Expected empty name violation")
	}
	if !foundNoNumbers {
		t.Error("Expected no numbers violation")
	}
	if !foundEmptyNumber {
		t.Error("Expected empty number violation")
	}
	if !foundInvalidFormat {
		t.Error("Expected invalid format violation")
	}
	if !foundDuplicateNumber {
		t.Error("Expected duplicate number violation")
	}
	if !foundDuplicateName {
		t.Error("Expected duplicate name violation")
	}
}

func TestContactsValidatorImpl_ValidateContactReferences(t *testing.T) {
	tempDir := t.TempDir()

	// Test contacts
	testContacts := []*contacts.Contact{
		{
			Name:    "John Doe",
			Numbers: []string{"+15551234567"},
		},
		{
			Name:    "Jane Smith",
			Numbers: []string{"+15559876543"},
		},
	}

	mockReader := &mockContactsReader{
		contacts: testContacts,
	}

	validator := NewContactsValidator(tempDir, mockReader)

	// Referenced contacts from calls and SMS
	callContacts := map[string]bool{
		"John Doe":     true,
		"Unknown User": true, // Not in contacts
		"":             true, // Empty (should be ignored)
		"(Unknown)":    true, // Special value (should be ignored)
	}

	smsContacts := map[string]bool{
		"Jane Smith":  true,
		"Bob Johnson": true, // Not in contacts
		"null":        true, // Special value (should be ignored)
	}

	violations := validator.ValidateContactReferences(callContacts, smsContacts)

	// Should have violations for missing contacts and orphaned contacts
	if len(violations) < 2 {
		t.Errorf("Expected at least 2 violations, got %d: %v", len(violations), violations)
	}

	// Check for specific violations
	foundMissingUnknownUser := false
	foundMissingBobJohnson := false

	for _, violation := range violations {
		switch violation.Message {
		case "Contact 'Unknown User' referenced in calls/SMS but not found in contacts.yaml":
			foundMissingUnknownUser = true
			if violation.Severity != SeverityError {
				t.Errorf("Expected error severity for missing contact reference, got %s", violation.Severity)
			}
		case "Contact 'Bob Johnson' referenced in calls/SMS but not found in contacts.yaml":
			foundMissingBobJohnson = true
		}
	}

	if !foundMissingUnknownUser {
		t.Error("Expected missing 'Unknown User' violation")
	}

	if !foundMissingBobJohnson {
		t.Error("Expected missing 'Bob Johnson' violation")
	}
}

func TestContactsValidatorImpl_ValidateContactReferences_EmptyContacts(t *testing.T) {
	tempDir := t.TempDir()

	// Test with no contacts loaded
	mockReader := &mockContactsReader{
		contacts: []*contacts.Contact{},
	}

	validator := NewContactsValidator(tempDir, mockReader)

	callContacts := map[string]bool{
		"Unknown User": true,
	}

	violations := validator.ValidateContactReferences(callContacts, make(map[string]bool))

	// Should have warning (not error) when no contacts are available
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation with empty contacts, got %d", len(violations))
	}

	if len(violations) > 0 && violations[0].Severity != SeverityWarning {
		t.Errorf("Expected warning severity with empty contacts, got %s", violations[0].Severity)
	}
}

func TestContactsValidatorImpl_ValidateContactReferences_OrphanedContacts(t *testing.T) {
	tempDir := t.TempDir()

	// Test contacts with some unreferenced
	testContacts := []*contacts.Contact{
		{
			Name:    "John Doe",
			Numbers: []string{"+15551234567"},
		},
		{
			Name:    "Unused Contact",
			Numbers: []string{"+15559999999"},
		},
	}

	mockReader := &mockContactsReader{
		contacts: testContacts,
	}

	validator := NewContactsValidator(tempDir, mockReader)

	// Only reference John Doe
	callContacts := map[string]bool{
		"John Doe": true,
	}

	violations := validator.ValidateContactReferences(callContacts, make(map[string]bool))

	// Should have violation for orphaned contact
	foundOrphanedContact := false
	for _, violation := range violations {
		if violation.Message == "Contact 'Unused Contact' defined but not referenced in any calls/SMS" {
			foundOrphanedContact = true
			if violation.Severity != SeverityWarning {
				t.Errorf("Expected warning severity for orphaned contact, got %s", violation.Severity)
			}
		}
	}

	if !foundOrphanedContact {
		t.Error("Expected orphaned contact violation")
	}
}

func TestContactsValidatorImpl_ErrorHandling(t *testing.T) {
	tempDir := t.TempDir()

	// Test with load error
	mockReader := &mockContactsReader{
		loadError: fmt.Errorf("failed to load contacts"),
	}

	validator := NewContactsValidator(tempDir, mockReader)

	// Data validation should fail with load error
	violations := validator.ValidateContactsData()
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation for load error, got %d", len(violations))
	}

	// Reference validation should not fail with load error (contacts are optional)
	violations = validator.ValidateContactReferences(make(map[string]bool), make(map[string]bool))
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations for reference check with load error, got %d", len(violations))
	}

	// Test with getAllContacts error
	mockReader.loadError = nil
	mockReader.getAllError = fmt.Errorf("failed to get all contacts")

	violations = validator.ValidateContactsData()
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation for getAllContacts error, got %d", len(violations))
	}
}

func TestContactsValidatorImpl_PhoneNumberValidation(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name           string
		numbers        []string
		expectWarnings int
	}{
		{
			name:           "valid numbers",
			numbers:        []string{"+15551234567", "555-123-4567", "(555) 123-4567", "555.123.4567"},
			expectWarnings: 0,
		},
		{
			name:           "unusual but acceptable",
			numbers:        []string{"15551234567", "555 123 4567"},
			expectWarnings: 0,
		},
		{
			name:           "invalid formats",
			numbers:        []string{"invalid@email.com", "not-a-phone", "abc123"},
			expectWarnings: 3,
		},
		{
			name:           "mixed valid and invalid",
			numbers:        []string{"+15551234567", "invalid@email.com", "555-123-4567"},
			expectWarnings: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockReader := &mockContactsReader{
				contacts: []*contacts.Contact{
					{
						Name:    "Test Contact",
						Numbers: tc.numbers,
					},
				},
			}

			validator := NewContactsValidator(tempDir, mockReader)
			violations := validator.ValidateContactsData()

			warningCount := 0
			for _, violation := range violations {
				if violation.Severity == SeverityWarning && violation.Message[:12] == "Phone number" {
					warningCount++
				}
			}

			if warningCount != tc.expectWarnings {
				t.Errorf("Expected %d phone format warnings, got %d: %v", tc.expectWarnings, warningCount, violations)
			}
		})
	}
}
