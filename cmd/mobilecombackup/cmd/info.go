package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/phillipgreenii/mobilecombackup/pkg/attachments"
	"github.com/phillipgreenii/mobilecombackup/pkg/calls"
	"github.com/phillipgreenii/mobilecombackup/pkg/contacts"
	"github.com/phillipgreenii/mobilecombackup/pkg/display"
	"github.com/phillipgreenii/mobilecombackup/pkg/repository/metadata"
	"github.com/phillipgreenii/mobilecombackup/pkg/repository/stats"
	"github.com/phillipgreenii/mobilecombackup/pkg/sms"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var (
	outputInfoJSON bool
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show repository information and statistics",
	Long: `Show comprehensive information about a mobilecombackup repository.

This command displays:
- Repository metadata (version, creation date)
- Call statistics by year with date ranges
- SMS/MMS statistics by year with type breakdown
- Attachment statistics with type distribution
- Contact information
- Rejection and error counts
- Basic validation status`,
	RunE: runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)

	// Local flags
	infoCmd.Flags().BoolVar(&outputInfoJSON, "json", false, "Output information in JSON format")
}

// RepositoryInfo contains all repository information (wrapper for stats.RepositoryStats)
type RepositoryInfo struct {
	Version      string                       `json:"version"`
	CreatedAt    time.Time                    `json:"created_at,omitempty"`
	Calls        map[string]stats.YearInfo    `json:"calls"` // year -> info
	SMS          map[string]stats.MessageInfo `json:"sms"`   // year -> info
	Attachments  AttachmentInfo               `json:"attachments"`
	Contacts     ContactInfo                  `json:"contacts"`
	Rejections   map[string]int               `json:"rejections,omitempty"` // component -> count
	Errors       map[string]int               `json:"errors,omitempty"`     // component -> count
	ValidationOK bool                         `json:"validation_ok"`
}

// AttachmentInfo contains attachment statistics (wrapper for stats.AttachmentInfo)
type AttachmentInfo struct {
	Count         int            `json:"count"`
	TotalSize     int64          `json:"total_size"`
	OrphanedCount int            `json:"orphaned_count"`
	ByType        map[string]int `json:"by_type"` // mime type -> count
}

// ContactInfo contains contact statistics (wrapper for stats.ContactInfo)
type ContactInfo struct {
	Count int `json:"count"`
}

func runInfo(_ *cobra.Command, _ []string) error {
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

	// Gather repository information
	info, err := gatherRepositoryInfo(absPath)
	if err != nil {
		PrintError("Failed to gather repository information: %v", err)
		os.Exit(2)
	}

	// Output results
	if outputInfoJSON {
		if err := outputInfoAsJSON(info); err != nil {
			return fmt.Errorf("failed to output JSON: %w", err)
		}
	} else {
		// Convert to display format and output
		displayInfo := convertToDisplayInfo(info)
		formatter := display.NewFormatter(os.Stdout, quiet)
		formatter.FormatRepositoryInfo(displayInfo, absPath)
	}

	return nil
}

func gatherRepositoryInfo(repoPath string) (*RepositoryInfo, error) {
	info := &RepositoryInfo{
		Calls:      make(map[string]stats.YearInfo),
		SMS:        make(map[string]stats.MessageInfo),
		Rejections: make(map[string]int),
		Errors:     make(map[string]int),
	}

	// Read repository metadata
	if err := readRepositoryMetadata(repoPath, info); err != nil {
		// Continue without metadata if file doesn't exist
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read metadata: %w", err)
		}
	}

	// Create readers
	callsReader := calls.NewXMLCallsReader(repoPath)
	smsReader := sms.NewXMLSMSReader(repoPath)
	attachmentReader := attachments.NewAttachmentManager(repoPath, afero.NewOsFs())
	contactsReader := contacts.NewContactsManager(repoPath)

	// Create stats gatherer
	gatherer := stats.NewStatsGatherer(
		repoPath,
		callsReader,
		smsReader,
		attachmentReader,
		contactsReader,
	)

	// Gather all statistics
	ctx := context.Background()
	repoStats, err := gatherer.GatherStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to gather statistics: %w", err)
	}

	// Convert stats to info format
	info.Calls = repoStats.Calls
	info.SMS = repoStats.SMS
	info.Errors = repoStats.Errors
	info.ValidationOK = repoStats.ValidationOK

	// Convert attachment stats
	info.Attachments = AttachmentInfo{
		Count:         repoStats.Attachments.Count,
		TotalSize:     repoStats.Attachments.TotalSize,
		OrphanedCount: repoStats.Attachments.Orphaned,
		ByType:        repoStats.Attachments.ByType,
	}

	// Convert contact stats
	info.Contacts = ContactInfo{
		Count: repoStats.Contacts.Count,
	}

	// Count rejections
	countRejections(repoPath, info)

	return info, nil
}

func readRepositoryMetadata(repoPath string, info *RepositoryInfo) error {
	reader := metadata.NewReader(repoPath)
	meta, err := reader.ReadMetadata()
	if err != nil {
		return err
	}

	info.Version = meta.Version
	info.CreatedAt = meta.CreatedAt

	return nil
}

// convertToDisplayInfo converts RepositoryInfo to display.RepositoryInfo
func convertToDisplayInfo(info *RepositoryInfo) *display.RepositoryInfo {
	return &display.RepositoryInfo{
		Version:   info.Version,
		CreatedAt: info.CreatedAt,
		Calls:     info.Calls,
		SMS:       info.SMS,
		Attachments: display.AttachmentInfo{
			Count:         info.Attachments.Count,
			TotalSize:     info.Attachments.TotalSize,
			OrphanedCount: info.Attachments.OrphanedCount,
			ByType:        info.Attachments.ByType,
		},
		Contacts:     display.ContactInfo{Count: info.Contacts.Count},
		Rejections:   info.Rejections,
		Errors:       info.Errors,
		ValidationOK: info.ValidationOK,
	}
}

func countRejections(repoPath string, info *RepositoryInfo) {
	rejectedDir := filepath.Join(repoPath, "rejected")

	counts, err := CountRejectionsByType(rejectedDir)
	if err != nil {
		// Silently ignore errors in rejection counting
		return
	}

	// Copy counts to info
	for rejectType, count := range counts {
		info.Rejections[rejectType] = count
	}
}

func outputInfoAsJSON(info *RepositoryInfo) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(info)
}
