package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Importer handles the overall import process
type Importer struct {
	options       *ImportOptions
	callsImporter *CallsImporter
	smsImporter   *SMSImporter
}

// NewImporter creates a new importer
func NewImporter(options *ImportOptions) (*Importer, error) {
	// Validate repository
	if err := validateRepository(options.RepoRoot); err != nil {
		return nil, fmt.Errorf("invalid repository: %w", err)
	}
	
	// Create calls importer
	callsImporter, err := NewCallsImporter(options)
	if err != nil {
		return nil, fmt.Errorf("failed to create calls importer: %w", err)
	}
	
	// Create SMS importer
	smsImporter := NewSMSImporter(options)
	
	return &Importer{
		options:       options,
		callsImporter: callsImporter,
		smsImporter:   smsImporter,
	}, nil
}

// Import performs the import operation
func (imp *Importer) Import() (*ImportSummary, error) {
	startTime := time.Now()
	summary := &ImportSummary{
		Calls: &EntityStats{
			YearStats: make(map[int]*YearStat),
			Total:     &YearStat{},
		},
		SMS: &EntityStats{
			YearStats: make(map[int]*YearStat),
			Total:     &YearStat{},
		},
		Attachments: &AttachmentStats{
			Total: &AttachmentStat{},
		},
		Rejections: make(map[string]*RejectionStats),
	}
	
	// Load existing repository
	if !imp.options.Quiet {
		fmt.Println("Loading existing repository...")
	}
	
	if imp.options.Filter == "" || imp.options.Filter == "calls" {
		if err := imp.callsImporter.LoadRepository(); err != nil {
			return nil, fmt.Errorf("failed to load calls repository: %w", err)
		}
	}
	
	if imp.options.Filter == "" || imp.options.Filter == "sms" {
		if err := imp.smsImporter.LoadRepository(); err != nil {
			return nil, fmt.Errorf("failed to load SMS repository: %w", err)
		}
	}
	
	// Find files to import
	files, err := imp.findFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to find files: %w", err)
	}
	
	summary.FilesProcessed = len(files)
	
	// Process each file
	for i, file := range files {
		if imp.options.ProgressReporter != nil {
			imp.options.ProgressReporter.StartFile(file, len(files), i+1)
		}
		
		stat, err := imp.processFile(file)
		if err != nil {
			if !imp.options.Quiet {
				fmt.Printf("Error processing %s: %v\n", file, err)
			}
			continue
		}
		
		// Update summary based on file type
		name := filepath.Base(file)
		if strings.HasPrefix(name, "calls") {
			imp.updateCallsSummary(summary.Calls, stat)
		} else if strings.HasPrefix(name, "sms") {
			imp.updateSMSSummary(summary.SMS, stat)
		}
		
		if imp.options.ProgressReporter != nil {
			imp.options.ProgressReporter.EndFile(file, stat)
		}
	}
	
	// Write repository (single write operation)
	if !imp.options.DryRun {
		if !imp.options.Quiet {
			fmt.Println("\nWriting repository...")
		}
		
		if imp.options.Filter == "" || imp.options.Filter == "calls" {
			if err := imp.callsImporter.WriteRepository(); err != nil {
				return nil, fmt.Errorf("failed to write calls repository: %w", err)
			}
		}
		
		if imp.options.Filter == "" || imp.options.Filter == "sms" {
			if err := imp.smsImporter.WriteRepository(); err != nil {
				return nil, fmt.Errorf("failed to write SMS repository: %w", err)
			}
		}
	}
	
	// Update final statistics
	imp.finalizeSummary(summary)
	
	// Generate summary.yaml file (only in non-dry-run mode)
	if !imp.options.DryRun {
		if err := generateSummaryFile(imp.options.RepoRoot); err != nil {
			// Log error but don't fail the import
			if !imp.options.Quiet {
				fmt.Printf("Warning: Failed to generate summary.yaml: %v\n", err)
			}
		}
		
		// Generate files.yaml manifest
		if err := generateManifestFile(imp.options.RepoRoot, "dev"); err != nil {
			// Log error but don't fail the import
			if !imp.options.Quiet {
				fmt.Printf("Warning: Failed to generate files.yaml: %v\n", err)
			}
		}
	}
	
	summary.Duration = time.Since(startTime)
	return summary, nil
}

// validateRepository checks if the repository structure is valid
func validateRepository(repoRoot string) error {
	// For now, just check if the directory exists
	info, err := os.Stat(repoRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("repository does not exist: %s", repoRoot)
		}
		return err
	}
	
	if !info.IsDir() {
		return fmt.Errorf("repository path is not a directory: %s", repoRoot)
	}
	
	// TODO: Use validation from FEAT-007 when available
	return nil
}

// findFiles finds all files to import based on options
func (imp *Importer) findFiles() ([]string, error) {
	var files []string
	
	for _, path := range imp.options.Paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("failed to stat %s: %w", path, err)
		}
		
		if info.IsDir() {
			// Scan directory for backup files
			dirFiles, err := imp.scanDirectory(path)
			if err != nil {
				return nil, fmt.Errorf("failed to scan directory %s: %w", path, err)
			}
			files = append(files, dirFiles...)
		} else {
			// Single file
			if imp.shouldProcessFile(path) {
				files = append(files, path)
			}
		}
	}
	
	return files, nil
}

// scanDirectory recursively scans a directory for backup files
func (imp *Importer) scanDirectory(dir string) ([]string, error) {
	var files []string
	
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.IsDir() {
			// Skip hidden directories
			if strings.HasPrefix(info.Name(), ".") && path != dir {
				return filepath.SkipDir
			}
			return nil
		}
		
		if imp.shouldProcessFile(path) {
			files = append(files, path)
		}
		
		return nil
	})
	
	return files, err
}

// shouldProcessFile checks if a file should be processed
func (imp *Importer) shouldProcessFile(path string) bool {
	name := filepath.Base(path)
	
	// Skip files already in repository structure
	if strings.Contains(path, "/calls/") || strings.Contains(path, "/sms/") {
		return false
	}
	
	// Check filter
	switch imp.options.Filter {
	case "calls":
		return strings.HasPrefix(name, "calls") && strings.HasSuffix(name, ".xml")
	case "sms":
		return strings.HasPrefix(name, "sms") && strings.HasSuffix(name, ".xml")
	default:
		// Process both
		return (strings.HasPrefix(name, "calls") || strings.HasPrefix(name, "sms")) && 
		       strings.HasSuffix(name, ".xml")
	}
}

// processFile processes a single file
func (imp *Importer) processFile(path string) (*YearStat, error) {
	name := filepath.Base(path)
	
	if strings.HasPrefix(name, "calls") {
		return imp.callsImporter.ImportFile(path)
	}
	
	if strings.HasPrefix(name, "sms") {
		return imp.smsImporter.ImportFile(path)
	}
	
	return nil, fmt.Errorf("unknown file type: %s", name)
}

// updateCallsSummary updates the calls summary with file statistics
func (imp *Importer) updateCallsSummary(stats *EntityStats, stat *YearStat) {
	// Update total
	stats.Total.Added += stat.Added
	stats.Total.Duplicates += stat.Duplicates
	stats.Total.Rejected += stat.Rejected
	stats.Total.Errors += stat.Errors
}

// updateSMSSummary updates the SMS summary with file statistics
func (imp *Importer) updateSMSSummary(stats *EntityStats, stat *YearStat) {
	// Update total
	stats.Total.Added += stat.Added
	stats.Total.Duplicates += stat.Duplicates
	stats.Total.Rejected += stat.Rejected
	stats.Total.Errors += stat.Errors
}

// finalizeSummary calculates final statistics
func (imp *Importer) finalizeSummary(summary *ImportSummary) {
	if imp.options.Filter == "" || imp.options.Filter == "calls" {
		// Get statistics from calls coalescer
		coalSummary := imp.callsImporter.GetSummary()
		summary.Calls.Total.Initial = coalSummary.Initial
		summary.Calls.Total.Final = coalSummary.Final
		
		// Calculate per-year statistics for calls
		for _, entry := range imp.callsImporter.coalescer.GetAll() {
			year := entry.Year()
			if _, exists := summary.Calls.YearStats[year]; !exists {
				summary.Calls.YearStats[year] = &YearStat{}
			}
			summary.Calls.YearStats[year].Final++
		}
	}
	
	if imp.options.Filter == "" || imp.options.Filter == "sms" {
		// Get statistics from SMS coalescer
		coalSummary := imp.smsImporter.GetSummary()
		summary.SMS.Total.Initial = coalSummary.Initial
		summary.SMS.Total.Final = coalSummary.Final
		
		// Calculate per-year statistics for SMS
		for _, entry := range imp.smsImporter.coalescer.GetAll() {
			year := entry.Year()
			if _, exists := summary.SMS.YearStats[year]; !exists {
				summary.SMS.YearStats[year] = &YearStat{}
			}
			summary.SMS.YearStats[year].Final++
		}
		
		// Get attachment statistics from SMS importer
		attachStats := imp.smsImporter.GetAttachmentStats()
		summary.Attachments.Total.Total = attachStats.Total
		summary.Attachments.Total.New = attachStats.New
		summary.Attachments.Total.Duplicates = attachStats.Duplicates
	}
	
	// Collect rejection statistics
	imp.collectRejectionStats(summary)
}

// collectRejectionStats collects rejection statistics from importers
func (imp *Importer) collectRejectionStats(summary *ImportSummary) {
	// TODO: Implement rejection tracking
	// For now, initialize with zero counts
	summary.Rejections["missing-timestamp"] = &RejectionStats{
		Count:  0,
		Reason: "missing-timestamp",
	}
	summary.Rejections["malformed-attachment"] = &RejectionStats{
		Count:  0,
		Reason: "malformed-attachment",
	}
	summary.Rejections["parse-error"] = &RejectionStats{
		Count:  0,
		Reason: "parse-error",
	}
}