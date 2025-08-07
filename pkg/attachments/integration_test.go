package attachments

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// copyFile copies a file from src to dst, creating directories as needed
func copyFile(src, dst string) error {
	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func TestAttachmentManager_Integration_WithTestData(t *testing.T) {
	// Check if test data exists
	testDataPath := "../../testdata/it/scenerio-00/original_repo_root/attachments"
	if _, err := os.Stat(testDataPath); os.IsNotExist(err) {
		t.Skip("Integration test data not available")
	}

	tempDir := t.TempDir()
	attachmentsPath := filepath.Join(tempDir, "attachments")

	// Copy test attachments to temp directory
	var hasFiles bool
	err := filepath.Walk(testDataPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from source
		relPath, err := filepath.Rel(testDataPath, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(attachmentsPath, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		hasFiles = true
		return copyFile(path, destPath)
	})

	if err != nil {
		t.Fatalf("Failed to copy test data: %v", err)
	}

	if !hasFiles {
		t.Skip("No attachment files found in test data")
	}

	manager := NewAttachmentManager(tempDir)

	// Test basic functionality with copied data
	attachments, err := manager.ListAttachments()
	if err != nil {
		t.Fatalf("ListAttachments failed with test data: %v", err)
	}

	// Verify we loaded some attachments
	if len(attachments) == 0 {
		t.Error("Expected to find some attachments in test data")
	}

	// Test that all attachments are accessible and verify their integrity
	for _, attachment := range attachments {
		// Test existence
		exists, err := manager.AttachmentExists(attachment.Hash)
		if err != nil {
			t.Errorf("AttachmentExists failed for %s: %v", attachment.Hash, err)
			continue
		}
		if !exists {
			t.Errorf("Attachment %s should exist", attachment.Hash)
			continue
		}

		// Test reading content
		content, err := manager.ReadAttachment(attachment.Hash)
		if err != nil {
			t.Errorf("ReadAttachment failed for %s: %v", attachment.Hash, err)
			continue
		}

		if int64(len(content)) != attachment.Size {
			t.Errorf("Content size mismatch for %s: expected %d, got %d", 
				attachment.Hash, attachment.Size, len(content))
		}

		// Test verification
		verified, err := manager.VerifyAttachment(attachment.Hash)
		if err != nil {
			t.Errorf("VerifyAttachment failed for %s: %v", attachment.Hash, err)
			continue
		}
		if !verified {
			t.Errorf("Attachment %s failed verification", attachment.Hash)
		}
	}
}

func TestAttachmentManager_Integration_EmptyRepository(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	// Test with empty repository (no attachments directory)
	attachments, err := manager.ListAttachments()
	if err != nil {
		t.Fatalf("ListAttachments should not fail for empty repository: %v", err)
	}

	if len(attachments) != 0 {
		t.Error("Empty repository should have 0 attachments")
	}

	// Test validation on empty repository
	err = manager.ValidateAttachmentStructure()
	if err != nil {
		t.Errorf("ValidateAttachmentStructure should not fail for empty repository: %v", err)
	}

	// Test statistics on empty repository
	stats, err := manager.GetAttachmentStats(make(map[string]bool))
	if err != nil {
		t.Fatalf("GetAttachmentStats failed: %v", err)
	}

	if stats.TotalCount != 0 || stats.TotalSize != 0 || stats.OrphanedCount != 0 || stats.CorruptedCount != 0 {
		t.Errorf("Expected all stats to be 0 for empty repository, got %+v", stats)
	}
}

func TestAttachmentManager_Integration_LargeRepository(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	// Create a repository with many attachments for performance testing
	numAttachments := 100
	attachmentHashes := make([]string, numAttachments)
	
	for i := 0; i < numAttachments; i++ {
		// Create unique content
		content := []byte(fmt.Sprintf("test attachment content %d", i))
		
		// Calculate hash
		hasher := sha256.New()
		hasher.Write(content)
		hash := fmt.Sprintf("%x", hasher.Sum(nil))
		attachmentHashes[i] = hash

		// Create attachment directory and file
		attachmentDir := filepath.Join(tempDir, "attachments", hash[:2])
		err := os.MkdirAll(attachmentDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create attachment directory: %v", err)
		}

		attachmentPath := filepath.Join(attachmentDir, hash)
		err = os.WriteFile(attachmentPath, content, 0644)
		if err != nil {
			t.Fatalf("Failed to write attachment file: %v", err)
		}
	}

	// Test ListAttachments performance
	attachments, err := manager.ListAttachments()
	if err != nil {
		t.Fatalf("ListAttachments failed with large repository: %v", err)
	}

	if len(attachments) != numAttachments {
		t.Errorf("Expected %d attachments, got %d", numAttachments, len(attachments))
	}

	// Test streaming performance
	var streamedCount int
	err = manager.StreamAttachments(func(attachment *Attachment) error {
		streamedCount++
		return nil
	})

	if err != nil {
		t.Fatalf("StreamAttachments failed: %v", err)
	}

	if streamedCount != numAttachments {
		t.Errorf("Expected to stream %d attachments, got %d", numAttachments, streamedCount)
	}

	// Test statistics with large repository
	referencedHashes := make(map[string]bool)
	// Mark half as referenced
	for i := 0; i < numAttachments/2; i++ {
		referencedHashes[attachmentHashes[i]] = true
	}

	stats, err := manager.GetAttachmentStats(referencedHashes)
	if err != nil {
		t.Fatalf("GetAttachmentStats failed: %v", err)
	}

	if stats.TotalCount != numAttachments {
		t.Errorf("Expected total count %d, got %d", numAttachments, stats.TotalCount)
	}

	expectedOrphaned := numAttachments - numAttachments/2
	if stats.OrphanedCount != expectedOrphaned {
		t.Errorf("Expected orphaned count %d, got %d", expectedOrphaned, stats.OrphanedCount)
	}
}

func TestAttachmentManager_Integration_CrossReference(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	// Create test attachments
	testContents := []string{
		"image content 1",
		"image content 2", 
		"orphaned content",
	}

	var hashes []string
	for _, content := range testContents {
		hasher := sha256.New()
		hasher.Write([]byte(content))
		hash := fmt.Sprintf("%x", hasher.Sum(nil))
		hashes = append(hashes, hash)

		attachmentDir := filepath.Join(tempDir, "attachments", hash[:2])
		err := os.MkdirAll(attachmentDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create attachment directory: %v", err)
		}

		attachmentPath := filepath.Join(attachmentDir, hash)
		err = os.WriteFile(attachmentPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write attachment file: %v", err)
		}
	}

	// Simulate SMS reader providing referenced hashes (first two attachments)
	referencedHashes := map[string]bool{
		hashes[0]: true,
		hashes[1]: true,
	}

	// Test orphaned detection
	orphaned, err := manager.FindOrphanedAttachments(referencedHashes)
	if err != nil {
		t.Fatalf("FindOrphanedAttachments failed: %v", err)
	}

	if len(orphaned) != 1 {
		t.Errorf("Expected 1 orphaned attachment, got %d", len(orphaned))
	}

	if len(orphaned) > 0 && orphaned[0].Hash != hashes[2] {
		t.Errorf("Expected orphaned hash %s, got %s", hashes[2], orphaned[0].Hash)
	}

	// Test statistics with cross-reference
	stats, err := manager.GetAttachmentStats(referencedHashes)
	if err != nil {
		t.Fatalf("GetAttachmentStats failed: %v", err)
	}

	if stats.TotalCount != 3 {
		t.Errorf("Expected total count 3, got %d", stats.TotalCount)
	}

	if stats.OrphanedCount != 1 {
		t.Errorf("Expected orphaned count 1, got %d", stats.OrphanedCount)
	}

	if stats.CorruptedCount != 0 {
		t.Errorf("Expected corrupted count 0, got %d", stats.CorruptedCount)
	}
}

func TestAttachmentManager_Integration_CorruptedFiles(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	// Create a corrupted attachment (content doesn't match hash)
	content := []byte("correct content")
	wrongHash := "3ceb5c413ee02895bf1f357a8c2cc2bec824f4d8aad13aeab69303f341c8b781" // Hash for different content

	attachmentDir := filepath.Join(tempDir, "attachments", wrongHash[:2])
	err := os.MkdirAll(attachmentDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create attachment directory: %v", err)
	}

	attachmentPath := filepath.Join(attachmentDir, wrongHash)
	err = os.WriteFile(attachmentPath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to write corrupted attachment: %v", err)
	}

	// Test verification fails
	verified, err := manager.VerifyAttachment(wrongHash)
	if err != nil {
		t.Fatalf("VerifyAttachment failed: %v", err)
	}

	if verified {
		t.Error("Verification should fail for corrupted attachment")
	}

	// Test statistics detect corruption
	stats, err := manager.GetAttachmentStats(make(map[string]bool))
	if err != nil {
		t.Fatalf("GetAttachmentStats failed: %v", err)
	}

	if stats.CorruptedCount != 1 {
		t.Errorf("Expected corrupted count 1, got %d", stats.CorruptedCount)
	}
}

func TestAttachmentManager_Integration_ValidationErrors(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	// Create various validation problems
	attachmentsDir := filepath.Join(tempDir, "attachments")

	// 1. File in root
	err := os.MkdirAll(attachmentsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create attachments directory: %v", err)
	}

	rootFile := filepath.Join(attachmentsDir, "invalid.txt")
	err = os.WriteFile(rootFile, []byte("should not be here"), 0644)
	if err != nil {
		t.Fatalf("Failed to create root file: %v", err)
	}

	// 2. Invalid directory name
	invalidDir := filepath.Join(attachmentsDir, "invalid_name")
	err = os.MkdirAll(invalidDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create invalid directory: %v", err)
	}

	// 3. Misplaced file
	validDir := filepath.Join(attachmentsDir, "ab")
	err = os.MkdirAll(validDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create valid directory: %v", err)
	}

	misplacedFile := filepath.Join(validDir, "cd1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcd")
	err = os.WriteFile(misplacedFile, []byte("misplaced"), 0644)
	if err != nil {
		t.Fatalf("Failed to create misplaced file: %v", err)
	}

	// Test validation catches all problems
	err = manager.ValidateAttachmentStructure()
	if err == nil {
		t.Error("ValidateAttachmentStructure should fail with multiple validation problems")
	}

	// Error message should contain all the problems
	errorMsg := err.Error()
	expectedSubstrings := []string{
		"file found in attachments root",
		"invalid directory name",
		"misplaced file",
	}

	for _, substring := range expectedSubstrings {
		if !contains(errorMsg, substring) {
			t.Errorf("Error message should contain '%s', got: %s", substring, errorMsg)
		}
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(substr) <= len(s) && containsRec(s, substr, 0)))
}

func containsRec(s, substr string, index int) bool {
	if index+len(substr) > len(s) {
		return false
	}
	if s[index:index+len(substr)] == substr {
		return true
	}
	return containsRec(s, substr, index+1)
}