package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phillipgreen/mobilecombackup/pkg/attachments"
	"github.com/phillipgreen/mobilecombackup/pkg/autofix"
	"github.com/phillipgreen/mobilecombackup/pkg/calls"
	"github.com/phillipgreen/mobilecombackup/pkg/contacts"
	"github.com/phillipgreen/mobilecombackup/pkg/sms"
	"github.com/phillipgreen/mobilecombackup/pkg/validation"
	"github.com/spf13/cobra"
)

var (
	outputJSON              bool
	removeOrphanAttachments bool
	validateDryRun          bool
	autofixEnabled          bool
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a mobilecombackup repository",
	Long: `Validate a mobilecombackup repository for structure, content, and consistency.

This command performs comprehensive validation including:
- Repository structure verification
- Manifest file validation
- Checksum verification
- Content format validation
- Cross-reference consistency checks`,
	RunE: runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)

	// Local flags
	validateCmd.Flags().BoolVar(&outputJSON, "output-json", false, "Output results in JSON format")
	validateCmd.Flags().BoolVar(&removeOrphanAttachments, "remove-orphan-attachments", false, "Remove orphaned attachment files")
	validateCmd.Flags().BoolVar(&validateDryRun, "dry-run", false, "Show what would be done without making changes")
	validateCmd.Flags().BoolVar(&autofixEnabled, "autofix", false, "Automatically fix safe validation violations")
}

// ValidationResult represents the result of validation
type ValidationResult struct {
	Valid         bool                             `json:"valid"`
	Violations    []validation.ValidationViolation `json:"violations"`
	OrphanRemoval *OrphanRemovalResult             `json:"orphan_removal,omitempty"`
	AutofixReport *autofix.AutofixReport           `json:"autofix_report,omitempty"`
}

// OrphanRemovalResult represents the result of orphan attachment removal
type OrphanRemovalResult struct {
	AttachmentsScanned int             `json:"attachments_scanned"`
	OrphansFound       int             `json:"orphans_found"`
	OrphansRemoved     int             `json:"orphans_removed"`
	BytesFreed         int64           `json:"bytes_freed"`
	RemovalFailures    int             `json:"removal_failures"`
	FailedRemovals     []FailedRemoval `json:"failed_removals,omitempty"`
}

// FailedRemoval represents a single failed removal
type FailedRemoval struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// ProgressReporter provides progress updates during validation
type ProgressReporter interface {
	StartPhase(phase string)
	UpdateProgress(current, total int)
	CompletePhase()
}

// ConsoleProgressReporter implements progress reporting to console
type ConsoleProgressReporter struct {
	quiet   bool
	verbose bool
	phase   string
}

func (r *ConsoleProgressReporter) StartPhase(phase string) {
	r.phase = phase
	if !r.quiet {
		fmt.Printf("Validating %s...", phase)
		if r.verbose {
			fmt.Println()
		}
	}
}

func (r *ConsoleProgressReporter) UpdateProgress(current, total int) {
	if !r.quiet && r.verbose {
		fmt.Printf("  Progress: %d/%d\r", current, total)
	}
}

func (r *ConsoleProgressReporter) CompletePhase() {
	if !r.quiet && !r.verbose {
		fmt.Println(" done")
	} else if !r.quiet && r.verbose {
		fmt.Printf("  Completed %s validation\n", r.phase)
	}
}

// NullProgressReporter discards all progress updates
type NullProgressReporter struct{}

func (r *NullProgressReporter) StartPhase(phase string)           {}
func (r *NullProgressReporter) UpdateProgress(current, total int) {}
func (r *NullProgressReporter) CompletePhase()                    {}

func runValidate(cmd *cobra.Command, args []string) error {
	// Resolve repository root
	repoPath := resolveRepoRoot()

	// Convert to absolute path
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to resolve repository path: %w", err)
	}

	// Check if repository exists
	if _, err := os.Stat(absPath); err != nil {
		if os.IsNotExist(err) {
			PrintError("Repository not found: %s", absPath)
			os.Exit(2)
		}
		return fmt.Errorf("failed to access repository: %w", err)
	}

	// Set up progress reporter
	var reporter ProgressReporter
	if outputJSON || quiet {
		reporter = &NullProgressReporter{}
	} else {
		reporter = &ConsoleProgressReporter{
			quiet:   quiet,
			verbose: verbose,
		}
	}

	// Create readers
	callsReader := calls.NewXMLCallsReader(absPath)
	smsReader := sms.NewXMLSMSReader(absPath)
	attachmentReader := attachments.NewAttachmentManager(absPath)
	contactsReader := contacts.NewContactsManager(absPath)

	// Create validator
	validator := validation.NewRepositoryValidator(
		absPath,
		callsReader,
		smsReader,
		attachmentReader,
		contactsReader,
	)

	// Run validation with progress reporting
	violations, err := validateWithProgress(validator, reporter)
	if err != nil {
		PrintError("Validation failed: %v", err)
		os.Exit(2)
	}

	// Create result
	result := ValidationResult{
		Valid:      len(violations) == 0,
		Violations: violations,
	}

	// Run autofix if requested
	if autofixEnabled {
		autofixReport, err := runAutofixWithProgress(violations, absPath, reporter)
		if err != nil {
			PrintError("Autofix failed: %v", err)
			os.Exit(3)
		}
		result.AutofixReport = autofixReport

		// Update result validity based on remaining violations
		if autofixReport.Summary.FixedCount > 0 {
			// Re-run validation to get current state after fixes
			updatedViolations, err := validateWithProgress(validator, reporter)
			if err != nil {
				PrintError("Post-autofix validation failed: %v", err)
				os.Exit(2)
			}
			result.Violations = updatedViolations
			result.Valid = len(updatedViolations) == 0
		}
	}

	// Run orphan removal if requested
	if removeOrphanAttachments {
		orphanResult, err := removeOrphanAttachmentsWithProgress(smsReader, attachmentReader, reporter)
		if err != nil {
			PrintError("Orphan removal failed: %v", err)
			os.Exit(2)
		}
		result.OrphanRemoval = orphanResult
	}

	// Output results
	if outputJSON {
		if err := outputJSONResult(result); err != nil {
			return fmt.Errorf("failed to output JSON: %w", err)
		}
	} else {
		outputTextResult(result, absPath)
	}

	// Set exit code based on autofix and validation results
	if autofixEnabled && result.AutofixReport != nil {
		// Autofix was run - use autofix exit codes
		if result.AutofixReport.Summary.ErrorCount > 0 {
			os.Exit(2) // Errors occurred during autofix
		} else if !result.Valid {
			os.Exit(1) // Some violations remain after autofix
		}
		// Exit 0 - all violations were fixed successfully
	} else {
		// Standard validation - exit 1 if invalid
		if !result.Valid {
			os.Exit(1)
		}
	}

	return nil
}

func resolveRepoRoot() string {
	// Priority: CLI flag > environment variable > current directory
	if repoRoot != "." {
		return repoRoot
	}

	if envRepo := os.Getenv("MB_REPO_ROOT"); envRepo != "" {
		return envRepo
	}

	return "."
}

func validateWithProgress(validator validation.RepositoryValidator, reporter ProgressReporter) ([]validation.ValidationViolation, error) {
	// For now, just run validation without progress reporting
	// Progress reporting will be added in the validation package in a future enhancement
	reporter.StartPhase("repository")
	report, err := validator.ValidateRepository()
	reporter.CompletePhase()

	if err != nil {
		return nil, err
	}

	return report.Violations, nil
}

func runAutofixWithProgress(violations []validation.ValidationViolation, repoPath string, reporter ProgressReporter) (*autofix.AutofixReport, error) {
	reporter.StartPhase("autofix")
	defer reporter.CompletePhase()

	// Create autofix progress reporter adapter
	autofixReporter := &AutofixProgressReporter{
		baseReporter:    reporter,
		verbose:         verbose,
		operationsTotal: len(violations), // Estimate based on violations count
	}

	// Create autofixer
	autofixer := autofix.NewAutofixer(repoPath, autofixReporter)

	// Set up autofix options
	options := autofix.AutofixOptions{
		DryRun:  validateDryRun,
		Verbose: verbose,
	}

	// Run autofix
	return autofixer.FixViolations(violations, options)
}

// AutofixProgressReporter adapts autofix progress reporting to validation progress reporting
type AutofixProgressReporter struct {
	baseReporter    ProgressReporter
	verbose         bool
	operationCount  int
	operationsTotal int
}

func (r *AutofixProgressReporter) StartOperation(operation string, details string) {
	r.operationCount++

	if !r.verbose {
		// Update progress every 100 operations or if it's the first/last operation
		if r.operationCount%100 == 0 || r.operationCount == 1 || r.operationCount == r.operationsTotal {
			fmt.Printf("  Processing autofix operations... %d/%d\r", r.operationCount, r.operationsTotal)
		}
		return
	}

	// Verbose mode: show detailed operation info
	fmt.Printf("  [%d/%d] Starting %s: %s\n", r.operationCount, r.operationsTotal, operation, details)
}

func (r *AutofixProgressReporter) CompleteOperation(success bool, details string) {
	if !r.verbose {
		// In non-verbose mode, just update the progress counter
		return
	}

	// Verbose mode: show completion status
	status := "✓"
	if !success {
		status = "✗"
	}
	fmt.Printf("    %s %s\n", status, details)
}

func (r *AutofixProgressReporter) ReportProgress(current, total int) {
	// Update the total count if we get more accurate information
	if total > 0 && r.operationsTotal == 0 {
		r.operationsTotal = total
	}

	if !r.verbose && total > 0 {
		// Show progress for large operations
		if current%1000 == 0 || current == total {
			fmt.Printf("  Processing... %d/%d\r", current, total)
		}
	}
}

func removeOrphanAttachmentsWithProgress(smsReader sms.SMSReader, attachmentReader *attachments.AttachmentManager, reporter ProgressReporter) (*OrphanRemovalResult, error) {
	reporter.StartPhase("orphan attachment removal")
	defer reporter.CompletePhase()

	// Get all attachment references from SMS messages
	referencedHashes, err := smsReader.GetAllAttachmentRefs()
	if err != nil {
		return nil, fmt.Errorf("failed to get attachment references: %w", err)
	}

	// Find orphaned attachments
	orphanedAttachments, err := attachmentReader.FindOrphanedAttachments(referencedHashes)
	if err != nil {
		return nil, fmt.Errorf("failed to find orphaned attachments: %w", err)
	}

	// Count total attachments scanned
	totalCount := 0
	err = attachmentReader.StreamAttachments(func(*attachments.Attachment) error {
		totalCount++
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count attachments: %w", err)
	}

	result := &OrphanRemovalResult{
		AttachmentsScanned: totalCount,
		OrphansFound:       len(orphanedAttachments),
		OrphansRemoved:     0,
		BytesFreed:         0,
		RemovalFailures:    0,
		FailedRemovals:     []FailedRemoval{},
	}

	// If dry run, don't actually remove files
	if validateDryRun {
		// Calculate potential bytes freed
		for _, attachment := range orphanedAttachments {
			result.BytesFreed += attachment.Size
		}
		result.OrphansRemoved = len(orphanedAttachments) // Would be removed
		return result, nil
	}

	// Remove orphaned attachments
	repoPath := attachmentReader.GetRepoPath()
	emptyDirs := make(map[string]bool) // Track directories that might become empty

	for _, attachment := range orphanedAttachments {
		fullPath := filepath.Join(repoPath, attachment.Path)

		// Track the directory for potential cleanup
		dir := filepath.Dir(attachment.Path)
		emptyDirs[dir] = true

		// Attempt to remove the file
		if err := os.Remove(fullPath); err != nil {
			result.RemovalFailures++
			result.FailedRemovals = append(result.FailedRemovals, FailedRemoval{
				Path:  attachment.Path,
				Error: err.Error(),
			})
		} else {
			result.OrphansRemoved++
			result.BytesFreed += attachment.Size
		}
	}

	// Clean up empty directories
	for dir := range emptyDirs {
		cleanupEmptyDirectory(filepath.Join(repoPath, dir))
	}

	return result, nil
}

// cleanupEmptyDirectory removes a directory if it's empty
func cleanupEmptyDirectory(dirPath string) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return // Can't read directory, skip cleanup
	}

	if len(entries) == 0 {
		// Directory is empty, try to remove it
		_ = os.Remove(dirPath)
		// Note: We ignore errors here as cleanup is best-effort
	}
}

func outputJSONResult(result ValidationResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func outputTextResult(result ValidationResult, repoPath string) {
	if quiet && result.Valid && result.OrphanRemoval == nil && result.AutofixReport == nil {
		// In quiet mode, only show output if there are violations, orphan removal, or autofix
		return
	}

	if !quiet {
		fmt.Printf("\nValidation Report for: %s\n", repoPath)
		fmt.Println(strings.Repeat("-", 60))
	}

	if result.Valid {
		if !quiet {
			fmt.Println("✓ Repository is valid")
		}
	} else {
		if !quiet {
			fmt.Printf("✗ Found %d violation(s)\n\n", len(result.Violations))
		}

		// Group violations by type
		violationsByType := make(map[string][]validation.ValidationViolation)
		for _, v := range result.Violations {
			violationsByType[string(v.Type)] = append(violationsByType[string(v.Type)], v)
		}

		// Display violations by type
		for vType, violations := range violationsByType {
			if !quiet {
				fmt.Printf("%s (%d):\n", vType, len(violations))
			}

			for _, v := range violations {
				fmt.Print("  ")
				if v.File != "" {
					fmt.Printf("%s: ", v.File)
				}
				fmt.Println(v.Message)
			}

			if !quiet {
				fmt.Println()
			}
		}
	}

	// Display autofix results if performed
	if result.AutofixReport != nil {
		if !quiet {
			fmt.Println()
			if validateDryRun {
				fmt.Println("Autofix Report (dry-run mode):")
			} else {
				fmt.Println("Autofix Report:")
			}
			fmt.Println(strings.Repeat("=", 24))
		}

		// Display fixed violations
		if len(result.AutofixReport.FixedViolations) > 0 {
			if !quiet {
				fmt.Println("\nFixed Violations:")
			}
			for _, fixed := range result.AutofixReport.FixedViolations {
				if validateDryRun {
					fmt.Printf("✓ Would fix: %s\n", fixed.Details)
				} else {
					fmt.Printf("✓ %s\n", fixed.Details)
				}
			}
		}

		// Display skipped violations
		if len(result.AutofixReport.SkippedViolations) > 0 {
			if !quiet {
				fmt.Println("\nRemaining Violations:")
			}
			for _, skipped := range result.AutofixReport.SkippedViolations {
				fmt.Printf("✗ %s: %s (%s)\n", skipped.OriginalViolation.File, skipped.OriginalViolation.Message, skipped.SkipReason)
			}
		}

		// Display errors
		if len(result.AutofixReport.Errors) > 0 {
			if !quiet {
				fmt.Println("\nErrors During Autofix:")
			}
			for _, autofixErr := range result.AutofixReport.Errors {
				fmt.Printf("⚠ Failed %s on %s: %s\n", autofixErr.Operation, autofixErr.File, autofixErr.Error)
			}
		}

		// Display summary
		if !quiet {
			fmt.Printf("\nSummary: %d violations fixed, %d remaining, %d errors\n",
				result.AutofixReport.Summary.FixedCount,
				result.AutofixReport.Summary.SkippedCount,
				result.AutofixReport.Summary.ErrorCount)
		}
	}

	// Display orphan removal results if performed
	if result.OrphanRemoval != nil {
		if !quiet {
			fmt.Println()
			fmt.Println("Orphan attachment removal:")
		}

		if validateDryRun {
			fmt.Printf("  Attachments scanned: %d\n", result.OrphanRemoval.AttachmentsScanned)
			fmt.Printf("  Orphans found: %d\n", result.OrphanRemoval.OrphansFound)
			fmt.Printf("  Would remove: %d (%.1f MB)\n",
				result.OrphanRemoval.OrphansRemoved,
				float64(result.OrphanRemoval.BytesFreed)/1024/1024)
		} else {
			fmt.Printf("  Attachments scanned: %d\n", result.OrphanRemoval.AttachmentsScanned)
			fmt.Printf("  Orphans found: %d\n", result.OrphanRemoval.OrphansFound)
			fmt.Printf("  Orphans removed: %d (%.1f MB freed)\n",
				result.OrphanRemoval.OrphansRemoved,
				float64(result.OrphanRemoval.BytesFreed)/1024/1024)

			if result.OrphanRemoval.RemovalFailures > 0 {
				fmt.Printf("  Removal failures: %d\n", result.OrphanRemoval.RemovalFailures)
				for _, failure := range result.OrphanRemoval.FailedRemovals {
					fmt.Printf("    - %s: %s\n", failure.Path, failure.Error)
				}
			}
		}
	}
}
