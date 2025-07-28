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
	var violations []ValidationViolation
	
	// Get available years from reader first
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
	
	// If no years available, calls directory is optional
	if len(years) == 0 {
		return violations
	}
	
	// Check if calls directory exists (only if we have years)
	callsDir := filepath.Join(v.repositoryRoot, "calls")
	if !dirExists(callsDir) {
		violations = append(violations, ValidationViolation{
			Type:     StructureViolation,
			Severity: SeverityError,
			File:     "calls/",
			Message:  "Required calls directory not found",
		})
		return violations
	}
	
	// Validate each year file exists and has correct naming
	for _, year := range years {
		expectedFileName := fmt.Sprintf("calls-%d.xml", year)
		expectedPath := filepath.Join(callsDir, expectedFileName)
		
		if !fileExists(expectedPath) {
			violations = append(violations, ValidationViolation{
				Type:     MissingFile,
				Severity: SeverityError,
				File:     filepath.Join("calls", expectedFileName),
				Message:  fmt.Sprintf("Call file for year %d not found: %s", year, expectedFileName),
			})
		}
	}
	
	return violations
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
	err := v.callsReader.StreamCalls(year, func(call calls.Call) error {
		callYear := call.Date.UTC().Year()
		if callYear != year {
			violations = append(violations, ValidationViolation{
				Type:     InvalidFormat,
				Severity: SeverityError,
				File:     filePath,
				Message:  fmt.Sprintf("Call dated %s belongs to year %d but found in %d file", 
					call.Date.Format("2006-01-02"), callYear, year),
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
	
	// Check counts for each year
	for _, year := range years {
		fileName := fmt.Sprintf("calls-%d.xml", year)
		filePath := filepath.Join("calls", fileName)
		
		// Get actual count from reader
		actualCount, err := v.callsReader.GetCallsCount(year)
		if err != nil {
			violations = append(violations, ValidationViolation{
				Type:     CountMismatch,
				Severity: SeverityError,
				File:     filePath,
				Message:  fmt.Sprintf("Failed to get call count for year %d: %v", year, err),
			})
			continue
		}
		
		// Check against expected count if provided
		if expectedCount, exists := expectedCounts[year]; exists {
			if actualCount != expectedCount {
				violations = append(violations, ValidationViolation{
					Type:     CountMismatch,
					Severity: SeverityError,
					File:     filePath,
					Message:  fmt.Sprintf("Call count mismatch for year %d", year),
					Expected: fmt.Sprintf("%d", expectedCount),
					Actual:   fmt.Sprintf("%d", actualCount),
				})
			}
		}
	}
	
	// Check for expected years that don't exist
	for expectedYear := range expectedCounts {
		found := false
		for _, year := range years {
			if year == expectedYear {
				found = true
				break
			}
		}
		
		if !found {
			violations = append(violations, ValidationViolation{
				Type:     MissingFile,
				Severity: SeverityError,
				File:     fmt.Sprintf("calls/calls-%d.xml", expectedYear),
				Message:  fmt.Sprintf("Expected call file for year %d not found", expectedYear),
			})
		}
	}
	
	return violations
}