package attachments

import (
	"testing"

	"github.com/spf13/afero"
)

// TestAttachmentManager_GetRepoPath tests the GetRepoPath method
func TestAttachmentManager_GetRepoPath(t *testing.T) {
	fs := afero.NewMemMapFs()
	repoPath := "/test/repo"

	manager := NewAttachmentManager(repoPath, fs)

	result := manager.GetRepoPath()
	if result != repoPath {
		t.Errorf("expected repo path %s, got %s", repoPath, result)
	}
}

// TestAttachmentManager_ReadAttachment_LegacyReadError tests error handling when reading legacy attachment fails
func TestAttachmentManager_ReadAttachment_LegacyReadError(t *testing.T) {
	fs := afero.NewMemMapFs()
	repoPath := "/test/repo"

	// Create attachment manager
	manager := NewAttachmentManager(repoPath, fs)

	// Create legacy attachment directory structure
	hash := "abc123def456"
	prefix := hash[:2]
	dirPath := repoPath + "/attachments/" + prefix
	if err := fs.MkdirAll(dirPath, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Create a legacy attachment file but make it a directory (to cause read error)
	attachmentPath := dirPath + "/" + hash
	if err := fs.Mkdir(attachmentPath, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Try to read the attachment - should fail because it's a directory
	_, err := manager.ReadAttachment(hash)
	if err == nil {
		t.Error("expected error when reading directory as file, got nil")
	}
}

// TestAttachmentExists_Error tests AttachmentExists when GetAttachment returns an error
func TestAttachmentExists_Error(t *testing.T) {
	fs := afero.NewMemMapFs()
	repoPath := "/test/repo"

	manager := NewAttachmentManager(repoPath, fs)

	// Use invalid hash to trigger error
	exists, err := manager.AttachmentExists("invalid")
	if err == nil {
		t.Error("expected error for invalid hash, got nil")
	}
	if exists {
		t.Error("expected exists to be false for invalid hash")
	}
}
