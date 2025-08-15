package sms

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEdgeCase_EmptyAttachment(t *testing.T) {
	extractor := NewAttachmentExtractor(t.TempDir())
	config := GetDefaultContentTypeConfig()

	part := &MMSPart{
		ContentType: "image/png",
		Data:        "",
	}

	result, err := extractor.ExtractAttachmentFromPart(part, config)
	if err != nil {
		t.Fatalf("Expected no error for empty data, got: %v", err)
	}

	if result.Action != ActionSkipped {
		t.Errorf("Expected skipped action for empty data, got: %s", result.Action)
	}
	if result.Reason != "no-data" {
		t.Errorf("Expected 'no-data' reason, got: %s", result.Reason)
	}
}

func TestEdgeCase_NullAttachment(t *testing.T) {
	extractor := NewAttachmentExtractor(t.TempDir())
	config := GetDefaultContentTypeConfig()

	part := &MMSPart{
		ContentType: "image/png",
		Data:        "null",
	}

	result, err := extractor.ExtractAttachmentFromPart(part, config)
	if err != nil {
		t.Fatalf("Expected no error for null data, got: %v", err)
	}

	if result.Action != ActionSkipped {
		t.Errorf("Expected skipped action for null data, got: %s", result.Action)
	}
	if result.Reason != "no-data" {
		t.Errorf("Expected 'no-data' reason, got: %s", result.Reason)
	}
}

func TestEdgeCase_CorruptBase64(t *testing.T) {
	extractor := NewAttachmentExtractor(t.TempDir())
	config := GetDefaultContentTypeConfig()

	tests := []struct {
		name string
		data string
	}{
		{"Invalid characters", strings.Repeat("invalid-chars-@#$%!!!", 50)},
		{"Wrong padding", "ABCD" + strings.Repeat("@", 100)}, // Invalid characters mixed in
		{"Truncated data", createTestPNGData()[:len(createTestPNGData())-10]},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			part := &MMSPart{
				ContentType: "image/png",
				Data:        tt.data,
			}

			result, err := extractor.ExtractAttachmentFromPart(part, config)

			// Some invalid base64 may be skipped due to size, others may fail to decode
			if err == nil {
				// If no error, it should have been skipped for another reason
				if result.Action != ActionSkipped {
					t.Errorf("Expected corrupt data to be skipped or error, got action: %s", result.Action)
				}
			} else {
				// If error, it should be a base64 decode error
				if !strings.Contains(err.Error(), "failed to decode base64 data") {
					t.Errorf("Expected base64 decode error, got: %v", err)
				}
			}
		})
	}
}

func TestEdgeCase_VeryLargeAttachment(t *testing.T) {
	extractor := NewAttachmentExtractor(t.TempDir())
	config := GetDefaultContentTypeConfig()

	// Create a large attachment (1MB of data)
	largeData := make([]byte, 1024*1024)
	_, err := rand.Read(largeData)
	if err != nil {
		t.Fatalf("Failed to generate random data: %v", err)
	}

	part := &MMSPart{
		ContentType: "image/jpeg",
		Filename:    "large_photo.jpg",
		Data:        base64.StdEncoding.EncodeToString(largeData),
	}

	result, err := extractor.ExtractAttachmentFromPart(part, config)
	if err != nil {
		t.Fatalf("Large attachment extraction failed: %v", err)
	}

	if result.Action != ActionExtracted {
		t.Errorf("Expected large attachment to be extracted, got: %s", result.Action)
	}
	if result.OriginalSize != int64(len(largeData)) {
		t.Errorf("Expected original size %d, got %d", len(largeData), result.OriginalSize)
	}

	// Verify file was created and has correct size
	tempDir := extractor.repoRoot
	fullPath := filepath.Join(tempDir, result.Path)
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		t.Fatalf("Large attachment file not found: %v", err)
	}

	if fileInfo.Size() != int64(len(largeData)) {
		t.Errorf("File size mismatch: expected %d, got %d", len(largeData), fileInfo.Size())
	}
}

func TestEdgeCase_ReadOnlyDirectory(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping read-only test when running as root")
	}

	tempDir := t.TempDir()

	// Make attachments directory read-only
	attachmentsDir := filepath.Join(tempDir, "attachments")
	_ = os.MkdirAll(attachmentsDir, 0750)
	_ = os.Chmod(attachmentsDir, 0444) // nolint:gosec // Intentionally restrictive for testing

	// Restore permissions after test
	defer func() { _ = os.Chmod(attachmentsDir, 0750) }() // nolint:gosec // Cleanup permissions

	extractor := NewAttachmentExtractor(tempDir)
	config := GetDefaultContentTypeConfig()

	part := &MMSPart{
		ContentType: "image/png",
		Data:        createTestPNGData(),
	}

	_, err := extractor.ExtractAttachmentFromPart(part, config)
	if err == nil {
		t.Fatal("Expected error for read-only directory")
	}

	// The error could be at directory creation or file checking stage
	if !strings.Contains(err.Error(), "failed to create attachment directory") &&
		!strings.Contains(err.Error(), "failed to check attachment existence") {
		t.Errorf("Expected permission-related error, got: %v", err)
	}
}

func TestEdgeCase_DiskFull(t *testing.T) {
	// This test is difficult to simulate reliably across different systems
	// We'll test a related scenario: permission denied on file write
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tempDir := t.TempDir()
	extractor := NewAttachmentExtractor(tempDir)
	config := GetDefaultContentTypeConfig()

	// First, extract an attachment to create the directory structure
	part := &MMSPart{
		ContentType: "image/png",
		Data:        createTestPNGData(),
	}

	result, err := extractor.ExtractAttachmentFromPart(part, config)
	if err != nil {
		t.Fatalf("Initial extraction failed: %v", err)
	}

	// Now make the attachment subdirectory read-only
	attachmentPath := filepath.Join(tempDir, result.Path)
	subDir := filepath.Dir(attachmentPath)
	_ = os.Chmod(subDir, 0444) // nolint:gosec // Intentionally restrictive for testing

	// Restore permissions after test
	defer func() { _ = os.Chmod(subDir, 0750) }() // nolint:gosec // Cleanup permissions

	// Try to extract a different attachment to the same subdirectory
	newPart := &MMSPart{
		ContentType: "image/jpeg",
		Data:        createTestJPEGData(),
	}

	_, err = extractor.ExtractAttachmentFromPart(newPart, config)
	if err == nil {
		t.Skip("Permission test doesn't work in this environment - skipping")
	}

	// The error could be at different stages depending on where permission is denied
	if !strings.Contains(err.Error(), "failed to write attachment file") &&
		!strings.Contains(err.Error(), "failed to create attachment directory") &&
		!strings.Contains(err.Error(), "failed to check attachment existence") {
		t.Errorf("Expected permission-related error, got: %v", err)
	}
}

func TestEdgeCase_ExtremelySmallAttachment(t *testing.T) {
	extractor := NewAttachmentExtractor(t.TempDir())
	config := GetDefaultContentTypeConfig()

	// Single byte attachment
	smallData := base64.StdEncoding.EncodeToString([]byte{0xFF})

	part := &MMSPart{
		ContentType: "image/png",
		Data:        smallData,
	}

	result, err := extractor.ExtractAttachmentFromPart(part, config)
	if err != nil {
		t.Fatalf("Small attachment processing failed: %v", err)
	}

	// Should be skipped due to size filter
	if result.Action != ActionSkipped {
		t.Errorf("Expected tiny attachment to be skipped, got: %s", result.Action)
	}
	if result.Reason != "too-small" {
		t.Errorf("Expected 'too-small' reason, got: %s", result.Reason)
	}
}

func TestEdgeCase_MultipleIdenticalAttachmentsInSameMessage(t *testing.T) {
	extractor := NewAttachmentExtractor(t.TempDir())
	config := GetDefaultContentTypeConfig()

	sameImageData := createTestPNGData()

	mms := &MMS{
		Date:    1640995200000,
		Address: "555-1234",
		Parts: []MMSPart{
			{
				Seq:         0,
				ContentType: "image/png",
				Filename:    "photo1.png",
				Data:        sameImageData,
			},
			{
				Seq:         1,
				ContentType: "image/png",
				Filename:    "photo1_copy.png", // Same data, different name
				Data:        sameImageData,
			},
		},
	}

	summary, err := extractor.ExtractAttachmentsFromMMS(mms, config)
	if err != nil {
		t.Fatalf("MMS extraction failed: %v", err)
	}

	// First should be extracted, second should be referenced
	if summary.ExtractedCount != 1 {
		t.Errorf("Expected 1 extraction, got %d", summary.ExtractedCount)
	}
	if summary.ReferencedCount != 1 {
		t.Errorf("Expected 1 reference, got %d", summary.ReferencedCount)
	}

	// Both parts should point to same file but preserve original filenames
	if mms.Parts[0].Path != mms.Parts[1].Path {
		t.Errorf("Expected same attachment path for identical content")
	}
	if mms.Parts[0].Filename == mms.Parts[1].Filename {
		t.Errorf("Original filenames should be preserved")
	}
}

func TestEdgeCase_InvalidContentTypeHandling(t *testing.T) {
	extractor := NewAttachmentExtractor(t.TempDir())
	config := GetDefaultContentTypeConfig()

	tests := []struct {
		name        string
		contentType string
		expectSkip  bool
	}{
		{"Empty content type", "", true},
		{"Malformed MIME type", "image", true},
		{"Unknown MIME type", "application/x-unknown-binary", true},
		{"Case sensitivity", "IMAGE/JPEG", false}, // Should still extract
		{"With parameters", "image/jpeg; charset=binary", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			part := &MMSPart{
				ContentType: tt.contentType,
				Data:        createTestPNGData(),
			}

			result, err := extractor.ExtractAttachmentFromPart(part, config)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.expectSkip && result.Action != ActionSkipped {
				t.Errorf("Expected %s to be skipped, got: %s", tt.contentType, result.Action)
			}
			if !tt.expectSkip && result.Action == ActionSkipped && result.Reason == "content-type-filtered" {
				t.Errorf("Expected %s to not be skipped for content type, but it was", tt.contentType)
			}
		})
	}
}

func TestEdgeCase_ConcurrentExtraction(t *testing.T) {
	// Test that concurrent extraction doesn't cause race conditions
	tempDir := t.TempDir()
	extractor := NewAttachmentExtractor(tempDir)
	config := GetDefaultContentTypeConfig()

	// Use the same attachment data to test deduplication under concurrency
	sameData := createTestJPEGData()

	// Create multiple parts with the same data
	parts := make([]*MMSPart, 10)
	for i := 0; i < 10; i++ {
		parts[i] = &MMSPart{
			Seq:         i,
			ContentType: "image/jpeg",
			Filename:    filepath.Join("photo", fmt.Sprintf("%d.jpg", i)),
			Data:        sameData,
		}
	}

	// Extract concurrently
	results := make([]*AttachmentExtractionResult, 10)
	errors := make([]error, 10)
	done := make(chan int, 10)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			defer func() { done <- idx }()
			result, err := extractor.ExtractAttachmentFromPart(parts[idx], config)
			results[idx] = result
			errors[idx] = err
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Check results
	extractedCount := 0
	referencedCount := 0
	var commonPath string

	for i := 0; i < 10; i++ {
		if errors[i] != nil {
			t.Errorf("Extraction %d failed: %v", i, errors[i])
			continue
		}

		result := results[i]
		switch result.Action {
		case ActionExtracted:
			extractedCount++
		case ActionReferenced:
			referencedCount++
		}

		// All should have the same path (deduplication)
		if commonPath == "" {
			commonPath = result.Path
		} else if result.Path != commonPath {
			t.Errorf("Path mismatch: expected %s, got %s", commonPath, result.Path)
		}
	}

	// Should have exactly one extraction and the rest references
	// (Though with concurrency, multiple extractions might happen if they start simultaneously)
	if extractedCount == 0 {
		t.Error("Expected at least one extraction")
	}
	if extractedCount+referencedCount != 10 {
		t.Errorf("Expected 10 total results, got %d extracted + %d referenced",
			extractedCount, referencedCount)
	}
}

func TestEdgeCase_DirectoryTraversalAttempt(t *testing.T) {
	// Test that malicious content can't cause directory traversal
	// This is primarily handled by the AttachmentManager's hash-based paths
	extractor := NewAttachmentExtractor(t.TempDir())
	config := GetDefaultContentTypeConfig()

	// Try with malicious filename
	part := &MMSPart{
		ContentType: "image/png",
		Filename:    "../../etc/passwd", // Malicious path attempt
		Data:        createTestPNGData(),
	}

	result, err := extractor.ExtractAttachmentFromPart(part, config)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}

	// Verify the path is still hash-based and safe
	if !strings.HasPrefix(result.Path, "attachments/") {
		t.Errorf("Path should start with 'attachments/', got: %s", result.Path)
	}

	// Verify no directory traversal occurred
	if strings.Contains(result.Path, "..") {
		t.Errorf("Path contains directory traversal: %s", result.Path)
	}

	// The malicious filename should be preserved as metadata but not affect storage
	if part.Filename != "../../etc/passwd" {
		t.Errorf("Original filename should be preserved as metadata")
	}
}
