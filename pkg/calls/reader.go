package calls

import "context"

// Reader reads call records from repository
type Reader interface {
	ReadCalls(ctx context.Context, year int) ([]Call, error)
	StreamCallsForYear(ctx context.Context, year int, callback func(Call) error) error
	GetAvailableYears(ctx context.Context) ([]int, error)
	GetCallsCount(ctx context.Context, year int) (int, error)
	ValidateCallsFile(ctx context.Context, year int) error
}
