// Package importer provides orchestration functionality for importing mobile backup data.
// It coordinates the processing of calls, SMS/MMS, and attachment data, handling
// deduplication, validation, and repository organization during the import process.
package importer

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/phillipgreen/mobilecombackup/pkg/calls"
	"github.com/phillipgreen/mobilecombackup/pkg/coalescer"
	"github.com/phillipgreen/mobilecombackup/pkg/security"
)

// CallsImporter handles importing call backup files
type CallsImporter struct {
	options         *ImportOptions
	coalescer       coalescer.Coalescer[calls.CallEntry]
	writer          *calls.XMLCallsWriter
	validator       *CallValidator
	rejWriter       RejectionWriter
	contactsManager ContactsManager // Now uses interface instead of concrete type
	yearTracker     *YearTracker
}

// NewCallsImporter creates a new calls importer
func NewCallsImporter(
	options *ImportOptions,
	contactsManager ContactsManager, // Now uses interface
	yearTracker *YearTracker,
) (*CallsImporter, error) {
	// Set defaults for size limits if not specified
	options.SetDefaults()

	writer, err := calls.NewXMLCallsWriter(filepath.Join(options.RepoRoot, "calls"))
	if err != nil {
		return nil, fmt.Errorf("failed to create calls writer: %w", err)
	}

	return &CallsImporter{
		options:         options,
		coalescer:       calls.NewCallCoalescer(),
		writer:          writer,
		validator:       NewCallValidator(),
		rejWriter:       NewXMLRejectionWriter(options.RepoRoot),
		contactsManager: contactsManager,
		yearTracker:     yearTracker,
	}, nil
}

// NewCallsImporterWithDependencies creates a new calls importer with injected dependencies
func NewCallsImporterWithDependencies(
	options *ImportOptions,
	contactsManager ContactsManager,
	yearTracker *YearTracker,
	callsReader CallsReader,
) (*CallsImporter, error) {
	// Set defaults for size limits if not specified
	options.SetDefaults()

	writer, err := calls.NewXMLCallsWriter(filepath.Join(options.RepoRoot, "calls"))
	if err != nil {
		return nil, fmt.Errorf("failed to create calls writer: %w", err)
	}

	return &CallsImporter{
		options:         options,
		coalescer:       calls.NewCallCoalescer(),
		writer:          writer,
		validator:       NewCallValidator(),
		rejWriter:       NewXMLRejectionWriter(options.RepoRoot),
		contactsManager: contactsManager,
		yearTracker:     yearTracker,
	}, nil
}

// LoadRepository loads existing calls for deduplication
func (ci *CallsImporter) LoadRepository() error {
	reader := calls.NewXMLCallsReader(ci.options.RepoRoot)

	// Stream all existing calls into the coalescer
	var existingCalls []calls.CallEntry
	err := reader.StreamCalls(func(call *calls.Call) error {
		entry := calls.CallEntry{Call: call}
		existingCalls = append(existingCalls, entry)

		// Track initial entry by year
		if ci.yearTracker != nil {
			ci.yearTracker.TrackInitialEntry(entry.Year())
		}

		return nil
	})

	if err != nil {
		// Empty repository is OK
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to stream existing calls: %w", err)
		}
	}

	// Load all existing calls at once
	if len(existingCalls) > 0 {
		if err := ci.coalescer.LoadExisting(existingCalls); err != nil {
			return fmt.Errorf("failed to load existing calls: %w", err)
		}
	}

	count := len(existingCalls)

	if ci.options.Verbose && !ci.options.Quiet {
		fmt.Printf("Loaded %d existing calls from repository\n", count)
	}

	return nil
}

// ImportFile imports calls from a single file
func (ci *CallsImporter) ImportFile(filename string) (*YearStat, error) {
	// Parse XML file
	root, err := ci.parseCallsXML(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to parse calls XML file %s: %w", filepath.Base(filename), err)
	}

	// Process all calls and track statistics
	stat, rejections := ci.processCalls(root.Calls)

	// Write rejections if any
	ci.writeRejections(filename, rejections, stat)

	return stat, nil
}

// WriteRepository writes all accumulated calls to the repository
func (ci *CallsImporter) WriteRepository() error {
	if ci.options.DryRun {
		return nil
	}

	// Get all calls sorted by timestamp
	allEntries := ci.coalescer.GetAll()

	// Group by year
	callsByYear := make(map[int][]*calls.Call)
	for _, entry := range allEntries {
		year := entry.Year()
		callsByYear[year] = append(callsByYear[year], entry.Call)
	}

	// Write each year
	for year, yearCalls := range callsByYear {
		filename := fmt.Sprintf("calls-%d.xml", year)
		if err := ci.writer.WriteCalls(filename, yearCalls); err != nil {
			return fmt.Errorf("failed to write calls for year %d: %w", year, err)
		}
	}

	return nil
}

// GetSummary returns the import summary
func (ci *CallsImporter) GetSummary() coalescer.Summary {
	return ci.coalescer.GetSummary()
}

// CallValidator validates call entries
type CallValidator struct{}

// NewCallValidator creates a new call validator
func NewCallValidator() *CallValidator {
	return &CallValidator{}
}

// Validate checks if a call is valid
func (v *CallValidator) Validate(call *calls.Call) []string {
	var violations []string

	// Required: valid timestamp
	if call.Date <= 0 {
		violations = append(violations, "missing-timestamp")
	}

	// Required: phone number
	if strings.TrimSpace(call.Number) == "" {
		violations = append(violations, "missing-number")
	}

	// Required: valid call type
	switch call.Type {
	case calls.Incoming, calls.Outgoing, calls.Missed, calls.Voicemail:
		// Valid
	default:
		violations = append(violations, fmt.Sprintf("invalid-type: %d", call.Type))
	}

	// Duration should be non-negative
	if call.Duration < 0 {
		violations = append(violations, "negative-duration")
	}

	return violations
}

// extractContact extracts contact information from a call record
// resolveContact attempts to resolve contact name from existing contacts.
// Returns true if the contact was successfully resolved from repository data.
func (ci *CallsImporter) resolveContact(call *calls.Call) bool {
	if ci.contactsManager == nil || call.Number == "" {
		return false
	}

	if knownName, exists := ci.contactsManager.GetContactByNumber(call.Number); exists {
		call.ContactName = knownName
		return true // Contact was resolved from existing data
	}

	return false // Contact was not found in existing data
}

func (ci *CallsImporter) extractContact(call *calls.Call, wasResolved bool) {
	if ci.contactsManager == nil {
		return
	}

	// Skip extraction if contact was already resolved from existing data
	if wasResolved {
		return
	}

	// Extract contact names if both number and contact name are present
	if call.Number != "" && call.ContactName != "" {
		// Split contact names by comma and process each separately
		contactNames := strings.Split(call.ContactName, ",")
		for _, name := range contactNames {
			name = strings.TrimSpace(name)
			if name != "" && !isUnknownContact(name) {
				ci.contactsManager.AddUnprocessedContact(call.Number, name)
			}
		}
	}
}

// parseCallsXML parses the XML file and returns the root structure
func (ci *CallsImporter) parseCallsXML(filename string) (*struct {
	XMLName xml.Name
	Count   string `xml:"count,attr"`
	Calls   []struct {
		XMLName      xml.Name
		Number       string `xml:"number,attr"`
		Duration     int    `xml:"duration,attr"`
		Date         int64  `xml:"date,attr"`
		Type         int    `xml:"type,attr"`
		ReadableDate string `xml:"readable_date,attr"`
		ContactName  string `xml:"contact_name,attr"`
	} `xml:"call"`
}, error) {
	file, err := os.Open(filename) // #nosec G304
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Apply size limit to prevent DoS attacks (BUG-051)
	limitedReader := &io.LimitedReader{R: file, N: ci.options.MaxXMLSize}

	// Parse the XML file with size limit
	decoder := security.NewSecureXMLDecoder(limitedReader)

	// Find the root element
	var root struct {
		XMLName xml.Name
		Count   string `xml:"count,attr"`
		Calls   []struct {
			XMLName      xml.Name
			Number       string `xml:"number,attr"`
			Duration     int    `xml:"duration,attr"`
			Date         int64  `xml:"date,attr"`
			Type         int    `xml:"type,attr"`
			ReadableDate string `xml:"readable_date,attr"`
			ContactName  string `xml:"contact_name,attr"`
		} `xml:"call"`
	}

	if err := decoder.Decode(&root); err != nil {
		// Check if the error was due to size limit exceeded
		if err == io.EOF && limitedReader.N == 0 {
			return nil, security.NewFileSizeLimitExceededError(
				filepath.Base(filename),
				ci.options.MaxXMLSize,
				0, // Don't know actual size
				"XML parsing")
		}
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	return &root, nil
}

// processCalls processes all calls and returns statistics and rejections
func (ci *CallsImporter) processCalls(xmlCalls []struct {
	XMLName      xml.Name
	Number       string `xml:"number,attr"`
	Duration     int    `xml:"duration,attr"`
	Date         int64  `xml:"date,attr"`
	Type         int    `xml:"type,attr"`
	ReadableDate string `xml:"readable_date,attr"`
	ContactName  string `xml:"contact_name,attr"`
}) (*YearStat, []RejectedEntry) {
	stat := &YearStat{}
	var rejections []RejectedEntry

	// Process each call
	for i, xmlCall := range xmlCalls {
		// Convert to Call struct
		call := &calls.Call{
			Number:       xmlCall.Number,
			Duration:     xmlCall.Duration,
			Date:         xmlCall.Date,
			Type:         calls.CallType(xmlCall.Type),
			ReadableDate: xmlCall.ReadableDate,
			ContactName:  xmlCall.ContactName,
		}

		// Validate the call
		violations := ci.validator.Validate(call)
		if len(violations) > 0 {
			// Capture the original XML for the rejection file
			callXML, _ := xml.Marshal(xmlCall)
			rejections = append(rejections, RejectedEntry{
				Line:       i + 2, // +2 for XML header and root element
				Data:       string(callXML),
				Violations: violations,
			})
			stat.Rejected++
			continue
		}

		// Resolve contact from existing contacts first, then extract if needed
		wasResolved := ci.resolveContact(call)
		ci.extractContact(call, wasResolved)

		// Add to coalescer (checks for duplicates)
		entry := calls.CallEntry{Call: call}
		wasAdded := ci.coalescer.Add(entry)

		// Update file-level statistics
		if wasAdded {
			stat.Added++
		} else {
			stat.Duplicates++
		}

		// Track entry by year
		if ci.yearTracker != nil {
			ci.yearTracker.TrackImportEntry(entry.Year(), wasAdded)
		}

		// Report progress every 100 entries
		if (i+1)%100 == 0 && ci.options.ProgressReporter != nil {
			ci.options.ProgressReporter.UpdateProgress(i+1, len(xmlCalls))
		}
	}

	return stat, rejections
}

// writeRejections writes rejection files if there are any rejections
func (ci *CallsImporter) writeRejections(filename string, rejections []RejectedEntry, stat *YearStat) {
	if len(rejections) > 0 && !ci.options.DryRun {
		rejFile, err := ci.rejWriter.WriteRejections(filename, rejections)
		if err != nil {
			stat.Errors++
			if !ci.options.Quiet {
				fmt.Printf("Warning: failed to write rejection file: %v\n", err)
			}
		} else if ci.options.Verbose && !ci.options.Quiet {
			fmt.Printf("Created rejection file: %s\n", rejFile)
		}
	}
}

// isUnknownContact checks if a contact name represents an unknown contact
// and should be ignored during contact extraction
func isUnknownContact(contactName string) bool {
	return contactName == "" || contactName == "(Unknown)" || contactName == "null"
}
