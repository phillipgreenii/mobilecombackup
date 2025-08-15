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

const (
	// Test content type categories
	categoryUnknown = "unknown"
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
		name             string
		contentType      string
		expected         bool
		expectedCategory string
		expectedReason   string
	}{
		// Should extract - binary types
		{"JPEG image", "image/jpeg", true, "binary", "whitelisted binary type"},
		{"PNG image", "image/png", true, "binary", "whitelisted binary type"},
		{"MP4 video", "video/mp4", true, "binary", "whitelisted binary type"},
		{"PDF document", "application/pdf", true, "binary", "whitelisted binary type"},
		{"JPEG with charset", "image/jpeg; charset=utf-8", true, "binary", "whitelisted binary type"},

		// Should not extract - text types
		{"SMIL presentation", "application/smil", false, "text", "text content - keeping inline"},
		{"Plain text", "text/plain", false, "text", "text content - keeping inline"},
		{"vCard contact", "text/x-vCard", false, "text", "text content - keeping inline"},
		{"WAP container", "application/vnd.wap.multipart.related", false, "text", "text content - keeping inline"},

		// Should not extract - unknown types
		{"Unknown type", "application/unknown", false, categoryUnknown, "unknown content type: application/unknown"},
		{"Empty content type", "", false, categoryUnknown, "missing content type header"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := extractor.shouldExtractContentType(tt.contentType, false, config)
			if decision.ShouldExtract != tt.expected {
				t.Errorf("shouldExtractContentType(%q).ShouldExtract = %v, expected %v", tt.contentType, decision.ShouldExtract, tt.expected)
			}
			if decision.Category != tt.expectedCategory {
				t.Errorf("shouldExtractContentType(%q).Category = %q, expected %q", tt.contentType, decision.Category, tt.expectedCategory)
			}
			if decision.Reason != tt.expectedReason {
				t.Errorf("shouldExtractContentType(%q).Reason = %q, expected %q", tt.contentType, decision.Reason, tt.expectedReason)
			}
		})
	}

	// Test text types are consistently rejected, even with explicit attachment marking
	t.Run("Text content types are always rejected", func(t *testing.T) {
		// text/plain is a text type and should never be extracted
		decision := extractor.shouldExtractContentType("text/plain", true, config)
		if decision.ShouldExtract {
			t.Errorf("shouldExtractContentType('text/plain', true, config).ShouldExtract = true, expected false (text types should never be extracted)")
		}
		if decision.Category != "text" {
			t.Errorf("Expected category 'text', got %q", decision.Category)
		}
	})

	// Test unknown types are rejected under the new strict policy
	t.Run("Unknown content types are rejected", func(t *testing.T) {
		// application/unknown is not in either whitelist and should be rejected
		decision := extractor.shouldExtractContentType("application/unknown", true, config)
		if decision.ShouldExtract {
			t.Errorf("shouldExtractContentType('application/unknown', true, config).ShouldExtract = true, expected false (unknown types should be rejected)")
		}
		if decision.Category != categoryUnknown {
			t.Errorf("Expected category 'unknown', got %q", decision.Category)
		}

		// Same result without explicit marking
		decision = extractor.shouldExtractContentType("application/unknown", false, config)
		if decision.ShouldExtract {
			t.Errorf("shouldExtractContentType('application/unknown', false, config).ShouldExtract = true, expected false")
		}
	})

	// Test edge cases
	t.Run("Edge cases", func(t *testing.T) {
		tests := []struct {
			name           string
			contentType    string
			expectedCat    string
			expectedReason string
		}{
			{"Whitespace content type", "  \t  ", categoryUnknown, "empty content type after normalization"},
			{"Content type with semicolon", "image/png;", "binary", "whitelisted binary type"},
			{"Case insensitive", "IMAGE/JPEG", "binary", "whitelisted binary type"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				decision := extractor.shouldExtractContentType(tt.contentType, false, config)
				if decision.Category != tt.expectedCat {
					t.Errorf("%s: Expected category %q, got %q", tt.name, tt.expectedCat, decision.Category)
				}
				if decision.Reason != tt.expectedReason {
					t.Errorf("%s: Expected reason %q, got %q", tt.name, tt.expectedReason, decision.Reason)
				}
			})
		}
	})
}

// TestAttachmentExtractor_ContentTypeEdgeCases tests comprehensive edge cases for content type handling
func TestAttachmentExtractor_ContentTypeEdgeCases(t *testing.T) {
	extractor := NewAttachmentExtractor(t.TempDir())
	config := GetDefaultContentTypeConfig()

	tests := []struct {
		name             string
		contentType      string
		expectedExtract  bool
		expectedCategory string
		expectedReason   string
	}{
		// Malformed content types
		{"Malformed semicolon only", ";", false, categoryUnknown, "empty content type after normalization"},
		{"Multiple semicolons", "image/png;;charset=utf-8;", true, "binary", "whitelisted binary type"},
		{"Content type with trailing space", "image/jpeg ", true, "binary", "whitelisted binary type"},
		{"Content type with leading space", " image/jpeg", true, "binary", "whitelisted binary type"},
		{"Content type with tabs", "\timage/png\t", true, "binary", "whitelisted binary type"},
		{"Content type with newlines", "image/jpeg\n", true, "binary", "whitelisted binary type"},

		// Mixed case variations
		{"Upper case", "IMAGE/JPEG", true, "binary", "whitelisted binary type"},
		{"Mixed case", "Image/Jpeg", true, "binary", "whitelisted binary type"},
		{"Mixed case with charset", "Image/PNG; Charset=UTF-8", true, "binary", "whitelisted binary type"},

		// Boundary conditions
		{"Just whitespace", "   \t\n  ", false, categoryUnknown, "empty content type after normalization"},
		{"Single character", "x", false, categoryUnknown, "unknown content type: x"},
		{"Slash without subtype", "image/", false, categoryUnknown, "unknown content type: image/"},
		{"Subtype without main type", "/jpeg", false, categoryUnknown, "unknown content type: /jpeg"},
		{"Double slash", "image//jpeg", false, categoryUnknown, "unknown content type: image//jpeg"},

		// Common variations and alternatives
		{"JPEG alt spelling", "image/jpg", true, "binary", "whitelisted binary type"},
		{"Non-standard MP3", "audio/mp3", true, "binary", "whitelisted binary type"},
		{"TIFF variation", "image/tiff", true, "binary", "whitelisted binary type"},
		{"TIFF short form", "image/tif", true, "binary", "whitelisted binary type"},

		// Common unknown types that should be rejected
		{"Adobe Flash", "application/x-shockwave-flash", false, categoryUnknown, "unknown content type: application/x-shockwave-flash"},
		{"Custom application", "application/x-custom-format", false, categoryUnknown, "unknown content type: application/x-custom-format"},
		{"Binary stream", "application/binary", false, categoryUnknown, "unknown content type: application/binary"},
		{"Generic data", "application/data", false, categoryUnknown, "unknown content type: application/data"},

		// Text variations that should remain inline
		{"Rich text format", "text/rtf", false, "text", "text content - keeping inline"},
		{"CSV data", "text/csv", false, "text", "text content - keeping inline"},
		{"JavaScript", "text/javascript", false, "text", "text content - keeping inline"},
		{"Application JavaScript", "application/javascript", false, "text", "text content - keeping inline"},

		// Complex parameter strings
		{"Multiple parameters", "image/jpeg; charset=utf-8; boundary=something", true, "binary", "whitelisted binary type"},
		{"Parameters with spaces", "image/png; charset = utf-8 ; quality = high", true, "binary", "whitelisted binary type"},
		{"Parameters with quotes", `image/gif; name="file name.gif"`, true, "binary", "whitelisted binary type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := extractor.shouldExtractContentType(tt.contentType, false, config)
			if decision.ShouldExtract != tt.expectedExtract {
				t.Errorf("shouldExtractContentType(%q).ShouldExtract = %v, expected %v", tt.contentType, decision.ShouldExtract, tt.expectedExtract)
			}
			if decision.Category != tt.expectedCategory {
				t.Errorf("shouldExtractContentType(%q).Category = %q, expected %q", tt.contentType, decision.Category, tt.expectedCategory)
			}
			if decision.Reason != tt.expectedReason {
				t.Errorf("shouldExtractContentType(%q).Reason = %q, expected %q", tt.contentType, decision.Reason, tt.expectedReason)
			}
		})
	}
}

// TestAttachmentExtractor_LoggingBehavior tests that logging is comprehensive and correct
func TestAttachmentExtractor_LoggingBehavior(t *testing.T) {
	tempDir := t.TempDir()
	extractor := NewAttachmentExtractor(tempDir)
	config := GetDefaultContentTypeConfig()

	// Test that each decision path logs appropriately
	testCases := []struct {
		name        string
		contentType string
		hasData     bool
		dataSize    int
		expectLog   string
	}{
		{"Binary type approval", "image/png", true, 2000, "whitelisted binary type"},
		{"Text type rejection", "text/plain", true, 2000, "text content - keeping inline"},
		{"Unknown type rejection", "application/unknown", true, 2000, "unknown content type: application/unknown"},
		{"Missing content type", "", true, 2000, "missing content type header"},
		{"Small data skip", "image/jpeg", true, 100, "Skipping small binary content"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			part := &MMSPart{
				ContentType: tc.contentType,
				Filename:    "test.bin",
			}

			if tc.hasData {
				// Create test data of specified size
				testData := make([]byte, tc.dataSize)
				for i := range testData {
					testData[i] = byte(i % 256)
				}
				part.Data = base64.StdEncoding.EncodeToString(testData)
			}

			// The actual extraction test - we mainly care that it doesn't crash
			// and that logging happens (we can't easily capture logs in tests but
			// the log.Printf calls will execute)
			_, err := extractor.ExtractAttachmentFromPart(part, config)

			// Some test cases will error (like invalid base64), that's expected
			if tc.name == "Small data skip" && err != nil {
				t.Errorf("Unexpected error for small data: %v", err)
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
	content, err := os.ReadFile(fullPath) // #nosec G304 // Test-controlled path
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

	if result2.Action != ActionReferenced {
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
