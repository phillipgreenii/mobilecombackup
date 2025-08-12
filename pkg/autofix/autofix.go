package autofix

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/validation"
)

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
			case validation.SizeMismatch:
				fixAction = OperationUpdateFile
			}
			report.FixedViolations = append(report.FixedViolations, FixedViolation{
				OriginalViolation: violation,
				FixAction:         fixAction,
				Details:           fmt.Sprintf("Would fix %s in %s", violation.Type, violation.File),
			})
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
	// This will be improved when we implement specific directory detection
	return violation.Type == validation.StructureViolation &&
		(violation.File == "calls/" || violation.File == "sms/" || violation.File == "attachments/")
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

	// This will be implemented when we add XML count fixing
	err := fmt.Errorf("XML count fixing not yet implemented")

	a.reporter.CompleteOperation(false, violation.File)

	return err
}

func (a *AutofixerImpl) fixSizeMismatch(violation validation.ValidationViolation) error {
	a.reporter.StartOperation(OperationUpdateFile, violation.File)

	// This will be implemented when we add size fixing in files.yaml
	err := fmt.Errorf("size mismatch fixing not yet implemented")

	a.reporter.CompleteOperation(false, violation.File)

	return err
}

func (a *AutofixerImpl) createMarkerFile() error {
	// Implementation will be added in the next task
	return fmt.Errorf("marker file creation not yet implemented")
}

func (a *AutofixerImpl) createEmptyContactsFile() error {
	// Implementation will be added in the next task
	return fmt.Errorf("contacts file creation not yet implemented")
}

func (a *AutofixerImpl) createSummaryFile() error {
	// Implementation will be added in the next task
	return fmt.Errorf("summary file creation not yet implemented")
}

func (a *AutofixerImpl) createFilesManifest() error {
	// Implementation will be added in the next task
	return fmt.Errorf("files manifest creation not yet implemented")
}

func (a *AutofixerImpl) createManifestChecksum() error {
	// Implementation will be added in the next task
	return fmt.Errorf("manifest checksum creation not yet implemented")
}
