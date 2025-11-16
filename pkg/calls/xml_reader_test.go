package calls

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestXMLCallsReader_ReadCalls(t *testing.T) {
	// Create a temporary test file
	tempDir := t.TempDir()
	repoRoot := tempDir
	callsDir := filepath.Join(repoRoot, "calls")
	err := os.MkdirAll(callsDir, 0750)
	if err != nil {
		t.Fatalf("Failed to create calls directory: %v", err)
	}

	// Create test XML content
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<calls count="3">
  <call number="5555550000" duration="0" date="1410881505425" type="3" readable_date="Sep 16, 2014 11:31:45 AM" contact_name="(Unknown)" />
  <call number="+15555550001" duration="43" date="1411053850787" type="1" readable_date="Sep 18, 2014 11:24:10 AM" contact_name="John Stuart" />
  <call number="5555550002" duration="120" date="1411140251000" type="2" readable_date="Sep 19, 2014 11:24:11 AM" contact_name="Jane Doe" />
</calls>`

	testFile := filepath.Join(callsDir, "calls-2014.xml")
	err = os.WriteFile(testFile, []byte(testXML), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	reader := NewXMLCallsReader(repoRoot)

	// Test ReadCalls
	calls, err := reader.ReadCalls(context.Background(), 2014)
	if err != nil {
		t.Fatalf("ReadCalls failed: %v", err)
	}

	if len(calls) != 3 {
		t.Fatalf("Expected 3 calls, got %d", len(calls))
	}

	// Verify first call
	call := calls[0]
	if call.Number != "5555550000" {
		t.Errorf("Expected number '5555550000', got '%s'", call.Number)
	}
	if call.Duration != 0 {
		t.Errorf("Expected duration 0, got %d", call.Duration)
	}
	if call.Type != Missed {
		t.Errorf("Expected type %d, got %d", Missed, call.Type)
	}
	if call.ContactName != "(Unknown)" {
		t.Errorf("Expected contact name '(Unknown)', got '%s'", call.ContactName)
	}

	// Verify date conversion
	expectedTimeMs := int64(1410881505425)
	if call.Date != expectedTimeMs {
		t.Errorf("Expected date %d, got %d", expectedTimeMs, call.Date)
	}

	// Verify second call
	call = calls[1]
	if call.Number != "+15555550001" {
		t.Errorf("Expected number '+15555550001', got '%s'", call.Number)
	}
	if call.Duration != 43 {
		t.Errorf("Expected duration 43, got %d", call.Duration)
	}
	if call.Type != Incoming {
		t.Errorf("Expected type %d, got %d", Incoming, call.Type)
	}
	if call.ContactName != "John Stuart" {
		t.Errorf("Expected contact name 'John Stuart', got '%s'", call.ContactName)
	}

	// Verify third call
	call = calls[2]
	if call.Number != "5555550002" {
		t.Errorf("Expected number '5555550002', got '%s'", call.Number)
	}
	if call.Duration != 120 {
		t.Errorf("Expected duration 120, got %d", call.Duration)
	}
	if call.Type != Outgoing {
		t.Errorf("Expected type %d, got %d", Outgoing, call.Type)
	}
	if call.ContactName != "Jane Doe" {
		t.Errorf("Expected contact name 'Jane Doe', got '%s'", call.ContactName)
	}
}

func TestXMLCallsReader_StreamCalls(t *testing.T) {
	// Create a temporary test file
	tempDir := t.TempDir()
	repoRoot := tempDir
	callsDir := filepath.Join(repoRoot, "calls")
	err := os.MkdirAll(callsDir, 0750)
	if err != nil {
		t.Fatalf("Failed to create calls directory: %v", err)
	}

	// Create test XML content
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<calls count="2">
  <call number="1234567890" duration="30" date="1410881505425" type="1" readable_date="Sep 16, 2014 11:31:45 AM" contact_name="Test User" />
  <call number="0987654321" duration="60" date="1411053850787" type="2" readable_date="Sep 18, 2014 11:24:10 AM" contact_name="Another User" />
</calls>`

	testFile := filepath.Join(callsDir, "calls-2014.xml")
	err = os.WriteFile(testFile, []byte(testXML), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	reader := NewXMLCallsReader(repoRoot)

	var streamedCalls []Call
	err = reader.StreamCallsForYear(context.Background(), 2014, func(call Call) error {
		streamedCalls = append(streamedCalls, call)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamCalls failed: %v", err)
	}

	if len(streamedCalls) != 2 {
		t.Fatalf("Expected 2 calls, got %d", len(streamedCalls))
	}

	// Verify first call
	call := streamedCalls[0]
	if call.Number != "1234567890" {
		t.Errorf("Expected number '1234567890', got '%s'", call.Number)
	}
	if call.Duration != 30 {
		t.Errorf("Expected duration 30, got %d", call.Duration)
	}
	if call.Type != Incoming {
		t.Errorf("Expected type %d, got %d", Incoming, call.Type)
	}
}

func TestXMLCallsReader_GetAvailableYears(t *testing.T) {
	// Create a temporary test directory structure
	tempDir := t.TempDir()
	repoRoot := tempDir
	callsDir := filepath.Join(repoRoot, "calls")
	err := os.MkdirAll(callsDir, 0750)
	if err != nil {
		t.Fatalf("Failed to create calls directory: %v", err)
	}

	// Create multiple year files
	years := []int{2014, 2015, 2016}
	for _, year := range years {
		testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<calls count="0">
</calls>`
		fileName := filepath.Join(callsDir, fmt.Sprintf("calls-%d.xml", year))
		err = os.WriteFile(fileName, []byte(testXML), 0600)
		if err != nil {
			t.Fatalf("Failed to create test file for year %d: %v", year, err)
		}
	}

	// Create a non-call file that should be ignored
	err = os.WriteFile(filepath.Join(callsDir, "sms-2014.xml"), []byte("dummy"), 0600)
	if err != nil {
		t.Fatalf("Failed to create dummy file: %v", err)
	}

	reader := NewXMLCallsReader(repoRoot)

	availableYears, err := reader.GetAvailableYears(context.Background())
	if err != nil {
		t.Fatalf("GetAvailableYears failed: %v", err)
	}

	if len(availableYears) != len(years) {
		t.Fatalf("Expected %d years, got %d", len(years), len(availableYears))
	}

	for i, expectedYear := range years {
		if availableYears[i] != expectedYear {
			t.Errorf("Expected year %d at index %d, got %d", expectedYear, i, availableYears[i])
		}
	}
}

func TestXMLCallsReader_GetCallsCount(t *testing.T) {
	// Create a temporary test file
	tempDir := t.TempDir()
	repoRoot := tempDir
	callsDir := filepath.Join(repoRoot, "calls")
	err := os.MkdirAll(callsDir, 0750)
	if err != nil {
		t.Fatalf("Failed to create calls directory: %v", err)
	}

	// Create test XML content with count attribute
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<calls count="42">
  <call number="1234567890" duration="30" date="1410881505425" type="1" readable_date="Sep 16, 2014 11:31:45 AM" contact_name="Test User" />
</calls>`

	testFile := filepath.Join(callsDir, "calls-2014.xml")
	err = os.WriteFile(testFile, []byte(testXML), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	reader := NewXMLCallsReader(repoRoot)

	count, err := reader.GetCallsCount(context.Background(), 2014)
	if err != nil {
		t.Fatalf("GetCallsCount failed: %v", err)
	}

	if count != 42 {
		t.Errorf("Expected count 42, got %d", count)
	}
}

func TestXMLCallsReader_ValidateCallsFile(t *testing.T) {
	// Create a temporary test file
	tempDir := t.TempDir()
	repoRoot := tempDir
	callsDir := filepath.Join(repoRoot, "calls")
	err := os.MkdirAll(callsDir, 0750)
	if err != nil {
		t.Fatalf("Failed to create calls directory: %v", err)
	}

	// Test valid file
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<calls count="2">
  <call number="1234567890" duration="30" date="1410881505425" type="1" readable_date="Sep 16, 2014 11:31:45 AM" contact_name="Test User" />
  <call number="0987654321" duration="60" date="1411053850787" type="2" readable_date="Sep 18, 2014 11:24:10 AM" contact_name="Another User" />
</calls>`

	testFile := filepath.Join(callsDir, "calls-2014.xml")
	err = os.WriteFile(testFile, []byte(testXML), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	reader := NewXMLCallsReader(repoRoot)

	err = reader.ValidateCallsFile(context.Background(), 2014)
	if err != nil {
		t.Errorf("ValidateCallsFile failed on valid file: %v", err)
	}
}

func TestXMLCallsReader_ValidateCallsFile_CountMismatch(t *testing.T) {
	// Create a temporary test file
	tempDir := t.TempDir()
	repoRoot := tempDir
	callsDir := filepath.Join(repoRoot, "calls")
	err := os.MkdirAll(callsDir, 0750)
	if err != nil {
		t.Fatalf("Failed to create calls directory: %v", err)
	}

	// Test file with count mismatch
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<calls count="3">
  <call number="1234567890" duration="30" date="1410881505425" type="1" readable_date="Sep 16, 2014 11:31:45 AM" contact_name="Test User" />
  <call number="0987654321" duration="60" date="1411053850787" type="2" readable_date="Sep 18, 2014 11:24:10 AM" contact_name="Another User" />
</calls>`

	testFile := filepath.Join(callsDir, "calls-2014.xml")
	err = os.WriteFile(testFile, []byte(testXML), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	reader := NewXMLCallsReader(repoRoot)

	err = reader.ValidateCallsFile(context.Background(), 2014)
	if err == nil {
		t.Error("ValidateCallsFile should have failed on count mismatch")
	}
}

func TestXMLCallsReader_MissingFile(t *testing.T) {
	tempDir := t.TempDir()
	reader := NewXMLCallsReader(tempDir)

	// Test reading non-existent file
	_, err := reader.ReadCalls(context.Background(), 2014)
	if err == nil {
		t.Error("ReadCalls should have failed on missing file")
	}

	err = reader.StreamCallsForYear(context.Background(), 2014, func(Call) error { return nil })
	if err == nil {
		t.Error("StreamCalls should have failed on missing file")
	}

	_, err = reader.GetCallsCount(context.Background(), 2014)
	if err == nil {
		t.Error("GetCallsCount should have failed on missing file")
	}

	err = reader.ValidateCallsFile(context.Background(), 2014)
	if err == nil {
		t.Error("ValidateCallsFile should have failed on missing file")
	}
}

func TestXMLCallsReader_EmptyCallsDirectory(t *testing.T) {
	tempDir := t.TempDir()
	reader := NewXMLCallsReader(tempDir)

	// Test with non-existent calls directory
	years, err := reader.GetAvailableYears(context.Background())
	if err != nil {
		t.Fatalf("GetAvailableYears should handle missing directory: %v", err)
	}

	if len(years) != 0 {
		t.Errorf("Expected empty years list, got %v", years)
	}
}

func TestCallTypeConstants(t *testing.T) {
	t.Parallel()

	// Test that the call type constants match the expected values
	if Incoming != 1 {
		t.Errorf("Expected Incoming to be 1, got %d", Incoming)
	}
	if Outgoing != 2 {
		t.Errorf("Expected Outgoing to be 2, got %d", Outgoing)
	}
	if Missed != 3 {
		t.Errorf("Expected Missed to be 3, got %d", Missed)
	}
	if Voicemail != 4 {
		t.Errorf("Expected Voicemail to be 4, got %d", Voicemail)
	}
}
