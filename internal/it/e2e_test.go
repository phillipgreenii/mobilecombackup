// TODO add integation build tag and update default command to include integration

package it_test

import (
	"path/filepath"
	"testing"

	"github.com/phillipgreen/mobilecombackup/internal/test_support"
	"github.com/phillipgreen/mobilecombackup/pkg/mobilecombackup"
)

func TestE2E(t *testing.T) {
	tmpdir := t.TempDir()
	err := test_support.CopyDir("../../testdata/example0/original_repo_root", tmpdir)
	if err != nil {
		t.Fatal(err)
	}

	repoDir := filepath.Join(tmpdir, "archive")
	pathToProcess := filepath.Join(tmpdir, "to_process")

	exitCode, _, err := mobilecombackup.Run([]string{
		"mobilecombackup-test",
		"-repo", repoDir,
		pathToProcess,
	})

	if err != nil {
		t.Errorf("err got %v, want nil", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode got %d, want 0", exitCode)
	}

	callLineCount, err := test_support.CountLines(filepath.Join(repoDir, "calls.xml"))
	if err != nil {
		t.Errorf("error while counting call lines: %v", err)
	}
	expectedCallLineCount := (2 + // header
		1 + // opening tag
		1 + // closing tag
		22) // count of calls
	if callLineCount != expectedCallLineCount {
		t.Errorf("callLineCount got %d, want %d", callLineCount, expectedCallLineCount)
	}
}
