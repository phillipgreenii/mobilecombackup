package sms

import "context"

// Reader reads SMS/MMS records from repository
type Reader interface {
	ReadMessages(ctx context.Context, year int) ([]Message, error)
	StreamMessagesForYear(ctx context.Context, year int, callback func(Message) error) error
	GetAttachmentRefs(ctx context.Context, year int) ([]string, error)
	GetAllAttachmentRefs(ctx context.Context) (map[string]bool, error)
	GetAvailableYears(ctx context.Context) ([]int, error)
	GetMessageCount(ctx context.Context, year int) (int, error)
	ValidateSMSFile(ctx context.Context, year int) error
}
