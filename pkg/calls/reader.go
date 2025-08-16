package calls

import "context"

// Reader reads call records from repository
type Reader interface {
	// Legacy methods (deprecated but maintained for backward compatibility)
	// These delegate to context versions with context.Background()
	ReadCalls(year int) ([]Call, error)
	StreamCallsForYear(year int, callback func(Call) error) error
	GetAvailableYears() ([]int, error)
	GetCallsCount(year int) (int, error)
	ValidateCallsFile(year int) error

	// Context-aware methods
	// These are the preferred methods for new code
	ReadCallsContext(ctx context.Context, year int) ([]Call, error)
	StreamCallsForYearContext(ctx context.Context, year int, callback func(Call) error) error
	GetAvailableYearsContext(ctx context.Context) ([]int, error)
	GetCallsCountContext(ctx context.Context, year int) (int, error)
	ValidateCallsFileContext(ctx context.Context, year int) error
}
