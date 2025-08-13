package contacts

// Contact represents a contact with associated phone numbers
type Contact struct {
	Name    string   `yaml:"name"`
	Numbers []string `yaml:"numbers"`
}

// UnprocessedEntry represents an unprocessed contact entry with structured data
type UnprocessedEntry struct {
	PhoneNumber  string   `yaml:"phone_number"`
	ContactNames []string `yaml:"contact_names"`
}

// ContactsData represents the YAML structure of contacts.yaml
type ContactsData struct {
	Contacts    []*Contact         `yaml:"contacts"`
	Unprocessed []UnprocessedEntry `yaml:"unprocessed,omitempty"`
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

	// AddUnprocessedContacts adds contacts from multi-address SMS parsing
	AddUnprocessedContacts(addresses, contactNames string) error

	// GetUnprocessedEntries returns all unprocessed entries in structured format
	GetUnprocessedEntries() []UnprocessedEntry
}

// ContactsWriter handles writing contact information to repository
type ContactsWriter interface {
	// SaveContacts writes the current state to contacts.yaml
	SaveContacts(path string) error
}
