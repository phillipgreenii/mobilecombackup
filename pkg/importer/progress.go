package importer

import (
	"fmt"
)

// ConsoleProgressReporter reports progress to the console
type ConsoleProgressReporter struct {
	quiet   bool
	verbose bool
}

// NewConsoleProgressReporter creates a new console progress reporter
func NewConsoleProgressReporter(quiet, verbose bool) *ConsoleProgressReporter {
	return &ConsoleProgressReporter{
		quiet:   quiet,
		verbose: verbose,
	}
}

// StartFile is called when starting to process a file
func (r *ConsoleProgressReporter) StartFile(filename string, totalFiles int, currentFile int) {
	if r.quiet {
		return
	}
	fmt.Printf("Processing: %s", filename)
	if r.verbose {
		fmt.Printf(" (%d/%d)", currentFile, totalFiles)
	}
}

// UpdateProgress is called periodically during file processing
func (r *ConsoleProgressReporter) UpdateProgress(processed, total int) {
	if r.quiet {
		return
	}
	// Print progress every 100 records
	if processed%100 == 0 {
		fmt.Printf(" (%d records)...", processed)
	}
}

// EndFile is called when finished processing a file
func (r *ConsoleProgressReporter) EndFile(filename string, summary *YearStat) {
	if r.quiet {
		return
	}
	fmt.Printf(" done")
	if r.verbose && summary != nil {
		fmt.Printf(" [Added: %d, Duplicates: %d, Rejected: %d]", 
			summary.Added, summary.Duplicates, summary.Rejected)
	}
	fmt.Println()
}