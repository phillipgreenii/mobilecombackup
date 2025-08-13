package cmd_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestInfoCommandEmptyRepository tests the info command on an empty repository
func TestInfoCommandEmptyRepository(t *testing.T) {
	// Build test binary
	testBin := filepath.Join(t.TempDir(), "mobilecombackup-test")
	buildCmd := exec.Command("go", "build", "-o", testBin, "../../../cmd/mobilecombackup")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build test binary: %v\nOutput: %s", err, output)
	}

	// Create empty repository
	repoRoot := filepath.Join(t.TempDir(), "empty-repo")

	// Initialize repository first
	initCmd := exec.Command(testBin, "--repo-root", repoRoot, "init")
	if output, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to initialize repository: %v\nOutput: %s", err, output)
	}

	// Run info command
	infoCmd := exec.Command(testBin, "--repo-root", repoRoot, "info")
	output, err := infoCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Info command failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)

	// Verify that the output shows zero counts for all entity types
	expectedContent := []string{
		"Calls:",
		"Total: 0 calls",
		"Messages:",
		"Total: 0 messages (0 SMS, 0 MMS)",
		"Attachments:",
		"Count: 0",
		"Total Size: 0 B",
		"Contacts: 0",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Expected output to contain %q, but it didn't.\nFull output:\n%s", expected, outputStr)
		}
	}

	// Verify it doesn't show validation issues for a valid empty repository
	if strings.Contains(outputStr, "Issues:") {
		t.Error("Empty repository should not show issues")
	}

	// Should show validation OK
	if !strings.Contains(outputStr, "Validation: OK") {
		t.Error("Empty repository should show Validation: OK")
	}
}

// TestInfoCommandJSON tests the info command JSON output on empty repository
func TestInfoCommandEmptyRepositoryJSON(t *testing.T) {
	// Build test binary
	testBin := filepath.Join(t.TempDir(), "mobilecombackup-test")
	buildCmd := exec.Command("go", "build", "-o", testBin, "../../../cmd/mobilecombackup")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build test binary: %v\nOutput: %s", err, output)
	}

	// Create empty repository
	repoRoot := filepath.Join(t.TempDir(), "empty-repo")

	// Initialize repository first
	initCmd := exec.Command(testBin, "--repo-root", repoRoot, "init")
	if output, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to initialize repository: %v\nOutput: %s", err, output)
	}

	// Run info command with JSON output
	infoCmd := exec.Command(testBin, "--repo-root", repoRoot, "info", "--json")
	output, err := infoCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Info command failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)

	// Verify JSON structure contains zero counts
	expectedJSONContent := []string{
		`"calls": {}`,
		`"sms": {}`,
		`"contacts": 0`,
		`"validation_ok": true`,
	}

	for _, expected := range expectedJSONContent {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Expected JSON output to contain %q, but it didn't.\nFull output:\n%s", expected, outputStr)
		}
	}
}
