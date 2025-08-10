package sms

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestContentTypeFiltering_SMILNotExtracted(t *testing.T) {
	extractor := NewAttachmentExtractor(t.TempDir())
	config := GetDefaultContentTypeConfig()

	// Create SMIL content that would normally be large enough to extract
	smilContent := `<smil>
		<head>
			<layout>
				<root-layout width="320" height="240"/>
				<region id="Image" left="0" top="0" width="320" height="180"/>
				<region id="Text" left="0" top="180" width="320" height="60"/>
			</layout>
		</head>
		<body>
			<par dur="5000ms">
				<img src="image001.jpg" region="Image"/>
				<text src="text001.txt" region="Text"/>
			</par>
		</body>
	</smil>` + strings.Repeat(" padding", 200) // Make it large enough

	part := &MMSPart{
		Seq:         0,
		ContentType: "application/smil",
		Data:        base64.StdEncoding.EncodeToString([]byte(smilContent)),
	}

	result, err := extractor.ExtractAttachmentFromPart(part, config)
	if err != nil {
		t.Fatalf("ExtractAttachmentFromPart failed: %v", err)
	}

	if result.Action != "skipped" {
		t.Errorf("Expected SMIL to be skipped, got action: %s", result.Action)
	}
	if result.Reason != "content-type-filtered" {
		t.Errorf("Expected reason 'content-type-filtered', got: %s", result.Reason)
	}
	if result.UpdatePart {
		t.Errorf("Expected UpdatePart to be false for skipped SMIL")
	}

	// Verify original part unchanged
	if part.Data == "" {
		t.Errorf("Expected original data to be preserved")
	}
	if part.Path != "" {
		t.Errorf("Expected Path to remain empty")
	}
}

func TestContentTypeFiltering_TextPlainNotExtracted(t *testing.T) {
	extractor := NewAttachmentExtractor(t.TempDir())
	config := GetDefaultContentTypeConfig()

	// Create large text content
	textContent := "This is a text message. " + strings.Repeat("More text content. ", 100)

	part := &MMSPart{
		Seq:         0,
		ContentType: "text/plain",
		Text:        textContent,
		Data:        base64.StdEncoding.EncodeToString([]byte(textContent)),
	}

	result, err := extractor.ExtractAttachmentFromPart(part, config)
	if err != nil {
		t.Fatalf("ExtractAttachmentFromPart failed: %v", err)
	}

	if result.Action != "skipped" {
		t.Errorf("Expected text/plain to be skipped, got action: %s", result.Action)
	}
	if result.Reason != "content-type-filtered" {
		t.Errorf("Expected reason 'content-type-filtered', got: %s", result.Reason)
	}

	// Verify text part unchanged
	if part.Text != textContent {
		t.Errorf("Expected text content to be preserved")
	}
}

func TestContentTypeFiltering_VCardNotExtracted(t *testing.T) {
	extractor := NewAttachmentExtractor(t.TempDir())
	config := GetDefaultContentTypeConfig()

	// Create vCard content
	vCardContent := `BEGIN:VCARD
VERSION:3.0
FN:John Doe
N:Doe;John;;;
ORG:Example Company
TITLE:Software Engineer
TEL;TYPE=WORK,VOICE:555-1234
TEL;TYPE=HOME,VOICE:555-5678
EMAIL:john.doe@example.com
URL:http://www.johndoe.com
ADR;TYPE=WORK:;;123 Main St;Anytown;ST;12345;USA
NOTE:This is a note about John Doe
END:VCARD` + strings.Repeat("\nEXTRA:padding", 50)

	part := &MMSPart{
		Seq:         0,
		ContentType: "text/x-vCard",
		Data:        base64.StdEncoding.EncodeToString([]byte(vCardContent)),
	}

	result, err := extractor.ExtractAttachmentFromPart(part, config)
	if err != nil {
		t.Fatalf("ExtractAttachmentFromPart failed: %v", err)
	}

	if result.Action != "skipped" {
		t.Errorf("Expected vCard to be skipped, got action: %s", result.Action)
	}
	if result.Reason != "content-type-filtered" {
		t.Errorf("Expected reason 'content-type-filtered', got: %s", result.Reason)
	}
}

func TestContentTypeFiltering_WAPContainerNotExtracted(t *testing.T) {
	extractor := NewAttachmentExtractor(t.TempDir())
	config := GetDefaultContentTypeConfig()

	// Create WAP container content
	wapContent := strings.Repeat("WAP container data ", 100)

	part := &MMSPart{
		Seq:         0,
		ContentType: "application/vnd.wap.multipart.related",
		Data:        base64.StdEncoding.EncodeToString([]byte(wapContent)),
	}

	result, err := extractor.ExtractAttachmentFromPart(part, config)
	if err != nil {
		t.Fatalf("ExtractAttachmentFromPart failed: %v", err)
	}

	if result.Action != "skipped" {
		t.Errorf("Expected WAP container to be skipped, got action: %s", result.Action)
	}
	if result.Reason != "content-type-filtered" {
		t.Errorf("Expected reason 'content-type-filtered', got: %s", result.Reason)
	}
}

func TestContentTypeFiltering_ImageExtracted(t *testing.T) {
	extractor := NewAttachmentExtractor(t.TempDir())
	config := GetDefaultContentTypeConfig()

	// Create image content (reuse helper from main test file)
	imageData := createTestPNGData()

	part := &MMSPart{
		Seq:         0,
		ContentType: "image/png",
		Filename:    "photo.png",
		Data:        imageData,
	}

	result, err := extractor.ExtractAttachmentFromPart(part, config)
	if err != nil {
		t.Fatalf("ExtractAttachmentFromPart failed: %v", err)
	}

	if result.Action != "extracted" {
		t.Errorf("Expected image to be extracted, got action: %s", result.Action)
	}
	if !result.UpdatePart {
		t.Errorf("Expected UpdatePart to be true for extracted image")
	}
}

func TestContentTypeFiltering_ContentTypeWithParameters(t *testing.T) {
	extractor := NewAttachmentExtractor(t.TempDir())
	config := GetDefaultContentTypeConfig()

	tests := []struct {
		name        string
		contentType string
		shouldSkip  bool
	}{
		{"JPEG with charset", "image/jpeg; charset=utf-8", false},
		{"Text with charset", "text/plain; charset=utf-8", true},
		{"SMIL with boundary", "application/smil; boundary=something", true},
		{"PDF with encoding", "application/pdf; encoding=binary", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use appropriate data based on whether it should be skipped
			var testData string
			if tt.shouldSkip {
				testData = base64.StdEncoding.EncodeToString([]byte(strings.Repeat("test content ", 100)))
			} else {
				testData = createTestPNGData()
			}

			part := &MMSPart{
				Seq:         0,
				ContentType: tt.contentType,
				Data:        testData,
			}

			result, err := extractor.ExtractAttachmentFromPart(part, config)
			if err != nil {
				t.Fatalf("ExtractAttachmentFromPart failed: %v", err)
			}

			if tt.shouldSkip {
				if result.Action != "skipped" || result.Reason != "content-type-filtered" {
					t.Errorf("Expected %s to be skipped due to content type, got action: %s, reason: %s",
						tt.contentType, result.Action, result.Reason)
				}
			} else {
				if result.Action == "skipped" && result.Reason == "content-type-filtered" {
					t.Errorf("Expected %s to not be filtered by content type, but it was skipped",
						tt.contentType)
				}
			}
		})
	}
}

func TestContentTypeFiltering_MixedMMSParts(t *testing.T) {
	extractor := NewAttachmentExtractor(t.TempDir())
	config := GetDefaultContentTypeConfig()

	// Create MMS with mix of extractable and non-extractable parts
	mms := &MMS{
		Date:    1640995200000, // 2022-01-01
		Address: "555-1234",
		Parts: []MMSPart{
			{
				Seq:         0,
				ContentType: "application/smil",
				Data:        base64.StdEncoding.EncodeToString([]byte(strings.Repeat("<smil>content</smil>", 50))),
			},
			{
				Seq:         1,
				ContentType: "text/plain",
				Text:        "Hello, this is the message text",
				Data:        "",
			},
			{
				Seq:         2,
				ContentType: "image/jpeg",
				Filename:    "photo.jpg",
				Data:        createTestJPEGData(),
			},
			{
				Seq:         3,
				ContentType: "text/x-vCard",
				Data:        base64.StdEncoding.EncodeToString([]byte(strings.Repeat("BEGIN:VCARD\nFN:Contact\nEND:VCARD\n", 30))),
			},
			{
				Seq:         4,
				ContentType: "image/png",
				Filename:    "screenshot.png",
				Data:        createTestPNGData(),
			},
		},
	}

	summary, err := extractor.ExtractAttachmentsFromMMS(mms, config)
	if err != nil {
		t.Fatalf("ExtractAttachmentsFromMMS failed: %v", err)
	}

	// Verify summary counts
	if summary.TotalParts != 5 {
		t.Errorf("Expected 5 total parts, got %d", summary.TotalParts)
	}
	if summary.ExtractedCount != 2 {
		t.Errorf("Expected 2 extracted (images), got %d", summary.ExtractedCount)
	}
	if summary.SkippedCount != 3 {
		t.Errorf("Expected 3 skipped (SMIL, text, vCard), got %d", summary.SkippedCount)
	}

	// Verify specific part behaviors
	parts := mms.Parts
	
	// SMIL part should be unchanged
	if parts[0].Data == "" {
		t.Errorf("Expected SMIL part data to be preserved")
	}
	if parts[0].Path != "" {
		t.Errorf("Expected SMIL part to have no path")
	}

	// Text part should be unchanged (and had no data to begin with)
	if parts[1].Text == "" {
		t.Errorf("Expected text part text to be preserved")
	}

	// Image parts should be extracted
	if parts[2].Data != "" {
		t.Errorf("Expected JPEG part data to be cleared after extraction")
	}
	if parts[2].Path == "" {
		t.Errorf("Expected JPEG part to have path set")
	}

	// vCard part should be unchanged
	if parts[3].Data == "" {
		t.Errorf("Expected vCard part data to be preserved")
	}

	// PNG part should be extracted
	if parts[4].Data != "" {
		t.Errorf("Expected PNG part data to be cleared after extraction")
	}
	if parts[4].Path == "" {
		t.Errorf("Expected PNG part to have path set")
	}
}