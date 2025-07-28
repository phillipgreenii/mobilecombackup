package attachments

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestAttachmentManager_GetAttachmentPath(t *testing.T) {
	manager := NewAttachmentManager("/tmp/test")

	tests := []struct {
		hash     string
		expected string
	}{
		{"3ceb5c413ee02895bf1f357a8c2cc2bec824f4d8aad13aeab69303f341c8b781", "attachments/3c/3ceb5c413ee02895bf1f357a8c2cc2bec824f4d8aad13aeab69303f341c8b781"},
		{"26fdc315fadc05db9f8f3236fc30b9f0ca044e56ec1e9450ccd5fdab900e9e46", "attachments/26/26fdc315fadc05db9f8f3236fc30b9f0ca044e56ec1e9450ccd5fdab900e9e46"},
		{"3CEB5C413EE02895BF1F357A8C2CC2BEC824F4D8AAD13AEAB69303F341C8B781", "attachments/3c/3ceb5c413ee02895bf1f357a8c2cc2bec824f4d8aad13aeab69303f341c8b781"}, // Uppercase converted to lowercase
		{"", ""},                  // Empty string
		{"a", ""},                 // Too short
		{"ab", "attachments/ab/ab"}, // Short but valid for path generation
	}

	for _, test := range tests {
		result := manager.GetAttachmentPath(test.hash)
		if result != test.expected {
			t.Errorf("GetAttachmentPath(%s): expected %s, got %s", test.hash, test.expected, result)
		}
	}
}

func TestIsValidHash(t *testing.T) {
	tests := []struct {
		hash  string
		valid bool
	}{
		{"3ceb5c413ee02895bf1f357a8c2cc2bec824f4d8aad13aeab69303f341c8b781", true},   // Valid lowercase (64 chars)
		{"3CEB5C413EE02895BF1F357A8C2CC2BEC824F4D8AAD13AEAB69303F341C8B781", false},  // Uppercase not allowed
		{"26fdc315fadc05db9f8f3236fc30b9f0ca044e56ec1e9450ccd5fdab900e9e46", true},   // Valid with numbers (64 chars)
		{"", false},                                                               // Empty
		{"3ceb5c413ee02895bf1f357a8c2cc2bec824f4d8aad13aeab69303f341c8b78", false},    // Too short (63 chars)
		{"3ceb5c413ee02895bf1f357a8c2cc2bec824f4d8aad13aeab69303f341c8b7812", false}, // Too long (65 chars)
		{"3ceb5c413ee02895bf1f357a8c2cc2bec824f4d8aad13aeab69303f341c8b78g", false}, // Invalid character g
		{"3ceb5c413ee02895bf1f357a8c2cc2bec824f4d8aad13aeab69303f341c8b78 ", false}, // Space character
	}

	for _, test := range tests {
		t.Logf("Testing hash: %s (length: %d)", test.hash, len(test.hash))
		result := isValidHash(test.hash)
		if result != test.valid {
			t.Errorf("isValidHash(%s): expected %v, got %v", test.hash, test.valid, result)
		}
	}
}

func TestAttachmentManager_GetAttachment_InvalidHash(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	_, err := manager.GetAttachment("invalid")
	if err == nil {
		t.Error("GetAttachment should fail for invalid hash")
	}
}

func TestAttachmentManager_GetAttachment_NonExistent(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	hash := "3ceb5c413ee02895bf1f357a8c2cc2bec824f4d8aad13aeab69303f341c8b781"
	attachment, err := manager.GetAttachment(hash)
	if err != nil {
		t.Fatalf("GetAttachment failed: %v", err)
	}

	if attachment.Hash != hash {
		t.Errorf("Expected hash %s, got %s", hash, attachment.Hash)
	}
	if attachment.Exists {
		t.Error("Attachment should not exist")
	}
	if attachment.Size != 0 {
		t.Errorf("Expected size 0, got %d", attachment.Size)
	}
}

func TestAttachmentManager_GetAttachment_Existing(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	// Create test content and calculate its hash
	content := []byte("test attachment content")
	hasher := sha256.New()
	hasher.Write(content)
	hash := fmt.Sprintf("%x", hasher.Sum(nil))

	// Create attachment directory structure and file
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

	// Test GetAttachment
	attachment, err := manager.GetAttachment(hash)
	if err != nil {
		t.Fatalf("GetAttachment failed: %v", err)
	}

	if attachment.Hash != hash {
		t.Errorf("Expected hash %s, got %s", hash, attachment.Hash)
	}
	if !attachment.Exists {
		t.Error("Attachment should exist")
	}
	if attachment.Size != int64(len(content)) {
		t.Errorf("Expected size %d, got %d", len(content), attachment.Size)
	}
}

func TestAttachmentManager_ReadAttachment(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	// Create test content and calculate its hash
	content := []byte("test attachment content for reading")
	hasher := sha256.New()
	hasher.Write(content)
	hash := fmt.Sprintf("%x", hasher.Sum(nil))

	// Create attachment directory structure and file
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

	// Test ReadAttachment
	readContent, err := manager.ReadAttachment(hash)
	if err != nil {
		t.Fatalf("ReadAttachment failed: %v", err)
	}

	if string(readContent) != string(content) {
		t.Errorf("Expected content %s, got %s", string(content), string(readContent))
	}
}

func TestAttachmentManager_ReadAttachment_InvalidHash(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	_, err := manager.ReadAttachment("invalid")
	if err == nil {
		t.Error("ReadAttachment should fail for invalid hash")
	}
}

func TestAttachmentManager_ReadAttachment_NonExistent(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	hash := "ab54363e39c1234567890abcdef1234567890abcdef1234567890abcdef123456"
	_, err := manager.ReadAttachment(hash)
	if err == nil {
		t.Error("ReadAttachment should fail for non-existent file")
	}
}

func TestAttachmentManager_VerifyAttachment(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	// Create test content and calculate its hash
	content := []byte("test content for verification")
	hasher := sha256.New()
	hasher.Write(content)
	hash := fmt.Sprintf("%x", hasher.Sum(nil))

	// Create attachment directory structure and file
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

	// Test verification - should pass
	verified, err := manager.VerifyAttachment(hash)
	if err != nil {
		t.Fatalf("VerifyAttachment failed: %v", err)
	}
	if !verified {
		t.Error("Attachment verification should pass for correct hash")
	}

	// Test with wrong hash - create file with different content
	wrongContent := []byte("different content")
	wrongHash := "26fdc315fadc05db9f8f3236fc30b9f0ca044e56ec1e9450ccd5fdab900e9e46"
	wrongDir := filepath.Join(tempDir, "attachments", wrongHash[:2])
	err = os.MkdirAll(wrongDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create wrong attachment directory: %v", err)
	}

	wrongPath := filepath.Join(wrongDir, wrongHash)
	err = os.WriteFile(wrongPath, wrongContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write wrong attachment file: %v", err)
	}

	// Test verification - should fail
	verified, err = manager.VerifyAttachment(wrongHash)
	if err != nil {
		t.Fatalf("VerifyAttachment failed: %v", err)
	}
	if verified {
		t.Error("Attachment verification should fail for incorrect hash")
	}
}

func TestAttachmentManager_AttachmentExists(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	hash := "3ceb5c413ee02895bf1f357a8c2cc2bec824f4d8aad13aeab69303f341c8b781"

	// Test non-existent
	exists, err := manager.AttachmentExists(hash)
	if err != nil {
		t.Fatalf("AttachmentExists failed: %v", err)
	}
	if exists {
		t.Error("AttachmentExists should return false for non-existent file")
	}

	// Create the attachment
	content := []byte("test content")
	attachmentDir := filepath.Join(tempDir, "attachments", hash[:2])
	err = os.MkdirAll(attachmentDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create attachment directory: %v", err)
	}

	attachmentPath := filepath.Join(attachmentDir, hash)
	err = os.WriteFile(attachmentPath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to write attachment file: %v", err)
	}

	// Test existing
	exists, err = manager.AttachmentExists(hash)
	if err != nil {
		t.Fatalf("AttachmentExists failed: %v", err)
	}
	if !exists {
		t.Error("AttachmentExists should return true for existing file")
	}
}

func TestAttachmentManager_StreamAttachments_EmptyRepository(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	var attachmentCount int
	err := manager.StreamAttachments(func(attachment *Attachment) error {
		attachmentCount++
		return nil
	})

	if err != nil {
		t.Fatalf("StreamAttachments failed: %v", err)
	}
	if attachmentCount != 0 {
		t.Errorf("Expected 0 attachments, got %d", attachmentCount)
	}
}

func TestAttachmentManager_StreamAttachments_WithAttachments(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	// Create test attachments
	testAttachments := []struct {
		content []byte
		hash    string
	}{
		{[]byte("content1"), ""},
		{[]byte("content2"), ""},
		{[]byte("content3"), ""},
	}

	// Calculate hashes and create files
	for i := range testAttachments {
		hasher := sha256.New()
		hasher.Write(testAttachments[i].content)
		testAttachments[i].hash = fmt.Sprintf("%x", hasher.Sum(nil))

		attachmentDir := filepath.Join(tempDir, "attachments", testAttachments[i].hash[:2])
		err := os.MkdirAll(attachmentDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create attachment directory: %v", err)
		}

		attachmentPath := filepath.Join(attachmentDir, testAttachments[i].hash)
		err = os.WriteFile(attachmentPath, testAttachments[i].content, 0644)
		if err != nil {
			t.Fatalf("Failed to write attachment file: %v", err)
		}
	}

	// Test streaming
	var streamedAttachments []*Attachment
	err := manager.StreamAttachments(func(attachment *Attachment) error {
		streamedAttachments = append(streamedAttachments, attachment)
		return nil
	})

	if err != nil {
		t.Fatalf("StreamAttachments failed: %v", err)
	}
	if len(streamedAttachments) != len(testAttachments) {
		t.Errorf("Expected %d attachments, got %d", len(testAttachments), len(streamedAttachments))
	}

	// Verify all attachments were found
	foundHashes := make(map[string]bool)
	for _, attachment := range streamedAttachments {
		foundHashes[attachment.Hash] = true
		if !attachment.Exists {
			t.Errorf("Attachment %s should exist", attachment.Hash)
		}
	}

	for _, testAttachment := range testAttachments {
		if !foundHashes[testAttachment.hash] {
			t.Errorf("Attachment %s not found in stream", testAttachment.hash)
		}
	}
}

func TestAttachmentManager_ListAttachments(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	// Create a test attachment
	content := []byte("list test content")
	hasher := sha256.New()
	hasher.Write(content)
	hash := fmt.Sprintf("%x", hasher.Sum(nil))

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

	// Test ListAttachments
	attachments, err := manager.ListAttachments()
	if err != nil {
		t.Fatalf("ListAttachments failed: %v", err)
	}

	if len(attachments) != 1 {
		t.Errorf("Expected 1 attachment, got %d", len(attachments))
	}

	if len(attachments) > 0 {
		attachment := attachments[0]
		if attachment.Hash != hash {
			t.Errorf("Expected hash %s, got %s", hash, attachment.Hash)
		}
		if !attachment.Exists {
			t.Error("Attachment should exist")
		}
		if attachment.Size != int64(len(content)) {
			t.Errorf("Expected size %d, got %d", len(content), attachment.Size)
		}
	}
}

func TestAttachmentManager_FindOrphanedAttachments(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	// Create test attachments
	testContents := [][]byte{
		[]byte("referenced content"),
		[]byte("orphaned content"),
	}

	var hashes []string
	for _, content := range testContents {
		hasher := sha256.New()
		hasher.Write(content)
		hash := fmt.Sprintf("%x", hasher.Sum(nil))
		hashes = append(hashes, hash)

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

	// Create referenced hashes map (only include first hash)
	referencedHashes := map[string]bool{
		hashes[0]: true,
	}

	// Find orphaned attachments
	orphaned, err := manager.FindOrphanedAttachments(referencedHashes)
	if err != nil {
		t.Fatalf("FindOrphanedAttachments failed: %v", err)
	}

	if len(orphaned) != 1 {
		t.Errorf("Expected 1 orphaned attachment, got %d", len(orphaned))
	}

	if len(orphaned) > 0 {
		if orphaned[0].Hash != hashes[1] {
			t.Errorf("Expected orphaned hash %s, got %s", hashes[1], orphaned[0].Hash)
		}
	}
}

func TestAttachmentManager_ValidateAttachmentStructure_EmptyRepository(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	err := manager.ValidateAttachmentStructure()
	if err != nil {
		t.Errorf("ValidateAttachmentStructure should not fail for empty repository: %v", err)
	}
}

func TestAttachmentManager_ValidateAttachmentStructure_ValidStructure(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	// Create valid attachment structure
	content := []byte("valid structure test")
	hasher := sha256.New()
	hasher.Write(content)
	hash := fmt.Sprintf("%x", hasher.Sum(nil))

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

	err = manager.ValidateAttachmentStructure()
	if err != nil {
		t.Errorf("ValidateAttachmentStructure should pass for valid structure: %v", err)
	}
}

func TestAttachmentManager_ValidateAttachmentStructure_InvalidDirectory(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	// Create invalid directory name (3 characters)
	invalidDir := filepath.Join(tempDir, "attachments", "abc")
	err := os.MkdirAll(invalidDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create invalid directory: %v", err)
	}

	err = manager.ValidateAttachmentStructure()
	if err == nil {
		t.Error("ValidateAttachmentStructure should fail for invalid directory name length")
	}
}

func TestAttachmentManager_ValidateAttachmentStructure_InvalidDirectoryFormat(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	// Create invalid directory name (uppercase)
	invalidDir := filepath.Join(tempDir, "attachments", "AB")
	err := os.MkdirAll(invalidDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create invalid directory: %v", err)
	}

	err = manager.ValidateAttachmentStructure()
	if err == nil {
		t.Error("ValidateAttachmentStructure should fail for invalid directory name format")
	}
}

func TestAttachmentManager_ValidateAttachmentStructure_FileInRoot(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	// Create file in attachments root (should be error)
	attachmentsDir := filepath.Join(tempDir, "attachments")
	err := os.MkdirAll(attachmentsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create attachments directory: %v", err)
	}

	invalidFile := filepath.Join(attachmentsDir, "invalid_file.txt")
	err = os.WriteFile(invalidFile, []byte("should not be here"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}

	err = manager.ValidateAttachmentStructure()
	if err == nil {
		t.Error("ValidateAttachmentStructure should fail for file in attachments root")
	}
}

func TestAttachmentManager_ValidateAttachmentStructure_MisplacedFile(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	// Create file in wrong directory (hash doesn't start with directory name)
	wrongDir := filepath.Join(tempDir, "attachments", "ab")
	err := os.MkdirAll(wrongDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Create file with hash that doesn't start with "ab"
	wrongFile := filepath.Join(wrongDir, "cd1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcd")
	err = os.WriteFile(wrongFile, []byte("misplaced"), 0644)
	if err != nil {
		t.Fatalf("Failed to write misplaced file: %v", err)
	}

	err = manager.ValidateAttachmentStructure()
	if err == nil {
		t.Error("ValidateAttachmentStructure should fail for misplaced file")
	}
}

func TestAttachmentManager_GetAttachmentStats(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewAttachmentManager(tempDir)

	// Create test attachments
	testContents := [][]byte{
		[]byte("referenced content 1"),
		[]byte("referenced content 2"),
		[]byte("orphaned content"),
	}

	var hashes []string
	for _, content := range testContents {
		hasher := sha256.New()
		hasher.Write(content)
		hash := fmt.Sprintf("%x", hasher.Sum(nil))
		hashes = append(hashes, hash)

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

	// Create referenced hashes map (only include first two hashes)
	referencedHashes := map[string]bool{
		hashes[0]: true,
		hashes[1]: true,
	}

	// Get stats
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

	expectedTotalSize := int64(len(testContents[0]) + len(testContents[1]) + len(testContents[2]))
	if stats.TotalSize != expectedTotalSize {
		t.Errorf("Expected total size %d, got %d", expectedTotalSize, stats.TotalSize)
	}
}