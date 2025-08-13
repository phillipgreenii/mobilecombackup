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
	unprocessed  map[string][]string // normalized number -> names
	loaded       bool
}

// NewContactsManager creates a new ContactsManager for the given repository path
func NewContactsManager(repoPath string) *ContactsManager {
	return &ContactsManager{
		repoPath:     repoPath,
		contacts:     make(map[string]*Contact),
		numberToName: make(map[string]string),
		unprocessed:  make(map[string][]string),
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
		// Try parsing with old format (string array)
		var oldFormatData struct {
			Contacts    []*Contact `yaml:"contacts"`
			Unprocessed []string   `yaml:"unprocessed,omitempty"`
		}
		if oldErr := yaml.Unmarshal(data, &oldFormatData); oldErr != nil {
			return fmt.Errorf("failed to parse contacts.yaml: %w", err)
		}

		// Convert old format to new format
		contactsData.Contacts = oldFormatData.Contacts
		for _, entry := range oldFormatData.Unprocessed {
			// Parse "phone: name" format
			parts := strings.SplitN(entry, ":", 2)
			if len(parts) == 2 {
				phone := strings.TrimSpace(parts[0])
				name := strings.TrimSpace(parts[1])
				if phone != "" && name != "" {
					contactsData.Unprocessed = append(contactsData.Unprocessed, UnprocessedEntry{
						PhoneNumber:  phone,
						ContactNames: []string{name},
					})
				}
			}
		}
	}

	// Clear existing data
	cm.contacts = make(map[string]*Contact)
	cm.numberToName = make(map[string]string)
	cm.unprocessed = make(map[string][]string)

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

	// Parse unprocessed section
	for _, entry := range contactsData.Unprocessed {
		phone := entry.PhoneNumber
		names := entry.ContactNames

		if phone != "" && len(names) > 0 {
			normalized := normalizePhoneNumber(phone)
			if normalized != "" {
				for _, name := range names {
					if name != "" {
						cm.unprocessed[normalized] = append(cm.unprocessed[normalized], name)
					}
				}
			}
		}
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

// AddUnprocessedContact adds a contact to the unprocessed section
func (cm *ContactsManager) AddUnprocessedContact(phone, name string) {
	if phone == "" || name == "" {
		return // Skip empty values
	}

	normalized := normalizePhoneNumber(phone)
	if normalized == "" {
		return // Skip invalid phone numbers
	}

	// Check if this exact combination already exists
	existingNames := cm.unprocessed[normalized]
	for _, existingName := range existingNames {
		if existingName == name {
			return // Duplicate, skip
		}
	}

	// Add the new name
	cm.unprocessed[normalized] = append(cm.unprocessed[normalized], name)
}

// GetUnprocessedContacts returns all unprocessed contacts
func (cm *ContactsManager) GetUnprocessedContacts() map[string][]string {
	// Return a deep copy to prevent modification
	result := make(map[string][]string)
	for phone, names := range cm.unprocessed {
		// Copy the slice
		namesCopy := make([]string, len(names))
		copy(namesCopy, names)
		result[phone] = namesCopy
	}
	return result
}

// AddUnprocessedContacts adds contacts from multi-address SMS parsing
func (cm *ContactsManager) AddUnprocessedContacts(addresses, contactNames string) error {
	// Parse addresses separated by ~
	addressList := strings.Split(addresses, "~")
	// Parse contact names separated by ,
	nameList := strings.Split(contactNames, ",")

	// Validate equal counts
	if len(addressList) != len(nameList) {
		return fmt.Errorf("address count (%d) does not match contact name count (%d)", len(addressList), len(nameList))
	}

	// Process each address-name pair
	for i, address := range addressList {
		name := nameList[i]

		// Skip empty values
		if address == "" || name == "" {
			continue
		}

		// Check if this number is known in main contacts (no normalization per requirement)
		if cm.IsKnownNumber(address) {
			continue // Skip known contacts
		}

		// Add to unprocessed using raw address (no normalization per requirement)
		cm.addUnprocessedEntry(address, name)
	}

	return nil
}

// GetUnprocessedEntries returns all unprocessed entries in structured format
func (cm *ContactsManager) GetUnprocessedEntries() []UnprocessedEntry {
	var entries []UnprocessedEntry

	// Group by raw phone number and sort
	phoneNumbers := make([]string, 0, len(cm.unprocessed))
	for phone := range cm.unprocessed {
		phoneNumbers = append(phoneNumbers, phone)
	}

	// Sort by raw phone number (lexicographic)
	for i := 0; i < len(phoneNumbers); i++ {
		for j := i + 1; j < len(phoneNumbers); j++ {
			if phoneNumbers[i] > phoneNumbers[j] {
				phoneNumbers[i], phoneNumbers[j] = phoneNumbers[j], phoneNumbers[i]
			}
		}
	}

	// Create entries with combined contact names
	for _, phone := range phoneNumbers {
		names := cm.unprocessed[phone]
		if len(names) > 0 {
			entry := UnprocessedEntry{
				PhoneNumber:  phone,
				ContactNames: make([]string, len(names)),
			}
			copy(entry.ContactNames, names)
			entries = append(entries, entry)
		}
	}

	return entries
}

// addUnprocessedEntry adds a single unprocessed entry (internal helper)
func (cm *ContactsManager) addUnprocessedEntry(phone, name string) {
	normalized := normalizePhoneNumber(phone)
	if normalized == "" {
		return // Skip invalid phone numbers
	}

	// Check if this exact combination already exists
	existingNames := cm.unprocessed[normalized]
	for _, existingName := range existingNames {
		if existingName == name {
			return // Duplicate, skip
		}
	}

	// Add the new name
	cm.unprocessed[normalized] = append(cm.unprocessed[normalized], name)
}

// SaveContacts writes the current state to contacts.yaml
func (cm *ContactsManager) SaveContacts(path string) error {
	// Prepare data structure for YAML
	contactsData := ContactsData{
		Contacts: make([]*Contact, 0, len(cm.contacts)),
	}

	// Add all contacts
	for _, contact := range cm.contacts {
		// Create a copy to avoid modifying original
		contactCopy := &Contact{
			Name:    contact.Name,
			Numbers: make([]string, len(contact.Numbers)),
		}
		copy(contactCopy.Numbers, contact.Numbers)
		contactsData.Contacts = append(contactsData.Contacts, contactCopy)
	}

	// Add unprocessed entries in structured format
	for phone, names := range cm.unprocessed {
		if len(names) > 0 {
			entry := UnprocessedEntry{
				PhoneNumber:  phone,
				ContactNames: make([]string, len(names)),
			}
			copy(entry.ContactNames, names)
			contactsData.Unprocessed = append(contactsData.Unprocessed, entry)
		}
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(&contactsData)
	if err != nil {
		return fmt.Errorf("failed to marshal contacts to YAML: %w", err)
	}

	// Write to temp file first for atomic operation
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write temp contacts file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename temp contacts file: %w", err)
	}

	return nil
}
