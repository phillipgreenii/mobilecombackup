package sms

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
)

// TestSMS_Timestamp tests the Timestamp method on SMS
func TestSMS_Timestamp(t *testing.T) {
	t.Parallel()

	sms := SMS{
		Date: 1410881505425, // milliseconds
	}

	ts := sms.Timestamp()
	expected := time.Unix(1410881505, 425000000).UTC()

	if !ts.Equal(expected) {
		t.Errorf("Expected timestamp %v, got %v", expected, ts)
	}
}

// TestMMS_Timestamp tests the Timestamp method on MMS
func TestMMS_Timestamp(t *testing.T) {
	t.Parallel()

	mms := MMS{
		Date: 1441140251000, // milliseconds
	}

	ts := mms.Timestamp()
	expected := time.Unix(1441140251, 0).UTC()

	if !ts.Equal(expected) {
		t.Errorf("Expected timestamp %v, got %v", expected, ts)
	}
}

// TestNewAttachmentExtractorWithLogger tests the constructor with logger
func TestNewAttachmentExtractorWithLogger(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for testing
	tempDir := t.TempDir()
	fs := afero.NewOsFs()

	// Test with nil logger
	extractor := NewAttachmentExtractorWithLogger(tempDir, nil, fs)
	if extractor == nil {
		t.Error("Expected non-nil extractor with nil logger")
	}

	// Should not panic when using extractor methods (verify it was created successfully)
	_ = GetDefaultContentTypeConfig()
}

// TestXMLSMSReader_StreamMessages tests the StreamMessages method
func TestXMLSMSReader_StreamMessages(t *testing.T) {
	t.Parallel()

	// Create a temporary repository with test data
	tempDir := t.TempDir()
	smsDir := filepath.Join(tempDir, "sms")
	err := os.MkdirAll(smsDir, 0750)
	if err != nil {
		t.Fatalf("Failed to create sms directory: %v", err)
	}

	// Create test XML with both SMS and MMS
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="3">
  <sms address="5551234567" date="1410881505425" type="1" body="Test SMS" />
  <mms date="1441140251000" ct_t="application/vnd.wap.multipart.related">
    <parts>
      <part ct="text/plain" text="Test MMS" />
    </parts>
    <addrs>
      <addr address="5559876543" type="151" />
    </addrs>
  </mms>
  <sms address="5555555555" date="1410881505426" type="2" body="Another SMS" />
</smses>`

	testFile := filepath.Join(smsDir, "sms-2014.xml")
	err = os.WriteFile(testFile, []byte(testXML), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	reader := NewXMLSMSReader(tempDir)

	t.Run("stream_all_messages", func(t *testing.T) {
		count := 0
		err := reader.StreamMessages(func(msg Message) error {
			count++
			return nil
		})

		if err != nil {
			t.Fatalf("StreamMessages failed: %v", err)
		}

		// Should get all 3 messages (2 SMS + 1 MMS)
		expectedCount := 3
		if count != expectedCount {
			t.Errorf("Expected %d messages, got %d", expectedCount, count)
		}
	})

	t.Run("empty_repository", func(t *testing.T) {
		emptyDir := t.TempDir()
		emptySMSDir := filepath.Join(emptyDir, "sms")
		err := os.MkdirAll(emptySMSDir, 0750)
		if err != nil {
			t.Fatalf("Failed to create empty sms directory: %v", err)
		}

		emptyReader := NewXMLSMSReader(emptyDir)

		count := 0
		err = emptyReader.StreamMessages(func(msg Message) error {
			count++
			return nil
		})

		if err != nil {
			t.Fatalf("StreamMessages failed on empty repository: %v", err)
		}

		if count != 0 {
			t.Errorf("Expected 0 messages in empty repository, got %d", count)
		}
	})
}
