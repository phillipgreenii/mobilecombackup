package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/attachments"
	"github.com/phillipgreen/mobilecombackup/pkg/calls"
	"github.com/phillipgreen/mobilecombackup/pkg/contacts"
	"github.com/phillipgreen/mobilecombackup/pkg/sms"
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

// RepositoryInfo contains all repository information
type RepositoryInfo struct {
	Version      string                    `json:"version"`
	CreatedAt    time.Time                 `json:"created_at,omitempty"`
	Calls        map[string]YearInfo       `json:"calls"`        // year -> info
	SMS          map[string]MessageInfo    `json:"sms"`          // year -> info  
	Attachments  AttachmentInfo            `json:"attachments"`
	Contacts     int                       `json:"contacts"`
	Rejections   map[string]int            `json:"rejections,omitempty"`    // component -> count
	Errors       map[string]int            `json:"errors,omitempty"`        // component -> count
	ValidationOK bool                      `json:"validation_ok"`
}

// YearInfo contains year-specific statistics
type YearInfo struct {
	Count     int       `json:"count"`
	Earliest  time.Time `json:"earliest,omitempty"`
	Latest    time.Time `json:"latest,omitempty"`
}

// MessageInfo contains message statistics
type MessageInfo struct {
	TotalCount int       `json:"total_count"`
	SMSCount   int       `json:"sms_count"`
	MMSCount   int       `json:"mms_count"`
	Earliest   time.Time `json:"earliest,omitempty"`
	Latest     time.Time `json:"latest,omitempty"`
}

// AttachmentInfo contains attachment statistics
type AttachmentInfo struct {
	Count         int              `json:"count"`
	TotalSize     int64            `json:"total_size"`
	OrphanedCount int              `json:"orphaned_count"`
	ByType        map[string]int   `json:"by_type"` // mime type -> count
}

// InfoMarkerFileContent represents the .mobilecombackup.yaml file structure
type InfoMarkerFileContent struct {
	RepositoryStructureVersion string `yaml:"repository_structure_version"`
	CreatedAt                  string `yaml:"created_at"`
	CreatedBy                  string `yaml:"created_by"`
}

func runInfo(cmd *cobra.Command, args []string) error {
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
		Calls:      make(map[string]YearInfo),
		SMS:        make(map[string]MessageInfo),
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
	attachmentReader := attachments.NewAttachmentManager(repoPath)
	contactsReader := contacts.NewContactsManager(repoPath)

	// Gather calls statistics
	if err := gatherCallsStats(callsReader, info); err != nil {
		info.Errors["calls"] = 1
	}

	// Gather SMS statistics
	if err := gatherSMSStats(smsReader, info); err != nil {
		info.Errors["sms"] = 1
	}

	// Gather attachment statistics
	if err := gatherAttachmentStats(attachmentReader, smsReader, info); err != nil {
		info.Errors["attachments"] = 1
	}

	// Gather contacts statistics
	if err := gatherContactsStats(contactsReader, info); err != nil {
		info.Errors["contacts"] = 1
	}

	// Count rejections
	countRejections(repoPath, info)

	// Basic validation check
	info.ValidationOK = len(info.Errors) == 0

	return info, nil
}

func readRepositoryMetadata(repoPath string, info *RepositoryInfo) error {
	markerPath := filepath.Join(repoPath, ".mobilecombackup.yaml")
	
	data, err := os.ReadFile(markerPath)
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

func gatherCallsStats(reader calls.CallsReader, info *RepositoryInfo) error {
	years, err := reader.GetAvailableYears()
	if err != nil {
		return err
	}

	for _, year := range years {
		yearStats := YearInfo{}

		// Get count
		count, err := reader.GetCallsCount(year)
		if err != nil {
			return err
		}
		yearStats.Count = count

		// Get date range by streaming calls
		var earliest, latest time.Time
		err = reader.StreamCallsForYear(year, func(call calls.Call) error {
			timestamp := call.Timestamp()
			if earliest.IsZero() || timestamp.Before(earliest) {
				earliest = timestamp
			}
			if latest.IsZero() || timestamp.After(latest) {
				latest = timestamp
			}
			return nil
		})
		if err != nil {
			return err
		}

		if !earliest.IsZero() {
			yearStats.Earliest = earliest
			yearStats.Latest = latest
		}

		info.Calls[fmt.Sprintf("%d", year)] = yearStats
	}

	return nil
}

func gatherSMSStats(reader sms.SMSReader, info *RepositoryInfo) error {
	years, err := reader.GetAvailableYears()
	if err != nil {
		return err
	}

	for _, year := range years {
		messageStats := MessageInfo{}

		// Get total count
		totalCount, err := reader.GetMessageCount(year)
		if err != nil {
			return err
		}
		messageStats.TotalCount = totalCount

		// Count SMS vs MMS and get date range by streaming messages
		var earliest, latest time.Time
		err = reader.StreamMessagesForYear(year, func(msg sms.Message) error {
			timestamp := time.Unix(msg.GetDate()/1000, (msg.GetDate()%1000)*int64(time.Millisecond))
			
			if earliest.IsZero() || timestamp.Before(earliest) {
				earliest = timestamp
			}
			if latest.IsZero() || timestamp.After(latest) {
				latest = timestamp
			}

			// Count SMS vs MMS by type assertion
			switch msg.(type) {
			case *sms.SMS:
				messageStats.SMSCount++
			case *sms.MMS:
				messageStats.MMSCount++
			}
			return nil
		})
		if err != nil {
			return err
		}

		if !earliest.IsZero() {
			messageStats.Earliest = earliest
			messageStats.Latest = latest
		}

		info.SMS[fmt.Sprintf("%d", year)] = messageStats
	}

	return nil
}

func gatherAttachmentStats(attachmentReader attachments.AttachmentReader, smsReader sms.SMSReader, info *RepositoryInfo) error {
	attachmentInfo := AttachmentInfo{
		ByType: make(map[string]int),
	}

	// Get all attachments
	attachmentList, err := attachmentReader.ListAttachments()
	if err != nil {
		return err
	}

	attachmentInfo.Count = len(attachmentList)

	// Calculate total size and type distribution
	for _, attachment := range attachmentList {
		attachmentInfo.TotalSize += attachment.Size
		
		// Determine type from file extension or content inspection
		mimeType := determineMimeType(attachment.Path)
		attachmentInfo.ByType[mimeType]++
	}

	// Find orphaned attachments
	referencedHashes, err := smsReader.GetAllAttachmentRefs()
	if err != nil {
		return err
	}

	orphanedAttachments, err := attachmentReader.FindOrphanedAttachments(referencedHashes)
	if err != nil {
		return err
	}
	attachmentInfo.OrphanedCount = len(orphanedAttachments)

	info.Attachments = attachmentInfo
	return nil
}

func gatherContactsStats(reader contacts.ContactsReader, info *RepositoryInfo) error {
	// Load contacts
	err := reader.LoadContacts()
	if err != nil {
		// Continue if contacts file doesn't exist
		if os.IsNotExist(err) {
			info.Contacts = 0
			return nil
		}
		return err
	}

	info.Contacts = reader.GetContactsCount()
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

func determineMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".mp4":
		return "video/mp4"
	case ".3gp":
		return "video/3gpp"
	case ".mp3":
		return "audio/mp3"
	case ".amr":
		return "audio/amr"
	default:
		return "application/octet-stream"
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

	fmt.Printf("Repository: %s\n", repoPath)
	if info.Version != "" {
		fmt.Printf("Version: %s\n", info.Version)
	}
	if !info.CreatedAt.IsZero() {
		fmt.Printf("Created: %s\n", info.CreatedAt.Format(time.RFC3339))
	}
	fmt.Println()

	// Calls statistics
	if len(info.Calls) > 0 {
		fmt.Println("Calls:")
		var totalCalls int
		var years []string
		for year := range info.Calls {
			years = append(years, year)
		}
		sort.Strings(years)

		for _, year := range years {
			yearInfo := info.Calls[year]
			totalCalls += yearInfo.Count
			fmt.Printf("  %s: %s calls", year, formatNumber(yearInfo.Count))
			if !yearInfo.Earliest.IsZero() && !yearInfo.Latest.IsZero() {
				fmt.Printf(" (%s - %s)",
					yearInfo.Earliest.Format("Jan 2"),
					yearInfo.Latest.Format("Jan 2"))
			}
			fmt.Println()
		}
		fmt.Printf("  Total: %s calls\n", formatNumber(totalCalls))
		fmt.Println()
	}

	// SMS/MMS statistics
	if len(info.SMS) > 0 {
		fmt.Println("Messages:")
		var totalMessages, totalSMS, totalMMS int
		var years []string
		for year := range info.SMS {
			years = append(years, year)
		}
		sort.Strings(years)

		for _, year := range years {
			msgInfo := info.SMS[year]
			totalMessages += msgInfo.TotalCount
			totalSMS += msgInfo.SMSCount
			totalMMS += msgInfo.MMSCount
			fmt.Printf("  %s: %s messages (%s SMS, %s MMS)", 
				year, 
				formatNumber(msgInfo.TotalCount),
				formatNumber(msgInfo.SMSCount),
				formatNumber(msgInfo.MMSCount))
			if !msgInfo.Earliest.IsZero() && !msgInfo.Latest.IsZero() {
				fmt.Printf(" (%s - %s)",
					msgInfo.Earliest.Format("Jan 2"),
					msgInfo.Latest.Format("Jan 2"))
			}
			fmt.Println()
		}
		fmt.Printf("  Total: %s messages (%s SMS, %s MMS)\n", 
			formatNumber(totalMessages), 
			formatNumber(totalSMS), 
			formatNumber(totalMMS))
		fmt.Println()
	}

	// Attachments statistics
	if info.Attachments.Count > 0 {
		fmt.Println("Attachments:")
		fmt.Printf("  Count: %s\n", formatNumber(info.Attachments.Count))
		fmt.Printf("  Total Size: %s\n", formatBytes(info.Attachments.TotalSize))
		
		if len(info.Attachments.ByType) > 0 {
			fmt.Println("  Types:")
			// Sort types by count (descending)
			var types []string
			for mimeType := range info.Attachments.ByType {
				types = append(types, mimeType)
			}
			sort.Slice(types, func(i, j int) bool {
				return info.Attachments.ByType[types[i]] > info.Attachments.ByType[types[j]]
			})
			
			for _, mimeType := range types {
				count := info.Attachments.ByType[mimeType]
				fmt.Printf("    %s: %s\n", mimeType, formatNumber(count))
			}
		}
		
		if info.Attachments.OrphanedCount > 0 {
			fmt.Printf("  Orphaned: %s\n", formatNumber(info.Attachments.OrphanedCount))
		}
		fmt.Println()
	}

	// Contacts
	if info.Contacts > 0 {
		fmt.Printf("Contacts: %s\n\n", formatNumber(info.Contacts))
	}

	// Rejections and errors
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

	// Validation status
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
	return fmt.Sprintf("%s", addCommas(fmt.Sprintf("%d", n)))
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