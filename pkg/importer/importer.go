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
		YearStats: make(map[int]*YearStat),
		Total:     &YearStat{},
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
		
		// Update summary
		imp.updateSummary(summary, stat)
		
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

// updateSummary updates the summary with file statistics
func (imp *Importer) updateSummary(summary *ImportSummary, stat *YearStat) {
	// Update total
	summary.Total.Added += stat.Added
	summary.Total.Duplicates += stat.Duplicates
	summary.Total.Rejected += stat.Rejected
	summary.Total.Errors += stat.Errors
}

// finalizeSummary calculates final statistics
func (imp *Importer) finalizeSummary(summary *ImportSummary) {
	if imp.options.Filter == "" || imp.options.Filter == "calls" {
		// Get statistics from calls coalescer
		coalSummary := imp.callsImporter.GetSummary()
		summary.Total.Initial += coalSummary.Initial
		summary.Total.Final += coalSummary.Final
		
		// Calculate per-year statistics
		for _, entry := range imp.callsImporter.coalescer.GetAll() {
			year := entry.Year()
			if _, exists := summary.YearStats[year]; !exists {
				summary.YearStats[year] = &YearStat{}
			}
			summary.YearStats[year].Final++
		}
	}
	
	if imp.options.Filter == "" || imp.options.Filter == "sms" {
		// Get statistics from SMS coalescer
		coalSummary := imp.smsImporter.GetSummary()
		summary.Total.Initial += coalSummary.Initial
		summary.Total.Final += coalSummary.Final
		
		// Calculate per-year statistics
		for _, entry := range imp.smsImporter.coalescer.GetAll() {
			year := entry.Year()
			if _, exists := summary.YearStats[year]; !exists {
				summary.YearStats[year] = &YearStat{}
			}
			summary.YearStats[year].Final++
		}
	}
}