package autofix

import (
	"crypto/sha256"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

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

// FileEntry represents a single file entry in files.yaml
type FileEntry struct {
	File      string `yaml:"file"`
	SHA256    string `yaml:"sha256"`
	SizeBytes int64  `yaml:"size_bytes"`
}

// FileManifest represents the structure of files.yaml
type FileManifest struct {
	Files []FileEntry `yaml:"files"`
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
}

// NewAutofixer creates a new autofixer instance
func NewAutofixer(repositoryRoot string, reporter ProgressReporter) Autofixer {
	if reporter == nil {
		reporter = &NullProgressReporter{}
	}
	return &AutofixerImpl{
		repositoryRoot: repositoryRoot,
		reporter:       reporter,
	}
}

// FixViolations attempts to fix the given validation violations
func (a *AutofixerImpl) FixViolations(violations []validation.ValidationViolation, options AutofixOptions) (*AutofixReport, error) {
	report := &AutofixReport{
		Timestamp:         time.Now().UTC(),
		RepositoryPath:    a.repositoryRoot,
		FixedViolations:   []FixedViolation{},
		SkippedViolations: []SkippedViolation{},
		Errors:            []AutofixError{},
	}

	// In dry-run mode, perform permission checks
	if options.DryRun {
		if err := a.checkPermissions(violations); err != nil {
			report.Errors = append(report.Errors, AutofixError{
				ViolationType: "",
				File:          a.repositoryRoot,
				Operation:     "permission_check",
				Error:         err.Error(),
			})
		}
	}

	// Phase 1: Create directories (must come first)
	directoryViolations := filterViolationsByTypes(violations, []validation.ViolationType{
		validation.StructureViolation,
	})

	for _, violation := range directoryViolations {
		if isDirectoryMissing(violation) {
			if options.DryRun {
				report.FixedViolations = append(report.FixedViolations, FixedViolation{
					OriginalViolation: violation,
					FixAction:         OperationCreateDirectory,
					Details:           fmt.Sprintf("Would create directory: %s", extractDirectoryFromViolation(violation)),
				})
				continue
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
	}

	// Phase 2: Create missing files
	fileViolations := filterViolationsByTypes(violations, []validation.ViolationType{
		validation.MissingFile,
		validation.MissingMarkerFile,
	})

	for _, violation := range fileViolations {
		if !a.CanFix(violation.Type) {
			report.SkippedViolations = append(report.SkippedViolations, SkippedViolation{
				OriginalViolation: violation,
				SkipReason:        "Not a safe autofix operation",
			})
			continue
		}

		if options.DryRun {
			report.FixedViolations = append(report.FixedViolations, FixedViolation{
				OriginalViolation: violation,
				FixAction:         OperationCreateFile,
				Details:           fmt.Sprintf("Would create file: %s", violation.File),
			})
			continue
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

	// Phase 3: Update existing files and content
	contentViolations := filterViolationsByTypes(violations, []validation.ViolationType{
		validation.CountMismatch,
		validation.SizeMismatch,
	})

	for _, violation := range contentViolations {
		if !a.CanFix(violation.Type) {
			report.SkippedViolations = append(report.SkippedViolations, SkippedViolation{
				OriginalViolation: violation,
				SkipReason:        "Not a safe autofix operation",
			})
			continue
		}

		var err error
		var fixAction string

		if options.DryRun {
			switch violation.Type {
			case validation.CountMismatch:
				fixAction = OperationUpdateXMLCount
				details := fmt.Sprintf("Would update count attribute in %s", violation.File)
				if violation.Expected != "" && violation.Actual != "" {
					details = fmt.Sprintf("Would update count attribute in %s (from %s to %s)", violation.File, violation.Actual, violation.Expected)
				}
				report.FixedViolations = append(report.FixedViolations, FixedViolation{
					OriginalViolation: violation,
					FixAction:         fixAction,
					Details:           details,
				})
			case validation.SizeMismatch:
				fixAction = OperationUpdateFile
				report.FixedViolations = append(report.FixedViolations, FixedViolation{
					OriginalViolation: violation,
					FixAction:         fixAction,
					Details:           "Would regenerate files.yaml with correct file sizes",
				})
			}
			continue
		}

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

	// Skip violations that cannot be safely fixed
	unsafeViolations := filterViolationsByTypes(violations, []validation.ViolationType{
		validation.ChecksumMismatch,
		validation.OrphanedAttachment,
	})

	for _, violation := range unsafeViolations {
		var reason string
		switch violation.Type {
		case validation.ChecksumMismatch:
			reason = "Autofix preserves existing checksums to detect corruption"
		case validation.OrphanedAttachment:
			reason = "Use --remove-orphan-attachments flag"
		default:
			reason = "Not a safe autofix operation"
		}

		report.SkippedViolations = append(report.SkippedViolations, SkippedViolation{
			OriginalViolation: violation,
			SkipReason:        reason,
		})
	}

	// Calculate summary
	report.Summary = AutofixSummary{
		FixedCount:   len(report.FixedViolations),
		SkippedCount: len(report.SkippedViolations),
		ErrorCount:   len(report.Errors),
	}

	return report, nil
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
	dirPath := filepath.Join(a.repositoryRoot, extractDirectoryFromViolation(violation))

	a.reporter.StartOperation(OperationCreateDirectory, dirPath)

	err := os.MkdirAll(dirPath, 0750)

	a.reporter.CompleteOperation(err == nil, fmt.Sprintf("Directory: %s", dirPath))

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

	filePath := filepath.Join(a.repositoryRoot, violation.File)

	// Read the XML file
	content, err := os.ReadFile(filePath)
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
	markerPath := filepath.Join(a.repositoryRoot, ".mobilecombackup.yaml")

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
	contactsPath := filepath.Join(a.repositoryRoot, "contacts.yaml")

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
	summaryPath := filepath.Join(a.repositoryRoot, "summary.yaml")

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
	manifestPath := filepath.Join(a.repositoryRoot, "files.yaml")

	// Scan repository for all files (excluding files.yaml itself and temp files)
	manifest := FileManifest{
		Files: []FileEntry{},
	}

	// First, count total files for progress reporting
	var totalFiles int
	err := filepath.Walk(a.repositoryRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(a.repositoryRoot, path)
			if relPath != "files.yaml" && relPath != "files.yaml.sha256" &&
				!strings.HasSuffix(relPath, ".tmp") && !strings.HasPrefix(filepath.Base(relPath), ".") {
				totalFiles++
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to count files: %w", err)
	}

	// Now process files with progress reporting
	var processedFiles int
	err = filepath.Walk(a.repositoryRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(a.repositoryRoot, path)
		if err != nil {
			return err
		}

		// Skip files.yaml itself, temp files, and hidden files
		if relPath == "files.yaml" || relPath == "files.yaml.sha256" ||
			strings.HasSuffix(relPath, ".tmp") || strings.HasPrefix(filepath.Base(relPath), ".") {
			return nil
		}

		processedFiles++

		// Report progress every 1000 files or for the first/last file
		if processedFiles%1000 == 0 || processedFiles == 1 || processedFiles == totalFiles {
			a.reporter.ReportProgress(processedFiles, totalFiles)
		}

		// Calculate SHA-256
		hash, err := calculateFileHash(path)
		if err != nil {
			return fmt.Errorf("failed to calculate hash for %s: %w", relPath, err)
		}

		// Add to manifest
		manifest.Files = append(manifest.Files, FileEntry{
			File:      relPath,
			SHA256:    hash,
			SizeBytes: info.Size(),
		})

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan repository: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal files manifest: %w", err)
	}

	// Write to file atomically
	tempPath := manifestPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary files manifest: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, manifestPath); err != nil {
		_ = os.Remove(tempPath) // Clean up temp file, ignore error
		return fmt.Errorf("failed to rename files manifest: %w", err)
	}

	return nil
}

func (a *AutofixerImpl) createManifestChecksum() error {
	manifestPath := filepath.Join(a.repositoryRoot, "files.yaml")
	checksumPath := filepath.Join(a.repositoryRoot, "files.yaml.sha256")

	// Calculate hash of files.yaml
	hash, err := calculateFileHash(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to calculate files.yaml hash: %w", err)
	}

	// Create checksum content (just the hash)
	checksumContent := hash + "\n"

	// Write to file atomically
	tempPath := checksumPath + ".tmp"
	if err := os.WriteFile(tempPath, []byte(checksumContent), 0644); err != nil {
		return fmt.Errorf("failed to write temporary checksum file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, checksumPath); err != nil {
		_ = os.Remove(tempPath) // Clean up temp file, ignore error
		return fmt.Errorf("failed to rename checksum file: %w", err)
	}

	return nil
}

// calculateFileHash calculates SHA-256 hash of a file
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
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
	filesToCheck := make(map[string]bool)

	for _, violation := range violations {
		switch violation.Type {
		case validation.MissingFile, validation.MissingMarkerFile:
			// Check if parent directory is writable
			filePath := filepath.Join(a.repositoryRoot, violation.File)
			parentDir := filepath.Dir(filePath)
			filesToCheck[parentDir] = true

		case validation.CountMismatch, validation.SizeMismatch:
			// Check if existing file is writable
			filePath := filepath.Join(a.repositoryRoot, violation.File)
			filesToCheck[filePath] = true

		case validation.StructureViolation:
			// Check if we can create directories
			if isDirectoryMissing(violation) {
				dirPath := filepath.Join(a.repositoryRoot, extractDirectoryFromViolation(violation))
				parentDir := filepath.Dir(dirPath)
				filesToCheck[parentDir] = true
			}
		}
	}

	// Check write permissions for all identified paths
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
		file, err := os.Create(tempFile)
		if err != nil {
			return err
		}
		_ = file.Close()
		_ = os.Remove(tempFile)
		return nil
	} else {
		// For files, check if we can open them for writing
		file, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			return err
		}
		_ = file.Close()
		return nil
	}
}
