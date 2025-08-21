package validation

import (
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/phillipgreenii/mobilecombackup/pkg/contacts"
)

// ContactsValidator validates contacts file and data using Reader
type ContactsValidator interface {
	// ValidateContactsStructure validates contacts.yaml file exists and is readable
	ValidateContactsStructure() []Violation

	// ValidateContactsData validates contacts data integrity and format
	ValidateContactsData() []Violation

	// ValidateContactReferences checks contact references in calls/SMS against contacts data
	ValidateContactReferences(callContacts, smsContacts map[string]bool) []Violation
}

// ContactsValidatorImpl implements ContactsValidator interface
type ContactsValidatorImpl struct {
	repositoryRoot string
	contactsReader contacts.Reader
}

// NewContactsValidator creates a new contacts validator
func NewContactsValidator(repositoryRoot string, contactsReader contacts.Reader) ContactsValidator {
	return &ContactsValidatorImpl{
		repositoryRoot: repositoryRoot,
		contactsReader: contactsReader,
	}
}

// ValidateContactsStructure validates contacts.yaml file exists and is readable
func (v *ContactsValidatorImpl) ValidateContactsStructure() []Violation {
	var violations []Violation

	// Check if contacts.yaml exists
	contactsFile := filepath.Join(v.repositoryRoot, "contacts.yaml")
	if !fileExists(contactsFile) {
		violations = append(violations, Violation{
			Type:     MissingFile,
			Severity: SeverityWarning, // Warning because contacts are optional
			File:     "contacts.yaml",
			Message:  "contacts.yaml file not found (contacts are optional)",
		})
		return violations
	}

	// Try to load contacts to validate file structure
	err := v.contactsReader.LoadContacts()
	if err != nil {
		violations = append(violations, Violation{
			Type:     InvalidFormat,
			Severity: SeverityError,
			File:     "contacts.yaml",
			Message:  fmt.Sprintf("Failed to load contacts.yaml: %v", err),
		})
	}

	return violations
}

// ValidateContactsData validates contacts data integrity and format
func (v *ContactsValidatorImpl) ValidateContactsData() []Violation {
	var violations []Violation

	// Load and get contacts for validation
	allContacts, setupViolations := v.loadContactsForValidation()
	violations = append(violations, setupViolations...)
	if len(allContacts) == 0 && len(setupViolations) > 0 {
		return violations
	}

	// Validate all contacts data integrity
	validationViolations := v.validateAllContactsData(allContacts)
	violations = append(violations, validationViolations...)

	return violations
}

// loadContactsForValidation loads contacts and returns them with any violations
func (v *ContactsValidatorImpl) loadContactsForValidation() ([]*contacts.Contact, []Violation) {
	var violations []Violation

	// Load contacts first
	err := v.contactsReader.LoadContacts()
	if err != nil {
		violations = append(violations, Violation{
			Type:     InvalidFormat,
			Severity: SeverityError,
			File:     "contacts.yaml",
			Message:  fmt.Sprintf("Cannot validate contacts data - failed to load: %v", err),
		})
		return nil, violations
	}

	// Get all contacts for validation
	allContacts, err := v.contactsReader.GetAllContacts()
	if err != nil {
		violations = append(violations, Violation{
			Type:     InvalidFormat,
			Severity: SeverityError,
			File:     "contacts.yaml",
			Message:  fmt.Sprintf("Failed to get contacts for validation: %v", err),
		})
		return nil, violations
	}

	return allContacts, violations
}

// validateAllContactsData validates all contacts and detects duplicates
func (v *ContactsValidatorImpl) validateAllContactsData(allContacts []*contacts.Contact) []Violation {
	var violations []Violation

	// Setup validation state
	seenNames := make(map[string]bool)
	seenNumbers := make(map[string]string) // number -> contact name
	phonePattern := regexp.MustCompile(`^\+?[\d\s\-\(\)\.]+$`)

	for i, contact := range allContacts {
		contactContext := fmt.Sprintf("contact %d", i+1)

		// Validate individual contact
		contactViolations := v.validateSingleContact(contact, contactContext, seenNames)
		violations = append(violations, contactViolations...)

		// Validate phone numbers and detect duplicates
		numberViolations := v.validateContactNumbers(contact, i, contactContext, seenNumbers, phonePattern)
		violations = append(violations, numberViolations...)
	}

	return violations
}

// validateSingleContact validates a single contact's basic properties
func (v *ContactsValidatorImpl) validateSingleContact(
	contact *contacts.Contact,
	contactContext string,
	seenNames map[string]bool,
) []Violation {
	var violations []Violation

	// Validate contact name
	if contact.Name == "" {
		violations = append(violations, Violation{
			Type:     InvalidFormat,
			Severity: SeverityError,
			File:     contactContext,
			Message:  "Contact name cannot be empty",
		})
		// Continue processing to check phone numbers even for invalid names
	}

	// Check for duplicate contact names
	if seenNames[contact.Name] {
		violations = append(violations, Violation{
			Type:     InvalidFormat,
			Severity: SeverityError,
			File:     contactContext,
			Message:  fmt.Sprintf("Duplicate contact name: %s", contact.Name),
		})
	}
	seenNames[contact.Name] = true

	// Check if contact has phone numbers
	if len(contact.Numbers) == 0 {
		violations = append(violations, Violation{
			Type:     InvalidFormat,
			Severity: SeverityWarning,
			File:     contactContext,
			Message:  fmt.Sprintf("Contact '%s' has no phone numbers", contact.Name),
		})
	}

	return violations
}

// validateContactNumbers validates phone numbers and detects duplicates
func (v *ContactsValidatorImpl) validateContactNumbers(
	contact *contacts.Contact,
	contactIndex int,
	contactContext string,
	seenNumbers map[string]string,
	phonePattern *regexp.Regexp,
) []Violation {
	var violations []Violation

	for j, number := range contact.Numbers {
		numberContext := fmt.Sprintf("%s number %d", contactContext, j+1)

		// Validate phone number format
		if number == "" {
			violations = append(violations, Violation{
				Type:     InvalidFormat,
				Severity: SeverityError,
				File:     numberContext,
				Message:  fmt.Sprintf("Empty phone number for contact '%s'", contact.Name),
			})
			continue
		}

		// Basic phone number format validation
		if !phonePattern.MatchString(number) {
			violations = append(violations, Violation{
				Type:     InvalidFormat,
				Severity: SeverityWarning,
				File:     numberContext,
				Message:  fmt.Sprintf("Phone number '%s' has unusual format for contact '%s'", number, contact.Name),
			})
		}

		// Check for duplicate phone numbers across contacts
		contactName := contact.Name
		if contactName == "" {
			contactName = fmt.Sprintf("(unnamed contact %d)", contactIndex+1)
		}

		if existingContact, exists := seenNumbers[number]; exists {
			violations = append(violations, Violation{
				Type:     InvalidFormat,
				Severity: SeverityError,
				File:     numberContext,
				Message: fmt.Sprintf("Phone number '%s' assigned to multiple contacts: '%s' and '%s'",
					number, existingContact, contactName),
			})
		} else {
			seenNumbers[number] = contactName
		}
	}

	return violations
}

// ValidateContactReferences checks contact references in calls/SMS against contacts data
func (v *ContactsValidatorImpl) ValidateContactReferences(callContacts, smsContacts map[string]bool) []Violation {
	var violations []Violation

	// Load contacts first
	err := v.contactsReader.LoadContacts()
	if err != nil {
		// If contacts can't be loaded, we can't validate references
		// This is not necessarily an error if contacts.yaml doesn't exist
		return violations
	}

	// Combine all referenced contact names
	allReferencedContacts := make(map[string]bool)
	for contact := range callContacts {
		allReferencedContacts[contact] = true
	}
	for contact := range smsContacts {
		allReferencedContacts[contact] = true
	}

	// Check each referenced contact
	for contactName := range allReferencedContacts {
		// Skip empty contact names and special values
		if contactName == "" || contactName == "(Unknown)" || contactName == "null" {
			continue
		}

		// Check if contact exists in contacts.yaml
		if !v.contactsReader.ContactExists(contactName) {
			// Determine severity based on whether we have any contacts loaded
			severity := SeverityWarning
			if v.contactsReader.GetContactsCount() > 0 {
				// If we have contacts but this one is missing, it's more concerning
				severity = SeverityError
			}

			violations = append(violations, Violation{
				Type:     MissingFile, // Using MissingFile for missing contact reference
				Severity: severity,
				File:     "contacts.yaml",
				Message:  fmt.Sprintf("Contact '%s' referenced in calls/SMS but not found in contacts.yaml", contactName),
			})
		}
	}

	// Note: Unused contacts are acceptable and should not be flagged as violations.
	// Users may maintain contacts that are not currently active in their data.

	return violations
}
