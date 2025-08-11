package importer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phillipgreen/mobilecombackup/pkg/calls"
	"github.com/phillipgreen/mobilecombackup/pkg/sms"
	"gopkg.in/yaml.v3"
)

func TestGenerateSummaryFile(t *testing.T) {
	// Create temp directory for test repository
	tempDir := t.TempDir()
	
	// Create test data structure
	callsDir := filepath.Join(tempDir, "calls")
	smsDir := filepath.Join(tempDir, "sms")
	attachmentsDir := filepath.Join(tempDir, "attachments", "ab")
	
	if err := os.MkdirAll(callsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(smsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(attachmentsDir, 0755); err != nil {
		t.Fatal(err)
	}
	
	// Create sample call files
	calls2014 := []*calls.Call{
		{Number: "5551234567", Type: calls.Incoming, Date: 1420070400000},
		{Number: "5559876543", Type: calls.Outgoing, Date: 1420156800000},
	}
	
	calls2015 := []*calls.Call{
		{Number: "5551234567", Type: calls.Incoming, Date: 1451606400000},
		{Number: "5559876543", Type: calls.Outgoing, Date: 1451692800000},
		{Number: "5555555555", Type: calls.Missed, Date: 1451779200000},
	}
	
	// Write call files
	writer, err := calls.NewXMLCallsWriter(callsDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteCalls("calls-2014.xml", calls2014); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteCalls("calls-2015.xml", calls2015); err != nil {
		t.Fatal(err)
	}
	
	// Create sample SMS files
	sms2014 := []sms.Message{
		&sms.SMS{Address: "5551234567", Body: "Test 1", Date: 1420070400000, Type: 1},
		&sms.SMS{Address: "5559876543", Body: "Test 2", Date: 1420156800000, Type: 2},
		&sms.MMS{Address: "5555555555", Date: 1420243200000, MsgBox: 1},
	}
	
	// Write SMS files
	smsWriter, err := sms.NewXMLSMSWriter(smsDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := smsWriter.WriteMessages("sms-2014.xml", sms2014); err != nil {
		t.Fatal(err)
	}
	
	// Create sample attachment files
	attachmentFiles := []string{
		filepath.Join(attachmentsDir, "abc123"),
		filepath.Join(attachmentsDir, "def456"),
	}
	for _, file := range attachmentFiles {
		if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	
	// Generate summary
	if err := generateSummaryFile(tempDir); err != nil {
		t.Fatalf("Failed to generate summary: %v", err)
	}
	
	// Verify summary file exists
	summaryPath := filepath.Join(tempDir, "summary.yaml")
	if _, err := os.Stat(summaryPath); err != nil {
		t.Fatalf("Summary file not created: %v", err)
	}
	
	// Read and verify summary content
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("Failed to read summary file: %v", err)
	}
	
	var summary SummaryFile
	if err := yaml.Unmarshal(data, &summary); err != nil {
		t.Fatalf("Failed to parse summary YAML: %v", err)
	}
	
	// Verify statistics
	if summary.Statistics.TotalCalls != 5 {
		t.Errorf("Expected 5 total calls, got %d", summary.Statistics.TotalCalls)
	}
	
	if summary.Statistics.TotalSMS != 3 {
		t.Errorf("Expected 3 total SMS, got %d", summary.Statistics.TotalSMS)
	}
	
	if summary.Statistics.TotalAttachments != 2 {
		t.Errorf("Expected 2 total attachments, got %d", summary.Statistics.TotalAttachments)
	}
	
	// Verify years covered
	expectedYears := []int{2014, 2015}
	if len(summary.Statistics.YearsCovered) != len(expectedYears) {
		t.Errorf("Expected %d years, got %d", len(expectedYears), len(summary.Statistics.YearsCovered))
	}
	
	for i, year := range expectedYears {
		if i >= len(summary.Statistics.YearsCovered) || summary.Statistics.YearsCovered[i] != year {
			t.Errorf("Expected year %d at index %d", year, i)
		}
	}
	
	// Verify timestamp format
	if summary.LastUpdated == "" {
		t.Error("LastUpdated should not be empty")
	}
}

func TestCalculateRepositoryStats(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	
	// Test empty repository
	stats, err := calculateRepositoryStats(tempDir)
	if err != nil {
		t.Fatalf("Failed to calculate stats for empty repo: %v", err)
	}
	
	if stats.TotalCalls != 0 || stats.TotalSMS != 0 || stats.TotalAttachments != 0 {
		t.Error("Empty repository should have zero counts")
	}
	
	if len(stats.YearsCovered) != 0 {
		t.Error("Empty repository should have no years covered")
	}
}

func TestCountAttachmentFiles(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	attachmentsDir := filepath.Join(tempDir, "attachments")
	
	// Test non-existent directory
	_, err := countAttachmentFiles(attachmentsDir)
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
	
	// Create directory structure
	subDir := filepath.Join(attachmentsDir, "ab")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	
	// Test empty directory
	count, err := countAttachmentFiles(attachmentsDir)
	if err != nil {
		t.Fatalf("Failed to count empty directory: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 files, got %d", count)
	}
	
	// Add files
	files := []string{
		filepath.Join(subDir, "file1"),
		filepath.Join(subDir, "file2"),
		filepath.Join(attachmentsDir, ".gitkeep"), // Should be ignored
	}
	
	for _, file := range files {
		if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	
	// Count again
	count, err = countAttachmentFiles(attachmentsDir)
	if err != nil {
		t.Fatalf("Failed to count files: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 files (excluding .gitkeep), got %d", count)
	}
}