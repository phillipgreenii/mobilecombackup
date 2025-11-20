package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/phillipgreenii/mobilecombackup/pkg/repository"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
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
	fs := afero.NewOsFs()
	validator := repository.NewValidator(fs)
	if err := validator.ValidateTargetDirectory(absPath); err != nil {
		return err
	}

	// Initialize repository
	creator := repository.NewCreator(fs, version)
	result, err := creator.Initialize(absPath, dryRun)
	if err != nil {
		return err
	}

	// Display results
	if !quiet {
		displayInitResult(result)
	}

	return nil
}

func displayInitResult(result *repository.InitResult) {
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
