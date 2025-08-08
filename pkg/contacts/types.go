package contacts

// Contact represents a contact with associated phone numbers
type Contact struct {
	Name    string   `yaml:"name"`
	Numbers []string `yaml:"numbers"`
}

// ContactsData represents the YAML structure of contacts.yaml
type ContactsData struct {
	Contacts []*Contact `yaml:"contacts"`
}

// ContactsReader reads contact information from repository
type ContactsReader interface {
	// LoadContacts loads all contacts from contacts.yaml
	LoadContacts() error

	// GetContactByNumber returns contact name for a phone number
	GetContactByNumber(number string) (string, bool)

	// GetNumbersByContact returns all numbers for a contact name
	GetNumbersByContact(name string) ([]string, bool)

	// GetAllContacts returns all contacts
	GetAllContacts() ([]*Contact, error)

	// ContactExists checks if a contact name exists
	ContactExists(name string) bool

	// IsKnownNumber checks if a number has a contact
	IsKnownNumber(number string) bool

	// GetContactsCount returns total number of contacts
	GetContactsCount() int
}
