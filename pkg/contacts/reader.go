package contacts

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ContactsManager provides contact management functionality
type ContactsManager struct {
	repoPath     string
	contacts     map[string]*Contact // name -> Contact
	numberToName map[string]string   // normalized number -> name
	loaded       bool
}

// NewContactsManager creates a new ContactsManager for the given repository path
func NewContactsManager(repoPath string) *ContactsManager {
	return &ContactsManager{
		repoPath:     repoPath,
		contacts:     make(map[string]*Contact),
		numberToName: make(map[string]string),
		loaded:       false,
	}
}

// LoadContacts loads all contacts from contacts.yaml
func (cm *ContactsManager) LoadContacts() error {
	contactsPath := filepath.Join(cm.repoPath, "contacts.yaml")

	// Check if file exists
	if _, err := os.Stat(contactsPath); os.IsNotExist(err) {
		// Missing contacts.yaml is not an error, just means no contacts
		cm.loaded = true
		return nil
	}

	data, err := os.ReadFile(contactsPath)
	if err != nil {
		return fmt.Errorf("failed to read contacts.yaml: %w", err)
	}

	var contactsData ContactsData
	if err := yaml.Unmarshal(data, &contactsData); err != nil {
		return fmt.Errorf("failed to parse contacts.yaml: %w", err)
	}

	// Clear existing data
	cm.contacts = make(map[string]*Contact)
	cm.numberToName = make(map[string]string)

	// Build lookup maps
	duplicateNumbers := make(map[string][]string) // normalized number -> contact names
	for _, contact := range contactsData.Contacts {
		if contact.Name == "" {
			return fmt.Errorf("contact with empty name found")
		}

		// Store contact
		cm.contacts[contact.Name] = contact

		// Build number->name mapping with normalization
		for _, number := range contact.Numbers {
			if number == "" {
				continue
			}

			normalized := normalizePhoneNumber(number)

			// Check for duplicates
			if existingName, exists := cm.numberToName[normalized]; exists {
				if existingName != contact.Name {
					duplicateNumbers[normalized] = append(duplicateNumbers[normalized], existingName, contact.Name)
				}
			} else {
				cm.numberToName[normalized] = contact.Name
			}
		}
	}

	// Report duplicate numbers as errors
	if len(duplicateNumbers) > 0 {
		var errors []string
		for number, names := range duplicateNumbers {
			// Remove duplicates from names list
			uniqueNames := make(map[string]bool)
			for _, name := range names {
				uniqueNames[name] = true
			}
			var nameList []string
			for name := range uniqueNames {
				nameList = append(nameList, name)
			}
			errors = append(errors, fmt.Sprintf("number %s assigned to multiple contacts: %v", number, nameList))
		}
		return fmt.Errorf("duplicate phone numbers found: %s", strings.Join(errors, "; "))
	}

	cm.loaded = true
	return nil
}

// normalizePhoneNumber normalizes a phone number for consistent lookup
func normalizePhoneNumber(number string) string {
	// Remove all non-digit characters
	re := regexp.MustCompile(`\D`)
	cleaned := re.ReplaceAllString(number, "")

	// Handle common US formats
	if len(cleaned) == 11 && cleaned[0] == '1' {
		// Remove leading 1 for US numbers (+1XXXXXXXXXX -> XXXXXXXXXX)
		cleaned = cleaned[1:]
	}

	return cleaned
}

// GetContactByNumber returns contact name for a phone number
func (cm *ContactsManager) GetContactByNumber(number string) (string, bool) {
	if !cm.loaded {
		return "", false
	}

	normalized := normalizePhoneNumber(number)
	name, exists := cm.numberToName[normalized]
	return name, exists
}

// GetNumbersByContact returns all numbers for a contact name
func (cm *ContactsManager) GetNumbersByContact(name string) ([]string, bool) {
	if !cm.loaded {
		return nil, false
	}

	contact, exists := cm.contacts[name]
	if !exists {
		return nil, false
	}

	// Return a copy to prevent modification
	numbers := make([]string, len(contact.Numbers))
	copy(numbers, contact.Numbers)
	return numbers, true
}

// GetAllContacts returns all contacts
func (cm *ContactsManager) GetAllContacts() ([]*Contact, error) {
	if !cm.loaded {
		if err := cm.LoadContacts(); err != nil {
			return nil, err
		}
	}

	contacts := make([]*Contact, 0, len(cm.contacts))
	for _, contact := range cm.contacts {
		// Return a copy to prevent modification
		contactCopy := &Contact{
			Name:    contact.Name,
			Numbers: make([]string, len(contact.Numbers)),
		}
		copy(contactCopy.Numbers, contact.Numbers)
		contacts = append(contacts, contactCopy)
	}

	return contacts, nil
}

// ContactExists checks if a contact name exists
func (cm *ContactsManager) ContactExists(name string) bool {
	if !cm.loaded {
		return false
	}

	_, exists := cm.contacts[name]
	return exists
}

// IsKnownNumber checks if a number has a contact
func (cm *ContactsManager) IsKnownNumber(number string) bool {
	_, exists := cm.GetContactByNumber(number)
	return exists
}

// GetContactsCount returns total number of contacts
func (cm *ContactsManager) GetContactsCount() int {
	if !cm.loaded {
		return 0
	}

	return len(cm.contacts)
}
