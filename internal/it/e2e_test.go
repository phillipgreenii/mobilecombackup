// TODO add integation build tag and update default command to include integration

package it_test

import (
	"path/filepath"
	"testing"

	"github.com/phillipgreen/mobilecombackup/internal/test_support"
	"github.com/phillipgreen/mobilecombackup/pkg/mobilecombackup"
)



func TestParseFlagsCorrect(t *testing.T) {
        // TODO list files in test_data/it/scenerio-* to build this list
        var scenario_paths = []string{
          "../../testdata/it/scenario-00"
        }

        var tests = []struct {
                args []string
                conf config
        }{
                {[]string{},
                        config{repoPath: ".", pathsToProcess: []string{}}},

                {[]string{"-repo", "r/path", "myPath1", "myPath2"},
                        config{repoPath: "r/path", pathsToProcess: []string{"myPath1", "myPath2"}}},
        }


	tmpdir := t.TempDir()
         // TODO add defer which removed tmpdir

        for _, scenario_path := range scenario_paths {
                var scenario = filepath.FileName(scenario)
                t.Run(scenario, func(t *testing.T) {
                        var repo_root = filepath.Join(tmpdir,scenario)
                        err := os.Mkdir(repo_root, 0755)
	                if err != nil {
	                	t.Fatal(err)
	                }
                        // TODO add defer which removes scenario_root
	                err := test_support.CopyDir(filepath.Join(scenario, "original_repo_root"), scenario_root)
	if err != nil {
		t.Fatal(err)
	}

                        var repo_root = filepath.Join(tmpdir,

                        

                        conf, output, err := parseFlags("prog", tt.args)
                        if err != nil {
                                t.Errorf("err got %v, want nil", err)
                        }
                        if output != "" {
                                t.Errorf("output got %q, want empty", output)
                        }
                        if !reflect.DeepEqual(*conf, tt.conf) {
                                t.Errorf("conf got %+v, want %+v", *conf, tt.conf)
                        }
                })
        }
}


> ls testdata/it/scenerio-00/
expected_repo_root  original_repo_root  to_process



  return os.Mkdir(filepath.Join(destination, relPath), 0755)


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
