package validation

import (
	"fmt"
	"path/filepath"

	"github.com/phillipgreen/mobilecombackup/pkg/calls"
)

// CallsValidator validates calls directory and files using CallsReader
type CallsValidator interface {
	// ValidateCallsStructure validates calls directory structure
	ValidateCallsStructure() []ValidationViolation

	// ValidateCallsContent validates calls file content and consistency
	ValidateCallsContent() []ValidationViolation

	// ValidateCallsCounts verifies call counts match manifest/summary
	ValidateCallsCounts(expectedCounts map[int]int) []ValidationViolation
}

// CallsValidatorImpl implements CallsValidator interface
type CallsValidatorImpl struct {
	repositoryRoot string
	callsReader    calls.CallsReader
}

// NewCallsValidator creates a new calls validator
func NewCallsValidator(repositoryRoot string, callsReader calls.CallsReader) CallsValidator {
	return &CallsValidatorImpl{
		repositoryRoot: repositoryRoot,
		callsReader:    callsReader,
	}
}

// ValidateCallsStructure validates calls directory structure
func (v *CallsValidatorImpl) ValidateCallsStructure() []ValidationViolation {
	adapter := NewCallsReaderAdapter(v.callsReader)
	config := StructureValidationConfig{
		DirectoryName: "calls",
		FilePrefix:    "calls",
		ContentType:   "call",
	}
	return ValidateDirectoryStructure(adapter, v.repositoryRoot, config)
}

// ValidateCallsContent validates calls file content and consistency
func (v *CallsValidatorImpl) ValidateCallsContent() []ValidationViolation {
	var violations []ValidationViolation

	// Get available years
	years, err := v.callsReader.GetAvailableYears()
	if err != nil {
		violations = append(violations, ValidationViolation{
			Type:     StructureViolation,
			Severity: SeverityError,
			File:     "calls/",
			Message:  fmt.Sprintf("Failed to read available call years: %v", err),
		})
		return violations
	}

	// Validate each year file
	for _, year := range years {
		fileName := fmt.Sprintf("calls-%d.xml", year)
		filePath := filepath.Join("calls", fileName)

		// Validate file structure and year consistency
		if err := v.callsReader.ValidateCallsFile(year); err != nil {
			violations = append(violations, ValidationViolation{
				Type:     InvalidFormat,
				Severity: SeverityError,
				File:     filePath,
				Message:  fmt.Sprintf("Call file validation failed: %v", err),
			})
			continue
		}

		// Validate that all calls in the file belong to the correct year
		violations = append(violations, v.validateCallsYearConsistency(year)...)
	}

	return violations
}

// validateCallsYearConsistency checks that all calls in a year file belong to that year
func (v *CallsValidatorImpl) validateCallsYearConsistency(year int) []ValidationViolation {
	var violations []ValidationViolation
	fileName := fmt.Sprintf("calls-%d.xml", year)
	filePath := filepath.Join("calls", fileName)

	// Stream calls to check year consistency without loading all into memory
	err := v.callsReader.StreamCallsForYear(year, func(call calls.Call) error {
		callYear := call.Timestamp().UTC().Year()
		if callYear != year {
			violations = append(violations, ValidationViolation{
				Type:     InvalidFormat,
				Severity: SeverityError,
				File:     filePath,
				Message: fmt.Sprintf("Call dated %s belongs to year %d but found in %d file",
					call.Timestamp().Format("2006-01-02"), callYear, year),
				Expected: fmt.Sprintf("year %d", year),
				Actual:   fmt.Sprintf("year %d", callYear),
			})
		}
		return nil
	})

	if err != nil {
		violations = append(violations, ValidationViolation{
			Type:     InvalidFormat,
			Severity: SeverityError,
			File:     filePath,
			Message:  fmt.Sprintf("Failed to stream calls for year consistency check: %v", err),
		})
	}

	return violations
}

// ValidateCallsCounts verifies call counts match manifest/summary
func (v *CallsValidatorImpl) ValidateCallsCounts(expectedCounts map[int]int) []ValidationViolation {
	adapter := NewCallsReaderAdapter(v.callsReader)
	config := StructureValidationConfig{
		DirectoryName: "calls",
		FilePrefix:    "calls",
		ContentType:   "call",
	}
	return ValidateDataCounts(adapter, expectedCounts, config)
}
