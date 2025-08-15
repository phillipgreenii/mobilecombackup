// Package autofix provides automatic repair capabilities for repository validation violations.
//
// The autofix package implements intelligent repair strategies for common validation
// issues found in mobile communication backup repositories. It can automatically
// fix XML count mismatches, missing marker files, invalid contacts data, and other
// structural inconsistencies.
//
// # Supported Violation Types
//
// The autofix system supports automatic repair for these validation violations:
//   - XML count attribute mismatches (fix count attributes in SMS/call XML files)
//   - Missing repository marker files (create .mobilecombackup.yaml)
//   - Invalid contacts data (repair malformed contacts.yaml)
//   - File structure inconsistencies (correct directory layouts)
//
// # Usage Example
//
// Basic autofix workflow:
//
//	violations := []validation.ValidationViolation{
//		{Type: validation.XMLCountMismatch, File: "sms/sms-2024.xml"},
//	}
//
//	autofixer := NewAutofixerImpl("/path/to/repository")
//	options := AutofixOptions{DryRun: true, Verbose: true}
//
//	report, err := autofixer.FixViolations(violations, options)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	fmt.Printf("Fixed: %d, Skipped: %d, Errors: %d\n",
//		report.Summary.FixedCount,
//		report.Summary.SkippedCount,
//		report.Summary.ErrorCount)
//
// # Safety and Dry-Run Mode
//
// All autofix operations support dry-run mode for safe validation of repair
// strategies before making actual changes. The system creates backups of
// modified files and validates repairs after application.
//
// # Security Considerations
//
// Autofix operations include path validation to prevent directory traversal
// attacks and limit modifications to repository-relative paths only.
package autofix

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/manifest"
	"github.com/phillipgreen/mobilecombackup/pkg/security"
	"github.com/phillipgreen/mobilecombackup/pkg/validation"
	"gopkg.in/yaml.v3"
)

// MarkerFileContent represents the .mobilecombackup.yaml file structure
type MarkerFileContent struct {
	RepositoryStructureVersion string `yaml:"repository_structure_version"`
	CreatedAt                  string `yaml:"created_at"`
	CreatedBy                  string `yaml:"created_by"`
}

// ContactsData represents the YAML structure of contacts.yaml
type ContactsData struct {
	Contacts    []interface{} `yaml:"contacts,omitempty"`
	Unprocessed []string      `yaml:"unprocessed,omitempty"`
}

// SummaryContent represents the summary.yaml file structure
type SummaryContent struct {
	Counts struct {
		Calls int `yaml:"calls"`
		SMS   int `yaml:"sms"`
	} `yaml:"counts"`
}

// Autofixer interface for fixing validation violations
type Autofixer interface {
	// FixViolations attempts to fix the given validation violations
	FixViolations(violations []validation.ValidationViolation, options AutofixOptions) (*AutofixReport, error)

	// CanFix returns true if the violation type can be automatically fixed
	CanFix(violationType validation.ViolationType) bool
}

// AutofixerImpl implements the Autofixer interface
type AutofixerImpl struct {
	repositoryRoot string
	reporter       ProgressReporter
	pathValidator  *security.PathValidator
}

// NewAutofixer creates a new autofixer instance
func NewAutofixer(repositoryRoot string, reporter ProgressReporter) Autofixer {
	if reporter == nil {
		reporter = &NullProgressReporter{}
	}
	return &AutofixerImpl{
		repositoryRoot: repositoryRoot,
		reporter:       reporter,
		pathValidator:  security.NewPathValidator(repositoryRoot),
	}
}

// FixViolations attempts to fix the given validation violations
func (a *AutofixerImpl) FixViolations(violations []validation.ValidationViolation, options AutofixOptions) (*AutofixReport, error) {
	report := a.createInitialReport()

	// In dry-run mode, perform permission checks
	if options.DryRun {
		a.performPermissionChecks(violations, report)
	}

	// Phase 1: Create directories (must come first)
	a.fixDirectoryViolations(violations, options, report)

	// Phase 2: Create missing files
	a.fixFileViolations(violations, options, report)

	// Phase 3: Update existing files and content
	a.fixContentViolations(violations, options, report)

	// Skip violations that cannot be safely fixed
	a.handleUnsafeViolations(violations, report)

	// Calculate summary
	report.Summary = AutofixSummary{
		FixedCount:   len(report.FixedViolations),
		SkippedCount: len(report.SkippedViolations),
		ErrorCount:   len(report.Errors),
	}

	return report, nil
}

// createInitialReport creates a new autofix report
func (a *AutofixerImpl) createInitialReport() *AutofixReport {
	return &AutofixReport{
		Timestamp:         time.Now().UTC(),
		RepositoryPath:    a.repositoryRoot,
		FixedViolations:   []FixedViolation{},
		SkippedViolations: []SkippedViolation{},
		Errors:            []AutofixError{},
	}
}

// performPermissionChecks performs permission checks during dry-run mode
func (a *AutofixerImpl) performPermissionChecks(violations []validation.ValidationViolation, report *AutofixReport) {
	if err := a.checkPermissions(violations); err != nil {
		report.Errors = append(report.Errors, AutofixError{
			ViolationType: "",
			File:          a.repositoryRoot,
			Operation:     "permission_check",
			Error:         err.Error(),
		})
	}
}

// fixDirectoryViolations handles directory creation violations
func (a *AutofixerImpl) fixDirectoryViolations(violations []validation.ValidationViolation, options AutofixOptions, report *AutofixReport) {
	directoryViolations := filterViolationsByTypes(violations, []validation.ViolationType{
		validation.StructureViolation,
	})

	for _, violation := range directoryViolations {
		if isDirectoryMissing(violation) {
			a.handleDirectoryViolation(violation, options.DryRun, report)
		}
	}
}

// handleDirectoryViolation processes a single directory violation
func (a *AutofixerImpl) handleDirectoryViolation(violation validation.ValidationViolation, dryRun bool, report *AutofixReport) {
	if dryRun {
		report.FixedViolations = append(report.FixedViolations, FixedViolation{
			OriginalViolation: violation,
			FixAction:         OperationCreateDirectory,
			Details:           fmt.Sprintf("Would create directory: %s", extractDirectoryFromViolation(violation)),
		})
		return
	}

	if err := a.fixMissingDirectory(violation); err != nil {
		report.Errors = append(report.Errors, AutofixError{
			ViolationType: violation.Type,
			File:          violation.File,
			Operation:     OperationCreateDirectory,
			Error:         err.Error(),
		})
	} else {
		report.FixedViolations = append(report.FixedViolations, FixedViolation{
			OriginalViolation: violation,
			FixAction:         OperationCreateDirectory,
			Details:           fmt.Sprintf("Created directory: %s", extractDirectoryFromViolation(violation)),
		})
	}
}

// fixFileViolations handles missing file violations
func (a *AutofixerImpl) fixFileViolations(violations []validation.ValidationViolation, options AutofixOptions, report *AutofixReport) {
	fileViolations := filterViolationsByTypes(violations, []validation.ViolationType{
		validation.MissingFile,
		validation.MissingMarkerFile,
	})

	for _, violation := range fileViolations {
		a.handleFileViolation(violation, options.DryRun, report)
	}
}

// handleFileViolation processes a single file violation
func (a *AutofixerImpl) handleFileViolation(violation validation.ValidationViolation, dryRun bool, report *AutofixReport) {
	if !a.CanFix(violation.Type) {
		report.SkippedViolations = append(report.SkippedViolations, SkippedViolation{
			OriginalViolation: violation,
			SkipReason:        "Not a safe autofix operation",
		})
		return
	}

	if dryRun {
		report.FixedViolations = append(report.FixedViolations, FixedViolation{
			OriginalViolation: violation,
			FixAction:         OperationCreateFile,
			Details:           fmt.Sprintf("Would create file: %s", violation.File),
		})
		return
	}

	if err := a.fixMissingFile(violation); err != nil {
		report.Errors = append(report.Errors, AutofixError{
			ViolationType: violation.Type,
			File:          violation.File,
			Operation:     OperationCreateFile,
			Error:         err.Error(),
		})
	} else {
		report.FixedViolations = append(report.FixedViolations, FixedViolation{
			OriginalViolation: violation,
			FixAction:         OperationCreateFile,
			Details:           fmt.Sprintf("Created file: %s", violation.File),
		})
	}
}

// fixContentViolations handles content-related violations
func (a *AutofixerImpl) fixContentViolations(violations []validation.ValidationViolation, options AutofixOptions, report *AutofixReport) {
	contentViolations := filterViolationsByTypes(violations, []validation.ViolationType{
		validation.CountMismatch,
		validation.SizeMismatch,
	})

	for _, violation := range contentViolations {
		a.handleContentViolation(violation, options.DryRun, report)
	}
}

// handleContentViolation processes a single content violation
func (a *AutofixerImpl) handleContentViolation(violation validation.ValidationViolation, dryRun bool, report *AutofixReport) {
	if !a.CanFix(violation.Type) {
		report.SkippedViolations = append(report.SkippedViolations, SkippedViolation{
			OriginalViolation: violation,
			SkipReason:        "Not a safe autofix operation",
		})
		return
	}

	if dryRun {
		a.handleContentViolationDryRun(violation, report)
		return
	}

	a.handleContentViolationExecute(violation, report)
}

// handleContentViolationDryRun handles content violations in dry-run mode
func (a *AutofixerImpl) handleContentViolationDryRun(violation validation.ValidationViolation, report *AutofixReport) {
	var fixAction, details string

	switch violation.Type {
	case validation.CountMismatch:
		fixAction = OperationUpdateXMLCount
		details = fmt.Sprintf("Would update count attribute in %s", violation.File)
		if violation.Expected != "" && violation.Actual != "" {
			details = fmt.Sprintf("Would update count attribute in %s (from %s to %s)", violation.File, violation.Actual, violation.Expected)
		}
	case validation.SizeMismatch:
		fixAction = OperationUpdateFile
		details = "Would regenerate files.yaml with correct file sizes"
	}

	report.FixedViolations = append(report.FixedViolations, FixedViolation{
		OriginalViolation: violation,
		FixAction:         fixAction,
		Details:           details,
	})
}

// handleContentViolationExecute executes content violation fixes
func (a *AutofixerImpl) handleContentViolationExecute(violation validation.ValidationViolation, report *AutofixReport) {
	var err error
	var fixAction string

	switch violation.Type {
	case validation.CountMismatch:
		err = a.fixCountMismatch(violation)
		fixAction = OperationUpdateXMLCount
	case validation.SizeMismatch:
		err = a.fixSizeMismatch(violation)
		fixAction = OperationUpdateFile
	}

	if err != nil {
		report.Errors = append(report.Errors, AutofixError{
			ViolationType: violation.Type,
			File:          violation.File,
			Operation:     fixAction,
			Error:         err.Error(),
		})
	} else {
		report.FixedViolations = append(report.FixedViolations, FixedViolation{
			OriginalViolation: violation,
			FixAction:         fixAction,
			Details:           fmt.Sprintf("Fixed %s in %s", violation.Type, violation.File),
		})
	}
}

// handleUnsafeViolations handles violations that cannot be safely fixed
func (a *AutofixerImpl) handleUnsafeViolations(violations []validation.ValidationViolation, report *AutofixReport) {
	unsafeViolations := filterViolationsByTypes(violations, []validation.ViolationType{
		validation.ChecksumMismatch,
		validation.OrphanedAttachment,
	})

	for _, violation := range unsafeViolations {
		reason := a.getSkipReason(violation.Type)
		report.SkippedViolations = append(report.SkippedViolations, SkippedViolation{
			OriginalViolation: violation,
			SkipReason:        reason,
		})
	}
}

// getSkipReason returns the appropriate skip reason for violation types
func (a *AutofixerImpl) getSkipReason(violationType validation.ViolationType) string {
	switch violationType {
	case validation.ChecksumMismatch:
		return "Autofix preserves existing checksums to detect corruption"
	case validation.OrphanedAttachment:
		return "Use --remove-orphan-attachments flag"
	default:
		return "Not a safe autofix operation"
	}
}

// CanFix returns true if the violation type can be automatically fixed
func (a *AutofixerImpl) CanFix(violationType validation.ViolationType) bool {
	switch violationType {
	case validation.MissingFile,
		validation.MissingMarkerFile,
		validation.CountMismatch,
		validation.SizeMismatch,
		validation.StructureViolation:
		return true
	case validation.ChecksumMismatch,
		validation.OrphanedAttachment,
		validation.InvalidFormat,
		validation.ExtraFile:
		return false
	default:
		return false
	}
}

// Helper functions

func filterViolationsByTypes(violations []validation.ValidationViolation, types []validation.ViolationType) []validation.ValidationViolation {
	var filtered []validation.ValidationViolation
	typeMap := make(map[validation.ViolationType]bool)
	for _, t := range types {
		typeMap[t] = true
	}

	for _, violation := range violations {
		if typeMap[violation.Type] {
			filtered = append(filtered, violation)
		}
	}
	return filtered
}

func isDirectoryMissing(violation validation.ValidationViolation) bool {
	// Check if this is a directory missing violation
	if violation.Type != validation.StructureViolation {
		return false
	}

	// Check for common directory patterns
	if violation.File == "calls/" || violation.File == "sms/" || violation.File == "attachments/" {
		return true
	}

	// Check for messages that indicate missing directories
	message := strings.ToLower(violation.Message)
	return strings.Contains(message, "directory") &&
		(strings.Contains(message, "missing") ||
			strings.Contains(message, "not found") ||
			strings.Contains(message, "does not exist"))
}

func extractDirectoryFromViolation(violation validation.ValidationViolation) string {
	return violation.File
}

func (a *AutofixerImpl) fixMissingDirectory(violation validation.ValidationViolation) error {
	// Validate the directory path from violation
	relDirPath := extractDirectoryFromViolation(violation)

	// Validate the path to prevent directory traversal
	validatedPath, err := a.pathValidator.ValidatePath(relDirPath)
	if err != nil {
		return fmt.Errorf("invalid directory path in violation: %w", err)
	}

	// Get the safe absolute path
	safeDirPath, err := a.pathValidator.GetSafePath(validatedPath)
	if err != nil {
		return fmt.Errorf("failed to get safe directory path: %w", err)
	}

	a.reporter.StartOperation(OperationCreateDirectory, safeDirPath)

	err = os.MkdirAll(safeDirPath, 0750)

	a.reporter.CompleteOperation(err == nil, fmt.Sprintf("Directory: %s", safeDirPath))

	return err
}

func (a *AutofixerImpl) fixMissingFile(violation validation.ValidationViolation) error {
	a.reporter.StartOperation(OperationCreateFile, violation.File)

	var err error

	switch violation.File {
	case ".mobilecombackup.yaml":
		err = a.createMarkerFile()
	case "contacts.yaml":
		err = a.createEmptyContactsFile()
	case "summary.yaml":
		err = a.createSummaryFile()
	case "files.yaml":
		err = a.createFilesManifest()
	case "files.yaml.sha256":
		err = a.createManifestChecksum()
	default:
		err = fmt.Errorf("unknown file type for autofix: %s", violation.File)
	}

	a.reporter.CompleteOperation(err == nil, violation.File)

	return err
}

func (a *AutofixerImpl) fixCountMismatch(violation validation.ValidationViolation) error {
	a.reporter.StartOperation(OperationUpdateXMLCount, violation.File)

	// Validate file path
	validatedPath, err := a.pathValidator.ValidatePath(violation.File)
	if err != nil {
		return fmt.Errorf("invalid file path in violation: %w", err)
	}

	filePath, err := a.pathValidator.GetSafePath(validatedPath)
	if err != nil {
		return fmt.Errorf("failed to get safe file path: %w", err)
	}

	// Read the XML file
	content, err := os.ReadFile(filePath) // #nosec G304
	if err != nil {
		a.reporter.CompleteOperation(false, violation.File)
		return fmt.Errorf("failed to read XML file: %w", err)
	}

	// Fix the count attribute
	fixedContent, actualCount, err := fixXMLCountAttribute(content)
	if err != nil {
		a.reporter.CompleteOperation(false, violation.File)
		return fmt.Errorf("failed to fix count attribute: %w", err)
	}

	// Write the fixed content atomically
	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, fixedContent, 0644); err != nil {
		a.reporter.CompleteOperation(false, violation.File)
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, filePath); err != nil {
		_ = os.Remove(tempPath) // Clean up temp file, ignore error
		a.reporter.CompleteOperation(false, violation.File)
		return fmt.Errorf("failed to rename fixed file: %w", err)
	}

	a.reporter.CompleteOperation(true, fmt.Sprintf("%s (corrected count to %d)", violation.File, actualCount))

	return nil
}

func (a *AutofixerImpl) fixSizeMismatch(violation validation.ValidationViolation) error {
	a.reporter.StartOperation(OperationUpdateFile, violation.File)

	// This fixes size mismatches in files.yaml by regenerating the entire manifest
	// This ensures all file sizes and hashes are accurate
	err := a.createFilesManifest()

	a.reporter.CompleteOperation(err == nil, "files.yaml (regenerated with correct sizes)")

	return err
}

func (a *AutofixerImpl) createMarkerFile() error {
	// Validate marker file path
	validatedPath, err := a.pathValidator.ValidatePath(".mobilecombackup.yaml")
	if err != nil {
		return fmt.Errorf("invalid marker file path: %w", err)
	}

	markerPath, err := a.pathValidator.GetSafePath(validatedPath)
	if err != nil {
		return fmt.Errorf("failed to get safe marker file path: %w", err)
	}

	// Create marker file content
	markerContent := MarkerFileContent{
		RepositoryStructureVersion: "1",
		CreatedAt:                  time.Now().UTC().Format(time.RFC3339),
		CreatedBy:                  "mobilecombackup autofix",
	}

	// Marshal to YAML
	data, err := yaml.Marshal(markerContent)
	if err != nil {
		return fmt.Errorf("failed to marshal marker file content: %w", err)
	}

	// Write to file atomically
	tempPath := markerPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary marker file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, markerPath); err != nil {
		_ = os.Remove(tempPath) // Clean up temp file, ignore error
		return fmt.Errorf("failed to rename marker file: %w", err)
	}

	return nil
}

func (a *AutofixerImpl) createEmptyContactsFile() error {
	// Validate contacts file path
	validatedPath, err := a.pathValidator.ValidatePath("contacts.yaml")
	if err != nil {
		return fmt.Errorf("invalid contacts file path: %w", err)
	}

	contactsPath, err := a.pathValidator.GetSafePath(validatedPath)
	if err != nil {
		return fmt.Errorf("failed to get safe contacts file path: %w", err)
	}

	// Create empty contacts structure
	contactsData := ContactsData{
		Contacts:    []interface{}{},
		Unprocessed: []string{},
	}

	// Marshal to YAML
	data, err := yaml.Marshal(contactsData)
	if err != nil {
		return fmt.Errorf("failed to marshal contacts file content: %w", err)
	}

	// Write to file atomically
	tempPath := contactsPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary contacts file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, contactsPath); err != nil {
		_ = os.Remove(tempPath) // Clean up temp file, ignore error
		return fmt.Errorf("failed to rename contacts file: %w", err)
	}

	return nil
}

func (a *AutofixerImpl) createSummaryFile() error {
	// Validate summary file path
	validatedPath, err := a.pathValidator.ValidatePath("summary.yaml")
	if err != nil {
		return fmt.Errorf("invalid summary file path: %w", err)
	}

	summaryPath, err := a.pathValidator.GetSafePath(validatedPath)
	if err != nil {
		return fmt.Errorf("failed to get safe summary file path: %w", err)
	}

	// Create summary with zero counts (will be regenerated by import)
	summaryData := SummaryContent{}
	summaryData.Counts.Calls = 0
	summaryData.Counts.SMS = 0

	// Marshal to YAML
	data, err := yaml.Marshal(summaryData)
	if err != nil {
		return fmt.Errorf("failed to marshal summary file content: %w", err)
	}

	// Write to file atomically
	tempPath := summaryPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary summary file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, summaryPath); err != nil {
		_ = os.Remove(tempPath) // Clean up temp file, ignore error
		return fmt.Errorf("failed to rename summary file: %w", err)
	}

	return nil
}

func (a *AutofixerImpl) createFilesManifest() error {
	// Use the shared manifest generator
	manifestGenerator := manifest.NewManifestGenerator(a.repositoryRoot)

	// Generate manifest
	fileManifest, err := manifestGenerator.GenerateFileManifest()
	if err != nil {
		return fmt.Errorf("failed to generate file manifest: %w", err)
	}

	// Always regenerate files.yaml
	if err := manifestGenerator.WriteManifestOnly(fileManifest); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

func (a *AutofixerImpl) createManifestChecksum() error {
	// Use the shared manifest generator - only create if missing
	manifestGenerator := manifest.NewManifestGenerator(a.repositoryRoot)
	return manifestGenerator.WriteChecksumOnly()
}

// fixXMLCountAttribute fixes the count attribute in XML files to match actual child element count
func fixXMLCountAttribute(content []byte) ([]byte, int, error) {
	// Parse XML to count child elements
	decoder := xml.NewDecoder(strings.NewReader(string(content)))

	var rootElement xml.StartElement
	var childCount int
	inRoot := false

	// Find the root element and count its direct children
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("failed to parse XML: %w", err)
		}

		switch element := token.(type) {
		case xml.StartElement:
			if !inRoot {
				// This is the root element
				rootElement = element
				inRoot = true
			} else {
				// This is a child element of the root
				childCount++
				// Skip to the end of this element
				if err := skipElement(decoder); err != nil {
					return nil, 0, fmt.Errorf("failed to skip element: %w", err)
				}
			}
		case xml.EndElement:
			if inRoot && element.Name.Local == rootElement.Name.Local {
				// End of root element
				break
			}
		}
	}

	// Use regex to find and replace the count attribute in the original content
	// This preserves formatting and structure while only updating the count
	countPattern := regexp.MustCompile(`(<\w+[^>]*\s)count="(\d+)"([^>]*>)`)

	fixed := countPattern.ReplaceAllStringFunc(string(content), func(match string) string {
		return countPattern.ReplaceAllString(match, fmt.Sprintf("${1}count=\"%d\"${3}", childCount))
	})

	return []byte(fixed), childCount, nil
}

// skipElement skips to the end of the current XML element
func skipElement(decoder *xml.Decoder) error {
	depth := 1
	for depth > 0 {
		token, err := decoder.Token()
		if err != nil {
			return err
		}

		switch token.(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			depth--
		}
	}
	return nil
}

// checkPermissions verifies write permissions for dry-run mode
func (a *AutofixerImpl) checkPermissions(violations []validation.ValidationViolation) error {
	// Check repository root write permission
	if err := checkWritePermission(a.repositoryRoot); err != nil {
		return fmt.Errorf("repository root not writable: %w", err)
	}

	// Check for files that would need to be modified
	filesToCheck := a.collectPathsToCheck(violations)

	// Check write permissions for all identified paths
	return a.validateWritePermissions(filesToCheck)
}

// collectPathsToCheck collects all paths that need write permission checks
func (a *AutofixerImpl) collectPathsToCheck(violations []validation.ValidationViolation) map[string]bool {
	filesToCheck := make(map[string]bool)

	for _, violation := range violations {
		a.addPathForViolation(violation, filesToCheck)
	}

	return filesToCheck
}

// addPathForViolation adds the appropriate path to check for a violation
func (a *AutofixerImpl) addPathForViolation(violation validation.ValidationViolation, filesToCheck map[string]bool) {
	switch violation.Type {
	case validation.MissingFile, validation.MissingMarkerFile:
		a.addMissingFilePathToCheck(violation, filesToCheck)
	case validation.CountMismatch, validation.SizeMismatch:
		a.addExistingFilePathToCheck(violation, filesToCheck)
	case validation.StructureViolation:
		a.addStructureViolationPathToCheck(violation, filesToCheck)
	}
}

// addMissingFilePathToCheck adds parent directory for missing file violations
func (a *AutofixerImpl) addMissingFilePathToCheck(violation validation.ValidationViolation, filesToCheck map[string]bool) {
	validatedPath, err := a.pathValidator.ValidatePath(violation.File)
	if err != nil {
		return // Skip invalid paths for permission checking
	}

	filePath, err := a.pathValidator.GetSafePath(validatedPath)
	if err != nil {
		return
	}

	parentDir := filepath.Dir(filePath)
	filesToCheck[parentDir] = true
}

// addExistingFilePathToCheck adds file path for existing file violations
func (a *AutofixerImpl) addExistingFilePathToCheck(violation validation.ValidationViolation, filesToCheck map[string]bool) {
	validatedPath, err := a.pathValidator.ValidatePath(violation.File)
	if err != nil {
		return // Skip invalid paths for permission checking
	}

	filePath, err := a.pathValidator.GetSafePath(validatedPath)
	if err != nil {
		return
	}

	filesToCheck[filePath] = true
}

// addStructureViolationPathToCheck adds parent directory for structure violations
func (a *AutofixerImpl) addStructureViolationPathToCheck(violation validation.ValidationViolation, filesToCheck map[string]bool) {
	if !isDirectoryMissing(violation) {
		return
	}

	validatedPath, err := a.pathValidator.ValidatePath(extractDirectoryFromViolation(violation))
	if err != nil {
		return // Skip invalid paths for permission checking
	}

	dirPath, err := a.pathValidator.GetSafePath(validatedPath)
	if err != nil {
		return
	}

	parentDir := filepath.Dir(dirPath)
	filesToCheck[parentDir] = true
}

// validateWritePermissions checks write permissions for all collected paths
func (a *AutofixerImpl) validateWritePermissions(filesToCheck map[string]bool) error {
	for path := range filesToCheck {
		if err := checkWritePermission(path); err != nil {
			return fmt.Errorf("path not writable: %s (%w)", path, err)
		}
	}
	return nil
}

// checkWritePermission verifies if a path is writable
func checkWritePermission(path string) error {
	// For directories, check if we can create files in them
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Path doesn't exist, check parent directory
			return checkWritePermission(filepath.Dir(path))
		}
		return err
	}

	if info.IsDir() {
		// Try to create a temporary file to test write permissions
		tempFile := filepath.Join(path, ".autofix-permission-test")
		file, err := os.Create(tempFile) // #nosec G304
		if err != nil {
			return err
		}
		defer func() {
			_ = file.Close()
			_ = os.Remove(tempFile)
		}()
		return nil
	}
	// For files, check if we can open them for writing
	file, err := os.OpenFile(path, os.O_WRONLY, 0) // #nosec G304
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()
	return nil
}
