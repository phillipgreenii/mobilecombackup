package contacts_test

import (
	"fmt"
	"log"

	"github.com/phillipgreen/mobilecombackup/pkg/contacts"
)

// Example demonstrates basic usage of the ContactsManager
func ExampleContactsManager() {
	// Create a manager for a repository
	manager := contacts.NewContactsManager("/path/to/repository")

	// Load contacts from contacts.yaml
	err := manager.LoadContacts()
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
func ExampleContactsManager_GetContactByNumber() {
	manager := contacts.NewContactsManager("/path/to/repository")
	_ = manager.LoadContacts()

	// All these formats will find the same contact if the number exists
	testNumbers := []string{
		"+15555551234",    // International format
		"15555551234",     // With country code
		"5555551234",      // 10 digits
		"(555) 555-1234",  // Formatted
		"555-555-1234",    // Dashed
		"555.555.1234",    // Dotted
		"555 555 1234",    // Spaced
	}

	for _, number := range testNumbers {
		name, found := manager.GetContactByNumber(number)
		if found {
			fmt.Printf("Number %s belongs to: %s\n", number, name)
		}
	}
}

// Example demonstrates getting all numbers for a contact
func ExampleContactsManager_GetNumbersByContact() {
	manager := contacts.NewContactsManager("/path/to/repository")
	_ = manager.LoadContacts()

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
func ExampleContactsManager_IsKnownNumber() {
	manager := contacts.NewContactsManager("/path/to/repository")
	_ = manager.LoadContacts()

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
func ExampleContactsManager_GetAllContacts() {
	manager := contacts.NewContactsManager("/path/to/repository")
	_ = manager.LoadContacts()

	contacts, err := manager.GetAllContacts()
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
func ExampleContactsManager_ContactExists() {
	manager := contacts.NewContactsManager("/path/to/repository")
	_ = manager.LoadContacts()

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
func ExampleContactsManager_unknownContact() {
	manager := contacts.NewContactsManager("/path/to/repository")
	_ = manager.LoadContacts()

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
func ExampleContactsManager_errorHandling() {
	manager := contacts.NewContactsManager("/path/to/repository")

	// LoadContacts may fail for various reasons
	err := manager.LoadContacts()
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
	_ = manager.LoadContacts()

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