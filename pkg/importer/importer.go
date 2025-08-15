package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/attachments"
	"github.com/phillipgreen/mobilecombackup/pkg/calls"
	"github.com/phillipgreen/mobilecombackup/pkg/contacts"
	"github.com/phillipgreen/mobilecombackup/pkg/sms"
	"github.com/phillipgreen/mobilecombackup/pkg/validation"
)

// YearTracker tracks per-year statistics during import
type YearTracker struct {
	initial    map[int]int // Entries loaded from existing repository
	added      map[int]int // New entries added during import
	duplicates map[int]int // Duplicate entries found during import
}

// NewYearTracker creates a new year tracker
func NewYearTracker() *YearTracker {
	return &YearTracker{
		initial:    make(map[int]int),
		added:      make(map[int]int),
		duplicates: make(map[int]int),
	}
}

// TrackInitialEntry tracks an entry loaded from existing repository
func (yt *YearTracker) TrackInitialEntry(year int) {
	yt.initial[year]++
}

// TrackImportEntry tracks an entry based on coalescer result
func (yt *YearTracker) TrackImportEntry(year int, wasAdded bool) {
	if wasAdded {
		yt.added[year]++
	} else {
		yt.duplicates[year]++
	}
}

// GetInitial returns initial count for a year
func (yt *YearTracker) GetInitial(year int) int {
	return yt.initial[year]
}

// GetAdded returns added count for a year
func (yt *YearTracker) GetAdded(year int) int {
	return yt.added[year]
}

// GetDuplicates returns duplicates count for a year
func (yt *YearTracker) GetDuplicates(year int) int {
	return yt.duplicates[year]
}

// ValidateYearStatistics validates that the mathematics work correctly for a year
func (yt *YearTracker) ValidateYearStatistics(year int, finalCount int) error {
	initial := yt.GetInitial(year)
	added := yt.GetAdded(year)
	duplicates := yt.GetDuplicates(year)

	// The mathematics should be: Initial + Added = Final
	expected := initial + added
	if expected != finalCount {
		return fmt.Errorf("year %d: mathematics error: Initial(%d) + Added(%d) = %d, but Final = %d",
			year, initial, added, expected, finalCount)
	}

	// Duplicates should not be included in Final count
	if duplicates < 0 {
		return fmt.Errorf("year %d: negative duplicates count: %d", year, duplicates)
	}

	return nil
}

// GetAllYears returns all years that have any activity
func (yt *YearTracker) GetAllYears() []int {
	yearsMap := make(map[int]bool)
	for year := range yt.initial {
		yearsMap[year] = true
	}
	for year := range yt.added {
		yearsMap[year] = true
	}
	for year := range yt.duplicates {
		yearsMap[year] = true
	}

	years := make([]int, 0, len(yearsMap))
	for year := range yearsMap {
		years = append(years, year)
	}
	return years
}

// Importer handles the overall import process
type Importer struct {
	options         *ImportOptions
	callsImporter   *CallsImporter
	smsImporter     *SMSImporter
	contactsManager *contacts.ContactsManager
	callsTracker    *YearTracker
	smsTracker      *YearTracker
}

// NewImporter creates a new importer
func NewImporter(options *ImportOptions) (*Importer, error) {
	// Validate repository
	if err := validateRepository(options.RepoRoot); err != nil {
		return nil, fmt.Errorf("invalid repository: %w", err)
	}

	// Create contacts manager
	contactsManager := contacts.NewContactsManager(options.RepoRoot)

	// Create year trackers
	callsTracker := NewYearTracker()
	smsTracker := NewYearTracker()

	// Create calls importer
	callsImporter, err := NewCallsImporter(options, contactsManager, callsTracker)
	if err != nil {
		return nil, fmt.Errorf("failed to create calls importer: %w", err)
	}

	// Create SMS importer
	smsImporter := NewSMSImporter(options, contactsManager, smsTracker)

	return &Importer{
		options:         options,
		callsImporter:   callsImporter,
		smsImporter:     smsImporter,
		contactsManager: contactsManager,
		callsTracker:    callsTracker,
		smsTracker:      smsTracker,
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

	// Load contacts.yaml
	if err := imp.contactsManager.LoadContacts(); err != nil {
		return nil, fmt.Errorf("failed to load contacts: %w", err)
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

		// Save contacts.yaml with any extracted contact names
		contactsPath := filepath.Join(imp.options.RepoRoot, "contacts.yaml")
		if err := imp.contactsManager.SaveContacts(contactsPath); err != nil {
			return nil, fmt.Errorf("failed to save contacts: %w", err)
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
	// Check basic directory existence first
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

	// Create readers required for validation
	callsReader := calls.NewXMLCallsReader(repoRoot)
	smsReader := sms.NewXMLSMSReader(repoRoot)
	attachmentReader := attachments.NewAttachmentManager(repoRoot)
	contactsReader := contacts.NewContactsManager(repoRoot)

	validator := validation.NewRepositoryValidator(
		repoRoot,
		callsReader,
		smsReader,
		attachmentReader,
		contactsReader,
	)

	report, err := validator.ValidateRepository()
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if report.Status != validation.Valid {
		return formatValidationError(report.Violations)
	}
	return nil
}

// formatValidationError formats validation violations into an error message
// matching the format used by the validate subcommand
func formatValidationError(violations []validation.ValidationViolation) error {
	if len(violations) == 0 {
		return nil
	}

	// Group violations by type (matching validate command format)
	violationsByType := make(map[string][]validation.ValidationViolation)
	for _, v := range violations {
		violationsByType[string(v.Type)] = append(violationsByType[string(v.Type)], v)
	}

	// Build error message
	var msg strings.Builder
	msg.WriteString("repository validation failed:")

	for vType, typeViolations := range violationsByType {
		msg.WriteString(fmt.Sprintf("\n  %s:", vType))
		for _, v := range typeViolations {
			msg.WriteString(fmt.Sprintf("\n    %s", v.Message))
			if v.File != "" {
				msg.WriteString(fmt.Sprintf(" (%s)", v.File))
			}
		}
	}

	return fmt.Errorf("%s", msg.String())
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

		// First, create YearStat entries for all years that had any activity
		allYears := make(map[int]bool)

		// Collect years from tracker activity
		for year := range imp.callsTracker.initial {
			allYears[year] = true
		}
		for year := range imp.callsTracker.added {
			allYears[year] = true
		}
		for year := range imp.callsTracker.duplicates {
			allYears[year] = true
		}

		// Initialize YearStat for all active years
		for year := range allYears {
			summary.Calls.YearStats[year] = &YearStat{
				Initial:    imp.callsTracker.GetInitial(year),
				Added:      imp.callsTracker.GetAdded(year),
				Duplicates: imp.callsTracker.GetDuplicates(year),
				Final:      0, // Will be counted below
			}
		}

		// Count final entries per year
		for _, entry := range imp.callsImporter.coalescer.GetAll() {
			year := entry.Year()
			if _, exists := summary.Calls.YearStats[year]; !exists {
				// This should not happen but be defensive
				summary.Calls.YearStats[year] = &YearStat{
					Initial:    imp.callsTracker.GetInitial(year),
					Added:      imp.callsTracker.GetAdded(year),
					Duplicates: imp.callsTracker.GetDuplicates(year),
					Final:      0,
				}
			}
			summary.Calls.YearStats[year].Final++
		}

		// Validate mathematics for all call years
		for year, stat := range summary.Calls.YearStats {
			if err := imp.callsTracker.ValidateYearStatistics(year, stat.Final); err != nil {
				if !imp.options.Quiet {
					fmt.Printf("Warning: Calls statistics validation failed: %v\n", err)
				}
			}
		}
	}

	if imp.options.Filter == "" || imp.options.Filter == "sms" {
		// Get statistics from SMS coalescer
		coalSummary := imp.smsImporter.GetSummary()
		summary.SMS.Total.Initial = coalSummary.Initial
		summary.SMS.Total.Final = coalSummary.Final

		// First, create YearStat entries for all years that had any activity
		allYears := make(map[int]bool)

		// Collect years from tracker activity
		for year := range imp.smsTracker.initial {
			allYears[year] = true
		}
		for year := range imp.smsTracker.added {
			allYears[year] = true
		}
		for year := range imp.smsTracker.duplicates {
			allYears[year] = true
		}

		// Initialize YearStat for all active years
		for year := range allYears {
			summary.SMS.YearStats[year] = &YearStat{
				Initial:    imp.smsTracker.GetInitial(year),
				Added:      imp.smsTracker.GetAdded(year),
				Duplicates: imp.smsTracker.GetDuplicates(year),
				Final:      0, // Will be counted below
			}
		}

		// Count final entries per year
		for _, entry := range imp.smsImporter.coalescer.GetAll() {
			year := entry.Year()
			if _, exists := summary.SMS.YearStats[year]; !exists {
				// This should not happen but be defensive
				summary.SMS.YearStats[year] = &YearStat{
					Initial:    imp.smsTracker.GetInitial(year),
					Added:      imp.smsTracker.GetAdded(year),
					Duplicates: imp.smsTracker.GetDuplicates(year),
					Final:      0,
				}
			}
			summary.SMS.YearStats[year].Final++
		}

		// Validate mathematics for all SMS years
		for year, stat := range summary.SMS.YearStats {
			if err := imp.smsTracker.ValidateYearStatistics(year, stat.Final); err != nil {
				if !imp.options.Quiet {
					fmt.Printf("Warning: SMS statistics validation failed: %v\n", err)
				}
			}
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
