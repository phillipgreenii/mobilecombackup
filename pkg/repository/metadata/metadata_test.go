package metadata

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestReader_ReadMetadata(t *testing.T) {
	t.Parallel()

	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "metadata-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write test metadata file
	markerContent := `repository_structure_version: "1"
created_at: "2024-01-15T10:30:00Z"
created_by: "mobilecombackup v1.0.0"
`
	markerPath := filepath.Join(tmpDir, ".mobilecombackup.yaml")
	if err := os.WriteFile(markerPath, []byte(markerContent), 0600); err != nil {
		t.Fatalf("Failed to write marker file: %v", err)
	}

	// Create reader and read metadata
	reader := NewReader(tmpDir)
	metadata, err := reader.ReadMetadata()
	if err != nil {
		t.Fatalf("ReadMetadata() failed: %v", err)
	}

	// Verify fields
	if metadata.Version != "1" {
		t.Errorf("Version = %s; want 1", metadata.Version)
	}

	if metadata.CreatedBy != "mobilecombackup v1.0.0" {
		t.Errorf("CreatedBy = %s; want mobilecombackup v1.0.0", metadata.CreatedBy)
	}

	expectedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	if !metadata.CreatedAt.Equal(expectedTime) {
		t.Errorf("CreatedAt = %v; want %v", metadata.CreatedAt, expectedTime)
	}
}

func TestReader_ReadMetadata_InvalidYAML(t *testing.T) {
	t.Parallel()

	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "metadata-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write invalid YAML
	markerPath := filepath.Join(tmpDir, ".mobilecombackup.yaml")
	if err := os.WriteFile(markerPath, []byte("invalid: [yaml content"), 0600); err != nil {
		t.Fatalf("Failed to write marker file: %v", err)
	}

	// Create reader and attempt to read metadata
	reader := NewReader(tmpDir)
	_, err = reader.ReadMetadata()
	if err == nil {
		t.Error("ReadMetadata() should have failed with invalid YAML")
	}
}

func TestReader_ReadMetadata_MissingFile(t *testing.T) {
	t.Parallel()

	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "metadata-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Don't write marker file
	reader := NewReader(tmpDir)
	_, err = reader.ReadMetadata()
	if err == nil {
		t.Error("ReadMetadata() should have failed with missing file")
	}
	if !os.IsNotExist(err) {
		t.Errorf("ReadMetadata() error should be IsNotExist, got: %v", err)
	}
}

func TestReader_ReadMetadata_InvalidDate(t *testing.T) {
	t.Parallel()

	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "metadata-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write metadata with invalid date
	markerContent := `repository_structure_version: "1"
created_at: "invalid-date"
created_by: "test"
`
	markerPath := filepath.Join(tmpDir, ".mobilecombackup.yaml")
	if err := os.WriteFile(markerPath, []byte(markerContent), 0600); err != nil {
		t.Fatalf("Failed to write marker file: %v", err)
	}

	// Create reader and read metadata
	reader := NewReader(tmpDir)
	metadata, err := reader.ReadMetadata()
	if err != nil {
		t.Fatalf("ReadMetadata() failed: %v", err)
	}

	// CreatedAt should be zero time when parsing fails
	if !metadata.CreatedAt.IsZero() {
		t.Errorf("CreatedAt should be zero time with invalid date, got: %v", metadata.CreatedAt)
	}
}

func TestReader_Exists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupFunc func(string) error
		want      bool
	}{
		{
			name: "file exists",
			setupFunc: func(tmpDir string) error {
				markerPath := filepath.Join(tmpDir, ".mobilecombackup.yaml")
				return os.WriteFile(markerPath, []byte("test"), 0600)
			},
			want: true,
		},
		{
			name:      "file does not exist",
			setupFunc: func(tmpDir string) error { return nil },
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tmpDir, err := os.MkdirTemp("", "metadata-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Setup test environment
			if err := tt.setupFunc(tmpDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			// Test Exists()
			reader := NewReader(tmpDir)
			got := reader.Exists()
			if got != tt.want {
				t.Errorf("Exists() = %v; want %v", got, tt.want)
			}
		})
	}
}

func TestMarkerFileContent_Unmarshal(t *testing.T) {
	t.Parallel()

	yamlData := []byte(`repository_structure_version: "1"
created_at: "2024-01-15T10:30:00Z"
created_by: "mobilecombackup v1.0.0"
`)

	var marker MarkerFileContent
	err := yaml.Unmarshal(yamlData, &marker)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if marker.RepositoryStructureVersion != "1" {
		t.Errorf("RepositoryStructureVersion = %s; want 1", marker.RepositoryStructureVersion)
	}

	if marker.CreatedAt != "2024-01-15T10:30:00Z" {
		t.Errorf("CreatedAt = %s; want 2024-01-15T10:30:00Z", marker.CreatedAt)
	}

	if marker.CreatedBy != "mobilecombackup v1.0.0" {
		t.Errorf("CreatedBy = %s; want mobilecombackup v1.0.0", marker.CreatedBy)
	}
}

func TestReader_ReadMetadata_EmptyCreatedAt(t *testing.T) {
	t.Parallel()

	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "metadata-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write metadata without created_at
	markerContent := `repository_structure_version: "1"
created_by: "test"
`
	markerPath := filepath.Join(tmpDir, ".mobilecombackup.yaml")
	if err := os.WriteFile(markerPath, []byte(markerContent), 0600); err != nil {
		t.Fatalf("Failed to write marker file: %v", err)
	}

	// Create reader and read metadata
	reader := NewReader(tmpDir)
	metadata, err := reader.ReadMetadata()
	if err != nil {
		t.Fatalf("ReadMetadata() failed: %v", err)
	}

	// CreatedAt should be zero time when not present
	if !metadata.CreatedAt.IsZero() {
		t.Errorf("CreatedAt should be zero time when not present, got: %v", metadata.CreatedAt)
	}
}
