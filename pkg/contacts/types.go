package contacts

import "context"

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

// Data represents the YAML structure of contacts.yaml
type Data struct {
	Contacts    []*Contact         `yaml:"contacts"`
	Unprocessed []UnprocessedEntry `yaml:"unprocessed,omitempty"`
}

// Reader reads contact information from repository
type Reader interface {
	// Legacy methods (deprecated but maintained for backward compatibility)
	// These delegate to context versions with context.Background()
	LoadContacts() error
	GetContactByNumber(number string) (string, bool)
	GetNumbersByContact(name string) ([]string, bool)
	GetAllContacts() ([]*Contact, error)
	ContactExists(name string) bool
	IsKnownNumber(number string) bool
	GetContactsCount() int
	AddUnprocessedContacts(addresses, contactNames string) error
	GetUnprocessedEntries() []UnprocessedEntry

	// Context-aware methods
	// These are the preferred methods for new code
	LoadContactsContext(ctx context.Context) error
	GetAllContactsContext(ctx context.Context) ([]*Contact, error)
	AddUnprocessedContactsContext(ctx context.Context, addresses, contactNames string) error
}

// Writer handles writing contact information to repository
type Writer interface {
	// Legacy methods (deprecated but maintained for backward compatibility)
	// These delegate to context versions with context.Background()
	SaveContacts(path string) error

	// Context-aware methods
	// These are the preferred methods for new code
	SaveContactsContext(ctx context.Context, path string) error
}
