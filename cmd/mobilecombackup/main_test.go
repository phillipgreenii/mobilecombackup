package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMainIntegration(t *testing.T) {
	// Build the binary for testing
	binPath := filepath.Join(t.TempDir(), "mobilecombackup")
	cmd := exec.Command("go", "build", "-o", binPath, "github.com/phillipgreen/mobilecombackup/cmd/mobilecombackup") // #nosec G204

	// Build the binary
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}

	tests := []struct {
		name         string
		args         []string
		wantExitCode int
		wantStdout   string
		wantStderr   string
	}{
		{
			name:         "no arguments shows help",
			args:         []string{},
			wantExitCode: 0,
			wantStdout:   "mobilecombackup processes call logs and SMS/MMS messages",
		},
		{
			name:         "help flag",
			args:         []string{"--help"},
			wantExitCode: 0,
			wantStdout:   "mobilecombackup processes call logs and SMS/MMS messages",
		},
		{
			name:         "version flag",
			args:         []string{"--version"},
			wantExitCode: 0,
			wantStdout:   "mobilecombackup version dev",
		},
		{
			name:         "unknown command",
			args:         []string{"unknown"},
			wantExitCode: 1,
			wantStderr:   `unknown command "unknown" for "mobilecombackup"`,
		},
		{
			name:         "unknown flag",
			args:         []string{"--unknown"},
			wantExitCode: 1,
			wantStderr:   "unknown flag: --unknown",
		},
		{
			name:         "multiple flags",
			args:         []string{"--repo-root", "/tmp", "--quiet", "--help"},
			wantExitCode: 0,
			wantStdout:   "mobilecombackup processes call logs and SMS/MMS messages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binPath, tt.args...) // #nosec G204

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			// Check exit code
			exitCode := 0
			if exitError, ok := err.(*exec.ExitError); ok {
				exitCode = exitError.ExitCode()
			}

			if exitCode != tt.wantExitCode {
				t.Errorf("Exit code = %d, want %d", exitCode, tt.wantExitCode)
			}

			// Check stdout
			if tt.wantStdout != "" && !strings.Contains(stdout.String(), tt.wantStdout) {
				t.Errorf("Stdout doesn't contain expected string.\nWant substring: %q\nGot: %q", tt.wantStdout, stdout.String())
			}

			// Check stderr
			if tt.wantStderr != "" && !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("Stderr doesn't contain expected string.\nWant substring: %q\nGot: %q", tt.wantStderr, stderr.String())
			}
		})
	}
}

func TestVersionInjection(t *testing.T) {
	// Build with version injection
	binPath := filepath.Join(t.TempDir(), "mobilecombackup")
	cmd := exec.Command("go", "build",
		"-ldflags", "-X main.Version=1.2.3",
		"-o", binPath, "github.com/phillipgreen/mobilecombackup/cmd/mobilecombackup") // #nosec G204

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build binary with version: %v\nOutput: %s", err, output)
	}

	// Test version output
	cmd = exec.Command(binPath, "--version") // #nosec G204
	output, err = cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run version command: %v", err)
	}

	want := "mobilecombackup version 1.2.3"
	if !strings.Contains(string(output), want) {
		t.Errorf("Version output incorrect.\nWant substring: %q\nGot: %q", want, string(output))
	}
}

func TestHelpSubcommand(t *testing.T) {
	// Build the binary
	binPath := filepath.Join(t.TempDir(), "mobilecombackup")
	cmd := exec.Command("go", "build", "-o", binPath, "github.com/phillipgreen/mobilecombackup/cmd/mobilecombackup") // #nosec G204

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}

	// Test help subcommand - Cobra provides automatic help command
	cmd = exec.Command(binPath, "help") // #nosec G204
	output, err = cmd.CombinedOutput()

	// Help command should succeed
	if err != nil {
		t.Fatalf("Help command failed: %v\nOutput: %s", err, output)
	}

	// Should show help text
	want := "Help about any command"
	if !strings.Contains(string(output), want) {
		t.Errorf("Help output doesn't contain expected text.\nWant substring: %q\nGot: %q", want, string(output))
	}
}
