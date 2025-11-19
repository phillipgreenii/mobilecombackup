package contacts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phillipgreenii/mobilecombackup/pkg/calls"
	"github.com/phillipgreenii/mobilecombackup/pkg/sms"
)

func TestContactExtractor_ExtractFromCall(t *testing.T) {
	t.Run("extracts single contact", func(t *testing.T) {
		manager := NewContactsManager("")
		extractor := NewContactExtractor(manager)

		call := &calls.Call{
			Number:      "+1234567890",
			ContactName: "John Doe",
		}

		extractor.ExtractFromCall(call)

		unprocessed := manager.GetUnprocessedEntries()
		if len(unprocessed) != 1 {
			t.Fatalf("Expected 1 unprocessed contact, got: %d", len(unprocessed))
		}

		if unprocessed[0].PhoneNumber != "+1234567890" {
			t.Errorf("Expected phone +1234567890, got: %s", unprocessed[0].PhoneNumber)
		}

		if len(unprocessed[0].ContactNames) != 1 || unprocessed[0].ContactNames[0] != "John Doe" {
			t.Errorf("Expected contact name 'John Doe', got: %v", unprocessed[0].ContactNames)
		}
	})

	t.Run("splits multiple contact names", func(t *testing.T) {
		manager := NewContactsManager("")
		extractor := NewContactExtractor(manager)

		call := &calls.Call{
			Number:      "+1234567890",
			ContactName: "John Doe, Jane Smith",
		}

		extractor.ExtractFromCall(call)

		unprocessed := manager.GetUnprocessedEntries()
		if len(unprocessed) != 2 {
			t.Fatalf("Expected 2 unprocessed contacts, got: %d", len(unprocessed))
		}
	})

	t.Run("ignores unknown contacts", func(t *testing.T) {
		manager := NewContactsManager("")
		extractor := NewContactExtractor(manager)

		unknownNames := []string{"(Unknown)", "null", ""}
		for _, name := range unknownNames {
			call := &calls.Call{
				Number:      "+1234567890",
				ContactName: name,
			}
			extractor.ExtractFromCall(call)
		}

		unprocessed := manager.GetUnprocessedEntries()
		if len(unprocessed) != 0 {
			t.Errorf("Expected 0 unprocessed contacts for unknown names, got: %d", len(unprocessed))
		}
	})

	t.Run("ignores calls without number", func(t *testing.T) {
		manager := NewContactsManager("")
		extractor := NewContactExtractor(manager)

		call := &calls.Call{
			Number:      "",
			ContactName: "John Doe",
		}

		extractor.ExtractFromCall(call)

		unprocessed := manager.GetUnprocessedEntries()
		if len(unprocessed) != 0 {
			t.Errorf("Expected 0 unprocessed contacts, got: %d", len(unprocessed))
		}
	})
}

func TestContactExtractor_ExtractFromSMS(t *testing.T) {
	t.Run("extracts single contact", func(t *testing.T) {
		manager := NewContactsManager("")
		extractor := NewContactExtractor(manager)

		smsMsg := sms.SMS{
			Address:     "+1234567890",
			ContactName: "John Doe",
		}

		extractor.ExtractFromSMS(smsMsg)

		unprocessed := manager.GetUnprocessedEntries()
		if len(unprocessed) != 1 {
			t.Fatalf("Expected 1 unprocessed contact, got: %d", len(unprocessed))
		}
	})

	t.Run("splits multiple contact names", func(t *testing.T) {
		manager := NewContactsManager("")
		extractor := NewContactExtractor(manager)

		smsMsg := sms.SMS{
			Address:     "+1234567890",
			ContactName: "John Doe, Jane Smith, Bob Wilson",
		}

		extractor.ExtractFromSMS(smsMsg)

		unprocessed := manager.GetUnprocessedEntries()
		if len(unprocessed) != 3 {
			t.Fatalf("Expected 3 unprocessed contacts, got: %d", len(unprocessed))
		}
	})
}

func TestContactExtractor_ExtractFromMMS(t *testing.T) {
	t.Run("extracts single contact", func(t *testing.T) {
		manager := NewContactsManager("")
		extractor := NewContactExtractor(manager)

		mmsMsg := sms.MMS{
			Address:     "+1234567890",
			ContactName: "John Doe",
		}

		extractor.ExtractFromMMS(mmsMsg)

		unprocessed := manager.GetUnprocessedEntries()
		if len(unprocessed) != 1 {
			t.Fatalf("Expected 1 unprocessed contact, got: %d", len(unprocessed))
		}
	})
}

func TestContactExtractor_ExtractFromCallsFile(t *testing.T) {
	t.Run("extracts from valid calls XML file", func(t *testing.T) {
		tmpDir := t.TempDir()
		callsFile := filepath.Join(tmpDir, "calls.xml")

		// Create test XML file
		xmlContent := `<?xml version='1.0' encoding='UTF-8'?>
<calls count="2">
  <call number="+1234567890" duration="30" date="1234567890000" type="1" readable_date="2009-02-13 23:31:30" contact_name="John Doe" />
  <call number="+0987654321" duration="45" date="1234567900000" type="2" readable_date="2009-02-13 23:31:40" contact_name="Jane Smith" />
</calls>`

		if err := os.WriteFile(callsFile, []byte(xmlContent), 0600); err != nil {
			t.Fatal(err)
		}

		manager := NewContactsManager("")
		extractor := NewContactExtractor(manager)

		count, err := extractor.ExtractFromCallsFile(callsFile)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if count != 2 {
			t.Errorf("Expected 2 calls processed, got: %d", count)
		}

		unprocessed := manager.GetUnprocessedEntries()
		if len(unprocessed) != 2 {
			t.Errorf("Expected 2 contacts extracted, got: %d", len(unprocessed))
		}
	})

	t.Run("error on non-existent file", func(t *testing.T) {
		manager := NewContactsManager("")
		extractor := NewContactExtractor(manager)

		_, err := extractor.ExtractFromCallsFile("/nonexistent/file.xml")
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})
}

func TestIsUnknownContact(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty string", "", true},
		{"Unknown", "(Unknown)", true},
		{"null", "null", true},
		{"valid name", "John Doe", false},
		{"whitespace", "  ", false}, // spaces are not considered unknown
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isUnknownContact(tt.input)
			if result != tt.expected {
				t.Errorf("isUnknownContact(%q) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}
