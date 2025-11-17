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
	LoadContacts(ctx context.Context) error
	GetContactByNumber(number string) (string, bool)
	GetNumbersByContact(name string) ([]string, bool)
	GetAllContacts(ctx context.Context) ([]*Contact, error)
	ContactExists(name string) bool
	IsKnownNumber(number string) bool
	GetContactsCount() int
	AddUnprocessedContacts(ctx context.Context, addresses, contactNames string) error
	GetUnprocessedEntries() []UnprocessedEntry
}

// Writer handles writing contact information to repository
type Writer interface {
	SaveContacts(ctx context.Context, path string) error
}
