package manifest

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManifestGenerator_GenerateFileManifest(t *testing.T) {
	// Create temporary repository
	tempDir := t.TempDir()

	// Create test structure
	createTestRepository(t, tempDir)

	generator := NewManifestGenerator(tempDir)
	manifest, err := generator.GenerateFileManifest()
	if err != nil {
		t.Fatalf("GenerateFileManifest failed: %v", err)
	}

	// Verify expected files are included
	expectedFiles := map[string]bool{
		".mobilecombackup.yaml": false,
		"contacts.yaml":         false,
		"summary.yaml":          false,
		"calls/calls-2023.xml":  false,
		"sms/sms-2023.xml":      false,
	}

	for _, entry := range manifest.Files {
		if _, expected := expectedFiles[entry.Name]; expected {
			expectedFiles[entry.Name] = true

			// Verify entry has required fields
			if entry.Checksum == "" || !strings.HasPrefix(entry.Checksum, "sha256:") {
				t.Errorf("File %s missing or invalid checksum: %s", entry.Name, entry.Checksum)
			}
			if entry.Size <= 0 {
				t.Errorf("File %s has invalid size: %d", entry.Name, entry.Size)
			}
		}
	}

	// Check that all expected files were found
	for file, found := range expectedFiles {
		if !found {
			t.Errorf("Expected file %s not found in manifest", file)
		}
	}

	// Verify excluded files are not included
	excludedFiles := []string{"files.yaml", "files.yaml.sha256", "rejected/rejected-calls.xml", ".hidden-file"}
	for _, entry := range manifest.Files {
		for _, excluded := range excludedFiles {
			if entry.Name == excluded {
				t.Errorf("Excluded file %s found in manifest", excluded)
			}
		}
	}
}

func TestManifestGenerator_CrossPlatformPaths(t *testing.T) {
	tempDir := t.TempDir()

	// Create nested directory structure
	nestedDir := filepath.Join(tempDir, "calls", "2023")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test file
	testFile := filepath.Join(nestedDir, "calls-jan.xml")
	if err := os.WriteFile(testFile, []byte("<calls></calls>"), 0644); err != nil {
		t.Fatal(err)
	}

	generator := NewManifestGenerator(tempDir)
	manifest, err := generator.GenerateFileManifest()
	if err != nil {
		t.Fatalf("GenerateFileManifest failed: %v", err)
	}

	// Find our test file in manifest
	var foundEntry *FileEntry
	for _, entry := range manifest.Files {
		if strings.Contains(entry.Name, "calls-jan.xml") {
			foundEntry = &entry
			break
		}
	}

	if foundEntry == nil {
		t.Fatal("Test file not found in manifest")
	}

	// Verify path uses forward slashes (cross-platform)
	expected := "calls/2023/calls-jan.xml"
	if foundEntry.Name != expected {
		t.Errorf("Expected path %s, got %s", expected, foundEntry.Name)
	}

	// Verify no backslashes (Windows compatibility)
	if strings.Contains(foundEntry.Name, "\\") {
		t.Errorf("Path contains backslashes: %s", foundEntry.Name)
	}
}

func TestManifestGenerator_WriteManifestFiles(t *testing.T) {
	tempDir := t.TempDir()
	createTestRepository(t, tempDir)

	generator := NewManifestGenerator(tempDir)
	manifest, err := generator.GenerateFileManifest()
	if err != nil {
		t.Fatal(err)
	}

	// Write manifest files
	if err := generator.WriteManifestFiles(manifest); err != nil {
		t.Fatalf("WriteManifestFiles failed: %v", err)
	}

	// Verify files.yaml exists and is valid
	manifestPath := filepath.Join(tempDir, "files.yaml")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Errorf("files.yaml not created: %v", err)
	}

	// Verify files.yaml.sha256 exists
	checksumPath := filepath.Join(tempDir, "files.yaml.sha256")
	if _, err := os.Stat(checksumPath); err != nil {
		t.Errorf("files.yaml.sha256 not created: %v", err)
	}

	// Verify checksum is correct
	expectedHash, err := calculateFileHash(manifestPath)
	if err != nil {
		t.Fatal(err)
	}

	checksumData, err := os.ReadFile(checksumPath)
	if err != nil {
		t.Fatal(err)
	}

	actualLine := strings.TrimSpace(string(checksumData))

	// Parse "hash  filename" format
	parts := strings.Fields(actualLine)
	if len(parts) != 2 {
		t.Errorf("Invalid checksum format: expected 'hash  filename', got %s", actualLine)
		return
	}

	actualHash := parts[0]
	actualFilename := parts[1]

	if actualHash != expectedHash {
		t.Errorf("Checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	if actualFilename != "files.yaml" {
		t.Errorf("Expected filename 'files.yaml', got %s", actualFilename)
	}
}

func TestManifestGenerator_WriteChecksumOnly(t *testing.T) {
	tempDir := t.TempDir()
	createTestRepository(t, tempDir)

	generator := NewManifestGenerator(tempDir)
	manifest, err := generator.GenerateFileManifest()
	if err != nil {
		t.Fatal(err)
	}

	// Write only the manifest first
	if err := generator.WriteManifestOnly(manifest); err != nil {
		t.Fatal(err)
	}

	checksumPath := filepath.Join(tempDir, "files.yaml.sha256")

	// Verify checksum doesn't exist yet
	if _, err := os.Stat(checksumPath); err == nil {
		t.Error("Checksum file should not exist yet")
	}

	// Write checksum only
	if err := generator.WriteChecksumOnly(); err != nil {
		t.Fatal(err)
	}

	// Verify checksum now exists
	if _, err := os.Stat(checksumPath); err != nil {
		t.Errorf("Checksum file not created: %v", err)
	}

	// Try to write checksum again - should not overwrite
	originalContent, err := os.ReadFile(checksumPath)
	if err != nil {
		t.Fatal(err)
	}

	// Modify the checksum file
	if err := os.WriteFile(checksumPath, []byte("modified"), 0644); err != nil {
		t.Fatal(err)
	}

	// WriteChecksumOnly should not overwrite existing file
	if err := generator.WriteChecksumOnly(); err != nil {
		t.Fatal(err)
	}

	modifiedContent, err := os.ReadFile(checksumPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(modifiedContent) == string(originalContent) {
		t.Error("Checksum file was overwritten when it shouldn't have been")
	}
}

func TestManifestGenerator_ShouldSkipFile(t *testing.T) {
	generator := NewManifestGenerator("/test")

	testCases := []struct {
		file     string
		expected bool
		reason   string
	}{
		{"files.yaml", true, "should skip files.yaml itself"},
		{"files.yaml.sha256", true, "should skip checksum file"},
		{".mobilecombackup.yaml", false, "should include marker file"},
		{"contacts.yaml", false, "should include contacts file"},
		{"rejected/calls.xml", true, "should skip rejected files"},
		{"rejected/sms/messages.xml", true, "should skip nested rejected files"},
		{"calls/calls-2023.xml", false, "should include normal call files"},
		{".hidden-file", true, "should skip hidden files"},
		{".DS_Store", true, "should skip system hidden files"},
		{"temp.tmp", true, "should skip temporary files"},
		{"data.tmp", true, "should skip all .tmp files"},
	}

	for _, tc := range testCases {
		result := generator.shouldSkipFile(tc.file)
		if result != tc.expected {
			t.Errorf("shouldSkipFile(%q): expected %v, got %v (%s)", tc.file, tc.expected, result, tc.reason)
		}
	}
}

func TestCalculateFileHash(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!"

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	hash, err := calculateFileHash(testFile)
	if err != nil {
		t.Fatalf("calculateFileHash failed: %v", err)
	}

	// Calculate expected hash
	expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(testContent)))

	if hash != expectedHash {
		t.Errorf("Hash mismatch: expected %s, got %s", expectedHash, hash)
	}
}

// Helper function to create a test repository structure
func createTestRepository(t *testing.T, tempDir string) {
	// Create directories
	dirs := []string{"calls", "sms", "attachments", "rejected"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tempDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create test files
	files := map[string]string{
		".mobilecombackup.yaml":       "repository_structure_version: '1'\n",
		"contacts.yaml":               "contacts: []\n",
		"summary.yaml":                "counts:\n  calls: 0\n  sms: 0\n",
		"calls/calls-2023.xml":        "<calls count='0'></calls>",
		"sms/sms-2023.xml":            "<sms count='0'></sms>",
		"rejected/rejected-calls.xml": "<rejected></rejected>",
		".hidden-file":                "hidden content",
	}

	for file, content := range files {
		path := filepath.Join(tempDir, file)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}
