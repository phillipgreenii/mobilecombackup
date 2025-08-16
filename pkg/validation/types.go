package validation

import (
	"time"
)

// FileManifest represents the structure of files.yaml
type FileManifest struct {
	Version   string      `yaml:"version"`
	Generated string      `yaml:"generated"`
	Generator string      `yaml:"generator"`
	Files     []FileEntry `yaml:"files"`
}

// FileEntry represents a single file entry in files.yaml
type FileEntry struct {
	Name     string `yaml:"name"`
	Size     int64  `yaml:"size"`
	Checksum string `yaml:"checksum"`
	Modified string `yaml:"modified"`
}

// Report represents the result of repository validation
type Report struct {
	Timestamp      time.Time   `yaml:"timestamp"`
	RepositoryPath string      `yaml:"repository_path"`
	Status         Status      `yaml:"status"`
	Violations     []Violation `yaml:"violations"`
}

// Status represents the overall validation result
type Status string

const (
	// Valid indicates the repository passed all validation checks
	Valid Status = "valid"
	// Invalid indicates the repository has validation violations
	Invalid Status = "invalid"
	// ErrorStatus indicates validation could not be completed due to errors
	ErrorStatus Status = "error"
)

// Violation represents a specific validation issue
type Violation struct {
	Type     ViolationType `yaml:"type"`
	Severity Severity      `yaml:"severity"`
	File     string        `yaml:"file"`
	Message  string        `yaml:"message"`
	Expected string        `yaml:"expected,omitempty"`
	Actual   string        `yaml:"actual,omitempty"`
}

// ViolationType categorizes different types of validation issues
type ViolationType string

const (
	// MissingFile indicates a file referenced in the manifest is missing
	MissingFile ViolationType = "missing_file"
	// ExtraFile indicates a file exists but is not in the manifest
	ExtraFile ViolationType = "extra_file"
	// ChecksumMismatch indicates file content doesn't match expected checksum
	ChecksumMismatch ViolationType = "checksum_mismatch"
	// InvalidFormat indicates a file has invalid structure or format
	InvalidFormat ViolationType = "invalid_format"
	// OrphanedAttachment indicates an attachment file has no references
	OrphanedAttachment ViolationType = "orphaned_attachment"
	// CountMismatch indicates record count doesn't match expected value
	CountMismatch ViolationType = "count_mismatch"
	// SizeMismatch indicates file size doesn't match manifest entry
	SizeMismatch ViolationType = "size_mismatch"
	// StructureViolation indicates repository structure is invalid
	StructureViolation ViolationType = "structure_violation"
	// MissingMarkerFile indicates the repository marker file is missing
	MissingMarkerFile ViolationType = "missing_marker_file"
	// UnsupportedVersion indicates the repository uses an unsupported format version
	UnsupportedVersion ViolationType = "unsupported_version"
	// FormatMismatch indicates file format doesn't match expected format
	FormatMismatch ViolationType = "format_mismatch"
	// UnknownFormat indicates file format cannot be determined
	UnknownFormat ViolationType = "unknown_format"
)

// Severity indicates the importance of a validation issue
type Severity string

const (
	// SeverityError indicates a critical validation failure
	SeverityError Severity = "error"
	// SeverityWarning indicates a non-critical validation issue
	SeverityWarning Severity = "warning"
)

// FixableViolation extends Violation with suggested fix content
type FixableViolation struct {
	Violation
	SuggestedFix string `yaml:"suggested_fix,omitempty"`
}

// ManifestValidator validates files.yaml structure and content
type ManifestValidator interface {
	// LoadManifest reads and parses files.yaml
	LoadManifest() (*FileManifest, error)

	// ValidateManifestFormat checks files.yaml structure and format
	ValidateManifestFormat(manifest *FileManifest) []Violation

	// CheckManifestCompleteness verifies all files are listed
	CheckManifestCompleteness(manifest *FileManifest) []Violation

	// VerifyManifestChecksum validates files.yaml.sha256
	VerifyManifestChecksum() error
}
