package importer

import (
	"time"
)

// ImportSummary tracks statistics for an import operation
type ImportSummary struct {
	// Statistics by entity type
	Calls       *EntityStats
	SMS         *EntityStats
	Attachments *AttachmentStats
	// Rejection statistics by type
	Rejections map[string]*RejectionStats
	// Files processed
	FilesProcessed int
	// Rejection files created
	RejectionFiles []string
	// Processing time
	Duration time.Duration
}

// SummaryFile represents the repository summary written to summary.yaml
type SummaryFile struct {
	LastUpdated string    `yaml:"last_updated"`
	Statistics  RepoStats `yaml:"statistics"`
}

// RepoStats contains repository-wide statistics
type RepoStats struct {
	TotalCalls       int   `yaml:"total_calls"`
	TotalSMS         int   `yaml:"total_sms"`
	TotalAttachments int   `yaml:"total_attachments"`
	YearsCovered     []int `yaml:"years_covered"`
}

// EntityStats tracks statistics for a specific entity type (calls or SMS)
type EntityStats struct {
	// Per-year statistics
	YearStats map[int]*YearStat
	// Total statistics across all years
	Total *YearStat
}

// AttachmentStats tracks statistics for attachments
type AttachmentStats struct {
	// Total statistics
	Total *AttachmentStat
}

// AttachmentStat tracks attachment statistics
type AttachmentStat struct {
	// Total attachment files
	Total int
	// New attachments added
	New int
	// Duplicate attachments
	Duplicates int
}

// RejectionStats tracks rejection statistics by reason
type RejectionStats struct {
	// Count of rejections
	Count int
	// Rejection reason
	Reason string
}

// YearStat tracks statistics for a specific year
type YearStat struct {
	// Initial count from repository
	Initial int
	// Final count after import
	Final int
	// Newly added entries
	Added int
	// Duplicate entries skipped
	Duplicates int
	// Rejected entries
	Rejected int
	// Errors encountered
	Errors int
}

// ImportOptions configures the import behavior
type ImportOptions struct {
	// Repository root path
	RepoRoot string
	// Files or directories to import
	Paths []string
	// Dry run mode - process without writing
	DryRun bool
	// Verbose output
	Verbose bool
	// Quiet mode - suppress progress
	Quiet bool
	// JSON output format
	JSON bool
	// Filter to process only specific types
	Filter string // "calls", "sms", or ""
	// Don't error on rejected entries
	NoErrorOnRejects bool
	// Progress reporter for UI updates
	ProgressReporter ProgressReporter
}

// ProgressReporter handles progress updates during import
type ProgressReporter interface {
	// StartFile is called when starting to process a file
	StartFile(filename string, totalFiles int, currentFile int)
	// UpdateProgress is called periodically during file processing
	UpdateProgress(processed, total int)
	// EndFile is called when finished processing a file
	EndFile(filename string, summary *YearStat)
}

// RejectionWriter handles writing rejected entries
type RejectionWriter interface {
	// WriteRejections writes rejected entries to a file
	WriteRejections(originalFile string, rejections []RejectedEntry) (string, error)
}

// RejectedEntry represents an entry that failed validation
type RejectedEntry struct {
	// Line number in source file (1-based)
	Line int
	// Original XML data
	Data string
	// Validation violations
	Violations []string
}