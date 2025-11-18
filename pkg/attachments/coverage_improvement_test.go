package attachments

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

// Tests for validateNewFormatAttachment function

func TestMigrationManager_ValidateNewFormatAttachment_Errors(t *testing.T) {
	t.Parallel()

	t.Run("missing metadata file", func(t *testing.T) {
		tmpDir := t.TempDir()
		mm := NewMigrationManager(tmpDir, afero.NewOsFs())
		mm.SetLogOutput(false)
		storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())

		// Create attachment directory without metadata
		hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		dirPath := filepath.Join(tmpDir, "attachments", hash[:2], hash)
		if err := os.MkdirAll(dirPath, 0750); err != nil {
			t.Fatal(err)
		}

		// Create attachment file but not metadata
		attachmentPath := filepath.Join(dirPath, "data")
		if err := os.WriteFile(attachmentPath, []byte(""), 0600); err != nil {
			t.Fatal(err)
		}

		// Should return true (has errors)
		hasErrors := mm.validateNewFormatAttachment(storage, hash)
		if !hasErrors {
			t.Error("Expected validation to fail when metadata is missing")
		}
	})

	t.Run("missing attachment file", func(t *testing.T) {
		tmpDir := t.TempDir()
		mm := NewMigrationManager(tmpDir, afero.NewOsFs())
		mm.SetLogOutput(false)
		storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())

		// Store attachment first
		hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		data := []byte("")
		metadata := AttachmentInfo{
			Hash:     hash,
			MimeType: "application/octet-stream",
			Size:     int64(len(data)),
		}

		if err := storage.Store(hash, data, metadata); err != nil {
			t.Fatal(err)
		}

		// Remove the attachment file but keep metadata
		attachmentPath, _ := storage.GetAttachmentFilePath(hash)
		if err := os.Remove(attachmentPath); err != nil {
			t.Fatal(err)
		}

		// Should return true (has errors)
		hasErrors := mm.validateNewFormatAttachment(storage, hash)
		if !hasErrors {
			t.Error("Expected validation to fail when attachment file is missing")
		}
	})

	t.Run("hash mismatch in metadata", func(t *testing.T) {
		tmpDir := t.TempDir()
		mm := NewMigrationManager(tmpDir, afero.NewOsFs())
		mm.SetLogOutput(false)
		storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())

		// Store attachment with correct hash
		hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		data := []byte("")
		metadata := AttachmentInfo{
			Hash:     hash,
			MimeType: "application/octet-stream",
			Size:     int64(len(data)),
		}

		if err := storage.Store(hash, data, metadata); err != nil {
			t.Fatal(err)
		}

		// Corrupt the metadata by changing the hash
		dirPath := filepath.Join(tmpDir, "attachments", hash[:2], hash)
		metadataPath := filepath.Join(dirPath, "metadata.yaml")

		corruptedMetadata := AttachmentInfo{
			Hash:     "wronghash123",
			MimeType: "application/octet-stream",
			Size:     0,
		}

		if data, err := yaml.Marshal(corruptedMetadata); err == nil {
			if err := os.WriteFile(metadataPath, data, 0600); err != nil {
				t.Fatal(err)
			}
		}

		// Should return true (has errors)
		hasErrors := mm.validateNewFormatAttachment(storage, hash)
		if !hasErrors {
			t.Error("Expected validation to fail when metadata hash doesn't match")
		}
	})

	t.Run("validation with logging enabled", func(t *testing.T) {
		tmpDir := t.TempDir()
		mm := NewMigrationManager(tmpDir, afero.NewOsFs())
		mm.SetLogOutput(true) // Enable logging to test logging paths
		storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())

		// Create attachment directory without metadata
		hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		dirPath := filepath.Join(tmpDir, "attachments", hash[:2], hash)
		if err := os.MkdirAll(dirPath, 0750); err != nil {
			t.Fatal(err)
		}

		// Should return true (has errors) and log the error
		hasErrors := mm.validateNewFormatAttachment(storage, hash)
		if !hasErrors {
			t.Error("Expected validation to fail when metadata is missing")
		}
	})
}

// Tests for GetMigrationStatus with mixed formats

func TestMigrationManager_GetMigrationStatus_MixedFormats(t *testing.T) {
	t.Parallel()

	t.Run("status with both legacy and new format", func(t *testing.T) {
		tmpDir := t.TempDir()
		mm := NewMigrationManager(tmpDir, afero.NewOsFs())
		mm.SetLogOutput(false)

		// Create a legacy attachment
		legacyHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		legacyPath := filepath.Join(tmpDir, "attachments", "e3", legacyHash)
		if err := os.MkdirAll(filepath.Dir(legacyPath), 0750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(legacyPath, []byte(""), 0600); err != nil {
			t.Fatal(err)
		}

		// Create a new format attachment
		storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())
		newHash := "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3"
		newData := []byte("hello")
		newMetadata := AttachmentInfo{
			Hash:     newHash,
			MimeType: "text/plain",
			Size:     int64(len(newData)),
		}
		if err := storage.Store(newHash, newData, newMetadata); err != nil {
			t.Fatal(err)
		}

		// Get migration status
		status, err := mm.GetMigrationStatus()
		if err != nil {
			t.Fatalf("Failed to get migration status: %v", err)
		}

		// Verify counts
		if status["total_count"].(int) != 2 {
			t.Errorf("Expected total_count 2, got %d", status["total_count"])
		}
		if status["legacy_count"].(int) != 1 {
			t.Errorf("Expected legacy_count 1, got %d", status["legacy_count"])
		}
		if status["new_count"].(int) != 1 {
			t.Errorf("Expected new_count 1, got %d", status["new_count"])
		}
		if status["migrated"].(bool) != false {
			t.Error("Expected migrated false when legacy attachments exist")
		}
	})

	t.Run("status with only new format indicates fully migrated", func(t *testing.T) {
		tmpDir := t.TempDir()
		mm := NewMigrationManager(tmpDir, afero.NewOsFs())
		mm.SetLogOutput(false)

		// Create only new format attachments
		storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())
		hash1 := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		hash2 := "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3"

		for _, hash := range []string{hash1, hash2} {
			data := []byte("test")
			metadata := AttachmentInfo{
				Hash:     hash,
				MimeType: "text/plain",
				Size:     int64(len(data)),
			}
			if err := storage.Store(hash, data, metadata); err != nil {
				t.Fatal(err)
			}
		}

		// Get migration status
		status, err := mm.GetMigrationStatus()
		if err != nil {
			t.Fatalf("Failed to get migration status: %v", err)
		}

		// Verify fully migrated
		if status["total_count"].(int) != 2 {
			t.Errorf("Expected total_count 2, got %d", status["total_count"])
		}
		if status["legacy_count"].(int) != 0 {
			t.Errorf("Expected legacy_count 0, got %d", status["legacy_count"])
		}
		if status["new_count"].(int) != 2 {
			t.Errorf("Expected new_count 2, got %d", status["new_count"])
		}
		if status["migrated"].(bool) != true {
			t.Error("Expected migrated true when only new format attachments exist")
		}
	})
}

// Tests for executeMigration error paths

func TestMigrationManager_ExecuteMigration_ErrorPaths(t *testing.T) {
	t.Parallel()

	t.Run("failed to remove legacy file", func(t *testing.T) {
		tmpDir := t.TempDir()
		mm := NewMigrationManager(tmpDir, afero.NewOsFs())
		mm.SetLogOutput(false)
		storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())

		// Create a fake attachment that doesn't exist
		hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		attachment := &Attachment{
			Hash: hash,
			Path: "attachments/e3/nonexistent",
		}

		data := []byte("")
		metadata := AttachmentInfo{
			Hash:     hash,
			MimeType: "application/octet-stream",
			Size:     int64(len(data)),
		}

		result := MigrationResult{
			Hash:       hash,
			LegacyPath: attachment.Path,
		}

		// Execute migration - should fail to remove non-existent file
		result = mm.executeMigration(attachment, storage, data, metadata, result)

		if result.Success {
			t.Error("Expected migration to fail when legacy file doesn't exist")
		}
		if result.Error == "" {
			t.Error("Expected error message")
		}
	})

	t.Run("successful migration with logging", func(t *testing.T) {
		tmpDir := t.TempDir()
		mm := NewMigrationManager(tmpDir, afero.NewOsFs())
		mm.SetLogOutput(true) // Enable logging to test logging path
		storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())

		// Create a real legacy attachment
		hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		legacyPath := filepath.Join(tmpDir, "attachments", "e3", hash)
		if err := os.MkdirAll(filepath.Dir(legacyPath), 0750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(legacyPath, []byte(""), 0600); err != nil {
			t.Fatal(err)
		}

		attachment := &Attachment{
			Hash: hash,
			Path: filepath.Join("attachments", "e3", hash),
		}

		data := []byte("")
		metadata := AttachmentInfo{
			Hash:     hash,
			MimeType: "application/octet-stream",
			Size:     int64(len(data)),
		}

		result := MigrationResult{
			Hash:       hash,
			LegacyPath: attachment.Path,
		}

		// Execute migration - should succeed and log
		result = mm.executeMigration(attachment, storage, data, metadata, result)

		if !result.Success {
			t.Errorf("Expected successful migration, got error: %s", result.Error)
		}
		if result.NewPath == "" {
			t.Error("Expected new path to be set")
		}
	})
}

// Tests for ValidateMigration error scenarios

func TestMigrationManager_ValidateMigration_WithErrors(t *testing.T) {
	t.Parallel()

	t.Run("validation detects corrupted attachments", func(t *testing.T) {
		tmpDir := t.TempDir()
		mm := NewMigrationManager(tmpDir, afero.NewOsFs())
		mm.SetLogOutput(false)
		storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())

		// Store valid attachment
		hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		data := []byte("")
		metadata := AttachmentInfo{
			Hash:     hash,
			MimeType: "application/octet-stream",
			Size:     int64(len(data)),
		}

		if err := storage.Store(hash, data, metadata); err != nil {
			t.Fatal(err)
		}

		// Corrupt the attachment by removing the file
		attachmentPath, _ := storage.GetAttachmentFilePath(hash)
		if err := os.Remove(attachmentPath); err != nil {
			t.Fatal(err)
		}

		// Validation should fail
		err := mm.ValidateMigration()
		if err == nil {
			t.Error("Expected validation to fail with corrupted attachment")
		}
	})

	t.Run("validation with logging enabled", func(t *testing.T) {
		tmpDir := t.TempDir()
		mm := NewMigrationManager(tmpDir, afero.NewOsFs())
		mm.SetLogOutput(true) // Enable logging to test logging paths
		storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())

		// Store attachment with corrupted metadata
		hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		data := []byte("")
		metadata := AttachmentInfo{
			Hash:     hash,
			MimeType: "application/octet-stream",
			Size:     int64(len(data)),
		}

		if err := storage.Store(hash, data, metadata); err != nil {
			t.Fatal(err)
		}

		// Corrupt metadata
		dirPath := filepath.Join(tmpDir, "attachments", hash[:2], hash)
		metadataPath := filepath.Join(dirPath, "metadata.yaml")
		corruptedMetadata := AttachmentInfo{
			Hash:     "wronghash",
			MimeType: "application/octet-stream",
			Size:     0,
		}

		if data, err := yaml.Marshal(corruptedMetadata); err == nil {
			if err := os.WriteFile(metadataPath, data, 0600); err != nil {
				t.Fatal(err)
			}
		}

		// Validation should fail and log errors
		err := mm.ValidateMigration()
		if err == nil {
			t.Error("Expected validation to fail with corrupted metadata")
		}
	})
}

// Tests for GetPath error paths

func TestDirectoryAttachmentStorage_GetPath_Errors(t *testing.T) {
	t.Parallel()

	t.Run("invalid hash format", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())

		// Try with invalid hash
		_, err := storage.GetPath("invalid")
		if err == nil {
			t.Error("Expected error for invalid hash")
		}
	})

	t.Run("non-existent attachment", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())

		// Try with valid hash but non-existent attachment
		hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		_, err := storage.GetPath(hash)
		if err == nil {
			t.Error("Expected error for non-existent attachment")
		}
	})
}

// Tests for Exists edge cases

func TestDirectoryAttachmentStorage_Exists_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("invalid hash returns false", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())

		// Invalid hash should return false
		exists := storage.Exists("invalid")
		if exists {
			t.Error("Expected false for invalid hash")
		}
	})

	t.Run("directory exists but no data file", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())

		// Create directory without data file
		hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		dirPath := filepath.Join(tmpDir, "attachments", hash[:2], hash)
		if err := os.MkdirAll(dirPath, 0750); err != nil {
			t.Fatal(err)
		}

		// Should return false
		exists := storage.Exists(hash)
		if exists {
			t.Error("Expected false when directory exists but data file is missing")
		}
	})
}

// Tests for GetMetadata error paths

func TestDirectoryAttachmentStorage_GetMetadata_Errors(t *testing.T) {
	t.Parallel()

	t.Run("corrupted metadata yaml", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())

		// Store valid attachment first
		hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		data := []byte("")
		metadata := AttachmentInfo{
			Hash:     hash,
			MimeType: "application/octet-stream",
			Size:     int64(len(data)),
		}

		if err := storage.Store(hash, data, metadata); err != nil {
			t.Fatal(err)
		}

		// Corrupt the metadata file
		dirPath := filepath.Join(tmpDir, "attachments", hash[:2], hash)
		metadataPath := filepath.Join(dirPath, "metadata.yaml")
		if err := os.WriteFile(metadataPath, []byte("invalid: yaml: content: [[["), 0600); err != nil {
			t.Fatal(err)
		}

		// Try to read metadata
		_, err := storage.GetMetadata(hash)
		if err == nil {
			t.Error("Expected error for corrupted metadata")
		}
	})

	t.Run("metadata file missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())

		// Create directory with data file but no metadata
		hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		dirPath := filepath.Join(tmpDir, "attachments", hash[:2], hash)
		if err := os.MkdirAll(dirPath, 0750); err != nil {
			t.Fatal(err)
		}

		dataPath := filepath.Join(dirPath, "data")
		if err := os.WriteFile(dataPath, []byte(""), 0600); err != nil {
			t.Fatal(err)
		}

		// Try to read metadata
		_, err := storage.GetMetadata(hash)
		if err == nil {
			t.Error("Expected error when metadata file is missing")
		}
	})
}
