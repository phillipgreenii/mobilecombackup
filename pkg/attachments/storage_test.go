package attachments

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDirectoryAttachmentStorage_Store(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "storage_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	storage := NewDirectoryAttachmentStorage(tmpDir)

	// Test data
	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	data := []byte("test content")
	metadata := AttachmentInfo{
		Hash:         hash,
		OriginalName: "test.txt",
		MimeType:     "text/plain",
		Size:         int64(len(data)),
		CreatedAt:    time.Now().UTC(),
		SourceMMS:    "test-mms",
	}

	// Test storing attachment
	err = storage.Store(hash, data, metadata)
	if err != nil {
		t.Fatalf("Failed to store attachment: %v", err)
	}

	// Verify directory structure
	expectedDir := filepath.Join(tmpDir, "attachments", "e3", hash)
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("Expected directory not created: %s", expectedDir)
	}

	// Verify attachment file
	attachmentFile := filepath.Join(expectedDir, "test.txt")
	if _, err := os.Stat(attachmentFile); os.IsNotExist(err) {
		t.Errorf("Expected attachment file not created: %s", attachmentFile)
	}

	// Verify metadata file
	metadataFile := filepath.Join(expectedDir, "metadata.yaml")
	if _, err := os.Stat(metadataFile); os.IsNotExist(err) {
		t.Errorf("Expected metadata file not created: %s", metadataFile)
	}

	// Verify file content
	storedData, err := os.ReadFile(attachmentFile)
	if err != nil {
		t.Fatalf("Failed to read stored file: %v", err)
	}
	if string(storedData) != string(data) {
		t.Errorf("Stored content mismatch: expected %s, got %s", string(data), string(storedData))
	}
}

func TestDirectoryAttachmentStorage_GetMetadata(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "storage_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	storage := NewDirectoryAttachmentStorage(tmpDir)

	// Test data
	hash := "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3"
	data := []byte("hello")
	metadata := AttachmentInfo{
		Hash:         hash,
		OriginalName: "hello.txt",
		MimeType:     "text/plain",
		Size:         int64(len(data)),
		CreatedAt:    time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		SourceMMS:    "test-source",
	}

	// Store attachment
	err = storage.Store(hash, data, metadata)
	if err != nil {
		t.Fatalf("Failed to store attachment: %v", err)
	}

	// Test retrieving metadata
	retrievedMetadata, err := storage.GetMetadata(hash)
	if err != nil {
		t.Fatalf("Failed to get metadata: %v", err)
	}

	// Verify metadata
	if retrievedMetadata.Hash != metadata.Hash {
		t.Errorf("Hash mismatch: expected %s, got %s", metadata.Hash, retrievedMetadata.Hash)
	}
	if retrievedMetadata.OriginalName != metadata.OriginalName {
		t.Errorf("OriginalName mismatch: expected %s, got %s", metadata.OriginalName, retrievedMetadata.OriginalName)
	}
	if retrievedMetadata.MimeType != metadata.MimeType {
		t.Errorf("MimeType mismatch: expected %s, got %s", metadata.MimeType, retrievedMetadata.MimeType)
	}
	if retrievedMetadata.Size != metadata.Size {
		t.Errorf("Size mismatch: expected %d, got %d", metadata.Size, retrievedMetadata.Size)
	}
	if retrievedMetadata.SourceMMS != metadata.SourceMMS {
		t.Errorf("SourceMMS mismatch: expected %s, got %s", metadata.SourceMMS, retrievedMetadata.SourceMMS)
	}
}

func TestDirectoryAttachmentStorage_GetPath(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "storage_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	storage := NewDirectoryAttachmentStorage(tmpDir)

	// Test data
	hash := "2cf24dba4f21d4288674d04eb6f1a906d4da8e88ba3b948ba84e0fb0e6d31e7"
	data := []byte("hello world")
	metadata := AttachmentInfo{
		Hash:         hash,
		OriginalName: "document.pdf",
		MimeType:     "application/pdf",
		Size:         int64(len(data)),
		CreatedAt:    time.Now().UTC(),
	}

	// Store attachment
	err = storage.Store(hash, data, metadata)
	if err != nil {
		t.Fatalf("Failed to store attachment: %v", err)
	}

	// Test getting path
	path, err := storage.GetPath(hash)
	if err != nil {
		t.Fatalf("Failed to get path: %v", err)
	}

	expectedPath := filepath.Join("attachments", "2c", hash, "document.pdf")
	if path != expectedPath {
		t.Errorf("Path mismatch: expected %s, got %s", expectedPath, path)
	}
}

func TestDirectoryAttachmentStorage_Exists(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "storage_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	storage := NewDirectoryAttachmentStorage(tmpDir)

	// Test non-existent attachment
	hash := "non-existent-hash"
	if storage.Exists(hash) {
		t.Error("Expected attachment to not exist")
	}

	// Test existing attachment
	existingHash := "e258d248fda94c63753607f7c4494ee0fcbe92f1a76bfdac795c9d84101eb317"
	data := []byte("test data")
	metadata := AttachmentInfo{
		Hash:         existingHash,
		OriginalName: "test.bin",
		MimeType:     "application/octet-stream",
		Size:         int64(len(data)),
		CreatedAt:    time.Now().UTC(),
	}

	err = storage.Store(existingHash, data, metadata)
	if err != nil {
		t.Fatalf("Failed to store attachment: %v", err)
	}

	if !storage.Exists(existingHash) {
		t.Error("Expected attachment to exist")
	}
}

func TestDirectoryAttachmentStorage_ReadAttachment(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "storage_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	storage := NewDirectoryAttachmentStorage(tmpDir)

	// Test data
	hash := "315f5bdb76d078c43b8ac0064e4a0164612b1fce77c869345bfc94c75894edd3"
	originalData := []byte("this is test content for reading")
	metadata := AttachmentInfo{
		Hash:         hash,
		OriginalName: "readme.txt",
		MimeType:     "text/plain",
		Size:         int64(len(originalData)),
		CreatedAt:    time.Now().UTC(),
	}

	// Store attachment
	err = storage.Store(hash, originalData, metadata)
	if err != nil {
		t.Fatalf("Failed to store attachment: %v", err)
	}

	// Read attachment
	readData, err := storage.ReadAttachment(hash)
	if err != nil {
		t.Fatalf("Failed to read attachment: %v", err)
	}

	// Verify content
	if string(readData) != string(originalData) {
		t.Errorf("Content mismatch: expected %s, got %s", string(originalData), string(readData))
	}
}

func TestDirectoryAttachmentStorage_StoreFromReader(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "storage_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	storage := NewDirectoryAttachmentStorage(tmpDir)

	// Test data
	hash := "03ac674216f3e15c761ee1a5e255f067953623c8b388b4459e13f978d7c846f4"
	originalData := []byte("hello from reader")
	reader := strings.NewReader(string(originalData))
	metadata := AttachmentInfo{
		Hash:         hash,
		OriginalName: "from_reader.txt",
		MimeType:     "text/plain",
		Size:         int64(len(originalData)),
		CreatedAt:    time.Now().UTC(),
	}

	// Store from reader
	err = storage.StoreFromReader(hash, reader, metadata)
	if err != nil {
		t.Fatalf("Failed to store from reader: %v", err)
	}

	// Verify storage
	if !storage.Exists(hash) {
		t.Error("Expected attachment to exist after storing from reader")
	}

	// Read back and verify
	readData, err := storage.ReadAttachment(hash)
	if err != nil {
		t.Fatalf("Failed to read attachment: %v", err)
	}

	if string(readData) != string(originalData) {
		t.Errorf("Content mismatch: expected %s, got %s", string(originalData), string(readData))
	}
}

func TestDirectoryAttachmentStorage_GenerateFilename(t *testing.T) {
	tests := []struct {
		name         string
		originalName string
		mimeType     string
		expected     string
	}{
		{"With original name", "photo.jpg", "image/jpeg", "photo.jpg"},
		{"Without original name - JPEG", "", "image/jpeg", "attachment.jpg"},
		{"Without original name - PNG", "", "image/png", "attachment.png"},
		{"Without original name - PDF", "", "application/pdf", "attachment.pdf"},
		{"Without original name - Unknown", "", "application/unknown", "attachment.bin"},
		{"Null original name", "null", "image/png", "attachment.png"},
		{"Empty mime type", "test.txt", "", "test.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateFilename(tt.originalName, tt.mimeType)
			if result != tt.expected {
				t.Errorf("GenerateFilename(%s, %s) = %s, want %s",
					tt.originalName, tt.mimeType, result, tt.expected)
			}
		})
	}
}

func TestGetFileExtension(t *testing.T) {
	tests := []struct {
		mimeType string
		expected string
	}{
		{"image/jpeg", "jpg"},
		{"image/png", "png"},
		{"application/pdf", "pdf"},
		{"video/mp4", "mp4"},
		{"audio/mpeg", "mp3"},
		{"text/plain", "bin"}, // Not in our mapping
		{"application/unknown", "bin"},
		{"", "bin"},
		{"image/jpeg; charset=utf-8", "jpg"}, // With parameters
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			result := GetFileExtension(tt.mimeType)
			if result != tt.expected {
				t.Errorf("GetFileExtension(%s) = %s, want %s",
					tt.mimeType, result, tt.expected)
			}
		})
	}
}

func TestDirectoryAttachmentStorage_ErrorHandling(t *testing.T) {
	// Test with invalid path
	storage := NewDirectoryAttachmentStorage("/invalid/path/that/should/not/exist")

	hash := "test-hash"
	data := []byte("test")
	metadata := AttachmentInfo{
		Hash:      hash,
		MimeType:  "text/plain",
		Size:      int64(len(data)),
		CreatedAt: time.Now().UTC(),
	}

	// Should fail to store
	err := storage.Store(hash, data, metadata)
	if err == nil {
		t.Error("Expected error when storing to invalid path")
	}

	// Should fail to read metadata
	_, err = storage.GetMetadata(hash)
	if err == nil {
		t.Error("Expected error when reading metadata from invalid path")
	}

	// Should fail to get path
	_, err = storage.GetPath(hash)
	if err == nil {
		t.Error("Expected error when getting path for non-existent attachment")
	}

	// Should return false for exists
	if storage.Exists(hash) {
		t.Error("Expected false for exists check on invalid path")
	}

	// Should fail to read attachment
	_, err = storage.ReadAttachment(hash)
	if err == nil {
		t.Error("Expected error when reading from invalid path")
	}
}

func TestDirectoryAttachmentStorage_StoreFromReader_ErrorHandling(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "storage_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	storage := NewDirectoryAttachmentStorage(tmpDir)

	// Test with reader that fails
	failingReader := &failingReader{}
	hash := "test-hash"
	metadata := AttachmentInfo{
		Hash:      hash,
		MimeType:  "text/plain",
		Size:      0,
		CreatedAt: time.Now().UTC(),
	}

	err = storage.StoreFromReader(hash, failingReader, metadata)
	if err == nil {
		t.Error("Expected error when storing from failing reader")
	}
}

// failingReader is a test helper that always returns an error
type failingReader struct{}

func (fr *failingReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}
