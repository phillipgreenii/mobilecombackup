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

// Tests for removeOrphanedAttachment error paths

func TestOrphanRemover_RemoveOrphanedAttachment_ErrorPaths(t *testing.T) {
	t.Parallel()

	t.Run("absolute path outside repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		fs := afero.NewOsFs()
		mgr := NewAttachmentManager(tmpDir, fs)
		smsReader := &MockSMSReader{refs: map[string]bool{}}
		remover := NewOrphanRemover(smsReader, mgr, false)

		// Create attachment with absolute path outside repo
		attachment := &Attachment{
			Hash: "abc123def456abc123def456abc123def456abc123def456abc123def456abc1",
			Path: "/tmp/outside/attachment.bin",
			Size: 100,
		}

		result := &OrphanRemovalResult{
			FailedRemovals: []FailedRemoval{},
		}
		emptyDirs := make(map[string]bool)

		// This should fail with path outside repository error
		remover.removeOrphanedAttachment(attachment, tmpDir, result, emptyDirs)

		if result.RemovalFailures != 1 {
			t.Errorf("Expected 1 removal failure, got %d", result.RemovalFailures)
		}
		if len(result.FailedRemovals) != 1 {
			t.Errorf("Expected 1 failed removal record, got %d", len(result.FailedRemovals))
		}
	})

	t.Run("path traversal attempt", func(t *testing.T) {
		tmpDir := t.TempDir()
		fs := afero.NewOsFs()
		mgr := NewAttachmentManager(tmpDir, fs)
		smsReader := &MockSMSReader{refs: map[string]bool{}}
		remover := NewOrphanRemover(smsReader, mgr, false)

		// Attempt path traversal
		attachment := &Attachment{
			Hash: "abc123def456abc123def456abc123def456abc123def456abc123def456abc1",
			Path: "../../etc/passwd",
			Size: 100,
		}

		result := &OrphanRemovalResult{
			FailedRemovals: []FailedRemoval{},
		}
		emptyDirs := make(map[string]bool)

		// This should fail with path outside repository error
		remover.removeOrphanedAttachment(attachment, tmpDir, result, emptyDirs)

		if result.RemovalFailures != 1 {
			t.Errorf("Expected 1 removal failure, got %d", result.RemovalFailures)
		}
	})
}

// Tests for ValidateAttachmentStructure

func TestAttachmentManager_ValidateAttachmentStructure_ErrorCases(t *testing.T) {
	t.Parallel()

	t.Run("file in attachments root", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := NewAttachmentManager(tmpDir, afero.NewOsFs())

		// Create file directly in attachments directory (invalid)
		attachmentsDir := filepath.Join(tmpDir, "attachments")
		if err := os.MkdirAll(attachmentsDir, 0750); err != nil {
			t.Fatal(err)
		}

		badFile := filepath.Join(attachmentsDir, "badfile.txt")
		if err := os.WriteFile(badFile, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}

		// Validation should fail
		err := mgr.ValidateAttachmentStructure()
		if err == nil {
			t.Error("Expected validation error for file in attachments root")
		}
	})

	t.Run("invalid directory name length", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := NewAttachmentManager(tmpDir, afero.NewOsFs())

		// Create directory with invalid name length
		attachmentsDir := filepath.Join(tmpDir, "attachments")
		badDir := filepath.Join(attachmentsDir, "abc") // 3 chars instead of 2
		if err := os.MkdirAll(badDir, 0750); err != nil {
			t.Fatal(err)
		}

		// Validation should fail
		err := mgr.ValidateAttachmentStructure()
		if err == nil {
			t.Error("Expected validation error for invalid directory name length")
		}
	})

	t.Run("invalid directory name format", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := NewAttachmentManager(tmpDir, afero.NewOsFs())

		// Create directory with invalid hex name (contains 'z')
		attachmentsDir := filepath.Join(tmpDir, "attachments")
		badDir := filepath.Join(attachmentsDir, "zz") // 'z' is not a hex character
		if err := os.MkdirAll(badDir, 0750); err != nil {
			t.Fatal(err)
		}

		// Validation should fail
		err := mgr.ValidateAttachmentStructure()
		if err == nil {
			t.Error("Expected validation error for invalid directory name format")
		}
	})

	t.Run("hash with wrong prefix", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := NewAttachmentManager(tmpDir, afero.NewOsFs())

		// Create attachment in wrong prefix directory
		hash := "abc123def456abc123def456abc123def456abc123def456abc123def456abc1"
		wrongPrefixDir := filepath.Join(tmpDir, "attachments", "de", hash) // hash starts with 'ab' not 'de'
		if err := os.MkdirAll(wrongPrefixDir, 0750); err != nil {
			t.Fatal(err)
		}

		dataFile := filepath.Join(wrongPrefixDir, "attachment.bin")
		if err := os.WriteFile(dataFile, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}

		// Validation should fail
		err := mgr.ValidateAttachmentStructure()
		if err == nil {
			t.Error("Expected validation error for hash in wrong prefix directory")
		}
	})

	t.Run("invalid hash format in filename", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := NewAttachmentManager(tmpDir, afero.NewOsFs())

		// Create file with invalid hash name (contains 'z')
		attachmentsDir := filepath.Join(tmpDir, "attachments", "ab")
		if err := os.MkdirAll(attachmentsDir, 0750); err != nil {
			t.Fatal(err)
		}

		invalidHash := filepath.Join(attachmentsDir, "abzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz") // contains 'z'
		if err := os.WriteFile(invalidHash, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}

		// Validation should fail
		err := mgr.ValidateAttachmentStructure()
		if err == nil {
			t.Error("Expected validation error for invalid hash format")
		}
	})
}

// Tests for ReadAttachment error cases

func TestAttachmentManager_ReadAttachment_Errors(t *testing.T) {
	t.Parallel()

	t.Run("invalid hash format", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := NewAttachmentManager(tmpDir, afero.NewOsFs())

		// Try to read with invalid hash
		_, err := mgr.ReadAttachment("invalid")
		if err == nil {
			t.Error("Expected error for invalid hash format")
		}
	})

	t.Run("attachment not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := NewAttachmentManager(tmpDir, afero.NewOsFs())

		// Try to read non-existent attachment
		hash := "abc123def456abc123def456abc123def456abc123def456abc123def456abc1"
		_, err := mgr.ReadAttachment(hash)
		if err == nil {
			t.Error("Expected error for non-existent attachment")
		}
	})
}

// Tests for Store error paths

func TestDirectoryAttachmentStorage_Store_Errors(t *testing.T) {
	t.Parallel()

	t.Run("empty hash", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())

		// Try to store with empty hash
		err := storage.Store("", []byte("data"), AttachmentInfo{
			Hash:     "",
			MimeType: "text/plain",
		})
		if err == nil {
			t.Error("Expected error for empty hash")
		}
	})

	t.Run("short hash", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())

		// Try to store with short hash (less than 2 chars)
		err := storage.Store("a", []byte("data"), AttachmentInfo{
			Hash:     "a",
			MimeType: "text/plain",
		})
		if err == nil {
			t.Error("Expected error for hash too short")
		}
	})
}

// Tests for hash validation edge cases that ARE actually validated

func TestDirectoryAttachmentStorage_HashValidation_RealCases(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		hash      string
		shouldErr bool
	}{
		{"valid hash", "abc123def456abc123def456abc123def456abc123def456abc123def456abc1", false},
		{"empty hash", "", true},
		{"single char", "a", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())

			// Use Store to trigger hash validation
			err := storage.Store(tc.hash, []byte("data"), AttachmentInfo{
				Hash:     tc.hash,
				MimeType: "text/plain",
				Size:     4,
			})

			if tc.shouldErr && err == nil {
				t.Errorf("Expected error for hash %q", tc.hash)
			}
			if !tc.shouldErr && err != nil {
				t.Errorf("Unexpected error for hash %q: %v", tc.hash, err)
			}
		})
	}
}

// Tests for GetAttachmentDirPath edge cases

func TestAttachmentManager_GetAttachmentDirPath_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("empty hash", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := NewAttachmentManager(tmpDir, afero.NewOsFs())

		path := mgr.GetAttachmentDirPath("")
		if path != "" {
			t.Errorf("Expected empty path for empty hash, got %q", path)
		}
	})

	t.Run("single char hash", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := NewAttachmentManager(tmpDir, afero.NewOsFs())

		path := mgr.GetAttachmentDirPath("a")
		if path != "" {
			t.Errorf("Expected empty path for single char hash, got %q", path)
		}
	})

	t.Run("uppercase hash normalized", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := NewAttachmentManager(tmpDir, afero.NewOsFs())

		path := mgr.GetAttachmentDirPath("ABCDEF")
		expected := filepath.Join("attachments", "ab", "abcdef")
		if path != expected {
			t.Errorf("Expected %q, got %q", expected, path)
		}
	})
}
