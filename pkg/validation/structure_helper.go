package validation

import (
	"fmt"
	"path/filepath"
)

// YearBasedReader is a common interface for readers that have year-based data
type YearBasedReader interface {
	GetAvailableYears() ([]int, error)
}

// CountBasedReader extends YearBasedReader with count functionality
type CountBasedReader interface {
	YearBasedReader
	GetCount(year int) (int, error)
}

// StructureValidationConfig contains configuration for directory structure validation
type StructureValidationConfig struct {
	DirectoryName string // "calls" or "sms"
	FilePrefix    string // "calls" or "sms"
	ContentType   string // "call" or "SMS"
}

// ValidateDirectoryStructure validates directory structure for year-based data
func ValidateDirectoryStructure(
	reader YearBasedReader,
	repositoryRoot string,
	config StructureValidationConfig,
) []ValidationViolation {
	var violations []ValidationViolation

	// Get available years from reader first
	years, err := reader.GetAvailableYears()
	if err != nil {
		violations = append(violations, ValidationViolation{
			Type:     StructureViolation,
			Severity: SeverityError,
			File:     config.DirectoryName + "/",
			Message:  fmt.Sprintf("Failed to read available %s years: %v", config.ContentType, err),
		})
		return violations
	}

	// If no years available, directory is optional
	if len(years) == 0 {
		return violations
	}

	// Check if directory exists (only if we have years)
	dataDir := filepath.Join(repositoryRoot, config.DirectoryName)
	if !dirExists(dataDir) {
		violations = append(violations, ValidationViolation{
			Type:     StructureViolation,
			Severity: SeverityError,
			File:     config.DirectoryName + "/",
			Message:  fmt.Sprintf("Required %s directory not found", config.DirectoryName),
		})
		return violations
	}

	// Validate each year file exists and has correct naming
	for _, year := range years {
		expectedFileName := fmt.Sprintf("%s-%d.xml", config.FilePrefix, year)
		expectedPath := filepath.Join(dataDir, expectedFileName)

		if !fileExists(expectedPath) {
			violations = append(violations, ValidationViolation{
				Type:     MissingFile,
				Severity: SeverityError,
				File:     filepath.Join(config.DirectoryName, expectedFileName),
				Message:  fmt.Sprintf("%s file for year %d not found: %s", config.ContentType, year, expectedFileName),
			})
		}
	}

	return violations
}

// ValidateDataCounts verifies data counts match expected counts for year-based data
func ValidateDataCounts(
	reader CountBasedReader,
	expectedCounts map[int]int,
	config StructureValidationConfig,
) []ValidationViolation {
	var violations []ValidationViolation

	// Get available years
	years, err := reader.GetAvailableYears()
	if err != nil {
		violations = append(violations, ValidationViolation{
			Type:     StructureViolation,
			Severity: SeverityError,
			File:     config.DirectoryName + "/",
			Message:  fmt.Sprintf("Failed to read available %s years: %v", config.ContentType, err),
		})
		return violations
	}

	// Check counts for each year
	for _, year := range years {
		fileName := fmt.Sprintf("%s-%d.xml", config.FilePrefix, year)
		filePath := filepath.Join(config.DirectoryName, fileName)

		// Get actual count from reader
		actualCount, err := reader.GetCount(year)
		if err != nil {
			violations = append(violations, ValidationViolation{
				Type:     CountMismatch,
				Severity: SeverityError,
				File:     filePath,
				Message:  fmt.Sprintf("Failed to get %s count for year %d: %v", config.ContentType, year, err),
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
					Message:  fmt.Sprintf("%s count mismatch for year %d", config.ContentType, year),
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
			fileName := fmt.Sprintf("%s-%d.xml", config.FilePrefix, expectedYear)
			filePath := filepath.Join(config.DirectoryName, fileName)
			violations = append(violations, ValidationViolation{
				Type:     MissingFile,
				Severity: SeverityError,
				File:     filePath,
				Message:  fmt.Sprintf("Expected %s file for year %d not found", config.ContentType, expectedYear),
			})
		}
	}

	return violations
}
