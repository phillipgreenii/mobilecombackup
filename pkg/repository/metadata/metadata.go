// Package metadata provides repository metadata operations for reading and parsing repository marker files.
package metadata

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// MarkerFileContent represents the .mobilecombackup.yaml file structure
type MarkerFileContent struct {
	RepositoryStructureVersion string `yaml:"repository_structure_version" json:"repository_structure_version"`
	CreatedAt                  string `yaml:"created_at" json:"created_at"`
	CreatedBy                  string `yaml:"created_by" json:"created_by"`
}

// RepositoryMetadata contains parsed repository metadata
type RepositoryMetadata struct {
	Version   string
	CreatedAt time.Time
	CreatedBy string
}

// Reader provides access to repository metadata
type Reader struct {
	repoPath string
}

// NewReader creates a new metadata reader
func NewReader(repoPath string) *Reader {
	return &Reader{repoPath: repoPath}
}

// ReadMetadata reads and parses the repository metadata file
func (r *Reader) ReadMetadata() (*RepositoryMetadata, error) {
	markerPath := filepath.Join(r.repoPath, ".mobilecombackup.yaml")

	data, err := os.ReadFile(markerPath) // #nosec G304
	if err != nil {
		return nil, err
	}

	var marker MarkerFileContent
	if err := yaml.Unmarshal(data, &marker); err != nil {
		return nil, fmt.Errorf("failed to parse marker file: %w", err)
	}

	metadata := &RepositoryMetadata{
		Version:   marker.RepositoryStructureVersion,
		CreatedBy: marker.CreatedBy,
	}

	if marker.CreatedAt != "" {
		if createdAt, err := time.Parse(time.RFC3339, marker.CreatedAt); err == nil {
			metadata.CreatedAt = createdAt
		}
	}

	return metadata, nil
}

// Exists checks if the repository metadata file exists
func (r *Reader) Exists() bool {
	markerPath := filepath.Join(r.repoPath, ".mobilecombackup.yaml")
	_, err := os.Stat(markerPath)
	return err == nil
}

// UnmarshalYAML unmarshals YAML data into a MarkerFileContent
func UnmarshalYAML(data []byte, marker *MarkerFileContent) error {
	return yaml.Unmarshal(data, marker)
}
