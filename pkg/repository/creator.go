package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/phillipgreenii/mobilecombackup/pkg/manifest"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

const (
	// Repository directory names
	CallsDir       = "calls"
	SMSDir         = "sms"
	AttachmentsDir = "attachments"
)

// MarkerFileContent represents the .mobilecombackup.yaml file structure
type MarkerFileContent struct {
	RepositoryStructureVersion string `yaml:"repository_structure_version"`
	CreatedAt                  string `yaml:"created_at"`
	CreatedBy                  string `yaml:"created_by"`
}

// SummaryContent represents the summary.yaml file structure
type SummaryContent struct {
	Counts struct {
		Calls int `yaml:"calls"`
		SMS   int `yaml:"sms"`
	} `yaml:"counts"`
}

// InitResult contains the result of initialization
type InitResult struct {
	RepoRoot string
	Created  []string // Directories and files created
	DryRun   bool
}

// Creator handles repository creation and initialization
type Creator struct {
	fs      afero.Fs
	version string
}

// NewCreator creates a new repository creator
func NewCreator(fs afero.Fs, version string) *Creator {
	return &Creator{
		fs:      fs,
		version: version,
	}
}

// Initialize creates a new repository with the standard directory structure
func (c *Creator) Initialize(repoRoot string, dryRun bool) (*InitResult, error) {
	result := &InitResult{
		RepoRoot: repoRoot,
		DryRun:   dryRun,
		Created:  []string{},
	}

	// If not dry run, set up rollback on failure
	var createdPaths []string
	rollback := c.createRollbackFunction(dryRun, &createdPaths)

	// Create directory structure
	if err := c.createRepositoryDirectories(repoRoot, dryRun, result, &createdPaths, rollback); err != nil {
		return nil, err
	}

	// Create repository files
	if err := c.createRepositoryFiles(repoRoot, dryRun, result, &createdPaths, rollback); err != nil {
		return nil, err
	}

	// Create manifest files
	if err := c.createManifestFiles(repoRoot, dryRun, result, &createdPaths, rollback); err != nil {
		return nil, err
	}

	return result, nil
}

// createRollbackFunction creates a rollback function for cleanup on failure
func (c *Creator) createRollbackFunction(dryRun bool, createdPaths *[]string) func() {
	return func() {
		if !dryRun {
			// Remove files and directories in reverse order
			for i := len(*createdPaths) - 1; i >= 0; i-- {
				_ = c.fs.RemoveAll((*createdPaths)[i])
			}
		}
	}
}

// createRepositoryDirectories creates the main repository directory structure
func (c *Creator) createRepositoryDirectories(
	repoRoot string, dryRun bool, result *InitResult, createdPaths *[]string, rollback func(),
) error {
	// Directories to create
	directories := []string{
		CallsDir,
		SMSDir,
		AttachmentsDir,
	}

	// Create root directory if it doesn't exist
	if _, err := c.fs.Stat(repoRoot); os.IsNotExist(err) {
		if !dryRun {
			if err := c.fs.MkdirAll(repoRoot, 0750); err != nil {
				return fmt.Errorf("failed to create repository root: %w", err)
			}
			*createdPaths = append(*createdPaths, repoRoot)
		}
		result.Created = append(result.Created, repoRoot)
	}

	// Create subdirectories
	for _, dir := range directories {
		dirPath := filepath.Join(repoRoot, dir)
		if !dryRun {
			if err := c.fs.MkdirAll(dirPath, 0750); err != nil {
				rollback()
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
			*createdPaths = append(*createdPaths, dirPath)
		}
		result.Created = append(result.Created, dirPath)
	}

	return nil
}

// createRepositoryFiles creates the core repository configuration files
func (c *Creator) createRepositoryFiles(repoRoot string, dryRun bool, result *InitResult, createdPaths *[]string, rollback func()) error {
	// Create marker file
	if err := c.createMarkerFile(repoRoot, dryRun, result, createdPaths, rollback); err != nil {
		return err
	}

	// Create empty contacts.yaml
	if err := c.createContactsFile(repoRoot, dryRun, result, createdPaths, rollback); err != nil {
		return err
	}

	// Create summary.yaml with zero counts
	if err := c.createSummaryFile(repoRoot, dryRun, result, createdPaths, rollback); err != nil {
		return err
	}

	return nil
}

// createMarkerFile creates the .mobilecombackup.yaml marker file
func (c *Creator) createMarkerFile(repoRoot string, dryRun bool, result *InitResult, createdPaths *[]string, rollback func()) error {
	markerPath := filepath.Join(repoRoot, ".mobilecombackup.yaml")
	markerContent := MarkerFileContent{
		RepositoryStructureVersion: "1",
		CreatedAt:                  time.Now().UTC().Format(time.RFC3339),
		CreatedBy:                  fmt.Sprintf("mobilecombackup v%s", c.version),
	}

	if !dryRun {
		data, err := yaml.Marshal(markerContent)
		if err != nil {
			rollback()
			return fmt.Errorf("failed to marshal marker file: %w", err)
		}
		if err := afero.WriteFile(c.fs, markerPath, data, 0600); err != nil {
			rollback()
			return fmt.Errorf("failed to create marker file: %w", err)
		}
		*createdPaths = append(*createdPaths, markerPath)
	}
	result.Created = append(result.Created, markerPath)
	return nil
}

// createContactsFile creates the empty contacts.yaml file
func (c *Creator) createContactsFile(repoRoot string, dryRun bool, result *InitResult, createdPaths *[]string, rollback func()) error {
	contactsPath := filepath.Join(repoRoot, "contacts.yaml")
	if !dryRun {
		// Write empty YAML array
		if err := afero.WriteFile(c.fs, contactsPath, []byte("contacts: []\n"), 0600); err != nil {
			rollback()
			return fmt.Errorf("failed to create contacts file: %w", err)
		}
		*createdPaths = append(*createdPaths, contactsPath)
	}
	result.Created = append(result.Created, contactsPath)
	return nil
}

// createSummaryFile creates the summary.yaml file with zero counts
func (c *Creator) createSummaryFile(repoRoot string, dryRun bool, result *InitResult, createdPaths *[]string, rollback func()) error {
	summaryPath := filepath.Join(repoRoot, "summary.yaml")
	summaryContent := SummaryContent{}
	summaryContent.Counts.Calls = 0
	summaryContent.Counts.SMS = 0

	if !dryRun {
		data, err := yaml.Marshal(summaryContent)
		if err != nil {
			rollback()
			return fmt.Errorf("failed to marshal summary file: %w", err)
		}
		if err := afero.WriteFile(c.fs, summaryPath, data, 0600); err != nil {
			rollback()
			return fmt.Errorf("failed to create summary file: %w", err)
		}
		*createdPaths = append(*createdPaths, summaryPath)
	}
	result.Created = append(result.Created, summaryPath)
	return nil
}

// createManifestFiles creates the files.yaml and files.yaml.sha256 manifest files
func (c *Creator) createManifestFiles(repoRoot string, dryRun bool, result *InitResult, createdPaths *[]string, rollback func()) error {
	// Create files.yaml and files.yaml.sha256 using manifest generator
	if !dryRun {
		manifestGenerator := manifest.NewManifestGenerator(repoRoot, c.fs)
		fileManifest, err := manifestGenerator.GenerateFileManifest()
		if err != nil {
			rollback()
			return fmt.Errorf("failed to generate file manifest: %w", err)
		}

		if err := manifestGenerator.WriteManifestFiles(fileManifest); err != nil {
			rollback()
			return fmt.Errorf("failed to write manifest files: %w", err)
		}
		*createdPaths = append(*createdPaths, filepath.Join(repoRoot, "files.yaml"))
		*createdPaths = append(*createdPaths, filepath.Join(repoRoot, "files.yaml.sha256"))
	}
	result.Created = append(result.Created, filepath.Join(repoRoot, "files.yaml"))
	result.Created = append(result.Created, filepath.Join(repoRoot, "files.yaml.sha256"))

	return nil
}
