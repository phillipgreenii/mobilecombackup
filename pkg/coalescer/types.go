package coalescer

import (
	"time"
)

// Entry represents a generic backup entry that can be coalesced
type Entry interface {
	// Hash returns a unique identifier for deduplication
	Hash() string
	// Timestamp returns the entry's timestamp for sorting
	Timestamp() time.Time
	// Year returns the year for partitioning
	Year() int
}

// Summary tracks the results of a coalescing operation
type Summary struct {
	// Existing entries loaded from repository
	Initial int
	// Total entries after coalescing (Initial + Added)
	Final int
	// New entries added (non-duplicates)
	Added int
	// Duplicate entries found and skipped
	Duplicates int
	// Invalid entries that were rejected
	Rejected int
	// Errors encountered during processing
	Errors int
}

// RejectedEntry represents an entry that failed validation
type RejectedEntry struct {
	// Original data that was rejected
	Data interface{}
	// Reasons why the entry was rejected
	Violations []string
	// Source file and line number
	SourceFile string
	Line       int
}

// Coalescer manages deduplication and accumulation of entries
type Coalescer[T Entry] interface {
	// LoadExisting loads entries from the repository for deduplication
	LoadExisting(entries []T) error
	
	// Add attempts to add an entry, returns true if added (not duplicate)
	Add(entry T) bool
	
	// GetAll returns all entries (existing + new) sorted by timestamp
	GetAll() []T
	
	// GetByYear returns entries for a specific year, sorted by timestamp
	GetByYear(year int) []T
	
	// GetSummary returns statistics about the coalescing operation
	GetSummary() Summary
	
	// Reset clears all entries and statistics
	Reset()
}

// ProgressReporter handles progress updates during processing
type ProgressReporter interface {
	// ReportProgress is called periodically during processing
	ReportProgress(processed, total int, currentFile string)
}