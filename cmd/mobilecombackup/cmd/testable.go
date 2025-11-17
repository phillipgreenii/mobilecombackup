package cmd

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/phillipgreenii/mobilecombackup/pkg/attachments"
	"github.com/phillipgreenii/mobilecombackup/pkg/calls"
	"github.com/phillipgreenii/mobilecombackup/pkg/contacts"
	"github.com/phillipgreenii/mobilecombackup/pkg/importer"
	"github.com/phillipgreenii/mobilecombackup/pkg/sms"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

// DetermineExitCode returns the appropriate exit code based on import summary and options.
// This is a pure function extracted for testability.
//
// Exit codes:
//   - 0: Success - all files processed without errors
//   - 1: Warning - some entries were rejected but import completed (unless allowRejects is true)
func DetermineExitCode(summary *importer.ImportSummary, allowRejects bool) int {
	totalRejected := summary.Calls.Total.Rejected + summary.SMS.Total.Rejected
	if totalRejected > 0 && !allowRejects {
		return 1
	}
	return 0
}

// CalculateTotalRejections returns the total number of rejections across all categories.
// This is a pure function extracted for testability.
func CalculateTotalRejections(summary *importer.ImportSummary) int {
	totalRejections := 0
	for _, stats := range summary.Rejections {
		totalRejections += stats.Count
	}
	return totalRejections
}

// HealthStatus represents the validation health of a repository
type HealthStatus int

const (
	// HealthOK indicates the repository is healthy with no errors
	HealthOK HealthStatus = iota
	// HealthEmpty indicates the repository exists but has no data
	HealthEmpty
	// HealthUnhealthy indicates the repository has errors
	HealthUnhealthy
)

// CalculateRepositoryHealth determines the health status of a repository based on info.
// This is a pure function extracted for testability.
func CalculateRepositoryHealth(info *RepositoryInfo) HealthStatus {
	if len(info.Errors) > 0 {
		return HealthUnhealthy
	}

	totalCalls := 0
	for _, yearInfo := range info.Calls {
		totalCalls += yearInfo.Count
	}

	totalMessages := 0
	for _, msgInfo := range info.SMS {
		totalMessages += msgInfo.TotalCount
	}

	if totalCalls == 0 && totalMessages == 0 {
		return HealthEmpty
	}

	return HealthOK
}

// ExitHandler provides an interface for handling process exit.
// This allows testing of code that would normally call os.Exit().
type ExitHandler interface {
	Exit(code int)
}

// OSExitHandler implements ExitHandler using os.Exit for production use.
type OSExitHandler struct{}

// Exit calls os.Exit with the given code.
func (h OSExitHandler) Exit(code int) {
	os.Exit(code)
}

// TestExitHandler implements ExitHandler for testing.
// It records exit calls instead of terminating the process.
type TestExitHandler struct {
	Code   int
	Called bool
}

// Exit records the exit code and marks that exit was called.
func (h *TestExitHandler) Exit(code int) {
	h.Code = code
	h.Called = true
}

// InfoContext holds dependencies for the info command.
// This enables dependency injection and testing.
type InfoContext struct {
	RepoPath string
	Fs       afero.Fs
	Output   io.Writer

	// Readers (injected for testing)
	CallsReader      calls.Reader
	SMSReader        sms.Reader
	AttachmentReader *attachments.AttachmentManager
	ContactsReader   contacts.Reader

	// Exit handler (injected for testing)
	ExitHandler ExitHandler

	// Options
	OutputJSON bool
	Quiet      bool
}

// NewInfoContext creates a production InfoContext with real dependencies.
func NewInfoContext(repoPath string, outputJSON, quiet bool) *InfoContext {
	fs := afero.NewOsFs()
	return &InfoContext{
		RepoPath:         repoPath,
		Fs:               fs,
		Output:           os.Stdout,
		CallsReader:      calls.NewXMLCallsReader(repoPath),
		SMSReader:        sms.NewXMLSMSReader(repoPath),
		AttachmentReader: attachments.NewAttachmentManager(repoPath, fs),
		ContactsReader:   contacts.NewContactsManager(repoPath),
		ExitHandler:      OSExitHandler{},
		OutputJSON:       outputJSON,
		Quiet:            quiet,
	}
}

// GatherRepositoryInfo gathers repository information using injected dependencies.
// This method is fully testable with mocked readers and filesystem.
func (ctx *InfoContext) GatherRepositoryInfo() (*RepositoryInfo, error) {
	info := &RepositoryInfo{
		Calls:      make(map[string]YearInfo),
		SMS:        make(map[string]MessageInfo),
		Rejections: make(map[string]int),
		Errors:     make(map[string]int),
	}

	// Read repository metadata using injected filesystem
	if err := ctx.readRepositoryMetadata(info); err != nil {
		// Continue without metadata if file doesn't exist
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read metadata: %w", err)
		}
	}

	// Use injected readers (can be mocked in tests!)
	// Note: these helper functions are still using the old package-level functions
	// but they accept Reader interfaces, so they're testable
	if err := gatherCallsStats(ctx.CallsReader, info); err != nil {
		info.Errors["calls"] = 1
	}

	if err := gatherSMSStats(ctx.SMSReader, info); err != nil {
		info.Errors["sms"] = 1
	}

	if err := gatherAttachmentStats(ctx.AttachmentReader, ctx.SMSReader, info); err != nil {
		info.Errors["attachments"] = 1
	}

	if err := gatherContactsStats(ctx.ContactsReader, info); err != nil {
		info.Errors["contacts"] = 1
	}

	// Count rejections using filesystem
	ctx.countRejections(info)

	// Basic validation check
	info.ValidationOK = len(info.Errors) == 0

	return info, nil
}

// readRepositoryMetadata reads repository metadata from filesystem
func (ctx *InfoContext) readRepositoryMetadata(info *RepositoryInfo) error {
	markerPath := fmt.Sprintf("%s/.mobilecombackup.yaml", ctx.RepoPath)

	// Use injected filesystem - can be memory FS in tests!
	data, err := afero.ReadFile(ctx.Fs, markerPath)
	if err != nil {
		return err
	}

	var marker InfoMarkerFileContent
	if err := yaml.Unmarshal(data, &marker); err != nil {
		return fmt.Errorf("failed to parse marker file: %w", err)
	}

	info.Version = marker.RepositoryStructureVersion
	if marker.CreatedAt != "" {
		if createdAt, err := time.Parse(time.RFC3339, marker.CreatedAt); err == nil {
			info.CreatedAt = createdAt
		}
	}

	return nil
}

// countRejections counts rejection files using the injected filesystem
func (ctx *InfoContext) countRejections(info *RepositoryInfo) {
	rejectedDir := fmt.Sprintf("%s/rejected", ctx.RepoPath)

	// Check if rejected directory exists using injected filesystem
	exists, err := afero.DirExists(ctx.Fs, rejectedDir)
	if err != nil || !exists {
		return
	}

	// Count rejection files by type using injected filesystem
	entries, err := afero.ReadDir(ctx.Fs, rejectedDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() && len(name) > 4 && name[len(name)-4:] == ".xml" {
			// Check if name contains "calls" or "sms"
			hasCall := false
			hasSMS := false
			for i := 0; i < len(name)-4; i++ {
				if i+5 <= len(name) && name[i:i+5] == "calls" {
					hasCall = true
					break
				}
				if i+3 <= len(name) && name[i:i+3] == "sms" {
					hasSMS = true
					break
				}
			}

			if hasCall {
				info.Rejections["calls"]++
			} else if hasSMS {
				info.Rejections["sms"]++
			}
		}
	}
}

// ImportContext holds dependencies for the import command.
// This enables dependency injection and testing.
type ImportContext struct {
	Options     *importer.ImportOptions
	Output      io.Writer
	ExitHandler ExitHandler

	// Options
	Quiet            bool
	OutputJSON       bool
	DryRun           bool
	NoErrorOnRejects bool
}

// NewImportContext creates a production ImportContext with real dependencies.
func NewImportContext(options *importer.ImportOptions, outputJSON, dryRun, noErrorOnRejects bool) *ImportContext {
	// JSON mode forces quiet to ensure clean JSON output
	effectiveQuiet := options.Quiet || outputJSON

	return &ImportContext{
		Options:          options,
		Output:           os.Stdout,
		ExitHandler:      OSExitHandler{},
		Quiet:            effectiveQuiet,
		OutputJSON:       outputJSON,
		DryRun:           dryRun,
		NoErrorOnRejects: noErrorOnRejects,
	}
}

// HandleImportError handles errors from the import process.
func (ctx *ImportContext) HandleImportError(err error) {
	if !ctx.Quiet {
		_, _ = fmt.Fprintf(ctx.Output, "Import failed: %v\n", err)
	}
	ctx.ExitHandler.Exit(2)
}

// HandleExitCode determines and sets the appropriate exit code based on import results.
func (ctx *ImportContext) HandleExitCode(summary *importer.ImportSummary) {
	exitCode := DetermineExitCode(summary, ctx.NoErrorOnRejects)
	if exitCode != 0 {
		ctx.ExitHandler.Exit(exitCode)
	}
}
