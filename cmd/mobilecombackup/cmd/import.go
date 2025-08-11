package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/phillipgreen/mobilecombackup/pkg/importer"
	"github.com/spf13/cobra"
)

var (
	importDryRun     bool
	importVerbose    bool
	importJSON       bool
	importFilter     string
	noErrorOnRejects bool
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
	// JSON mode forces quiet to ensure clean JSON output
	effectiveQuiet := quiet || importJSON
	options := &importer.ImportOptions{
		RepoRoot: resolvedRepoRoot,
		Paths:    paths,
		DryRun:   importDryRun,
		Verbose:  importVerbose,
		Quiet:    effectiveQuiet,
		Filter:   importFilter,
	}

	// Create progress reporter if not quiet
	if !effectiveQuiet {
		options.ProgressReporter = &consoleProgressReporter{}
	}

	// Create importer
	imp, err := importer.NewImporter(options)
	if err != nil {
		if !effectiveQuiet {
			PrintError("Failed to initialize importer: %v", err)
		}
		// Exit with code 2 for initialization failures, unless in test mode
		if !testing.Testing() {
			os.Exit(2)
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
	} else if !effectiveQuiet {
		displaySummary(summary, importDryRun)
	}

	// Determine exit code
	totalRejected := summary.Calls.Total.Rejected + summary.SMS.Total.Rejected
	if totalRejected > 0 && !noErrorOnRejects {
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

	// Display calls statistics
	if importFilter == "" || importFilter == "calls" {
		fmt.Println("\nCalls:")
		for year, stats := range summary.Calls.YearStats {
			fmt.Printf("  %d: %d entries (%d new, %d duplicates)\n",
				year, stats.Final, stats.Added, stats.Duplicates)
		}
		if len(summary.Calls.YearStats) > 0 {
			fmt.Printf("  Total: %d entries (%d new, %d duplicates)\n",
				summary.Calls.Total.Final, summary.Calls.Total.Added, summary.Calls.Total.Duplicates)
		}
	}

	// Display SMS statistics
	if importFilter == "" || importFilter == "sms" {
		fmt.Println("SMS:")
		for year, stats := range summary.SMS.YearStats {
			fmt.Printf("  %d: %d entries (%d new, %d duplicates)\n",
				year, stats.Final, stats.Added, stats.Duplicates)
		}
		if len(summary.SMS.YearStats) > 0 {
			fmt.Printf("  Total: %d entries (%d new, %d duplicates)\n",
				summary.SMS.Total.Final, summary.SMS.Total.Added, summary.SMS.Total.Duplicates)
		}
	}

	// Display attachment statistics
	if (importFilter == "" || importFilter == "sms") && summary.Attachments.Total.Total > 0 {
		fmt.Println("Attachments:")
		fmt.Printf("  Total: %d files (%d new, %d duplicates)\n",
			summary.Attachments.Total.Total, summary.Attachments.Total.New,
			summary.Attachments.Total.Duplicates)
	}

	// Display rejection statistics
	totalRejections := 0
	for _, stats := range summary.Rejections {
		totalRejections += stats.Count
	}

	if totalRejections > 0 {
		fmt.Println("Rejections:")
		if importFilter == "" || importFilter == "calls" {
			callsRej := 0
			for _, stats := range summary.Rejections {
				if stats.Count > 0 {
					callsRej += stats.Count
				}
			}
			if callsRej > 0 {
				fmt.Printf("  Calls: %d", callsRej)
				// Show breakdown by reason
				first := true
				for reason, stats := range summary.Rejections {
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
		}

		if importFilter == "" || importFilter == "sms" {
			// TODO: Track SMS-specific rejections separately
			fmt.Printf("  SMS: 0\n")
		}

		fmt.Printf("  Total: %d\n", totalRejections)
	}

	// Display rejection files if any
	if len(summary.RejectionFiles) > 0 {
		fmt.Println("\nRejected entries saved to: rejected/")
	}

	// Display timing
	fmt.Printf("\nFiles processed: %d\n", summary.FilesProcessed)
	fmt.Printf("Time taken: %.1fs\n", summary.Duration.Seconds())
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
			"years": summary.Calls.YearStats,
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
			"years": summary.SMS.YearStats,
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
