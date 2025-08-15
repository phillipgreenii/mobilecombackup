package autofix

import (
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/validation"
)

// AutofixReport represents the result of autofix operations
type AutofixReport struct {
	Timestamp         time.Time          `json:"timestamp"`
	RepositoryPath    string             `json:"repository_path"`
	FixedViolations   []FixedViolation   `json:"fixed_violations"`
	SkippedViolations []SkippedViolation `json:"skipped_violations"`
	Errors            []AutofixError     `json:"errors"`
	Summary           AutofixSummary     `json:"summary"`
}

// FixedViolation represents a violation that was successfully fixed
type FixedViolation struct {
	OriginalViolation validation.ValidationViolation `json:"original_violation"`
	FixAction         string                         `json:"fix_action"`
	Details           string                         `json:"details,omitempty"`
}

// SkippedViolation represents a violation that was skipped for safety
type SkippedViolation struct {
	OriginalViolation validation.ValidationViolation `json:"original_violation"`
	SkipReason        string                         `json:"skip_reason"`
}

// AutofixError represents an error that occurred during autofix
type AutofixError struct {
	ViolationType validation.ViolationType `json:"violation_type"`
	File          string                   `json:"file"`
	Operation     string                   `json:"operation"`
	Error         string                   `json:"error"`
}

// AutofixSummary provides counts of autofix results
type AutofixSummary struct {
	FixedCount   int `json:"fixed_count"`
	SkippedCount int `json:"skipped_count"`
	ErrorCount   int `json:"error_count"`
}

// AutofixOptions controls autofix behavior
type AutofixOptions struct {
	DryRun  bool `json:"dry_run"`
	Verbose bool `json:"verbose"`
}

// Autofix operation types for progress reporting
const (
	OperationCreateDirectory    = "create_directory"
	OperationCreateFile         = "create_file"
	OperationUpdateFile         = "update_file"
	OperationUpdateXMLCount     = "update_xml_count"
	OperationRegenerateManifest = "regenerate_manifest"
	OperationGenerateChecksum   = "generate_checksum"
)

// ProgressReporter provides progress updates during autofix operations
type ProgressReporter interface {
	StartOperation(operation string, details string)
	CompleteOperation(success bool, details string)
	ReportProgress(current, total int)
}

// NullProgressReporter discards all progress updates
type NullProgressReporter struct{}

// StartOperation starts an operation (no-op for null reporter)
func (r *NullProgressReporter) StartOperation(operation string, details string) {}

// CompleteOperation completes an operation (no-op for null reporter)
func (r *NullProgressReporter) CompleteOperation(success bool, details string) {}

// ReportProgress reports progress (no-op for null reporter)
func (r *NullProgressReporter) ReportProgress(current, total int) {}
