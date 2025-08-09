package importer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phillipgreen/mobilecombackup/pkg/sms"
)

func TestSMSImporter_BUG016_MessagesNotWritten(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	toProcess := filepath.Join(tempDir, "to_process")

	// Initialize repository structure
	if err := os.MkdirAll(filepath.Join(repoRoot, "sms"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(toProcess, 0755); err != nil {
		t.Fatal(err)
	}

	// Copy test SMS file
	testData := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="2">
  <sms protocol="0" address="5551234567" date="1389830400000" type="1" subject="null" body="Hello world" toa="null" sc_toa="null" service_center="null" read="1" status="-1" locked="0" date_sent="0" sub_id="-1" readable_date="Jan 15, 2014 4:00:00 PM" contact_name="Test Contact" />
  <sms protocol="0" address="5559876543" date="1389834000000" type="2" subject="null" body="Reply message" toa="null" sc_toa="null" service_center="null" read="1" status="-1" locked="0" date_sent="0" sub_id="-1" readable_date="Jan 15, 2014 5:00:00 PM" contact_name="Another Contact" />
</smses>`

	smsFile := filepath.Join(toProcess, "sms-test.xml")
	if err := os.WriteFile(smsFile, []byte(testData), 0644); err != nil {
		t.Fatal(err)
	}

	// Create importer
	options := &ImportOptions{
		RepoRoot: repoRoot,
		Paths:    []string{smsFile},
		Filter:   "sms",
		Quiet:    true,
	}

	importer := NewSMSImporter(options)

	// Import the file
	summary, err := importer.Import()
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify import summary
	if summary.Total.Added != 2 {
		t.Errorf("Expected 2 messages added, got %d", summary.Total.Added)
	}

	// Check if SMS file was created in repository
	smsRepoFile := filepath.Join(repoRoot, "sms", "sms-2014.xml")
	if _, err := os.Stat(smsRepoFile); os.IsNotExist(err) {
		t.Errorf("Expected SMS file not created: %s", smsRepoFile)
	}

	// Read and verify the contents
	reader := sms.NewXMLSMSReader(repoRoot)
	messages, err := reader.ReadMessages(2014)
	if err != nil {
		t.Fatalf("Failed to read messages: %v", err)
	}

	// BUG-016: This is where the bug manifests - no messages are written
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages in repository, got %d", len(messages))
		
		// Read the actual file content to debug
		content, _ := os.ReadFile(smsRepoFile)
		t.Logf("Actual file content:\n%s", string(content))
	}

	// Verify individual messages
	if len(messages) > 0 {
		if messages[0].GetAddress() != "5551234567" {
			t.Errorf("First message has wrong address: %s", messages[0].GetAddress())
		}
	}

	if len(messages) > 1 {
		if messages[1].GetAddress() != "5559876543" {
			t.Errorf("Second message has wrong address: %s", messages[1].GetAddress())
		}
	}
}