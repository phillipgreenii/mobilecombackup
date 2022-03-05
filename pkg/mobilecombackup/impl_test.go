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

	mockCC := mockCallCoalescer{total: 10}

	processor := processorState{
		repoDir,
		&mockCC,
	}

	result, err := processor.Process(pathToProcess)
	if err != nil {
		t.Errorf("err got %v, want nil", err)
	}

	if result.Calls.Total != 38 {
		t.Errorf("total got %d, want 38", result.Calls.Total)
	}
	if result.Calls.New != 28 {
		t.Errorf("new got %d, want 28", result.Calls.New)
	}
	if len(mockCC.pathsCoalesced) != 2 {
		t.Errorf("pathsCoalesced got %d, want 2", len(mockCC.pathsCoalesced))
	}
	if mockCC.flushes != 1 {
		t.Errorf("flushes got %d, want 1", mockCC.flushes)
	}
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
	entriesAdded := len(filepath.Base(filePath))
	mcc.pathsCoalesced = append(mcc.pathsCoalesced, filePath)
	mcc.total += entriesAdded

	var result coalescer.Result
	result.New = entriesAdded
	result.Total = mcc.total

	return result, nil
}

func (mcc *mockCallCoalescer) Flush() error {
	mcc.flushes += 1

	return nil
}
