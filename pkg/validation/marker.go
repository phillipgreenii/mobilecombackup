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
	ValidateMarkerFile() ([]Violation, bool, error)

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
func (v *MarkerFileValidatorImpl) ValidateMarkerFile() ([]Violation, bool, error) {
	markerPath := filepath.Join(v.repositoryRoot, ".mobilecombackup.yaml")

	// Load and parse marker file
	markerContent, rawData, violations, err := v.loadMarkerFile(markerPath)
	if err != nil {
		return violations, false, err
	}
	if len(violations) > 0 && markerContent == nil {
		// Missing file is supported, but malformed content is not
		if len(violations) > 0 && violations[0].Type == MissingMarkerFile {
			return violations, true, nil // Missing file is supported
		}
		return violations, false, nil // Malformed content can't determine version
	}

	// Validate marker content
	validationViolations, versionSupported := v.validateMarkerContent(markerContent, rawData)
	violations = append(violations, validationViolations...)

	return violations, versionSupported, nil
}

// loadMarkerFile loads and parses the marker file
func (v *MarkerFileValidatorImpl) loadMarkerFile(
	markerPath string,
) (*MarkerFileContent, map[string]interface{}, []Violation, error) {
	var violations []Violation

	// Check if file exists
	file, err := os.Open(markerPath) // nolint:gosec // Validation requires file access
	if err != nil {
		if os.IsNotExist(err) {
			violations = append(violations, Violation{
				Type:     MissingMarkerFile,
				Severity: SeverityError,
				File:     ".mobilecombackup.yaml",
				Message:  "Repository marker file is missing",
				Expected: "Marker file with repository metadata",
				Actual:   "File not found",
			})
			return nil, nil, violations, nil // Version supported since file is missing
		}
		return nil, nil, nil, fmt.Errorf("failed to open marker file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read marker file: %w", err)
	}

	// Parse YAML
	markerContent, rawData, parseViolations := v.parseMarkerContent(content)
	violations = append(violations, parseViolations...)

	return markerContent, rawData, violations, nil
}

// parseMarkerContent parses YAML content into structured data
func (v *MarkerFileValidatorImpl) parseMarkerContent(content []byte) (*MarkerFileContent, map[string]interface{}, []Violation) {
	var violations []Violation

	// Validate YAML structure first
	var rawData map[string]interface{}
	if err := yaml.Unmarshal(content, &rawData); err != nil {
		violations = append(violations, Violation{
			Type:     InvalidFormat,
			Severity: SeverityError,
			File:     ".mobilecombackup.yaml",
			Message:  fmt.Sprintf("Invalid YAML syntax: %v", err),
		})
		return nil, nil, violations // Can't determine version support
	}

	// Parse into structured content
	var markerContent MarkerFileContent
	if err := yaml.Unmarshal(content, &markerContent); err != nil {
		violations = append(violations, Violation{
			Type:     InvalidFormat,
			Severity: SeverityError,
			File:     ".mobilecombackup.yaml",
			Message:  fmt.Sprintf("Failed to parse marker file content: %v", err),
		})
		return nil, rawData, violations
	}

	return &markerContent, rawData, violations
}

// validateMarkerContent validates marker file fields and structure
func (v *MarkerFileValidatorImpl) validateMarkerContent(
	markerContent *MarkerFileContent,
	rawData map[string]interface{},
) ([]Violation, bool) {
	var violations []Violation
	versionSupported := true

	// Validate required fields
	if markerContent.RepositoryStructureVersion == "" {
		violations = append(violations, Violation{
			Type:     InvalidFormat,
			Severity: SeverityError,
			File:     ".mobilecombackup.yaml",
			Message:  "Missing required field: repository_structure_version",
		})
	} else if markerContent.RepositoryStructureVersion != "1" {
		violations = append(violations, Violation{
			Type:     UnsupportedVersion,
			Severity: SeverityError,
			File:     ".mobilecombackup.yaml",
			Message:  fmt.Sprintf("Unsupported repository structure version: %s", markerContent.RepositoryStructureVersion),
			Expected: "1",
			Actual:   markerContent.RepositoryStructureVersion,
		})
		versionSupported = false // Version not supported
	}

	if markerContent.CreatedAt == "" {
		violations = append(violations, Violation{
			Type:     InvalidFormat,
			Severity: SeverityError,
			File:     ".mobilecombackup.yaml",
			Message:  "Missing required field: created_at",
		})
	} else {
		// Validate RFC3339 timestamp
		if _, err := time.Parse(time.RFC3339, markerContent.CreatedAt); err != nil {
			violations = append(violations, Violation{
				Type:     InvalidFormat,
				Severity: SeverityError,
				File:     ".mobilecombackup.yaml",
				Message:  fmt.Sprintf("Invalid RFC3339 timestamp in created_at: %s", markerContent.CreatedAt),
			})
		}
	}

	if markerContent.CreatedBy == "" {
		violations = append(violations, Violation{
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
	if markerContent.RepositoryStructureVersion == "" {
		versionSupported = true
	}

	return violations, versionSupported
}

// GetSuggestedFix returns the suggested content for a missing marker file
func (v *MarkerFileValidatorImpl) GetSuggestedFix() string {
	return fmt.Sprintf(`repository_structure_version: "1"
created_at: "%s"
created_by: "mobilecombackup v1.0.0"`, time.Now().UTC().Format(time.RFC3339))
}
