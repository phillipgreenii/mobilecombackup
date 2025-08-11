package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGenerateManifestFile(t *testing.T) {
	// Create temp directory with test files
	tempDir := t.TempDir()

	// Create test repository structure
	dirs := []string{
		filepath.Join(tempDir, "calls"),
		filepath.Join(tempDir, "sms"),
		filepath.Join(tempDir, "attachments", "ab"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create test files
	testFiles := map[string]string{
		".mobilecombackup.yaml":  "version: 1.0\ncreated: 2025-08-09T10:00:00Z\n",
		"summary.yaml":           "last_updated: 2025-08-09T10:00:00Z\n",
		"calls/calls-2014.xml":   "<?xml version='1.0'?><calls count=\"2\"></calls>",
		"sms/sms-2014.xml":       "<?xml version='1.0'?><smses count=\"3\"></smses>",
		"attachments/ab/abc123":  "test attachment data",
		"contacts/contacts.yaml": "contacts:\n  - name: Test\n    phones: [\"5551234567\"]\n",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Generate manifest
	if err := generateManifestFile(tempDir, "test-v1.0"); err != nil {
		t.Fatalf("Failed to generate manifest: %v", err)
	}

	// Verify files.yaml exists
	manifestPath := filepath.Join(tempDir, "files.yaml")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("Manifest file not created: %v", err)
	}

	// Verify files.yaml.sha256 exists
	checksumPath := filepath.Join(tempDir, "files.yaml.sha256")
	if _, err := os.Stat(checksumPath); err != nil {
		t.Fatalf("Checksum file not created: %v", err)
	}

	// Read and parse manifest
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("Failed to read manifest: %v", err)
	}

	var manifest FileManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("Failed to parse manifest: %v", err)
	}

	// Verify manifest metadata
	if manifest.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", manifest.Version)
	}

	if manifest.Generator != "mobilecombackup test-v1.0" {
		t.Errorf("Expected generator 'mobilecombackup test-v1.0', got %s", manifest.Generator)
	}

	if manifest.Generated == "" {
		t.Error("Generated timestamp should not be empty")
	}

	// Verify all test files are in manifest
	expectedFiles := []string{
		".mobilecombackup.yaml",
		"summary.yaml",
		"calls/calls-2014.xml",
		"sms/sms-2014.xml",
		"attachments/ab/abc123",
		"contacts/contacts.yaml",
	}

	if len(manifest.Files) != len(expectedFiles) {
		t.Errorf("Expected %d files, got %d", len(expectedFiles), len(manifest.Files))
	}

	// Check each file
	fileMap := make(map[string]FileEntry)
	for _, entry := range manifest.Files {
		fileMap[entry.Name] = entry
	}

	for _, expectedFile := range expectedFiles {
		entry, exists := fileMap[expectedFile]
		if !exists {
			t.Errorf("Expected file %s not found in manifest", expectedFile)
			continue
		}

		// Verify checksum format
		if !strings.HasPrefix(entry.Checksum, "sha256:") {
			t.Errorf("File %s checksum should start with 'sha256:', got %s",
				expectedFile, entry.Checksum)
		}

		// Verify size
		if entry.Size == 0 {
			t.Errorf("File %s should have non-zero size", expectedFile)
		}

		// Verify modified timestamp
		if entry.Modified == "" {
			t.Errorf("File %s should have modified timestamp", expectedFile)
		}
	}

	// Verify files.yaml and files.yaml.sha256 are NOT in manifest
	for _, entry := range manifest.Files {
		if entry.Name == "files.yaml" || entry.Name == "files.yaml.sha256" {
			t.Errorf("File %s should not be in manifest", entry.Name)
		}
	}
}

func TestCalculateFileChecksum(t *testing.T) {
	// Create temp file with known content
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!"

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Calculate checksum
	checksum, err := calculateFileChecksum(testFile)
	if err != nil {
		t.Fatalf("Failed to calculate checksum: %v", err)
	}

	// Expected SHA-256 of "Hello, World!"
	expected := "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f"

	if checksum != expected {
		t.Errorf("Expected checksum %s, got %s", expected, checksum)
	}
}

func TestCollectRepositoryFiles(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create nested directory structure
	dirs := []string{
		filepath.Join(tempDir, "a", "b"),
		filepath.Join(tempDir, "c"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create files
	files := []string{
		"file1.txt",
		"a/file2.txt",
		"a/b/file3.txt",
		"c/file4.txt",
		"files.yaml",        // Should be excluded
		"files.yaml.sha256", // Should be excluded
	}

	for _, file := range files {
		path := filepath.Join(tempDir, file)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Collect files
	entries, err := collectRepositoryFiles(tempDir)
	if err != nil {
		t.Fatalf("Failed to collect files: %v", err)
	}

	// Should have 4 files (excluding files.yaml and files.yaml.sha256)
	if len(entries) != 4 {
		t.Errorf("Expected 4 files, got %d", len(entries))
	}

	// Verify files are sorted
	for i := 1; i < len(entries); i++ {
		if entries[i-1].Name >= entries[i].Name {
			t.Error("Files should be sorted by name")
		}
	}

	// Verify excluded files are not present
	for _, entry := range entries {
		if entry.Name == "files.yaml" || entry.Name == "files.yaml.sha256" {
			t.Errorf("File %s should be excluded", entry.Name)
		}
	}
}

func TestVerifyManifest(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"file1.txt": "content1",
		"file2.txt": "content2",
	}

	for name, content := range testFiles {
		path := filepath.Join(tempDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Generate manifest
	if err := generateManifestFile(tempDir, "test"); err != nil {
		t.Fatalf("Failed to generate manifest: %v", err)
	}

	// Verify should pass
	if err := verifyManifest(tempDir); err != nil {
		t.Errorf("Verify should pass for valid manifest: %v", err)
	}

	// Modify a file
	if err := os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("modified"), 0644); err != nil {
		t.Fatal(err)
	}

	// Verify should fail
	if err := verifyManifest(tempDir); err == nil {
		t.Error("Verify should fail when file is modified")
	}
}

func TestGenerateManifestChecksum(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "files.yaml")

	// Create a test manifest file
	testContent := "test manifest content"
	if err := os.WriteFile(manifestPath, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Generate checksum file
	if err := generateManifestChecksum(manifestPath); err != nil {
		t.Fatalf("Failed to generate checksum: %v", err)
	}

	// Read checksum file
	checksumPath := manifestPath + ".sha256"
	data, err := os.ReadFile(checksumPath)
	if err != nil {
		t.Fatalf("Failed to read checksum file: %v", err)
	}

	// Verify format: "checksum  filename\n"
	content := string(data)
	if !strings.HasSuffix(content, "  files.yaml\n") {
		t.Errorf("Checksum file should end with '  files.yaml\\n', got: %q", content)
	}

	// Verify checksum length (64 hex chars)
	parts := strings.Fields(content)
	if len(parts) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(parts))
	}

	if len(parts[0]) != 64 {
		t.Errorf("Expected 64-char checksum, got %d chars", len(parts[0]))
	}
}
