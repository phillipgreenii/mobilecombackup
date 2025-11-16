package importer

import (
	"testing"

	"github.com/phillipgreenii/mobilecombackup/pkg/calls"
	"github.com/spf13/afero"
)

// TestResolveContact_EmptyNumber tests resolveContact with empty phone number
func TestResolveContact_EmptyNumber(t *testing.T) {
	fs := afero.NewMemMapFs()
	repoPath := "/test/repo"

	// Create basic import options
	options := &ImportOptions{
		RepoRoot:       repoPath,
		Paths:          []string{},
		Quiet:          true,
		MaxXMLSize:     500 * 1024 * 1024,
		MaxMessageSize: 10 * 1024 * 1024,
		Fs:             fs,
	}

	// Create mock contacts manager
	contactsManager := NewMockContactsManager()

	// Create tracker
	tracker := NewYearTracker()

	// Create importer
	importer, err := NewCallsImporter(options, contactsManager, tracker)
	if err != nil {
		t.Fatalf("failed to create calls importer: %v", err)
	}

	// Test call with empty number
	call := &calls.Call{
		Number:      "",
		ContactName: "Test Contact",
	}

	// resolveContact should return false for empty number
	resolved := importer.resolveContact(call)
	if resolved {
		t.Error("expected resolveContact to return false for empty number")
	}
}

// TestResolveContact_NilContactsManager tests resolveContact with nil contacts manager
func TestResolveContact_NilContactsManager(t *testing.T) {
	fs := afero.NewMemMapFs()
	repoPath := "/test/repo"

	// Create basic import options
	options := &ImportOptions{
		RepoRoot:       repoPath,
		Paths:          []string{},
		Quiet:          true,
		MaxXMLSize:     500 * 1024 * 1024,
		MaxMessageSize: 10 * 1024 * 1024,
		Fs:             fs,
	}

	// Create tracker
	tracker := NewYearTracker()

	// Create importer with nil contacts manager
	importer, err := NewCallsImporter(options, nil, tracker)
	if err != nil {
		t.Fatalf("failed to create calls importer: %v", err)
	}

	// Test call
	call := &calls.Call{
		Number:      "555-1234",
		ContactName: "Test Contact",
	}

	// resolveContact should return false when contactsManager is nil
	resolved := importer.resolveContact(call)
	if resolved {
		t.Error("expected resolveContact to return false when contactsManager is nil")
	}
}
