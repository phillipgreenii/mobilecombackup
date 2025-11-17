package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillipgreenii/mobilecombackup/pkg/logging"
	"github.com/phillipgreenii/mobilecombackup/pkg/manifest"

	"github.com/spf13/afero"
)

// setupTestRepository creates a complete test repository with all required files
func setupTestRepository(t *testing.T, repoRoot string) {
	t.Helper()

	// Create directory structure
	dirs := []string{
		repoRoot,
		filepath.Join(repoRoot, "calls"),
		filepath.Join(repoRoot, "sms"),
		filepath.Join(repoRoot, "attachments"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create marker file
	markerContent := `repository_structure_version: "1"
created_at: "2024-01-15T10:30:00Z"
created_by: "mobilecombackup v1.0.0"
`
	markerPath := filepath.Join(repoRoot, ".mobilecombackup.yaml")
	if err := os.WriteFile(markerPath, []byte(markerContent), 0600); err != nil {
		t.Fatalf("Failed to create marker file: %v", err)
	}

	// Create empty contacts file
	contactsPath := filepath.Join(repoRoot, "contacts.yaml")
	contactsContent := "contacts: []\n"
	if err := os.WriteFile(contactsPath, []byte(contactsContent), 0600); err != nil {
		t.Fatalf("Failed to create contacts file: %v", err)
	}

	// Create summary file
	summaryPath := filepath.Join(repoRoot, "summary.yaml")
	summaryContent := `counts:
  calls: 0
  sms: 0
`
	if err := os.WriteFile(summaryPath, []byte(summaryContent), 0600); err != nil {
		t.Fatalf("Failed to create summary file: %v", err)
	}

	// Generate and write manifest files
	manifestGenerator := manifest.NewManifestGenerator(repoRoot, afero.NewOsFs())
	fileManifest, err := manifestGenerator.GenerateFileManifest()
	if err != nil {
		t.Fatalf("Failed to generate file manifest: %v", err)
	}

	if err := manifestGenerator.WriteManifestFiles(fileManifest); err != nil {
		t.Fatalf("Failed to write manifest files: %v", err)
	}
}

func TestImporter_ContactExtraction_SMS(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="3">
  <sms address="5551234567" contact_name="John Doe" body="Hello" date="1609459200000" type="2" readable_date="Jan 1, 2021 12:00:00 AM" />
  <sms address="5559876543" contact_name="Jane Smith" body="Hi there" date="1609459260000" type="1" readable_date="Jan 1, 2021 12:01:00 AM" />
  <sms address="5551111111" contact_name="Bob Ross" body="Happy trees" date="1609459320000" type="2" readable_date="Jan 1, 2021 12:02:00 AM" />
</smses>`

	testContactExtraction(t, testXML, "sms-test.xml", "sms", 3, 0)
}

func TestImporter_ContactExtraction_Calls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<calls count="3">
  <call number="5551234567" contact_name="John Doe" duration="120" date="1609459200000" type="1" readable_date="Jan 1, 2021 12:00:00 AM" />
  <call number="5559876543" contact_name="Jane Smith" duration="60" date="1609459260000" type="2" readable_date="Jan 1, 2021 12:01:00 AM" />
  <call number="5551111111" contact_name="Bob Ross" duration="300" date="1609459320000" type="3" readable_date="Jan 1, 2021 12:02:00 AM" />
</calls>`

	testContactExtraction(t, testXML, "calls-test.xml", "calls", 0, 3)
}

func TestImporter_ContactExtraction_MMS(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")

	// Create complete repository structure
	setupTestRepository(t, repoRoot)

	// Create test MMS file with contact names (only primary address has contact name)
	testFile := filepath.Join(tempDir, "sms-mms-test.xml")
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="2">
  <mms date="1609459200000" msg_box="1" address="5551234567" contact_name="John Doe" m_type="132" readable_date="Jan 1, 2021 12:00:00 AM">
    <parts>
      <part seq="0" ct="text/plain" text="Hello MMS" />
    </parts>
    <addrs>
      <addr address="5559876543" type="151" charset="106" />
    </addrs>
  </mms>
  <mms date="1609459260000" msg_box="2" address="5555555555" contact_name="Jane Smith" m_type="128" readable_date="Jan 1, 2021 12:01:00 AM">
    <parts>
      <part seq="0" ct="text/plain" text="Another MMS" />
    </parts>
  </mms>
</smses>`

	if err := os.WriteFile(testFile, []byte(testXML), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run import
	options := &ImportOptions{
		RepoRoot: repoRoot,
		Paths:    []string{testFile},
		Filter:   "sms",
		DryRun:   false,
		Quiet:    true,
		Fs:       afero.NewOsFs(),
	}

	importer, err := NewImporter(options, logging.NewNullLogger())
	if err != nil {
		t.Fatalf("Failed to create importer: %v", err)
	}

	summary, err := importer.Import()
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify import worked
	if summary.SMS.Total.Added != 2 {
		t.Errorf("Expected 2 MMS messages imported, got %d", summary.SMS.Total.Added)
	}

	// Check contacts.yaml was created and contains extracted contacts
	contactsPath := filepath.Join(repoRoot, "contacts.yaml")
	if _, err := os.Stat(contactsPath); os.IsNotExist(err) {
		t.Fatal("contacts.yaml should exist after import")
	}

	// Read contacts.yaml and verify content
	content, err := os.ReadFile(contactsPath) // nolint:gosec // Test-controlled path
	if err != nil {
		t.Fatalf("Failed to read contacts.yaml: %v", err)
	}

	yamlStr := string(content)

	// Verify only primary addresses were extracted (MMS additional addresses don't have contact names)
	expectedNumbers := []string{
		"5551234567",
		"5555555555",
	}
	expectedNames := []string{
		"John Doe",
		"Jane Smith",
	}

	for _, number := range expectedNumbers {
		if !strings.Contains(yamlStr, fmt.Sprintf("phone_number: \"%s\"", number)) {
			t.Errorf("Expected to find phone number '%s' in contacts.yaml", number)
		}
	}

	for _, name := range expectedNames {
		if !strings.Contains(yamlStr, name) {
			t.Errorf("Expected to find contact name '%s' in contacts.yaml", name)
		}
	}

	// Should NOT contain the additional address since it doesn't have a contact name
	if strings.Contains(yamlStr, fmt.Sprintf("phone_number: \"%s\"", "5559876543")) {
		t.Error("Should not extract contacts from MMS additional addresses without contact names")
	}
}

func TestImporter_ContactExtraction_ExistingContacts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")

	// Create basic repository structure (without files.yaml)
	dirs := []string{
		repoRoot,
		filepath.Join(repoRoot, "calls"),
		filepath.Join(repoRoot, "sms"),
		filepath.Join(repoRoot, "attachments"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create marker file
	markerContent := `repository_structure_version: "1"
created_at: "2024-01-15T10:30:00Z"
created_by: "mobilecombackup v1.0.0"
`
	markerPath := filepath.Join(repoRoot, ".mobilecombackup.yaml")
	if err := os.WriteFile(markerPath, []byte(markerContent), 0600); err != nil {
		t.Fatalf("Failed to create marker file: %v", err)
	}

	// Create summary file
	summaryPath := filepath.Join(repoRoot, "summary.yaml")
	summaryContent := `counts:
  calls: 0
  sms: 0
`
	if err := os.WriteFile(summaryPath, []byte(summaryContent), 0600); err != nil {
		t.Fatalf("Failed to create summary file: %v", err)
	}

	// Create existing contacts.yaml with custom content
	contactsPath := filepath.Join(repoRoot, "contacts.yaml")
	existingYaml := `contacts:
  - name: "Existing Contact"
    numbers:
      - "5550000000"
unprocessed:
  - "5551111111: Previous Entry"
`
	if err := os.WriteFile(contactsPath, []byte(existingYaml), 0600); err != nil {
		t.Fatalf("Failed to create existing contacts.yaml: %v", err)
	}

	// Generate and write manifest files AFTER creating custom contacts.yaml
	manifestGenerator := manifest.NewManifestGenerator(repoRoot, afero.NewOsFs())
	fileManifest, err := manifestGenerator.GenerateFileManifest()
	if err != nil {
		t.Fatalf("Failed to generate file manifest: %v", err)
	}

	if err := manifestGenerator.WriteManifestFiles(fileManifest); err != nil {
		t.Fatalf("Failed to write manifest files: %v", err)
	}

	// Create test SMS file
	testFile := filepath.Join(tempDir, "sms-test.xml")
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="2">
  <sms address="5551234567" contact_name="John Doe" body="Hello" date="1609459200000" type="2" readable_date="Jan 1, 2021 12:00:00 AM" />
  <sms address="5559876543" contact_name="Jane Smith" body="Hi there" date="1609459260000" type="1" readable_date="Jan 1, 2021 12:01:00 AM" />
</smses>`

	if err := os.WriteFile(testFile, []byte(testXML), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run import
	options := &ImportOptions{
		RepoRoot: repoRoot,
		Paths:    []string{testFile},
		Filter:   "sms",
		DryRun:   false,
		Quiet:    true,
		Fs:       afero.NewOsFs(),
	}

	importer, err := NewImporter(options, logging.NewNullLogger())
	if err != nil {
		t.Fatalf("Failed to create importer: %v", err)
	}

	_, err = importer.Import()
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Read updated contacts.yaml and verify content
	content, err := os.ReadFile(contactsPath) // nolint:gosec // Test-controlled path
	if err != nil {
		t.Fatalf("Failed to read contacts.yaml: %v", err)
	}

	yamlStr := string(content)

	// Verify existing content is preserved
	if !strings.Contains(yamlStr, "name: Existing Contact") {
		t.Error("Should preserve existing contacts")
	}
	// Old format should be converted to new structured format
	if !strings.Contains(yamlStr, fmt.Sprintf("phone_number: \"%s\"", "5551111111")) {
		t.Error("Should preserve existing unprocessed entries in new format")
	}
	if !strings.Contains(yamlStr, "Previous Entry") {
		t.Error("Should preserve existing unprocessed entry names")
	}

	// Verify new contacts were added in structured format
	if !strings.Contains(yamlStr, fmt.Sprintf("phone_number: \"%s\"", "5551234567")) {
		t.Error("Should add new extracted contacts")
	}
	if !strings.Contains(yamlStr, "John Doe") {
		t.Error("Should add new extracted contacts")
	}
	if !strings.Contains(yamlStr, fmt.Sprintf("phone_number: \"%s\"", "5559876543")) {
		t.Error("Should add new extracted contacts")
	}
	if !strings.Contains(yamlStr, "Jane Smith") {
		t.Error("Should add new extracted contacts")
	}
}

func TestImporter_ContactExtraction_DuplicateNames(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")

	// Create complete repository structure
	setupTestRepository(t, repoRoot)

	// Create test SMS file with duplicate names for same number
	testFile := filepath.Join(tempDir, "sms-test.xml")
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="4">
  <sms address="5551234567" contact_name="John Doe" body="Hello 1" date="1609459200000" type="2" readable_date="Jan 1, 2021 12:00:00 AM" />
  <sms address="5551234567" contact_name="Johnny" body="Hello 2" date="1609459260000" type="1" readable_date="Jan 1, 2021 12:01:00 AM" />
  <sms address="5551234567" contact_name="John Doe" body="Hello 3" date="1609459320000" type="2" readable_date="Jan 1, 2021 12:02:00 AM" />
  <sms address="5551234567" contact_name="J. Doe" body="Hello 4" date="1609459380000" type="1" readable_date="Jan 1, 2021 12:03:00 AM" />
</smses>`

	if err := os.WriteFile(testFile, []byte(testXML), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run import
	options := &ImportOptions{
		RepoRoot: repoRoot,
		Paths:    []string{testFile},
		Filter:   "sms",
		DryRun:   false,
		Quiet:    true,
		Fs:       afero.NewOsFs(),
	}

	importer, err := NewImporter(options, logging.NewNullLogger())
	if err != nil {
		t.Fatalf("Failed to create importer: %v", err)
	}

	_, err = importer.Import()
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Read contacts.yaml and verify content
	contactsPath := filepath.Join(repoRoot, "contacts.yaml")
	content, err := os.ReadFile(contactsPath) // nolint:gosec // Test-controlled path
	if err != nil {
		t.Fatalf("Failed to read contacts.yaml: %v", err)
	}

	yamlStr := string(content)

	// Verify all different names are included in the new structured format
	expectedNames := []string{
		"John Doe",
		"Johnny",
		"J. Doe",
	}

	for _, expected := range expectedNames {
		if !strings.Contains(yamlStr, expected) {
			t.Errorf("Expected to find '%s' in contacts.yaml", expected)
		}
	}

	// Verify phone number appears in structured format
	if !strings.Contains(yamlStr, fmt.Sprintf("phone_number: \"%s\"", "5551234567")) {
		t.Error("Expected to find phone number in structured format")
	}

	// Verify "John Doe" appears only once despite being duplicated in input
	johnDoeCount := strings.Count(yamlStr, "John Doe")
	if johnDoeCount != 1 {
		t.Errorf("Expected 'John Doe' to appear once, found %d times", johnDoeCount)
	}
}

func TestImporter_ContactExtraction_EmptyContactNames(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")

	// Create complete repository structure
	setupTestRepository(t, repoRoot)

	// Create test SMS file with empty/missing contact names
	testFile := filepath.Join(tempDir, "sms-test.xml")
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="3">
  <sms address="5551234567" contact_name="John Doe" body="Has contact name" date="1609459200000" type="2" readable_date="Jan 1, 2021 12:00:00 AM" />
  <sms address="5559876543" contact_name="" body="Empty contact name" date="1609459260000" type="1" readable_date="Jan 1, 2021 12:01:00 AM" />
  <sms address="5551111111" body="No contact name attr" date="1609459320000" type="2" readable_date="Jan 1, 2021 12:02:00 AM" />
</smses>`

	if err := os.WriteFile(testFile, []byte(testXML), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run import
	options := &ImportOptions{
		RepoRoot: repoRoot,
		Paths:    []string{testFile},
		Filter:   "sms",
		DryRun:   false,
		Quiet:    true,
		Fs:       afero.NewOsFs(),
	}

	importer, err := NewImporter(options, logging.NewNullLogger())
	if err != nil {
		t.Fatalf("Failed to create importer: %v", err)
	}

	_, err = importer.Import()
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Read contacts.yaml and verify content
	contactsPath := filepath.Join(repoRoot, "contacts.yaml")
	content, err := os.ReadFile(contactsPath) // nolint:gosec // Test-controlled path
	if err != nil {
		t.Fatalf("Failed to read contacts.yaml: %v", err)
	}

	yamlStr := string(content)

	// Should only have the valid contact in structured format
	if !strings.Contains(yamlStr, fmt.Sprintf("phone_number: \"%s\"", "5551234567")) {
		t.Error("Should extract valid contact phone number")
	}
	if !strings.Contains(yamlStr, "John Doe") {
		t.Error("Should extract valid contact name")
	}

	// Should NOT have empty or missing contact names
	if strings.Contains(yamlStr, fmt.Sprintf("phone_number: \"%s\"", "5559876543")) {
		t.Error("Should not extract empty contact names")
	}
	if strings.Contains(yamlStr, fmt.Sprintf("phone_number: \"%s\"", "5551111111")) {
		t.Error("Should not extract missing contact names")
	}
}

func TestImporter_ContactExtraction_DryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")

	// Create complete repository structure
	setupTestRepository(t, repoRoot)

	// Create test SMS file
	testFile := filepath.Join(tempDir, "sms-test.xml")
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="1">
  <sms address="5551234567" contact_name="John Doe" body="Hello" date="1609459200000" type="2" readable_date="Jan 1, 2021 12:00:00 AM" />
</smses>`

	if err := os.WriteFile(testFile, []byte(testXML), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run import in dry-run mode
	options := &ImportOptions{
		RepoRoot: repoRoot,
		Paths:    []string{testFile},
		Filter:   "sms",
		DryRun:   true,
		Quiet:    true,
		Fs:       afero.NewOsFs(),
	}

	importer, err := NewImporter(options, logging.NewNullLogger())
	if err != nil {
		t.Fatalf("Failed to create importer: %v", err)
	}

	_, err = importer.Import()
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify contacts.yaml was NOT modified in dry-run mode (should stay empty)
	contactsPath := filepath.Join(repoRoot, "contacts.yaml")
	content, err := os.ReadFile(contactsPath) // nolint:gosec // Test-controlled path
	if err != nil {
		t.Fatalf("Failed to read contacts.yaml: %v", err)
	}

	yamlStr := string(content)
	if strings.Contains(yamlStr, "John Doe") || strings.Contains(yamlStr, "unprocessed:") {
		t.Error("contacts.yaml should not be modified in dry-run mode")
	}
}

func TestImporter_ContactExtraction_PhoneNumberNormalization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")

	// Create complete repository structure
	setupTestRepository(t, repoRoot)

	// Create test SMS file with various phone number formats
	testFile := filepath.Join(tempDir, "sms-test.xml")
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="4">
  <sms address="+15551234567" contact_name="John Doe" body="Format 1" date="1609459200000" type="2" readable_date="Jan 1, 2021 12:00:00 AM" />
  <sms address="15551234567" contact_name="John Doe Alt" body="Format 2" date="1609459260000" type="1" readable_date="Jan 1, 2021 12:01:00 AM" />
  <sms address="(555) 123-4567" contact_name="John Doe Formatted" body="Format 3" date="1609459320000" type="2" readable_date="Jan 1, 2021 12:02:00 AM" />
  <sms address="555-123-4567" contact_name="John Doe Dashed" body="Format 4" date="1609459380000" type="1" readable_date="Jan 1, 2021 12:03:00 AM" />
</smses>`

	if err := os.WriteFile(testFile, []byte(testXML), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run import
	options := &ImportOptions{
		RepoRoot: repoRoot,
		Paths:    []string{testFile},
		Filter:   "sms",
		DryRun:   false,
		Quiet:    true,
		Fs:       afero.NewOsFs(),
	}

	importer, err := NewImporter(options, logging.NewNullLogger())
	if err != nil {
		t.Fatalf("Failed to create importer: %v", err)
	}

	_, err = importer.Import()
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Read contacts.yaml and verify content
	contactsPath := filepath.Join(repoRoot, "contacts.yaml")
	content, err := os.ReadFile(contactsPath) // nolint:gosec // Test-controlled path
	if err != nil {
		t.Fatalf("Failed to read contacts.yaml: %v", err)
	}

	yamlStr := string(content)

	// All should be normalized to the same number: 5551234567
	// and all different contact name variations should be preserved
	expectedNames := []string{
		"John Doe",
		"John Doe Alt",
		"John Doe Formatted",
		"John Doe Dashed",
	}

	for _, expected := range expectedNames {
		if !strings.Contains(yamlStr, expected) {
			t.Errorf("Expected to find '%s' in contacts.yaml", expected)
		}
	}

	// Verify normalized phone number appears in structured format
	if !strings.Contains(yamlStr, fmt.Sprintf("phone_number: \"%s\"", "5551234567")) {
		t.Error("Expected to find normalized phone number in structured format")
	}

	// Verify we don't have any other phone number formats in the output
	unwantedFormats := []string{
		"+15551234567",
		"15551234567",
		"(555) 123-4567",
		"555-123-4567",
	}

	for _, unwanted := range unwantedFormats {
		if strings.Contains(yamlStr, fmt.Sprintf("phone_number: \"%s\"", unwanted)) {
			t.Errorf("Should not find unnormalized format '%s' in contacts.yaml", unwanted)
		}
	}
}

// testContactExtraction is a helper function to test contact extraction for different file types
func testContactExtraction(t *testing.T, testXML, fileName, filter string, expectedAdded, expectedCalls int) {
	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")

	// Create complete repository structure
	setupTestRepository(t, repoRoot)

	// Create test file with contact names
	testFile := filepath.Join(tempDir, fileName)
	if err := os.WriteFile(testFile, []byte(testXML), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run import
	options := &ImportOptions{
		RepoRoot: repoRoot,
		Paths:    []string{testFile},
		Filter:   filter,
		DryRun:   false,
		Quiet:    true,
		Fs:       afero.NewOsFs(),
	}

	importer, err := NewImporter(options, logging.NewNullLogger())
	if err != nil {
		t.Fatalf("Failed to create importer: %v", err)
	}

	summary, err := importer.Import()
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify import worked
	if filter == "sms" && summary.SMS.Total.Added != expectedAdded {
		t.Errorf("Expected %d SMS messages imported, got %d", expectedAdded, summary.SMS.Total.Added)
	}
	if filter == "calls" && summary.Calls.Total.Added != expectedCalls {
		t.Errorf("Expected %d call records imported, got %d", expectedCalls, summary.Calls.Total.Added)
	}

	// Check contacts.yaml was created and contains extracted contacts
	contactsPath := filepath.Join(repoRoot, "contacts.yaml")
	if _, err := os.Stat(contactsPath); os.IsNotExist(err) {
		t.Fatal("contacts.yaml should exist after import")
	}

	// Read contacts.yaml and verify content
	content, err := os.ReadFile(contactsPath) // nolint:gosec // Test-controlled path
	if err != nil {
		t.Fatalf("Failed to read contacts.yaml: %v", err)
	}

	yamlStr := string(content)

	// Verify all contacts were extracted
	expectedNumbers := []string{
		"5551234567",
		"5559876543",
		"5551111111",
	}
	expectedNames := []string{
		"John Doe",
		"Jane Smith",
		"Bob Ross",
	}

	for _, number := range expectedNumbers {
		if !strings.Contains(yamlStr, fmt.Sprintf("phone_number: \"%s\"", number)) {
			t.Errorf("Expected to find phone number '%s' in contacts.yaml", number)
		}
	}

	for _, name := range expectedNames {
		if !strings.Contains(yamlStr, name) {
			t.Errorf("Expected to find contact name '%s' in contacts.yaml", name)
		}
	}

	// Verify it has the unprocessed section
	if !strings.Contains(yamlStr, "unprocessed:") {
		t.Error("contacts.yaml should have unprocessed section")
	}
}
