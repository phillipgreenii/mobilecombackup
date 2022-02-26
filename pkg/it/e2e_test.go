// TODO add integation build tag and update default command to include integration

package it_test

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillipgreen/mobilecombackup/pkg/mobilecombackup"
)

func TestE2E(t *testing.T) {
	tmpdir := t.TempDir()
	err := copyDir("../../testdata", tmpdir)
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

	callLineCount, err := countLines(filepath.Join(repoDir, "calls.xml"))
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

// based on https://stackoverflow.com/a/64733815/388006
// and https://golangbyexample.com/copy-file-go/
func copyDir(source, destination string) error {
	var err error = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		var relPath string = strings.Replace(path, source, "", 1)
		if relPath == "" {
			return nil
		}
		if info.IsDir() {
			return os.Mkdir(filepath.Join(destination, relPath), 0755)
		} else {
			return copyFile(filepath.Join(source, relPath),
				filepath.Join(destination, relPath))
		}
	})
	return err
}

func copyFile(source, destination string) error {
	s, err := os.Open(source)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer d.Close()

	//This will copy
	_, err = io.Copy(d, s)

	return err
}

func countLines(source string) (int, error) {
	file, _ := os.Open(source)
	defer file.Close()
	fileScanner := bufio.NewScanner(file)
	lineCount := 0
	for fileScanner.Scan() {
		lineCount++
	}
	return lineCount, nil
}
