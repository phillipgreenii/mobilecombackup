package cmd

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phillipgreen/mobilecombackup/pkg/calls"
	"github.com/phillipgreen/mobilecombackup/pkg/contacts"
	"github.com/phillipgreen/mobilecombackup/pkg/manifest"
	"github.com/phillipgreen/mobilecombackup/pkg/sms"
	"github.com/spf13/cobra"
)

var (
	reprocessDryRun bool
)

// reprocessContactsCmd represents the reprocess-contacts command
var reprocessContactsCmd = &cobra.Command{
	Use:   "reprocess-contacts",
	Short: "Reprocess contact extraction from existing backup data",
	Long: `Reprocess contact extraction by re-running contact extraction on existing 
backup files in the repository. This is useful when you have updated your 
contacts.yaml file and want to re-extract contact names from the raw backup data.

This command will:
1. Scan existing call and SMS/MMS files in the repository
2. Re-extract contact information using current contact parsing logic
3. Update contacts.yaml with newly discovered contacts in the unprocessed section
4. Preserve existing manually processed contacts

Example:
  mobilecombackup reprocess-contacts
  mobilecombackup reprocess-contacts --dry-run
  mobilecombackup reprocess-contacts --repo-root /path/to/repo`,
	RunE: runReprocessContacts,
}

func init() {
	rootCmd.AddCommand(reprocessContactsCmd)

	// Local flags
	reprocessContactsCmd.Flags().BoolVar(&reprocessDryRun, "dry-run", false,
		"Preview what would be reprocessed without making changes")
}

func runReprocessContacts(_ *cobra.Command, args []string) error {
	// Setup and validation
	resolvedRepoRoot, contactsManager, initialStats, err := setupReprocessContacts()
	if err != nil {
		return err
	}

	// Set up manifest update to run at the end
	defer updateManifestAfterReprocess(resolvedRepoRoot)

	// Process files and collect new contacts
	newUnprocessedEntries, err := processFilesForContacts(resolvedRepoRoot, initialStats)
	if err != nil {
		return err
	}

	// Handle results based on what was found
	return handleReprocessResults(resolvedRepoRoot, contactsManager, newUnprocessedEntries, initialStats)
}

// setupReprocessContacts handles initial setup and validation
func setupReprocessContacts() (string, *contacts.Manager, reprocessInitialStats, error) {
	// Determine repository root
	resolvedRepoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", nil, reprocessInitialStats{}, fmt.Errorf("failed to resolve repository root: %w", err)
	}

	// Check if this is a valid repository
	markerFile := filepath.Join(resolvedRepoRoot, ".mobilecombackup.yaml")
	if _, err := os.Stat(markerFile); os.IsNotExist(err) {
		return "", nil, reprocessInitialStats{}, fmt.Errorf("not a mobilecombackup repository (missing %s)", markerFile)
	}

	// Print initial status
	printReprocessStartMessage(resolvedRepoRoot)

	// Initialize contacts manager
	contactsManager := contacts.NewContactsManager(resolvedRepoRoot)

	// Load existing contacts
	if err := contactsManager.LoadContacts(); err != nil {
		return "", nil, reprocessInitialStats{}, fmt.Errorf("failed to load existing contacts: %w", err)
	}

	// Collect initial statistics
	initialStats := reprocessInitialStats{
		ContactsCount:    contactsManager.GetContactsCount(),
		UnprocessedCount: len(contactsManager.GetUnprocessedEntries()),
	}

	if verbose && !quiet {
		PrintVerbose("Initial state: %d processed contacts, %d unprocessed entries",
			initialStats.ContactsCount, initialStats.UnprocessedCount)
	}

	return resolvedRepoRoot, contactsManager, initialStats, nil
}

// processFilesForContacts processes all files and collects new contacts
func processFilesForContacts(resolvedRepoRoot string, initialStats reprocessInitialStats) ([]contacts.UnprocessedEntry, error) {
	// Create a temporary contacts manager for reprocessing
	tempContactsManager := contacts.NewContactsManager("")

	// Process calls files
	callsStats, err := reprocessCallsFiles(resolvedRepoRoot, tempContactsManager)
	if err != nil {
		return nil, fmt.Errorf("failed to reprocess calls files: %w", err)
	}

	// Process SMS files
	smsStats, err := reprocessSMSFiles(resolvedRepoRoot, tempContactsManager)
	if err != nil {
		return nil, fmt.Errorf("failed to reprocess SMS files: %w", err)
	}

	// Get newly discovered contacts
	newUnprocessedEntries := tempContactsManager.GetUnprocessedEntries()

	// Report processing results
	printReprocessResults(callsStats, smsStats, newUnprocessedEntries)

	return newUnprocessedEntries, nil
}

// handleReprocessResults handles the results and performs final operations
func handleReprocessResults(resolvedRepoRoot string, contactsManager *contacts.Manager,
	newUnprocessedEntries []contacts.UnprocessedEntry, initialStats reprocessInitialStats) error {

	if len(newUnprocessedEntries) == 0 {
		if !quiet {
			PrintInfo("No new contacts found to add")
		}
		return nil
	}

	// Display new contacts that would be added
	printNewContactsDetails(newUnprocessedEntries)

	if reprocessDryRun {
		if !quiet {
			PrintInfo("Dry run complete - no changes made")
		}
		return nil
	}

	// Merge and save new contacts
	return mergeAndSaveContacts(resolvedRepoRoot, contactsManager, newUnprocessedEntries, initialStats)
}

// mergeAndSaveContacts merges new contacts and saves them
func mergeAndSaveContacts(resolvedRepoRoot string, contactsManager *contacts.Manager,
	newUnprocessedEntries []contacts.UnprocessedEntry, initialStats reprocessInitialStats) error {

	// Merge new contacts with existing ones
	for _, entry := range newUnprocessedEntries {
		for _, name := range entry.ContactNames {
			contactsManager.AddUnprocessedContact(entry.PhoneNumber, name)
		}
	}

	// Save updated contacts
	contactsPath := filepath.Join(resolvedRepoRoot, "contacts.yaml")
	if err := contactsManager.SaveContacts(contactsPath); err != nil {
		return fmt.Errorf("failed to save updated contacts: %w", err)
	}

	// Report final results
	finalUnprocessedCount := len(contactsManager.GetUnprocessedEntries())
	newContactsAdded := finalUnprocessedCount - initialStats.UnprocessedCount

	if !quiet {
		PrintInfo("Successfully updated contacts.yaml")
		PrintInfo("Added %d new unprocessed contact entries", newContactsAdded)
	}

	return nil
}

// printReprocessStartMessage prints the initial reprocess message
func printReprocessStartMessage(resolvedRepoRoot string) {
	if !quiet {
		PrintInfo("Reprocessing contacts in repository: %s", resolvedRepoRoot)
		if reprocessDryRun {
			PrintInfo("Running in dry-run mode - no changes will be made")
		}
	}
}

// printReprocessResults prints the file processing results
func printReprocessResults(callsStats, smsStats *ReprocessingStats, newUnprocessedEntries []contacts.UnprocessedEntry) {
	if !quiet {
		PrintInfo("Reprocessing complete:")
		PrintInfo("  Calls files processed: %d", callsStats.FilesProcessed)
		PrintInfo("  Calls records processed: %d", callsStats.RecordsProcessed)
		PrintInfo("  SMS files processed: %d", smsStats.FilesProcessed)
		PrintInfo("  SMS records processed: %d", smsStats.RecordsProcessed)
		PrintInfo("  New unprocessed contacts found: %d", len(newUnprocessedEntries))
	}
}

// printNewContactsDetails prints details of newly found contacts
func printNewContactsDetails(newUnprocessedEntries []contacts.UnprocessedEntry) {
	if verbose && !quiet {
		PrintVerbose("New contacts found:")
		for _, entry := range newUnprocessedEntries {
			for _, name := range entry.ContactNames {
				PrintVerbose("  %s -> %s", entry.PhoneNumber, name)
			}
		}
	}
}

// updateManifestAfterReprocess updates the manifest files after reprocessing
func updateManifestAfterReprocess(resolvedRepoRoot string) {
	if !reprocessDryRun {
		// Update manifest to reflect the current state of contacts.yaml file
		if verbose && !quiet {
			PrintVerbose("Updating manifest files after contacts reprocessing")
		}

		manifestGenerator := manifest.NewManifestGenerator(resolvedRepoRoot)
		fileManifest, err := manifestGenerator.GenerateFileManifest()
		if err != nil && !quiet {
			PrintInfo("Warning: failed to generate updated file manifest: %v", err)
			return
		}

		if err := manifestGenerator.WriteManifestFiles(fileManifest); err != nil && !quiet {
			PrintInfo("Warning: failed to write updated manifest files: %v", err)
			return
		}

		if !quiet {
			PrintInfo("Updated manifest files (files.yaml, files.yaml.sha256)")
		}
	}
}

// reprocessInitialStats holds initial statistics before reprocessing
type reprocessInitialStats struct {
	ContactsCount    int
	UnprocessedCount int
}

// ReprocessingStats holds statistics about the reprocessing operation
type ReprocessingStats struct {
	FilesProcessed   int
	RecordsProcessed int
}

// extractorFunc defines the signature for file processing functions
type extractorFunc func(string, *contacts.Manager) (int, error)

// reprocessFiles is a generic function to process XML files in a directory
func reprocessFiles(
	repoRoot, dirName string,
	contactsManager *contacts.Manager,
	extractor extractorFunc,
) (*ReprocessingStats, error) {
	stats := &ReprocessingStats{}

	dataDir := filepath.Join(repoRoot, dirName)
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		return stats, nil // No directory
	}

	// Find all XML files
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return stats, fmt.Errorf("failed to read %s directory: %w", dirName, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if filepath.Ext(filename) != ".xml" {
			continue
		}

		filePath := filepath.Join(dataDir, filename)

		// Process the file for contact extraction only
		recordsCount, err := extractor(filePath, contactsManager)
		if err != nil {
			if verbose && !quiet {
				PrintVerbose("Warning: failed to process %s file %s: %v", dirName, filename, err)
			}
			continue
		}

		stats.FilesProcessed++
		stats.RecordsProcessed += recordsCount
		if verbose && !quiet {
			PrintVerbose("Processed %s file: %s (%d records)", dirName, filename, recordsCount)
		}
	}

	return stats, nil
}

// reprocessCallsFiles processes all calls files in the repository
func reprocessCallsFiles(repoRoot string, contactsManager *contacts.Manager) (*ReprocessingStats, error) {
	return reprocessFiles(repoRoot, "calls", contactsManager, extractContactsFromCallsFile)
}

// reprocessSMSFiles processes all SMS files in the repository
func reprocessSMSFiles(repoRoot string, contactsManager *contacts.Manager) (*ReprocessingStats, error) {
	return reprocessFiles(repoRoot, "sms", contactsManager, extractContactsFromSMSFile)
}

// extractContactsFromCallsFile extracts contacts from a single calls file
func extractContactsFromCallsFile(filePath string, contactsManager *contacts.Manager) (int, error) {
	file, err := os.Open(filePath) // #nosec G304
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Parse the XML file using the same structure as the importer
	decoder := xml.NewDecoder(file)

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
		return 0, fmt.Errorf("failed to parse XML: %w", err)
	}

	// Extract contacts from each call
	for _, xmlCall := range root.Calls {
		call := calls.Call{
			Number:       xmlCall.Number,
			Duration:     xmlCall.Duration,
			Date:         xmlCall.Date,
			Type:         calls.CallType(xmlCall.Type),
			ReadableDate: xmlCall.ReadableDate,
			ContactName:  xmlCall.ContactName,
		}
		extractContactFromCall(&call, contactsManager)
	}

	return len(root.Calls), nil
}

// extractContactsFromSMSFile extracts contacts from a single SMS file
func extractContactsFromSMSFile(filePath string, contactsManager *contacts.Manager) (int, error) {
	// Create an SMS reader to parse the XML file
	reader := sms.NewXMLSMSReader("")

	// Read all messages from the file
	file, err := os.Open(filePath) // #nosec G304
	if err != nil {
		return 0, fmt.Errorf("failed to open SMS file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Count messages processed
	messageCount := 0

	// Stream messages and extract contacts from each
	err = reader.StreamMessagesFromReader(file, func(msg sms.Message) error {
		messageCount++
		switch message := msg.(type) {
		case sms.SMS:
			extractContactFromSMS(message, contactsManager)
		case sms.MMS:
			extractContactFromMMS(message, contactsManager)
		}
		return nil
	})

	if err != nil {
		return messageCount, fmt.Errorf("failed to stream messages from file: %w", err)
	}

	return messageCount, nil
}

// extractContactFromCall extracts contact information from a call record
func extractContactFromCall(call *calls.Call, contactsManager *contacts.Manager) {
	// Extract contact names if both number and contact name are present
	if call.Number != "" && call.ContactName != "" {
		// Split contact names by comma and process each separately
		contactNames := strings.Split(call.ContactName, ",")
		for _, name := range contactNames {
			name = strings.TrimSpace(name)
			if name != "" && !isUnknownContact(name) {
				contactsManager.AddUnprocessedContact(call.Number, name)
			}
		}
	}
}

// extractContactFromSMS extracts contact information from an SMS message
func extractContactFromSMS(smsMsg sms.SMS, contactsManager *contacts.Manager) {
	// Extract primary address contact, splitting multiple contact names
	if smsMsg.Address != "" && smsMsg.ContactName != "" {
		// Split contact names by comma and process each separately
		contactNames := strings.Split(smsMsg.ContactName, ",")
		for _, name := range contactNames {
			name = strings.TrimSpace(name)
			if name != "" && !isUnknownContact(name) {
				contactsManager.AddUnprocessedContact(smsMsg.Address, name)
			}
		}
	}
}

// extractContactFromMMS extracts contact information from an MMS message
func extractContactFromMMS(mmsMsg sms.MMS, contactsManager *contacts.Manager) {
	// Extract primary address contact, splitting multiple contact names
	if mmsMsg.Address != "" && mmsMsg.ContactName != "" {
		// Split contact names by comma and process each separately
		contactNames := strings.Split(mmsMsg.ContactName, ",")
		for _, name := range contactNames {
			name = strings.TrimSpace(name)
			if name != "" && !isUnknownContact(name) {
				contactsManager.AddUnprocessedContact(mmsMsg.Address, name)
			}
		}
	}
}

// isUnknownContact checks if a contact name represents an unknown contact
// and should be ignored during contact extraction
func isUnknownContact(contactName string) bool {
	return contactName == "" || contactName == "(Unknown)" || contactName == "null"
}
