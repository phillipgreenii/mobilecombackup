package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/phillipgreenii/mobilecombackup/pkg/manifest"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	// Repository directory names
	callsDir       = "calls"
	smsDir         = "sms"
	attachmentsDir = "attachments"
)

var (
	dryRun bool
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new mobilecombackup repository",
	Long: `Initialize a new mobilecombackup repository with the required directory structure.

Creates the following directories:
- calls/      For call log XML files
- sms/        For SMS/MMS XML files  
- attachments/ For extracted attachment files

Also creates:
- .mobilecombackup.yaml  Repository marker file with version info
- contacts.yaml          Empty contacts file
- summary.yaml           Initial summary with zero counts
- files.yaml             File manifest tracking all repository files
- files.yaml.sha256      Checksum file for manifest verification`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Local flags
	initCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview actions without creating directories")
}

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

// node represents a tree node for display
type node struct {
	name     string
	children []string
}

func runInit(_ *cobra.Command, _ []string) error {
	// Get repository root (from global flag or current directory)
	targetDir := repoRoot
	absPath, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to resolve repository path: %w", err)
	}

	// Validate target directory
	if err := validateTargetDirectory(absPath); err != nil {
		return err
	}

	// Initialize repository
	result, err := initializeRepository(absPath, dryRun, quiet)
	if err != nil {
		return err
	}

	// Display results
	if !quiet {
		displayInitResult(result)
	}

	return nil
}

func validateTargetDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return handleStatError(err, path)
	}

	if !info.IsDir() {
		return fmt.Errorf("path exists but is not a directory: %s", path)
	}

	return validateDirectoryContents(path)
}

// handleStatError handles errors from os.Stat
func handleStatError(err error, _ string) error {
	if os.IsNotExist(err) {
		// Directory doesn't exist - this is OK, we'll create it
		return nil
	}
	return fmt.Errorf("failed to check directory: %w", err)
}

// validateDirectoryContents validates that the directory is suitable for initialization
func validateDirectoryContents(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	if err := checkForExistingRepository(entries); err != nil {
		return err
	}

	// Warn if directory is not empty
	if len(entries) > 0 {
		return fmt.Errorf("directory is not empty")
	}

	return nil
}

// checkForExistingRepository checks if directory already contains a repository
func checkForExistingRepository(entries []os.DirEntry) error {
	for _, entry := range entries {
		if entry.Name() == ".mobilecombackup.yaml" {
			return fmt.Errorf("directory already contains a mobilecombackup repository")
		}
		if isRepositoryDirectory(entry.Name()) {
			return fmt.Errorf("directory appears to be a repository (found %s/ directory)", entry.Name())
		}
	}
	return nil
}

// isRepositoryDirectory checks if a directory name indicates a repository structure
func isRepositoryDirectory(name string) bool {
	return name == callsDir || name == smsDir || name == attachmentsDir
}

func initializeRepository(repoRoot string, dryRun bool, _ bool) (*InitResult, error) {
	result := &InitResult{
		RepoRoot: repoRoot,
		DryRun:   dryRun,
		Created:  []string{},
	}

	// If not dry run, set up rollback on failure
	var createdPaths []string
	rollback := createRollbackFunction(dryRun, &createdPaths)

	// Create directory structure
	if err := createRepositoryDirectories(repoRoot, dryRun, result, &createdPaths, rollback); err != nil {
		return nil, err
	}

	// Create repository files
	if err := createRepositoryFiles(repoRoot, dryRun, result, &createdPaths, rollback); err != nil {
		return nil, err
	}

	// Create manifest files
	if err := createManifestFiles(repoRoot, dryRun, result, &createdPaths, rollback); err != nil {
		return nil, err
	}

	return result, nil
}

// createRollbackFunction creates a rollback function for cleanup on failure
func createRollbackFunction(dryRun bool, createdPaths *[]string) func() {
	return func() {
		if !dryRun {
			// Remove files and directories in reverse order
			for i := len(*createdPaths) - 1; i >= 0; i-- {
				_ = os.RemoveAll((*createdPaths)[i])
			}
		}
	}
}

// createRepositoryDirectories creates the main repository directory structure
func createRepositoryDirectories(
	repoRoot string, dryRun bool, result *InitResult, createdPaths *[]string, rollback func(),
) error {
	// Directories to create
	directories := []string{
		"calls",
		"sms",
		"attachments",
	}

	// Create root directory if it doesn't exist
	if _, err := os.Stat(repoRoot); os.IsNotExist(err) {
		if !dryRun {
			if err := os.MkdirAll(repoRoot, 0750); err != nil {
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
			if err := os.MkdirAll(dirPath, 0750); err != nil {
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
func createRepositoryFiles(repoRoot string, dryRun bool, result *InitResult, createdPaths *[]string, rollback func()) error {
	// Create marker file
	if err := createMarkerFile(repoRoot, dryRun, result, createdPaths, rollback); err != nil {
		return err
	}

	// Create empty contacts.yaml
	if err := createContactsFile(repoRoot, dryRun, result, createdPaths, rollback); err != nil {
		return err
	}

	// Create summary.yaml with zero counts
	if err := createSummaryFile(repoRoot, dryRun, result, createdPaths, rollback); err != nil {
		return err
	}

	return nil
}

// createMarkerFile creates the .mobilecombackup.yaml marker file
func createMarkerFile(repoRoot string, dryRun bool, result *InitResult, createdPaths *[]string, rollback func()) error {
	markerPath := filepath.Join(repoRoot, ".mobilecombackup.yaml")
	markerContent := MarkerFileContent{
		RepositoryStructureVersion: "1",
		CreatedAt:                  time.Now().UTC().Format(time.RFC3339),
		CreatedBy:                  fmt.Sprintf("mobilecombackup v%s", version),
	}

	if !dryRun {
		data, err := yaml.Marshal(markerContent)
		if err != nil {
			rollback()
			return fmt.Errorf("failed to marshal marker file: %w", err)
		}
		if err := os.WriteFile(markerPath, data, 0600); err != nil {
			rollback()
			return fmt.Errorf("failed to create marker file: %w", err)
		}
		*createdPaths = append(*createdPaths, markerPath)
	}
	result.Created = append(result.Created, markerPath)
	return nil
}

// createContactsFile creates the empty contacts.yaml file
func createContactsFile(repoRoot string, dryRun bool, result *InitResult, createdPaths *[]string, rollback func()) error {
	contactsPath := filepath.Join(repoRoot, "contacts.yaml")
	if !dryRun {
		// Write empty YAML array
		if err := os.WriteFile(contactsPath, []byte("contacts: []\n"), 0600); err != nil {
			rollback()
			return fmt.Errorf("failed to create contacts file: %w", err)
		}
		*createdPaths = append(*createdPaths, contactsPath)
	}
	result.Created = append(result.Created, contactsPath)
	return nil
}

// createSummaryFile creates the summary.yaml file with zero counts
func createSummaryFile(repoRoot string, dryRun bool, result *InitResult, createdPaths *[]string, rollback func()) error {
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
		if err := os.WriteFile(summaryPath, data, 0600); err != nil {
			rollback()
			return fmt.Errorf("failed to create summary file: %w", err)
		}
		*createdPaths = append(*createdPaths, summaryPath)
	}
	result.Created = append(result.Created, summaryPath)
	return nil
}

// createManifestFiles creates the files.yaml and files.yaml.sha256 manifest files
func createManifestFiles(repoRoot string, dryRun bool, result *InitResult, createdPaths *[]string, rollback func()) error {
	// Create files.yaml and files.yaml.sha256 using manifest generator
	if !dryRun {
		manifestGenerator := manifest.NewManifestGenerator(repoRoot, afero.NewOsFs())
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

func displayInitResult(result *InitResult) {
	if result.DryRun {
		fmt.Println("DRY RUN: No files or directories were created")
		fmt.Println()
	}

	fmt.Printf("Initialized mobilecombackup repository in: %s\n", result.RepoRoot)
	fmt.Println()

	// Display tree-style output
	fmt.Println("Created structure:")

	// Convert absolute paths to relative paths and organize
	tree := make(map[string]*node)
	root := &node{name: filepath.Base(result.RepoRoot)}
	tree[""] = root

	for _, created := range result.Created {
		rel, _ := filepath.Rel(result.RepoRoot, created)
		if rel == "." {
			continue
		}

		parts := strings.Split(rel, string(filepath.Separator))
		parent := ""

		for i, part := range parts {
			current := filepath.Join(parent, part)

			if _, exists := tree[current]; !exists {
				tree[current] = &node{name: part}

				// Add to parent's children
				if parentNode, ok := tree[parent]; ok {
					parentNode.children = append(parentNode.children, current)
				}
			}

			if i < len(parts)-1 {
				parent = current
			}
		}
	}

	// Print tree - start with no prefix since we're at root
	printTree("", root, tree, true, "", true)
}

func printTree(indent string, node *node, tree map[string]*node, isLast bool, _ string, isRoot bool) {
	// Print current node
	if isRoot {
		// Root node - just print name
		fmt.Println(node.name)
	} else {
		// Non-root nodes get tree connectors
		connector := "├── "
		if isLast {
			connector = "└── "
		}
		fmt.Printf("%s%s%s\n", indent, connector, node.name)
	}

	// Update indent for children
	childIndent := indent
	if !isRoot {
		if isLast {
			childIndent += "    "
		} else {
			childIndent += "│   "
		}
	}

	// Print children
	for i, childPath := range node.children {
		if child, ok := tree[childPath]; ok {
			isLastChild := i == len(node.children)-1
			printTree(childIndent, child, tree, isLastChild, childPath, false)
		}
	}
}
