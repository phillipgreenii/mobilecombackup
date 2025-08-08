package calls

// CallsReader reads call records from repository
type CallsReader interface {
	// ReadCalls reads all calls from a specific year file
	ReadCalls(year int) ([]Call, error)

	// StreamCallsForYear streams calls from a year file for memory efficiency
	StreamCallsForYear(year int, callback func(Call) error) error

	// GetAvailableYears returns list of years with call data
	GetAvailableYears() ([]int, error)

	// GetCallsCount returns total number of calls for a year
	GetCallsCount(year int) (int, error)

	// ValidateCallsFile validates XML structure and year consistency
	ValidateCallsFile(year int) error
}
