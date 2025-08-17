package calls

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

// copyFile copies a file from src to dst, creating directories as needed
func copyFile(src, dst string) error {
	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(dst), 0750); err != nil {
		return err
	}

	srcFile, err := os.Open(src) // #nosec G304
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	dstFile, err := os.Create(dst) // #nosec G304
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func TestXMLCallsReader_Integration_WithTestData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	// Use existing test data from testdata/archive/calls.xml
	repoRoot := "../../testdata/archive"
	_ = NewXMLCallsReader(repoRoot)

	// We need to copy the test file to the expected location structure
	// Create a temp directory and copy test data to expected structure
	tempDir := t.TempDir()
	callsDir := filepath.Join(tempDir, "calls")
	err := copyFile("../../testdata/archive/calls.xml", filepath.Join(callsDir, "calls-2014.xml"))
	if err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}

	reader := NewXMLCallsReader(tempDir)

	// Test ReadCalls with real data
	calls, err := reader.ReadCalls(2014)
	if err != nil {
		t.Fatalf("ReadCalls failed with real data: %v", err)
	}

	// The test file has 16 calls
	if len(calls) != 16 {
		t.Fatalf("Expected 16 calls from test data, got %d", len(calls))
	}

	// Test some specific calls from the test data
	// First call should be a missed call
	firstCall := calls[0]
	if firstCall.Type != Missed {
		t.Errorf("Expected first call to be missed call, got type %d", firstCall.Type)
	}

	// Test that dates are properly converted
	for i, call := range calls {
		if call.Date == 0 {
			t.Errorf("Call %d has zero date", i)
		}
		timestamp := call.Timestamp()
		if timestamp.Year() < 2014 || timestamp.Year() > 2015 {
			t.Errorf("Call %d has unexpected year %d", i, timestamp.Year())
		}
	}

	// Test streaming with real data
	var streamedCount int
	err = reader.StreamCallsForYear(2014, func(_ Call) error {
		streamedCount++
		return nil
	})
	if err != nil {
		t.Fatalf("StreamCalls failed with real data: %v", err)
	}

	if streamedCount != 16 {
		t.Errorf("Expected to stream 16 calls, got %d", streamedCount)
	}
}

func TestXMLCallsReader_Integration_LargeFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	// Use the larger test file from testdata/to_process/00/calls-test.xml
	tempDir := t.TempDir()
	callsDir := filepath.Join(tempDir, "calls")
	err := copyFile("../../testdata/to_process/00/calls-test.xml", filepath.Join(callsDir, "calls-2014.xml"))
	if err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}

	reader := NewXMLCallsReader(tempDir)

	// Test GetCallsCount with larger file - note the count attribute is wrong in test data
	count, err := reader.GetCallsCount(2014)
	if err != nil {
		t.Fatalf("GetCallsCount failed: %v", err)
	}

	// The test file says count="56" but actually has only 12 calls
	if count != 56 {
		t.Fatalf("Expected 56 calls (from count attribute) in test file, got %d", count)
	}

	// Test validation with larger file - this should fail due to count mismatch
	err = reader.ValidateCallsFile(2014)
	if err == nil {
		t.Fatalf("ValidateCallsFile should have failed due to count mismatch in test file")
	}

	// Test streaming for memory efficiency
	var streamedCalls []Call
	err = reader.StreamCallsForYear(2014, func(call Call) error {
		streamedCalls = append(streamedCalls, call)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamCalls failed: %v", err)
	}

	if len(streamedCalls) != 12 {
		t.Errorf("Expected to stream 12 actual calls, got %d", len(streamedCalls))
	}

	// Verify some properties of the streamed calls
	var incomingCount, outgoingCount, missedCount int
	for _, call := range streamedCalls {
		switch call.Type {
		case Incoming:
			incomingCount++
		case Outgoing:
			outgoingCount++
		case Missed:
			missedCount++
		}

		// Verify all calls have valid phone numbers
		if call.Number == "" {
			t.Error("Found call with empty phone number")
		}

		// Verify contact names are present (even if "(Unknown)")
		if call.ContactName == "" {
			t.Error("Found call with empty contact name")
		}
	}

	// Should have a mix of call types
	if incomingCount == 0 && outgoingCount == 0 && missedCount == 0 {
		t.Error("No calls found with valid types")
	}
}

func TestXMLCallsReader_Integration_ScenarioData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	// Use integration test scenario data
	tempDir := t.TempDir()
	callsDir := filepath.Join(tempDir, "calls")

	// Copy the scenario test data
	err := copyFile("../../testdata/it/scenerio-00/to_process/00/calls-test.xml", filepath.Join(callsDir, "calls-2014.xml"))
	if err != nil {
		t.Fatalf("Failed to setup scenario test data: %v", err)
	}

	reader := NewXMLCallsReader(tempDir)

	// Test that available years includes 2014
	years, err := reader.GetAvailableYears()
	if err != nil {
		t.Fatalf("GetAvailableYears failed: %v", err)
	}

	if len(years) != 1 || years[0] != 2014 {
		t.Fatalf("Expected [2014], got %v", years)
	}

	// Test count matches the scenario data - note this is from count attribute
	count, err := reader.GetCallsCount(2014)
	if err != nil {
		t.Fatalf("GetCallsCount failed: %v", err)
	}

	if count != 56 {
		t.Fatalf("Expected 56 calls (from count attribute) in scenario data, got %d", count)
	}

	// Test reading all calls
	calls, err := reader.ReadCalls(2014)
	if err != nil {
		t.Fatalf("ReadCalls failed: %v", err)
	}

	if len(calls) != 12 {
		t.Errorf("Expected 12 actual calls, got %d", len(calls))
	}

	// Check year distribution in the scenario data
	var year2014Count, year2015Count int
	for _, call := range calls {
		timestamp := call.Timestamp()
		switch timestamp.Year() {
		case 2014:
			year2014Count++
		case 2015:
			year2015Count++
		default:
			t.Errorf("Unexpected year %d in call data", timestamp.Year())
		}
	}

	// Verify we have calls from both years (this is actually mixed year data)
	if year2014Count == 0 {
		t.Error("Expected some calls from 2014")
	}
	if year2015Count == 0 {
		t.Error("Expected some calls from 2015")
	}

	// Test validation fails due to count mismatch in the test data
	err = reader.ValidateCallsFile(2014)
	if err == nil {
		t.Errorf("ValidateCallsFile should have failed due to count mismatch in test data")
	}
}
