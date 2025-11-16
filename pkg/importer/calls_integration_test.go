package importer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phillipgreenii/mobilecombackup/pkg/calls"
	"github.com/phillipgreenii/mobilecombackup/pkg/contacts"

	"github.com/spf13/afero"
)

func TestCallsImporter_ImportEmptyRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary directory for test
	tmpDir := t.TempDir()
	repoRoot := filepath.Join(tmpDir, "repo")

	// Create test backup file
	testFile := filepath.Join(tmpDir, "calls-test.xml")
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<calls count="3">
  <call number="+15551234567" duration="120" date="1609459200000" type="1" readable_date="Jan 1, 2021 12:00:00 AM" contact_name="John Doe" />
  <call number="+15559876543" duration="60" date="1609545600000" type="2" readable_date="Jan 2, 2021 12:00:00 AM" contact_name="Jane Smith" />
  <call number="+15555555555" duration="0" date="1609632000000" type="3" readable_date="Jan 3, 2021 12:00:00 AM" contact_name="" />
</calls>`

	if err := os.WriteFile(testFile, []byte(testXML), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create importer
	options := &ImportOptions{
		RepoRoot: repoRoot,
		DryRun:   false,
		Fs: afero.NewOsFs(),
	}

	// Create contacts manager for test
	contactsManager := contacts.NewContactsManager(repoRoot)

	yearTracker := NewYearTracker()
	importer, err := NewCallsImporter(options, contactsManager, yearTracker)
	if err != nil {
		t.Fatalf("Failed to create importer: %v", err)
	}

	// Load repository (should be empty)
	if err := importer.LoadRepository(); err != nil {
		t.Fatalf("Failed to load repository: %v", err)
	}

	// Import file
	stat, err := importer.ImportFile(testFile)
	if err != nil {
		t.Fatalf("Failed to import file: %v", err)
	}

	// Check statistics
	if stat.Added != 3 {
		t.Errorf("Expected 3 added, got %d", stat.Added)
	}
	if stat.Duplicates != 0 {
		t.Errorf("Expected 0 duplicates, got %d", stat.Duplicates)
	}
	if stat.Rejected != 0 {
		t.Errorf("Expected 0 rejected, got %d", stat.Rejected)
	}

	// Write repository
	if err := importer.WriteRepository(); err != nil {
		t.Fatalf("Failed to write repository: %v", err)
	}

	// Verify files were created
	expectedFile := filepath.Join(repoRoot, "calls", "calls-2021.xml")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Errorf("Expected file not created: %v", err)
	}

	// Read back and verify
	reader := calls.NewXMLCallsReader(repoRoot)
	readCalls, err := reader.ReadCalls(2021)
	if err != nil {
		t.Fatalf("Failed to read calls: %v", err)
	}

	if len(readCalls) != 3 {
		t.Errorf("Expected 3 calls, got %d", len(readCalls))
	}
}

func TestCallsImporter_DuplicateDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	// Create temporary directory for test
	tmpDir := t.TempDir()
	repoRoot := filepath.Join(tmpDir, "repo")

	// Create existing repository with one call
	callsDir := filepath.Join(repoRoot, "calls")
	if err := os.MkdirAll(callsDir, 0750); err != nil {
		t.Fatalf("Failed to create calls directory: %v", err)
	}

	existingXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<calls count="1">
  <call number="+15551234567" duration="120" date="1609459200000" type="1" readable_date="Jan 1, 2021 12:00:00 AM" contact_name="John Doe" />
</calls>`

	if err := os.WriteFile(filepath.Join(callsDir, "calls-2021.xml"), []byte(existingXML), 0600); err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	// Create test backup file with duplicate and new calls
	testFile := filepath.Join(tmpDir, "calls-test.xml")
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<calls count="3">
  <call number="+15551234567" duration="120" date="1609459200000" type="1" readable_date="Jan 1, 2021 12:00:00 AM EST" contact_name="John" />
  <call number="+15559876543" duration="60" date="1609545600000" type="2" readable_date="Jan 2, 2021 12:00:00 AM" contact_name="Jane Smith" />
  <call number="+15555555555" duration="0" date="1609632000000" type="3" readable_date="Jan 3, 2021 12:00:00 AM" contact_name="" />
</calls>`

	if err := os.WriteFile(testFile, []byte(testXML), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create importer
	options := &ImportOptions{
		RepoRoot: repoRoot,
		DryRun:   false,
		Fs: afero.NewOsFs(),
	}

	// Create contacts manager for test
	contactsManager := contacts.NewContactsManager(repoRoot)

	yearTracker := NewYearTracker()
	importer, err := NewCallsImporter(options, contactsManager, yearTracker)
	if err != nil {
		t.Fatalf("Failed to create importer: %v", err)
	}

	// Load repository
	t.Logf("Loading repository from: %s", repoRoot)
	if err := importer.LoadRepository(); err != nil {
		t.Fatalf("Failed to load repository: %v", err)
	}

	// Check what was loaded
	loadedSummary := importer.GetSummary()
	t.Logf("After LoadRepository: Initial=%d", loadedSummary.Initial)

	// Import file
	stat, err := importer.ImportFile(testFile)
	if err != nil {
		t.Fatalf("Failed to import file: %v", err)
	}

	// Check statistics
	if stat.Added != 2 {
		t.Errorf("Expected 2 added, got %d", stat.Added)
	}
	if stat.Duplicates != 1 {
		t.Errorf("Expected 1 duplicate, got %d", stat.Duplicates)
	}

	// Get summary
	summary := importer.GetSummary()
	t.Logf("Summary: Initial=%d, Final=%d, Added=%d, Duplicates=%d",
		summary.Initial, summary.Final, summary.Added, summary.Duplicates)

	if summary.Initial != 1 {
		t.Errorf("Expected Initial=1, got %d", summary.Initial)
	}
	if summary.Final != 3 {
		t.Errorf("Expected Final=3, got %d", summary.Final)
	}
}

func TestCallsImporter_InvalidEntries(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	// Create temporary directory for test
	tmpDir := t.TempDir()
	repoRoot := filepath.Join(tmpDir, "repo")

	// Create test backup file with invalid entries
	testFile := filepath.Join(tmpDir, "calls-test.xml")
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<calls count="5">
  <call number="+15551234567" duration="120" date="1609459200000" type="1" readable_date="Jan 1, 2021 12:00:00 AM" contact_name="Valid Call" />
  <call number="" duration="60" date="1609545600000" type="2" readable_date="Jan 2, 2021 12:00:00 AM" contact_name="Missing Number" />
  <call number="+15555555555" duration="-10" date="1609632000000" type="3" readable_date="Jan 3, 2021 12:00:00 AM" contact_name="Negative Duration" />
  <call number="+15551111111" duration="30" date="0" type="1" readable_date="" contact_name="Missing Timestamp" />
  <call number="+15552222222" duration="45" date="1609718400000" type="99" readable_date="Jan 4, 2021 12:00:00 AM" contact_name="Invalid Type" />
</calls>`

	if err := os.WriteFile(testFile, []byte(testXML), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create importer
	options := &ImportOptions{
		RepoRoot: repoRoot,
		DryRun:   false,
		Fs: afero.NewOsFs(),
	}

	// Create contacts manager for test
	contactsManager := contacts.NewContactsManager(repoRoot)

	yearTracker := NewYearTracker()
	importer, err := NewCallsImporter(options, contactsManager, yearTracker)
	if err != nil {
		t.Fatalf("Failed to create importer: %v", err)
	}

	// Import file
	stat, err := importer.ImportFile(testFile)
	if err != nil {
		t.Fatalf("Failed to import file: %v", err)
	}

	// Check statistics
	if stat.Added != 1 {
		t.Errorf("Expected 1 added, got %d", stat.Added)
	}
	if stat.Rejected != 4 {
		t.Errorf("Expected 4 rejected, got %d", stat.Rejected)
	}

	// Check rejection files were created in the calls subdirectory
	callsRejectedDir := filepath.Join(repoRoot, "rejected", "calls")
	entries, err := os.ReadDir(callsRejectedDir)
	if err != nil {
		t.Fatalf("Failed to read calls rejected directory: %v", err)
	}

	// Should have 1 rejection file (XML format)
	if len(entries) != 1 {
		t.Errorf("Expected 1 rejection file, got %d", len(entries))
		for _, entry := range entries {
			t.Logf("Found file: %s", entry.Name())
		}
	}

	// Find the rejection XML file
	var rejectionFile string
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".xml" {
			rejectionFile = filepath.Join(callsRejectedDir, entry.Name())
			break
		}
	}

	if rejectionFile == "" {
		t.Fatal("Rejection XML file not found")
	}

	// Read rejection file and verify it contains the rejected entries
	rejectionData, err := os.ReadFile(rejectionFile) // nolint:gosec // Test-controlled path
	if err != nil {
		t.Fatalf("Failed to read rejection file: %v", err)
	}

	rejectionStr := string(rejectionData)

	// Verify the XML contains rejected entries for all invalid calls
	expectedRejections := []string{
		`number=""`,      // Missing Number call
		`duration="-10"`, // Negative Duration call
		`date="0"`,       // Missing Timestamp call
		`type="99"`,      // Invalid Type call
	}

	for _, expected := range expectedRejections {
		if !contains(rejectionStr, expected) {
			t.Errorf("Expected rejection %q not found in rejection file", expected)
		}
	}
}

func TestCallsImporter_OrderPreservation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	// Create temporary directory for test
	tmpDir := t.TempDir()
	repoRoot := filepath.Join(tmpDir, "repo")

	// Create test backup file with calls having same timestamp
	testFile := filepath.Join(tmpDir, "calls-test.xml")
	testXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<calls count="5">
  <call number="+15551111111" duration="10" date="1609459200000" type="1" readable_date="Jan 1, 2021 12:00:00 AM" contact_name="First" />
  <call number="+15552222222" duration="20" date="1609459200000" type="2" readable_date="Jan 1, 2021 12:00:00 AM" contact_name="Second" />
  <call number="+15553333333" duration="30" date="1609459200000" type="1" readable_date="Jan 1, 2021 12:00:00 AM" contact_name="Third" />
  <call number="+15554444444" duration="40" date="1609459200001" type="2" readable_date="Jan 1, 2021 12:00:01 AM" contact_name="Fourth" />
  <call number="+15555555555" duration="50" date="1609459200000" type="3" readable_date="Jan 1, 2021 12:00:00 AM" contact_name="Fifth" />
</calls>`

	if err := os.WriteFile(testFile, []byte(testXML), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create importer
	options := &ImportOptions{
		RepoRoot: repoRoot,
		DryRun:   false,
		Fs: afero.NewOsFs(),
	}

	// Create contacts manager for test
	contactsManager := contacts.NewContactsManager(repoRoot)

	yearTracker := NewYearTracker()
	importer, err := NewCallsImporter(options, contactsManager, yearTracker)
	if err != nil {
		t.Fatalf("Failed to create importer: %v", err)
	}

	// Import and write
	if _, err := importer.ImportFile(testFile); err != nil {
		t.Fatalf("Failed to import file: %v", err)
	}

	if err := importer.WriteRepository(); err != nil {
		t.Fatalf("Failed to write repository: %v", err)
	}

	// Read back and verify order
	reader := calls.NewXMLCallsReader(repoRoot)
	readCalls, err := reader.ReadCalls(2021)
	if err != nil {
		t.Fatalf("Failed to read calls: %v", err)
	}

	if len(readCalls) != 5 {
		t.Fatalf("Expected 5 calls, got %d", len(readCalls))
	}

	// Verify the calls with same timestamp appear in some order
	// (exact order is not guaranteed but they should be grouped together)
	sameTimestampCount := 0
	for i, call := range readCalls {
		if call.Date == 1609459200000 {
			sameTimestampCount++
			if i >= 4 {
				t.Error("Calls with same timestamp should appear before the one with later timestamp")
			}
		}
	}

	if sameTimestampCount != 4 {
		t.Errorf("Expected 4 calls with same timestamp, got %d", sameTimestampCount)
	}

	// The fourth call (with timestamp+1) should be last
	if readCalls[4].Number != "+15554444444" {
		t.Error("Call with later timestamp should be last")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[0:len(substr)] == substr || contains(s[1:], substr)))
}
