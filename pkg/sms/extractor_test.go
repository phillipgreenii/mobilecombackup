package sms

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Helper to create test PNG data (make it large enough to pass size filter)
func createTestPNGData() string {
	// Create a larger PNG-like file (> 1KB when base64 encoded)
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, // IHDR length
		0x49, 0x48, 0x44, 0x52, // IHDR
		0x00, 0x00, 0x00, 0x01, // Width: 1
		0x00, 0x00, 0x00, 0x01, // Height: 1
		0x08, 0x02, 0x00, 0x00, 0x00, // Bit depth, color type, compression, filter, interlace
		0x90, 0x77, 0x53, 0xDE, // CRC
	}
	// Add padding to make it larger than 1KB when base64 encoded
	padding := make([]byte, 1000)
	for i := range padding {
		padding[i] = byte(i % 256)
	}
	pngData = append(pngData, padding...)
	return base64.StdEncoding.EncodeToString(pngData)
}

// Helper to create test JPEG data (make it large enough to pass size filter)
func createTestJPEGData() string {
	// Create a larger JPEG-like file (> 1KB when base64 encoded)
	jpegData := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, // SOI + APP0
		0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, // APP0 header
		0x01, 0x01, 0x00, 0x48, 0x00, 0x48, 0x00, 0x00, // JFIF data
		0xFF, 0xD9, // EOI
	}
	// Add padding to make it larger than 1KB when base64 encoded
	padding := make([]byte, 1000)
	for i := range padding {
		padding[i] = byte((i + 100) % 256)
	}
	jpegData = append(jpegData, padding...)
	return base64.StdEncoding.EncodeToString(jpegData)
}

// Helper to calculate expected hash
func calculateExpectedHash(base64Data string) string {
	decoded, _ := base64.StdEncoding.DecodeString(base64Data)
	hasher := sha256.New()
	hasher.Write(decoded)
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func TestAttachmentExtractor_ShouldExtractContentType(t *testing.T) {
	extractor := NewAttachmentExtractor(t.TempDir())
	config := GetDefaultContentTypeConfig()

	tests := []struct {
		name        string
		contentType string
		expected    bool
	}{
		// Should extract
		{"JPEG image", "image/jpeg", true},
		{"PNG image", "image/png", true},
		{"MP4 video", "video/mp4", true},
		{"PDF document", "application/pdf", true},
		{"JPEG with charset", "image/jpeg; charset=utf-8", true},
		
		// Should not extract
		{"SMIL presentation", "application/smil", false},
		{"Plain text", "text/plain", false},
		{"vCard contact", "text/x-vCard", false},
		{"WAP container", "application/vnd.wap.multipart.related", false},
		{"Unknown type", "application/unknown", false},
		{"Empty content type", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.shouldExtractContentType(tt.contentType, config)
			if result != tt.expected {
				t.Errorf("shouldExtractContentType(%q) = %v, expected %v", tt.contentType, result, tt.expected)
			}
		})
	}
}

func TestAttachmentExtractor_ExtractAttachmentFromPart_Image(t *testing.T) {
	tempDir := t.TempDir()
	extractor := NewAttachmentExtractor(tempDir)
	config := GetDefaultContentTypeConfig()

	// Create test PNG data
	pngData := createTestPNGData()
	expectedHash := calculateExpectedHash(pngData)

	part := &MMSPart{
		Seq:         0,
		ContentType: "image/png",
		Filename:    "test.png",
		Data:        pngData,
	}

	result, err := extractor.ExtractAttachmentFromPart(part, config)
	if err != nil {
		t.Fatalf("ExtractAttachmentFromPart failed: %v", err)
	}

	// Verify extraction result
	if result.Action != "extracted" {
		t.Errorf("Expected action 'extracted', got %q", result.Action)
	}
	if result.Hash != expectedHash {
		t.Errorf("Expected hash %s, got %s", expectedHash, result.Hash)
	}

	expectedPath := filepath.Join("attachments", expectedHash[:2], expectedHash)
	if result.Path != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, result.Path)
	}

	// Verify file was created
	fullPath := filepath.Join(tempDir, result.Path)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Errorf("Attachment file was not created at %s", fullPath)
	}

	// Verify file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read attachment file: %v", err)
	}

	originalData, _ := base64.StdEncoding.DecodeString(pngData)
	if len(content) != len(originalData) {
		t.Errorf("File content length mismatch: expected %d, got %d", len(originalData), len(content))
	}
}

func TestAttachmentExtractor_ExtractAttachmentFromPart_Duplicate(t *testing.T) {
	tempDir := t.TempDir()
	extractor := NewAttachmentExtractor(tempDir)
	config := GetDefaultContentTypeConfig()

	// Create test data
	jpegData := createTestJPEGData()
	expectedHash := calculateExpectedHash(jpegData)

	// First extraction
	part1 := &MMSPart{
		Seq:         0,
		ContentType: "image/jpeg",
		Filename:    "photo1.jpg",
		Data:        jpegData,
	}

	result1, err := extractor.ExtractAttachmentFromPart(part1, config)
	if err != nil {
		t.Fatalf("First extraction failed: %v", err)
	}

	if result1.Action != "extracted" {
		t.Errorf("Expected first action 'extracted', got %q", result1.Action)
	}

	// Second extraction with same data
	part2 := &MMSPart{
		Seq:         1,
		ContentType: "image/jpeg",
		Filename:    "photo2.jpg", // Different filename, same data
		Data:        jpegData,
	}

	result2, err := extractor.ExtractAttachmentFromPart(part2, config)
	if err != nil {
		t.Fatalf("Second extraction failed: %v", err)
	}

	if result2.Action != "referenced" {
		t.Errorf("Expected second action 'referenced', got %q", result2.Action)
	}
	
	if result2.Hash != expectedHash {
		t.Errorf("Hash mismatch: expected %s, got %s", expectedHash, result2.Hash)
	}
}

func TestAttachmentExtractor_ExtractAttachmentFromPart_SkipConditions(t *testing.T) {
	tempDir := t.TempDir()
	extractor := NewAttachmentExtractor(tempDir)
	config := GetDefaultContentTypeConfig()

	tests := []struct {
		name           string
		part           *MMSPart
		expectedAction string
		expectedReason string
	}{
		{
			name: "No data",
			part: &MMSPart{
				ContentType: "image/png",
				Data:        "",
			},
			expectedAction: "skipped",
			expectedReason: "no-data",
		},
		{
			name: "Null data",
			part: &MMSPart{
				ContentType: "image/png",
				Data:        "null",
			},
			expectedAction: "skipped",
			expectedReason: "no-data",
		},
		{
			name: "SMIL content type",
			part: &MMSPart{
				ContentType: "application/smil",
				Data:        createTestPNGData(),
			},
			expectedAction: "skipped",
			expectedReason: "content-type-filtered",
		},
		{
			name: "Text content type",
			part: &MMSPart{
				ContentType: "text/plain",
				Data:        base64.StdEncoding.EncodeToString([]byte("Hello world")),
			},
			expectedAction: "skipped",
			expectedReason: "content-type-filtered",
		},
		{
			name: "Small data",
			part: &MMSPart{
				ContentType: "image/png",
				Data:        base64.StdEncoding.EncodeToString([]byte("small")),
			},
			expectedAction: "skipped",
			expectedReason: "too-small",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractor.ExtractAttachmentFromPart(tt.part, config)
			if err != nil {
				t.Fatalf("ExtractAttachmentFromPart failed: %v", err)
			}

			if result.Action != tt.expectedAction {
				t.Errorf("Expected action %q, got %q", tt.expectedAction, result.Action)
			}
			if result.Reason != tt.expectedReason {
				t.Errorf("Expected reason %q, got %q", tt.expectedReason, result.Reason)
			}
			if result.UpdatePart {
				t.Errorf("Expected UpdatePart to be false for skipped extraction")
			}
		})
	}
}

func TestAttachmentExtractor_ExtractAttachmentFromPart_InvalidBase64(t *testing.T) {
	tempDir := t.TempDir()
	extractor := NewAttachmentExtractor(tempDir)
	config := GetDefaultContentTypeConfig()

	// Create invalid base64 data that's large enough to pass the size filter
	invalidData := strings.Repeat("invalid-base64-data!!!", 100) // Make it large enough
	part := &MMSPart{
		ContentType: "image/png",
		Data:        invalidData,
	}

	_, err := extractor.ExtractAttachmentFromPart(part, config)
	if err == nil {
		t.Fatal("Expected error for invalid base64 data")
	}

	if !strings.Contains(err.Error(), "failed to decode base64 data") {
		t.Errorf("Expected base64 decode error, got: %v", err)
	}
}

func TestAttachmentExtractor_ExtractAttachmentsFromMMS(t *testing.T) {
	tempDir := t.TempDir()
	extractor := NewAttachmentExtractor(tempDir)
	config := GetDefaultContentTypeConfig()

	// Create MMS with multiple parts
	mms := &MMS{
		Date:    time.Now().Unix() * 1000,
		Address: "555-1234",
		Parts: []MMSPart{
			{
				Seq:         0,
				ContentType: "text/plain",
				Text:        "Hello",
				Data:        "",
			},
			{
				Seq:         1,
				ContentType: "image/png",
				Filename:    "photo.png",
				Data:        createTestPNGData(),
			},
			{
				Seq:         2,
				ContentType: "application/smil",
				Data:        base64.StdEncoding.EncodeToString([]byte("<smil></smil>")),
			},
			{
				Seq:         3,
				ContentType: "image/jpeg",
				Filename:    "photo2.jpg",
				Data:        createTestJPEGData(),
			},
		},
	}

	summary, err := extractor.ExtractAttachmentsFromMMS(mms, config)
	if err != nil {
		t.Fatalf("ExtractAttachmentsFromMMS failed: %v", err)
	}

	// Verify summary
	if summary.TotalParts != 4 {
		t.Errorf("Expected 4 total parts, got %d", summary.TotalParts)
	}
	if summary.ExtractedCount != 2 {
		t.Errorf("Expected 2 extracted attachments, got %d", summary.ExtractedCount)
	}
	if summary.SkippedCount != 2 {
		t.Errorf("Expected 2 skipped parts, got %d", summary.SkippedCount)
	}

	// Verify parts were updated correctly
	extractedParts := 0
	for _, part := range mms.Parts {
		if part.Path != "" {
			extractedParts++
			if part.Data != "" {
				t.Errorf("Expected Data to be cleared when Path is set")
			}
			if part.OriginalSize == 0 {
				t.Errorf("Expected OriginalSize to be set")
			}
			if part.ExtractionDate == "" {
				t.Errorf("Expected ExtractionDate to be set")
			}
		}
	}

	if extractedParts != 2 {
		t.Errorf("Expected 2 parts to have Path set, got %d", extractedParts)
	}
}

func TestUpdatePartWithExtraction(t *testing.T) {
	part := &MMSPart{
		Seq:         0,
		ContentType: "image/png",
		Data:        createTestPNGData(),
	}

	result := &AttachmentExtractionResult{
		Action:         "extracted",
		Hash:           "abc123",
		Path:           "attachments/ab/abc123",
		OriginalSize:   1024,
		UpdatePart:     true,
		ExtractionDate: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	UpdatePartWithExtraction(part, result)

	// Verify updates
	if part.Data != "" {
		t.Errorf("Expected Data to be cleared, got %q", part.Data)
	}
	if part.Path != result.Path {
		t.Errorf("Expected Path %q, got %q", result.Path, part.Path)
	}
	if part.OriginalSize != result.OriginalSize {
		t.Errorf("Expected OriginalSize %d, got %d", result.OriginalSize, part.OriginalSize)
	}
	if part.AttachmentRef != result.Path {
		t.Errorf("Expected AttachmentRef %q, got %q", result.Path, part.AttachmentRef)
	}

	expectedDate := "2024-01-15T10:30:00Z"
	if part.ExtractionDate != expectedDate {
		t.Errorf("Expected ExtractionDate %q, got %q", expectedDate, part.ExtractionDate)
	}
}

func TestUpdatePartWithExtraction_NoUpdate(t *testing.T) {
	originalData := createTestPNGData()
	part := &MMSPart{
		Seq:         0,
		ContentType: "image/png",
		Data:        originalData,
	}

	result := &AttachmentExtractionResult{
		Action:     "skipped",
		Reason:     "content-type-filtered",
		UpdatePart: false,
	}

	UpdatePartWithExtraction(part, result)

	// Verify no changes
	if part.Data != originalData {
		t.Errorf("Expected Data to remain unchanged")
	}
	if part.Path != "" {
		t.Errorf("Expected Path to remain empty")
	}
	if part.OriginalSize != 0 {
		t.Errorf("Expected OriginalSize to remain 0")
	}
}

func TestAttachmentExtractionStats_AddMMSExtractionSummary(t *testing.T) {
	stats := NewAttachmentExtractionStats()

	summary := &MMSExtractionSummary{
		MessageDate:         time.Now().Unix() * 1000,
		TotalParts:          4,
		ExtractedCount:      2,
		ReferencedCount:     1,
		SkippedCount:        1,
		TotalExtractedSize:  2048,
		TotalReferencedSize: 1024,
	}

	stats.AddMMSExtractionSummary(summary)

	if stats.TotalMessages != 1 {
		t.Errorf("Expected TotalMessages 1, got %d", stats.TotalMessages)
	}
	if stats.MessagesWithParts != 1 {
		t.Errorf("Expected MessagesWithParts 1, got %d", stats.MessagesWithParts)
	}
	if stats.ExtractedCount != 2 {
		t.Errorf("Expected ExtractedCount 2, got %d", stats.ExtractedCount)
	}
	if stats.ReferencedCount != 1 {
		t.Errorf("Expected ReferencedCount 1, got %d", stats.ReferencedCount)
	}
	if stats.TotalExtractedSize != 2048 {
		t.Errorf("Expected TotalExtractedSize 2048, got %d", stats.TotalExtractedSize)
	}

	// Add another summary
	summary2 := &MMSExtractionSummary{
		MessageDate:         time.Now().Unix() * 1000,
		TotalParts:          2,
		ExtractedCount:      1,
		ReferencedCount:     0,
		SkippedCount:        1,
		TotalExtractedSize:  512,
		TotalReferencedSize: 0,
	}

	stats.AddMMSExtractionSummary(summary2)

	if stats.TotalMessages != 2 {
		t.Errorf("Expected TotalMessages 2, got %d", stats.TotalMessages)
	}
	if stats.ExtractedCount != 3 {
		t.Errorf("Expected ExtractedCount 3, got %d", stats.ExtractedCount)
	}
	if stats.TotalExtractedSize != 2560 {
		t.Errorf("Expected TotalExtractedSize 2560, got %d", stats.TotalExtractedSize)
	}
}

func TestGetDefaultContentTypeConfig(t *testing.T) {
	config := GetDefaultContentTypeConfig()

	// Check that expected extractable types are present
	expectedExtractable := []string{
		"image/jpeg", "image/png", "video/mp4", "audio/mpeg", "application/pdf",
	}
	
	for _, expected := range expectedExtractable {
		found := false
		for _, extractable := range config.ExtractableTypes {
			if extractable == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected extractable type %q not found", expected)
		}
	}

	// Check that expected skipped types are present
	expectedSkipped := []string{
		"application/smil", "text/plain", "text/x-vCard",
	}
	
	for _, expected := range expectedSkipped {
		found := false
		for _, skipped := range config.SkippedTypes {
			if skipped == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected skipped type %q not found", expected)
		}
	}
}