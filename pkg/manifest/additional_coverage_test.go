package manifest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

// Tests for WriteManifestFiles function

func TestGenerator_WriteManifestFiles(t *testing.T) {
	t.Parallel()

	t.Run("writes both manifest and checksum", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create test files
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test content"), 0600); err != nil {
			t.Fatal(err)
		}

		generator := NewManifestGenerator(tmpDir, afero.NewOsFs())

		// Generate manifest
		manifest, err := generator.GenerateFileManifest()
		if err != nil {
			t.Fatalf("Failed to generate manifest: %v", err)
		}

		// Write manifest files
		err = generator.WriteManifestFiles(manifest)
		if err != nil {
			t.Fatalf("WriteManifestFiles failed: %v", err)
		}

		// Verify files.yaml exists
		manifestPath := filepath.Join(tmpDir, "files.yaml")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			t.Error("files.yaml was not created")
		}

		// Verify files.yaml.sha256 exists
		checksumPath := filepath.Join(tmpDir, "files.yaml.sha256")
		if _, err := os.Stat(checksumPath); os.IsNotExist(err) {
			t.Error("files.yaml.sha256 was not created")
		}
	})

}

// Tests for hash calculation via GenerateFileManifest

func TestGenerator_HashCalculation(t *testing.T) {
	t.Parallel()

	t.Run("calculates correct hash for files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create test file with known content
		testFile := filepath.Join(tmpDir, "test.txt")
		content := "Hello, World!"
		if err := os.WriteFile(testFile, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}

		generator := NewManifestGenerator(tmpDir, afero.NewOsFs())

		// Generate manifest (which calculates hashes)
		manifest, err := generator.GenerateFileManifest()
		if err != nil {
			t.Fatalf("GenerateFileManifest failed: %v", err)
		}

		// Find our test file
		var found bool
		expectedHash := "sha256:dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f"
		for _, entry := range manifest.Files {
			if entry.Name == "test.txt" {
				found = true
				if entry.Checksum != expectedHash {
					t.Errorf("Expected hash %s, got %s", expectedHash, entry.Checksum)
				}
			}
		}
		if !found {
			t.Error("test.txt not found in manifest")
		}
	})

	t.Run("handles empty file", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create empty file
		testFile := filepath.Join(tmpDir, "empty.txt")
		if err := os.WriteFile(testFile, []byte(""), 0600); err != nil {
			t.Fatal(err)
		}

		generator := NewManifestGenerator(tmpDir, afero.NewOsFs())

		// Generate manifest
		manifest, err := generator.GenerateFileManifest()
		if err != nil {
			t.Fatalf("GenerateFileManifest failed: %v", err)
		}

		// SHA-256 of empty string
		expectedHash := "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		for _, entry := range manifest.Files {
			if entry.Name == "empty.txt" {
				if entry.Checksum != expectedHash {
					t.Errorf("Expected hash %s for empty file, got %s", expectedHash, entry.Checksum)
				}
			}
		}
	})
}

// Tests for WriteChecksumOnly function

func TestGenerator_WriteChecksumOnly(t *testing.T) {
	t.Parallel()

	t.Run("creates checksum when it doesn't exist", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create manifest file
		manifestPath := filepath.Join(tmpDir, "files.yaml")
		if err := os.WriteFile(manifestPath, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}

		generator := NewManifestGenerator(tmpDir, afero.NewOsFs())

		// Write checksum only
		err := generator.WriteChecksumOnly()
		if err != nil {
			t.Fatalf("WriteChecksumOnly failed: %v", err)
		}

		// Verify checksum exists
		checksumPath := filepath.Join(tmpDir, "files.yaml.sha256")
		if _, err := os.Stat(checksumPath); os.IsNotExist(err) {
			t.Error("Checksum file was not created")
		}
	})

	t.Run("doesn't overwrite existing checksum", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create manifest file
		manifestPath := filepath.Join(tmpDir, "files.yaml")
		if err := os.WriteFile(manifestPath, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}

		// Create existing checksum
		checksumPath := filepath.Join(tmpDir, "files.yaml.sha256")
		originalContent := "original checksum content"
		if err := os.WriteFile(checksumPath, []byte(originalContent), 0600); err != nil {
			t.Fatal(err)
		}

		generator := NewManifestGenerator(tmpDir, afero.NewOsFs())

		// Try to write checksum
		err := generator.WriteChecksumOnly()
		if err != nil {
			t.Fatalf("WriteChecksumOnly failed: %v", err)
		}

		// Verify original content wasn't changed
		data, err := os.ReadFile(checksumPath)
		if err != nil {
			t.Fatalf("Failed to read checksum: %v", err)
		}

		if string(data) != originalContent {
			t.Error("Existing checksum file was overwritten")
		}
	})
}

// Tests for WriteManifestOnly function

func TestGenerator_WriteManifestOnly(t *testing.T) {
	t.Parallel()

	t.Run("writes only manifest without checksum", func(t *testing.T) {
		tmpDir := t.TempDir()

		generator := NewManifestGenerator(tmpDir, afero.NewOsFs())

		manifest := &FileManifest{
			Version:   "1.0",
			Files:     []FileEntry{},
			Generator: "test",
		}

		// Write manifest only
		err := generator.WriteManifestOnly(manifest)
		if err != nil {
			t.Fatalf("WriteManifestOnly failed: %v", err)
		}

		// Verify manifest exists
		manifestPath := filepath.Join(tmpDir, "files.yaml")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			t.Error("Manifest file was not created")
		}

		// Verify checksum doesn't exist
		checksumPath := filepath.Join(tmpDir, "files.yaml.sha256")
		if _, err := os.Stat(checksumPath); !os.IsNotExist(err) {
			t.Error("Checksum file should not exist")
		}
	})
}
