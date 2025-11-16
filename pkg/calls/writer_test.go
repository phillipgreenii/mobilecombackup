package calls

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewXMLCallsWriter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		repoPath  string
		shouldErr bool
	}{
		{
			name:      "valid directory",
			repoPath:  filepath.Join(t.TempDir(), "calls"),
			shouldErr: false,
		},
		{
			name:      "nested directory",
			repoPath:  filepath.Join(t.TempDir(), "repo", "calls"),
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer, err := NewXMLCallsWriter(tt.repoPath)

			if tt.shouldErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if writer == nil {
				t.Fatal("Writer is nil")
			}

			// Verify directory was created
			if _, err := os.Stat(tt.repoPath); os.IsNotExist(err) {
				t.Errorf("Directory %s was not created", tt.repoPath)
			}
		})
	}
}

func TestXMLCallsWriter_WriteCalls(t *testing.T) {
	t.Parallel()

	repoPath := filepath.Join(t.TempDir(), "calls")
	writer, err := NewXMLCallsWriter(repoPath)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	calls := []*Call{
		{
			Number:   "1234567890",
			Duration: 120,
			Date:     1234567890000,
			Type:     Incoming,
		},
		{
			Number:   "0987654321",
			Duration: 60,
			Date:     1234567891000,
			Type:     Outgoing,
		},
	}

	filename := "2009_calls.xml"
	err = writer.WriteCalls(filename, calls)
	if err != nil {
		t.Fatalf("WriteCalls failed: %v", err)
	}

	// Verify file was created
	filePath := filepath.Join(repoPath, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("File %s was not created", filePath)
		return
	}

	// Read and verify file contents
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	contentStr := string(content)

	// Verify XML header
	if !strings.HasPrefix(contentStr, "<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>") {
		t.Error("XML header is missing or incorrect")
	}

	// Verify count attribute
	if !strings.Contains(contentStr, `count="2"`) {
		t.Error("Count attribute is missing or incorrect")
	}

	// Verify call numbers
	if !strings.Contains(contentStr, "1234567890") {
		t.Error("First call number not found in XML")
	}
	if !strings.Contains(contentStr, "0987654321") {
		t.Error("Second call number not found in XML")
	}

	// Verify readable_date was set (should be in EST)
	if !strings.Contains(contentStr, "readable_date=") {
		t.Error("readable_date attribute not found in XML")
	}

	// Parse XML to verify structure
	var root struct {
		XMLName xml.Name `xml:"calls"`
		Count   int      `xml:"count,attr"`
		Calls   []*Call  `xml:"call"`
	}
	err = xml.Unmarshal(content, &root)
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	if root.Count != 2 {
		t.Errorf("Count = %d, expected 2", root.Count)
	}
	if len(root.Calls) != 2 {
		t.Errorf("Calls count = %d, expected 2", len(root.Calls))
	}
}

func TestXMLCallsWriter_WriteCalls_Empty(t *testing.T) {
	t.Parallel()

	repoPath := filepath.Join(t.TempDir(), "calls")
	writer, err := NewXMLCallsWriter(repoPath)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	calls := []*Call{}

	filename := "2009_empty.xml"
	err = writer.WriteCalls(filename, calls)
	if err != nil {
		t.Fatalf("WriteCalls failed: %v", err)
	}

	// Verify file was created
	filePath := filepath.Join(repoPath, filename)
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// Parse XML
	var root struct {
		XMLName xml.Name `xml:"calls"`
		Count   int      `xml:"count,attr"`
		Calls   []*Call  `xml:"call"`
	}
	err = xml.Unmarshal(content, &root)
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	if root.Count != 0 {
		t.Errorf("Count = %d, expected 0", root.Count)
	}
	if len(root.Calls) != 0 {
		t.Errorf("Calls count = %d, expected 0", len(root.Calls))
	}
}

func TestXMLCallsWriter_WriteCalls_ReadableDateEST(t *testing.T) {
	t.Parallel()

	repoPath := filepath.Join(t.TempDir(), "calls")
	writer, err := NewXMLCallsWriter(repoPath)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	// Use a known timestamp: Feb 13, 2009 23:31:30 UTC
	// Should be Feb 13, 2009 6:31:30 PM EST
	calls := []*Call{
		{
			Number:   "1234567890",
			Duration: 120,
			Date:     1234567890000,
			Type:     Incoming,
		},
	}

	filename := "test.xml"
	err = writer.WriteCalls(filename, calls)
	if err != nil {
		t.Fatalf("WriteCalls failed: %v", err)
	}

	// Read file
	filePath := filepath.Join(repoPath, filename)
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// Parse XML
	var root struct {
		Calls []*Call `xml:"call"`
	}
	err = xml.Unmarshal(content, &root)
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	if len(root.Calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(root.Calls))
	}

	readableDate := root.Calls[0].ReadableDate

	// Verify format (should be in EST)
	// Expected format: "Feb 13, 2009 6:31:30 PM"
	if !strings.Contains(readableDate, "Feb 13, 2009") {
		t.Errorf("ReadableDate = %s, expected to contain 'Feb 13, 2009'", readableDate)
	}
	if !strings.Contains(readableDate, "PM") {
		t.Errorf("ReadableDate = %s, expected to contain 'PM'", readableDate)
	}
}

func TestXMLCallsWriter_WriteCalls_AllCallTypes(t *testing.T) {
	t.Parallel()

	repoPath := filepath.Join(t.TempDir(), "calls")
	writer, err := NewXMLCallsWriter(repoPath)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	calls := []*Call{
		{Number: "1111111111", Duration: 10, Date: 1000000000000, Type: Incoming},
		{Number: "2222222222", Duration: 20, Date: 2000000000000, Type: Outgoing},
		{Number: "3333333333", Duration: 0, Date: 3000000000000, Type: Missed},
		{Number: "4444444444", Duration: 30, Date: 4000000000000, Type: Voicemail},
	}

	filename := "all_types.xml"
	err = writer.WriteCalls(filename, calls)
	if err != nil {
		t.Fatalf("WriteCalls failed: %v", err)
	}

	// Verify all calls were written
	filePath := filepath.Join(repoPath, filename)
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var root struct {
		XMLName xml.Name `xml:"calls"`
		Count   int      `xml:"count,attr"`
		Calls   []*Call  `xml:"call"`
	}
	err = xml.Unmarshal(content, &root)
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	if root.Count != 4 {
		t.Errorf("Count = %d, expected 4", root.Count)
	}
	if len(root.Calls) != 4 {
		t.Errorf("Calls count = %d, expected 4", len(root.Calls))
	}

	// Verify each type
	types := map[CallType]bool{
		Incoming:  false,
		Outgoing:  false,
		Missed:    false,
		Voicemail: false,
	}

	for _, call := range root.Calls {
		types[call.Type] = true
	}

	for callType, found := range types {
		if !found {
			t.Errorf("CallType %d not found in output", callType)
		}
	}
}

func TestXMLCallsWriter_WriteCalls_SpecialCharacters(t *testing.T) {
	t.Parallel()

	repoPath := filepath.Join(t.TempDir(), "calls")
	writer, err := NewXMLCallsWriter(repoPath)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	calls := []*Call{
		{
			Number:      "+1 (234) 567-8900",
			Duration:    120,
			Date:        1234567890000,
			Type:        Incoming,
			ContactName: "O'Brien & Co. <test@example.com>",
		},
	}

	filename := "special.xml"
	err = writer.WriteCalls(filename, calls)
	if err != nil {
		t.Fatalf("WriteCalls failed: %v", err)
	}

	// Read and parse file
	filePath := filepath.Join(repoPath, filename)
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var root struct {
		Calls []*Call `xml:"call"`
	}
	err = xml.Unmarshal(content, &root)
	if err != nil {
		t.Fatalf("Failed to parse XML with special characters: %v", err)
	}

	if len(root.Calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(root.Calls))
	}

	// Verify special characters were preserved
	if root.Calls[0].Number != "+1 (234) 567-8900" {
		t.Errorf("Number = %q, expected '+1 (234) 567-8900'", root.Calls[0].Number)
	}
	expectedContact := "O'Brien & Co. <test@example.com>"
	if root.Calls[0].ContactName != expectedContact {
		t.Errorf("ContactName = %q, expected %q", root.Calls[0].ContactName, expectedContact)
	}
}

func TestXMLCallsWriter_WriteCalls_UpdatesReadableDate(t *testing.T) {
	t.Parallel()

	repoPath := filepath.Join(t.TempDir(), "calls")
	writer, err := NewXMLCallsWriter(repoPath)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	// Create call with existing readable_date that should be overwritten
	oldDate := "Old Date String"
	calls := []*Call{
		{
			Number:       "1234567890",
			Duration:     120,
			Date:         1234567890000,
			Type:         Incoming,
			ReadableDate: oldDate,
		},
	}

	filename := "test.xml"
	err = writer.WriteCalls(filename, calls)
	if err != nil {
		t.Fatalf("WriteCalls failed: %v", err)
	}

	// Verify readable_date was updated
	filePath := filepath.Join(repoPath, filename)
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var root struct {
		Calls []*Call `xml:"call"`
	}
	err = xml.Unmarshal(content, &root)
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	if len(root.Calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(root.Calls))
	}

	readableDate := root.Calls[0].ReadableDate

	// Should NOT be the old date
	if readableDate == oldDate {
		t.Error("ReadableDate was not updated")
	}

	// Should be in correct format
	if !strings.Contains(readableDate, "2009") {
		t.Errorf("ReadableDate = %s, expected to contain year 2009", readableDate)
	}
}

func TestXMLCallsWriter_WriteCalls_FilePermissions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping file permission test in short mode")
	}

	t.Parallel()

	repoPath := filepath.Join(t.TempDir(), "calls")
	writer, err := NewXMLCallsWriter(repoPath)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	calls := []*Call{
		{Number: "test", Duration: 10, Date: 1000000000000, Type: Incoming},
	}

	filename := "test.xml"
	err = writer.WriteCalls(filename, calls)
	if err != nil {
		t.Fatalf("WriteCalls failed: %v", err)
	}

	// Verify file was created and is readable
	filePath := filepath.Join(repoPath, filename)
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	if info.IsDir() {
		t.Error("Expected file, got directory")
	}

	// Verify we can read it
	_, err = os.ReadFile(filePath)
	if err != nil {
		t.Errorf("Failed to read created file: %v", err)
	}
}
