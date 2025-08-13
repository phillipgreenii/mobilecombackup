package sms

import (
	"os"
	"testing"
)

// TestAttachmentExtraction_Debug tests attachment extraction with actual test data
func TestAttachmentExtraction_Debug(t *testing.T) {
	// Use actual test data file
	testFile := "../../testdata/to_process/sms-test.xml"

	// Check if test file exists
	if _, err := os.Stat(testFile); err != nil {
		t.Skipf("Test file not found: %v", err)
	}

	// Create temp repository
	tempRepo := t.TempDir()

	// Create SMS reader
	reader := NewXMLSMSReader("")

	// Read test file
	file, err := os.Open(testFile)
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer func() { _ = file.Close() }()

	var messageCount int
	var mmsCount int
	var mmsWithAttachments int

	err = reader.StreamMessagesFromReader(file, func(msg Message) error {
		messageCount++
		t.Logf("Message %d: Type=%T, Date=%d", messageCount, msg, msg.GetDate())

		if mms, ok := msg.(*MMS); ok {
			mmsCount++
			t.Logf("  MMS with %d parts", len(mms.Parts))

			hasAttachments := false
			for i, part := range mms.Parts {
				t.Logf("    Part %d: CT=%s, DataLen=%d", i, part.ContentType, len(part.Data))
				if part.ContentType == "image/png" && len(part.Data) > 0 {
					hasAttachments = true
					t.Logf("      Found PNG attachment! Data preview: %s...", part.Data[:min(50, len(part.Data))])
				}
			}

			if hasAttachments {
				mmsWithAttachments++

				// Test extraction
				extractor := NewAttachmentExtractor(tempRepo)
				config := GetDefaultContentTypeConfig()

				t.Logf("  Testing extraction...")
				summary, err := extractor.ExtractAttachmentsFromMMS(mms, config)
				if err != nil {
					t.Errorf("  ERROR during extraction: %v", err)
				} else {
					t.Logf("  Extraction result: %d extracted, %d referenced, %d skipped",
						summary.ExtractedCount, summary.ReferencedCount, summary.SkippedCount)

					for i, result := range summary.Results {
						t.Logf("    Part %d result: Action=%s, Reason=%s", i, result.Action, result.Reason)
						if result.UpdatePart {
							t.Logf("      Path=%s, Hash=%s", result.Path, result.Hash)
						}
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to stream messages: %v", err)
	}

	t.Logf("Summary: Total messages: %d, MMS messages: %d, MMS with attachments: %d",
		messageCount, mmsCount, mmsWithAttachments)

	// Verify we found the expected data
	if messageCount == 0 {
		t.Error("No messages found in test file")
	}
	if mmsCount == 0 {
		t.Error("No MMS messages found in test file")
	}
	if mmsWithAttachments == 0 {
		t.Error("No MMS messages with attachments found")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
