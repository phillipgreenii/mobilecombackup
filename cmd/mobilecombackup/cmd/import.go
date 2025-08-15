package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/phillipgreen/mobilecombackup/pkg/importer"
	"github.com/phillipgreen/mobilecombackup/pkg/security"
	"github.com/spf13/cobra"
)

var (
	importDryRun      bool
	importJSON        bool
	importFilter      string
	noErrorOnRejects  bool
	maxXMLSizeStr     string
	maxMessageSizeStr string
)

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import [flags] [paths...]",
	Short: "Import mobile backup files into the repository",
	Long: `Import mobile backup files into the repository.

Before processing any files, validates the repository structure to ensure it is valid
and properly initialized. If validation fails, the import exits immediately without
processing any files.

Scans the specified paths (or current directory if none provided) for backup files
matching the patterns calls*.xml and sms*.xml, processes them to remove duplicates,
and imports new entries into the repository organized by year.

The repository location is determined by (in order of precedence):
1. --repo-root flag
2. MB_REPO_ROOT environment variable  
3. Current directory

Exit Codes:
  0  Success - all files processed without errors
  1  Warning - some entries were rejected but import completed
  2  Error - repository validation failed or import could not complete

Arguments:
  paths    Files or directories to import (default: current directory)`,
	Example: `  # Import specific files
  mobilecombackup import --repo-root /path/to/repo backup1.xml backup2.xml
  
  # Scan directory for backup files
  mobilecombackup import --repo-root /path/to/repo /path/to/backups/
  
  # Preview import without changes
  mobilecombackup import --repo-root /path/to/repo --dry-run backup.xml
  
  # Import only call logs
  mobilecombackup import --repo-root /path/to/repo --filter calls backups/`,
	RunE: runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)

	// Local flags
	importCmd.Flags().BoolVar(&importDryRun, "dry-run", false, "Preview import without making changes")
	importCmd.Flags().BoolVar(&importJSON, "json", false, "Output summary in JSON format")
	importCmd.Flags().StringVar(&importFilter, "filter", "", "Process only specific type: calls, sms")
	importCmd.Flags().BoolVar(&noErrorOnRejects, "no-error-on-rejects", false,
		"Don't exit with error code if rejects found")
	importCmd.Flags().StringVar(&maxXMLSizeStr, "max-xml-size", "500MB", "Maximum XML file size (e.g., 500MB, 1GB)")
	importCmd.Flags().StringVar(&maxMessageSizeStr, "max-message-size", "10MB", "Maximum message size (e.g., 10MB, 100MB)")
}

func runImport(cmd *cobra.Command, args []string) error {
	resolvedRepoRoot := resolveImportRepoRoot()
	paths, err := validateAndPreparePaths(cmd, args, resolvedRepoRoot)
	if err != nil {
		return err
	}

	options, err := createImportOptions(resolvedRepoRoot, paths)
	if err != nil {
		return err
	}

	imp, err := createImporter(options)
	if err != nil {
		return err
	}

	summary, err := imp.Import()
	if err != nil {
		handleImportError(err)
		return err
	}

	displayResults(summary)
	handleExitCode(summary)
	return nil
}

// validateAndPreparePaths validates the import arguments and prepares the paths list
func validateAndPreparePaths(cmd *cobra.Command, args []string, resolvedRepoRoot string) ([]string, error) {
	paths := args
	if len(paths) == 0 {
		// No paths specified
		if resolvedRepoRoot == "." && os.Getenv("MB_REPO_ROOT") == "" && repoRoot == "." {
			// No repository specified anywhere - error
			return nil, fmt.Errorf("no repository specified and no files to import\n\n%s", cmd.UsageString())
		}
		// Use current directory as import source
		paths = []string{"."}
	}

	// Validate filter
	if importFilter != "" && importFilter != callsDir && importFilter != smsDir {
		return nil, fmt.Errorf("invalid filter value: %s (must be '%s' or '%s')", importFilter, callsDir, smsDir)
	}

	return paths, nil
}

// createImportOptions creates and configures the import options
func createImportOptions(resolvedRepoRoot string, paths []string) (*importer.ImportOptions, error) {
	// Parse size limits
	maxXMLSize, err := security.ParseSize(maxXMLSizeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid max-xml-size: %w", err)
	}

	maxMessageSize, err := security.ParseSize(maxMessageSizeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid max-message-size: %w", err)
	}

	// JSON mode forces quiet to ensure clean JSON output
	effectiveQuiet := quiet || importJSON
	options := &importer.ImportOptions{
		RepoRoot:       resolvedRepoRoot,
		Paths:          paths,
		DryRun:         importDryRun,
		Verbose:        verbose,
		Quiet:          effectiveQuiet,
		Filter:         importFilter,
		MaxXMLSize:     maxXMLSize,
		MaxMessageSize: maxMessageSize,
	}

	// Create progress reporter if not quiet
	if !effectiveQuiet {
		options.ProgressReporter = &consoleProgressReporter{}
	}

	return options, nil
}

// createImporter creates and validates the importer instance
func createImporter(options *importer.ImportOptions) (*importer.Importer, error) {
	imp, err := importer.NewImporter(options)
	if err != nil {
		effectiveQuiet := options.Quiet
		if !effectiveQuiet {
			PrintError("Failed to initialize importer: %v", err)
		}
		// Exit with code 2 for initialization failures, unless in test mode
		if !testing.Testing() {
			os.Exit(2)
		}
		return nil, err
	}
	return imp, nil
}

// handleImportError handles errors from the import process
func handleImportError(err error) {
	if !quiet {
		PrintError("Import failed: %v", err)
	}
	os.Exit(2)
}

// displayResults displays the import results based on the output format
func displayResults(summary *importer.ImportSummary) {
	effectiveQuiet := quiet || importJSON
	if importJSON {
		displayJSONSummary(summary)
	} else if !effectiveQuiet {
		displaySummary(summary, importDryRun)
	}
}

// handleExitCode determines and sets the appropriate exit code
func handleExitCode(summary *importer.ImportSummary) {
	totalRejected := summary.Calls.Total.Rejected + summary.SMS.Total.Rejected
	if totalRejected > 0 && !noErrorOnRejects {
		os.Exit(1)
	}
}

// resolveImportRepoRoot determines the repository root directory
func resolveImportRepoRoot() string {
	// Command flag takes precedence
	if repoRoot != "." {
		return repoRoot
	}

	// Environment variable is second priority
	if envRoot := os.Getenv("MB_REPO_ROOT"); envRoot != "" {
		return envRoot
	}

	// Default to current directory
	return "."
}

// consoleProgressReporter implements progress reporting to console
type consoleProgressReporter struct {
	currentFile string
	fileCount   int
	fileIndex   int
}

func (r *consoleProgressReporter) StartFile(path string, totalFiles, currentFile int) {
	r.currentFile = filepath.Base(path)
	r.fileCount = totalFiles
	r.fileIndex = currentFile
	fmt.Printf("  Processing: %s ", r.currentFile)
}

func (r *consoleProgressReporter) UpdateProgress(processed, _ int) {
	if processed > 0 && processed%100 == 0 {
		fmt.Printf("(%d records)... ", processed)
	}
}

func (r *consoleProgressReporter) EndFile(_ string, stat *importer.YearStat) {
	fmt.Println("done")
}

// displaySummary displays the import summary in human-readable format
func displaySummary(summary *importer.ImportSummary, dryRun bool) {
	if dryRun {
		fmt.Println("\nDRY RUN: No changes were made to the repository")
	}

	fmt.Println("\nImport Summary:")

	displayCallsStatistics(summary)
	displaySMSStatistics(summary)
	displayAttachmentStatistics(summary)
	displayRejectionStatistics(summary)
	displayRejectionFiles(summary)
	displayTiming(summary)
}

// displayCallsStatistics displays call statistics in the summary
func displayCallsStatistics(summary *importer.ImportSummary) {
	if importFilter != "" && importFilter != "calls" {
		return
	}

	fmt.Println("\nCalls:")
	if len(summary.Calls.YearStats) > 0 {
		displayYearStats(summary.Calls.YearStats)
		fmt.Printf("  Total: %d entries (%d new, %d duplicates)\n",
			summary.Calls.Total.Final, summary.Calls.Total.Added, summary.Calls.Total.Duplicates)
	}
}

// displaySMSStatistics displays SMS statistics in the summary
func displaySMSStatistics(summary *importer.ImportSummary) {
	if importFilter != "" && importFilter != smsDir {
		return
	}

	fmt.Println("SMS:")
	if len(summary.SMS.YearStats) > 0 {
		displayYearStats(summary.SMS.YearStats)
		fmt.Printf("  Total: %d entries (%d new, %d duplicates)\n",
			summary.SMS.Total.Final, summary.SMS.Total.Added, summary.SMS.Total.Duplicates)
	}
}

// displayYearStats displays year-by-year statistics
func displayYearStats(yearStats map[int]*importer.YearStat) {
	// Sort years in ascending order for consistent display
	years := make([]int, 0, len(yearStats))
	for year := range yearStats {
		years = append(years, year)
	}
	sort.Ints(years)

	for _, year := range years {
		stats := yearStats[year]
		fmt.Printf("  %d: %d entries (%d new, %d duplicates)\n",
			year, stats.Final, stats.Added, stats.Duplicates)
	}
}

// displayAttachmentStatistics displays attachment statistics in the summary
func displayAttachmentStatistics(summary *importer.ImportSummary) {
	if (importFilter == "" || importFilter == "sms") && summary.Attachments.Total.Total > 0 {
		fmt.Println("Attachments:")
		fmt.Printf("  Total: %d files (%d new, %d duplicates)\n",
			summary.Attachments.Total.Total, summary.Attachments.Total.New,
			summary.Attachments.Total.Duplicates)
	}
}

// displayRejectionStatistics displays rejection statistics in the summary
func displayRejectionStatistics(summary *importer.ImportSummary) {
	totalRejections := calculateTotalRejections(summary)
	if totalRejections == 0 {
		return
	}

	fmt.Println("Rejections:")
	displayCallsRejections(summary)
	displaySMSRejections()
	fmt.Printf("  Total: %d\n", totalRejections)
}

// calculateTotalRejections calculates the total number of rejections
func calculateTotalRejections(summary *importer.ImportSummary) int {
	totalRejections := 0
	for _, stats := range summary.Rejections {
		totalRejections += stats.Count
	}
	return totalRejections
}

// displayCallsRejections displays call-specific rejection statistics
func displayCallsRejections(summary *importer.ImportSummary) {
	if importFilter != "" && importFilter != "calls" {
		return
	}

	callsRej := 0
	for _, stats := range summary.Rejections {
		if stats.Count > 0 {
			callsRej += stats.Count
		}
	}

	if callsRej > 0 {
		fmt.Printf("  Calls: %d", callsRej)
		displayRejectionBreakdown(summary.Rejections)
	}
}

// displayRejectionBreakdown displays rejection breakdown by reason
func displayRejectionBreakdown(rejections map[string]*importer.RejectionStats) {
	first := true
	for reason, stats := range rejections {
		if stats.Count > 0 {
			if first {
				fmt.Printf(" (%s: %d", reason, stats.Count)
				first = false
			} else {
				fmt.Printf(", %s: %d", reason, stats.Count)
			}
		}
	}
	if !first {
		fmt.Println(")")
	} else {
		fmt.Println()
	}
}

// displaySMSRejections displays SMS-specific rejection statistics
func displaySMSRejections() {
	if importFilter == "" || importFilter == "sms" {
		// TODO: Track SMS-specific rejections separately
		fmt.Printf("  SMS: 0\n")
	}
}

// displayRejectionFiles displays information about rejection files
func displayRejectionFiles(summary *importer.ImportSummary) {
	if len(summary.RejectionFiles) > 0 {
		fmt.Println("\nRejected entries saved to: rejected/")
	}
}

// displayTiming displays timing information
func displayTiming(summary *importer.ImportSummary) {
	fmt.Printf("\nFiles processed: %d\n", summary.FilesProcessed)
	fmt.Printf("Time taken: %.1fs\n", summary.Duration.Seconds())
}

// sortYearStatsForJSON converts YearStats map to an ordered map with years sorted in ascending order
func sortYearStatsForJSON(yearStats map[int]*importer.YearStat) map[string]interface{} {
	if len(yearStats) == 0 {
		return make(map[string]interface{})
	}

	// Sort years in ascending order
	years := make([]int, 0, len(yearStats))
	for year := range yearStats {
		years = append(years, year)
	}
	sort.Ints(years)

	// Create ordered map using sorted years
	result := make(map[string]interface{})
	for _, year := range years {
		result[fmt.Sprintf("%d", year)] = yearStats[year]
	}
	return result
}

// displayJSONSummary displays the import summary in JSON format
func displayJSONSummary(summary *importer.ImportSummary) {
	output := map[string]interface{}{
		"files_processed":  summary.FilesProcessed,
		"duration_seconds": summary.Duration.Seconds(),
		"total": map[string]interface{}{
			"initial":    summary.Calls.Total.Initial + summary.SMS.Total.Initial,
			"final":      summary.Calls.Total.Final + summary.SMS.Total.Final,
			"added":      summary.Calls.Total.Added + summary.SMS.Total.Added,
			"duplicates": summary.Calls.Total.Duplicates + summary.SMS.Total.Duplicates,
			"rejected":   summary.Calls.Total.Rejected + summary.SMS.Total.Rejected,
			"errors":     summary.Calls.Total.Errors + summary.SMS.Total.Errors,
		},
		"calls": map[string]interface{}{
			"total": map[string]interface{}{
				"initial":    summary.Calls.Total.Initial,
				"final":      summary.Calls.Total.Final,
				"added":      summary.Calls.Total.Added,
				"duplicates": summary.Calls.Total.Duplicates,
				"rejected":   summary.Calls.Total.Rejected,
				"errors":     summary.Calls.Total.Errors,
			},
			"years": sortYearStatsForJSON(summary.Calls.YearStats),
		},
		"sms": map[string]interface{}{
			"total": map[string]interface{}{
				"initial":    summary.SMS.Total.Initial,
				"final":      summary.SMS.Total.Final,
				"added":      summary.SMS.Total.Added,
				"duplicates": summary.SMS.Total.Duplicates,
				"rejected":   summary.SMS.Total.Rejected,
				"errors":     summary.SMS.Total.Errors,
			},
			"years": sortYearStatsForJSON(summary.SMS.YearStats),
		},
		"attachments": map[string]interface{}{
			"total":      summary.Attachments.Total.Total,
			"new":        summary.Attachments.Total.New,
			"duplicates": summary.Attachments.Total.Duplicates,
		},
		"rejections":      summary.Rejections,
		"rejection_files": summary.RejectionFiles,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON output: %v\n", err)
		os.Exit(2)
	}
}
