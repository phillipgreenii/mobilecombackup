package importer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phillipgreenii/mobilecombackup/pkg/logging"

	"github.com/spf13/afero"
)

func TestSMSImporter_BUG023_FilesInCorrectDirectory(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	toProcess := filepath.Join(tempDir, "to_process")

	// Create complete repository structure
	setupTestRepository(t, repoRoot)

	if err := os.MkdirAll(toProcess, 0750); err != nil {
		t.Fatal(err)
	}

	// Create test SMS file with data from 2024
	testData := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="1">
  <sms protocol="0" address="5551234567" date="1704067200000" type="1" subject="null" 
       body="Test message 2024" toa="null" sc_toa="null" service_center="null" 
       read="1" status="-1" locked="0" date_sent="0" sub_id="-1" 
       readable_date="Jan 1, 2024 12:00:00 AM" contact_name="Test Contact" />
</smses>`

	smsFile := filepath.Join(toProcess, "sms-2024.xml")
	if err := os.WriteFile(smsFile, []byte(testData), 0600); err != nil {
		t.Fatal(err)
	}

	// Create importer
	options := &ImportOptions{
		RepoRoot: repoRoot,
		Paths:    []string{smsFile},
		Filter:   "sms",
		Quiet:    true,
		Fs:       afero.NewOsFs(),
	}

	importer, err := NewImporter(options, logging.NewNullLogger())
	if err != nil {
		t.Fatalf("Failed to create importer: %v", err)
	}

	// Import the file
	_, err = importer.Import()
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// BUG-023: Verify SMS file is in sms/ directory, not root
	wrongLocation := filepath.Join(repoRoot, "sms-2024.xml")
	if _, err := os.Stat(wrongLocation); err == nil {
		t.Errorf("BUG-023: SMS file found in wrong location (root): %s", wrongLocation)
	}

	// Verify SMS file is in correct location
	correctLocation := filepath.Join(repoRoot, "sms", "sms-2024.xml")
	if _, err := os.Stat(correctLocation); os.IsNotExist(err) {
		t.Errorf("SMS file not found in correct location: %s", correctLocation)
	}

	// Read the file to ensure it has content
	content, err := os.ReadFile(correctLocation) // #nosec G304 // Test-controlled path
	if err != nil {
		t.Fatalf("Failed to read SMS file: %v", err)
	}

	// Verify it's not empty
	if len(content) < 100 {
		t.Errorf("SMS file seems too small: %d bytes", len(content))
	}

	// Verify the message is in the file
	if string(content) == `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>\n<smses count="1"></smses>` {
		t.Error("SMS file is empty (only has wrapper)")
	}
}
