package sms

import (
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
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func TestXMLSMSReader_Integration_WithTestData(t *testing.T) {
	// Use existing test data from testdata/to_process/sms-test.xml
	tempDir := t.TempDir()
	smsDir := filepath.Join(tempDir, "sms")
	err := copyFile("../../testdata/to_process/sms-test.xml", filepath.Join(smsDir, "sms-2013.xml"))
	if err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}

	reader := NewXMLSMSReader(tempDir)

	// Test ReadMessages with real data
	messages, err := reader.ReadMessages(2013)
	if err != nil {
		t.Fatalf("ReadMessages failed with real data: %v", err)
	}

	// The test file has 15 messages (count="15") with a mix of SMS and MMS
	if len(messages) != 15 {
		t.Fatalf("Expected 15 messages from test data, got %d", len(messages))
	}

	// Count SMS vs MMS messages
	var smsCount, mmsCount int
	var receivedCount, sentCount int
	for _, msg := range messages {
		switch msg.(type) {
		case SMS:
			smsCount++
		case MMS:
			mmsCount++
		}

		switch msg.GetType() {
		case ReceivedMessage:
			receivedCount++
		case SentMessage:
			sentCount++
		}

		// Validate all messages have required fields
		if msg.GetDate().IsZero() {
			t.Error("Found message with zero date")
		}
		if msg.GetAddress() == "" {
			t.Error("Found message with empty address")
		}
		if msg.GetContactName() == "" {
			t.Error("Found message with empty contact name")
		}
	}

	// Test data should contain both SMS and MMS
	if smsCount == 0 {
		t.Error("Expected some SMS messages in test data")
	}
	if mmsCount == 0 {
		t.Error("Expected some MMS messages in test data")
	}

	// Should have both received and sent messages
	if receivedCount == 0 {
		t.Error("Expected some received messages")
	}
	if sentCount == 0 {
		t.Error("Expected some sent messages")
	}

	// Test streaming with real data
	var streamedCount int
	err = reader.StreamMessages(2013, func(msg Message) error {
		streamedCount++
		return nil
	})
	if err != nil {
		t.Fatalf("StreamMessages failed with real data: %v", err)
	}

	if streamedCount != 15 {
		t.Errorf("Expected to stream 15 messages, got %d", streamedCount)
	}
}

func TestXMLSMSReader_Integration_GetMessageCount(t *testing.T) {
	tempDir := t.TempDir()
	smsDir := filepath.Join(tempDir, "sms")
	err := copyFile("../../testdata/to_process/sms-test.xml", filepath.Join(smsDir, "sms-2013.xml"))
	if err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}

	reader := NewXMLSMSReader(tempDir)

	// Test GetMessageCount with real data
	count, err := reader.GetMessageCount(2013)
	if err != nil {
		t.Fatalf("GetMessageCount failed: %v", err)
	}

	// The test file declares count="15"
	if count != 15 {
		t.Fatalf("Expected 15 messages (from count attribute) in test file, got %d", count)
	}
}

func TestXMLSMSReader_Integration_GetAvailableYears(t *testing.T) {
	tempDir := t.TempDir()
	smsDir := filepath.Join(tempDir, "sms")
	
	// Copy test files to multiple years
	err := copyFile("../../testdata/to_process/sms-test.xml", filepath.Join(smsDir, "sms-2013.xml"))
	if err != nil {
		t.Fatalf("Failed to setup 2013 test data: %v", err)
	}
	
	err = copyFile("../../testdata/it/scenerio-00/original_repo_root/sms/sms-2015.xml", filepath.Join(smsDir, "sms-2015.xml"))
	if err != nil {
		t.Fatalf("Failed to setup 2015 test data: %v", err)
	}

	reader := NewXMLSMSReader(tempDir)

	years, err := reader.GetAvailableYears()
	if err != nil {
		t.Fatalf("GetAvailableYears failed: %v", err)
	}

	// Should find both years, sorted
	if len(years) != 2 {
		t.Fatalf("Expected 2 years, got %d: %v", len(years), years)
	}
	if years[0] != 2013 || years[1] != 2015 {
		t.Errorf("Expected [2013, 2015], got %v", years)
	}
}

func TestXMLSMSReader_Integration_MMSParts(t *testing.T) {
	tempDir := t.TempDir()
	smsDir := filepath.Join(tempDir, "sms")
	err := copyFile("../../testdata/to_process/sms-test.xml", filepath.Join(smsDir, "sms-2013.xml"))
	if err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}

	reader := NewXMLSMSReader(tempDir)

	// Find MMS messages and verify parts are parsed correctly
	var mmsMessages []MMS
	err = reader.StreamMessages(2013, func(msg Message) error {
		if mms, ok := msg.(MMS); ok {
			mmsMessages = append(mmsMessages, mms)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("StreamMessages failed: %v", err)
	}

	if len(mmsMessages) == 0 {
		t.Fatal("Expected to find MMS messages in test data")
	}

	// Verify MMS parts are parsed correctly
	foundTextPart := false
	foundSMILPart := false
	
	for _, mms := range mmsMessages {
		if len(mms.Parts) == 0 {
			t.Error("Found MMS with no parts")
			continue
		}

		for _, part := range mms.Parts {
			switch part.ContentType {
			case "text/plain":
				foundTextPart = true
				if part.Text == "" {
					t.Error("Found text/plain part with empty text")
				}
			case "application/smil":
				foundSMILPart = true
				if part.Text == "" {
					t.Error("Found application/smil part with empty text")
				}
			}
		}
	}

	if !foundTextPart {
		t.Error("Expected to find text/plain parts in MMS messages")
	}
	if !foundSMILPart {
		t.Error("Expected to find application/smil parts in MMS messages")
	}
}

func TestXMLSMSReader_Integration_ValidateWithActualData(t *testing.T) {
	tempDir := t.TempDir()
	smsDir := filepath.Join(tempDir, "sms")
	err := copyFile("../../testdata/to_process/sms-test.xml", filepath.Join(smsDir, "sms-2013.xml"))
	if err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}

	reader := NewXMLSMSReader(tempDir)

	// The test data has mixed years (similar to calls test data)
	// So validation should fail due to year inconsistency
	err = reader.ValidateSMSFile(2013)
	if err == nil {
		t.Errorf("ValidateSMSFile should have failed due to year inconsistency in test data")
	}
}

func TestXMLSMSReader_Integration_YearConsistency(t *testing.T) {
	tempDir := t.TempDir()
	smsDir := filepath.Join(tempDir, "sms")
	err := copyFile("../../testdata/to_process/sms-test.xml", filepath.Join(smsDir, "sms-2013.xml"))
	if err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}

	reader := NewXMLSMSReader(tempDir)

	// Check year distribution in the test data
	var year2013Count, year2014Count, year2015Count int
	err = reader.StreamMessages(2013, func(msg Message) error {
		msgYear := msg.GetDate().Year()
		switch msgYear {
		case 2013:
			year2013Count++
		case 2014:
			year2014Count++
		case 2015:
			year2015Count++
		default:
			t.Errorf("Unexpected year %d in test data", msgYear)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("StreamMessages failed: %v", err)
	}

	// The test data has mixed years (like calls test data)
	t.Logf("Year distribution: 2013=%d, 2014=%d, 2015=%d", year2013Count, year2014Count, year2015Count)
	
	// Verify we have messages from multiple years
	if year2013Count == 0 {
		t.Error("Expected some messages from 2013")
	}
	if year2014Count == 0 {
		t.Error("Expected some messages from 2014")
	}
}

func TestXMLSMSReader_Integration_GetAttachmentRefs(t *testing.T) {
	tempDir := t.TempDir()
	smsDir := filepath.Join(tempDir, "sms")
	err := copyFile("../../testdata/to_process/sms-test.xml", filepath.Join(smsDir, "sms-2013.xml"))
	if err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}

	reader := NewXMLSMSReader(tempDir)

	// Get attachment references
	refs, err := reader.GetAttachmentRefs(2013)
	if err != nil {
		t.Fatalf("GetAttachmentRefs failed: %v", err)
	}

	// The test data MMS messages may not have extracted attachments yet
	// So this might return empty results, which is fine
	t.Logf("Found %d attachment references", len(refs))

	// Test GetAllAttachmentRefs
	allRefs, err := reader.GetAllAttachmentRefs()
	if err != nil {
		t.Fatalf("GetAllAttachmentRefs failed: %v", err)
	}

	t.Logf("Found %d total attachment references across all years", len(allRefs))
}

func TestXMLSMSReader_Integration_MessageTypes(t *testing.T) {
	tempDir := t.TempDir()
	smsDir := filepath.Join(tempDir, "sms")
	err := copyFile("../../testdata/to_process/sms-test.xml", filepath.Join(smsDir, "sms-2013.xml"))
	if err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}

	reader := NewXMLSMSReader(tempDir)

	// Verify message type detection works correctly
	var smsReceived, smsSent, mmsReceived, mmsSent int
	
	err = reader.StreamMessages(2013, func(msg Message) error {
		switch m := msg.(type) {
		case SMS:
			if m.Type == ReceivedMessage {
				smsReceived++
			} else if m.Type == SentMessage {
				smsSent++
			}
		case MMS:
			// MMS type is determined by msg_box attribute
			if m.MsgBox == 1 {
				mmsReceived++
			} else if m.MsgBox == 2 {
				mmsSent++
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("StreamMessages failed: %v", err)
	}

	t.Logf("SMS: %d received, %d sent", smsReceived, smsSent)
	t.Logf("MMS: %d received, %d sent", mmsReceived, mmsSent)

	// Should have at least some messages of each type
	if smsReceived == 0 && smsSent == 0 {
		t.Error("Expected to find some SMS messages")
	}
	if mmsReceived == 0 && mmsSent == 0 {
		t.Error("Expected to find some MMS messages")
	}
}