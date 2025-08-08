package importer

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/coalescer"
	"github.com/phillipgreen/mobilecombackup/pkg/sms"
)

// SMSImporter handles SMS/MMS import operations
type SMSImporter struct {
	options     *ImportOptions
	coalescer   coalescer.Coalescer[sms.MessageEntry]
	smsWriter   *sms.XMLSMSWriter
	// TODO: Implement rejection writer
	// rejectWriter *XMLRejectionWriter
}


// NewSMSImporter creates a new SMS importer
func NewSMSImporter(options *ImportOptions) *SMSImporter {
	return &SMSImporter{
		options:   options,
		coalescer: coalescer.NewCoalescer[sms.MessageEntry](),
	}
}

// SetFilesToImport sets the files to import (for single file processing)
func (si *SMSImporter) SetFilesToImport(files []string) {
	si.options.Paths = files
}

// Import performs the SMS import operation
func (si *SMSImporter) Import() (*ImportSummary, error) {
	summary := &ImportSummary{
		YearStats:      make(map[int]*YearStat),
		Total:          &YearStat{},
		RejectionFiles: []string{},
	}

	// 1. Load existing repository for deduplication
	if err := si.LoadRepository(); err != nil {
		return nil, fmt.Errorf("failed to load repository: %w", err)
	}

	// 2. Process each import file
	for _, path := range si.options.Paths {
		// TODO: Handle directories by walking them
		fileStat, err := si.processFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to process file %s: %w", path, err)
		}
		summary.FilesProcessed++
		// Aggregate file stats into total
		summary.Total.Added += fileStat.Added
		summary.Total.Duplicates += fileStat.Duplicates
		summary.Total.Rejected += fileStat.Rejected
		summary.Total.Errors += fileStat.Errors
	}

	// 3. Write to repository (single write)
	if !si.options.DryRun {
		if err := si.WriteRepository(); err != nil {
			return nil, fmt.Errorf("failed to write repository: %w", err)
		}
	}

	return summary, nil
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
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := sms.NewXMLSMSReader("")
	lineNumber := 0
	err = reader.StreamMessagesFromReader(file, func(msg sms.Message) error {
		lineNumber++
		
		// Validate message
		violations := si.validateMessage(msg)
		if len(violations) > 0 {
			summary.Rejected++
			// TODO: Write rejection to file
			// Need to implement XMLRejectionWriter for SMS
			return nil
		}

		// Create entry and check for duplicates
		entry := sms.NewMessageEntry(msg)
		added := si.coalescer.Add(entry)
		if added {
			summary.Added++
		} else {
			summary.Duplicates++
		}

		// Report progress
		if lineNumber%100 == 0 && si.options.ProgressReporter != nil {
			si.options.ProgressReporter.UpdateProgress(lineNumber, 0)
		}

		return nil
	})

	if err != nil {
		summary.Errors++
		return summary, err
	}

	// Write violations file if there were rejections
	if summary.Rejected > 0 && !si.options.DryRun {
		// TODO: Write violations file
	}

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
			// Check for malformed attachment data
			if part.Data != "" && part.Data != "null" {
				// Try to decode base64
				// This would be done in attachment extraction phase
			}
		}
	}

	return violations
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
		writer, err := sms.NewXMLSMSWriter(si.options.RepoRoot)
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

// writeViolationsFile writes the violations YAML file
func (si *SMSImporter) writeViolationsFile(path string, violations []RejectedEntry) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create violations file: %w", err)
	}
	defer file.Close()

	// Write YAML format
	fmt.Fprintln(file, "violations:")
	for _, v := range violations {
		fmt.Fprintf(file, "  - line: %d\n", v.Line)
		fmt.Fprintln(file, "    violations:")
		for _, violation := range v.Violations {
			fmt.Fprintf(file, "      - %s\n", violation)
		}
	}

	return nil
}

// GetSummary returns the coalescer summary
func (si *SMSImporter) GetSummary() coalescer.Summary {
	return si.coalescer.GetSummary()
}

// calculateFileHash calculates SHA-256 hash of a file
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := hasher.Write([]byte(filePath)); err != nil {
		return "", err
	}
	
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}