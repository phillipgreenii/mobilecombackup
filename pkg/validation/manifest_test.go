package validation

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestManifestValidatorImpl_LoadManifest(t *testing.T) {
	tempDir := t.TempDir()
	validator := NewManifestValidator(tempDir)

	// Test missing files.yaml
	_, err := validator.LoadManifest()
	if err == nil {
		t.Error("Expected error for missing files.yaml")
	}

	// Create valid files.yaml
	manifest := FileManifest{
		Files: []FileEntry{
			{Name: "summary.yaml", Checksum: "sha256:abc123", Size: 100},
			{Name: "contacts.yaml", Checksum: "sha256:def456", Size: 200},
		},
	}

	manifestData, err := yaml.Marshal(manifest)
	if err != nil {
		t.Fatalf("Failed to marshal manifest: %v", err)
	}

	manifestPath := filepath.Join(tempDir, "files.yaml")
	err = os.WriteFile(manifestPath, manifestData, 0644)
	if err != nil {
		t.Fatalf("Failed to write files.yaml: %v", err)
	}

	// Test successful load
	loaded, err := validator.LoadManifest()
	if err != nil {
		t.Fatalf("Failed to load manifest: %v", err)
	}

	if len(loaded.Files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(loaded.Files))
	}

	if loaded.Files[0].Name != "summary.yaml" {
		t.Errorf("Expected first file to be summary.yaml, got %s", loaded.Files[0].Name)
	}
}

func TestManifestValidatorImpl_ValidateManifestFormat(t *testing.T) {
	validator := NewManifestValidator("/tmp")

	tests := []struct {
		name      string
		manifest  FileManifest
		wantCount int
	}{
		{
			name: "valid manifest",
			manifest: FileManifest{
				Files: []FileEntry{
					{Name: "summary.yaml", Checksum: "sha256:a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3", Size: 100},
					{Name: "calls/calls-2015.xml", Checksum: "sha256:b7c9c1c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9", Size: 200},
				},
			},
			wantCount: 0,
		},
		{
			name: "duplicate files",
			manifest: FileManifest{
				Files: []FileEntry{
					{Name: "summary.yaml", Checksum: "sha256:a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3", Size: 100},
					{Name: "summary.yaml", Checksum: "sha256:b7c9c1c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9", Size: 200},
				},
			},
			wantCount: 1,
		},
		{
			name: "invalid SHA-256",
			manifest: FileManifest{
				Files: []FileEntry{
					{Name: "summary.yaml", Checksum: "sha256:invalid", Size: 100},
				},
			},
			wantCount: 1,
		},
		{
			name: "invalid size",
			manifest: FileManifest{
				Files: []FileEntry{
					{Name: "summary.yaml", Checksum: "sha256:a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3", Size: -1},
				},
			},
			wantCount: 1,
		},
		{
			name: "absolute path",
			manifest: FileManifest{
				Files: []FileEntry{
					{Name: "/absolute/path.yaml", Checksum: "sha256:a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3", Size: 100},
				},
			},
			wantCount: 1,
		},
		{
			name: "path with ..",
			manifest: FileManifest{
				Files: []FileEntry{
					{Name: "../outside.yaml", Checksum: "sha256:a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3", Size: 100},
				},
			},
			wantCount: 1,
		},
		{
			name: "includes files.yaml",
			manifest: FileManifest{
				Files: []FileEntry{
					{Name: "files.yaml", Checksum: "sha256:a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3", Size: 100},
				},
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := validator.ValidateManifestFormat(&tt.manifest)
			if len(violations) != tt.wantCount {
				t.Errorf("Expected %d violations, got %d: %v", tt.wantCount, len(violations), violations)
			}
		})
	}
}

func TestManifestValidatorImpl_CheckManifestCompleteness(t *testing.T) {
	tempDir := t.TempDir()
	validator := NewManifestValidator(tempDir)

	// Create some test files
	testFiles := []string{
		"summary.yaml",
		"contacts.yaml",
		"calls/calls-2015.xml",
		"sms/sms-2015.xml",
		"attachments/ab/ab123456",
	}

	for _, file := range testFiles {
		fullPath := filepath.Join(tempDir, file)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		err = os.WriteFile(fullPath, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	tests := []struct {
		name      string
		manifest  FileManifest
		wantCount int
	}{
		{
			name: "complete manifest",
			manifest: FileManifest{
				Files: []FileEntry{
					{Name: "summary.yaml", Checksum: "sha256:abc123", Size: 100},
					{Name: "contacts.yaml", Checksum: "sha256:def456", Size: 200},
					{Name: "calls/calls-2015.xml", Checksum: "sha256:ghi789", Size: 300},
					{Name: "sms/sms-2015.xml", Checksum: "sha256:jkl012", Size: 400},
					{Name: "attachments/ab/ab123456", Checksum: "sha256:mno345", Size: 500},
				},
			},
			wantCount: 0,
		},
		{
			name: "missing file in manifest",
			manifest: FileManifest{
				Files: []FileEntry{
					{Name: "summary.yaml", Checksum: "sha256:abc123", Size: 100},
					{Name: "contacts.yaml", Checksum: "sha256:def456", Size: 200},
					// Missing other files
				},
			},
			wantCount: 3, // 3 files not in manifest
		},
		{
			name: "extra file in manifest",
			manifest: FileManifest{
				Files: []FileEntry{
					{Name: "summary.yaml", Checksum: "sha256:abc123", Size: 100},
					{Name: "contacts.yaml", Checksum: "sha256:def456", Size: 200},
					{Name: "calls/calls-2015.xml", Checksum: "sha256:ghi789", Size: 300},
					{Name: "sms/sms-2015.xml", Checksum: "sha256:jkl012", Size: 400},
					{Name: "attachments/ab/ab123456", Checksum: "sha256:mno345", Size: 500},
					{Name: "nonexistent.yaml", Checksum: "sha256:pqr678", Size: 600}, // Extra file
				},
			},
			wantCount: 1, // 1 file in manifest but not on disk
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := validator.CheckManifestCompleteness(&tt.manifest)
			if len(violations) != tt.wantCount {
				t.Errorf("Expected %d violations, got %d: %v", tt.wantCount, len(violations), violations)
			}
		})
	}
}

func TestManifestValidatorImpl_VerifyManifestChecksum(t *testing.T) {
	tempDir := t.TempDir()
	validator := NewManifestValidator(tempDir)

	// Test missing files.yaml.sha256
	err := validator.VerifyManifestChecksum()
	if err == nil {
		t.Error("Expected error for missing files.yaml.sha256")
	}

	// Create files.yaml
	manifestContent := []byte("test manifest content")
	manifestPath := filepath.Join(tempDir, "files.yaml")
	err = os.WriteFile(manifestPath, manifestContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create files.yaml: %v", err)
	}

	// Calculate correct checksum
	hasher := sha256.New()
	hasher.Write(manifestContent)
	correctChecksum := fmt.Sprintf("%x", hasher.Sum(nil))

	// Test with correct checksum
	checksumPath := filepath.Join(tempDir, "files.yaml.sha256")
	err = os.WriteFile(checksumPath, []byte(correctChecksum), 0644)
	if err != nil {
		t.Fatalf("Failed to create files.yaml.sha256: %v", err)
	}

	err = validator.VerifyManifestChecksum()
	if err != nil {
		t.Errorf("Expected no error with correct checksum, got: %v", err)
	}

	// Test with incorrect checksum
	wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"
	err = os.WriteFile(checksumPath, []byte(wrongChecksum), 0644)
	if err != nil {
		t.Fatalf("Failed to write wrong checksum: %v", err)
	}

	err = validator.VerifyManifestChecksum()
	if err == nil {
		t.Error("Expected error with wrong checksum")
	}

	// Test with invalid checksum format
	invalidChecksum := "invalid"
	err = os.WriteFile(checksumPath, []byte(invalidChecksum), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid checksum: %v", err)
	}

	err = validator.VerifyManifestChecksum()
	if err == nil {
		t.Error("Expected error with invalid checksum format")
	}
}
