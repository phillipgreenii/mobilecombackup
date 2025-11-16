package cmd

import (
	"testing"
)

// TestCreateImportOptions tests the createImportOptions function
func TestCreateImportOptions(t *testing.T) {
	tests := []struct {
		name              string
		repoRoot          string
		paths             []string
		dryRun            bool
		verbose           bool
		quiet             bool
		json              bool
		filter            string
		maxXMLSize        string
		maxMessageSize    string
		expectError       bool
		expectedQuiet     bool
		description       string
	}{
		{
			name:           "basic options",
			repoRoot:       "/test/repo",
			paths:          []string{"file1.xml", "file2.xml"},
			dryRun:         false,
			verbose:        false,
			quiet:          false,
			json:           false,
			filter:         "",
			maxXMLSize:     "500MB",
			maxMessageSize: "10MB",
			expectError:    false,
			expectedQuiet:  false,
			description:    "Should create options with defaults",
		},
		{
			name:           "json forces quiet",
			repoRoot:       "/test/repo",
			paths:          []string{"file.xml"},
			dryRun:         false,
			verbose:        false,
			quiet:          false,
			json:           true,
			filter:         "",
			maxXMLSize:     "500MB",
			maxMessageSize: "10MB",
			expectError:    false,
			expectedQuiet:  true,
			description:    "JSON mode should force quiet",
		},
		{
			name:           "invalid max xml size",
			repoRoot:       "/test/repo",
			paths:          []string{"file.xml"},
			dryRun:         false,
			verbose:        false,
			quiet:          false,
			json:           false,
			filter:         "",
			maxXMLSize:     "invalid",
			maxMessageSize: "10MB",
			expectError:    true,
			description:    "Should error on invalid max XML size",
		},
		{
			name:           "invalid max message size",
			repoRoot:       "/test/repo",
			paths:          []string{"file.xml"},
			dryRun:         false,
			verbose:        false,
			quiet:          false,
			json:           false,
			filter:         "",
			maxXMLSize:     "500MB",
			maxMessageSize: "invalid",
			expectError:    true,
			description:    "Should error on invalid max message size",
		},
		{
			name:           "with filter",
			repoRoot:       "/test/repo",
			paths:          []string{"file.xml"},
			dryRun:         false,
			verbose:        false,
			quiet:          false,
			json:           false,
			filter:         "calls",
			maxXMLSize:     "500MB",
			maxMessageSize: "10MB",
			expectError:    false,
			description:    "Should accept filter option",
		},
		{
			name:           "dry run mode",
			repoRoot:       "/test/repo",
			paths:          []string{"file.xml"},
			dryRun:         true,
			verbose:        false,
			quiet:          false,
			json:           false,
			filter:         "",
			maxXMLSize:     "500MB",
			maxMessageSize: "10MB",
			expectError:    false,
			description:    "Should accept dry run mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global variables
			importDryRun = tt.dryRun
			verbose = tt.verbose
			quiet = tt.quiet
			importJSON = tt.json
			importFilter = tt.filter
			maxXMLSizeStr = tt.maxXMLSize
			maxMessageSizeStr = tt.maxMessageSize

			options, err := createImportOptions(tt.repoRoot, tt.paths)

			if tt.expectError {
				if err == nil {
					t.Errorf("%s: expected error, got nil", tt.description)
				}
				return
			}

			if err != nil {
				t.Errorf("%s: expected no error, got: %v", tt.description, err)
				return
			}

			if options == nil {
				t.Errorf("%s: expected non-nil options", tt.description)
				return
			}

			// Verify options
			if options.RepoRoot != tt.repoRoot {
				t.Errorf("%s: expected RepoRoot %s, got %s", tt.description, tt.repoRoot, options.RepoRoot)
			}

			if len(options.Paths) != len(tt.paths) {
				t.Errorf("%s: expected %d paths, got %d", tt.description, len(tt.paths), len(options.Paths))
			}

			if options.DryRun != tt.dryRun {
				t.Errorf("%s: expected DryRun %v, got %v", tt.description, tt.dryRun, options.DryRun)
			}

			if options.Quiet != tt.expectedQuiet {
				t.Errorf("%s: expected Quiet %v, got %v", tt.description, tt.expectedQuiet, options.Quiet)
			}

			if options.Filter != tt.filter {
				t.Errorf("%s: expected Filter %s, got %s", tt.description, tt.filter, options.Filter)
			}

			if options.Fs == nil {
				t.Errorf("%s: expected non-nil Fs", tt.description)
			}
		})
	}
}

// TestCreateImporter tests the createImporter function
func TestCreateImporter(t *testing.T) {
	// Note: createImporter calls NewImporter which validates the repository.
	// Repository validation uses os.Stat() directly, not the afero filesystem,
	// making it difficult to mock for testing. Additionally, createImporter
	// calls os.Exit(2) on failure, which would terminate the test process.
	// This function would need refactoring to be testable.
	t.Skip("createImporter uses os.Stat and os.Exit, making it difficult to test")
}

// TestValidateAndPreparePaths tests the validateAndPreparePaths function
func TestValidateAndPreparePaths(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		repoRoot    string
		expectError bool
		expectPaths int
		description string
	}{
		{
			name:        "explicit paths",
			args:        []string{"file1.xml", "file2.xml"},
			repoRoot:    "/test/repo",
			expectError: false,
			expectPaths: 2,
			description: "Should accept explicit paths",
		},
		{
			name:        "no paths uses repo root",
			args:        []string{},
			repoRoot:    "/test/repo",
			expectError: false,
			expectPaths: 1,
			description: "Should use repo root when no paths given",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock command
			cmd := importCmd

			paths, err := validateAndPreparePaths(cmd, tt.args, tt.repoRoot)

			if tt.expectError {
				if err == nil {
					t.Errorf("%s: expected error, got nil", tt.description)
				}
				return
			}

			if err != nil {
				t.Errorf("%s: expected no error, got: %v", tt.description, err)
				return
			}

			if len(paths) != tt.expectPaths {
				t.Errorf("%s: expected %d paths, got %d", tt.description, tt.expectPaths, len(paths))
			}
		})
	}
}
