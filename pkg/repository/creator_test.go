package repository

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

func TestCreator_Initialize(t *testing.T) {
	t.Run("creates repository structure successfully", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		creator := NewCreator(fs, "test-1.0.0")

		result, err := creator.Initialize("/test/repo", false)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if result.RepoRoot != "/test/repo" {
			t.Errorf("Expected repo root /test/repo, got: %s", result.RepoRoot)
		}

		if result.DryRun {
			t.Error("Expected DryRun to be false")
		}

		// Verify directories were created
		expectedDirs := []string{
			"/test/repo",
			"/test/repo/calls",
			"/test/repo/sms",
			"/test/repo/attachments",
		}

		for _, dir := range expectedDirs {
			exists, err := afero.DirExists(fs, dir)
			if err != nil {
				t.Errorf("Error checking directory %s: %v", dir, err)
			}
			if !exists {
				t.Errorf("Expected directory to exist: %s", dir)
			}
		}

		// Verify files were created
		expectedFiles := []string{
			"/test/repo/.mobilecombackup.yaml",
			"/test/repo/contacts.yaml",
			"/test/repo/summary.yaml",
			"/test/repo/files.yaml",
			"/test/repo/files.yaml.sha256",
		}

		for _, file := range expectedFiles {
			exists, err := afero.Exists(fs, file)
			if err != nil {
				t.Errorf("Error checking file %s: %v", file, err)
			}
			if !exists {
				t.Errorf("Expected file to exist: %s", file)
			}
		}
	})

	t.Run("dry run mode", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		creator := NewCreator(fs, "test-1.0.0")

		result, err := creator.Initialize("/test/repo", true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if !result.DryRun {
			t.Error("Expected DryRun to be true")
		}

		// Verify nothing was actually created
		exists, _ := afero.DirExists(fs, "/test/repo")
		if exists {
			t.Error("In dry run mode, no directories should be created")
		}
	})

	t.Run("creates all expected items in result", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		creator := NewCreator(fs, "test-1.0.0")

		result, err := creator.Initialize("/test/repo", false)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Check that all expected paths are in Created list
		expectedCount := 9 // repo + 3 dirs + 5 files
		if len(result.Created) != expectedCount {
			t.Errorf("Expected %d items created, got: %d", expectedCount, len(result.Created))
		}
	})
}

func TestCreator_CreateMarkerFile(t *testing.T) {
	t.Run("creates marker file with correct content", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		creator := NewCreator(fs, "test-1.0.0")

		// Create repo directory first
		_ = fs.MkdirAll("/test/repo", 0750)

		result := &InitResult{Created: []string{}}
		var createdPaths []string
		rollback := creator.createRollbackFunction(false, &createdPaths)

		err := creator.createMarkerFile("/test/repo", false, result, &createdPaths, rollback)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Verify file exists
		markerPath := filepath.Join("/test/repo", ".mobilecombackup.yaml")
		exists, err := afero.Exists(fs, markerPath)
		if err != nil || !exists {
			t.Error("Expected marker file to exist")
		}

		// Read and verify content
		content, err := afero.ReadFile(fs, markerPath)
		if err != nil {
			t.Fatalf("Failed to read marker file: %v", err)
		}

		contentStr := string(content)
		if !contains(contentStr, "repository_structure_version: \"1\"") {
			t.Error("Marker file should contain repository_structure_version")
		}
		if !contains(contentStr, "created_by: mobilecombackup v") {
			t.Error("Marker file should contain created_by field")
		}
	})
}

func TestCreator_CreateContactsFile(t *testing.T) {
	t.Run("creates empty contacts file", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		creator := NewCreator(fs, "test-1.0.0")

		// Create repo directory first
		_ = fs.MkdirAll("/test/repo", 0750)

		result := &InitResult{Created: []string{}}
		var createdPaths []string
		rollback := creator.createRollbackFunction(false, &createdPaths)

		err := creator.createContactsFile("/test/repo", false, result, &createdPaths, rollback)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Verify file exists and has correct content
		contactsPath := filepath.Join("/test/repo", "contacts.yaml")
		content, err := afero.ReadFile(fs, contactsPath)
		if err != nil {
			t.Fatalf("Failed to read contacts file: %v", err)
		}

		expected := "contacts: []\n"
		if string(content) != expected {
			t.Errorf("Expected contacts file content %q, got: %q", expected, string(content))
		}
	})
}

func TestCreator_CreateSummaryFile(t *testing.T) {
	t.Run("creates summary file with zero counts", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		creator := NewCreator(fs, "test-1.0.0")

		// Create repo directory first
		_ = fs.MkdirAll("/test/repo", 0750)

		result := &InitResult{Created: []string{}}
		var createdPaths []string
		rollback := creator.createRollbackFunction(false, &createdPaths)

		err := creator.createSummaryFile("/test/repo", false, result, &createdPaths, rollback)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Verify file exists
		summaryPath := filepath.Join("/test/repo", "summary.yaml")
		content, err := afero.ReadFile(fs, summaryPath)
		if err != nil {
			t.Fatalf("Failed to read summary file: %v", err)
		}

		contentStr := string(content)
		if !contains(contentStr, "calls: 0") {
			t.Error("Summary file should contain calls: 0")
		}
		if !contains(contentStr, "sms: 0") {
			t.Error("Summary file should contain sms: 0")
		}
	})
}

func TestCreator_CreateRepositoryDirectories(t *testing.T) {
	t.Run("creates all required directories", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		creator := NewCreator(fs, "test-1.0.0")

		result := &InitResult{Created: []string{}}
		var createdPaths []string
		rollback := creator.createRollbackFunction(false, &createdPaths)

		err := creator.createRepositoryDirectories("/test/repo", false, result, &createdPaths, rollback)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Verify all directories exist
		expectedDirs := []string{
			"/test/repo",
			"/test/repo/calls",
			"/test/repo/sms",
			"/test/repo/attachments",
		}

		for _, dir := range expectedDirs {
			exists, err := afero.DirExists(fs, dir)
			if err != nil || !exists {
				t.Errorf("Expected directory to exist: %s", dir)
			}
		}
	})

	t.Run("dry run creates no directories", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		creator := NewCreator(fs, "test-1.0.0")

		result := &InitResult{Created: []string{}}
		var createdPaths []string
		rollback := creator.createRollbackFunction(true, &createdPaths)

		err := creator.createRepositoryDirectories("/test/repo", true, result, &createdPaths, rollback)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Verify directory was NOT created
		exists, _ := afero.DirExists(fs, "/test/repo")
		if exists {
			t.Error("In dry run mode, directory should not be created")
		}

		// But result should still list what would be created
		if len(result.Created) == 0 {
			t.Error("Result should list directories even in dry run")
		}
	})
}

func TestCreator_Rollback(t *testing.T) {
	t.Run("rollback removes created files", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		creator := NewCreator(fs, "test-1.0.0")

		// Create some files
		_ = fs.MkdirAll("/test/repo", 0750)
		_ = afero.WriteFile(fs, "/test/repo/file1.txt", []byte("test"), 0600)
		_ = afero.WriteFile(fs, "/test/repo/file2.txt", []byte("test"), 0600)

		createdPaths := []string{
			"/test/repo/file1.txt",
			"/test/repo/file2.txt",
			"/test/repo",
		}

		rollback := creator.createRollbackFunction(false, &createdPaths)
		rollback()

		// Verify files were removed
		for _, path := range createdPaths {
			exists, _ := afero.Exists(fs, path)
			if exists {
				t.Errorf("Expected path to be removed: %s", path)
			}
		}
	})

	t.Run("rollback in dry run mode does nothing", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		creator := NewCreator(fs, "test-1.0.0")

		// Create a file
		_ = fs.MkdirAll("/test/repo", 0750)
		_ = afero.WriteFile(fs, "/test/repo/file.txt", []byte("test"), 0600)

		createdPaths := []string{"/test/repo/file.txt"}

		rollback := creator.createRollbackFunction(true, &createdPaths)
		rollback()

		// Verify file still exists (rollback shouldn't remove in dry run)
		exists, _ := afero.Exists(fs, "/test/repo/file.txt")
		if !exists {
			t.Error("In dry run mode, rollback should not remove files")
		}
	})
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
