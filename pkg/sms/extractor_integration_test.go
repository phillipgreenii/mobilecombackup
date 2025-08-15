package sms

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/attachments"
)

func TestAttachmentExtraction_EndToEnd_ImportFlow(t *testing.T) {
	tempDir := t.TempDir()

	// Create test MMS with attachments
	mms := &MMS{
		Date:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC).Unix() * 1000,
		MsgBox:  1,
		Address: "555-1234",
		Parts: []MMSPart{
			{
				Seq:         0,
				ContentType: "text/plain",
				Text:        "Check out this photo!",
			},
			{
				Seq:         1,
				ContentType: "image/jpeg",
				Filename:    "vacation.jpg",
				Data:        createTestJPEGData(),
			},
			{
				Seq:         2,
				ContentType: "application/smil",
				Data:        base64.StdEncoding.EncodeToString([]byte(strings.Repeat("<smil>layout</smil>", 30))),
			},
		},
	}

	// Initialize extractor
	extractor := NewAttachmentExtractor(tempDir)
	config := GetDefaultContentTypeConfig()

	// Extract attachments (simulating import process)
	summary, err := extractor.ExtractAttachmentsFromMMS(mms, config)
	if err != nil {
		t.Fatalf("Attachment extraction failed: %v", err)
	}

	// Verify extraction results
	if summary.ExtractedCount != 1 {
		t.Errorf("Expected 1 attachment extracted, got %d", summary.ExtractedCount)
	}
	if summary.SkippedCount != 2 {
		t.Errorf("Expected 2 parts skipped, got %d", summary.SkippedCount)
	}

	// Verify MMS was modified correctly
	extractedPart := mms.Parts[1]
	if extractedPart.Data != "" {
		t.Errorf("Expected attachment data to be cleared after extraction")
	}
	if extractedPart.Path == "" {
		t.Fatalf("Expected attachment path to be set")
	}
	if extractedPart.OriginalSize == 0 {
		t.Errorf("Expected original size to be set")
	}
	if extractedPart.ExtractionDate == "" {
		t.Errorf("Expected extraction date to be set")
	}

	// Verify file was created
	fullPath := filepath.Join(tempDir, extractedPart.Path)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Errorf("Attachment file not created at %s", fullPath)
	}

	// Verify attachment manager can read the file
	attachmentManager := attachments.NewAttachmentManager(tempDir)

	// Extract hash from path
	pathParts := strings.Split(extractedPart.Path, "/")
	hash := pathParts[len(pathParts)-1]

	exists, err := attachmentManager.AttachmentExists(hash)
	if err != nil {
		t.Fatalf("Failed to check attachment existence: %v", err)
	}
	if !exists {
		t.Errorf("Attachment not found by AttachmentManager")
	}

	// Verify hash integrity
	verified, err := attachmentManager.VerifyAttachment(hash)
	if err != nil {
		t.Fatalf("Failed to verify attachment: %v", err)
	}
	if !verified {
		t.Errorf("Attachment hash verification failed")
	}
}

func TestAttachmentExtraction_XMLSerialization(t *testing.T) {
	// Test that extracted MMS can be properly serialized to XML
	mms := &MMS{
		Date:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC).Unix() * 1000,
		MsgBox:  1,
		Address: "555-1234",
		Parts: []MMSPart{
			{
				Seq:         0,
				ContentType: "image/png",
				Filename:    "screenshot.png",
				Data:        createTestPNGData(),
			},
		},
	}

	// Extract attachments
	extractor := NewAttachmentExtractor(t.TempDir())
	config := GetDefaultContentTypeConfig()

	_, err := extractor.ExtractAttachmentsFromMMS(mms, config)
	if err != nil {
		t.Fatalf("Attachment extraction failed: %v", err)
	}

	// Serialize to XML
	xmlData, err := xml.MarshalIndent(mms, "", "  ")
	if err != nil {
		t.Fatalf("XML marshaling failed: %v", err)
	}

	xmlString := string(xmlData)

	// Verify XML contains attachment metadata but not original data
	if strings.Contains(xmlString, "data=") && strings.Contains(xmlString, createTestPNGData()[:50]) {
		t.Errorf("XML should not contain original base64 data after extraction")
	}
	if !strings.Contains(xmlString, "path=") {
		t.Errorf("XML should contain path attribute")
	}
	if !strings.Contains(xmlString, "original_size=") {
		t.Errorf("XML should contain original_size attribute")
	}
	if !strings.Contains(xmlString, "extraction_date=") {
		t.Errorf("XML should contain extraction_date attribute")
	}
}

func TestAttachmentExtraction_MultipleMessagesDuplicateAttachments(t *testing.T) {
	tempDir := t.TempDir()
	extractor := NewAttachmentExtractor(tempDir)
	config := GetDefaultContentTypeConfig()

	// Same attachment data
	sameImageData := createTestJPEGData()

	// First MMS with attachment
	mms1 := &MMS{
		Date:    time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC).Unix() * 1000,
		Address: "555-1111",
		Parts: []MMSPart{
			{
				Seq:         0,
				ContentType: "image/jpeg",
				Filename:    "photo1.jpg",
				Data:        sameImageData,
			},
		},
	}

	// Second MMS with same attachment (different filename)
	mms2 := &MMS{
		Date:    time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC).Unix() * 1000,
		Address: "555-2222",
		Parts: []MMSPart{
			{
				Seq:         0,
				ContentType: "image/jpeg",
				Filename:    "photo2.jpg", // Different filename, same content
				Data:        sameImageData,
			},
		},
	}

	// Extract from first message
	summary1, err := extractor.ExtractAttachmentsFromMMS(mms1, config)
	if err != nil {
		t.Fatalf("First extraction failed: %v", err)
	}

	if summary1.ExtractedCount != 1 {
		t.Errorf("Expected 1 extraction from first message, got %d", summary1.ExtractedCount)
	}
	if summary1.ReferencedCount != 0 {
		t.Errorf("Expected 0 references from first message, got %d", summary1.ReferencedCount)
	}

	// Extract from second message
	summary2, err := extractor.ExtractAttachmentsFromMMS(mms2, config)
	if err != nil {
		t.Fatalf("Second extraction failed: %v", err)
	}

	if summary2.ExtractedCount != 0 {
		t.Errorf("Expected 0 extractions from second message, got %d", summary2.ExtractedCount)
	}
	if summary2.ReferencedCount != 1 {
		t.Errorf("Expected 1 reference from second message, got %d", summary2.ReferencedCount)
	}

	// Both messages should have the same attachment path
	if mms1.Parts[0].Path != mms2.Parts[0].Path {
		t.Errorf("Expected same attachment path, got %s vs %s",
			mms1.Parts[0].Path, mms2.Parts[0].Path)
	}

	// But different filenames should be preserved as metadata
	if mms1.Parts[0].Filename == mms2.Parts[0].Filename {
		t.Errorf("Original filenames should be preserved even with duplicate attachments")
	}
}

func TestAttachmentExtraction_RepoStructureAfterExtraction(t *testing.T) {
	tempDir := t.TempDir()
	extractor := NewAttachmentExtractor(tempDir)
	config := GetDefaultContentTypeConfig()

	// Create MMS with multiple different attachments
	mms := &MMS{
		Date:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).Unix() * 1000,
		Address: "555-3333",
		Parts: []MMSPart{
			{
				Seq:         0,
				ContentType: "image/png",
				Filename:    "screenshot.png",
				Data:        createTestPNGData(),
			},
			{
				Seq:         1,
				ContentType: "image/jpeg",
				Filename:    "photo.jpg",
				Data:        createTestJPEGData(),
			},
		},
	}

	// Extract attachments
	_, err := extractor.ExtractAttachmentsFromMMS(mms, config)
	if err != nil {
		t.Fatalf("Attachment extraction failed: %v", err)
	}

	// Verify repository structure
	attachmentsDir := filepath.Join(tempDir, "attachments")
	if _, err := os.Stat(attachmentsDir); os.IsNotExist(err) {
		t.Fatalf("Attachments directory not created")
	}

	// Check that attachments are in proper hash-based subdirectories
	entries, err := os.ReadDir(attachmentsDir)
	if err != nil {
		t.Fatalf("Failed to read attachments directory: %v", err)
	}

	foundAttachments := 0
	for _, entry := range entries {
		if entry.IsDir() && len(entry.Name()) == 2 {
			// This is a hash prefix directory
			subDir := filepath.Join(attachmentsDir, entry.Name())
			subEntries, err := os.ReadDir(subDir)
			if err != nil {
				t.Errorf("Failed to read subdirectory %s: %v", entry.Name(), err)
				continue
			}

			for _, subEntry := range subEntries {
				if !subEntry.IsDir() && len(subEntry.Name()) == 64 {
					// This looks like an attachment file
					if !strings.HasPrefix(subEntry.Name(), entry.Name()) {
						t.Errorf("Attachment %s in wrong subdirectory %s",
							subEntry.Name(), entry.Name())
					}
					foundAttachments++
				}
			}
		}
	}

	if foundAttachments != 2 {
		t.Errorf("Expected 2 attachment files, found %d", foundAttachments)
	}

	// Verify attachment manager can validate the structure
	attachmentManager := attachments.NewAttachmentManager(tempDir)
	if err := attachmentManager.ValidateAttachmentStructure(); err != nil {
		t.Errorf("Attachment structure validation failed: %v", err)
	}
}

func TestAttachmentExtraction_LargeMessageBatch(t *testing.T) {
	tempDir := t.TempDir()
	extractor := NewAttachmentExtractor(tempDir)
	config := GetDefaultContentTypeConfig()
	stats := NewAttachmentExtractionStats()

	// Process multiple messages to test performance and memory usage
	for i := 0; i < 50; i++ {
		// Create unique attachment data for some variety
		// Decode, modify, and re-encode to ensure valid base64
		originalData, _ := base64.StdEncoding.DecodeString(createTestPNGData())
		originalData = append(originalData, byte(i))
		imageData := base64.StdEncoding.EncodeToString(originalData)

		mms := &MMS{
			Date:    time.Date(2024, 1, 15, 10, i, 0, 0, time.UTC).Unix() * 1000,
			Address: "555-4444",
			Parts: []MMSPart{
				{
					Seq:         0,
					ContentType: "text/plain",
					Text:        "Message text",
				},
				{
					Seq:         1,
					ContentType: "image/png",
					Filename:    fmt.Sprintf("image%03d.png", i),
					Data:        imageData,
				},
			},
		}

		summary, err := extractor.ExtractAttachmentsFromMMS(mms, config)
		if err != nil {
			t.Fatalf("Extraction failed for message %d: %v", i, err)
		}

		stats.AddMMSExtractionSummary(summary)
	}

	// Verify overall statistics
	if stats.TotalMessages != 50 {
		t.Errorf("Expected 50 messages processed, got %d", stats.TotalMessages)
	}
	if stats.MessagesWithParts != 50 {
		t.Errorf("Expected 50 messages with parts, got %d", stats.MessagesWithParts)
	}
	if stats.TotalParts != 100 {
		t.Errorf("Expected 100 total parts, got %d", stats.TotalParts)
	}
	if stats.ExtractedCount != 50 {
		t.Errorf("Expected 50 attachments extracted, got %d", stats.ExtractedCount)
	}
	if stats.SkippedCount != 50 {
		t.Errorf("Expected 50 parts skipped (text), got %d", stats.SkippedCount)
	}
}
