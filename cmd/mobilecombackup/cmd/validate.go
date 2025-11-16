package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phillipgreenii/mobilecombackup/pkg/attachments"
	"github.com/phillipgreenii/mobilecombackup/pkg/autofix"
	"github.com/phillipgreenii/mobilecombackup/pkg/calls"
	"github.com/phillipgreenii/mobilecombackup/pkg/contacts"
	"github.com/phillipgreenii/mobilecombackup/pkg/security"
	"github.com/phillipgreenii/mobilecombackup/pkg/sms"
	"github.com/phillipgreenii/mobilecombackup/pkg/validation"
	"github.com/spf13/afero"
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
	validateCmd.Flags().BoolVar(&removeOrphanAttachments, "remove-orphan-attachments", false,
		"Remove orphaned attachment files")
	validateCmd.Flags().BoolVar(&validateDryRun, "dry-run", false, "Show what would be done without making changes")
	validateCmd.Flags().BoolVar(&autofixEnabled, "autofix", false, "Automatically fix safe validation violations")
}

// ValidationResult represents the result of validation
type ValidationResult struct {
	Valid         bool                   `json:"valid"`
	Violations    []validation.Violation `json:"violations"`
	OrphanRemoval *OrphanRemovalResult   `json:"orphan_removal,omitempty"`
	AutofixReport *autofix.Report        `json:"autofix_report,omitempty"`
}

// PathValidator for secure path operations
var pathValidator *security.PathValidator

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

// StartPhase begins a new validation phase
func (r *ConsoleProgressReporter) StartPhase(phase string) {
	r.phase = phase
	if !r.quiet {
		fmt.Printf("Validating %s...", phase)
		if r.verbose {
			fmt.Println()
		}
	}
}

// UpdateProgress reports progress during validation
func (r *ConsoleProgressReporter) UpdateProgress(current, total int) {
	if !r.quiet && r.verbose {
		fmt.Printf("  Progress: %d/%d\r", current, total)
	}
}

// CompletePhase marks the current validation phase as complete
func (r *ConsoleProgressReporter) CompletePhase() {
	if !r.quiet && !r.verbose {
		fmt.Println(" done")
	} else if !r.quiet && r.verbose {
		fmt.Printf("  Completed %s validation\n", r.phase)
	}
}

// NullProgressReporter discards all progress updates
type NullProgressReporter struct{}

// StartPhase is a no-op for the null reporter
func (r *NullProgressReporter) StartPhase(_ string) {}

// UpdateProgress is a no-op for the null reporter
func (r *NullProgressReporter) UpdateProgress(_, _ int) {}

// CompletePhase is a no-op for the null reporter
func (r *NullProgressReporter) CompletePhase() {}

func runValidate(_ *cobra.Command, _ []string) error {
	// Initialize validation environment
	absPath, reporter, err := initializeValidationEnvironment()
	if err != nil {
		return err
	}

	// Set up validation components
	validator := createValidationComponents(absPath)

	// Run initial validation
	result, err := executeInitialValidation(validator, reporter)
	if err != nil {
		PrintError("%v", err)
		os.Exit(2)
	}

	// Process additional features (autofix, orphan removal)
	if err := processAdditionalFeatures(result, validator, absPath, reporter); err != nil {
		return err
	}

	// Output results and handle exit codes
	return finalizeValidationResults(result, absPath)
}

// initializeValidationEnvironment sets up the basic validation environment
func initializeValidationEnvironment() (string, ProgressReporter, error) {
	// Resolve repository root
	repoPath := resolveRepoRoot()

	// Convert to absolute path
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve repository path: %w", err)
	}

	// Check if repository exists
	if err := validateRepositoryExists(absPath); err != nil {
		return "", nil, err
	}

	// Set up progress reporter
	reporter := createProgressReporter()

	return absPath, reporter, nil
}

// validateRepositoryExists checks if the repository path exists and is accessible
func validateRepositoryExists(absPath string) error {
	if _, err := os.Stat(absPath); err != nil {
		if os.IsNotExist(err) {
			PrintError("Repository not found: %s", absPath)
			os.Exit(2)
		}
		return fmt.Errorf("failed to access repository: %w", err)
	}
	return nil
}

// createProgressReporter creates the appropriate progress reporter based on output settings
func createProgressReporter() ProgressReporter {
	if outputJSON || quiet {
		return &NullProgressReporter{}
	}
	return &ConsoleProgressReporter{
		quiet:   quiet,
		verbose: verbose,
	}
}

// createValidationComponents sets up all the validation readers and validator
func createValidationComponents(absPath string) validation.RepositoryValidator {
	// Create readers
	callsReader := calls.NewXMLCallsReader(absPath)
	smsReader := sms.NewXMLSMSReader(absPath)
	attachmentReader := attachments.NewAttachmentManager(absPath, afero.NewOsFs())
	contactsReader := contacts.NewContactsManager(absPath)

	// Create validator
	return validation.NewRepositoryValidator(
		absPath,
		callsReader,
		smsReader,
		attachmentReader,
		contactsReader,
		afero.NewOsFs(),
	)
}

// executeInitialValidation runs the base validation and creates the initial result
func executeInitialValidation(validator validation.RepositoryValidator, reporter ProgressReporter) (*ValidationResult, error) {
	// Run validation with progress reporting
	violations, err := validateWithProgress(validator, reporter)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Create result
	return &ValidationResult{
		Valid:      len(violations) == 0,
		Violations: violations,
	}, nil
}

// processAdditionalFeatures handles autofix and orphan removal if requested
func processAdditionalFeatures(
	result *ValidationResult, validator validation.RepositoryValidator, absPath string, reporter ProgressReporter,
) error {
	// Run autofix if requested
	if autofixEnabled {
		if err := processAutofix(result, validator, absPath, reporter); err != nil {
			return err
		}
	}

	// Run orphan removal if requested
	if removeOrphanAttachments {
		if err := processOrphanRemoval(result, absPath, reporter); err != nil {
			return err
		}
	}

	return nil
}

// processAutofix handles autofix functionality
func processAutofix(
	result *ValidationResult, validator validation.RepositoryValidator, absPath string, reporter ProgressReporter,
) error {
	autofixReport, err := runAutofixWithProgress(result.Violations, absPath, reporter)
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

	return nil
}

// processOrphanRemoval handles orphan attachment removal
func processOrphanRemoval(result *ValidationResult, absPath string, reporter ProgressReporter) error {
	smsReader := sms.NewXMLSMSReader(absPath)
	attachmentReader := attachments.NewAttachmentManager(absPath, afero.NewOsFs())

	orphanResult, err := removeOrphanAttachmentsWithProgress(smsReader, attachmentReader, reporter)
	if err != nil {
		PrintError("Orphan removal failed: %v", err)
		os.Exit(2)
	}
	result.OrphanRemoval = orphanResult

	return nil
}

// finalizeValidationResults outputs results and sets appropriate exit codes
func finalizeValidationResults(result *ValidationResult, absPath string) error {
	// Output results
	if outputJSON {
		if err := outputJSONResult(*result); err != nil {
			return fmt.Errorf("failed to output JSON: %w", err)
		}
	} else {
		outputTextResult(*result, absPath)
	}

	// Set exit code based on autofix and validation results
	return handleValidationExitCodes(result)
}

// handleValidationExitCodes sets the appropriate exit code based on validation results
func handleValidationExitCodes(result *ValidationResult) error {
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

func validateWithProgress(
	validator validation.RepositoryValidator,
	reporter ProgressReporter,
) ([]validation.Violation, error) {
	// For now, just run validation without progress reporting
	// Progress reporting will be added in the validation package in a future enhancement
	reporter.StartPhase("repository")
	report, err := validator.ValidateRepositoryContext(context.Background())
	reporter.CompletePhase()

	if err != nil {
		return nil, err
	}

	return report.Violations, nil
}

func runAutofixWithProgress(
	violations []validation.Violation,
	repoPath string,
	reporter ProgressReporter,
) (*autofix.Report, error) {
	reporter.StartPhase("autofix")
	defer reporter.CompletePhase()

	// Create autofix progress reporter adapter
	autofixReporter := &AutofixProgressReporter{
		baseReporter:    reporter,
		verbose:         verbose,
		operationsTotal: len(violations), // Estimate based on violations count
	}

	// Create autofixer
	autofixer := autofix.NewAutofixer(repoPath, autofixReporter, afero.NewOsFs())

	// Set up autofix options
	options := autofix.Options{
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

// StartOperation reports the start of an autofix operation
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

// CompleteOperation reports the completion of an autofix operation
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

// ReportProgress reports overall progress of autofix operations
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

func removeOrphanAttachmentsWithProgress(
	smsReader sms.Reader,
	attachmentReader *attachments.AttachmentManager,
	reporter ProgressReporter,
) (*OrphanRemovalResult, error) {
	reporter.StartPhase("orphan attachment removal")
	defer reporter.CompletePhase()

	orphanedAttachments, totalCount, err := findOrphanedAttachments(smsReader, attachmentReader)
	if err != nil {
		return nil, err
	}

	result := createOrphanRemovalResult(totalCount, orphanedAttachments)

	if validateDryRun {
		return handleDryRunOrphanRemoval(result, orphanedAttachments), nil
	}

	return executeOrphanRemoval(result, orphanedAttachments, attachmentReader), nil
}

// findOrphanedAttachments finds and returns orphaned attachments
func findOrphanedAttachments(
	smsReader sms.Reader, attachmentReader *attachments.AttachmentManager,
) ([]*attachments.Attachment, int, error) {
	// Get all attachment references from SMS messages
	referencedHashes, err := smsReader.GetAllAttachmentRefs()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get attachment references: %w", err)
	}

	// Find orphaned attachments
	orphanedAttachments, err := attachmentReader.FindOrphanedAttachments(referencedHashes)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find orphaned attachments: %w", err)
	}

	// Count total attachments scanned
	totalCount := 0
	err = attachmentReader.StreamAttachments(func(*attachments.Attachment) error {
		totalCount++
		return nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count attachments: %w", err)
	}

	return orphanedAttachments, totalCount, nil
}

// createOrphanRemovalResult creates an initial orphan removal result
func createOrphanRemovalResult(totalCount int, orphanedAttachments []*attachments.Attachment) *OrphanRemovalResult {
	return &OrphanRemovalResult{
		AttachmentsScanned: totalCount,
		OrphansFound:       len(orphanedAttachments),
		OrphansRemoved:     0,
		BytesFreed:         0,
		RemovalFailures:    0,
		FailedRemovals:     []FailedRemoval{},
	}
}

// handleDryRunOrphanRemoval handles dry run mode for orphan removal
func handleDryRunOrphanRemoval(
	result *OrphanRemovalResult, orphanedAttachments []*attachments.Attachment,
) *OrphanRemovalResult {
	// Calculate potential bytes freed
	for _, attachment := range orphanedAttachments {
		result.BytesFreed += attachment.Size
	}
	result.OrphansRemoved = len(orphanedAttachments) // Would be removed
	return result
}

// executeOrphanRemoval executes the actual removal of orphaned attachments
func executeOrphanRemoval(
	result *OrphanRemovalResult,
	orphanedAttachments []*attachments.Attachment,
	attachmentReader *attachments.AttachmentManager,
) *OrphanRemovalResult {
	repoPath := attachmentReader.GetRepoPath()
	emptyDirs := make(map[string]bool) // Track directories that might become empty

	// SECURITY FIX: Initialize path validator for secure directory operations
	if pathValidator == nil {
		pathValidator = security.NewPathValidator(repoPath)
	}

	for _, attachment := range orphanedAttachments {
		removeOrphanedAttachment(attachment, repoPath, result, emptyDirs)
	}

	// Clean up empty directories - with path validation
	for dir := range emptyDirs {
		// SECURITY FIX: Validate directory path before cleanup
		safeDirPath, err := pathValidator.JoinAndValidate(dir)
		if err != nil {
			// Skip cleanup for invalid directory paths
			continue
		}
		cleanupEmptyDirectory(safeDirPath)
	}

	return result
}

// removeOrphanedAttachment removes a single orphaned attachment
func removeOrphanedAttachment(
	attachment *attachments.Attachment,
	repoPath string,
	result *OrphanRemovalResult,
	emptyDirs map[string]bool,
) {
	// SECURITY FIX: Use PathValidator to prevent path traversal attacks
	// Initialize path validator if not already done
	if pathValidator == nil {
		pathValidator = security.NewPathValidator(repoPath)
	}

	// Securely construct the full path for attachment removal
	// Since attachment paths come from AttachmentManager (trusted source),
	// we can safely join them with the repo path
	var fullPath string

	if filepath.IsAbs(attachment.Path) {
		fullPath = attachment.Path
	} else {
		fullPath = filepath.Join(repoPath, attachment.Path)
	}

	// Validate that the final path is within the repository bounds
	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		result.RemovalFailures++
		result.FailedRemovals = append(result.FailedRemovals, FailedRemoval{
			Path:  attachment.Path,
			Error: fmt.Sprintf("failed to resolve repo path: %v", err),
		})
		return
	}

	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		result.RemovalFailures++
		result.FailedRemovals = append(result.FailedRemovals, FailedRemoval{
			Path:  attachment.Path,
			Error: fmt.Sprintf("failed to resolve attachment path: %v", err),
		})
		return
	}

	// Ensure the path is within repository boundaries
	if !strings.HasPrefix(absFullPath+string(filepath.Separator), absRepoPath+string(filepath.Separator)) {
		result.RemovalFailures++
		result.FailedRemovals = append(result.FailedRemovals, FailedRemoval{
			Path:  attachment.Path,
			Error: fmt.Sprintf("path %s is outside repository %s", absFullPath, absRepoPath),
		})
		return
	}

	// Track the directory for potential cleanup
	dir := filepath.Dir(attachment.Path)
	emptyDirs[dir] = true

	// Attempt to remove the file
	if err := os.Remove(absFullPath); err != nil {
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
		// Don't return early - still need to display orphan removal results
	} else {
		// Handle invalid repository
		if !quiet {
			fmt.Printf("✗ Found %d violation(s)\n\n", len(result.Violations))
		}
		displayViolationsByType(result.Violations)
	}

	// Display autofix results if performed
	if result.AutofixReport != nil {
		displayAutofixReport(result.AutofixReport)
	}

	// Display orphan removal results if performed
	if result.OrphanRemoval != nil {
		displayOrphanRemovalResults(result.OrphanRemoval, validateDryRun, quiet)
	}
}

// displayOrphanRemovalResults displays the results of orphan attachment removal
func displayOrphanRemovalResults(orphanRemoval *OrphanRemovalResult, isDryRun bool, quiet bool) {
	if orphanRemoval == nil {
		return
	}

	if !quiet {
		fmt.Println()
		fmt.Println("Orphan attachment removal:")
	}

	if isDryRun {
		displayOrphanRemovalDryRun(orphanRemoval, quiet)
	} else {
		displayOrphanRemovalActual(orphanRemoval, quiet)
	}
}

// displayOrphanRemovalDryRun displays dry-run results for orphan removal
func displayOrphanRemovalDryRun(orphanRemoval *OrphanRemovalResult, quiet bool) {
	if !quiet {
		fmt.Printf("  Attachments scanned: %d\n", orphanRemoval.AttachmentsScanned)
		fmt.Printf("  Orphans found: %d\n", orphanRemoval.OrphansFound)
	}
	fmt.Printf("  Would remove: %d (%.1f MB)\n",
		orphanRemoval.OrphansRemoved,
		float64(orphanRemoval.BytesFreed)/1024/1024)
}

// displayOrphanRemovalActual displays actual results for orphan removal
func displayOrphanRemovalActual(orphanRemoval *OrphanRemovalResult, quiet bool) {
	if !quiet {
		fmt.Printf("  Attachments scanned: %d\n", orphanRemoval.AttachmentsScanned)
		fmt.Printf("  Orphans found: %d\n", orphanRemoval.OrphansFound)
	}
	fmt.Printf("  Orphans removed: %d (%.1f MB freed)\n",
		orphanRemoval.OrphansRemoved,
		float64(orphanRemoval.BytesFreed)/1024/1024)

	if orphanRemoval.RemovalFailures > 0 {
		fmt.Printf("  Removal failures: %d\n", orphanRemoval.RemovalFailures)
		for _, failure := range orphanRemoval.FailedRemovals {
			fmt.Printf("    - %s: %s\n", failure.Path, failure.Error)
		}
	}
}

// displayViolationsByType groups and displays violations by type
func displayViolationsByType(violations []validation.Violation) {
	// Group violations by type
	violationsByType := make(map[string][]validation.Violation)
	for _, v := range violations {
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

// displayAutofixReport displays the autofix report results
func displayAutofixReport(report *autofix.Report) {
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
	if len(report.FixedViolations) > 0 {
		if !quiet {
			fmt.Println("\nFixed Violations:")
		}
		for _, fixed := range report.FixedViolations {
			if validateDryRun {
				fmt.Printf("✓ Would fix: %s\n", fixed.Details)
			} else {
				fmt.Printf("✓ %s\n", fixed.Details)
			}
		}
	}

	// Display skipped violations
	if len(report.SkippedViolations) > 0 {
		if !quiet {
			fmt.Println("\nRemaining Violations:")
		}
		for _, skipped := range report.SkippedViolations {
			fmt.Printf("✗ %s: %s (%s)\n", skipped.OriginalViolation.File,
				skipped.OriginalViolation.Message, skipped.SkipReason)
		}
	}

	// Display errors
	if len(report.Errors) > 0 {
		if !quiet {
			fmt.Println("\nErrors During Autofix:")
		}
		for _, autofixErr := range report.Errors {
			fmt.Printf("⚠ Failed %s on %s: %s\n", autofixErr.Operation,
				autofixErr.File, autofixErr.Error)
		}
	}

	// Display summary
	if !quiet {
		fmt.Printf("\nSummary: %d violations fixed, %d remaining, %d errors\n",
			report.Summary.FixedCount,
			report.Summary.SkippedCount,
			report.Summary.ErrorCount)
	}
}
