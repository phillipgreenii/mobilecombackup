package validation

import (
	"context"
	"github.com/phillipgreenii/mobilecombackup/pkg/calls"
	"github.com/phillipgreenii/mobilecombackup/pkg/sms"
)

// CallsReaderAdapter adapts calls.Reader to CountBasedReader interface
type CallsReaderAdapter struct {
	reader calls.Reader
}

// NewCallsReaderAdapter creates a new CallsReaderAdapter
func NewCallsReaderAdapter(reader calls.Reader) *CallsReaderAdapter {
	return &CallsReaderAdapter{reader: reader}
}

// GetAvailableYears implements YearBasedReader interface
func (c *CallsReaderAdapter) GetAvailableYears(ctx context.Context) ([]int, error) {
	return c.reader.GetAvailableYears(ctx)
}

// GetCount implements CountBasedReader interface
func (c *CallsReaderAdapter) GetCount(year int) (int, error) {
	return c.reader.GetCallsCount(context.Background(), year)
}

// SMSReaderAdapter adapts sms.Reader to CountBasedReader interface
type SMSReaderAdapter struct {
	reader sms.Reader
}

// NewSMSReaderAdapter creates a new SMSReaderAdapter
func NewSMSReaderAdapter(reader sms.Reader) *SMSReaderAdapter {
	return &SMSReaderAdapter{reader: reader}
}

// GetAvailableYears implements YearBasedReader interface
func (s *SMSReaderAdapter) GetAvailableYears(ctx context.Context) ([]int, error) {
	return s.reader.GetAvailableYears(ctx)
}

// GetCount implements CountBasedReader interface
func (s *SMSReaderAdapter) GetCount(year int) (int, error) {
	return s.reader.GetMessageCount(context.Background(), year)
}
