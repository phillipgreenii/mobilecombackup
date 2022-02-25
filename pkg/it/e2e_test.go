// TODO add integation build tag and update default command to include integration

package it_test

import (
	"github.com/phillipgreen/mobilecombackup/cmd/mobilecombackup"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2E(t *testing.T) {
	tmpdir := t.TempDir()
	err := copyDir("../../testdata", tmpdir)
	if err != nil {
		t.Fatal(err)
	}

	exitCode, _, err := mobilecombackup.Run([]string{
		"mobilecombackup-test",
		"-repo", filepath.Join(tmpdir, "archive"),
		filepath.Join(tmpdir, "to_process"),
	})

	if err != nil {
		t.Errorf("err got %v, want nil", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode got %d, want 0", exitCode)
	}

	// need to check that coalesced calls.xml matches expected value
	// to make this run, the package was changed on cmd/mobilecombackup/main.go which doesn't seem to allow it to be buildable, so may need to refactor the code some more.
  // perhaps result of Run could include result counts?
	t.Errorf("implement me")
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
