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

// ValidationReport represents the result of repository validation
type ValidationReport struct {
	Timestamp      time.Time             `yaml:"timestamp"`
	RepositoryPath string                `yaml:"repository_path"`
	Status         ValidationStatus      `yaml:"status"`
	Violations     []ValidationViolation `yaml:"violations"`
}

// ValidationStatus represents the overall validation result
type ValidationStatus string

const (
	Valid       ValidationStatus = "valid"
	Invalid     ValidationStatus = "invalid"
	ErrorStatus ValidationStatus = "error"
)

// ValidationViolation represents a specific validation issue
type ValidationViolation struct {
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
	MissingFile        ViolationType = "missing_file"
	ExtraFile          ViolationType = "extra_file"
	ChecksumMismatch   ViolationType = "checksum_mismatch"
	InvalidFormat      ViolationType = "invalid_format"
	OrphanedAttachment ViolationType = "orphaned_attachment"
	CountMismatch      ViolationType = "count_mismatch"
	SizeMismatch       ViolationType = "size_mismatch"
	StructureViolation ViolationType = "structure_violation"
	MissingMarkerFile  ViolationType = "missing_marker_file"
	UnsupportedVersion ViolationType = "unsupported_version"
	FormatMismatch     ViolationType = "format_mismatch"
	UnknownFormat      ViolationType = "unknown_format"
)

// Severity indicates the importance of a validation issue
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// FixableViolation extends ValidationViolation with suggested fix content
type FixableViolation struct {
	ValidationViolation
	SuggestedFix string `yaml:"suggested_fix,omitempty"`
}

// ManifestValidator validates files.yaml structure and content
type ManifestValidator interface {
	// LoadManifest reads and parses files.yaml
	LoadManifest() (*FileManifest, error)

	// ValidateManifestFormat checks files.yaml structure and format
	ValidateManifestFormat(manifest *FileManifest) []ValidationViolation

	// CheckManifestCompleteness verifies all files are listed
	CheckManifestCompleteness(manifest *FileManifest) []ValidationViolation

	// VerifyManifestChecksum validates files.yaml.sha256
	VerifyManifestChecksum() error
}
