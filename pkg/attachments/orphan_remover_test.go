package attachments

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

// MockSMSReader for testing
type MockSMSReader struct {
	refs map[string]bool
	err  error
}

func (m *MockSMSReader) GetAllAttachmentRefs(ctx context.Context) (map[string]bool, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.refs, nil
}

func TestOrphanRemover_RemoveOrphans(t *testing.T) {
	t.Run("no orphans found", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create attachment manager
		fs := afero.NewOsFs()
		mgr := NewAttachmentManager(tmpDir, fs)

		// Create an attachment and reference it (must be 64-char SHA-256 hash)
		hash := "abc123def456abc123def456abc123def456abc123def456abc123def456abc1"
		createTestAttachment(t, tmpDir, hash, []byte("test data"))

		// Mock SMS reader that references the attachment
		smsReader := &MockSMSReader{
			refs: map[string]bool{hash: true},
		}

		remover := NewOrphanRemover(smsReader, mgr, false)
		result, err := remover.RemoveOrphans(context.Background())

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if result.OrphansFound != 0 {
			t.Errorf("Expected 0 orphans, got: %d", result.OrphansFound)
		}

		if result.AttachmentsScanned != 1 {
			t.Errorf("Expected 1 attachment scanned, got: %d", result.AttachmentsScanned)
		}
	})

	t.Run("orphans found and removed", func(t *testing.T) {
		tmpDir := t.TempDir()

		fs := afero.NewOsFs()
		mgr := NewAttachmentManager(tmpDir, fs)

		// Create two attachments, only reference one (must be 64-char SHA-256 hashes)
		hash1 := "abc123def456abc123def456abc123def456abc123def456abc123def456abc1"
		hash2 := "def456abc123def456abc123def456abc123def456abc123def456abc123def4"
		createTestAttachment(t, tmpDir, hash1, []byte("test data 1"))
		createTestAttachment(t, tmpDir, hash2, []byte("test data 2 longer"))

		// Mock SMS reader that only references hash1
		smsReader := &MockSMSReader{
			refs: map[string]bool{hash1: true},
		}

		remover := NewOrphanRemover(smsReader, mgr, false)
		result, err := remover.RemoveOrphans(context.Background())

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if result.AttachmentsScanned != 2 {
			t.Errorf("Expected 2 attachments scanned, got: %d", result.AttachmentsScanned)
		}

		if result.OrphansFound != 1 {
			t.Errorf("Expected 1 orphan, got: %d", result.OrphansFound)
		}

		if result.OrphansRemoved != 1 {
			t.Errorf("Expected 1 orphan removed, got: %d", result.OrphansRemoved)
		}

		if result.BytesFreed != 18 { // "test data 2 longer" = 18 bytes
			t.Errorf("Expected 18 bytes freed, got: %d", result.BytesFreed)
		}

		// Verify the orphan was actually removed
		orphanPath := filepath.Join(tmpDir, "attachments", hash2[:2], hash2, "attachment.bin")
		if _, err := os.Stat(orphanPath); !os.IsNotExist(err) {
			t.Error("Orphan attachment should have been removed")
		}

		// Verify referenced attachment still exists
		refPath := filepath.Join(tmpDir, "attachments", hash1[:2], hash1, "attachment.bin")
		if _, err := os.Stat(refPath); err != nil {
			t.Error("Referenced attachment should still exist")
		}
	})

	t.Run("dry run mode", func(t *testing.T) {
		tmpDir := t.TempDir()

		fs := afero.NewOsFs()
		mgr := NewAttachmentManager(tmpDir, fs)

		// Create orphaned attachment (must be 64-char SHA-256 hash with valid hex prefix)
		hash := "0a0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcd"
		createTestAttachment(t, tmpDir, hash, []byte("orphan data"))

		// Mock SMS reader with no references
		smsReader := &MockSMSReader{
			refs: map[string]bool{},
		}

		remover := NewOrphanRemover(smsReader, mgr, true) // dry run
		result, err := remover.RemoveOrphans(context.Background())

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if result.OrphansFound != 1 {
			t.Errorf("Expected 1 orphan found, got: %d", result.OrphansFound)
		}

		if result.OrphansRemoved != 1 {
			t.Errorf("Expected to report 1 would-be removed, got: %d", result.OrphansRemoved)
		}

		if result.BytesFreed != 11 { // "orphan data" = 11 bytes
			t.Errorf("Expected 11 bytes would be freed, got: %d", result.BytesFreed)
		}

		// Verify the file was NOT actually removed in dry run
		orphanPath := filepath.Join(tmpDir, "attachments", hash[:2], hash, "attachment.bin")
		if _, err := os.Stat(orphanPath); err != nil {
			t.Error("In dry run mode, orphan should NOT be removed")
		}
	})

	t.Run("multiple orphans", func(t *testing.T) {
		tmpDir := t.TempDir()

		fs := afero.NewOsFs()
		mgr := NewAttachmentManager(tmpDir, fs)

		// Create multiple orphans (must be 64-char SHA-256 hashes with valid hex prefixes)
		createTestAttachment(t, tmpDir, "0a0001456789abcdef0123456789abcdef0123456789abcdef0123456789abcd", []byte("data1"))
		createTestAttachment(t, tmpDir, "0b0002456789abcdef0123456789abcdef0123456789abcdef0123456789abcd", []byte("data2"))
		createTestAttachment(t, tmpDir, "0c0003456789abcdef0123456789abcdef0123456789abcdef0123456789abcd", []byte("data3"))

		// Mock SMS reader with no references
		smsReader := &MockSMSReader{
			refs: map[string]bool{},
		}

		remover := NewOrphanRemover(smsReader, mgr, false)
		result, err := remover.RemoveOrphans(context.Background())

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if result.OrphansFound != 3 {
			t.Errorf("Expected 3 orphans, got: %d", result.OrphansFound)
		}

		if result.OrphansRemoved != 3 {
			t.Errorf("Expected 3 orphans removed, got: %d", result.OrphansRemoved)
		}

		if result.BytesFreed != 15 { // 5 + 5 + 5 = 15 bytes
			t.Errorf("Expected 15 bytes freed, got: %d", result.BytesFreed)
		}
	})

	t.Run("removal failure handling", func(t *testing.T) {
		tmpDir := t.TempDir()

		fs := afero.NewOsFs()
		mgr := NewAttachmentManager(tmpDir, fs)

		// Create orphaned attachment (must be 64-char SHA-256 hash with valid hex prefix)
		hash := "0d0004456789abcdef0123456789abcdef0123456789abcdef0123456789abcd"
		createTestAttachment(t, tmpDir, hash, []byte("orphan data"))

		// Make the file read-only to trigger removal failure
		orphanPath := filepath.Join(tmpDir, "attachments", hash[:2], hash, "attachment.bin")
		if err := os.Chmod(filepath.Dir(orphanPath), 0400); err != nil {
			t.Skip("Cannot set read-only permissions on this system")
		}
		defer func() {
			_ = os.Chmod(filepath.Dir(orphanPath), 0750)
		}()

		// Mock SMS reader with no references
		smsReader := &MockSMSReader{
			refs: map[string]bool{},
		}

		remover := NewOrphanRemover(smsReader, mgr, false)
		result, err := remover.RemoveOrphans(context.Background())

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if result.OrphansFound != 1 {
			t.Errorf("Expected 1 orphan found, got: %d", result.OrphansFound)
		}

		// On some systems this might succeed despite read-only parent
		if result.RemovalFailures > 0 {
			if result.OrphansRemoved != 0 {
				t.Error("If removal failed, OrphansRemoved should be 0")
			}

			if len(result.FailedRemovals) != 1 {
				t.Errorf("Expected 1 failed removal recorded, got: %d", len(result.FailedRemovals))
			}
		}
	})
}

func TestOrphanRemover_CleanupEmptyDirectory(t *testing.T) {
	t.Run("removes empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		fs := afero.NewOsFs()
		mgr := NewAttachmentManager(tmpDir, fs)

		// Create empty directory
		emptyDir := filepath.Join(tmpDir, "empty")
		if err := os.MkdirAll(emptyDir, 0750); err != nil {
			t.Fatal(err)
		}

		smsReader := &MockSMSReader{refs: map[string]bool{}}
		remover := NewOrphanRemover(smsReader, mgr, false)

		// Clean up the empty directory
		remover.cleanupEmptyDirectory(emptyDir)

		// Verify it was removed
		if _, err := os.Stat(emptyDir); !os.IsNotExist(err) {
			t.Error("Empty directory should have been removed")
		}
	})

	t.Run("does not remove non-empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		fs := afero.NewOsFs()
		mgr := NewAttachmentManager(tmpDir, fs)

		// Create directory with a file
		nonEmptyDir := filepath.Join(tmpDir, "nonempty")
		if err := os.MkdirAll(nonEmptyDir, 0750); err != nil {
			t.Fatal(err)
		}

		testFile := filepath.Join(nonEmptyDir, "file.txt")
		if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}

		smsReader := &MockSMSReader{refs: map[string]bool{}}
		remover := NewOrphanRemover(smsReader, mgr, false)

		// Attempt to clean up
		remover.cleanupEmptyDirectory(nonEmptyDir)

		// Verify it was NOT removed
		if _, err := os.Stat(nonEmptyDir); err != nil {
			t.Error("Non-empty directory should NOT have been removed")
		}
	})
}

// Helper function to create a test attachment
func createTestAttachment(t *testing.T, repoPath, hash string, data []byte) {
	t.Helper()

	dirPath := filepath.Join(repoPath, "attachments", hash[:2], hash)
	if err := os.MkdirAll(dirPath, 0750); err != nil {
		t.Fatal(err)
	}

	// Create the attachment file with proper filename (use .bin extension)
	filename := "attachment.bin"
	dataPath := filepath.Join(dirPath, filename)
	if err := os.WriteFile(dataPath, data, 0600); err != nil {
		t.Fatal(err)
	}

	// Create metadata file with all required fields
	metadataPath := filepath.Join(dirPath, "metadata.yaml")
	metadata := fmt.Sprintf(`hash: %s
size: %d
mime_type: application/octet-stream
original_name: %s
created_at: 2024-01-01T00:00:00Z
`, hash, len(data), filename)
	if err := os.WriteFile(metadataPath, []byte(metadata), 0600); err != nil {
		t.Fatal(err)
	}
}
