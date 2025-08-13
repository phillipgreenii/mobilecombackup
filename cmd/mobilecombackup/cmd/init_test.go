package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestValidateTargetDirectory(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) string
		wantError string
	}{
		{
			name: "non-existent directory is valid",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "new-repo")
			},
			wantError: "",
		},
		{
			name: "empty directory is valid",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantError: "",
		},
		{
			name: "directory with .mobilecombackup.yaml",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				f, _ := os.Create(filepath.Join(dir, ".mobilecombackup.yaml"))
				_ = f.Close()
				return dir
			},
			wantError: "already contains a mobilecombackup repository",
		},
		{
			name: "directory with calls subdirectory",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				_ = os.Mkdir(filepath.Join(dir, "calls"), 0755)
				return dir
			},
			wantError: "appears to be a repository",
		},
		{
			name: "directory with sms subdirectory",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				_ = os.Mkdir(filepath.Join(dir, "sms"), 0755)
				return dir
			},
			wantError: "appears to be a repository",
		},
		{
			name: "directory with attachments subdirectory",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				_ = os.Mkdir(filepath.Join(dir, "attachments"), 0755)
				return dir
			},
			wantError: "appears to be a repository",
		},
		{
			name: "non-empty directory",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				f, _ := os.Create(filepath.Join(dir, "some-file.txt"))
				_ = f.Close()
				return dir
			},
			wantError: "directory is not empty",
		},
		{
			name: "path is a file",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "file.txt")
				f, _ := os.Create(path)
				_ = f.Close()
				return path
			},
			wantError: "not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			err := validateTargetDirectory(path)

			if tt.wantError == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantError)
				} else if !strings.Contains(err.Error(), tt.wantError) {
					t.Errorf("expected error containing %q, got %q", tt.wantError, err.Error())
				}
			}
		})
	}
}

func TestInitializeRepository(t *testing.T) {
	tests := []struct {
		name         string
		dryRun       bool
		quiet        bool
		checkCreated bool
	}{
		{
			name:         "normal initialization",
			dryRun:       false,
			quiet:        false,
			checkCreated: true,
		},
		{
			name:         "dry run",
			dryRun:       true,
			quiet:        false,
			checkCreated: false,
		},
		{
			name:         "quiet mode",
			dryRun:       false,
			quiet:        true,
			checkCreated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoRoot := filepath.Join(t.TempDir(), "test-repo")

			result, err := initializeRepository(repoRoot, tt.dryRun, tt.quiet)
			if err != nil {
				t.Fatalf("initializeRepository failed: %v", err)
			}

			// Check result
			if result.RepoRoot != repoRoot {
				t.Errorf("result.RepoRoot = %q, want %q", result.RepoRoot, repoRoot)
			}

			if result.DryRun != tt.dryRun {
				t.Errorf("result.DryRun = %v, want %v", result.DryRun, tt.dryRun)
			}

			// Expected created items
			expectedCreated := []string{
				repoRoot,
				filepath.Join(repoRoot, "calls"),
				filepath.Join(repoRoot, "sms"),
				filepath.Join(repoRoot, "attachments"),
				filepath.Join(repoRoot, ".mobilecombackup.yaml"),
				filepath.Join(repoRoot, "contacts.yaml"),
				filepath.Join(repoRoot, "summary.yaml"),
				filepath.Join(repoRoot, "files.yaml"),
				filepath.Join(repoRoot, "files.yaml.sha256"),
			}

			if len(result.Created) != len(expectedCreated) {
				t.Errorf("created %d items, want %d", len(result.Created), len(expectedCreated))
			}

			// Check actual file creation
			if tt.checkCreated {
				// Check directories
				for _, dir := range []string{"calls", "sms", "attachments"} {
					dirPath := filepath.Join(repoRoot, dir)
					info, err := os.Stat(dirPath)
					if err != nil {
						t.Errorf("directory %s not created: %v", dir, err)
					} else if !info.IsDir() {
						t.Errorf("%s is not a directory", dir)
					}
				}

				// Check marker file
				markerPath := filepath.Join(repoRoot, ".mobilecombackup.yaml")
				data, err := os.ReadFile(markerPath)
				if err != nil {
					t.Errorf("marker file not created: %v", err)
				} else {
					var marker MarkerFileContent
					if err := yaml.Unmarshal(data, &marker); err != nil {
						t.Errorf("invalid marker file: %v", err)
					} else {
						if marker.RepositoryStructureVersion != "1" {
							t.Errorf("marker version = %q, want \"1\"", marker.RepositoryStructureVersion)
						}
						if marker.CreatedAt == "" {
							t.Error("marker CreatedAt is empty")
						}
						if marker.CreatedBy == "" {
							t.Error("marker CreatedBy is empty")
						}
					}
				}

				// Check contacts.yaml
				contactsPath := filepath.Join(repoRoot, "contacts.yaml")
				data, err = os.ReadFile(contactsPath)
				if err != nil {
					t.Errorf("contacts file not created: %v", err)
				} else if !strings.Contains(string(data), "contacts: []") {
					t.Errorf("contacts file has unexpected content: %s", string(data))
				}

				// Check summary.yaml
				summaryPath := filepath.Join(repoRoot, "summary.yaml")
				data, err = os.ReadFile(summaryPath)
				if err != nil {
					t.Errorf("summary file not created: %v", err)
				} else {
					var summary SummaryContent
					if err := yaml.Unmarshal(data, &summary); err != nil {
						t.Errorf("invalid summary file: %v", err)
					} else {
						if summary.Counts.Calls != 0 {
							t.Errorf("summary calls count = %d, want 0", summary.Counts.Calls)
						}
						if summary.Counts.SMS != 0 {
							t.Errorf("summary SMS count = %d, want 0", summary.Counts.SMS)
						}
					}
				}
			} else {
				// In dry run, nothing should be created
				if _, err := os.Stat(repoRoot); !os.IsNotExist(err) {
					t.Error("dry run created repository root")
				}
			}
		})
	}
}

func TestInitializeRepositoryPermissions(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	// Create a directory without write permission
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "no-write")
	_ = os.Mkdir(repoRoot, 0555) // read+execute only

	_, err := initializeRepository(repoRoot, false, true)
	if err == nil {
		t.Error("expected error for directory without write permission")
	}
}

func TestPrintTree(t *testing.T) {
	// This is a display function, so we just ensure it doesn't panic
	tree := map[string]*node{
		"": {
			name:     "repo",
			children: []string{"calls", "sms", ".mobilecombackup.yaml"},
		},
		"calls": {
			name:     "calls",
			children: []string{},
		},
		"sms": {
			name:     "sms",
			children: []string{},
		},
		".mobilecombackup.yaml": {
			name:     ".mobilecombackup.yaml",
			children: []string{},
		},
	}

	// Should not panic
	printTree("", tree[""], tree, true, "", true)
}
