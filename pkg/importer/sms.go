package importer

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/coalescer"
	"github.com/phillipgreen/mobilecombackup/pkg/contacts"
	"github.com/phillipgreen/mobilecombackup/pkg/security"
	"github.com/phillipgreen/mobilecombackup/pkg/sms"
)

// SMSImporter handles SMS/MMS import operations
type SMSImporter struct {
	options             *ImportOptions
	coalescer           coalescer.Coalescer[sms.MessageEntry]
	contactsManager     *contacts.ContactsManager
	attachmentExtractor *sms.AttachmentExtractor
	attachmentStats     *sms.AttachmentExtractionStats
	contentTypeConfig   sms.ContentTypeConfig
	yearTracker         *YearTracker
	// TODO: Implement rejection writer
	// rejectWriter *XMLRejectionWriter
}

// NewSMSImporter creates a new SMS importer
func NewSMSImporter(options *ImportOptions, contactsManager *contacts.ContactsManager, yearTracker *YearTracker) *SMSImporter {
	// Set defaults for size limits if not specified
	options.SetDefaults()

	return &SMSImporter{
		options:             options,
		coalescer:           coalescer.NewCoalescer[sms.MessageEntry](),
		contactsManager:     contactsManager,
		attachmentExtractor: sms.NewAttachmentExtractor(options.RepoRoot),
		attachmentStats:     sms.NewAttachmentExtractionStats(),
		contentTypeConfig:   sms.GetDefaultContentTypeConfig(),
		yearTracker:         yearTracker,
	}
}

// SetFilesToImport sets the files to import (for single file processing)
func (si *SMSImporter) SetFilesToImport(files []string) {
	si.options.Paths = files
}

// LoadRepository loads existing SMS/MMS from the repository for deduplication
func (si *SMSImporter) LoadRepository() error {
	reader := sms.NewXMLSMSReader(si.options.RepoRoot)

	// Get all available years
	years, err := reader.GetAvailableYears()
	if err != nil {
		return fmt.Errorf("failed to get available years: %w", err)
	}

	// Load messages from each year
	totalLoaded := 0
	for _, year := range years {
		yearCount := 0
		err := reader.StreamMessagesForYear(year, func(msg sms.Message) error {
			entry := sms.NewMessageEntry(msg)
			si.coalescer.Add(entry)
			yearCount++
			totalLoaded++

			// Track initial entry by year
			if si.yearTracker != nil {
				si.yearTracker.TrackInitialEntry(entry.Year())
			}

			// Report progress every 100 messages
			if totalLoaded%100 == 0 && si.options.ProgressReporter != nil {
				si.options.ProgressReporter.UpdateProgress(totalLoaded, 0)
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to load messages from year %d: %w", year, err)
		}

		if !si.options.Quiet && si.options.Verbose {
			fmt.Printf("Loaded %d messages from %d\n", yearCount, year)
		}
	}

	if !si.options.Quiet {
		fmt.Printf("Repository loaded: %d existing messages\n", totalLoaded)
	}

	return nil
}

// ImportFile imports a single SMS backup file
func (si *SMSImporter) ImportFile(filePath string) (*YearStat, error) {
	return si.processFile(filePath)
}

// processFile processes a single SMS backup file
func (si *SMSImporter) processFile(filePath string) (*YearStat, error) {
	summary := &YearStat{}

	// Calculate file hash for rejection file naming
	fileHash, err := calculateFileHash(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate file hash: %w", err)
	}

	// Create rejection writer for this file
	timestamp := time.Now().Format("20060102-150405")
	_ = filepath.Join(si.options.RepoRoot, "rejected",
		fmt.Sprintf("sms-%s-%s-rejects.xml", fileHash[:8], timestamp))
	_ = filepath.Join(si.options.RepoRoot, "rejected",
		fmt.Sprintf("sms-%s-%s-violations.yaml", fileHash[:8], timestamp))

	// TODO: Create rejection writer
	// si.rejectWriter = NewXMLRejectionWriter(rejectPath)
	// defer si.rejectWriter.Close()

	// Process messages from file
	file, err := os.Open(filePath) // #nosec G304
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Apply size limit to prevent DoS attacks (BUG-051)
	limitedReader := &io.LimitedReader{R: file, N: si.options.MaxXMLSize}

	reader := sms.NewXMLSMSReader("")
	lineNumber := 0
	err = reader.StreamMessagesFromReader(limitedReader, func(msg sms.Message) error {
		lineNumber++

		// Validate message
		violations := si.validateMessage(msg)
		if len(violations) > 0 {
			summary.Rejected++
			// TODO: Write rejection to file
			// Need to implement XMLRejectionWriter for SMS
			return nil
		}

		// Check message size limit (BUG-051)
		if err := si.validateMessageSize(msg); err != nil {
			summary.Rejected++
			// TODO: Write rejection with size limit reason
			return nil
		}

		// Extract attachments from MMS (after validation, before deduplication)
		if mmsMsg, ok := msg.(sms.MMS); ok {
			extractionSummary, err := si.attachmentExtractor.ExtractAttachmentsFromMMS(&mmsMsg, si.contentTypeConfig)
			if err != nil {
				// Reject the entire message if attachment extraction fails
				summary.Rejected++
				// TODO: Write rejection with reason "attachment-extraction-error"
				return nil
			}

			// Update overall attachment statistics
			si.attachmentStats.AddMMSExtractionSummary(extractionSummary)

			// Update the message interface with the modified MMS
			msg = mmsMsg
		}

		// Extract contact information for valid messages
		si.extractContacts(msg)

		// Create entry and check for duplicates
		entry := sms.NewMessageEntry(msg)
		added := si.coalescer.Add(entry)

		// Update file-level statistics
		if added {
			summary.Added++
		} else {
			summary.Duplicates++
		}

		// Track entry by year
		if si.yearTracker != nil {
			si.yearTracker.TrackImportEntry(entry.Year(), added)
		}

		// Report progress
		if lineNumber%100 == 0 && si.options.ProgressReporter != nil {
			si.options.ProgressReporter.UpdateProgress(lineNumber, 0)
		}

		return nil
	})

	if err != nil {
		// Check if the error was due to size limit exceeded
		if err == io.EOF && limitedReader.N == 0 {
			summary.Errors++
			return summary, security.NewFileSizeLimitExceededError(
				filepath.Base(filePath),
				si.options.MaxXMLSize,
				0, // Don't know actual size
				"SMS XML parsing")
		}
		summary.Errors++
		return summary, err
	}

	// Note: Violation file writing is handled by rejection writer when rejections occur

	return summary, nil
}

// validateMessage validates an SMS/MMS message
func (si *SMSImporter) validateMessage(msg sms.Message) []string {
	var violations []string

	// Required: date field must be non-zero
	if msg.GetDate() == 0 {
		violations = append(violations, "missing-timestamp")
	}

	// Required: address field must be non-empty
	if msg.GetAddress() == "" {
		violations = append(violations, "missing-address")
	}

	// For SMS, type must be valid
	if smsMsg, ok := msg.(sms.SMS); ok {
		if smsMsg.Type != sms.ReceivedMessage && smsMsg.Type != sms.SentMessage {
			violations = append(violations, "invalid-type")
		}
	}

	// For MMS, msg_box must be valid
	if mmsMsg, ok := msg.(sms.MMS); ok {
		if mmsMsg.MsgBox != 1 && mmsMsg.MsgBox != 2 {
			violations = append(violations, "invalid-msg-box")
		}

		// Validate parts
		for i, part := range mmsMsg.Parts {
			if part.ContentType == "" {
				violations = append(violations, fmt.Sprintf("part-%d-missing-content-type", i))
			}
			// Note: Attachment data validation is performed during extraction phase
		}
	}

	return violations
}

// validateMessageSize validates that a message doesn't exceed size limits (BUG-051)
func (si *SMSImporter) validateMessageSize(msg sms.Message) error {
	// Calculate approximate message size
	// For SMS: body + address + basic fields
	// For MMS: body + address + all parts data

	var messageSize int64

	// Base message size (address, date, type fields)
	messageSize += int64(len(msg.GetAddress()))
	messageSize += 50 // Estimated for date, type, and other basic fields

	if smsMsg, ok := msg.(sms.SMS); ok {
		// SMS: add body size
		messageSize += int64(len(smsMsg.Body))
	} else if mmsMsg, ok := msg.(sms.MMS); ok {
		// MMS: add subject and all parts
		messageSize += int64(len(mmsMsg.Sub))
		for _, part := range mmsMsg.Parts {
			messageSize += int64(len(part.Data))
			messageSize += int64(len(part.ContentType))
			messageSize += int64(len(part.Name))
		}
	}

	// Check against limit
	if messageSize > si.options.MaxMessageSize {
		return security.NewFileSizeLimitExceededError(
			"message",
			si.options.MaxMessageSize,
			messageSize,
			"Message size validation")
	}

	return nil
}

// WriteRepository writes the coalesced messages to the repository
func (si *SMSImporter) WriteRepository() error {
	// Get all entries from coalescer
	entries := si.coalescer.GetAll()

	// Sort by timestamp
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp().Before(entries[j].Timestamp())
	})

	// Partition by year
	yearMap := make(map[int][]sms.MessageEntry)
	for _, entry := range entries {
		year := entry.Year()
		yearMap[year] = append(yearMap[year], entry)
	}

	// Write each year
	for year, yearEntries := range yearMap {
		// Convert entries back to messages
		messages := make([]sms.Message, len(yearEntries))
		for i, entry := range yearEntries {
			messages[i] = entry.Message
		}

		// Create writer for this year
		smsDir := filepath.Join(si.options.RepoRoot, "sms")
		writer, err := sms.NewXMLSMSWriter(smsDir)
		if err != nil {
			return fmt.Errorf("failed to create writer: %w", err)
		}
		filename := fmt.Sprintf("sms-%d.xml", year)
		if err := writer.WriteMessages(filename, messages); err != nil {
			return fmt.Errorf("failed to write messages for year %d: %w", year, err)
		}

		if !si.options.Quiet {
			fmt.Printf("Wrote %d messages to year %d\n", len(messages), year)
		}
	}

	return nil
}

// GetSummary returns the coalescer summary
func (si *SMSImporter) GetSummary() coalescer.Summary {
	return si.coalescer.GetSummary()
}

// GetAttachmentStats returns attachment statistics
func (si *SMSImporter) GetAttachmentStats() *AttachmentStat {
	return &AttachmentStat{
		Total:      si.attachmentStats.ExtractedCount + si.attachmentStats.ReferencedCount,
		New:        si.attachmentStats.ExtractedCount,
		Duplicates: si.attachmentStats.ReferencedCount,
	}
}

// calculateFileHash calculates SHA-256 hash of a file
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath) // #nosec G304
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	hasher := sha256.New()
	if _, err := hasher.Write([]byte(filePath)); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// extractContacts extracts contact names from SMS/MMS messages
func (si *SMSImporter) extractContacts(msg sms.Message) {
	if si.contactsManager == nil {
		return
	}

	// Extract from SMS
	if smsMsg, ok := msg.(sms.SMS); ok {
		si.extractSMSContact(smsMsg)
		return
	}

	// Extract from MMS (both primary address and additional addresses)
	if mmsMsg, ok := msg.(sms.MMS); ok {
		si.extractMMSContacts(mmsMsg)
		return
	}
}

// extractSMSContact extracts contact from SMS message
func (si *SMSImporter) extractSMSContact(smsMsg sms.SMS) {
	si.processContactInfo(smsMsg.Address, smsMsg.ContactName)
}

// extractMMSContacts extracts contacts from MMS message
func (si *SMSImporter) extractMMSContacts(mmsMsg sms.MMS) {
	si.processContactInfo(mmsMsg.Address, mmsMsg.ContactName)

	// Note: MMSAddress does not contain ContactName fields,
	// so additional addresses cannot be processed for contact extraction
}

// processContactInfo processes contact information from address and contact name
func (si *SMSImporter) processContactInfo(address, contactName string) {
	// Extract primary address contact, handling multiple addresses and contact names
	if address == "" || contactName == "" {
		return
	}

	// Check if this is a multi-address message (contains ~ separator)
	if strings.Contains(address, "~") {
		// Use the multi-address parsing method that handles ~ and , separators
		err := si.contactsManager.AddUnprocessedContacts(address, contactName)
		if err != nil {
			// If multi-address parsing fails, don't use fallback - this prevents double processing
			// Just skip this contact extraction
			return
		}
		return
	}

	// Single address - split contact names by comma and process each separately
	contactNames := strings.Split(contactName, ",")
	for _, name := range contactNames {
		name = strings.TrimSpace(name)
		if name != "" && !isUnknownContact(name) {
			si.contactsManager.AddUnprocessedContact(address, name)
		}
	}
}
