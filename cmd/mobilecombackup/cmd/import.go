package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/phillipgreen/mobilecombackup/pkg/importer"
	"github.com/spf13/cobra"
)

var (
	importDryRun         bool
	importVerbose        bool
	importJSON           bool
	importFilter         string
	noErrorOnRejects     bool
)

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import [flags] [paths...]",
	Short: "Import mobile backup files into the repository",
	Long: `Import mobile backup files into the repository.

Scans the specified paths (or current directory if none provided) for backup files
matching the patterns calls*.xml and sms*.xml, processes them to remove duplicates,
and imports new entries into the repository organized by year.

The repository location is determined by (in order of precedence):
1. --repo-root flag
2. MB_REPO_ROOT environment variable  
3. Current directory

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
	importCmd.Flags().BoolVar(&importVerbose, "verbose", false, "Enable verbose output")
	importCmd.Flags().BoolVar(&importJSON, "json", false, "Output summary in JSON format")
	importCmd.Flags().StringVar(&importFilter, "filter", "", "Process only specific type: calls, sms")
	importCmd.Flags().BoolVar(&noErrorOnRejects, "no-error-on-rejects", false, "Don't exit with error code if rejects found")
}

func runImport(cmd *cobra.Command, args []string) error {
	// Determine repository root
	resolvedRepoRoot := resolveImportRepoRoot()
	
	// Get paths to import
	paths := args
	if len(paths) == 0 {
		// No paths specified
		if resolvedRepoRoot == "." && os.Getenv("MB_REPO_ROOT") == "" && repoRoot == "." {
			// No repository specified anywhere - error
			return fmt.Errorf("no repository specified and no files to import\n\n%s", cmd.UsageString())
		}
		// Use current directory as import source
		paths = []string{"."}
	}

	// Validate filter
	if importFilter != "" && importFilter != "calls" && importFilter != "sms" {
		return fmt.Errorf("invalid filter value: %s (must be 'calls' or 'sms')", importFilter)
	}

	// Create import options
	options := &importer.ImportOptions{
		RepoRoot: resolvedRepoRoot,
		Paths:    paths,
		DryRun:   importDryRun,
		Verbose:  importVerbose,
		Quiet:    quiet,
		Filter:   importFilter,
	}

	// Create progress reporter if not quiet
	if !quiet {
		options.ProgressReporter = &consoleProgressReporter{}
	}

	// Create importer
	imp, err := importer.NewImporter(options)
	if err != nil {
		if !quiet {
			PrintError("Failed to initialize importer: %v", err)
		}
		return err
	}

	// Run import
	summary, err := imp.Import()
	if err != nil {
		if !quiet {
			PrintError("Import failed: %v", err)
		}
		os.Exit(2)
	}

	// Display results
	if importJSON {
		displayJSONSummary(summary)
	} else if !quiet {
		displaySummary(summary, importDryRun)
	}

	// Determine exit code
	if summary.Total.Rejected > 0 && !noErrorOnRejects {
		os.Exit(1)
	}

	return nil
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

func (r *consoleProgressReporter) UpdateProgress(processed, rejected int) {
	if processed > 0 && processed%100 == 0 {
		fmt.Printf("(%d records)... ", processed)
	}
}

func (r *consoleProgressReporter) EndFile(path string, stat *importer.YearStat) {
	fmt.Println("done")
}

// displaySummary displays the import summary in human-readable format
func displaySummary(summary *importer.ImportSummary, dryRun bool) {
	if dryRun {
		fmt.Println("\nDRY RUN: No changes were made to the repository")
	}
	
	fmt.Println("\nImport Summary:")
	fmt.Println("              Initial     Final     Delta     Duplicates    Rejected")
	
	// Display totals for each type
	if importFilter == "" || importFilter == "calls" {
		fmt.Printf("Calls Total   %7d   %7d   %7d   %11d   %9d\n",
			summary.Total.Initial, summary.Total.Final, 
			summary.Total.Added, summary.Total.Duplicates, summary.Total.Rejected)
	}
	
	if importFilter == "" || importFilter == "sms" {
		fmt.Printf("SMS Total     %7d   %7d   %7d   %11d   %9d\n",
			summary.Total.Initial, summary.Total.Final,
			summary.Total.Added, summary.Total.Duplicates, summary.Total.Rejected)
	}
	
	// Display per-year breakdown if available
	if len(summary.YearStats) > 0 {
		for year, stats := range summary.YearStats {
			fmt.Printf("  %d        %7d   %7d   %7d   %11d   %9d\n",
				year, stats.Initial, stats.Final,
				stats.Added, stats.Duplicates, stats.Rejected)
		}
	}
	
	fmt.Printf("\nFiles processed: %d\n", summary.FilesProcessed)
	
	// Display rejection files if any
	if len(summary.RejectionFiles) > 0 {
		fmt.Println("Rejected files created:")
		for _, file := range summary.RejectionFiles {
			fmt.Printf("  - %s\n", file)
		}
	}
	
	// Display timing
	fmt.Printf("\nTime taken: %.1fs\n", summary.Duration.Seconds())
}

// displayJSONSummary displays the import summary in JSON format
func displayJSONSummary(summary *importer.ImportSummary) {
	output := map[string]interface{}{
		"files_processed": summary.FilesProcessed,
		"duration_seconds": summary.Duration.Seconds(),
		"total": map[string]interface{}{
			"initial":    summary.Total.Initial,
			"final":      summary.Total.Final,
			"added":      summary.Total.Added,
			"duplicates": summary.Total.Duplicates,
			"rejected":   summary.Total.Rejected,
			"errors":     summary.Total.Errors,
		},
		"years": summary.YearStats,
		"rejection_files": summary.RejectionFiles,
	}
	
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.Encode(output)
}