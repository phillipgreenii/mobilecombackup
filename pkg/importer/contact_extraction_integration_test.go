package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImporter_ContactExtraction_SMS(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")

	// Create repository structure
	if err := os.MkdirAll(filepath.Join(repoRoot, "sms"), 0755); err != nil {
		t.Fatalf("Failed to create repo structure: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, "calls"), 0755); err != nil {
		t.Fatalf("Failed to create repo structure: %v", err)
	}

	// Create test SMS file with contact names
	testFile := filepath.Join(tempDir, "sms-test.xml")
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="3">
  <sms address="5551234567" contact_name="John Doe" body="Hello" date="1609459200000" type="2" readable_date="Jan 1, 2021 12:00:00 AM" />
  <sms address="5559876543" contact_name="Jane Smith" body="Hi there" date="1609459260000" type="1" readable_date="Jan 1, 2021 12:01:00 AM" />
  <sms address="5551111111" contact_name="Bob Ross" body="Happy trees" date="1609459320000" type="2" readable_date="Jan 1, 2021 12:02:00 AM" />
</smses>`

	if err := os.WriteFile(testFile, []byte(testXML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run import
	options := &ImportOptions{
		RepoRoot: repoRoot,
		Paths:    []string{testFile},
		Filter:   "sms",
		DryRun:   false,
		Quiet:    true,
	}

	importer, err := NewImporter(options)
	if err != nil {
		t.Fatalf("Failed to create importer: %v", err)
	}

	summary, err := importer.Import()
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify import worked
	if summary.SMS.Total.Added != 3 {
		t.Errorf("Expected 3 SMS messages imported, got %d", summary.SMS.Total.Added)
	}

	// Check contacts.yaml was created and contains extracted contacts
	contactsPath := filepath.Join(repoRoot, "contacts.yaml")
	if _, err := os.Stat(contactsPath); os.IsNotExist(err) {
		t.Fatal("contacts.yaml should exist after import")
	}

	// Read contacts.yaml and verify content
	content, err := os.ReadFile(contactsPath)
	if err != nil {
		t.Fatalf("Failed to read contacts.yaml: %v", err)
	}

	yamlStr := string(content)

	// Verify all contacts were extracted
	expectedContacts := []string{
		"5551234567: John Doe",
		"5559876543: Jane Smith",
		"5551111111: Bob Ross",
	}

	for _, expected := range expectedContacts {
		if !strings.Contains(yamlStr, expected) {
			t.Errorf("Expected to find '%s' in contacts.yaml", expected)
		}
	}

	// Verify it has the unprocessed section
	if !strings.Contains(yamlStr, "unprocessed:") {
		t.Error("contacts.yaml should have unprocessed section")
	}
}

func TestImporter_ContactExtraction_Calls(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")

	// Create repository structure
	if err := os.MkdirAll(filepath.Join(repoRoot, "sms"), 0755); err != nil {
		t.Fatalf("Failed to create repo structure: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, "calls"), 0755); err != nil {
		t.Fatalf("Failed to create repo structure: %v", err)
	}

	// Create test calls file with contact names
	testFile := filepath.Join(tempDir, "calls-test.xml")
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<calls count="3">
  <call number="5551234567" contact_name="John Doe" duration="120" date="1609459200000" type="1" readable_date="Jan 1, 2021 12:00:00 AM" />
  <call number="5559876543" contact_name="Jane Smith" duration="60" date="1609459260000" type="2" readable_date="Jan 1, 2021 12:01:00 AM" />
  <call number="5551111111" contact_name="Bob Ross" duration="300" date="1609459320000" type="3" readable_date="Jan 1, 2021 12:02:00 AM" />
</calls>`

	if err := os.WriteFile(testFile, []byte(testXML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run import
	options := &ImportOptions{
		RepoRoot: repoRoot,
		Paths:    []string{testFile},
		Filter:   "calls",
		DryRun:   false,
		Quiet:    true,
	}

	importer, err := NewImporter(options)
	if err != nil {
		t.Fatalf("Failed to create importer: %v", err)
	}

	summary, err := importer.Import()
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify import worked
	if summary.Calls.Total.Added != 3 {
		t.Errorf("Expected 3 call records imported, got %d", summary.Calls.Total.Added)
	}

	// Check contacts.yaml was created and contains extracted contacts
	contactsPath := filepath.Join(repoRoot, "contacts.yaml")
	if _, err := os.Stat(contactsPath); os.IsNotExist(err) {
		t.Fatal("contacts.yaml should exist after import")
	}

	// Read contacts.yaml and verify content
	content, err := os.ReadFile(contactsPath)
	if err != nil {
		t.Fatalf("Failed to read contacts.yaml: %v", err)
	}

	yamlStr := string(content)

	// Verify all contacts were extracted
	expectedContacts := []string{
		"5551234567: John Doe",
		"5559876543: Jane Smith",
		"5551111111: Bob Ross",
	}

	for _, expected := range expectedContacts {
		if !strings.Contains(yamlStr, expected) {
			t.Errorf("Expected to find '%s' in contacts.yaml", expected)
		}
	}
}

func TestImporter_ContactExtraction_MMS(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")

	// Create repository structure
	if err := os.MkdirAll(filepath.Join(repoRoot, "sms"), 0755); err != nil {
		t.Fatalf("Failed to create repo structure: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, "calls"), 0755); err != nil {
		t.Fatalf("Failed to create repo structure: %v", err)
	}

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

	if err := os.WriteFile(testFile, []byte(testXML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run import
	options := &ImportOptions{
		RepoRoot: repoRoot,
		Paths:    []string{testFile},
		Filter:   "sms",
		DryRun:   false,
		Quiet:    true,
	}

	importer, err := NewImporter(options)
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
	content, err := os.ReadFile(contactsPath)
	if err != nil {
		t.Fatalf("Failed to read contacts.yaml: %v", err)
	}

	yamlStr := string(content)

	// Verify only primary addresses were extracted (MMS additional addresses don't have contact names)
	expectedContacts := []string{
		"5551234567: John Doe",
		"5555555555: Jane Smith",
	}

	for _, expected := range expectedContacts {
		if !strings.Contains(yamlStr, expected) {
			t.Errorf("Expected to find '%s' in contacts.yaml", expected)
		}
	}

	// Should NOT contain the additional address since it doesn't have a contact name
	if strings.Contains(yamlStr, "5559876543:") {
		t.Error("Should not extract contacts from MMS additional addresses without contact names")
	}
}

func TestImporter_ContactExtraction_ExistingContacts(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")

	// Create repository structure
	if err := os.MkdirAll(filepath.Join(repoRoot, "sms"), 0755); err != nil {
		t.Fatalf("Failed to create repo structure: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, "calls"), 0755); err != nil {
		t.Fatalf("Failed to create repo structure: %v", err)
	}

	// Create existing contacts.yaml
	contactsPath := filepath.Join(repoRoot, "contacts.yaml")
	existingYaml := `contacts:
  - name: "Existing Contact"
    numbers:
      - "5550000000"
unprocessed:
  - "5551111111: Previous Entry"
`
	if err := os.WriteFile(contactsPath, []byte(existingYaml), 0644); err != nil {
		t.Fatalf("Failed to create existing contacts.yaml: %v", err)
	}

	// Create test SMS file
	testFile := filepath.Join(tempDir, "sms-test.xml")
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="2">
  <sms address="5551234567" contact_name="John Doe" body="Hello" date="1609459200000" type="2" readable_date="Jan 1, 2021 12:00:00 AM" />
  <sms address="5559876543" contact_name="Jane Smith" body="Hi there" date="1609459260000" type="1" readable_date="Jan 1, 2021 12:01:00 AM" />
</smses>`

	if err := os.WriteFile(testFile, []byte(testXML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run import
	options := &ImportOptions{
		RepoRoot: repoRoot,
		Paths:    []string{testFile},
		Filter:   "sms",
		DryRun:   false,
		Quiet:    true,
	}

	importer, err := NewImporter(options)
	if err != nil {
		t.Fatalf("Failed to create importer: %v", err)
	}

	_, err = importer.Import()
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Read updated contacts.yaml and verify content
	content, err := os.ReadFile(contactsPath)
	if err != nil {
		t.Fatalf("Failed to read contacts.yaml: %v", err)
	}

	yamlStr := string(content)

	// Verify existing content is preserved
	if !strings.Contains(yamlStr, "name: Existing Contact") {
		t.Error("Should preserve existing contacts")
	}
	if !strings.Contains(yamlStr, "5551111111: Previous Entry") {
		t.Error("Should preserve existing unprocessed entries")
	}

	// Verify new contacts were added
	if !strings.Contains(yamlStr, "5551234567: John Doe") {
		t.Error("Should add new extracted contacts")
	}
	if !strings.Contains(yamlStr, "5559876543: Jane Smith") {
		t.Error("Should add new extracted contacts")
	}
}

func TestImporter_ContactExtraction_DuplicateNames(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")

	// Create repository structure
	if err := os.MkdirAll(filepath.Join(repoRoot, "sms"), 0755); err != nil {
		t.Fatalf("Failed to create repo structure: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, "calls"), 0755); err != nil {
		t.Fatalf("Failed to create repo structure: %v", err)
	}

	// Create test SMS file with duplicate names for same number
	testFile := filepath.Join(tempDir, "sms-test.xml")
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="4">
  <sms address="5551234567" contact_name="John Doe" body="Hello 1" date="1609459200000" type="2" readable_date="Jan 1, 2021 12:00:00 AM" />
  <sms address="5551234567" contact_name="Johnny" body="Hello 2" date="1609459260000" type="1" readable_date="Jan 1, 2021 12:01:00 AM" />
  <sms address="5551234567" contact_name="John Doe" body="Hello 3" date="1609459320000" type="2" readable_date="Jan 1, 2021 12:02:00 AM" />
  <sms address="5551234567" contact_name="J. Doe" body="Hello 4" date="1609459380000" type="1" readable_date="Jan 1, 2021 12:03:00 AM" />
</smses>`

	if err := os.WriteFile(testFile, []byte(testXML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run import
	options := &ImportOptions{
		RepoRoot: repoRoot,
		Paths:    []string{testFile},
		Filter:   "sms",
		DryRun:   false,
		Quiet:    true,
	}

	importer, err := NewImporter(options)
	if err != nil {
		t.Fatalf("Failed to create importer: %v", err)
	}

	_, err = importer.Import()
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Read contacts.yaml and verify content
	contactsPath := filepath.Join(repoRoot, "contacts.yaml")
	content, err := os.ReadFile(contactsPath)
	if err != nil {
		t.Fatalf("Failed to read contacts.yaml: %v", err)
	}

	yamlStr := string(content)

	// Verify all different names are included
	expectedNames := []string{
		"5551234567: John Doe",
		"5551234567: Johnny",
		"5551234567: J. Doe",
	}

	for _, expected := range expectedNames {
		if !strings.Contains(yamlStr, expected) {
			t.Errorf("Expected to find '%s' in contacts.yaml", expected)
		}
	}

	// Verify "John Doe" appears only once despite being duplicated in input
	johnDoeCount := strings.Count(yamlStr, "5551234567: John Doe")
	if johnDoeCount != 1 {
		t.Errorf("Expected 'John Doe' to appear once, found %d times", johnDoeCount)
	}
}

func TestImporter_ContactExtraction_EmptyContactNames(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")

	// Create repository structure
	if err := os.MkdirAll(filepath.Join(repoRoot, "sms"), 0755); err != nil {
		t.Fatalf("Failed to create repo structure: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, "calls"), 0755); err != nil {
		t.Fatalf("Failed to create repo structure: %v", err)
	}

	// Create test SMS file with empty/missing contact names
	testFile := filepath.Join(tempDir, "sms-test.xml")
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="3">
  <sms address="5551234567" contact_name="John Doe" body="Has contact name" date="1609459200000" type="2" readable_date="Jan 1, 2021 12:00:00 AM" />
  <sms address="5559876543" contact_name="" body="Empty contact name" date="1609459260000" type="1" readable_date="Jan 1, 2021 12:01:00 AM" />
  <sms address="5551111111" body="No contact name attr" date="1609459320000" type="2" readable_date="Jan 1, 2021 12:02:00 AM" />
</smses>`

	if err := os.WriteFile(testFile, []byte(testXML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run import
	options := &ImportOptions{
		RepoRoot: repoRoot,
		Paths:    []string{testFile},
		Filter:   "sms",
		DryRun:   false,
		Quiet:    true,
	}

	importer, err := NewImporter(options)
	if err != nil {
		t.Fatalf("Failed to create importer: %v", err)
	}

	_, err = importer.Import()
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Read contacts.yaml and verify content
	contactsPath := filepath.Join(repoRoot, "contacts.yaml")
	content, err := os.ReadFile(contactsPath)
	if err != nil {
		t.Fatalf("Failed to read contacts.yaml: %v", err)
	}

	yamlStr := string(content)

	// Should only have the valid contact
	if !strings.Contains(yamlStr, "5551234567: John Doe") {
		t.Error("Should extract valid contact name")
	}

	// Should NOT have empty or missing contact names
	if strings.Contains(yamlStr, "5559876543:") {
		t.Error("Should not extract empty contact names")
	}
	if strings.Contains(yamlStr, "5551111111:") {
		t.Error("Should not extract missing contact names")
	}
}

func TestImporter_ContactExtraction_DryRun(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")

	// Create repository structure
	if err := os.MkdirAll(filepath.Join(repoRoot, "sms"), 0755); err != nil {
		t.Fatalf("Failed to create repo structure: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, "calls"), 0755); err != nil {
		t.Fatalf("Failed to create repo structure: %v", err)
	}

	// Create test SMS file
	testFile := filepath.Join(tempDir, "sms-test.xml")
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="1">
  <sms address="5551234567" contact_name="John Doe" body="Hello" date="1609459200000" type="2" readable_date="Jan 1, 2021 12:00:00 AM" />
</smses>`

	if err := os.WriteFile(testFile, []byte(testXML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run import in dry-run mode
	options := &ImportOptions{
		RepoRoot: repoRoot,
		Paths:    []string{testFile},
		Filter:   "sms",
		DryRun:   true,
		Quiet:    true,
	}

	importer, err := NewImporter(options)
	if err != nil {
		t.Fatalf("Failed to create importer: %v", err)
	}

	_, err = importer.Import()
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify contacts.yaml was NOT created in dry-run mode
	contactsPath := filepath.Join(repoRoot, "contacts.yaml")
	if _, err := os.Stat(contactsPath); !os.IsNotExist(err) {
		t.Error("contacts.yaml should not be created in dry-run mode")
	}
}

func TestImporter_ContactExtraction_PhoneNumberNormalization(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")

	// Create repository structure
	if err := os.MkdirAll(filepath.Join(repoRoot, "sms"), 0755); err != nil {
		t.Fatalf("Failed to create repo structure: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, "calls"), 0755); err != nil {
		t.Fatalf("Failed to create repo structure: %v", err)
	}

	// Create test SMS file with various phone number formats
	testFile := filepath.Join(tempDir, "sms-test.xml")
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="4">
  <sms address="+15551234567" contact_name="John Doe" body="Format 1" date="1609459200000" type="2" readable_date="Jan 1, 2021 12:00:00 AM" />
  <sms address="15551234567" contact_name="John Doe Alt" body="Format 2" date="1609459260000" type="1" readable_date="Jan 1, 2021 12:01:00 AM" />
  <sms address="(555) 123-4567" contact_name="John Doe Formatted" body="Format 3" date="1609459320000" type="2" readable_date="Jan 1, 2021 12:02:00 AM" />
  <sms address="555-123-4567" contact_name="John Doe Dashed" body="Format 4" date="1609459380000" type="1" readable_date="Jan 1, 2021 12:03:00 AM" />
</smses>`

	if err := os.WriteFile(testFile, []byte(testXML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run import
	options := &ImportOptions{
		RepoRoot: repoRoot,
		Paths:    []string{testFile},
		Filter:   "sms",
		DryRun:   false,
		Quiet:    true,
	}

	importer, err := NewImporter(options)
	if err != nil {
		t.Fatalf("Failed to create importer: %v", err)
	}

	_, err = importer.Import()
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Read contacts.yaml and verify content
	contactsPath := filepath.Join(repoRoot, "contacts.yaml")
	content, err := os.ReadFile(contactsPath)
	if err != nil {
		t.Fatalf("Failed to read contacts.yaml: %v", err)
	}

	yamlStr := string(content)

	// All should be normalized to the same number: 5551234567
	// and all different contact name variations should be preserved
	expectedNames := []string{
		"5551234567: John Doe",
		"5551234567: John Doe Alt",
		"5551234567: John Doe Formatted",
		"5551234567: John Doe Dashed",
	}

	for _, expected := range expectedNames {
		if !strings.Contains(yamlStr, expected) {
			t.Errorf("Expected to find '%s' in contacts.yaml", expected)
		}
	}

	// Verify we don't have any other phone number formats in the output
	unwantedFormats := []string{
		"+15551234567:",
		"15551234567:",
		"(555) 123-4567:",
		"555-123-4567:",
	}

	for _, unwanted := range unwantedFormats {
		if strings.Contains(yamlStr, unwanted) {
			t.Errorf("Should not find unnormalized format '%s' in contacts.yaml", unwanted)
		}
	}
}
