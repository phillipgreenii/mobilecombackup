package sms

// SMSReader reads SMS/MMS records from repository
type SMSReader interface {
	// ReadMessages reads all messages from a specific year
	ReadMessages(year int) ([]Message, error)

	// StreamMessages streams messages for memory efficiency
	StreamMessages(year int, callback func(Message) error) error

	// GetAttachmentRefs returns all attachment references in a year
	GetAttachmentRefs(year int) ([]string, error)

	// GetAllAttachmentRefs returns all attachment references across all years
	GetAllAttachmentRefs() (map[string]bool, error)

	// GetAvailableYears returns list of years with SMS data
	GetAvailableYears() ([]int, error)

	// GetMessageCount returns total number of messages for a year
	GetMessageCount(year int) (int, error)

	// ValidateSMSFile validates XML structure and year consistency
	ValidateSMSFile(year int) error
}
