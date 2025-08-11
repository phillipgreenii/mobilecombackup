package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/calls"
	"github.com/phillipgreen/mobilecombackup/pkg/sms"
	"gopkg.in/yaml.v3"
)

// generateSummaryFile creates a summary.yaml file with repository statistics
func generateSummaryFile(repoRoot string) error {
	stats, err := calculateRepositoryStats(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to calculate repository stats: %w", err)
	}

	summary := &SummaryFile{
		LastUpdated: time.Now().UTC().Format(time.RFC3339),
		Statistics:  *stats,
	}

	// Write summary to file
	summaryPath := filepath.Join(repoRoot, "summary.yaml")
	file, err := os.Create(summaryPath)
	if err != nil {
		return fmt.Errorf("failed to create summary file: %w", err)
	}
	defer func() { _ = file.Close() }()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(summary); err != nil {
		return fmt.Errorf("failed to write summary: %w", err)
	}

	return nil
}

// calculateRepositoryStats counts all entities in the repository
func calculateRepositoryStats(repoRoot string) (*RepoStats, error) {
	stats := &RepoStats{
		YearsCovered: []int{},
	}

	yearsMap := make(map[int]bool)

	// Count calls
	callsReader := calls.NewXMLCallsReader(repoRoot)
	years, err := callsReader.GetAvailableYears()
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to get call years: %w", err)
	}

	for _, year := range years {
		yearsMap[year] = true
		count, err := callsReader.GetCallsCount(year)
		if err != nil {
			return nil, fmt.Errorf("failed to count calls for year %d: %w", year, err)
		}
		stats.TotalCalls += count
	}

	// Count SMS/MMS
	smsReader := sms.NewXMLSMSReader(repoRoot)
	years, err = smsReader.GetAvailableYears()
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to get SMS years: %w", err)
	}

	for _, year := range years {
		yearsMap[year] = true
		count, err := smsReader.GetMessageCount(year)
		if err != nil {
			return nil, fmt.Errorf("failed to count SMS for year %d: %w", year, err)
		}
		stats.TotalSMS += count
	}

	// Count attachments
	attachmentsDir := filepath.Join(repoRoot, "attachments")
	if info, err := os.Stat(attachmentsDir); err == nil && info.IsDir() {
		count, err := countAttachmentFiles(attachmentsDir)
		if err != nil {
			return nil, fmt.Errorf("failed to count attachments: %w", err)
		}
		stats.TotalAttachments = count
	}

	// Convert years map to sorted slice
	for year := range yearsMap {
		stats.YearsCovered = append(stats.YearsCovered, year)
	}
	sort.Ints(stats.YearsCovered)

	return stats, nil
}

// countAttachmentFiles counts all files in the attachments directory
func countAttachmentFiles(attachmentsDir string) (int, error) {
	count := 0

	err := filepath.Walk(attachmentsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip non-attachment files (like .gitkeep)
		if filepath.Base(path) == ".gitkeep" {
			return nil
		}

		count++
		return nil
	})

	return count, err
}
