package validation

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// MarkerFileContent represents the .mobilecombackup.yaml file structure
type MarkerFileContent struct {
	RepositoryStructureVersion string `yaml:"repository_structure_version"`
	CreatedAt                  string `yaml:"created_at"`
	CreatedBy                  string `yaml:"created_by"`
}

// MarkerFileValidator validates the .mobilecombackup.yaml marker file
type MarkerFileValidator interface {
	// ValidateMarkerFile checks the marker file exists and has valid content
	ValidateMarkerFile() ([]ValidationViolation, bool, error)

	// GetSuggestedFix returns the suggested content for a missing marker file
	GetSuggestedFix() string
}

// MarkerFileValidatorImpl implements MarkerFileValidator
type MarkerFileValidatorImpl struct {
	repositoryRoot string
	logger         *log.Logger
}

// NewMarkerFileValidator creates a new marker file validator
func NewMarkerFileValidator(repositoryRoot string) MarkerFileValidator {
	return &MarkerFileValidatorImpl{
		repositoryRoot: repositoryRoot,
		logger:         log.New(os.Stderr, "[MarkerValidator] ", log.LstdFlags),
	}
}

// ValidateMarkerFile validates the .mobilecombackup.yaml marker file
// Returns: violations, versionSupported, error
func (v *MarkerFileValidatorImpl) ValidateMarkerFile() ([]ValidationViolation, bool, error) {
	var violations []ValidationViolation
	markerPath := filepath.Join(v.repositoryRoot, ".mobilecombackup.yaml")

	// Check if file exists
	file, err := os.Open(markerPath) // nolint:gosec // Validation requires file access
	if err != nil {
		if os.IsNotExist(err) {
			violations = append(violations, ValidationViolation{
				Type:     MissingMarkerFile,
				Severity: SeverityError,
				File:     ".mobilecombackup.yaml",
				Message:  "Repository marker file is missing",
				Expected: "Marker file with repository metadata",
				Actual:   "File not found",
			})
			return violations, true, nil // Version supported since file is missing
		}
		return nil, false, fmt.Errorf("failed to open marker file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read marker file: %w", err)
	}

	// Validate YAML structure first
	var rawData map[string]interface{}
	if err := yaml.Unmarshal(content, &rawData); err != nil {
		violations = append(violations, ValidationViolation{
			Type:     InvalidFormat,
			Severity: SeverityError,
			File:     ".mobilecombackup.yaml",
			Message:  fmt.Sprintf("Invalid YAML syntax: %v", err),
		})
		return violations, false, nil // Can't determine version support
	}

	// Parse into structured content
	var markerContent MarkerFileContent
	if err := yaml.Unmarshal(content, &markerContent); err != nil {
		violations = append(violations, ValidationViolation{
			Type:     InvalidFormat,
			Severity: SeverityError,
			File:     ".mobilecombackup.yaml",
			Message:  fmt.Sprintf("Failed to parse marker file content: %v", err),
		})
		return violations, false, nil
	}

	// Validate required fields
	if markerContent.RepositoryStructureVersion == "" {
		violations = append(violations, ValidationViolation{
			Type:     InvalidFormat,
			Severity: SeverityError,
			File:     ".mobilecombackup.yaml",
			Message:  "Missing required field: repository_structure_version",
		})
	} else if markerContent.RepositoryStructureVersion != "1" {
		violations = append(violations, ValidationViolation{
			Type:     UnsupportedVersion,
			Severity: SeverityError,
			File:     ".mobilecombackup.yaml",
			Message:  fmt.Sprintf("Unsupported repository structure version: %s", markerContent.RepositoryStructureVersion),
			Expected: "1",
			Actual:   markerContent.RepositoryStructureVersion,
		})
		return violations, false, nil // Version not supported
	}

	if markerContent.CreatedAt == "" {
		violations = append(violations, ValidationViolation{
			Type:     InvalidFormat,
			Severity: SeverityError,
			File:     ".mobilecombackup.yaml",
			Message:  "Missing required field: created_at",
		})
	} else {
		// Validate RFC3339 timestamp
		if _, err := time.Parse(time.RFC3339, markerContent.CreatedAt); err != nil {
			violations = append(violations, ValidationViolation{
				Type:     InvalidFormat,
				Severity: SeverityError,
				File:     ".mobilecombackup.yaml",
				Message:  fmt.Sprintf("Invalid RFC3339 timestamp in created_at: %s", markerContent.CreatedAt),
			})
		}
	}

	if markerContent.CreatedBy == "" {
		violations = append(violations, ValidationViolation{
			Type:     InvalidFormat,
			Severity: SeverityError,
			File:     ".mobilecombackup.yaml",
			Message:  "Missing required field: created_by",
		})
	}

	// Check for extra fields and log warnings
	for key := range rawData {
		switch key {
		case "repository_structure_version", "created_at", "created_by":
			// Expected fields
		default:
			v.logger.Printf("Warning: unexpected field '%s' in marker file", key)
		}
	}

	// Return true for version supported if version is "1" or missing
	versionSupported := markerContent.RepositoryStructureVersion == "" || markerContent.RepositoryStructureVersion == "1"

	return violations, versionSupported, nil
}

// GetSuggestedFix returns the suggested content for a missing marker file
func (v *MarkerFileValidatorImpl) GetSuggestedFix() string {
	return fmt.Sprintf(`repository_structure_version: "1"
created_at: "%s"
created_by: "mobilecombackup v1.0.0"`, time.Now().UTC().Format(time.RFC3339))
}
