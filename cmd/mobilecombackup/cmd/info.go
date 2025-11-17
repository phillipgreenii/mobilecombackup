package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/phillipgreenii/mobilecombackup/pkg/attachments"
	"github.com/phillipgreenii/mobilecombackup/pkg/calls"
	"github.com/phillipgreenii/mobilecombackup/pkg/contacts"
	"github.com/phillipgreenii/mobilecombackup/pkg/repository/stats"
	"github.com/phillipgreenii/mobilecombackup/pkg/sms"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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

// InfoMarkerFileContent represents the .mobilecombackup.yaml file structure
type InfoMarkerFileContent struct {
	RepositoryStructureVersion string `yaml:"repository_structure_version" json:"repository_structure_version"`
	CreatedAt                  string `yaml:"created_at" json:"created_at"`
	CreatedBy                  string `yaml:"created_by" json:"created_by"`
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
		outputTextInfo(info, absPath)
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
	markerPath := filepath.Join(repoPath, ".mobilecombackup.yaml")

	data, err := os.ReadFile(markerPath) // #nosec G304
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

func countRejections(repoPath string, info *RepositoryInfo) {
	rejectedDir := filepath.Join(repoPath, "rejected")

	// Check if rejected directory exists
	if _, err := os.Stat(rejectedDir); os.IsNotExist(err) {
		return
	}

	// Count rejection files by type
	entries, err := os.ReadDir(rejectedDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".xml") {
			if strings.Contains(entry.Name(), "calls") {
				info.Rejections["calls"]++
			} else if strings.Contains(entry.Name(), "sms") {
				info.Rejections["sms"]++
			}
		}
	}
}

func outputInfoAsJSON(info *RepositoryInfo) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(info)
}

func outputTextInfo(info *RepositoryInfo, repoPath string) {
	if quiet {
		return
	}

	printRepositoryHeader(info, repoPath)
	printCallsStatistics(info)
	printMessagesStatistics(info)
	printAttachmentsStatistics(info)
	printContacts(info)
	printIssues(info)
	printValidationStatus(info)
}

// printRepositoryHeader prints the repository header information
func printRepositoryHeader(info *RepositoryInfo, repoPath string) {
	fmt.Printf("Repository: %s\n", repoPath)
	if info.Version != "" {
		fmt.Printf("Version: %s\n", info.Version)
	}
	if !info.CreatedAt.IsZero() {
		fmt.Printf("Created: %s\n", info.CreatedAt.Format(time.RFC3339))
	}
	fmt.Println()
}

// printCallsStatistics prints call statistics
func printCallsStatistics(info *RepositoryInfo) {
	fmt.Println("Calls:")
	if len(info.Calls) > 0 {
		var totalCalls int
		years := getSortedCallsYears(info.Calls)

		for _, year := range years {
			yearInfo := info.Calls[year]
			totalCalls += yearInfo.Count
			fmt.Printf("  %s: %s calls", year, formatNumber(yearInfo.Count))
			printDateRange(yearInfo.Earliest, yearInfo.Latest)
			fmt.Println()
		}
		fmt.Printf("  Total: %s calls\n", formatNumber(totalCalls))
	} else {
		fmt.Println("  Total: 0 calls")
	}
	fmt.Println()
}

// printMessagesStatistics prints SMS/MMS statistics
func printMessagesStatistics(info *RepositoryInfo) {
	fmt.Println("Messages:")
	if len(info.SMS) > 0 {
		totalMessages, totalSMS, totalMMS := calculateMessageTotals(info)
		years := getSortedMessageYears(info.SMS)

		for _, year := range years {
			msgInfo := info.SMS[year]
			fmt.Printf("  %s: %s messages (%s SMS, %s MMS)",
				year,
				formatNumber(msgInfo.TotalCount),
				formatNumber(msgInfo.SMSCount),
				formatNumber(msgInfo.MMSCount))
			printDateRange(msgInfo.Earliest, msgInfo.Latest)
			fmt.Println()
		}
		fmt.Printf("  Total: %s messages (%s SMS, %s MMS)\n",
			formatNumber(totalMessages),
			formatNumber(totalSMS),
			formatNumber(totalMMS))
	} else {
		fmt.Println("  Total: 0 messages (0 SMS, 0 MMS)")
	}
	fmt.Println()
}

// printAttachmentsStatistics prints attachment statistics
func printAttachmentsStatistics(info *RepositoryInfo) {
	fmt.Println("Attachments:")
	if info.Attachments.Count > 0 {
		fmt.Printf("  Count: %s\n", formatNumber(info.Attachments.Count))
		fmt.Printf("  Total Size: %s\n", formatBytes(info.Attachments.TotalSize))

		printAttachmentTypes(info)
		printOrphanedAttachments(info)
	} else {
		fmt.Println("  Count: 0")
		fmt.Println("  Total Size: 0 B")
	}
	fmt.Println()
}

// getSortedCallsYears returns sorted years from calls data
func getSortedCallsYears(calls map[string]stats.YearInfo) []string {
	years := make([]string, 0, len(calls))
	for year := range calls {
		years = append(years, year)
	}
	sort.Strings(years)
	return years
}

// getSortedMessageYears returns sorted years from SMS data
func getSortedMessageYears(sms map[string]stats.MessageInfo) []string {
	years := make([]string, 0, len(sms))
	for year := range sms {
		years = append(years, year)
	}
	sort.Strings(years)
	return years
}

// calculateMessageTotals calculates total message counts
func calculateMessageTotals(info *RepositoryInfo) (totalMessages, totalSMS, totalMMS int) {
	for _, msgInfo := range info.SMS {
		totalMessages += msgInfo.TotalCount
		totalSMS += msgInfo.SMSCount
		totalMMS += msgInfo.MMSCount
	}
	return
}

// printDateRange prints date range if available
func printDateRange(earliest, latest time.Time) {
	if !earliest.IsZero() && !latest.IsZero() {
		fmt.Printf(" (%s - %s)",
			earliest.Format("Jan 2"),
			latest.Format("Jan 2"))
	}
}

// printAttachmentTypes prints attachment types breakdown
func printAttachmentTypes(info *RepositoryInfo) {
	if len(info.Attachments.ByType) > 0 {
		fmt.Println("  Types:")
		types := getSortedAttachmentTypes(info.Attachments.ByType)
		for _, mimeType := range types {
			count := info.Attachments.ByType[mimeType]
			fmt.Printf("    %s: %s\n", mimeType, formatNumber(count))
		}
	}
}

// getSortedAttachmentTypes returns attachment types sorted by count
func getSortedAttachmentTypes(byType map[string]int) []string {
	types := make([]string, 0, len(byType))
	for mimeType := range byType {
		types = append(types, mimeType)
	}
	sort.Slice(types, func(i, j int) bool {
		return byType[types[i]] > byType[types[j]]
	})
	return types
}

// printOrphanedAttachments prints orphaned attachment count
func printOrphanedAttachments(info *RepositoryInfo) {
	if info.Attachments.OrphanedCount > 0 {
		fmt.Printf("  Orphaned: %s\n", formatNumber(info.Attachments.OrphanedCount))
	}
}

// printContacts prints contact count
func printContacts(info *RepositoryInfo) {
	fmt.Printf("Contacts: %s\n\n", formatNumber(info.Contacts.Count))
}

// printIssues prints rejection and error statistics
func printIssues(info *RepositoryInfo) {
	if len(info.Rejections) > 0 || len(info.Errors) > 0 {
		fmt.Println("Issues:")
		for component, count := range info.Rejections {
			fmt.Printf("  Rejections (%s): %s\n", component, formatNumber(count))
		}
		for component, count := range info.Errors {
			fmt.Printf("  Errors (%s): %s\n", component, formatNumber(count))
		}
		fmt.Println()
	}
}

// printValidationStatus prints validation status
func printValidationStatus(info *RepositoryInfo) {
	if info.ValidationOK {
		fmt.Println("Validation: OK")
	} else {
		fmt.Println("Validation: Issues detected")
	}
}

func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return addCommas(fmt.Sprintf("%d", n))
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func addCommas(s string) string {
	n := len(s)
	if n <= 3 {
		return s
	}

	var result strings.Builder
	for i, r := range s {
		if i > 0 && (n-i)%3 == 0 {
			result.WriteString(",")
		}
		result.WriteRune(r)
	}
	return result.String()
}
