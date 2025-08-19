// Package contacts provides functionality for managing and processing contact information
// from mobile backup files, including YAML-based contact storage and phone number normalization.
package contacts

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Manager provides contact management functionality
type Manager struct {
	repoPath     string
	contacts     map[string]*Contact // name -> Contact
	numberToName map[string]string   // normalized number -> name
	unprocessed  map[string][]string // normalized number -> names
	loaded       bool
}

// NewContactsManager creates a new Manager for the given repository path
func NewContactsManager(repoPath string) *Manager {
	return &Manager{
		repoPath:     repoPath,
		contacts:     make(map[string]*Contact),
		numberToName: make(map[string]string),
		unprocessed:  make(map[string][]string),
		loaded:       false,
	}
}

// LoadContacts loads all contacts from contacts.yaml
func (cm *Manager) LoadContacts() error {
	contactsData, err := cm.loadContactsFile()
	if err != nil {
		return err
	}

	// Initialize data structures
	cm.contacts = make(map[string]*Contact)
	cm.numberToName = make(map[string]string)
	cm.unprocessed = make(map[string][]string)

	// Build lookup maps and validate
	if err := cm.buildContactMaps(contactsData.Contacts); err != nil {
		return err
	}

	// Process unprocessed entries
	cm.processUnprocessedEntries(contactsData.Unprocessed)

	cm.loaded = true
	return nil
}

// loadContactsFile reads and parses the contacts.yaml file
func (cm *Manager) loadContactsFile() (*Data, error) {
	contactsPath := filepath.Join(cm.repoPath, "contacts.yaml")

	// Check if file exists
	if _, err := os.Stat(contactsPath); os.IsNotExist(err) {
		// Missing contacts.yaml is not an error, just means no contacts
		return &Data{}, nil
	}

	data, err := os.ReadFile(contactsPath) // #nosec G304
	if err != nil {
		return nil, fmt.Errorf("failed to read contacts.yaml: %w", err)
	}

	var contactsData Data
	if err := yaml.Unmarshal(data, &contactsData); err != nil {
		// Try parsing with old format (string array)
		if convertedData, convertErr := cm.convertOldFormat(data); convertErr == nil {
			return convertedData, nil
		}
		return nil, fmt.Errorf("failed to parse contacts.yaml: %w", err)
	}

	return &contactsData, nil
}

// convertOldFormat converts old string array format to new format
func (cm *Manager) convertOldFormat(data []byte) (*Data, error) {
	var oldFormatData struct {
		Contacts    []*Contact `yaml:"contacts"`
		Unprocessed []string   `yaml:"unprocessed,omitempty"`
	}
	if err := yaml.Unmarshal(data, &oldFormatData); err != nil {
		return nil, err
	}

	// Convert old format to new format
	contactsData := &Data{
		Contacts: oldFormatData.Contacts,
	}

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

	return contactsData, nil
}

// buildContactMaps builds lookup maps for contacts and validates for duplicates
func (cm *Manager) buildContactMaps(contacts []*Contact) error {
	duplicateNumbers := make(map[string][]string) // normalized number -> contact names

	for _, contact := range contacts {
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
	return cm.validateNoDuplicateNumbers(duplicateNumbers)
}

// validateNoDuplicateNumbers checks for and reports duplicate phone numbers
func (cm *Manager) validateNoDuplicateNumbers(duplicateNumbers map[string][]string) error {
	if len(duplicateNumbers) == 0 {
		return nil
	}

	errors := make([]string, 0, len(duplicateNumbers))
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

// processUnprocessedEntries processes the unprocessed entries section
func (cm *Manager) processUnprocessedEntries(unprocessed []UnprocessedEntry) {
	for _, entry := range unprocessed {
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
}

// isUnknownContact checks if a contact name represents an unknown contact placeholder
func isUnknownContact(contactName string) bool {
	unknownIndicators := []string{"(Unknown)", "null", ""}
	for _, indicator := range unknownIndicators {
		if contactName == indicator {
			return true
		}
	}
	return false
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
func (cm *Manager) GetContactByNumber(number string) (string, bool) {
	if !cm.loaded {
		return "", false
	}

	normalized := normalizePhoneNumber(number)
	name, exists := cm.numberToName[normalized]
	return name, exists
}

// GetNumbersByContact returns all numbers for a contact name
func (cm *Manager) GetNumbersByContact(name string) ([]string, bool) {
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
func (cm *Manager) GetAllContacts() ([]*Contact, error) {
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
func (cm *Manager) ContactExists(name string) bool {
	if !cm.loaded {
		return false
	}

	_, exists := cm.contacts[name]
	return exists
}

// IsKnownNumber checks if a number has a contact
func (cm *Manager) IsKnownNumber(number string) bool {
	_, exists := cm.GetContactByNumber(number)
	return exists
}

// GetContactsCount returns total number of contacts
func (cm *Manager) GetContactsCount() int {
	if !cm.loaded {
		return 0
	}

	return len(cm.contacts)
}

// AddUnprocessedContact adds a contact to the unprocessed section
func (cm *Manager) AddUnprocessedContact(phone, name string) {
	if phone == "" || name == "" {
		return // Skip empty values
	}

	normalized := normalizePhoneNumber(phone)
	if normalized == "" {
		return // Skip invalid phone numbers
	}

	existingNames := cm.unprocessed[normalized]

	// Check if this exact combination already exists
	for _, existingName := range existingNames {
		if existingName == name {
			return // Duplicate, skip
		}
	}

	// Handle unknown contact replacement logic
	if isUnknownContact(name) {
		// Only add unknown contact if no real contacts exist for this number
		hasRealContacts := false
		for _, existingName := range existingNames {
			if !isUnknownContact(existingName) {
				hasRealContacts = true
				break
			}
		}

		if hasRealContacts {
			return // Don't add unknown contact when real contacts exist
		}
	} else {
		// This is a real contact - remove any unknown contacts for this number
		var filteredNames []string
		for _, existingName := range existingNames {
			if !isUnknownContact(existingName) {
				filteredNames = append(filteredNames, existingName)
			}
		}
		cm.unprocessed[normalized] = filteredNames
	}

	// Add the new name
	cm.unprocessed[normalized] = append(cm.unprocessed[normalized], name)
}

// GetUnprocessedContacts returns all unprocessed contacts
func (cm *Manager) GetUnprocessedContacts() map[string][]string {
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
func (cm *Manager) AddUnprocessedContacts(addresses, contactNames string) error {
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

		// Normalize address for consistent checking with stored contacts
		normalized := normalizePhoneNumber(address)
		if cm.IsKnownNumber(normalized) {
			continue // Skip known contacts
		}

		// Add to unprocessed using raw address (preserved for backwards compatibility)
		cm.addUnprocessedEntry(address, name)
	}

	return nil
}

// GetUnprocessedEntries returns all unprocessed entries in structured format
func (cm *Manager) GetUnprocessedEntries() []UnprocessedEntry {
	var entries []UnprocessedEntry

	// Group by raw phone number and sort
	phoneNumbers := make([]string, 0, len(cm.unprocessed))
	for phone := range cm.unprocessed {
		phoneNumbers = append(phoneNumbers, phone)
	}

	// Sort by raw phone number (lexicographic)
	sort.Strings(phoneNumbers)

	// Create entries with combined contact names
	for _, phone := range phoneNumbers {
		names := cm.unprocessed[phone]
		if len(names) > 0 {
			entry := UnprocessedEntry{
				PhoneNumber:  phone,
				ContactNames: make([]string, len(names)),
			}
			copy(entry.ContactNames, names)

			// Sort contact names for consistent ordering
			sort.Strings(entry.ContactNames)

			entries = append(entries, entry)
		}
	}

	return entries
}

// addUnprocessedEntry adds a single unprocessed entry (internal helper)
func (cm *Manager) addUnprocessedEntry(phone, name string) {
	normalized := normalizePhoneNumber(phone)
	if normalized == "" {
		return // Skip invalid phone numbers
	}

	existingNames := cm.unprocessed[normalized]

	// Check if this exact combination already exists
	for _, existingName := range existingNames {
		if existingName == name {
			return // Duplicate, skip
		}
	}

	// Handle unknown contact replacement logic
	if isUnknownContact(name) {
		// Only add unknown contact if no real contacts exist for this number
		hasRealContacts := false
		for _, existingName := range existingNames {
			if !isUnknownContact(existingName) {
				hasRealContacts = true
				break
			}
		}

		if hasRealContacts {
			return // Don't add unknown contact when real contacts exist
		}
	} else {
		// This is a real contact - remove any unknown contacts for this number
		var filteredNames []string
		for _, existingName := range existingNames {
			if !isUnknownContact(existingName) {
				filteredNames = append(filteredNames, existingName)
			}
		}
		cm.unprocessed[normalized] = filteredNames
	}

	// Add the new name
	cm.unprocessed[normalized] = append(cm.unprocessed[normalized], name)
}

// SaveContacts writes the current state to contacts.yaml
func (cm *Manager) SaveContacts(path string) error {
	// Prepare data structure for YAML
	contactsData := Data{
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

	// Add unprocessed entries using sorted method to ensure consistent ordering
	contactsData.Unprocessed = cm.GetUnprocessedEntries()

	// Marshal to YAML
	yamlData, err := yaml.Marshal(&contactsData)
	if err != nil {
		return fmt.Errorf("failed to marshal contacts to YAML: %w", err)
	}

	// Write to temp file first for atomic operation
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, yamlData, 0600); err != nil {
		return fmt.Errorf("failed to write temp contacts file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename temp contacts file: %w", err)
	}

	return nil
}

// Context-aware method implementations

// LoadContactsContext loads all contacts from contacts.yaml with context support
func (cm *Manager) LoadContactsContext(ctx context.Context) error {
	// Check context before starting
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return cm.LoadContacts()
}

// GetAllContactsContext returns all contacts with context support
func (cm *Manager) GetAllContactsContext(ctx context.Context) ([]*Contact, error) {
	// Check context before starting
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return cm.GetAllContacts()
}

// AddUnprocessedContactsContext adds contacts from multi-address SMS parsing with context support
func (cm *Manager) AddUnprocessedContactsContext(ctx context.Context, addresses, contactNames string) error {
	// Check context before starting
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return cm.AddUnprocessedContacts(addresses, contactNames)
}

// SaveContactsContext writes the current state to contacts.yaml with context support
func (cm *Manager) SaveContactsContext(ctx context.Context, path string) error {
	// Check context before starting
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return cm.SaveContacts(path)
}
