package importer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/phillipgreenii/mobilecombackup/pkg/attachments"
)

// Test mock implementations

func TestMockContactsManager(t *testing.T) {
	t.Parallel()

	t.Run("LoadContacts with error", func(t *testing.T) {
		mock := NewMockContactsManager()
		expectedErr := fmt.Errorf("load error")
		mock.SetLoadError(expectedErr)

		err := mock.LoadContacts(context.Background())
		if err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("LoadContacts without error", func(t *testing.T) {
		mock := NewMockContactsManager()

		err := mock.LoadContacts(context.Background())
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("SaveContacts with error", func(t *testing.T) {
		mock := NewMockContactsManager()
		expectedErr := fmt.Errorf("save error")
		mock.SetSaveError(expectedErr)

		err := mock.SaveContacts(context.Background(), "/test/path")
		if err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("SaveContacts without error", func(t *testing.T) {
		mock := NewMockContactsManager()

		err := mock.SaveContacts(context.Background(), "/test/path")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("AddUnprocessedContacts", func(t *testing.T) {
		mock := NewMockContactsManager()

		err := mock.AddUnprocessedContacts(context.Background(), "+1234567890", "John Doe")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify it was added
		if len(mock.unprocessedEntries) != 1 {
			t.Errorf("Expected 1 unprocessed entry, got %d", len(mock.unprocessedEntries))
		}
	})

	t.Run("AddUnprocessedContact", func(t *testing.T) {
		mock := NewMockContactsManager()

		mock.AddUnprocessedContact("+1234567890", "John Doe")
		mock.AddUnprocessedContact("+1234567890", "Jane Doe")

		// Verify both were added
		entries := mock.unprocessedEntries["+1234567890"]
		if len(entries) != 2 {
			t.Errorf("Expected 2 entries, got %d", len(entries))
		}
	})
}

func TestMockAttachmentStorage(t *testing.T) {
	t.Parallel()

	t.Run("Store without error", func(t *testing.T) {
		mock := NewMockAttachmentStorage()
		data := []byte("test data")
		metadata := attachments.AttachmentInfo{
			MimeType: "image/jpeg",
			Size:     int64(len(data)),
		}

		err := mock.Store("abc123", data, metadata)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify stored
		stored, exists := mock.GetStoredData("abc123")
		if !exists {
			t.Error("Expected data to be stored")
		}
		if string(stored) != string(data) {
			t.Errorf("Expected data %s, got %s", data, stored)
		}
	})

	t.Run("Store with error", func(t *testing.T) {
		mock := NewMockAttachmentStorage()
		expectedErr := fmt.Errorf("store error")
		mock.SetStoreError(expectedErr)

		err := mock.Store("abc123", []byte("data"), attachments.AttachmentInfo{})
		if err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("GetPath without error", func(t *testing.T) {
		mock := NewMockAttachmentStorage()
		if err := mock.Store("abc123", []byte("data"), attachments.AttachmentInfo{}); err != nil {
			t.Fatal(err)
		}

		path, err := mock.GetPath("abc123")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if path != "attachments/ab/abc123" {
			t.Errorf("Expected path 'attachments/ab/abc123', got %s", path)
		}
	})

	t.Run("GetPath with error", func(t *testing.T) {
		mock := NewMockAttachmentStorage()
		expectedErr := fmt.Errorf("path error")
		mock.SetPathError(expectedErr)

		_, err := mock.GetPath("abc123")
		if err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("GetPath for non-existent attachment", func(t *testing.T) {
		mock := NewMockAttachmentStorage()

		_, err := mock.GetPath("nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent attachment")
		}
	})

	t.Run("Exists", func(t *testing.T) {
		mock := NewMockAttachmentStorage()
		if err := mock.Store("abc123", []byte("data"), attachments.AttachmentInfo{}); err != nil {
			t.Fatal(err)
		}

		if !mock.Exists("abc123") {
			t.Error("Expected attachment to exist")
		}

		if mock.Exists("nonexistent") {
			t.Error("Expected attachment to not exist")
		}
	})

	t.Run("GetStoredData", func(t *testing.T) {
		mock := NewMockAttachmentStorage()
		data := []byte("test data")
		if err := mock.Store("abc123", data, attachments.AttachmentInfo{}); err != nil {
			t.Fatal(err)
		}

		stored, exists := mock.GetStoredData("abc123")
		if !exists {
			t.Error("Expected data to exist")
		}
		if string(stored) != string(data) {
			t.Errorf("Expected data %s, got %s", data, stored)
		}

		_, exists = mock.GetStoredData("nonexistent")
		if exists {
			t.Error("Expected data to not exist")
		}
	})
}

// Test generateRepositoryFiles error paths

func TestGenerateRepositoryFiles(t *testing.T) {
	t.Parallel()

	t.Run("handles summary generation error with logging", func(t *testing.T) {
		// Create a temp directory
		tempDir := t.TempDir()

		// Create an invalid repository (no calls/sms directories)
		// This will cause calculateRepositoryStats to fail

		imp := &Importer{
			options: &ImportOptions{
				RepoRoot: tempDir,
				Quiet:    false, // Enable warnings to test logging path
			},
		}

		// generateRepositoryFiles should not panic, just log warning
		imp.generateRepositoryFiles()
		// No assertion needed - we're testing it doesn't panic and logs warning
	})

	t.Run("handles manifest generation error with logging", func(t *testing.T) {
		// Create read-only temp directory to force write error
		tempDir := t.TempDir()

		// Create valid structure for summary but make directory read-only
		// to cause manifest write to fail
		imp := &Importer{
			options: &ImportOptions{
				RepoRoot: tempDir,
				Quiet:    false, // Enable warnings to test logging path
			},
		}

		// Make directory read-only after creating it
		if err := os.Chmod(tempDir, 0400); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := os.Chmod(tempDir, 0750); err != nil {
				t.Logf("Failed to restore permissions: %v", err)
			}
		}()

		// Should handle error gracefully and log warning
		imp.generateRepositoryFiles()
	})

	t.Run("handles both errors with quiet mode", func(t *testing.T) {
		tempDir := t.TempDir()

		imp := &Importer{
			options: &ImportOptions{
				RepoRoot: tempDir,
				Quiet:    true, // Suppress warnings
			},
		}

		// Should handle errors gracefully without logging
		imp.generateRepositoryFiles()
	})
}

// Test scanDirectory function

func TestScanDirectory(t *testing.T) {
	t.Parallel()

	t.Run("scans directory with XML files", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create test files
		files := []string{
			"calls-2023.xml",
			"calls-2024.xml",
			"sms-2023.xml",
			"other.txt",
		}

		for _, file := range files {
			path := filepath.Join(tempDir, file)
			if err := os.WriteFile(path, []byte("test"), 0600); err != nil {
				t.Fatal(err)
			}
		}

		imp := &Importer{
			options: &ImportOptions{
				Filter: "", // No filter
			},
		}

		result, err := imp.scanDirectory(tempDir)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Should find 3 XML files (calls and sms)
		if len(result) != 3 {
			t.Errorf("Expected 3 files, got %d", len(result))
		}
	})

	t.Run("skips hidden directories", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create hidden directory
		hiddenDir := filepath.Join(tempDir, ".hidden")
		if err := os.MkdirAll(hiddenDir, 0750); err != nil {
			t.Fatal(err)
		}

		// Create file in hidden directory
		hiddenFile := filepath.Join(hiddenDir, "calls-2023.xml")
		if err := os.WriteFile(hiddenFile, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}

		// Create visible file
		visibleFile := filepath.Join(tempDir, "calls-2023.xml")
		if err := os.WriteFile(visibleFile, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}

		imp := &Importer{
			options: &ImportOptions{
				Filter: "",
			},
		}

		result, err := imp.scanDirectory(tempDir)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Should only find visible file
		if len(result) != 1 {
			t.Errorf("Expected 1 file, got %d", len(result))
		}
	})

	t.Run("respects filter option", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create test files
		files := []string{
			"calls-2023.xml",
			"sms-2023.xml",
		}

		for _, file := range files {
			path := filepath.Join(tempDir, file)
			if err := os.WriteFile(path, []byte("test"), 0600); err != nil {
				t.Fatal(err)
			}
		}

		imp := &Importer{
			options: &ImportOptions{
				Filter: "calls", // Only calls
			},
		}

		result, err := imp.scanDirectory(tempDir)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Should only find calls file
		if len(result) != 1 {
			t.Errorf("Expected 1 file, got %d", len(result))
		}
		if len(result) > 0 && filepath.Base(result[0]) != "calls-2023.xml" {
			t.Errorf("Expected to find calls-2023.xml, got %s", filepath.Base(result[0]))
		}
	})

	t.Run("handles nested directories", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create nested structure
		nestedDir := filepath.Join(tempDir, "backups", "2023")
		if err := os.MkdirAll(nestedDir, 0750); err != nil {
			t.Fatal(err)
		}

		// Create file in nested directory
		nestedFile := filepath.Join(nestedDir, "calls-2023.xml")
		if err := os.WriteFile(nestedFile, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}

		imp := &Importer{
			options: &ImportOptions{
				Filter: "",
			},
		}

		result, err := imp.scanDirectory(tempDir)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Should find nested file
		if len(result) != 1 {
			t.Errorf("Expected 1 file, got %d", len(result))
		}
	})

	t.Run("skips files in repository structure", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create repository structure
		callsDir := filepath.Join(tempDir, "calls")
		if err := os.MkdirAll(callsDir, 0750); err != nil {
			t.Fatal(err)
		}

		// Create file in repository structure
		repoFile := filepath.Join(callsDir, "calls-2023.xml")
		if err := os.WriteFile(repoFile, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}

		// Create file outside repository structure
		importFile := filepath.Join(tempDir, "calls-import.xml")
		if err := os.WriteFile(importFile, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}

		imp := &Importer{
			options: &ImportOptions{
				Filter: "",
			},
		}

		result, err := imp.scanDirectory(tempDir)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Should only find import file, not repository file
		if len(result) != 1 {
			t.Errorf("Expected 1 file, got %d", len(result))
		}
	})
}

// Test findFiles function

func TestFindFiles(t *testing.T) {
	t.Parallel()

	t.Run("finds single file", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "calls-2023.xml")
		if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}

		imp := &Importer{
			options: &ImportOptions{
				Paths:  []string{testFile},
				Filter: "",
			},
		}

		files, err := imp.findFiles()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(files) != 1 {
			t.Errorf("Expected 1 file, got %d", len(files))
		}
	})

	t.Run("finds directory files", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create test files
		files := []string{
			"calls-2023.xml",
			"sms-2023.xml",
		}

		for _, file := range files {
			path := filepath.Join(tempDir, file)
			if err := os.WriteFile(path, []byte("test"), 0600); err != nil {
				t.Fatal(err)
			}
		}

		imp := &Importer{
			options: &ImportOptions{
				Paths:  []string{tempDir},
				Filter: "",
			},
		}

		result, err := imp.findFiles()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(result) != 2 {
			t.Errorf("Expected 2 files, got %d", len(result))
		}
	})

	t.Run("handles non-existent path", func(t *testing.T) {
		imp := &Importer{
			options: &ImportOptions{
				Paths:  []string{"/nonexistent/path"},
				Filter: "",
			},
		}

		_, err := imp.findFiles()
		if err == nil {
			t.Error("Expected error for non-existent path")
		}
	})

	t.Run("handles multiple paths", func(t *testing.T) {
		tempDir1 := t.TempDir()
		tempDir2 := t.TempDir()

		file1 := filepath.Join(tempDir1, "calls-2023.xml")
		file2 := filepath.Join(tempDir2, "sms-2023.xml")

		if err := os.WriteFile(file1, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file2, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}

		imp := &Importer{
			options: &ImportOptions{
				Paths:  []string{tempDir1, tempDir2},
				Filter: "",
			},
		}

		files, err := imp.findFiles()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(files) != 2 {
			t.Errorf("Expected 2 files, got %d", len(files))
		}
	})
}

// Test ensureYearStatExists function

func TestEnsureYearStatExists(t *testing.T) {
	t.Parallel()

	t.Run("creates YearStat when missing", func(t *testing.T) {
		imp := &Importer{}
		yearStats := make(map[int]*YearStat)
		tracker := NewYearTracker()

		// Add some tracking data
		tracker.TrackInitialEntry(2023)
		tracker.TrackImportEntry(2023, true)  // added
		tracker.TrackImportEntry(2023, false) // duplicate

		// Ensure stat exists
		imp.ensureYearStatExists(yearStats, 2023, tracker)

		// Verify it was created
		stat, exists := yearStats[2023]
		if !exists {
			t.Fatal("Expected YearStat to be created")
		}

		if stat.Initial != 1 {
			t.Errorf("Expected Initial=1, got %d", stat.Initial)
		}
		if stat.Added != 1 {
			t.Errorf("Expected Added=1, got %d", stat.Added)
		}
		if stat.Duplicates != 1 {
			t.Errorf("Expected Duplicates=1, got %d", stat.Duplicates)
		}
		if stat.Final != 0 {
			t.Errorf("Expected Final=0, got %d", stat.Final)
		}
	})

	t.Run("does not overwrite existing YearStat", func(t *testing.T) {
		imp := &Importer{}
		yearStats := make(map[int]*YearStat)
		tracker := NewYearTracker()

		// Pre-create a YearStat
		yearStats[2023] = &YearStat{
			Initial:    10,
			Added:      20,
			Duplicates: 5,
			Final:      30,
		}

		// Ensure stat exists
		imp.ensureYearStatExists(yearStats, 2023, tracker)

		// Verify it was NOT overwritten
		stat := yearStats[2023]
		if stat.Initial != 10 {
			t.Errorf("Expected Initial=10, got %d", stat.Initial)
		}
		if stat.Added != 20 {
			t.Errorf("Expected Added=20, got %d", stat.Added)
		}
		if stat.Final != 30 {
			t.Errorf("Expected Final=30, got %d", stat.Final)
		}
	})
}

// Test shouldProcessFile function

func TestShouldProcessFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		filter   string
		expected bool
	}{
		// No filter cases
		{"calls file, no filter", "/path/calls-2023.xml", "", true},
		{"sms file, no filter", "/path/sms-2023.xml", "", true},
		{"non-XML file, no filter", "/path/file.txt", "", false},
		{"calls file in repo structure", "/path/calls/calls-2023.xml", "", false},
		{"sms file in repo structure", "/path/sms/sms-2023.xml", "", false},

		// Filter = "calls"
		{"calls file with calls filter", "/path/calls-2023.xml", "calls", true},
		{"sms file with calls filter", "/path/sms-2023.xml", "calls", false},
		{"non-XML with calls filter", "/path/calls.txt", "calls", false},

		// Filter = "sms"
		{"calls file with sms filter", "/path/calls-2023.xml", "sms", false},
		{"sms file with sms filter", "/path/sms-2023.xml", "sms", true},
		{"non-XML with sms filter", "/path/sms.txt", "sms", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imp := &Importer{
				options: &ImportOptions{
					Filter: tt.filter,
				},
			}

			result := imp.shouldProcessFile(tt.path)
			if result != tt.expected {
				t.Errorf("shouldProcessFile(%q, filter=%q) = %v; want %v",
					tt.path, tt.filter, result, tt.expected)
			}
		})
	}
}

func TestConsoleProgressReporter(t *testing.T) {
	t.Run("creates quiet reporter", func(t *testing.T) {
		reporter := NewConsoleProgressReporter(true, false)
		if reporter == nil {
			t.Error("NewConsoleProgressReporter returned nil")
		}
	})

	t.Run("quiet mode suppresses output", func(t *testing.T) {
		reporter := NewConsoleProgressReporter(true, false) // quiet mode
		// Should not panic and not print
		reporter.StartFile("test.xml", 10, 1)
		reporter.UpdateProgress(50, 100)
		reporter.EndFile("test.xml", nil)
	})

	t.Run("normal mode shows progress", func(t *testing.T) {
		reporter := NewConsoleProgressReporter(false, false) // normal mode
		// Should not panic
		reporter.StartFile("test.xml", 10, 1)
		reporter.UpdateProgress(100, 200) // Will print at 100
		reporter.EndFile("test.xml", nil)
	})

	t.Run("verbose mode shows details", func(t *testing.T) {
		reporter := NewConsoleProgressReporter(false, true) // verbose mode
		summary := &YearStat{
			Added:      10,
			Duplicates: 5,
			Rejected:   2,
		}
		reporter.StartFile("test.xml", 10, 5)
		reporter.UpdateProgress(100, 200)
		reporter.EndFile("test.xml", summary)
		// Should not panic
	})
}
