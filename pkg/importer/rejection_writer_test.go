package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestXMLRejectionWriter_NoDirectoryCreatedOnInit(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create rejection writer
	_ = NewXMLRejectionWriter(tempDir)

	// Verify rejected directory was NOT created
	rejectedDir := filepath.Join(tempDir, "rejected")
	if _, err := os.Stat(rejectedDir); err == nil {
		t.Error("Rejected directory should not be created on initialization")
	}
}

func TestXMLRejectionWriter_DirectoryCreatedOnFirstRejection(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "calls-test.xml")
	if err := os.WriteFile(testFile, []byte("test data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create rejection writer
	writer := NewXMLRejectionWriter(tempDir)

	// Write rejections
	rejections := []RejectedEntry{
		{
			Line:       1,
			Data:       `<call number="123" date="" />`,
			Violations: []string{"missing-timestamp"},
		},
	}

	rejFile, err := writer.WriteRejections(testFile, rejections)
	if err != nil {
		t.Fatalf("Failed to write rejections: %v", err)
	}

	// Verify rejected directory structure was created
	rejectedDir := filepath.Join(tempDir, "rejected")
	if _, err := os.Stat(rejectedDir); err != nil {
		t.Error("Rejected directory should be created when writing rejections")
	}

	// Verify subdirectories
	for _, subdir := range []string{"calls", "sms"} {
		path := filepath.Join(rejectedDir, subdir)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("Rejected subdirectory %s should be created", subdir)
		}
	}

	// Verify rejection file exists
	if rejFile == "" {
		t.Error("Rejection file path should not be empty")
	}
	if _, err := os.Stat(rejFile); err != nil {
		t.Errorf("Rejection file should exist at %s", rejFile)
	}
}

func TestXMLRejectionWriter_EmptyRejectionsNoDirectory(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "calls-test.xml")
	if err := os.WriteFile(testFile, []byte("test data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create rejection writer
	writer := NewXMLRejectionWriter(tempDir)

	// Write empty rejections
	rejFile, err := writer.WriteRejections(testFile, []RejectedEntry{})
	if err != nil {
		t.Fatalf("Failed to write empty rejections: %v", err)
	}

	// Should return empty path
	if rejFile != "" {
		t.Error("Empty rejections should return empty path")
	}

	// Verify rejected directory was NOT created
	rejectedDir := filepath.Join(tempDir, "rejected")
	if _, err := os.Stat(rejectedDir); err == nil {
		t.Error("Rejected directory should not be created for empty rejections")
	}
}

func TestXMLRejectionWriter_TypeSubdirectories(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create rejection writer
	writer := NewXMLRejectionWriter(tempDir)

	tests := []struct {
		filename    string
		expectedDir string
	}{
		{"calls-2014.xml", "calls"},
		{"sms-2014.xml", "sms"},
		{"unknown.xml", ""}, // Root rejected directory
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			// Create test file
			testFile := filepath.Join(tempDir, tt.filename)
			if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
				t.Fatal(err)
			}

			// Write rejections
			rejections := []RejectedEntry{{Line: 1, Data: "<test />", Violations: []string{"test"}}}
			rejFile, err := writer.WriteRejections(testFile, rejections)
			if err != nil {
				t.Fatalf("Failed to write rejections: %v", err)
			}

			// Verify file is in correct subdirectory
			if tt.expectedDir != "" {
				expectedPath := filepath.Join(tempDir, "rejected", tt.expectedDir)
				if !strings.Contains(rejFile, expectedPath) {
					t.Errorf("Expected rejection file in %s, got %s", expectedPath, rejFile)
				}
			}
		})
	}
}

func TestXMLRejectionWriter_ConcurrentWrites(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create rejection writer
	writer := NewXMLRejectionWriter(tempDir)

	// Run concurrent writes
	done := make(chan bool, 2)

	go func() {
		testFile := filepath.Join(tempDir, "calls-1.xml")
		_ = os.WriteFile(testFile, []byte("test"), 0644)
		rejections := []RejectedEntry{{Line: 1, Data: "<call />", Violations: []string{"test"}}}
		_, err := writer.WriteRejections(testFile, rejections)
		if err != nil {
			t.Errorf("Concurrent write 1 failed: %v", err)
		}
		done <- true
	}()

	go func() {
		testFile := filepath.Join(tempDir, "sms-1.xml")
		_ = os.WriteFile(testFile, []byte("test"), 0644)
		rejections := []RejectedEntry{{Line: 1, Data: "<sms />", Violations: []string{"test"}}}
		_, err := writer.WriteRejections(testFile, rejections)
		if err != nil {
			t.Errorf("Concurrent write 2 failed: %v", err)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Verify directory was created only once
	rejectedDir := filepath.Join(tempDir, "rejected")
	if _, err := os.Stat(rejectedDir); err != nil {
		t.Error("Rejected directory should be created")
	}
}

func TestXMLRejectionWriter_FileFormat(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "calls-test.xml")
	if err := os.WriteFile(testFile, []byte("test data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create rejection writer
	writer := NewXMLRejectionWriter(tempDir)

	// Write rejections
	rejections := []RejectedEntry{
		{
			Line:       1,
			Data:       `<call number="123" date="" />`,
			Violations: []string{"missing-timestamp"},
		},
		{
			Line:       2,
			Data:       `<call number="456" type="invalid" />`,
			Violations: []string{"invalid-type"},
		},
	}

	rejFile, err := writer.WriteRejections(testFile, rejections)
	if err != nil {
		t.Fatalf("Failed to write rejections: %v", err)
	}

	// Read rejection file
	content, err := os.ReadFile(rejFile)
	if err != nil {
		t.Fatalf("Failed to read rejection file: %v", err)
	}

	// Verify XML format
	contentStr := string(content)
	if !strings.Contains(contentStr, `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>`) {
		t.Error("Rejection file should have XML header")
	}

	// Debug: print actual content
	t.Logf("Rejection file content:\n%s", contentStr)

	if !strings.Contains(contentStr, `<calls count="2">`) {
		t.Error("Rejection file should have root element with count")
	}

	// Verify original XML is preserved
	if !strings.Contains(contentStr, `<call number="123" date="" />`) {
		t.Error("Rejection file should contain original XML")
	}

	// Verify file name format (should contain hash and timestamp)
	baseName := filepath.Base(rejFile)
	if !strings.HasPrefix(baseName, "calls-test-") {
		t.Errorf("Rejection file should start with original name, got %s", baseName)
	}
	if !strings.HasSuffix(baseName, ".xml") {
		t.Errorf("Rejection file should end with .xml, got %s", baseName)
	}

	// Should have hash (8 chars) and timestamp in name
	parts := strings.Split(baseName, "-")
	if len(parts) < 4 { // calls-test-hash-timestamp.xml
		t.Errorf("Rejection file name should have hash and timestamp, got %s", baseName)
	}
}
