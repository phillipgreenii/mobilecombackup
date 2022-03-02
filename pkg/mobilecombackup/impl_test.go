package mobilecombackup

import (
	"github.com/phillipgreen/mobilecombackup/internal/test_support"
	"github.com/phillipgreen/mobilecombackup/pkg/coalescer"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcess(t *testing.T) {
	tmpdir := t.TempDir()
	err := test_support.CopyDir("../../testdata", tmpdir)
	if err != nil {
		t.Fatal(err)
	}

	repoDir := filepath.Join(tmpdir, "archive")
	pathToProcess := filepath.Join(tmpdir, "to_process")

	mockCC := mockCallCoalescer{}

	processor := processorState{
		repoDir,
		mockCC,
	}

	result, err := processor.Process(pathToProcess)
	if err != nil {
		t.Errorf("err got %v, want nil", err)
	}

	// TODO check files, result, flush
}

type mockCallCoalescer struct {
	pathsCoalesced []string
	total          int
	flushes        int
}

func (mcc *mockCallCoalescer) Supports(filePath string) (bool, error) {

	return strings.Contains(path.Base(filePath), "call"), nil
}

func (mcc *mockCallCoalescer) Coalesce(filePath string) (coalescer.Result, error) {
	mcc.pathsCoalesced = append(mcc.pathsCoalesced, filePath)
	mcc.total += len(mcc.pathsCoalesced)

	var result coalescer.Result
	result.New = len(mcc.pathsCoalesced)
	result.Total = mcc.total

	return result, nil
}

func (mcc *mockCallCoalescer) Flush() error {

	mcc.flushes += 1

	return nil
}
