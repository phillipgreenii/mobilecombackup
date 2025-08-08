package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func initRootCmd() {
	rootCmd.SetVersionTemplate("mobilecombackup version {{.Version}}\n")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "Suppress non-error output")
	rootCmd.PersistentFlags().StringVar(&repoRoot, "repo-root", ".", "Path to repository root")
}

func TestRootCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		wantOut        string
		wantErr        string
		wantExitCode   bool // true if should exit with error
	}{
		{
			name:    "no arguments shows help",
			args:    []string{},
			wantOut: "mobilecombackup processes call logs and SMS/MMS messages",
		},
		{
			name:    "help flag shows help",
			args:    []string{"--help"},
			wantOut: "mobilecombackup processes call logs and SMS/MMS messages",
		},
		{
			name:    "short help flag shows help",
			args:    []string{"-h"},
			wantOut: "mobilecombackup processes call logs and SMS/MMS messages",
		},
		{
			name:    "version flag shows version",
			args:    []string{"--version"},
			wantOut: "mobilecombackup version test-version",
		},
		{
			name:    "short version flag shows version",
			args:    []string{"-v"},
			wantOut: "mobilecombackup version test-version",
		},
		{
			name:         "unknown command shows error",
			args:         []string{"unknown"},
			wantErr:      `unknown command "unknown" for "mobilecombackup"`,
			wantExitCode: true,
		},
		{
			name:         "unknown flag shows error",
			args:         []string{"--unknown"},
			wantErr:      "unknown flag: --unknown",
			wantExitCode: true,
		},
		{
			name:    "global flags parsed",
			args:    []string{"--repo-root", "/custom/path", "--quiet", "--help"},
			wantOut: "mobilecombackup processes call logs and SMS/MMS messages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset command for each test
			rootCmd.ResetCommands()
			rootCmd.ResetFlags()
			initRootCmd() // Re-initialize flags

			// Set test version
			SetVersion("test-version")

			// Capture output
			outBuf := bytes.NewBuffer(nil)
			errBuf := bytes.NewBuffer(nil)
			rootCmd.SetOut(outBuf)
			rootCmd.SetErr(errBuf)

			// Set args
			rootCmd.SetArgs(tt.args)

			// Execute
			err := rootCmd.Execute()

			// Check error
			if tt.wantExitCode && err == nil {
				t.Errorf("Expected error but got none")
			} else if !tt.wantExitCode && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check output
			if tt.wantOut != "" && !strings.Contains(outBuf.String(), tt.wantOut) {
				t.Errorf("Output doesn't contain expected string.\nWant substring: %q\nGot: %q", tt.wantOut, outBuf.String())
			}

			// Check error output
			if tt.wantErr != "" && !strings.Contains(errBuf.String(), tt.wantErr) {
				t.Errorf("Error output doesn't contain expected string.\nWant substring: %q\nGot: %q", tt.wantErr, errBuf.String())
			}
		})
	}
}

func TestGlobalFlags(t *testing.T) {
	// Reset
	rootCmd.ResetCommands()
	rootCmd.ResetFlags()
	initRootCmd()

	// Test repo-root flag
	rootCmd.SetArgs([]string{"--repo-root", "/custom/path", "--help"})
	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if repoRoot != "/custom/path" {
		t.Errorf("repo-root flag not set correctly. Got: %q, Want: %q", repoRoot, "/custom/path")
	}

	// Reset and test quiet flag
	rootCmd.ResetCommands()
	rootCmd.ResetFlags()
	quiet = false
	initRootCmd()

	rootCmd.SetArgs([]string{"--quiet", "--help"})
	err = rootCmd.Execute()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !quiet {
		t.Error("quiet flag not set correctly")
	}
}

func TestVersionTemplate(t *testing.T) {
	// Reset
	rootCmd.ResetCommands()
	rootCmd.ResetFlags()
	initRootCmd()

	SetVersion("1.2.3")

	outBuf := bytes.NewBuffer(nil)
	rootCmd.SetOut(outBuf)
	rootCmd.SetArgs([]string{"--version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	want := "mobilecombackup version 1.2.3\n"
	if outBuf.String() != want {
		t.Errorf("Version output incorrect.\nGot: %q\nWant: %q", outBuf.String(), want)
	}
}