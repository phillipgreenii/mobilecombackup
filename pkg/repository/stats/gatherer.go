package stats

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/phillipgreenii/mobilecombackup/pkg/attachments"
	"github.com/phillipgreenii/mobilecombackup/pkg/calls"
	"github.com/phillipgreenii/mobilecombackup/pkg/contacts"
	"github.com/phillipgreenii/mobilecombackup/pkg/sms"
)

// StatsGatherer collects statistics from a repository
type StatsGatherer struct {
	callsReader      calls.Reader
	smsReader        sms.Reader
	attachmentReader attachments.AttachmentReader
	contactsReader   contacts.Reader
	repoRoot         string
}

// NewStatsGatherer creates a new stats gatherer
func NewStatsGatherer(
	repoRoot string,
	callsReader calls.Reader,
	smsReader sms.Reader,
	attachmentReader attachments.AttachmentReader,
	contactsReader contacts.Reader,
) *StatsGatherer {
	return &StatsGatherer{
		callsReader:      callsReader,
		smsReader:        smsReader,
		attachmentReader: attachmentReader,
		contactsReader:   contactsReader,
		repoRoot:         repoRoot,
	}
}

// GatherStats collects all repository statistics
func (g *StatsGatherer) GatherStats(ctx context.Context) (*RepositoryStats, error) {
	stats := &RepositoryStats{
		Calls:      make(map[string]YearInfo),
		SMS:        make(map[string]MessageInfo),
		Rejections: make(map[string]int),
		Errors:     make(map[string]int),
	}

	// Gather calls statistics
	if callsStats, err := g.GatherCallsStats(ctx); err != nil {
		stats.Errors["calls"] = 1
	} else {
		stats.Calls = callsStats
	}

	// Gather SMS statistics
	if smsStats, err := g.GatherSMSStats(ctx); err != nil {
		stats.Errors["sms"] = 1
	} else {
		stats.SMS = smsStats
	}

	// Gather attachment statistics
	if attachmentStats, err := g.GatherAttachmentStats(ctx); err != nil {
		stats.Errors["attachments"] = 1
	} else {
		stats.Attachments = attachmentStats
	}

	// Gather contacts statistics
	if contactStats, err := g.GatherContactsStats(ctx); err != nil {
		stats.Errors["contacts"] = 1
	} else {
		stats.Contacts = contactStats
	}

	// Basic validation check
	stats.ValidationOK = len(stats.Errors) == 0

	return stats, nil
}

// GatherCallsStats gathers call statistics by year
func (g *StatsGatherer) GatherCallsStats(ctx context.Context) (map[string]YearInfo, error) {
	result := make(map[string]YearInfo)

	years, err := g.callsReader.GetAvailableYears(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get available years: %w", err)
	}

	for _, year := range years {
		yearStats := YearInfo{}

		// Get count
		count, err := g.callsReader.GetCallsCount(ctx, year)
		if err != nil {
			return nil, fmt.Errorf("failed to get calls count for year %d: %w", year, err)
		}
		yearStats.Count = count

		// Get date range by streaming calls
		var earliest, latest time.Time
		err = g.callsReader.StreamCallsForYear(ctx, year, func(call calls.Call) error {
			timestamp := call.Timestamp()
			earliest, latest = updateDateRange(earliest, latest, timestamp)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to stream calls for year %d: %w", year, err)
		}

		yearStats.Earliest = earliest
		yearStats.Latest = latest

		result[fmt.Sprintf("%d", year)] = yearStats
	}

	return result, nil
}

// GatherSMSStats gathers SMS/MMS statistics by year
func (g *StatsGatherer) GatherSMSStats(ctx context.Context) (map[string]MessageInfo, error) {
	result := make(map[string]MessageInfo)

	years, err := g.smsReader.GetAvailableYears(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get available years: %w", err)
	}

	for _, year := range years {
		yearStats, err := g.gatherSMSStatsForYear(ctx, year)
		if err != nil {
			return nil, fmt.Errorf("failed to gather SMS stats for year %d: %w", year, err)
		}
		result[fmt.Sprintf("%d", year)] = yearStats
	}

	return result, nil
}

// gatherSMSStatsForYear gathers SMS statistics for a specific year
func (g *StatsGatherer) gatherSMSStatsForYear(ctx context.Context, year int) (MessageInfo, error) {
	messageStats := MessageInfo{}

	// Get total count
	totalCount, err := g.smsReader.GetMessageCount(ctx, year)
	if err != nil {
		return messageStats, fmt.Errorf("failed to get message count: %w", err)
	}
	messageStats.TotalCount = totalCount

	// Count SMS vs MMS and get date range by streaming messages
	var earliest, latest time.Time
	err = g.smsReader.StreamMessagesForYear(ctx, year, func(msg sms.Message) error {
		timestamp := time.Unix(msg.GetDate()/1000, (msg.GetDate()%1000)*int64(time.Millisecond))
		earliest, latest = updateDateRange(earliest, latest, timestamp)
		countMessageType(msg, &messageStats)
		return nil
	})
	if err != nil {
		return messageStats, fmt.Errorf("failed to stream messages: %w", err)
	}

	messageStats.Earliest = earliest
	messageStats.Latest = latest

	return messageStats, nil
}

// GatherAttachmentStats gathers attachment statistics
func (g *StatsGatherer) GatherAttachmentStats(ctx context.Context) (AttachmentInfo, error) {
	attachmentInfo := AttachmentInfo{
		ByType: make(map[string]int),
	}

	// Get all attachments
	attachmentList, err := g.attachmentReader.ListAttachments()
	if err != nil {
		return attachmentInfo, fmt.Errorf("failed to list attachments: %w", err)
	}

	attachmentInfo.Count = len(attachmentList)

	// Calculate total size and type distribution
	for _, attachment := range attachmentList {
		attachmentInfo.TotalSize += attachment.Size

		// Determine type from file extension
		mimeType := determineMimeTypeFromPath(attachment.Path)
		attachmentInfo.ByType[mimeType]++
	}

	// Get referenced attachments
	referencedHashes, err := g.smsReader.GetAllAttachmentRefs(ctx)
	if err != nil {
		return attachmentInfo, fmt.Errorf("failed to get attachment references: %w", err)
	}

	// Find orphaned attachments
	orphanedAttachments, err := g.attachmentReader.FindOrphanedAttachments(referencedHashes)
	if err != nil {
		return attachmentInfo, fmt.Errorf("failed to find orphaned attachments: %w", err)
	}

	attachmentInfo.Referenced = attachmentInfo.Count - len(orphanedAttachments)
	attachmentInfo.Orphaned = len(orphanedAttachments)

	return attachmentInfo, nil
}

// GatherContactsStats gathers contact statistics
func (g *StatsGatherer) GatherContactsStats(ctx context.Context) (ContactInfo, error) {
	contactInfo := ContactInfo{}

	// Load contacts
	err := g.contactsReader.LoadContacts(ctx)
	if err != nil {
		return contactInfo, fmt.Errorf("failed to load contacts: %w", err)
	}

	// Get contact count
	contactInfo.Count = g.contactsReader.GetContactsCount()

	// Get all contacts to count names
	allContacts, err := g.contactsReader.GetAllContacts(ctx)
	if err != nil {
		return contactInfo, fmt.Errorf("failed to get all contacts: %w", err)
	}

	// Count contacts with names vs phone-only
	for _, contact := range allContacts {
		if contact.Name != "" && contact.Name != "(Unknown)" {
			contactInfo.WithNames++
		} else {
			contactInfo.PhoneOnly++
		}
	}

	return contactInfo, nil
}

// updateDateRange updates the earliest and latest timestamps
func updateDateRange(earliest, latest, timestamp time.Time) (time.Time, time.Time) {
	if earliest.IsZero() || timestamp.Before(earliest) {
		earliest = timestamp
	}
	if latest.IsZero() || timestamp.After(latest) {
		latest = timestamp
	}
	return earliest, latest
}

// countMessageType counts SMS vs MMS messages
func countMessageType(msg sms.Message, messageStats *MessageInfo) {
	switch msg.(type) {
	case sms.SMS:
		messageStats.SMSCount++
	case sms.MMS:
		messageStats.MMSCount++
	}
}

// determineMimeTypeFromPath determines the MIME type from a file path
func determineMimeTypeFromPath(path string) string {
	ext := filepath.Ext(path)
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
	case ".amr":
		return "audio/amr"
	case ".mp3":
		return "audio/mp3"
	default:
		return "application/octet-stream"
	}
}
