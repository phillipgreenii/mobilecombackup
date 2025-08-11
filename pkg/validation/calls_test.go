package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/calls"
)

// mockCallsReader implements CallsReader for testing
type mockCallsReader struct {
	availableYears []int
	calls          map[int][]calls.Call
	counts         map[int]int
	validateError  map[int]error
}

func (m *mockCallsReader) ReadCalls(year int) ([]calls.Call, error) {
	if callList, exists := m.calls[year]; exists {
		return callList, nil
	}
	return nil, fmt.Errorf("no calls for year %d", year)
}

func (m *mockCallsReader) StreamCallsForYear(year int, callback func(calls.Call) error) error {
	if callList, exists := m.calls[year]; exists {
		for _, call := range callList {
			if err := callback(call); err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("no calls for year %d", year)
}

func (m *mockCallsReader) GetAvailableYears() ([]int, error) {
	return m.availableYears, nil
}

func (m *mockCallsReader) GetCallsCount(year int) (int, error) {
	if count, exists := m.counts[year]; exists {
		return count, nil
	}
	return 0, fmt.Errorf("no count for year %d", year)
}

func (m *mockCallsReader) ValidateCallsFile(year int) error {
	if err, exists := m.validateError[year]; exists {
		return err
	}
	return nil
}

func TestCallsValidatorImpl_ValidateCallsStructure(t *testing.T) {
	tempDir := t.TempDir()

	mockReader := &mockCallsReader{
		availableYears: []int{2015, 2016},
	}

	validator := NewCallsValidator(tempDir, mockReader)

	// Test missing calls directory
	violations := validator.ValidateCallsStructure()
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation for missing calls directory, got %d", len(violations))
	}

	// Create calls directory
	callsDir := filepath.Join(tempDir, "calls")
	err := os.MkdirAll(callsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create calls directory: %v", err)
	}

	// Test missing call files
	violations = validator.ValidateCallsStructure()
	if len(violations) != 2 { // Missing both year files
		t.Errorf("Expected 2 violations for missing call files, got %d", len(violations))
	}

	// Create call files
	for _, year := range mockReader.availableYears {
		fileName := fmt.Sprintf("calls-%d.xml", year)
		filePath := filepath.Join(callsDir, fileName)
		err := os.WriteFile(filePath, []byte("<calls></calls>"), 0644)
		if err != nil {
			t.Fatalf("Failed to create call file: %v", err)
		}
	}

	// Test with all files present
	violations = validator.ValidateCallsStructure()
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations with all files present, got %d: %v", len(violations), violations)
	}
}

func TestCallsValidatorImpl_ValidateCallsContent(t *testing.T) {
	tempDir := t.TempDir()

	mockReader := &mockCallsReader{
		availableYears: []int{2015, 2016},
		calls: map[int][]calls.Call{
			2015: {
				{
					Number:       "+15551234567",
					Duration:     120,
					Date:         time.Date(2015, 6, 15, 14, 30, 0, 0, time.UTC).UnixMilli(),
					Type:         calls.Incoming,
					ReadableDate: "2015-06-15 14:30:00",
					ContactName:  "John Doe",
				},
				{
					Number:       "+15559876543",
					Duration:     45,
					Date:         time.Date(2016, 2, 10, 9, 15, 0, 0, time.UTC).UnixMilli(), // Wrong year!
					Type:         calls.Outgoing,
					ReadableDate: "2016-02-10 09:15:00",
					ContactName:  "Jane Smith",
				},
			},
			2016: {
				{
					Number:       "+15555555555",
					Duration:     300,
					Date:         time.Date(2016, 12, 25, 18, 0, 0, 0, time.UTC).UnixMilli(),
					Type:         calls.Missed,
					ReadableDate: "2016-12-25 18:00:00",
					ContactName:  "Bob Johnson",
				},
			},
		},
		validateError: map[int]error{
			2016: fmt.Errorf("validation failed for 2016"),
		},
	}

	validator := NewCallsValidator(tempDir, mockReader)

	violations := validator.ValidateCallsContent()

	// Should have violations for:
	// 1. Year consistency issue in 2015 file (call from 2016)
	// 2. Validation error for 2016 file
	if len(violations) < 2 {
		t.Errorf("Expected at least 2 violations, got %d: %v", len(violations), violations)
	}

	// Check for year consistency violation
	foundYearViolation := false
	foundValidationViolation := false

	for _, violation := range violations {
		if violation.File == "calls/calls-2015.xml" && violation.Type == InvalidFormat {
			foundYearViolation = true
		}
		if violation.File == "calls/calls-2016.xml" && violation.Type == InvalidFormat {
			foundValidationViolation = true
		}
	}

	if !foundYearViolation {
		t.Error("Expected year consistency violation for 2015 file")
	}

	if !foundValidationViolation {
		t.Error("Expected validation error violation for 2016 file")
	}
}

func TestCallsValidatorImpl_ValidateCallsCounts(t *testing.T) {
	tempDir := t.TempDir()

	mockReader := &mockCallsReader{
		availableYears: []int{2015, 2016},
		counts: map[int]int{
			2015: 25,
			2016: 30,
		},
	}

	validator := NewCallsValidator(tempDir, mockReader)

	// Test with matching counts
	expectedCounts := map[int]int{
		2015: 25,
		2016: 30,
	}

	violations := validator.ValidateCallsCounts(expectedCounts)
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations with matching counts, got %d: %v", len(violations), violations)
	}

	// Test with mismatched counts
	expectedCountsMismatch := map[int]int{
		2015: 20, // Wrong count
		2016: 30,
	}

	violations = validator.ValidateCallsCounts(expectedCountsMismatch)
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation with mismatched count, got %d: %v", len(violations), violations)
	}

	if len(violations) > 0 && violations[0].Type != CountMismatch {
		t.Errorf("Expected CountMismatch violation, got %s", violations[0].Type)
	}

	// Test with missing expected year
	expectedCountsExtra := map[int]int{
		2015: 25,
		2016: 30,
		2017: 10, // Year that doesn't exist
	}

	violations = validator.ValidateCallsCounts(expectedCountsExtra)
	if len(violations) != 1 {
		t.Errorf("Expected 1 violation for missing year, got %d: %v", len(violations), violations)
	}

	if len(violations) > 0 && violations[0].Type != MissingFile {
		t.Errorf("Expected MissingFile violation, got %s", violations[0].Type)
	}
}

func TestCallsValidatorImpl_ValidateCallsContent_ValidYear(t *testing.T) {
	tempDir := t.TempDir()

	// Test with all calls in correct years
	mockReader := &mockCallsReader{
		availableYears: []int{2015},
		calls: map[int][]calls.Call{
			2015: {
				{
					Number:       "+15551234567",
					Duration:     120,
					Date:         time.Date(2015, 6, 15, 14, 30, 0, 0, time.UTC).UnixMilli(),
					Type:         calls.Incoming,
					ReadableDate: "2015-06-15 14:30:00",
					ContactName:  "John Doe",
				},
				{
					Number:       "+15559876543",
					Duration:     45,
					Date:         time.Date(2015, 12, 31, 23, 59, 0, 0, time.UTC).UnixMilli(),
					Type:         calls.Outgoing,
					ReadableDate: "2015-12-31 23:59:00",
					ContactName:  "Jane Smith",
				},
			},
		},
	}

	validator := NewCallsValidator(tempDir, mockReader)

	violations := validator.ValidateCallsContent()
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations with correct years, got %d: %v", len(violations), violations)
	}
}

func TestCallsValidatorImpl_ErrorHandling(t *testing.T) {
	tempDir := t.TempDir()

	// Test with reader that returns errors
	mockReader := &mockCallsReader{
		availableYears: nil, // Will cause GetAvailableYears to return empty slice
	}

	// Override to return error
	originalReader := mockReader
	errorReader := &mockCallsReader{
		availableYears: []int{2015},
		counts:         map[int]int{},
	}

	// Test error in GetCallsCount
	validator := NewCallsValidator(tempDir, errorReader)
	violations := validator.ValidateCallsCounts(map[int]int{2015: 10})

	if len(violations) == 0 {
		t.Error("Expected violation when GetCallsCount returns error")
	}

	// Test with original empty reader
	validator = NewCallsValidator(tempDir, originalReader)
	violations = validator.ValidateCallsStructure()

	// Empty years should not cause violations (empty repository is valid)
	if len(violations) != 0 {
		t.Errorf("Expected 0 violations with empty years, got %d: %v", len(violations), violations)
	}
}
