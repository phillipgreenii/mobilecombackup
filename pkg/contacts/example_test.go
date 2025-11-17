package contacts_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/phillipgreenii/mobilecombackup/pkg/contacts"
)

const (
	// Test constants
	exampleTempDir = "/tmp/example"
)

// Example demonstrates basic usage of the Manager
func ExampleManager() {
	// Create a manager for a repository
	manager := contacts.NewContactsManager("/path/to/repository")

	// Load contacts from contacts.yaml
	err := manager.LoadContacts(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	// Get contact count
	count := manager.GetContactsCount()
	fmt.Printf("Loaded %d contacts\n", count)

	// Look up a contact by phone number
	name, found := manager.GetContactByNumber("5555551234")
	if found {
		fmt.Printf("Contact: %s\n", name)
	}
}

// Example demonstrates phone number lookup with various formats
func ExampleManager_GetContactByNumber() {
	manager := contacts.NewContactsManager("/path/to/repository")
	_ = manager.LoadContacts(context.Background())

	// All these formats will find the same contact if the number exists
	testNumbers := []string{
		"+15555551234",   // International format
		"15555551234",    // With country code
		"5555551234",     // 10 digits
		"(555) 555-1234", // Formatted
		"555-555-1234",   // Dashed
		"555.555.1234",   // Dotted
		"555 555 1234",   // Spaced
	}

	for _, number := range testNumbers {
		name, found := manager.GetContactByNumber(number)
		if found {
			fmt.Printf("Number %s belongs to: %s\n", number, name)
		}
	}
}

// Example demonstrates getting all numbers for a contact
func ExampleManager_GetNumbersByContact() {
	manager := contacts.NewContactsManager("/path/to/repository")
	_ = manager.LoadContacts(context.Background())

	// Get all numbers for a specific contact
	numbers, found := manager.GetNumbersByContact("Bob Ross")
	if found {
		fmt.Printf("Bob Ross has %d phone numbers:\n", len(numbers))
		for _, number := range numbers {
			fmt.Printf("  %s\n", number)
		}
	}
}

// Example demonstrates checking if numbers are known
func ExampleManager_IsKnownNumber() {
	manager := contacts.NewContactsManager("/path/to/repository")
	_ = manager.LoadContacts(context.Background())

	testNumbers := []string{
		"5555551234",
		"9999999999",
		"+15555555678",
	}

	for _, number := range testNumbers {
		if manager.IsKnownNumber(number) {
			fmt.Printf("Number %s is known\n", number)
		} else {
			fmt.Printf("Number %s is unknown\n", number)
		}
	}
}

// Example demonstrates iterating through all contacts
func ExampleManager_GetAllContacts() {
	manager := contacts.NewContactsManager("/path/to/repository")
	_ = manager.LoadContacts(context.Background())

	contacts, err := manager.GetAllContacts(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d contacts:\n", len(contacts))
	for _, contact := range contacts {
		fmt.Printf("Contact: %s\n", contact.Name)
		for _, number := range contact.Numbers {
			fmt.Printf("  Number: %s\n", number)
		}
	}
}

// Example demonstrates contact existence checking
func ExampleManager_ContactExists() {
	manager := contacts.NewContactsManager("/path/to/repository")
	_ = manager.LoadContacts(context.Background())

	contacts := []string{
		"Bob Ross",
		"Jane Smith",
		"Unknown Person",
	}

	for _, name := range contacts {
		if manager.ContactExists(name) {
			fmt.Printf("Contact '%s' exists\n", name)
		} else {
			fmt.Printf("Contact '%s' does not exist\n", name)
		}
	}
}

// Example demonstrates handling the special "<unknown>" contact
func ExampleManager_unknownContact() {
	manager := contacts.NewContactsManager("/path/to/repository")
	_ = manager.LoadContacts(context.Background())

	// Check if a number belongs to unknown contacts
	name, found := manager.GetContactByNumber("8888888888")
	if found && name == "<unknown>" {
		fmt.Println("This number belongs to unknown contacts")
	}

	// Get all unknown numbers
	numbers, found := manager.GetNumbersByContact("<unknown>")
	if found {
		fmt.Printf("Unknown contact has %d numbers\n", len(numbers))
		for _, number := range numbers {
			fmt.Printf("  Unknown number: %s\n", number)
		}
	}
}

// Example demonstrates error handling for common scenarios
func ExampleManager_errorHandling() {
	manager := contacts.NewContactsManager("/path/to/repository")

	// LoadContacts may fail for various reasons
	err := manager.LoadContacts(context.Background())
	if err != nil {
		// Handle different types of errors
		fmt.Printf("Failed to load contacts: %v\n", err)

		// Still safe to use other methods, they'll return empty results
		count := manager.GetContactsCount()
		fmt.Printf("Contact count: %d\n", count) // Will be 0

		_, found := manager.GetContactByNumber("5555551234")
		fmt.Printf("Number found: %v\n", found) // Will be false
	}
}

// Example demonstrates phone number normalization behavior
func Example_phoneNumberNormalization() {
	manager := contacts.NewContactsManager("/path/to/repository")
	_ = manager.LoadContacts(context.Background())

	// These variations all normalize to the same number
	variations := []string{
		"+1-555-555-1234",
		"1-555-555-1234",
		"(555) 555-1234",
		"555.555.1234",
		"555 555 1234",
		"5555551234",
	}

	fmt.Println("Phone number normalization examples:")
	for _, variation := range variations {
		name, found := manager.GetContactByNumber(variation)
		if found {
			// All variations should find the same contact
			fmt.Printf("%s -> %s\n", variation, name)
		}
	}
}

// ExampleManager_AddUnprocessedContacts demonstrates the new multi-address parsing functionality
func ExampleManager_AddUnprocessedContacts() {
	// Create a manager for a temporary directory
	tempDir := exampleTempDir
	_ = os.MkdirAll(tempDir, 0750)
	defer func() { _ = os.RemoveAll(tempDir) }()

	manager := contacts.NewContactsManager(tempDir)

	// Single address and contact name
	err := manager.AddUnprocessedContacts(context.Background(), "5551234567", "John Doe")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Multiple addresses separated by ~ with corresponding names separated by ,
	err = manager.AddUnprocessedContacts(context.Background(), "5559876543~5551111111", "Jane Smith,Bob Wilson")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Get structured unprocessed entries (sorted by phone number)
	entries := manager.GetUnprocessedEntries()

	for _, entry := range entries {
		fmt.Printf("Phone: %s, Names: %v\n", entry.PhoneNumber, entry.ContactNames)
	}

	// Output:
	// Phone: 5551111111, Names: [Bob Wilson]
	// Phone: 5551234567, Names: [John Doe]
	// Phone: 5559876543, Names: [Jane Smith]
}

// ExampleManager_AddUnprocessedContacts_countMismatch demonstrates validation error handling
func ExampleManager_AddUnprocessedContacts_countMismatch() {
	manager := contacts.NewContactsManager("")

	// Address count doesn't match contact name count - this will return an error
	err := manager.AddUnprocessedContacts(context.Background(), "5551234567~5559876543", "John Doe")
	if err != nil {
		fmt.Printf("Validation error: %v\n", err)
	}

	// Output:
	// Validation error: address count (2) does not match contact name count (1)
}

// ExampleManager_GetUnprocessedEntries demonstrates the new structured format
func ExampleManager_GetUnprocessedEntries() {
	tempDir := exampleTempDir
	_ = os.MkdirAll(tempDir, 0750)
	defer func() { _ = os.RemoveAll(tempDir) }()

	contactsPath := filepath.Join(tempDir, "contacts.yaml")

	// Create a contacts.yaml file with the new structured unprocessed format
	yamlContent := `contacts:
  - name: "Alice Johnson"
    numbers: ["5554567890"]

unprocessed:
  - phone_number: "5551234567"
    contact_names: ["John Doe", "Johnny"]
  - phone_number: "5559876543"
    contact_names: ["Jane Smith"]
`

	_ = os.WriteFile(contactsPath, []byte(yamlContent), 0600)

	manager := contacts.NewContactsManager(tempDir)
	err := manager.LoadContacts(context.Background())
	if err != nil {
		fmt.Printf("Error loading contacts: %v\n", err)
		return
	}

	// Get unprocessed entries in structured format
	entries := manager.GetUnprocessedEntries()
	fmt.Printf("Unprocessed entries: %d\n", len(entries))
	for _, entry := range entries {
		fmt.Printf("  %s: %v\n", entry.PhoneNumber, entry.ContactNames)
	}

	// Output:
	// Unprocessed entries: 2
	//   5551234567: [John Doe Johnny]
	//   5559876543: [Jane Smith]
}

// Example_knownContactFiltering demonstrates how known contacts are excluded during processing
func Example_knownContactFiltering() {
	tempDir := exampleTempDir
	_ = os.MkdirAll(tempDir, 0750)
	defer func() { _ = os.RemoveAll(tempDir) }()

	contactsPath := filepath.Join(tempDir, "contacts.yaml")

	// Create contacts.yaml with existing contact
	yamlContent := `contacts:
  - name: "Alice Johnson"
    numbers: ["5551234567"]
`
	_ = os.WriteFile(contactsPath, []byte(yamlContent), 0600)

	manager := contacts.NewContactsManager(tempDir)
	_ = manager.LoadContacts(context.Background())

	// Try to add both known and unknown contacts
	_ = manager.AddUnprocessedContacts(context.Background(), "5551234567~5559876543", "John Doe,Jane Smith")

	entries := manager.GetUnprocessedEntries()
	fmt.Printf("Unprocessed entries: %d\n", len(entries))
	for _, entry := range entries {
		fmt.Printf("  %s: %v\n", entry.PhoneNumber, entry.ContactNames)
	}

	// Output:
	// Unprocessed entries: 1
	//   5559876543: [Jane Smith]
}
