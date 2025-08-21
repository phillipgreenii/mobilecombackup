package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillipgreenii/mobilecombackup/pkg/manifest"
	"gopkg.in/yaml.v3"
)

func TestReprocessContactsManifestUpdate(t *testing.T) {
	// Create a temporary test directory
	tempDir, err := os.MkdirTemp("", "reprocess-contacts-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Initialize a basic repository structure
	if err := setupBasicRepository(tempDir); err != nil {
		t.Fatalf("Failed to setup repository: %v", err)
	}

	// Create a test calls.xml file with contact information
	callsDir := filepath.Join(tempDir, "calls")
	if err := os.MkdirAll(callsDir, 0750); err != nil {
		t.Fatalf("Failed to create calls directory: %v", err)
	}

	testCallsXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<calls count="1">
  <call number="+1234567890" duration="30" date="1640995200000" type="1" 
        readable_date="Jan 1, 2022 12:00:00 AM" contact_name="Test Contact" />
</calls>`

	callsFile := filepath.Join(callsDir, "calls_2022.xml")
	if err := os.WriteFile(callsFile, []byte(testCallsXML), 0600); err != nil {
		t.Fatalf("Failed to write test calls file: %v", err)
	}

	// Generate initial manifest
	manifestGenerator := manifest.NewManifestGenerator(tempDir)
	initialManifest, err := manifestGenerator.GenerateFileManifest()
	if err != nil {
		t.Fatalf("Failed to generate initial manifest: %v", err)
	}

	if err := manifestGenerator.WriteManifestFiles(initialManifest); err != nil {
		t.Fatalf("Failed to write initial manifest: %v", err)
	}

	// Get the initial checksum of contacts.yaml
	initialContactsInfo := getFileInfoFromManifest(t, tempDir, "contacts.yaml")
	if initialContactsInfo == nil {
		t.Fatal("contacts.yaml not found in initial manifest")
	}

	// Run reprocess-contacts command
	err = runReprocessContactsTestHelper(tempDir)
	if err != nil {
		t.Fatalf("Failed to run reprocess-contacts: %v", err)
	}

	// Find contacts.yaml in updated manifest
	updatedContactsInfo := getFileInfoFromManifest(t, tempDir, "contacts.yaml")
	if updatedContactsInfo == nil {
		t.Fatal("contacts.yaml not found in updated manifest")
	}

	// Verify that the checksum changed (contacts.yaml was updated)
	if initialContactsInfo.Checksum == updatedContactsInfo.Checksum {
		t.Error("contacts.yaml checksum did not change after reprocessing")
	}

	// Verify that the manifest checksum file was also updated
	manifestChecksumPath := filepath.Join(tempDir, "files.yaml.sha256")
	if _, err := os.Stat(manifestChecksumPath); os.IsNotExist(err) {
		t.Error("files.yaml.sha256 was not created")
	}

	// Verify checksum file contains proper format
	checksumContent, err := os.ReadFile(manifestChecksumPath) // nolint:gosec // Test-controlled path
	if err != nil {
		t.Fatalf("Failed to read checksum file: %v", err)
	}

	checksumStr := string(checksumContent)
	if !strings.Contains(checksumStr, "files.yaml") {
		t.Errorf("Checksum file does not contain proper format, got: %s", checksumStr)
	}

	t.Logf("Successfully verified manifest update - initial checksum: %s, updated checksum: %s",
		initialContactsInfo.Checksum, updatedContactsInfo.Checksum)
}

// setupBasicRepository creates the minimal repository structure needed for testing
func setupBasicRepository(dir string) error {
	// Create marker file
	markerContent := `repository_structure_version: "1"
created_at: "2024-01-01T00:00:00Z"
created_by: "test"`

	if err := os.WriteFile(filepath.Join(dir, ".mobilecombackup.yaml"), []byte(markerContent), 0600); err != nil {
		return err
	}

	// Create empty contacts.yaml
	if err := os.WriteFile(filepath.Join(dir, "contacts.yaml"), []byte("contacts: []\n"), 0600); err != nil {
		return err
	}

	// Create summary.yaml
	summaryContent := `counts:
  calls: 0
  sms: 0`

	if err := os.WriteFile(filepath.Join(dir, "summary.yaml"), []byte(summaryContent), 0600); err != nil {
		return err
	}

	// Create required directories
	for _, subdir := range []string{"calls", "sms", "attachments"} {
		if err := os.MkdirAll(filepath.Join(dir, subdir), 0750); err != nil {
			return err
		}
	}

	return nil
}

// runReprocessContactsTestHelper runs the reprocess-contacts logic without CLI overhead
func runReprocessContactsTestHelper(testRepoRoot string) error {
	// Set up test variables
	oldRepoRoot := repoRoot
	oldQuiet := quiet
	oldVerbose := verbose
	oldReprocessDryRun := reprocessDryRun

	defer func() {
		repoRoot = oldRepoRoot
		quiet = oldQuiet
		verbose = oldVerbose
		reprocessDryRun = oldReprocessDryRun
	}()

	// Configure for testing
	repoRoot = testRepoRoot
	quiet = true
	verbose = false
	reprocessDryRun = false

	// Call the reprocess-contacts function directly
	return runReprocessContacts(nil, []string{})
}

// getFileInfoFromManifest reads the manifest and returns info for a specific file
func getFileInfoFromManifest(t *testing.T, repoRoot, filename string) *manifest.FileEntry {
	t.Helper()

	manifestData, err := readManifestFile(repoRoot)
	if err != nil {
		t.Fatalf("Failed to read manifest: %v", err)
	}

	for _, file := range manifestData.Files {
		if file.Name == filename {
			return &file
		}
	}

	return nil
}

// readManifestFile reads and parses the files.yaml manifest
func readManifestFile(repoRoot string) (*manifest.FileManifest, error) {
	manifestPath := filepath.Join(repoRoot, "files.yaml")
	data, err := os.ReadFile(manifestPath) // nolint:gosec // Test-controlled path
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	var fileManifest manifest.FileManifest
	if err := yaml.Unmarshal(data, &fileManifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &fileManifest, nil
}
