package attachments

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
)

func TestMigrationManager_GetMigrationStatus(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "migration_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	migrationManager := NewMigrationManager(tmpDir, afero.NewOsFs())

	// Test empty repository
	status, err := migrationManager.GetMigrationStatus()
	if err != nil {
		t.Fatalf("Failed to get migration status: %v", err)
	}

	if status["total_count"].(int) != 0 {
		t.Errorf("Expected total_count 0, got %d", status["total_count"])
	}
	if status["legacy_count"].(int) != 0 {
		t.Errorf("Expected legacy_count 0, got %d", status["legacy_count"])
	}
	if status["new_count"].(int) != 0 {
		t.Errorf("Expected new_count 0, got %d", status["new_count"])
	}
	if status["migrated"].(bool) != false {
		t.Errorf("Expected migrated false, got %t", status["migrated"])
	}
}

func TestMigrationManager_MigrateAllAttachments_EmptyRepository(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "migration_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	migrationManager := NewMigrationManager(tmpDir, afero.NewOsFs())
	migrationManager.SetLogOutput(false) // Disable logging for tests

	// Test migration of empty repository
	summary, err := migrationManager.MigrateAllAttachments()
	if err != nil {
		t.Fatalf("Failed to migrate attachments: %v", err)
	}

	if summary.TotalFound != 0 {
		t.Errorf("Expected TotalFound 0, got %d", summary.TotalFound)
	}
	if summary.Migrated != 0 {
		t.Errorf("Expected Migrated 0, got %d", summary.Migrated)
	}
	if summary.Failed != 0 {
		t.Errorf("Expected Failed 0, got %d", summary.Failed)
	}
}

func TestMigrationManager_MigrateAllAttachments_WithLegacyAttachment(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "migration_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a legacy attachment
	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	legacyPath := filepath.Join(tmpDir, "attachments", "e3", hash)
	err = os.MkdirAll(filepath.Dir(legacyPath), 0750)
	if err != nil {
		t.Fatalf("Failed to create legacy directory: %v", err)
	}

	// Write test content (empty string has the hash above)
	err = os.WriteFile(legacyPath, []byte(""), 0600)
	if err != nil {
		t.Fatalf("Failed to create legacy file: %v", err)
	}

	migrationManager := NewMigrationManager(tmpDir, afero.NewOsFs())
	migrationManager.SetLogOutput(false) // Disable logging for tests

	// Test migration
	summary, err := migrationManager.MigrateAllAttachments()
	if err != nil {
		t.Fatalf("Failed to migrate attachments: %v", err)
	}

	if summary.TotalFound != 1 {
		t.Errorf("Expected TotalFound 1, got %d", summary.TotalFound)
	}
	if summary.Migrated != 1 {
		t.Errorf("Expected Migrated 1, got %d", summary.Migrated)
	}
	if summary.Failed != 0 {
		t.Errorf("Expected Failed 0, got %d", summary.Failed)
		if len(summary.Results) > 0 && summary.Results[0].Error != "" {
			t.Errorf("Migration error: %s", summary.Results[0].Error)
		}
	}

	// Verify the legacy file is replaced by a directory (new format)
	if stat, err := os.Stat(legacyPath); err != nil {
		t.Errorf("Expected path to exist as directory for new format: %v", err)
	} else if !stat.IsDir() {
		t.Errorf("Expected legacy file path to be converted to directory, but it's still a file. Mode: %v", stat.Mode())
	}

	// Verify the new format exists
	storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())
	if !storage.Exists(hash) {
		t.Error("Expected new format attachment to exist")
	}

	// Verify metadata
	metadata, err := storage.GetMetadata(hash)
	if err != nil {
		t.Fatalf("Failed to read migrated metadata: %v", err)
	}

	if metadata.Hash != hash {
		t.Errorf("Expected hash %s, got %s", hash, metadata.Hash)
	}
	if metadata.MimeType == "" {
		t.Error("Expected MIME type to be set")
	}
}

func TestMigrationManager_DryRun(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "migration_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a legacy attachment
	hash := "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3"
	legacyPath := filepath.Join(tmpDir, "attachments", "a6", hash)
	err = os.MkdirAll(filepath.Dir(legacyPath), 0750)
	if err != nil {
		t.Fatalf("Failed to create legacy directory: %v", err)
	}

	// Write test content ("hello" has the hash above)
	err = os.WriteFile(legacyPath, []byte("hello"), 0600)
	if err != nil {
		t.Fatalf("Failed to create legacy file: %v", err)
	}

	migrationManager := NewMigrationManager(tmpDir, afero.NewOsFs())
	migrationManager.SetDryRun(true)
	migrationManager.SetLogOutput(false) // Disable logging for tests

	// Test dry run migration
	summary, err := migrationManager.MigrateAllAttachments()
	if err != nil {
		t.Fatalf("Failed to migrate attachments in dry run: %v", err)
	}

	if summary.TotalFound != 1 {
		t.Errorf("Expected TotalFound 1, got %d", summary.TotalFound)
	}
	if summary.Migrated != 1 {
		t.Errorf("Expected Migrated 1, got %d", summary.Migrated)
	}

	// Verify the legacy file still exists (dry run shouldn't modify anything)
	if _, err := os.Stat(legacyPath); os.IsNotExist(err) {
		t.Error("Expected legacy file to still exist in dry run")
	}

	// Verify the new format doesn't exist (dry run shouldn't create actual new format)
	storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())
	if storage.Exists(hash) {
		t.Error("Expected new format attachment to not exist in dry run")
	}
}

func TestMigrationManager_ValidateMigration(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "migration_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	migrationManager := NewMigrationManager(tmpDir, afero.NewOsFs())
	migrationManager.SetLogOutput(false) // Disable logging for tests

	// Test validation of empty repository
	err = migrationManager.ValidateMigration()
	if err != nil {
		t.Fatalf("Validation failed for empty repository: %v", err)
	}

	// Create a new format attachment
	storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())

	// Use a proper hash for testing
	properHash := "2cf24dba4f21d4288674d04eb799d064e77f5d841b3d9e1e3c5d4c72a0d72c6e"
	data := []byte("hello")
	metadata := AttachmentInfo{
		Hash:      properHash,
		MimeType:  "text/plain",
		Size:      int64(len(data)),
		CreatedAt: migrationManager.GetCurrentTime(),
	}

	err = storage.Store(properHash, data, metadata)
	if err != nil {
		t.Fatalf("Failed to store test attachment: %v", err)
	}

	// Test validation with new format attachment
	err = migrationManager.ValidateMigration()
	if err != nil {
		t.Fatalf("Validation failed with new format attachment: %v", err)
	}
}

func TestMigrationManager_DefaultLogOutputDisabled(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "migration_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Test that new migration manager has logging disabled by default
	migrationManager := NewMigrationManager(tmpDir, afero.NewOsFs())

	// The logOutput field is not exported, but we can test the behavior
	// by running a migration and checking that no panics occur
	// (indirect verification that logOutput defaults to false)
	summary, err := migrationManager.MigrateAllAttachments()
	if err != nil {
		t.Fatalf("Failed to migrate attachments with default settings: %v", err)
	}

	// Should have no issues with empty repository
	if summary.TotalFound != 0 {
		t.Errorf("Expected TotalFound 0 in empty repo, got %d", summary.TotalFound)
	}

	// Test that logging can be explicitly enabled
	migrationManager.SetLogOutput(true)
	summary2, err := migrationManager.MigrateAllAttachments()
	if err != nil {
		t.Fatalf("Failed to migrate attachments with logging enabled: %v", err)
	}

	if summary2.TotalFound != 0 {
		t.Errorf("Expected TotalFound 0 in empty repo with logging, got %d", summary2.TotalFound)
	}
}

func TestDetectMimeTypeFromContent(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{"Empty data", []byte{}, "application/octet-stream"},
		{"PNG signature", []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, "image/png"},
		{"JPEG signature", []byte{0xFF, 0xD8, 0xFF, 0xE0}, "image/jpeg"},
		{"GIF87a signature", []byte("GIF87a123456"), "image/gif"},
		{"GIF89a signature", []byte("GIF89a123456"), "image/gif"},
		{"PDF signature", []byte("%PDF-1.4"), "application/pdf"},
		{"ZIP signature", []byte{0x50, 0x4B, 0x03, 0x04, 0x14, 0x00}, "application/zip"},
		{"Text content", []byte("hello world this is text"), "text/plain"},
		{"Binary content", []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0xFF}, "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectMimeTypeFromContent(tt.data, "test-hash")
			if result != tt.expected {
				t.Errorf("detectMimeTypeFromContent() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestIsTextContent(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{"Empty data", []byte{}, true},
		{"Plain text", []byte("hello world"), true},
		{"Text with newlines", []byte("hello\nworld\n"), true},
		{"Text with tabs", []byte("hello\tworld"), true},
		{"Text with special chars", []byte("hello@world.com"), true},
		{"Mostly binary", []byte{0x00, 0x01, 0x02, 0x03}, false},
		{"Mixed content - mostly text", []byte("hello\x00world"), false}, // One null byte makes it binary
		{"All printable ASCII", []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTextContent(tt.data)
			if result != tt.expected {
				t.Errorf("isTextContent() = %t, want %t", result, tt.expected)
			}
		})
	}
}

// GetCurrentTime is a helper method to get current time (for testing)
func (mm *MigrationManager) GetCurrentTime() time.Time {
	return time.Now().UTC()
}
