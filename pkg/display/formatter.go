// Package display provides formatting and output functionality for repository information.
package display

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/phillipgreenii/mobilecombackup/pkg/repository/stats"
)

// RepositoryInfo represents repository information for display
type RepositoryInfo struct {
	Version      string
	CreatedAt    time.Time
	Calls        map[string]stats.YearInfo
	SMS          map[string]stats.MessageInfo
	Attachments  AttachmentInfo
	Contacts     ContactInfo
	Rejections   map[string]int
	Errors       map[string]int
	ValidationOK bool
}

// AttachmentInfo contains attachment statistics for display
type AttachmentInfo struct {
	Count         int
	TotalSize     int64
	OrphanedCount int
	ByType        map[string]int
}

// ContactInfo contains contact statistics for display
type ContactInfo struct {
	Count int
}

// Formatter handles repository information formatting and output
type Formatter struct {
	writer io.Writer
	quiet  bool
}

// NewFormatter creates a new Formatter instance
func NewFormatter(writer io.Writer, quiet bool) *Formatter {
	return &Formatter{
		writer: writer,
		quiet:  quiet,
	}
}

// FormatRepositoryInfo formats and outputs repository information
func (f *Formatter) FormatRepositoryInfo(info *RepositoryInfo, repoPath string) {
	if f.quiet {
		return
	}

	f.printRepositoryHeader(info, repoPath)
	f.printCallsStatistics(info)
	f.printMessagesStatistics(info)
	f.printAttachmentsStatistics(info)
	f.printContacts(info)
	f.printIssues(info)
	f.printValidationStatus(info)
}

// printRepositoryHeader prints the repository header information
func (f *Formatter) printRepositoryHeader(info *RepositoryInfo, repoPath string) {
	_, _ = fmt.Fprintf(f.writer, "Repository: %s\n", repoPath)
	if info.Version != "" {
		_, _ = fmt.Fprintf(f.writer, "Version: %s\n", info.Version)
	}
	if !info.CreatedAt.IsZero() {
		_, _ = fmt.Fprintf(f.writer, "Created: %s\n", info.CreatedAt.Format(time.RFC3339))
	}
	_, _ = fmt.Fprintln(f.writer)
}

// printCallsStatistics prints call statistics
func (f *Formatter) printCallsStatistics(info *RepositoryInfo) {
	_, _ = fmt.Fprintln(f.writer, "Calls:")
	if len(info.Calls) > 0 {
		var totalCalls int
		years := getSortedCallsYears(info.Calls)

		for _, year := range years {
			yearInfo := info.Calls[year]
			totalCalls += yearInfo.Count
			_, _ = fmt.Fprintf(f.writer, "  %s: %s calls", year, FormatNumber(yearInfo.Count))
			f.printDateRange(yearInfo.Earliest, yearInfo.Latest)
			_, _ = fmt.Fprintln(f.writer)
		}
		_, _ = fmt.Fprintf(f.writer, "  Total: %s calls\n", FormatNumber(totalCalls))
	} else {
		_, _ = fmt.Fprintln(f.writer, "  Total: 0 calls")
	}
	_, _ = fmt.Fprintln(f.writer)
}

// printMessagesStatistics prints SMS/MMS statistics
func (f *Formatter) printMessagesStatistics(info *RepositoryInfo) {
	_, _ = fmt.Fprintln(f.writer, "Messages:")
	if len(info.SMS) > 0 {
		totalMessages, totalSMS, totalMMS := calculateMessageTotals(info)
		years := getSortedMessageYears(info.SMS)

		for _, year := range years {
			msgInfo := info.SMS[year]
			_, _ = fmt.Fprintf(f.writer, "  %s: %s messages (%s SMS, %s MMS)",
				year,
				FormatNumber(msgInfo.TotalCount),
				FormatNumber(msgInfo.SMSCount),
				FormatNumber(msgInfo.MMSCount))
			f.printDateRange(msgInfo.Earliest, msgInfo.Latest)
			_, _ = fmt.Fprintln(f.writer)
		}
		_, _ = fmt.Fprintf(f.writer, "  Total: %s messages (%s SMS, %s MMS)\n",
			FormatNumber(totalMessages),
			FormatNumber(totalSMS),
			FormatNumber(totalMMS))
	} else {
		_, _ = fmt.Fprintln(f.writer, "  Total: 0 messages (0 SMS, 0 MMS)")
	}
	_, _ = fmt.Fprintln(f.writer)
}

// printAttachmentsStatistics prints attachment statistics
func (f *Formatter) printAttachmentsStatistics(info *RepositoryInfo) {
	_, _ = fmt.Fprintln(f.writer, "Attachments:")
	if info.Attachments.Count > 0 {
		_, _ = fmt.Fprintf(f.writer, "  Count: %s\n", FormatNumber(info.Attachments.Count))
		_, _ = fmt.Fprintf(f.writer, "  Total Size: %s\n", FormatBytes(info.Attachments.TotalSize))

		f.printAttachmentTypes(info)
		f.printOrphanedAttachments(info)
	} else {
		_, _ = fmt.Fprintln(f.writer, "  Count: 0")
		_, _ = fmt.Fprintln(f.writer, "  Total Size: 0 B")
	}
	_, _ = fmt.Fprintln(f.writer)
}

// printAttachmentTypes prints attachment types breakdown
func (f *Formatter) printAttachmentTypes(info *RepositoryInfo) {
	if len(info.Attachments.ByType) > 0 {
		_, _ = fmt.Fprintln(f.writer, "  Types:")
		types := getSortedAttachmentTypes(info.Attachments.ByType)
		for _, mimeType := range types {
			count := info.Attachments.ByType[mimeType]
			_, _ = fmt.Fprintf(f.writer, "    %s: %s\n", mimeType, FormatNumber(count))
		}
	}
}

// printOrphanedAttachments prints orphaned attachment count
func (f *Formatter) printOrphanedAttachments(info *RepositoryInfo) {
	if info.Attachments.OrphanedCount > 0 {
		_, _ = fmt.Fprintf(f.writer, "  Orphaned: %s\n", FormatNumber(info.Attachments.OrphanedCount))
	}
}

// printContacts prints contact count
func (f *Formatter) printContacts(info *RepositoryInfo) {
	_, _ = fmt.Fprintf(f.writer, "Contacts: %s\n\n", FormatNumber(info.Contacts.Count))
}

// printIssues prints rejection and error statistics
func (f *Formatter) printIssues(info *RepositoryInfo) {
	if len(info.Rejections) > 0 || len(info.Errors) > 0 {
		_, _ = fmt.Fprintln(f.writer, "Issues:")
		for component, count := range info.Rejections {
			_, _ = fmt.Fprintf(f.writer, "  Rejections (%s): %s\n", component, FormatNumber(count))
		}
		for component, count := range info.Errors {
			_, _ = fmt.Fprintf(f.writer, "  Errors (%s): %s\n", component, FormatNumber(count))
		}
		_, _ = fmt.Fprintln(f.writer)
	}
}

// printValidationStatus prints validation status
func (f *Formatter) printValidationStatus(info *RepositoryInfo) {
	if info.ValidationOK {
		_, _ = fmt.Fprintln(f.writer, "Validation: OK")
	} else {
		_, _ = fmt.Fprintln(f.writer, "Validation: Issues detected")
	}
}

// printDateRange prints date range if available
func (f *Formatter) printDateRange(earliest, latest time.Time) {
	if !earliest.IsZero() && !latest.IsZero() {
		_, _ = fmt.Fprintf(f.writer, " (%s - %s)",
			earliest.Format("Jan 2"),
			latest.Format("Jan 2"))
	}
}

// FormatNumber formats a number with comma separators
func FormatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return addCommas(fmt.Sprintf("%d", n))
}

// FormatBytes formats byte counts with appropriate units
func FormatBytes(bytes int64) string {
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

// addCommas adds comma separators to a number string
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
