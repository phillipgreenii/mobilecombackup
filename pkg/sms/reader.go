package sms

import "context"

// Reader reads SMS/MMS records from repository
type Reader interface {
	// Legacy methods (deprecated but maintained for backward compatibility)
	// These delegate to context versions with context.Background()
	ReadMessages(year int) ([]Message, error)
	StreamMessagesForYear(year int, callback func(Message) error) error
	GetAttachmentRefs(year int) ([]string, error)
	GetAllAttachmentRefs() (map[string]bool, error)
	GetAvailableYears() ([]int, error)
	GetMessageCount(year int) (int, error)
	ValidateSMSFile(year int) error

	// Context-aware methods
	// These are the preferred methods for new code
	ReadMessagesContext(ctx context.Context, year int) ([]Message, error)
	StreamMessagesForYearContext(ctx context.Context, year int, callback func(Message) error) error
	GetAttachmentRefsContext(ctx context.Context, year int) ([]string, error)
	GetAllAttachmentRefsContext(ctx context.Context) (map[string]bool, error)
	GetAvailableYearsContext(ctx context.Context) ([]int, error)
	GetMessageCountContext(ctx context.Context, year int) (int, error)
	ValidateSMSFileContext(ctx context.Context, year int) error
}
